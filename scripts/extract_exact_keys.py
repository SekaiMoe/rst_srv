import struct
import gzip
from Crypto.Cipher import AES

# 1. 加载密文与元数据
with open("masterdata_payload.enc", "rb") as f:
    enc_data = f.read()

with open("global-metadata.dat", "rb") as f:
    meta = f.read()

with open("libil2cpp.so", "rb") as f:
    so = f.read()

# 2. 从 metadata 解析 StringLiteral (v31)
sl_offset, sl_size = struct.unpack("<II", meta[0x40:0x48])
sld_offset, sld_size = struct.unpack("<II", meta[0x48:0x50])

print(f"[*] StringLiteral 偏移: 0x{sl_offset:X}, 字符串总数: {sl_size // 8}")

def get_str(index):
    pos = sl_offset + index * 8
    length, data_offset = struct.unpack("<II", meta[pos:pos+8])
    real_pos = sld_offset + data_offset
    return meta[real_pos:real_pos+length]

# 3. 在 libil2cpp.so 中搜索指向 stringLiteral 的指针表
# 扫描所有可能的连续字符串字面量
all_literals = []
total_count = sl_size // 8
for i in range(total_count):
    try:
        raw = get_str(i)
        all_literals.append((i, raw))
    except Exception:
        pass

print(f"[*] 成功提取完整字面量池: {len(all_literals)} 个")

# 4. 收集所有长度在 16~64 之间的候选
candidates = []
for idx, raw in all_literals:
    if len(raw) in (16, 24, 32):
        candidates.append((idx, raw, "raw"))
    # Base64 解码候选
    if len(raw) in (24, 44):
        try:
            import base64
            b64 = base64.b64decode(raw)
            if len(b64) in (16, 32):
                candidates.append((idx, b64, "base64"))
        except Exception:
            pass

print(f"[*] 提取到候选 Key/IV 数量: {len(candidates)}")

# 5. 精确匹配解密
found = False
for k_idx, k, k_type in candidates:
    for iv_idx, iv, iv_type in candidates:
        if len(iv) != 16:
            continue
        try:
            cipher = AES.new(k, AES.MODE_CBC, iv)
            head = cipher.decrypt(enc_data[:32])
            
            # SQLite 校验
            if head.startswith(b"SQLite format 3\x00"):
                print("\n" + "="*50)
                print("[🎉 命中真实 SQLite 密钥!]")
                print(f"Key [Index {k_idx}] ({k_type}): {k} (hex: {k.hex()})")
                print(f"IV  [Index {iv_idx}] ({iv_type}): {iv} (hex: {iv.hex()})")
                print("="*50)
                
                dec = AES.new(k, AES.MODE_CBC, iv).decrypt(enc_data)
                with open("masterdata.db", "wb") as out_f:
                    out_f.write(dec)
                print(f"[*] 完整 SQLite 数据库已保存为 masterdata.db ({len(dec)} 字节)")
                found = True
                break
                
            # Gzip 校验
            if head.startswith(b"\x1f\x8b\x08"):
                dec = AES.new(k, AES.MODE_CBC, iv).decrypt(enc_data)
                plain = gzip.decompress(dec)
                print("\n" + "="*50)
                print("[🎉 命中真实 Gzip 压缩密钥!]")
                print(f"Key [Index {k_idx}] ({k_type}): {k} (hex: {k.hex()})")
                print(f"IV  [Index {iv_idx}] ({iv_type}): {iv} (hex: {iv.hex()})")
                print("="*50)
                with open("masterdata.db", "wb") as out_f:
                    out_f.write(plain)
                print(f"[*] 已解压并保存为 masterdata.db ({len(plain)} 字节)")
                found = True
                break
        except Exception:
            continue
    if found:
        break

if not found:
    print("[-] 正在排查前 30 个非空字面量：")
    for idx, raw in all_literals[:30]:
        print(f"[{idx:4d}] {repr(raw)}")
