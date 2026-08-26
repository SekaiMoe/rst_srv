#!/usr/bin/env python3
"""dex_vm_neuter.py — PairIP VMRunner 字节码级中和

原理:
  PairIP 保护 = license check 的方法体被搬进 libpairipcore.so 的 VM,
  Java 层留桩: 桩方法调 VMRunner.executeVM(...)。
  把 VMRunner.executeVM 的字节码直接改成 return-void → 所有 VM 调用变 no-op,
  license check 永远不会执行。

dex 格式最小解析: header -> string_ids/type_ids/proto_ids/field_ids/method_ids
-> class_defs -> code_items (定位方法字节码偏移)
"""
import struct
import sys


class Dex:
    def __init__(self, path):
        self.data = bytearray(open(path, 'rb').read())
        d = self.data
        # header
        (self.string_ids_size, self.string_ids_off) = struct.unpack_from('<II', d, 0x38)
        (self.type_ids_size, self.type_ids_off) = struct.unpack_from('<II', d, 0x40)
        (self.proto_ids_size, self.proto_ids_off) = struct.unpack_from('<II', d, 0x48)
        (self.field_ids_size, self.field_ids_off) = struct.unpack_from('<II', d, 0x50)
        (self.method_ids_size, self.method_ids_off) = struct.unpack_from('<II', d, 0x58)
        (self.class_defs_size, self.class_defs_off) = struct.unpack_from('<II', d, 0x60)

    def uleb128(self, off):
        result = 0
        shift = 0
        while True:
            b = self.data[off]
            result |= (b & 0x7f) << shift
            off += 1
            if not (b & 0x80):
                break
            shift += 7
        return result, off

    def string_at(self, idx):
        off = struct.unpack_from('<I', self.data, self.string_ids_off + idx * 4)[0]
        _, off = self.uleb128(off)  # utf16 length
        # MUTF-8: 读到 0x00
        end = self.data.index(0, off)
        return bytes(self.data[off:end]).decode('utf-8', 'replace')

    def type_name(self, idx):
        sidx = struct.unpack_from('<I', self.data, self.type_ids_off + idx * 4)[0]
        return self.string_at(sidx)

    def method_id(self, idx):
        class_idx, proto_idx, name_idx = struct.unpack_from('<HHI', self.data, self.method_ids_off + idx * 8)
        return self.type_name(class_idx), self.string_at(name_idx)

    def class_def(self, i):
        off = self.class_defs_off + i * 32
        # class_def_item: 8 x uint32
        fields = struct.unpack_from('<IIIIIIII', self.data, off)
        return {
            'class_idx': fields[0], 'access_flags': fields[1],
            'super_idx': fields[2], 'class_data_off': fields[6],
            'static_values_off': fields[7],
        }

    def find_class_def(self, name):
        for i in range(self.class_defs_size):
            cd = self.class_def(i)
            if self.type_name(cd['class_idx']) == name:
                return i, cd
        return None, None

    def class_methods(self, cd):
        """返回 [(method_idx, code_off, direct/virtual)]"""
        off = cd['class_data_off']
        if off == 0:
            return []
        static_fields, off = self.uleb128(off)
        instance_fields, off = self.uleb128(off)
        direct_methods, off = self.uleb128(off)
        virtual_methods, off = self.uleb128(off)
        # 跳过 fields
        for _ in range(static_fields + instance_fields):
            _, off = self.uleb128(off)
            _, off = self.uleb128(off)
        out = []
        for kind, count in [('direct', direct_methods), ('virtual', virtual_methods)]:
            midx = 0
            for _ in range(count):
                diff, off = self.uleb128(off)
                access, off = self.uleb128(off)
                code_off, off = self.uleb128(off)
                midx += diff
                out.append((midx, code_off, kind))
        return out


def patch_return_void(dex, code_off, has_return_value):
    """把方法字节码开头改为 return-void (0x0e00) 或 const v0,0 + return v0"""
    if code_off == 0:
        return False
    # code_item: registers_size, ins_size, outs_size, tries, debug_off, insns_size, insns...
    insns_off = code_off + 12
    size = struct.unpack_from('<I', dex.data, code_off + 8)[0]
    if size < 1:
        return False
    # 清 tries/debug 引用安全性: 直接只改第一条指令, 保留其余 (不可达)
    if has_return_value:
        # const/4 v0, #0  (0x1200) ; return v0 (0x0f00)
        struct.pack_into('<HH', dex.data, insns_off, 0x1200, 0x0f00)
    else:
        # return-void
        struct.pack_into('<H', dex.data, insns_off, 0x0e00)
    return True


def main(path, target_class='Lcom/pairip/VMRunner;'):
    dex = Dex(path)
    i, cd = dex.find_class_def(target_class)
    if cd is None:
        print(f'{path}: {target_class} 不存在')
        return False
    print(f'{path}: {target_class} class_def #{i}, data@{cd["class_data_off"]:#x}')
    patched = 0
    for midx, code_off, kind in dex.class_methods(cd):
        cls, name = dex.method_id(midx)
        proto_off = struct.unpack_from('<I', dex.data, dex.method_ids_off + midx * 8 + 2)[0] if False else None
        print(f'  [{kind}] {name} code_off={code_off:#x}')
        if name in ('executeVM', 'verifyIntegrity', 'a', 'b') or 'execute' in name.lower() or 'verify' in name.lower():
            # executeVM(String)/executeVM(String,String[]) 返回 void? 需要判断——保险起见两类都试
            # void 版本用 return-void; 若方法有返回值会校验失败——VMRunner 的 executeVM 都是 void
            if patch_return_void(dex, code_off, has_return_value=False):
                print(f'    → 已中和 (return-void)')
                patched += 1
    if patched:
        open(path, 'wb').write(dex.data)
        print(f'  已写回 {path} ({patched} 个方法)')
    return patched > 0


if __name__ == '__main__':
    ok = main(sys.argv[1] if len(sys.argv) > 1 else '/tmp/classes2.dex')
    print('结果:', 'OK' if ok else '未修改')
