import capstone

SO_PATH = "libil2cpp.so"

offsets = {
    "get_AesInitVector": 0x1C3DE98,
    "get_AesKey": 0x1C3DEF0,
    ".cctor": 0x1C3E7D4
}

md = capstone.Cs(capstone.CS_ARCH_ARM64, capstone.CS_MODE_ARM)

with open(SO_PATH, "rb") as f:
    for name, offset in offsets.items():
        f.seek(offset)
        code = f.read(64)
        print(f"\n=== {name} (Offset 0x{offset:X}) ===")
        for i in md.disasm(code, offset):
            print(f"0x{i.address:x}:\t{i.mnemonic:<8} {i.op_str}")
