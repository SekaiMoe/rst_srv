import json
import gzip
import os
from Crypto.Cipher import AES

# 1. 加载密文
with open("masterdata_payload.enc", "rb") as f:
    enc_data = f.read()

# 2. 加载 script.json
json_file = "script.json" if os.path.exists("script.json") else "dump_out/script.json"
print(f"[*] 读取 {json_file}...")
with open(json_file, "r", encoding="utf-8") as f:
    data = json.load(f)

script_strings = data.get("ScriptString", [])
print(f"[*] 共有 {len(script_strings)} 个字符串字面量记录。")

# 提取候选 Key 与 IV
# 支持 UTF-8 编码字节以及 Base64 预解码
candidates = []
for item in script_strings:
    val = item.get("value", "")
    addr = item.get("address", 0)
    
    # 原始 UTF-8 字节
    raw = val.encode("utf-8")
    if len(raw) in (16, 24, 32):
        candidates.append((addr, raw, val, "raw"))
    elif len(raw) > 16:
        candidates.append((addr, raw[:16], val[:16], "slice_16"))
        if len(raw) >= 32:
            candidates.append((addr, raw[:32], val[:32], "slice_32"))
            
    # 尝试 Base64 解码候选
    if len(val) in (24, 44):
        try:
            import base64
            b64_bytes = base64.b64decode(val)
            if len(b64_bytes) in (16, 24, 32):
                candidates.append((addr, b64_bytes, val, "base64"))
        except Exception:
            pass

print(f"[*] 生成了 {len(candidates)} 个候选密钥/向量，开始批量验证...")

found = False
for addr_k, k, val_k, type_k in candidates:
    if len(k) not in (16, 24, 32):
        continue
    for addr_iv, iv, val_iv, type_iv in candidates:
        if len(iv) != 16:
            continue
        try:
            cipher = AES.new(k, AES.MODE_CBC, iv)
            head = cipher.decrypt(enc_data[:32])
            
            # 1. 验证标准 SQLite 头部
            if head.startswith(b"SQLite format 3\x00"):
                print("\n" + "="*50)
                print("[🎉🎉🎉 成功命中真实 SQLite 密钥!]")
                print(f"Key [0x{addr_k:X}] ({type_k}): {val_k} | hex: {k.hex()}")
                print(f"IV  [0x{addr_iv:X}] ({type_iv}): {val_iv} | hex: {iv.hex()}")
                print("="*50)
                
                cipher_full = AES.new(k, AES.MODE_CBC, iv)
                dec = cipher_full.decrypt(enc_data)
                with open("masterdata.db", "wb") as out_f:
                    out_f.write(dec)
                print(f"[*] 完整数据库已保存为 masterdata.db ({len(dec)} 字节)")
                found = True
                break
                
            # 2. 验证 Gzip 压缩头部 (1f 8b 08)
            if head.startswith(b"\x1f\x8b\x08"):
                cipher_full = AES.new(k, AES.MODE_CBC, iv)
                dec = cipher_full.decrypt(enc_data)
                plain = gzip.decompress(dec)
                print("\n" + "="*50)
                print("[🎉🎉🎉 成功命中真实 Gzip 压缩密钥!]")
                print(f"Key [0x{addr_k:X}] ({type_k}): {val_k} | hex: {k.hex()}")
                print(f"IV  [0x{addr_iv:X}] ({type_iv}): {val_iv} | hex: {iv.hex()}")
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
    print("[-] 未在 ScriptString 中直接找到成对明文，列出地址在 0x401D000 附近的字符串条目：")
    for item in script_strings:
        addr = item.get("address", 0)
        if 0x401C000 <= addr <= 0x401E000:
            print(f" -> [0x{addr:X}] {item.get('value')}")
