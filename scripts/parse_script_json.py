import json
import os

json_path = "script.json" if os.path.exists("script.json") else "dump_out/script.json"

if not os.path.exists(json_path):
    print(f"[-] 未找到 {json_path}，跳过 JSON 解析。")
else:
    with open(json_path, "r", encoding="utf-8") as f:
        data = json.load(f)
    
    # 查找字符串字面量映射 (StringLiterals)
    literals = data.get("ScriptString", [])
    print(f"[*] 找到 {len(literals)} 个字符串字面量定义...")
    
    # 查找与 0x401D4A0, 0x401D4A8 或周边相关的字符串
    for item in literals:
        val = item.get("value", "")
        # 长度在 8 到 64 之间且不含换行的字符串
        if 8 <= len(val) <= 64 and "\n" not in val:
            addr = item.get("address", 0)
            print(f" -> [0x{addr:X}] {val}")
