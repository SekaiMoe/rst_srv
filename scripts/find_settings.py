import os
import re

# 搜索所有的 bundle 或 dat 文件
for root, _, files in os.walk("."):
    for file in files:
        if file.endswith(".py") or file.endswith(".enc") or file.endswith(".cs"):
            continue
        p = os.path.join(root, file)
        try:
            with open(p, "rb") as f:
                content = f.read()
                if b"EncrypterSettings" in content or b"一度設定したら" in content:
                    print(f"[+] 发现 EncrypterSettings 相关资源: {p}")
        except Exception:
            pass
