#!/usr/bin/env python3
"""Re:ステージ！プリズムステップ masterdata 完整解密管线
CDN masterdata_encrypted (UnityFS bundle)
  └─ TextAsset "masterdata_encrypted" (Rijndael-256-CBC 加密)
       └─ 解密后 = 内层 UnityFS bundle
            └─ MonoBehaviour (typed) = 全部 93 张游戏数据表
                 └─ masterdata.json
"""
import UnityPy
import json
from py3rijndael import Rijndael

KEY = b'JmcdTcW7rmAvLhggfReqLxz7qp2GPwuX'
IV = b'Ysyi3dgMF9KUuVRJ9jj4LgfuWdVG77EC'
C = Rijndael(KEY, block_size=32)


def rijndael256_cbc_decrypt_raw(data: bytes) -> bytes:
    out = bytearray()
    prev = IV
    for i in range(0, len(data) - len(data) % 32, 32):
        block = data[i:i+32]
        out += bytes(a ^ b for a, b in zip(C.decrypt(block), prev))
        prev = block
    return bytes(out)


def decrypt_masterdata(cdn_file: str, out_json: str = 'masterdata.json',
                       keep_intermediate: bool = False):
    # 第一层: CDN bundle 里的 TextAsset
    env1 = UnityPy.load(cdn_file)
    payload = None
    for obj in env1.objects:
        if obj.type.name == 'TextAsset':
            d = obj.read()
            s = d.m_Script if hasattr(d, 'm_Script') else d.script
            if isinstance(s, str):
                s = s.encode('utf-8', 'surrogateescape')
            payload = s
            break
    assert payload, '未找到 TextAsset'

    # 第二层: Rijndael-256-CBC 解密 -> 内层 UnityFS
    inner = rijndael256_cbc_decrypt_raw(payload)
    if keep_intermediate:
        open('masterdata_inner.unityfs', 'wb').write(inner)

    # 第三层: 内层 bundle 的 MonoBehaviour = 数据本体
    env2 = UnityPy.load(inner if isinstance(inner, bytes) else 'masterdata_inner.unityfs')
    for obj in env2.objects:
        if obj.type.name == 'MonoBehaviour':
            tt = obj.read_typetree()
            meta_keys = {'m_GameObject', 'm_Enabled', 'm_Script', 'm_Name'}
            tables = {k: v for k, v in tt.items() if k not in meta_keys}
            with open(out_json, 'w') as f:
                json.dump(tables, f, ensure_ascii=False, indent=1)
            print(f'✅ {out_json}: {len(tables)} 张表')
            return tables
    raise RuntimeError('内层未找到 MonoBehaviour')


if __name__ == '__main__':
    import sys
    src = sys.argv[1] if len(sys.argv) > 1 else 'masterdata_encrypted_new'
    decrypt_masterdata(src, keep_intermediate=True)
