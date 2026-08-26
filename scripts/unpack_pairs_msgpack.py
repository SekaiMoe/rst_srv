import msgpack
import json

def to_serializable(obj):
    if isinstance(obj, bytes):
        try:
            return obj.decode("utf-8")
        except UnicodeDecodeError:
            return f"HEX::{obj.hex()}"
    elif isinstance(obj, (list, tuple)):
        # 如果是 key-value pairs 结构 [(k, v), ...] 且 key 都是简单类型，转为 dict 字典
        if obj and all(isinstance(x, (list, tuple)) and len(x) == 2 for x in obj):
            try:
                res = {}
                for k, v in obj:
                    k_str = to_serializable(k)
                    if not isinstance(k_str, str):
                        k_str = str(k_str)
                    res[k_str] = to_serializable(v)
                return res
            except Exception:
                pass
        return [to_serializable(item) for item in obj]
    elif isinstance(obj, dict):
        return {str(to_serializable(k)): to_serializable(v) for k, v in obj.items()}
    else:
        return obj

with open("masterdata_decrypted.bin", "rb") as f:
    data = f.read()

print(f"[*] 载入数据大小: {len(data)} 字节，开始无损结构化解析...")

unpacker = msgpack.Unpacker(
    raw=True, 
    strict_map_key=False, 
    object_pairs_hook=list
)
unpacker.feed(data)

records = []
for i, obj in enumerate(unpacker):
    try:
        norm = to_serializable(obj)
        records.append(norm)
    except Exception as e:
        print(f"[-] 第 {i} 个条目转换异常: {e}")

print(f"\n[🎉🎉🎉 成功提取出 {len(records)} 个数据表/对象条目！]")

final_data = records[0] if len(records) == 1 else records

if isinstance(final_data, dict):
    print(f"[*] 顶层数据表 (Tables): {list(final_data.keys())[:25]}")
elif isinstance(final_data, list) and len(final_data) > 0:
    if isinstance(final_data[0], dict):
        print(f"[*] 第一张表字段: {list(final_data[0].keys())[:20]}")

out_file = "masterdata.json"
with open(out_file, "w", encoding="utf-8") as f:
    json.dump(final_data, f, ensure_ascii=False, indent=2)

print(f"[*] 已完整导出为 {out_file} (体积: {len(open(out_file, 'rb').read())} 字节)")
