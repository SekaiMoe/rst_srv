#!/usr/bin/env python3
"""patch_client_url.py — 把客户端 global-metadata.dat 里的服务器 URL 改成自己的私服

原理:
  IL2CPP 的字符串字面量存在 global-metadata.dat 的 stringLiteral 表 (length, offset)
  + stringLiteralData 数据块中。替换为 **不超过原长度** 的新串时:
    - 新串写到原 offset (尾部残留字节无引用, 无害)
    - 更新该条目的 length 字段
    - 其他字面量的 offset 全部不动 → 文件结构不变

  因此 https://api.rst-game.com/ (25字节) 与 https://rs.rst-game.com/ (24字节)
  可以原位替换为 http://<你的IP:端口>/ —— 客户端从此走明文 HTTP 连私服,
  不需要 DNS 劫持、不需要 HTTPS 证书。

用法:
  # 查看当前 URL
  python3 patch_client_url.py global-metadata.dat

  # 替换 (两个 URL 都指向同一台私服)
  python3 patch_client_url.py global-metadata.dat --url http://192.168.1.5:8443/

  # API 与 CDN 分开
  python3 patch_client_url.py global-metadata.dat \\
      --api http://192.168.1.5:8443/ --cdn http://192.168.1.5:8443/

注意:
  1. 新 URL 必须 ≤ 原长度 (api ≤25, cdn ≤24), 且以 / 结尾
  2. 改完后 APK 还要允许明文流量 (Android 9+ 默认禁止):
     apktool 反编译 → AndroidManifest.xml 的 <application> 加
     android:usesCleartextTraffic="true" → 回编译 → 重签名
  3. 若游戏对 metadata 做了校验 (少见), 启动会闪退 —— 可再 NOP 掉
     libil2cpp.so 里的校验函数 (本作未见明显校验)
"""
import argparse
import shutil
import struct
import sys

ORIG_API = "https://api.rst-game.com/"
ORIG_CDN = "https://rs.rst-game.com/"


def parse_literals(meta):
    sl_off, sl_size, sld_off, _ = struct.unpack_from("<IIII", meta, 8)
    return sl_off, sl_size, sld_off


def find(meta, target: bytes):
    sl_off, sl_size, sld_off = parse_literals(meta)
    hits = []
    for i in range(sl_size // 8):
        length, off = struct.unpack_from("<II", meta, sl_off + i * 8)
        s = meta[sld_off + off : sld_off + off + length]
        if s == target:
            hits.append((i, length, off))
    return hits


def patch(meta, old: str, new: str):
    if not new.endswith("/"):
        raise SystemExit(f"新 URL 必须以 / 结尾: {new}")
    ob, nb = old.encode(), new.encode()
    if len(nb) > len(ob):
        raise SystemExit(f"新 URL 太长 ({len(nb)} > 原长 {len(ob)}): {new}\n"
                         f"  提示: 用短 IP/域名, 或省略端口 (默认80)")
    meta = bytearray(meta)
    hits = find(meta, ob)
    if not hits:
        raise SystemExit(f"未找到字面量: {old}")
    sl_off, _, sld_off = parse_literals(meta)
    for i, old_len, off in hits:
        meta[sld_off + off : sld_off + off + len(nb)] = nb        # 原位写入
        struct.pack_into("<I", meta, sl_off + i * 8, len(nb))     # 更新长度
    return bytes(meta), hits


def show(meta):
    sl_off, sl_size, sld_off = parse_literals(meta)
    print("当前 rst-game 相关字面量:")
    for i in range(sl_size // 8):
        length, off = struct.unpack_from("<II", meta, sl_off + i * 8)
        s = meta[sld_off + off : sld_off + off + length]
        if b"rst-game" in s or b"rst-project" in s:
            print(f"  #{i} len={length}: {s.decode()}")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("file", help="global-metadata.dat 路径")
    ap.add_argument("--url", help="API 与 CDN 共用的私服地址 (如 http://192.168.1.5:8443/)")
    ap.add_argument("--api", help="仅 API (api.rst-game.com) 的替换")
    ap.add_argument("--cdn", help="仅 CDN (rs.rst-game.com) 的替换")
    ap.add_argument("-o", "--out", help="输出文件 (默认: 原文件.patched)")
    args = ap.parse_args()

    meta = open(args.file, "rb").read()
    show(meta)

    if not (args.url or args.api or args.cdn):
        print("\n(仅查看模式; 用 --url / --api / --cdn 执行替换)")
        return

    api_new = args.api or args.url
    cdn_new = args.cdn or args.url

    total = 0
    if api_new:
        meta, hits = patch(meta, ORIG_API, api_new)
        print(f"\n[patch] {ORIG_API} -> {api_new} ({len(hits)} 处)")
        total += len(hits)
    if cdn_new:
        meta, hits = patch(meta, ORIG_CDN, cdn_new)
        print(f"[patch] {ORIG_CDN} -> {cdn_new} ({len(hits)} 处)")
        total += len(hits)

    out = args.out or args.file + ".patched"
    shutil.copy(args.file, args.file + ".bak")
    open(out, "wb").write(meta)
    print(f"[ok] 已写入 {out} (共替换 {total} 处, 原文件备份为 .bak)")

    # 自检: 重新解析
    print("\n=== 自检 ===")
    show(meta)


if __name__ == "__main__":
    main()
