import json
import gzip
import zlib
from Crypto.Cipher import AES

# 从 dump.cs 提取的明文常量
KEY_STR = "dXWXKKLrVLDgdXHmKmMdVReuepMXrqu4"
IV_STR  = "k3qCmcxzzWfdCHUUVvvmAaZGbQKanCUM"

key = KEY_STR.encode("utf-8")
# AES-CBC 标准 IV 截取前 16 字节；若为 Rijndael-256 则使用 32 字节
iv_16 = IV_STR[:16].encode("utf-8")
iv_32 = IV_STR.encode("utf-8")

with open("masterdata_payload.enc", "rb") as f:
    enc_data = f.read()

print(f"[*] 载入密文大小: {len(enc_data)} 字节")

# 1. 尝试标准 AES-256-CBC (16字节 IV)
def try_unpack(raw_bytes, desc):
    # 尝试直接作为 UTF-8 JSON
    try:
        txt = raw_bytes.decode("utf-8").strip()
        if txt.startswith("{") or txt.startswith("["):
            with open("masterdata.json", "w", encoding="utf-8") as out:
                out.write(txt)
            print(f"[🎉] 成功解密为 JSON ({desc})! 已保存为 masterdata.json")
            return True
    except Exception:
        pass

    # 尝试 Gzip
    try:
        plain = gzip.decompress(raw_bytes)
        with open("masterdata_unpacked.json", "wb") as out:
            out.write(plain)
        print(f"[🎉] 成功解密并解压 Gzip ({desc})! 已保存为 masterdata_unpacked.json")
        return True
    except Exception:
        pass

    # 尝试 Zlib
    try:
        plain = zlib.decompress(raw_bytes)
        with open("masterdata_unpacked.json", "wb") as out:
            out.write(plain)
        print(f"[🎉] 成功解密并解压 Zlib ({desc})! 已保存为 masterdata_unpacked.json")
        return True
    except Exception:
        pass

    # 尝试 MsgPack
    try:
        import msgpack
        obj = msgpack.unpackb(raw_bytes, raw=False)
        with open("masterdata.json", "w", encoding="utf-8") as out:
            json.dump(obj, out, ensure_ascii=False, indent=2)
        print(f"[🎉] 成功解密为 MessagePack ({desc})! 已保存为 masterdata.json")
        return True
    except Exception:
        pass

    return False

# 模式测试
print("[*] 正在执行解密...")
# AES-256-CBC (16 字节 IV)
c1 = AES.new(key, AES.MODE_CBC, iv_16)
d1 = c1.decrypt(enc_data)
if not try_unpack(d1, "AES-256-CBC (IV16)"):
    # 如果有 PKCS7 Padding，剥离末尾填充
    try:
        pad_len = d1[-1]
        if pad_len < 32:
            try_unpack(d1[:-pad_len], "AES-256-CBC (IV16 + Unpad)")
    except Exception:
        pass

# 若未直接识别，输出解密后前 64 字节供排查
print("\n[*] 解密后前 64 字节 Hex:")
print(d1[:64].hex(' '))
print("[*] 解密后前 64 字节 ASCII (若可读):")
print("".join(chr(b) if 32 <= b <= 126 else "." for b in d1[:64]))

with open("masterdata_decrypted.bin", "wb") as out:
    out.write(d1)
print(f"[*] 原始解密流已保存至 masterdata_decrypted.bin ({len(d1)} 字节)")
