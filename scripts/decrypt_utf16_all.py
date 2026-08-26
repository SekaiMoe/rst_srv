import struct
import gzip
import zlib
import base64
import hashlib
from Crypto.Cipher import AES

with open("global-metadata.dat", "rb") as f:
    meta = f.read()

with open("masterdata_payload.enc", "rb") as f:
    enc_data = f.read()

# 解析 Metadata v31
sl_offset, sl_size, sld_offset, sld_size = struct.unpack("<IIII", meta[8:24])
count = sl_size // 8

raw_candidates = set()

for i in range(count):
    entry_pos = sl_offset + i * 8
    length, data_offset = struct.unpack("<II", meta[entry_pos:entry_pos+8])
    if data_offset + length <= sld_size:
        raw = meta[sld_offset + data_offset : sld_offset + data_offset + length]
        if not raw:
            continue
            
        # 1. 尝试直接作为 UTF-16-LE 解码成文本
        try:
            txt_u16 = raw.decode("utf-16le")
            if txt_u16.isprintable() and len(txt_u16) >= 4:
                raw_candidates.add(txt_u16.encode("utf-8"))
        except Exception:
            pass
            
        # 2. 尝试作为 UTF-8 解码成文本
        try:
            txt_u8 = raw.decode("utf-8")
            if txt_u8.isprintable() and len(txt_u8) >= 4:
                raw_candidates.add(txt_u8.encode("utf-8"))
        except Exception:
            pass

        # 3. 原始 Raw 字节
        raw_candidates.add(raw)

print(f"[*] 收集到有效文本与字节池: {len(raw_candidates)} 条")

# 构造密钥库
keys = []
ivs = []

for item in raw_candidates:
    # 长度正好为 16, 24, 32
    if len(item) in (16, 24, 32):
        keys.append(item)
    if len(item) == 16:
        ivs.append(item)
        
    # MD5 派生 (16字节)
    keys.append(hashlib.md5(item).digest())
    ivs.append(hashlib.md5(item).digest())
    
    # SHA256 派生 (32字节)
    keys.append(hashlib.sha256(item).digest())

    # Base64 解码
    if len(item) in (24, 44):
        try:
            b64_dec = base64.b64decode(item)
            if len(b64_dec) in (16, 24, 32):
                keys.append(b64_dec)
            if len(b64_dec) == 16:
                ivs.append(b64_dec)
        except Exception:
            pass

keys = list(set(keys))
ivs = list(set(ivs))
print(f"[*] 扩展衍生候选 Key: {len(keys)} 个, IV: {len(ivs)} 个，开始多模式验证...")

found = False

def check_and_save(dec_bytes, desc):
    # 检查 SQLite
    if dec_bytes.startswith(b"SQLite format 3\x00"):
        print(f"\n[🎉🎉🎉 成功解密出 SQLite 数据库! 模式: {desc}]")
        with open("masterdata.db", "wb") as f:
            f.write(dec_bytes)
        return True
    # 检查 Gzip
    if dec_bytes.startswith(b"\x1f\x8b"):
        try:
            plain = gzip.decompress(dec_bytes)
            print(f"\n[🎉🎉🎉 成功解密并解压 Gzip 数据库! 模式: {desc}]")
            with open("masterdata.db", "wb") as f:
                f.write(plain)
            return True
        except Exception:
            pass
    # 检查 Zlib
    if dec_bytes.startswith(b"\x78\x9c") or dec_bytes.startswith(b"\x78\x01"):
        try:
            plain = zlib.decompress(dec_bytes)
            print(f"\n[🎉🎉🎉 成功解密并解压 Zlib 数据库! 模式: {desc}]")
            with open("masterdata.db", "wb") as f:
                f.write(plain)
            return True
        except Exception:
            pass
    return False

# 1. 尝试 AES-CBC
for k in keys:
    if len(k) not in (16, 24, 32):
        continue
    for iv in ivs:
        if len(iv) != 16:
            continue
        try:
            c = AES.new(k, AES.MODE_CBC, iv)
            head = c.decrypt(enc_data[:64])
            if check_and_save(head, "AES-CBC Head Check"):
                # 全量解密
                full_dec = AES.new(k, AES.MODE_CBC, iv).decrypt(enc_data)
                check_and_save(full_dec, "AES-CBC Full")
                print(f"Key (Hex): {k.hex()} | (Str): {repr(k)}")
                print(f"IV  (Hex): {iv.hex()} | (Str): {repr(iv)}")
                found = True
                break
        except Exception:
            continue
    if found:
        break

# 2. 尝试 AES-ECB
if not found:
    print("[*] 正在尝试 AES-ECB 模式...")
    for k in keys:
        if len(k) not in (16, 24, 32):
            continue
        try:
            c = AES.new(k, AES.MODE_ECB)
            head = c.decrypt(enc_data[:64])
            if check_and_save(head, "AES-ECB Head Check"):
                full_dec = AES.new(k, AES.MODE_ECB).decrypt(enc_data)
                check_and_save(full_dec, "AES-ECB Full")
                print(f"Key (Hex): {k.hex()} | (Str): {repr(k)}")
                found = True
                break
        except Exception:
            continue

if not found:
    print("[-] 批量字典未直接命中。")
