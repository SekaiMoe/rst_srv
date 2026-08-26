#!/usr/bin/env python3
"""批量下载全部 CRI 音频/视频资源 (基于 cri_audio_version_list 清单)"""
import json
import os
import sys
import time
import threading
import urllib.request
import urllib.error
from concurrent.futures import ThreadPoolExecutor, as_completed

BASE_URL = "https://rs.rst-game.com/CRI/"
UA = "UnityPlayer/6000.0.58f2 (UnityWebRequest/1.0, libcurl/8.10.1-DEV)"
OUT_DIR = "audio_downloads"
FAIL_LOG = "download_failures.txt"
PROGRESS_FILE = "download_progress.txt"

entries = json.load(open("cri_audio_list.json"))
os.makedirs(OUT_DIR, exist_ok=True)

lock = threading.Lock()
stats = {"ok": 0, "skip": 0, "fail": 0, "bytes": 0, "mismatch": 0}
failures = []


def fetch(url, timeout=60):
    req = urllib.request.Request(url, headers={"User-Agent": UA})
    return urllib.request.urlopen(req, timeout=timeout).read()


def download(idx_entry):
    i, e = idx_entry
    name, expect = e["name"], e["size"]
    out = os.path.join(OUT_DIR, name)
    if os.path.exists(out) and os.path.getsize(out) == expect:
        return ("skip", name, 0)
    last_err = None
    for attempt in range(4):
        try:
            data = fetch(BASE_URL + name, timeout=90)
            if len(data) != expect:
                # 大小不匹配也保存, 但标记
                with open(out + ".mismatch", "w") as f:
                    f.write(f"expect {expect} got {len(data)}\n")
                with open(out, "wb") as f:
                    f.write(data)
                return ("mismatch", name, len(data))
            with open(out, "wb") as f:
                f.write(data)
            return ("ok", name, len(data))
        except Exception as ex:
            last_err = ex
            time.sleep(1.5 * (attempt + 1))
    return ("fail", name, str(last_err))


t0 = time.time()
with ThreadPoolExecutor(max_workers=12) as ex:
    futs = [ex.submit(download, (i, e)) for i, e in enumerate(entries)]
    for n, fut in enumerate(as_completed(futs), 1):
        kind, name, info = fut.result()
        with lock:
            stats[kind if kind in stats else "fail"] += 1
            if kind in ("ok", "skip"):
                stats["bytes"] += info if kind == "ok" else 0
            if kind in ("fail", "mismatch"):
                failures.append(f"{name}\t{kind}\t{info}")
            if n % 25 == 0 or n == len(entries):
                el = time.time() - t0
                mb = stats["bytes"] / 1048576
                print(f"[{n}/{len(entries)}] ok={stats['ok']} skip={stats['skip']} "
                      f"fail={stats['fail']} mism={stats['mismatch']} "
                      f"{mb:.0f}MB {el/60:.1f}min {mb/max(el,1):.1f}MB/s", flush=True)
                with open(PROGRESS_FILE, "w") as f:
                    f.write(f"{n}/{len(entries)} {stats}\n")

with open(FAIL_LOG, "w") as f:
    f.write("\n".join(failures))
print(f"DONE {stats} elapsed={(time.time()-t0)/60:.1f}min failures={len(failures)}")
