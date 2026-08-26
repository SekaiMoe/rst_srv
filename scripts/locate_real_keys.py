import re

with open("global-metadata.dat", "rb") as f:
    meta = f.read()

# 1. 查找包含 AES 相关关键词的所有上下文
keywords = [b"AES_KEY", b"AES_INIT_VECTOR", b"AesKey", b"AesInitVector", b"masterdata"]

print("[*] 正在分析 metadata 中与 AES/MasterData 相关的上下文...")
for kw in keywords:
    for match in re.finditer(re.escape(kw), meta):
        pos = match.start()
        # 提取前后 256 字节的可见字符串
        start = max(0, pos - 128)
        end = min(len(meta), pos + 128)
        snippet = meta[start:end]
        
        # 提取其中长度在 8 到 64 之间的字符串
        found_strs = re.findall(rb'[a-zA-Z0-9_\-\+\/=\$\#\@\!]{8,64}', snippet)
        print(f"\n--- 关键词 [{kw.decode()}] 偏移 0x{pos:X} ---")
        for s in found_strs:
            print("  ->", s.decode('utf-8', errors='ignore'))
