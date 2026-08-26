import subprocess

base_url = "https://rs.rst-game.com/Android2/"
ua = "UnityPlayer/6000.0.58f2 (UnityWebRequest/1.0, libcurl/8.10.1-DEV)"

# 常见的子资源命名后缀
names = [
    "masterdata_encrypted",
    "masterdata",
    "cri_audio_version_list",
    "cri_audio_version_list.txt",
    "cri_audio_version_list.bytes",
    "masterdata_encrypted.bytes",
    "CAB-f2e58451e7a5e1daf3afd941582652c8"
]

for name in names:
    url = f"{base_url}{name}"
    cmd = ["curl", "-s", "-I", "-A", ua, url]
    res = subprocess.run(cmd, capture_output=True, text=True)
    first_line = res.stdout.split("\n")[0] if res.stdout else "No response"
    print(f"{name:40s} -> {first_line}")
