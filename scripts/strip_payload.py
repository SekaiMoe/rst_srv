with open("masterdata_raw_extracted.bin", "rb") as f:
    data = f.read()

# 头部 0x1C (28 字节) 之后为纯密文
payload = data[0x1C:]
with open("masterdata_payload.enc", "wb") as f:
    f.write(payload)

print(f"[*] 密文已保存到 masterdata_payload.enc，体积: {len(payload)} 字节")
