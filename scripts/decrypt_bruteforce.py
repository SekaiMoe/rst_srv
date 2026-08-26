import re
import gzip
from Crypto.Cipher import AES

# 1. 读取密文
with open("masterdata_payload.enc", "rb") as f:
    enc_data = f.read()

# 2. 读取 metadata 中的所有候选字符串
with open("global-metadata.dat", "rb") as f:
    meta_data = f.read()

# 提取 16/32 字节候选
raw_candidates = re.findall(rb'[a-zA-Z0-9_\-+=/]{16,32}', meta_data)
keys = list(set([c[:16] for c in raw_candidates if len(c) >= 16] + [c[:32] for c in raw_candidates if len(c) >= 32]))
ivs = list(set([c[:16] for c in raw_candidates if len(c) >= 16]))

print(f"[*] 准备测试 {len(keys)} 个 Key 和 {len(ivs)} 个 IV 组合...")

found = False
for k in keys:
    for iv in ivs:
        try:
            cipher = AES.new(k, AES.MODE_CBC, iv)
            # 先只解密前 64 字节快速验证头部
            head = cipher.decrypt(enc_data[:64])
            
            # 校验是否为 SQLite 或 Gzip
            if head.startswith(b"SQLite format 3") or head.startswith(b"\x1f\x8b"):
                print("\n[🎉🎉🎉 成功命中解密 Key!]")
                print(f"Key: {k.decode('utf-8', errors='ignore')} (hex: {k.hex()})")
                print(f"IV:  {iv.decode('utf-8', errors='ignore')} (hex: {iv.hex()})")
                
                # 全量解密并保存
                cipher_full = AES.new(k, AES.MODE_CBC, iv)
                dec = cipher_full.decrypt(enc_data)
                
                # 如果是 gzip 压缩则解压
                try:
                    plain_db = gzip.decompress(dec)
                except:
                    plain_db = dec
                    
                with open("masterdata.db", "wb") as out_f:
                    out_f.write(plain_db)
                print(f"[*] 全量数据库已保存为 masterdata.db ({len(plain_db)} 字节)")
                found = True
                break
        except Exception:
            continue
    if found:
        break

if not found:
    print("[-] Metadata 字符串直接组合未命中，Key 可能经过了 Hash/派生运算。")
