# 使用开源社区通用的 il2cpp metadata 结构定位方法
import subprocess, sys

# 推荐直接使用已经预编译好的开源单文件或 Python 库
print("[*] 正在安装与配置 pyil2cpp 解析环境...")
subprocess.run([sys.executable, "-m", "pip", "install", "pyil2cpp", "capstone", "--break-system-packages"])
