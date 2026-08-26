import json
import zlib
import gzip

with open("masterdata_decrypted.bin", "rb") as f:
    data = f.read()

print(f"[*] 载入明文大小: {len(data)} 字节")
print(f"[*] 头部 32 字节 Hex: {data[:32].hex(' ')}")

# 1. 尝试 MessagePack 解析 (日系音游 MasterData 最常用格式)
try:
    import msgpack
    obj = msgpack.unpackb(data, raw=False, strict_map_key=False)
    print("\n[🎉🎉🎉 成功解析为 MessagePack 对象!]")
    if isinstance(obj, dict):
        print(f"[*] 顶级数据表 (Tables): {list(obj.keys())}")
    elif isinstance(obj, list):
        print(f"[*] 数据列表长度: {len(obj)}")
        
    with open("masterdata.json", "w", encoding="utf-8") as out:
        json.dump(obj, out, ensure_ascii=False, indent=2)
    print(" -> 已成功保存为结构化明文: masterdata.json")
    exit(0)
except ImportError:
    print("[-] 未安装 msgpack，可通过 pip install msgpack 安装")
except Exception as e:
    print(f"[-] MsgPack 解析异常: {e}")

# 2. 尝试 Brotli 解压
try:
    import brotli
    decompressed = brotli.decompress(data)
    print(f"\n[🎉 成功通过 Brotli 解压! 解压大小: {len(decompressed)} 字节]")
    with open("masterdata_unpacked.json", "wb") as out:
        out.write(decompressed)
    exit(0)
except Exception:
    pass

# 3. 扫描明文中的 ASCII / 键名特征
import re
ascii_strings = re.findall(rb'[a-zA-Z0-9_]{4,}', data)
print(f"\n[*] 在二进制流中扫描到 {len(ascii_strings)} 个标识符，前 20 个标识符:")
for s in ascii_strings[:20]:
    print(" ->", s.decode('utf-8', errors='ignore'))
