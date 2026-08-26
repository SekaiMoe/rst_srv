import base64
import gzip
import zlib
from Crypto.Cipher import AES

KEY_STR = "dXWXKKLrVLDgdXHmKmMdVReuepMXrqu4"
IV_STR  = "k3qCmcxzzWfdCHUUVvvmAaZGbQKanCUM"

key_bytes = KEY_STR.encode("utf-8")
iv_16     = IV_STR[:16].encode("utf-8")
iv_32     = IV_STR.encode("utf-8")

with open("masterdata_raw_extracted.bin", "rb") as f:
    raw_file = f.read()

# 候选密文段
payloads = [
    ("Offset 0x1C (标准 Payload)", raw_file[0x1C:]),
    ("Offset 0x18 (含长度头)", raw_file[0x18:]),
    ("Offset 0x20 (对齐偏移)", raw_file[0x20:]),
    ("Full Raw File", raw_file)
]

# 检查是否为 Base64
try:
    for desc, p in list(payloads):
        decoded = base64.b64decode(p, validate=False)
        if len(decoded) > 1000:
            payloads.append((f"{desc} -> Base64Decoded", decoded))
except Exception:
    pass

def check_valid(data, desc):
    if not data or len(data) < 16:
        return False
    # 1. 纯文本 JSON / 常见字符
    if data[0:1] in (b"{", b"[") or data.startswith(b"SQLite format 3\x00"):
        print(f"\n[🎉🎉🎉 成功命中!] 格式: 明文/SQLite | 模式: {desc}")
        with open("masterdata.json", "wb") as out:
            out.write(data)
        return True
    # 2. Gzip (1f 8b)
    if data.startswith(b"\x1f\x8b"):
        try:
            plain = gzip.decompress(data)
            print(f"\n[🎉🎉🎉 成功命中!] 格式: Gzip压缩 | 模式: {desc}")
            with open("masterdata.json", "wb") as out:
                out.write(plain)
            return True
        except Exception:
            pass
    # 3. Zlib (78 9c / 78 01 / 78 da)
    if data.startswith(b"\x78\x9c") or data.startswith(b"\x78\x01") or data.startswith(b"\x78\xda"):
        try:
            plain = zlib.decompress(data)
            print(f"\n[🎉🎉🎉 成功命中!] 格式: Zlib压缩 | 模式: {desc}")
            with open("masterdata.json", "wb") as out:
                out.write(plain)
            return True
        except Exception:
            pass
    return False

print("[*] 开始多模式矩阵测试...")

# 测试 1: 标准 AES-256-CBC (16字节块, 16字节IV)
for p_desc, p in payloads:
    rem = len(p) % 16
    test_p = p[:-rem] if rem != 0 else p
    try:
        dec = AES.new(key_bytes, AES.MODE_CBC, iv_16).decrypt(test_p)
        if check_valid(dec, f"AES-256-CBC (IV16) on {p_desc}"):
            exit(0)
    except Exception:
        pass

# 测试 2: 标准 AES-256-CBC (使用后16字节作为IV)
for p_desc, p in payloads:
    rem = len(p) % 16
    test_p = p[:-rem] if rem != 0 else p
    try:
        dec = AES.new(key_bytes, AES.MODE_CBC, IV_STR[16:].encode("utf-8")).decrypt(test_p)
        if check_valid(dec, f"AES-256-CBC (IV后16字节) on {p_desc}"):
            exit(0)
    except Exception:
        pass

# 测试 3: AES-256-ECB
for p_desc, p in payloads:
    rem = len(p) % 16
    test_p = p[:-rem] if rem != 0 else p
    try:
        dec = AES.new(key_bytes, AES.MODE_ECB).decrypt(test_p)
        if check_valid(dec, f"AES-256-ECB on {p_desc}"):
            exit(0)
    except Exception:
        pass

print("[-] 矩阵测试未直接命中，请查看汇编反编译结果以定位准确加解密调用流程。")
