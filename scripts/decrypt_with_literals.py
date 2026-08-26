import gzip
import struct
from Crypto.Cipher import AES

with open("global-metadata.dat", "rb") as f:
    meta = f.read()

with open("masterdata_payload.enc", "rb") as f:
    enc_data = f.read()

sl_offset, sl_size = struct.unpack("<II", meta[0x40:0x48])
sld_offset, sld_size = struct.unpack("<II", meta[0x48:0x50])

count = sl_size // 8
literals = []
for i in range(count):
    entry_pos = sl_offset + i * 8
    length, data_offset = struct.unpack("<II", meta[entry_pos:entry_pos+8])
    real_pos = sld_offset + data_offset
    raw = meta[real_pos:real_pos+length]
    if len(raw) in (16, 24, 32):
        literals.append(raw)

print(f"[*] 共有 {len(literals)} 个严格符合 16/24/32 字节的真实字符串字面量")

found = False
for k in literals:
    for iv in literals:
        if len(iv) != 16:
            continue
        try:
            cipher = AES.new(k, AES.MODE_CBC, iv)
            head = cipher.decrypt(enc_data[:32])
            
            # 1. 验证是否直接是 SQLite
            if head.startswith(b"SQLite format 3\x00"):
                print(f"\n[🎉 成功命中真实 SQLite Key!]")
                print(f"Key: {k.decode('utf-8', errors='ignore')}")
                print(f"IV:  {iv.decode('utf-8', errors='ignore')}")
                
                cipher_full = AES.new(k, AES.MODE_CBC, iv)
                dec = cipher_full.decrypt(enc_data)
                with open("masterdata.db", "wb") as out:
                    out.write(dec)
                print(f"[*] 已成功解密为 masterdata.db ({len(dec)} 字节)")
                found = True
                break
                
            # 2. 验证是否为 Gzip 压缩流 (1f 8b 08)
            if head.startswith(b"\x1f\x8b\x08"):
                cipher_full = AES.new(k, AES.MODE_CBC, iv)
                dec = cipher_full.decrypt(enc_data)
                plain = gzip.decompress(dec)
                print(f"\n[🎉 成功命中真实 Gzip 压缩 Key!]")
                print(f"Key: {k.decode('utf-8', errors='ignore')}")
                print(f"IV:  {iv.decode('utf-8', errors='ignore')}")
                with open("masterdata.db", "wb") as out:
                    out.write(plain)
                print(f"[*] 已解密并解压为 masterdata.db ({len(plain)} 字节)")
                found = True
                break
        except Exception:
            continue
    if found:
        break

if not found:
    print("[-] 未直接命中纯字符串，正在测试 ASCII 字节与 UTF-8 编码组合...")
