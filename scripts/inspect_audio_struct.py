import re

with open("cri_audio_mono.bin", "rb") as f:
    data = f.read()

# 定位 bgm_adv_009.acb 的位置
pos = data.find(b"bgm_adv_009.acb")
if pos != -1:
    start = max(0, pos - 64)
    end = min(len(data), pos + 128)
    snippet = data[start:end]
    
    print(f"[*] 命中偏移 0x{pos:X} 前后 192 字节 Hex Dump:")
    for i in range(0, len(snippet), 16):
        chunk = snippet[i:i+16]
        hex_str = " ".join(f"{b:02x}" for b in chunk)
        ascii_str = "".join(chr(b) if 32 <= b <= 126 else "." for b in chunk)
        print(f"{i:04x}: {hex_str:<48} | {ascii_str}")

# 提取文件中所有包含 '/' 或以 http 开头的路径字符串
print("\n[*] 探测可能存在的子路径前缀:")
path_strings = re.findall(rb'[a-zA-Z0-9_\-\/]{2,}\/[a-zA-Z0-9_\-\/\.]*', data)
for p in set(path_strings):
    print(" ->", p.decode('utf-8', errors='ignore'))
