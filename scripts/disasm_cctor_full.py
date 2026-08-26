import capstone

SO_PATH = "libil2cpp.so"
CCTOR_OFFSET = 0x1C3E7D4

md = capstone.Cs(capstone.CS_ARCH_ARM64, capstone.CS_MODE_ARM)

with open(SO_PATH, "rb") as f:
    f.seek(CCTOR_OFFSET)
    code = f.read(256)
    
print(f"=== .cctor 完整反汇编 (Offset 0x{CCTOR_OFFSET:X}) ===")
for i in md.disasm(code, CCTOR_OFFSET):
    print(f"0x{i.address:x}:\t{i.mnemonic:<8} {i.op_str}")
