import os
import re
import requests
import concurrent.futures

# 1. 提取所有音频文件名
with open("cri_audio_mono.bin", "rb") as f:
    content = f.read()

# 匹配所有 .acb 和 .awb 文件名
matches = re.findall(rb'[a-zA-Z0-9_\-]+\.(?:acb|awb)', content)
audio_files = sorted(list(set(m.decode('utf-8') for m in matches)))

print(f"[*] 成功解析出 {len(audio_files)} 个音频文件！")

os.makedirs("audio_downloads", exist_ok=True)
BASE_URL = "https://rs.rst-game.com/Android2/"
HEADERS = {
    "User-Agent": "UnityPlayer/6000.0.58f2 (UnityWebRequest/1.0, libcurl/8.10.1-DEV)",
    "X-Unity-Version": "6000.0.58f2"
}

def download_audio(filename):
    out_path = os.path.join("audio_downloads", filename)
    if os.path.exists(out_path) and os.path.getsize(out_path) > 0:
        return f"[SKIP] {filename}"
    
    url = f"{BASE_URL}{filename}"
    try:
        r = requests.get(url, headers=HEADERS, timeout=20)
        if r.status_code == 200:
            with open(out_path, "wb") as f:
                f.write(r.content)
            return f"[OK] {filename} ({len(r.content)} bytes)"
        else:
            return f"[FAIL {r.status_code}] {filename}"
    except Exception as e:
        return f"[ERR] {filename}: {e}"

# 16 线程并发下载
print("[*] 开始多线程下载音频资源...")
with concurrent.futures.ThreadPoolExecutor(max_workers=16) as executor:
    results = executor.map(download_audio, audio_files)
    for res in results:
        print(res)

print("[*] 音频下载任务全部完成！")
