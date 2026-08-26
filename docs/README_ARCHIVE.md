# Re:ステージ！プリズムステップ (PRISMSTEP) 游戏数据存档

> 服务期间: 2017-07-31 ~ 2026-07-31 (9 年)
> 本存档仅为个人纪念用途的私有备份，请勿公开分发（官方已明确请求）

## 存档内容

| 目录/文件 | 内容 | 状态 |
|---|---|---|
| `audio_downloads/` | 全部 CRIWARE 音频/视频 (4659 .acb + 14 .awb + 42 .usm, ~17GB) | 下载中 |
| `bundles/` | 全部 AssetBundle 原始下载 (20248 个, ~6.6GB) | 下载中 |
| `bundles_decrypted/` | 解密后的标准 Unity bundle (可直接用 AssetStudio 打开) | 待批量处理 |
| `masterdata.json` | 最终版完整游戏数据库 (93 张表, 27MB, 含最终章) | ✅ 完成 |
| `assetbundle_versions_decrypted.txt` | 全量资源清单 (名称:大小) | ✅ 完成 |
| `cri_audio_list.json` | 音频资源清单 (名称/版本CRC/大小) | ✅ 完成 |
| `libil2cpp.so` / `global-metadata.dat` / `dump.cs` | 游戏代码 dump | ✅ 完成 |
| `Android_masterdata2` / `masterdata_encrypted_new` | 系统数据包 (最终版) | ✅ 完成 |

## 待补充（需要用户自行获取）

- **APK 安装包** (.apks): 百度网盘 `pan.baidu.com/s/1pEDS7x1NNnR-JVr1BkAMUw` 提取码 `x9ma`
- **设备缓存** `/sdcard/Android/data/com.ponycanyon.game.prismstep/`（如有登录期数据、留言等）

## 技术结构

CDN: `https://rs.rst-game.com/` (AWS S3 + CloudFront, 停服后仍在线)

- `Android2/<name>` — AssetBundle（`assetbundle_versions.txt` 清单内）
- `CRI/<name>` — CRIWARE 音频（`cri_audio_version_list` 清单内）
- `Android2/assetbundle_versions.txt` — Base64 + Rijndael-256-CBC 加密的全量清单
- `Android2/Android_masterdata2` — 内含清单 bundle + masterdata bundle

加密算法（.NET `RijndaelManaged`, 256-bit block, CBC, ZeroPadding）:

```
KEY = JmcdTcW7rmAvLhggfReqLxz7qp2GPwuX  (32字节)
IV  = Ysyi3dgMF9KUuVRJ9jj4LgfuWdVG77EC  (32字节, 256bit块)
```

数据洋葱结构:
```
CDN bundle (UnityFS)
 └─ TextAsset (密文) --Rijndael解密--> 内层 UnityFS bundle
     └─ MonoBehaviour / Texture2D / Sprite ... (明文资源)
```

## 工具脚本

| 脚本 | 用途 |
|---|---|
| `download_all_game_audio.py` | 按 `cri_audio_list.json` 批量下载音频 |
| `download_all_bundles.py` | 按 `bundle_list.json` 批量下载 AssetBundle |
| `decrypt_pipeline.py` | 批量解密 bundle → 标准 .unity3d |
| `decrypt_masterdata.py` | 解密最终版 masterdata → masterdata.json |

## 服务器镜像 (mirror/)

`mirror/` 目录完整还原了 CDN 的 URL 结构，可直接作为静态服务器根目录：

```
mirror/
├── index.html                    # 原站根页面 (30字节, 1:1还原)
├── Android2/                     # 20249 个文件, 与清单零误差
│   ├── assetbundle_versions.txt  # 全量清单 (Base64+Rijndael密文)
│   ├── masterdata_encrypted      # 最终版 masterdata (1734941字节, 96章)
│   ├── Android_masterdata2
│   ├── cri_audio_version_list
│   └── ... (全部 20244 个资源 bundle, 原始加密状态)
└── CRI/                          # 4704 个音频/视频 (.acb/.awb/.usm)
```

**启动方式**：

```bash
# 快速测试
cd mirror && python3 -m http.server 8080

# 生产部署 (nginx)
cp mirror/nginx.conf.sample /etc/nginx/sites-available/rst-mirror.conf
# 修改其中的 root 路径后启用
```

**客户端接入**：手机端把 `rs.rst-game.com` 解析到镜像服务器 IP（DNS/hosts）。
注意原站是 HTTPS，若客户端强制 TLS 需要 nginx 配证书并处理安卓 CA 信任问题
（或用 mitm 方案）；建议先抓包确认客户端实际请求的是 http 还是 https。

**重要**：镜像里必须放原始加密文件（`mirror/` 已是）；不要把解密后的
`bundles_decrypted/` 放上服务器，客户端会自己解密。

## 资源查看

- 解密后 bundle: `python3 decrypt_pipeline_fast.py bundles bundles_decrypted 8`
  （约 9 分钟，生成可直接用 AssetStudio/AnimeStudio 打开的标准 .unity3d）
- **vgmstream / vgmtoolbox**: 播放/解包 `.acb/.awb` 音频（内含 HCA）
- **.usm**: UsmTool / vgmtoolkit 解包视频

## 复现命令

```bash
# 1. 下载清单并解密
curl -A "UnityPlayer/6000.0.58f2" https://rs.rst-game.com/Android2/assetbundle_versions.txt -o assetbundle_versions.txt
# (见 decrypt_masterdata.py 同款 Rijndael-256-CBC 解密, base64 后 decrypt)

# 2. 批量下载
python3 download_all_game_audio.py   # 音频 17GB
python3 download_all_bundles.py      # bundle 6.6GB

# 3. 批量解密
python3 decrypt_pipeline.py bundles bundles_decrypted --workers 8
```

## 无 HTTPS 接入 (重要)

不需要证书/DNS 劫持: `patch_client_url.py` 原位替换 `global-metadata.dat` 里的
`https://api.rst-game.com/` → `http://<你的IP>/` (≤24字符), 加 `usesCleartextTraffic`
重打包 APK 即可明文直连私服。完整步骤见 **PATCH_CLIENT_HTTPS.md** (已验证:
文件大小不变、字面量重解析正确)。

## API 接口 (2026-08-26 探明)

- `api.rst-game.com` **仍在线**（全部返回停服 JSON，路由和 handler 还在运行）
- 协议: HTTPS POST 表单 + JSON 响应 `{code, message}`
- **116 个端点** + 296 个请求字段 + 73 个 ApiManager 类 → 详见 **API_DOCUMENTATION.md**
- 签名机制 (`MMSignature`/`Signature`/`Signature2`) 具体算法需反汇编 libil2cpp.so（私服可绕过）
- 搭私服完整要素: CDN 镜像(✅) + API 实现(端点/字段已列全) + HTTPS 证书方案
