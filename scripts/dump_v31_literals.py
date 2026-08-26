import struct
import gzip
from Crypto.Cipher import AES

with open("global-metadata.dat", "rb") as f:
    meta = f.read()

with open("masterdata_payload.enc", "rb") as f:
    enc_data = f.read()

# 读取 Header
magic, version = struct.unpack("<II", meta[:8])
print(f"[*] Metadata Version: {version}")

# Offset 0x40 ~ 0x50
sl_offset, sl_size = struct.unpack("<II", meta[0x40:0x48])
sld_offset, sld_size = struct.unpack("<II", meta[0x48:0x50])

count = sl_size // 8
print(f"[*] 解析 StringLiteral 总数: {count}")

all_strings = []
for i in range(count):
    entry_pos = sl_offset + i * 8
    length, data_offset = struct.unpack("<II", meta[entry_pos:entry_pos+8])
    if data_offset + length <= sld_size:
        real_pos = sld_offset + data_offset
        raw = meta[real_pos:real_pos+length]
        all_strings.append((i, raw))

print(f"[*] 成功提取完整字面量: {len(all_strings)} 个")

# 筛选适合作为 AES Key (16/24/32 字节) 和 IV (16 字节) 的字符串
keys = []
ivs = []

for idx, raw in all_strings:
    # 纯字符串或二进制 raw
    if len(raw) in (16, 24, 32):
        keys.append((idx, raw))
    if len(raw) == 16:
        ivs.append((idx, raw))
    # 兼容由 Base64 编码的 24 字符 / 44 字符字符串 (解码后为 16/32 字节)
    if len(raw) in (24, 44):
        try:
            import base64
            b64_dec = base64.b64decode(raw)
            if len(b64_dec) in (16, 24, 32):
                keys.append((idx, b64_dec))
            if len(b64_dec) == 16:
                ivs.append((idx, b64_dec))
        except Exception:
            pass

print(f"[*] 筛选出候选 Key: {len(keys)} 个, 候选 IV: {len(ivs)} 个")

# 自动化解密测试 (带严格校验)
found = False
for k_idx, k in keys:
    for iv_idx, iv in ivs:
        try:
            cipher = AES.new(k, AES.MODE_CBC, iv)
            head = cipher.decrypt(enc_data[:32])
            
            # 1. 验证 SQLite 头部
            if head.startswith(b"SQLite format 3\x00"):
                print(f"\n[🎉🎉🎉 成功命中 SQLite Key!]")
                print(f"Key (Index {k_idx}): {k} | hex: {k.hex()}")
                print(f"IV  (Index {iv_idx}): {iv} | hex: {iv.hex()}")
                
                cipher_full = AES.new(k, AES.MODE_CBC, iv)
                dec = cipher_full.decrypt(enc_data)
                with open("masterdata.db", "wb") as f:
                    f.write(dec)
                print(f"[*] 已成功解密为 masterdata.db ({len(dec)} 字节)")
                found = True
                break
                
            # 2. 验证 Gzip / Deflate 头部
            if head.startswith(b"\x1f\x8b\x08"):
                cipher_full = AES.new(k, AES.MODE_CBC, iv)
                dec = cipher_full.decrypt(enc_data)
                plain = gzip.decompress(dec)
                print(f"\n[🎉🎉🎉 成功命中 Gzip 压缩 Key!]")
                print(f"Key (Index {k_idx}): {k} | hex: {k.hex()}")
                print(f"IV  (Index {iv_idx}): {iv} | hex: {iv.hex()}")
                with open("masterdata.db", "wb") as f:
                    f.write(plain)
                print(f"[*] 已解压并保存为 masterdata.db ({len(plain)} 字节)")
                found = True
                break
        except Exception:
            continue
    if found:
        break

if not found:
    print("[-] 纯 StringLiteral 组合未命中，打印前 50 个提取出的字符串供人工排查：")
    for idx, raw in all_strings[:50]:
        print(f"[{idx:5d}] {raw}")
