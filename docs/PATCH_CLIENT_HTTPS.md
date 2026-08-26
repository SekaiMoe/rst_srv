# 无 HTTPS 接入指南 — 明文 HTTP 直连私服

> 原理: 客户端的服务器地址是 `global-metadata.dat` 里的**明文字面量**，
> 原位替换成 `http://你的IP/` 即可。**不需要证书、不需要 DNS 劫持**。

## 第一步: 补丁 metadata（已完成工具）

```bash
# 查看
python3 patch_client_url.py global-metadata.dat

# 替换 (注意: CDN 槽位最长 24 字符; 省略端口 = 80)
python3 patch_client_url.py global-metadata.dat --url http://192.168.1.55/

# 或 API/CDN 分开两台机器
python3 patch_client_url.py global-metadata.dat \
    --api http://192.168.1.55/ --cdn http://192.168.1.56/
```

- 长度限制: API 槽 ≤25 字符, CDN 槽 ≤24 字符（IP 不带端口最稳; 域名也行如 `http://rst.local/`）
- 原文件自动备份 `.bak`, 输出 `<原名>.patched`
- 文件大小不变、仅改动字面量本身 → 不影响 IL2CPP 加载（已验证: 12652180 字节, 31 字节差异）

## 第二步: 允许明文流量

Android 9+ 默认禁止 HTTP。改 APK 的 AndroidManifest.xml：

```bash
apktool d ReStage.apk -o app
# 编辑 app/AndroidManifest.xml, <application ...> 标签加属性:
#   android:usesCleartextTraffic="true"
# 把 global-metadata.dat.patched 覆盖到:
#   app/assets/bin/Data/Managed/Metadata/global-metadata.dat
apktool b app -o ReStage-patched.apk
```

## 第三步: 重签名

```bash
keytool -genkey -v -keystore my.keystore -alias rst -keyalg RSA -validity 10000
apksigner sign --ks my.keystore --out ReStage-signed.apk ReStage-patched.apk
adb install ReStage-signed.apk
```

(需先卸载官方版——签名不同无法覆盖安装; 存档在本私服是新的)

## 第四步: 私服监听 80 端口

```bash
cd server
./rstserver -addr :80 -mirror ../mirror -data data
```

手机和服务器同一局域网即可。gin 已同时挂载:
- API 路由 (`/account/*` 等 105 端点)
- CDN 静态 (`/Android2/*` `/CRI/*`)

即两个 URL 指向同一地址也完全正常。

## 故障排查

| 症状 | 原因 | 处理 |
|---|---|---|
| 装不上 | 签名冲突 | 先卸载官方版 |
| 启动闪退 ( logo 前 ) | metadata 被校验 (本作未见) 或补丁超长 | 确认用 `.patched` 文件; 检查长度限制 |
| 启动闪退 ( logo 后 ) | `usesCleartextTraffic` 没加成功 | 用 apktool 确认 manifest 属性在 `<application>` 上 |
| 卡标题界面/重试 | 服务器地址不通 | 手机浏览器访问 `http://192.168.1.55/healthz` 应返回 ok |
| 登录失败看日志 | 响应字段不匹配 | tail server.log, 客户端走到哪个端点一目了然 |

## 备注

- 安卓对 `NetworkOnMainThreadException`/明文限制在 targetSdk≥28 生效; Unity 6000 的包 targetSdk ≥34, manifest 补丁**必须做**
- Photon 联机部分 (`_NS/ExitGames` 等) 走独立服务器, 私服未实现, 不影响单人内容
- 抓包调试也可以用这个补丁版: 全明文 HTTP, 随手 charles/wireshark 即可看协议

---

# 2026-08-27 重大更新: PairIP 完整破解方案 (v9, 已实测可玩)

## 背景: 之前不知道的坑

这个包是 **Play 官方 Split APK + PairIP 三层保护**:

1. **Java 层 VM 加密**: R8 把全部字符串常量 (1251 个, 27 个合成类) 和 1 个反射 Method 桥
   (`MessagingUnityPlayerActivity.onCreate`) 挪进静态字段, 由 `libpairipcore.so` 的自定义 VM
   在启动时解密回填。签名不对 → 解密出垃圾 → 2 秒内 SEGV (ExecuteProgram+0x3f2cc)。
2. **Native 层签名校验**: `libunity.so` 的 NEEDED 里有 `libpairipcore.so`, 构造函数直接调它,
   宿主 APK 签名不对 → 构造时跳垃圾地址 (fault 0x1920740)。
3. 百度盘流传的 "原包" 是被转存者重签过的 (CN=np), **先天就是死的**, 别用它当基础包。

## v9 方案 (客户端)

基于 Play 版 base.apk (`/sd/base.apk`, dex 与百度包逐字节相同):

| 层 | 处理 |
|---|---|
| 字符串 | 从活进程 dumpheap (debuggable 探针包) → `parse_hprof3.py` 提取 1251 字段 → const-string 烤进 27 个类 |
| Method 桥 | `onCreate` 改回直接调类内自带的 `onCreate$002` (super 包装) |
| VM | `VMRunner.invoke` 返回 null; `<clinit>` 保留 loadLibrary("pairipcore") (libunity 依赖它存在) |
| SignatureCheck/LicenseClient | 麻醉 (return-void); manifest 删 pairip Application 入口 + LicenseActivity |
| 网络 | `usesCleartextTraffic` + network_security_config 全放行 + metadata URL → `http://127.0.0.1:8080/` |

构建: 只重编 classes2.dex (apktool --no-src + 混合构建), 其余 dex 原样拷贝。
工具链: `apt install default-jdk-headless apksigner zipalign apktool` (Debian forky)。

## Native 签名校验的解法: bind mount (需 root)

```bash
# 每次 (重)装后执行一次, 或用开机脚本 (已装: /data/adb/service.d/rst-bind.sh)
P=$(pm path com.ponycanyon.game.prismstep | head -1 | cut -d: -f2)
mount --bind /data/local/tmp/playcert.apk "$P"
```

`playcert.apk` = Play 签名的 base.apk 原包。libpairipcore 读宿主 APK 验签时读到的是 Play
证书 → 校验通过 → libunity 正常初始化。重装后路径变化, 脚本用 pm path 动态获取。

## 服务器

- `rstserver -addr :8080` (127.0.0.1 直连, 手机本机免 WiFi)
- 客户端启动到标题页后 (等待玩家点击) 才开始发 API 请求

## 关键文件

- `/home/user/repo/b/apkbuild/` — 全套构建产物与工具
  - `app_src/` 可再构建的 smali 源 (已含全部补丁)
  - `rst.hprof` 字符串来源的堆转储; `pairip_strings.json` / `all_strings.json` 提取结果
  - `parse_hprof3.py` / `parse_hprof_all.py` hprof 解析器 (Android 实测布局)
  - `rst.keystore` 签名密钥 (pass: rstpass)
- `/data/local/tmp/playcert.apk` — bind mount 用的 Play 签名包
- `/data/adb/service.d/rst-bind.sh` — 开机自动挂载
