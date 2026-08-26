import struct
import re

# 1. 读取 metadata
with open("global-metadata.dat", "rb") as f:
    meta = f.read()

# 检查 metadata 头部结构
magic, version = struct.unpack("<II", meta[:8])
print(f"[*] Metadata Version: {version}, Magic: 0x{magic:X}")

# 读取 StringLiteral 与 StringLiteralData 偏移
# IL2CPP v24+ 标准头部:
# offset 0x40: stringLiteralOffset, stringLiteralSize
# offset 0x48: stringLiteralDataOffset, stringLiteralDataSize
sl_offset, sl_size = struct.unpack("<II", meta[0x40:0x48])
sld_offset, sld_size = struct.unpack("<II", meta[0x48:0x50])

print(f"[*] StringLiteral Offset: 0x{sl_offset:X}, Count: {sl_size // 8}")
print(f"[*] StringLiteralData Offset: 0x{sld_offset:X}, Size: {sld_size} bytes")

def get_string_literal(index):
    # 每个 StringLiteral 结构体: 4字节长度, 4字节数据段偏移
    entry_pos = sl_offset + index * 8
    length, data_offset = struct.unpack("<II", meta[entry_pos:entry_pos+8])
    real_pos = sld_offset + data_offset
    return meta[real_pos:real_pos+length].decode('utf-8', errors='ignore')

# 2. 从 libil2cpp.so 读取 metadataRegistration 中的全局字符串字面量表
with open("libil2cpp.so", "rb") as f:
    so = f.read()

# 在 metadata 字符串数据中扫描所有 16~32 字节的字符串
print("\n[*] 正在提取所有 StringLiteral 中的可疑 AES 密钥/向量...")
all_literals = []
count = sl_size // 8
for i in range(count):
    try:
        s = get_string_literal(i)
        if 8 <= len(s) <= 64:
            all_literals.append((i, s))
    except Exception:
        pass

# 打印出所有看起来像 Key/IV 的字面量（排除代码类名和常见英文单词）
candidates = []
for idx, s in all_literals:
    # 过滤明显的类名、通用路径
    if not any(k in s for k in ["System.", "UnityEngine", "/", "<", ">", "Exception", "Component", "ToString"]):
        candidates.append((idx, s))
        print(f" -> Index [{idx:5d}] : '{s}' (len: {len(s)})")

