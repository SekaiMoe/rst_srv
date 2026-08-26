#!/usr/bin/env python3
"""高速批量解密: bundles/ -> bundles_decrypted/ (多进程 + C 库解密)"""
import os
import sys
import ctypes
import traceback
from multiprocessing import Pool

KEY = b'JmcdTcW7rmAvLhggfReqLxz7qp2GPwuX'
IV = b'Ysyi3dgMF9KUuVRJ9jj4LgfuWdVG77EC'

_lib = None

def _init_worker():
    global _lib
    _lib = ctypes.CDLL(os.path.abspath('librijndael256.so'))
    _lib.rijndael256_cbc_decrypt.argtypes = [ctypes.c_char_p, ctypes.c_size_t,
                                             ctypes.c_char_p, ctypes.c_char_p]

def decrypt_raw(data: bytes) -> bytes:
    buf = ctypes.create_string_buffer(data, len(data))
    _lib.rijndael256_cbc_decrypt(buf, len(data), KEY, IV)
    return buf.raw[:len(data)]

def process_bundle(args):
    in_path, out_dir = args
    import UnityPy
    base = os.path.basename(in_path)
    try:
        env = UnityPy.load(in_path)
        saved = []
        for obj in env.objects:
            if obj.type.name != 'TextAsset':
                continue
            d = obj.read()
            s = d.m_Script if hasattr(d, 'm_Script') else d.script
            if isinstance(s, str):
                s = s.encode('utf-8', 'surrogateescape')
            if not s:
                continue
            if s[:7] == b'UnityFS':
                # 本身就是明文 bundle, 直接保存
                out = s
                ext = '.unity3d'
            else:
                out = decrypt_raw(s)
                if out[:7] != b'UnityFS':
                    # 解密后不是 bundle: 原始数据原样保存
                    out, ext = s, '.bin'
                else:
                    ext = '.unity3d'
            name = d.m_Name if d.m_Name else 'noname'
            op = os.path.join(out_dir, f'{base}__{name}{ext}')
            with open(op, 'wb') as f:
                f.write(out)
            saved.append(name)
        return (base, len(saved), None)
    except Exception as e:
        return (base, 0, f'{e}\n{traceback.format_exc()[-500:]}')

if __name__ == '__main__':
    in_dir = sys.argv[1] if len(sys.argv) > 1 else 'bundles'
    out_dir = sys.argv[2] if len(sys.argv) > 2 else 'bundles_decrypted'
    workers = int(sys.argv[3]) if len(sys.argv) > 3 else os.cpu_count()
    os.makedirs(out_dir, exist_ok=True)

    files = sorted(os.path.join(in_dir, f) for f in os.listdir(in_dir))
    # 跳过已处理
    existing = set(os.listdir(out_dir))
    todo = [(f, out_dir) for f in files
            if not any(e.startswith(os.path.basename(f) + '__') for e in existing)]
    print(f'[*] 总 {len(files)}, 待处理 {len(todo)}, {workers} 进程', flush=True)

    ok = fail = 0
    errors = []
    with Pool(workers, initializer=_init_worker) as pool:
        for n, (base, cnt, err) in enumerate(pool.imap_unordered(process_bundle, todo, chunksize=64), 1):
            if err:
                fail += 1
                errors.append(f'{base}: {err}')
            else:
                ok += 1
            if n % 500 == 0 or n == len(todo):
                print(f'[{n}/{len(todo)}] ok={ok} fail={fail}', flush=True)

    with open('decrypt_errors.txt', 'w') as f:
        f.write('\n'.join(errors))
    print(f'DONE ok={ok} fail={fail}')
