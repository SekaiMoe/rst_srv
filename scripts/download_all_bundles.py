#!/usr/bin/env python3
"""批量下载全部 AssetBundle (基于社区解密清单 bundle_list.json)"""
import json
import os
import time
import threading
import urllib.request
from concurrent.futures import ThreadPoolExecutor, as_completed

BASE_URLS = ["https://rs.rst-game.com/Android2/", "https://rs.rst-game.com/Android/"]
UA = "UnityPlayer/6000.0.58f2 (UnityWebRequest/1.0, libcurl/8.10.1-DEV)"
OUT_DIR = "bundles"
entries = json.load(open("bundle_list.json"))
items = list(entries.items())
os.makedirs(OUT_DIR, exist_ok=True)

lock = threading.Lock()
stats = {"ok": 0, "skip": 0, "fail": 0, "bytes": 0}
failures = []


def fetch(url, timeout=90):
    req = urllib.request.Request(url, headers={"User-Agent": UA})
    return urllib.request.urlopen(req, timeout=timeout).read()


def download(item):
    name, expect = item
    out = os.path.join(OUT_DIR, name.replace("/", "_"))
    if os.path.exists(out) and os.path.getsize(out) == expect:
        return ("skip", name, 0)
    last_err = None
    for attempt in range(4):
        for base in BASE_URLS[:1] if attempt < 2 else BASE_URLS:  # 先 Android2, 失败再试 Android/
            try:
                data = fetch(base + name)
                if len(data) != expect:
                    continue
                with open(out, "wb") as f:
                    f.write(data)
                return ("ok", name, len(data))
            except urllib.error.HTTPError as e:
                if e.code == 404:
                    break  # 404 换目录试
                last_err = e
            except Exception as ex:
                last_err = ex
        time.sleep(1.5 * (attempt + 1))
    return ("fail", name, str(last_err))


t0 = time.time()
with ThreadPoolExecutor(max_workers=32) as ex:
    futs = [ex.submit(download, it) for it in items]
    for n, fut in enumerate(as_completed(futs), 1):
        kind, name, info = fut.result()
        with lock:
            stats[kind] += 1
            if kind == "ok":
                stats["bytes"] += info
            if kind == "fail":
                failures.append(f"{name}\t{info}")
            if n % 100 == 0 or n == len(items):
                el = time.time() - t0
                print(f"[{n}/{len(items)}] ok={stats['ok']} skip={stats['skip']} fail={stats['fail']} "
                      f"{stats['bytes']/1048576:.0f}MB {el/60:.1f}min", flush=True)

with open("bundle_failures.txt", "w") as f:
    f.write("\n".join(failures))
print(f"DONE {stats} elapsed={(time.time()-t0)/60:.1f}min failures={len(failures)}")
