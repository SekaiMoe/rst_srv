import msgpack
import json
import base64

def safe_decode(obj):
    if isinstance(obj, bytes):
        try:
            return obj.decode("utf-8")
        except UnicodeDecodeError:
            # 遇到非法 UTF-8，回退为 Hex 字符串表示
            return f"HEX::{obj.hex()}"
    elif isinstance(obj, list):
        return [safe_decode(item) for item in obj]
    elif isinstance(obj, dict):
        return {safe_decode(key): safe_decode(value) for key, value in obj.items()}
    else:
        return obj

with open("masterdata_decrypted.bin", "rb") as f:
    data = f.read()

print(f"[*] 载入数据大小: {len(data)} 字节，开始流式二进制解析...")

# raw=True 保证不会因为非 UTF-8 字符抛出崩溃
unpacker = msgpack.Unpacker(raw=True, strict_map_key=False)
unpacker.feed(data)

records = []
for i, obj in enumerate(unpacker):
    records.append(safe_decode(obj))

print(f"[🎉🎉🎉 成功安全提取出 {len(records)} 个数据对象 / 数据表！]")

if len(records) == 1:
    final_data = records[0]
else:
    final_data = records

# 检查对象结构并预览键名
if isinstance(final_data, dict):
    print(f"[*] 顶层数据表 (Tables): {list(final_data.keys())[:20]}")
elif isinstance(final_data, list):
    print(f"[*] 列表总长度: {len(final_data)}")
    if len(final_data) > 0 and isinstance(final_data[0], dict):
        print(f"[*] 第一条记录字段: {list(final_data[0].keys())}")

out_file = "masterdata.json"
with open(out_file, "w", encoding="utf-8") as f:
    json.dump(final_data, f, ensure_ascii=False, indent=2)

print(f"[*] 已完整无损导出为 {out_file} (文件大小: {len(open(out_file, 'rb').read())} 字节)")
