import re

with open("global-metadata.dat", "rb") as f:
    data = f.read()

# 提取长度为 16 或 32 的纯 ASCII 可见字符串（排除了纯下划线/空格）
candidates = re.findall(rb'[a-zA-Z0-9!@#$%^&*()_+\-=\[\]{};:,.<>?/]{16,32}', data)
unique_candidates = sorted(list(set(candidates)))

print(f"[*] 从 global-metadata.dat 提取到 {len(unique_candidates)} 个候选 Key/IV 字符串：")
for c in unique_candidates:
    s = c.decode('utf-8', errors='ignore')
    # 过滤掉明显的类名、系统命名
    if not any(k in s for k in ["UnityEngine", "System.", "Exception", "Callback", "Android", "Component"]):
        print(f" -> '{s}' (len: {len(s)})")
