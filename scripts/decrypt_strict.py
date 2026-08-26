import re
import gzip
import zlib
from Crypto.Cipher import AES

with open("masterdata_payload.enc", "rb") as f:
    enc_data = f.read()

with open("global-metadata.dat", "rb") as f:
    meta = f.read()

# 提取所有长度为 16、24、32 的可见字符串
raw_strings = set(re.findall(rb'[a-zA-Z0-9_\-\+\/=\$\#\@\!]{8,40}', meta))
candidates = []
for s in raw_strings:
    if len(s) == 16 or len(s) == 24 or len(s) == 32:
        candidates.append(s)
    elif len(s) > 16:
        candidates.append(s[:16])
        if len(s) >= 32:
            candidates.append(s[:32])

candidates = list(set(candidates))
print(f"[*] 严格测试候选列表: {len(candidates)} 个候选 Key/IV")

found = False
for k in candidates:
    if len(k) not in (16, 24, 32):
        continue
    for iv in candidates:
        if len(iv) != 16:
            continue
        try:
            cipher = AES.new(k, AES.MODE_CBC, iv)
            head = cipher.decrypt(enc_data[:32])
            
            # 严格 SQLite 校验
            if head.startswith(b"SQLite format 3\x00"):
                print(f"\n[🎉 命中 SQLite Key!]")
                cipher_full = AES.new(k, AES.MODE_CBC, iv)
                dec = cipher_full.decrypt(enc_data)
                with open("masterdata.db", "wb") as f:
                    f.write(dec)
                print(f"Key: {k.decode('utf-8', errors='ignore')}")
                print(f"IV:  {iv.decode('utf-8', errors='ignore')}")
                found = True
                break
                
            # 严格 Gzip / Zlib 校验 (必须前 3 字节是 1f 8b 08 且能完整解压)
            if head.startswith(b"\x1f\x8b\x08"):
                cipher_full = AES.new(k, AES.MODE_CBC, iv)
                dec = cipher_full.decrypt(enc_data)
                plain = gzip.decompress(dec)
                print(f"\n[🎉 命中 Gzip 压缩 Key! 解压大小: {len(plain)} 字节]")
                with open("masterdata.db", "wb") as f:
                    f.write(plain)
                print(f"Key: {k.decode('utf-8', errors='ignore')}")
                print(f"IV:  {iv.decode('utf-8', errors='ignore')}")
                found = True
                break
        except Exception:
            continue
    if found:
        break

if not found:
    print("[-] 静态明文字符串未直接命中，说明 Key 在 libil2cpp.so 内部被代码拼接、Base64 或 Hash 派生。")
