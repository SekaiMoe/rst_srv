import capstone

SO_PATH = "libil2cpp.so"
DECRYPT_OFFSET = 0x1C3E2C8

md = capstone.Cs(capstone.CS_ARCH_ARM64, capstone.CS_MODE_ARM)

with open(SO_PATH, "rb") as f:
    f.seek(DECRYPT_OFFSET)
    code = f.read(160)
    
print(f"=== Decrypt 函数汇编 (Offset 0x{DECRYPT_OFFSET:X}) ===")
for i in md.disasm(code, DECRYPT_OFFSET):
    print(f"0x{i.address:x}:\t{i.mnemonic:<8} {i.op_str}")
