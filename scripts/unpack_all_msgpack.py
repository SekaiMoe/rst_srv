import msgpack
import json

with open("masterdata_decrypted.bin", "rb") as f:
    data = f.read()

print(f"[*] 载入数据大小: {len(data)} 字节，开始流式解析 MessagePack...")

unpacker = msgpack.Unpacker(raw=False, strict_map_key=False)
unpacker.feed(data)

records = []
for i, obj in enumerate(unpacker):
    records.append(obj)

print(f"[🎉🎉🎉 成功提取出 {len(records)} 个数据对象 / 数据表！]")

# 如果只有一个外层对象包裹
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

# 导出为格式化 JSON 文件
out_file = "masterdata.json"
with open(out_file, "w", encoding="utf-8") as f:
    json.dump(final_data, f, ensure_ascii=False, indent=2)

print(f"[*] 已完整导出为 {out_file} (文件大小: {len(open(out_file, 'rb').read())} 字节)")
