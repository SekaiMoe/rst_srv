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
