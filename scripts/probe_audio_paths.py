import requests

test_file = "bgm_adv_009.acb"
headers = {
    "User-Agent": "UnityPlayer/6000.0.58f2 (UnityWebRequest/1.0, libcurl/8.10.1-DEV)",
    "X-Unity-Version": "6000.0.58f2"
}

# 常见日系 Unity 音游 CDN 音频路径模式
candidate_paths = [
    f"https://rs.rst-game.com/Android2/Audio/{test_file}",
    f"https://rs.rst-game.com/Android2/audio/{test_file}",
    f"https://rs.rst-game.com/Android2/CRIWARE/{test_file}",
    f"https://rs.rst-game.com/Android2/criware/{test_file}",
    f"https://rs.rst-game.com/Android2/Sound/{test_file}",
    f"https://rs.rst-game.com/Android2/sound/{test_file}",
    f"https://rs.rst-game.com/Android2/Cri/{test_file}",
    f"https://rs.rst-game.com/Android2/acb/{test_file}",
    f"https://rs.rst-game.com/Android/Audio/{test_file}",
    f"https://rs.rst-game.com/Audio/{test_file}",
    f"https://rs.rst-game.com/cri/{test_file}",
    f"https://rs.rst-game.com/sound/{test_file}"
]

print(f"[*] 正在探测 {test_file} 的有效 URL...")
for url in candidate_paths:
    try:
        r = requests.head(url, headers=headers, timeout=5)
        print(f"[{r.status_code}] {url}")
        if r.status_code == 200:
            print(f"\n[!!!] 命中真实路径: {url}")
            break
    except Exception as e:
        print(f"[ERR] {url}: {e}")
