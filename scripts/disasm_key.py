import capstone

# 将此处替换为 grep 查到的实际 Offset (十六进制)
OFFSET_KEY = 0x1234560  
SO_PATH = "libil2cpp.so"

with open(SO_PATH, "rb") as f:
    f.seek(OFFSET_KEY)
    code = f.read(128)

print(f"[*] 反汇编 Offset 0x{OFFSET_KEY:X} 前 32 条指令:")
md = capstone.Cs(capstone.CS_ARCH_ARM64, capstone.CS_MODE_ARM)
for i in md.disasm(code, OFFSET_KEY):
    print(f"0x{i.address:x}:\t{i.mnemonic}\t{i.op_str}")
