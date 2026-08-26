import struct
import gzip
from Crypto.Cipher import AES

with open("global-metadata.dat", "rb") as f:
    meta = f.read()

with open("masterdata_payload.enc", "rb") as f:
    enc_data = f.read()

# v31 Header 结构解析
magic, version = struct.unpack("<II", meta[:8])
sl_offset, sl_size, sld_offset, sld_size = struct.unpack("<IIII", meta[8:24])

print(f"[*] Metadata v{version} 头部校验:")
print(f" -> StringLiteralOffset: 0x{sl_offset:X}, Size: {sl_size} bytes ({sl_size // 8} 条)")
print(f" -> StringLiteralDataOffset: 0x{sld_offset:X}, Size: {sld_size} bytes")

count = sl_size // 8
candidates = []

for i in range(count):
    entry_pos = sl_offset + i * 8
    length, data_offset = struct.unpack("<II", meta[entry_pos:entry_pos+8])
    if data_offset + length <= sld_size:
        raw = meta[sld_offset + data_offset : sld_offset + data_offset + length]
        
        # 16, 24, 32 字节原生字符串
        if len(raw) in (16, 24, 32):
            candidates.append((i, raw, "raw"))
        
        # Base64 解码候选 (长度 24 或 44)
        if len(raw) in (24, 44):
            try:
                import base64
                b64 = base64.b64decode(raw)
                if len(b64) in (16, 32):
                    candidates.append((i, b64, "base64"))
            except Exception:
                pass

print(f"[*] 筛选出 {len(candidates)} 个候选 Key/IV，正在严格匹配解密...")

found = False
for k_idx, k, k_type in candidates:
    for iv_idx, iv, iv_type in candidates:
        if len(iv) != 16:
            continue
        try:
            cipher = AES.new(k, AES.MODE_CBC, iv)
            head = cipher.decrypt(enc_data[:32])
            
            # 1. 严格 SQLite 校验
            if head.startswith(b"SQLite format 3\x00"):
                print("\n" + "="*50)
                print("[🎉 成功解密出 SQLite 数据库!]")
                print(f"Key [Index {k_idx}] ({k_type}): {k.decode('utf-8', errors='ignore')} (hex: {k.hex()})")
                print(f"IV  [Index {iv_idx}] ({iv_type}): {iv.decode('utf-8', errors='ignore')} (hex: {iv.hex()})")
                print("="*50)
                
                dec = AES.new(k, AES.MODE_CBC, iv).decrypt(enc_data)
                with open("masterdata.db", "wb") as f:
                    f.write(dec)
                print(f"[*] 数据库已成功保存为 masterdata.db ({len(dec)} 字节)")
                found = True
                break
                
            # 2. 严格 Gzip 校验
            if head.startswith(b"\x1f\x8b\x08"):
                dec = AES.new(k, AES.MODE_CBC, iv).decrypt(enc_data)
                plain = gzip.decompress(dec)
                print("\n" + "="*50)
                print("[🎉 成功解密并解压 Gzip 数据库!]")
                print(f"Key [Index {k_idx}] ({k_type}): {k.decode('utf-8', errors='ignore')} (hex: {k.hex()})")
                print(f"IV  [Index {iv_idx}] ({iv_type}): {iv.decode('utf-8', errors='ignore')} (hex: {iv.hex()})")
                print("="*50)
                with open("masterdata.db", "wb") as f:
                    f.write(plain)
                print(f"[*] 数据库已成功保存为 masterdata.db ({len(plain)} 字节)")
                found = True
                break
        except Exception:
            continue
    if found:
        break

if not found:
    print("[-] 打印前 20 个解析出的字符串字面量:")
    for i in range(min(20, count)):
        entry_pos = sl_offset + i * 8
        length, data_offset = struct.unpack("<II", meta[entry_pos:entry_pos+8])
        print(f"[{i:4d}]", meta[sld_offset + data_offset : sld_offset + data_offset + length])
