import json
import gzip
import zlib
from rijndael import rijndael

KEY_STR = "dXWXKKLrVLDgdXHmKmMdVReuepMXrqu4"
IV_STR  = "k3qCmcxzzWfdCHUUVvvmAaZGbQKanCUM"

key = KEY_STR.encode("utf-8")
iv  = IV_STR.encode("utf-8")

# 初始化 Rijndael (block_size = 32 字节 = 256 位)
r = rijndael(key, block_size=32)

with open("masterdata_payload.enc", "rb") as f:
    enc_data = f.read()

print(f"[*] 密文体积: {len(enc_data)} 字节 (总计 {len(enc_data) // 32} 个 256-bit 块)")

# 执行 CBC 模式解密
decrypted = bytearray()
prev_block = iv

for i in range(0, len(enc_data), 32):
    block = enc_data[i:i+32]
    # Rijndael 块解密
    dec_block = r.decrypt(block)
    # CBC 异或上一个密文块
    plain_block = bytes(a ^ b for a, b in zip(dec_block, prev_block))
    decrypted.extend(plain_block)
    prev_block = block

raw_data = bytes(decrypted)

# 去除 PKCS7 填充
pad_len = raw_data[-1]
if 0 < pad_len <= 32 and all(b == pad_len for b in raw_data[-pad_len:]):
    unpadded_data = raw_data[:-pad_len]
    print(f"[*] 成功移除 PKCS7 填充 (填充字节数: {pad_len})")
else:
    unpadded_data = raw_data

print(f"[*] 解密明文前 64 字节 Hex:")
print(unpadded_data[:64].hex(' '))
print(f"[*] 解密明文前 64 字节 ASCII:")
print("".join(chr(b) if 32 <= b <= 126 else "." for b in unpadded_data[:64]))

# 尝试解析为 JSON、MsgPack 或解压
def save_result(data):
    # 1. 尝试直接作为 UTF-8 JSON
    try:
        txt = data.decode("utf-8").strip()
        if txt.startswith("{") or txt.startswith("["):
            with open("masterdata.json", "w", encoding="utf-8") as out:
                out.write(txt)
            print("\n[🎉🎉🎉 成功还原为 JSON 明文！]")
            print(" -> 已保存为 masterdata.json")
            return True
    except Exception:
        pass

    # 2. 尝试 Gzip 解压
    try:
        plain = gzip.decompress(data)
        with open("masterdata.json", "wb") as out:
            out.write(plain)
        print("\n[🎉🎉🎉 成功解压 Gzip 压缩包！]")
        print(" -> 已保存为 masterdata.json")
        return True
    except Exception:
        pass

    # 3. 尝试 Zlib 解压
    try:
        plain = zlib.decompress(data)
        with open("masterdata.json", "wb") as out:
            out.write(plain)
        print("\n[🎉🎉🎉 成功解压 Zlib 压缩包！]")
        print(" -> 已保存为 masterdata.json")
        return True
    except Exception:
        pass

    # 4. 尝试 MsgPack 解析
    try:
        import msgpack
        obj = msgpack.unpackb(data, raw=False)
        with open("masterdata.json", "w", encoding="utf-8") as out:
            json.dump(obj, out, ensure_ascii=False, indent=2)
        print("\n[🎉🎉🎉 成功解析 MessagePack 数据！]")
        print(" -> 已保存为 masterdata.json")
        return True
    except Exception:
        pass

    with open("masterdata_plain.bin", "wb") as out:
        out.write(data)
    print("\n[*] 明文已保存为 masterdata_plain.bin")

save_result(unpadded_data)
