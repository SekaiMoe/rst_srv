import os

target1 = "EncrypterSettings".encode("utf-8")
target2 = "一度設定したら".encode("utf-8")

print("[*] 正在扫描当前目录及子目录下的所有 Unity 资产...")
for root, _, files in os.walk("."):
    for file in files:
        if file.endswith((".py", ".enc", ".cs", ".so", ".dat")):
            continue
        p = os.path.join(root, file)
        try:
            with open(p, "rb") as f:
                content = f.read()
                if target1 in content or target2 in content:
                    print(f"[+] 发现包含 EncrypterSettings 的资源: {p} (大小: {len(content)} 字节)")
        except Exception:
            pass
print("[*] 扫描结束。")
