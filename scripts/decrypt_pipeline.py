#!/usr/bin/env python3
"""rst-game AssetBundle 批量解密管线
用法:
  python3 decrypt_pipeline.py <输入目录> <输出目录> [--workers N]

处理逻辑 (来自社区验证 + dump 分析):
  1. CDN 下载的 bundle (如 Android2/Android_card2) 是标准 UnityFS, 内含 TextAsset
  2. TextAsset 内容是 Rijndael-256bit-CBC(ZeroPadding) 加密的另一个 UnityFS
  3. 解密后即为标准 AssetBundle, 可直接用 AssetStudio/UnityPy 浏览导出
"""
import os
import sys
import UnityPy
from py3rijndael import Rijndael
from concurrent.futures import ThreadPoolExecutor, as_completed

KEY = b'JmcdTcW7rmAvLhggfReqLxz7qp2GPwuX'
IV = b'Ysyi3dgMF9KUuVRJ9jj4LgfuWdVG77EC'

# 线程安全的解密器工厂: py3rijndael 的 Rijndael 对象无状态, 可共享
_C = Rijndael(KEY, block_size=32)


def rijndael256_cbc_decrypt_raw(data: bytes) -> bytes:
    """Rijndael-256bit块 CBC 解密, 不剥离 padding (保持输出==输入长度)"""
    out = bytearray()
    prev = IV
    for i in range(0, len(data) - len(data) % 32, 32):
        block = data[i:i+32]
        dec = _C.decrypt(block)
        out += bytes(a ^ b for a, b in zip(dec, prev))
        prev = block
    return bytes(out)


def process_bundle(in_path: str, out_path: str) -> str:
    """解包一层 UnityFS, 解密其中的 TextAsset, 保存为标准 bundle"""
    env = UnityPy.load(in_path)
    saved = 0
    base = os.path.splitext(os.path.basename(out_path))[0]
    for obj in env.objects:
        if obj.type.name != 'TextAsset':
            continue
        d = obj.read()
        s = d.m_Script if hasattr(d, 'm_Script') else d.script
        if isinstance(s, str):
            s = s.encode('utf-8', 'surrogateescape')
        if not s:
            continue
        # 只有加密的 (解密后是 UnityFS) 才处理
        if s[:7] == b'UnityFS':
            open(os.path.join(out_dir_holder[0], base + '__' + d.m_Name + '.unity3d'), 'wb').write(s)
            saved += 1
            continue
        dec = rijndael256_cbc_decrypt_raw(s)
        if dec[:7] != b'UnityFS':
            # 不是加密 bundle, 原样保存 (可能是原始数据)
            open(os.path.join(out_dir_holder[0], base + '__' + d.m_Name + '.bin'), 'wb').write(s)
            saved += 1
            continue
        open(os.path.join(out_dir_holder[0], base + '__' + d.m_Name + '.unity3d'), 'wb').write(dec)
        saved += 1
    return f"{os.path.basename(in_path)}: {saved} assets"


if __name__ == '__main__':
    in_dir, out_dir = sys.argv[1], sys.argv[2]
    workers = int(sys.argv[sys.argv.index('--workers') + 1]) if '--workers' in sys.argv else 8
    os.makedirs(out_dir, exist_ok=True)
    out_dir_holder = [out_dir]

    files = [os.path.join(in_dir, f) for f in os.listdir(in_dir)
             if os.path.isfile(os.path.join(in_dir, f))]
    print(f"[*] {len(files)} 个 bundle 待处理, 输出到 {out_dir}")

    done = fail = 0
    with ThreadPoolExecutor(max_workers=workers) as ex:
        futs = {ex.submit(process_bundle, f, f): f for f in files}
        for fut in as_completed(futs):
            f = futs[fut]
            try:
                msg = fut.result()
                done += 1
                if done % 200 == 0:
                    print(f"[{done}/{len(files)}] {msg}", flush=True)
            except Exception as e:
                fail += 1
                print(f"[FAIL] {f}: {e}", flush=True)
    print(f"DONE ok={done} fail={fail}")
