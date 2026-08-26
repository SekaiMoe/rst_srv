import ctypes
import os

# 检查当前环境架构
import platform
print(f"[*] 当前系统架构: {platform.machine()}")

# 如果在 aarch64 (ARM64) 机器或 Termux 上：
if "aarch64" in platform.machine() or "arm64" in platform.machine():
    try:
        lib = ctypes.CDLL("./libil2cpp.so")
        # 直接调用 get_AesKey 和 get_AesInitVector
        # 注意: il2cpp 函数签名需要传入 MethodInfo* (如果参数为空传 NULL)
        pass
    except Exception as e:
        print(f"加载 so 异常 (缺少依赖库属正常): {e}")
