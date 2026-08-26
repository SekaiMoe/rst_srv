with open("Android_masterdata2", "rb") as f:
    buf = f.read()

# 搜索两个目标资产的字符串位置
targets = [b"masterdata_encrypted", b"cri_audio_version_li"]

for t in targets:
    pos = buf.find(t)
    if pos != -1:
        print(f"[+] Found {t.decode('utf-8', errors='ignore')} at offset: 0x{pos:X} ({pos})")

# 导出 masterdata_encrypted 及其后续数据段
pos = buf.find(b"masterdata_encrypted")
if pos != -1:
    # 截取从资源标记后到文件末尾的二进制数据
    payload = buf[pos:]
    with open("masterdata_encrypted.raw", "wb") as f:
        f.write(payload)
    print(f"[*] Exported masterdata_encrypted.raw ({len(payload)} bytes)")
