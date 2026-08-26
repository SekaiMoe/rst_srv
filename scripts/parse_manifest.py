import re

with open("Android_masterdata2", "rb") as f:
    data = f.read()

# 1. 提取所有可见的 ASCII / UTF-8 字符串 (长度 >= 4)
strings = re.findall(rb'[a-zA-Z0-9_\-\.\/]{4,}', data)
decoded_strings = [s.decode('utf-8', errors='ignore') for s in strings]

print("=== 文件中包含的所有资源标识/路径字符串 ===")
seen = set()
for s in decoded_strings:
    if s not in seen and not s.startswith("Unity") and s != "6000.0.58f2":
        seen.add(s)
        print(s)

# 2. 检查是否有明确的子包名或哈希 (CAB- / masterdata 等)
print("\n=== 探测到的潜在资源文件名 ===")
candidates = [s for s in seen if any(k in s.lower() for k in ["master", "audio", "data", "cab-", "bytes", "android"])]
for c in candidates:
    print(f" -> {c}")
