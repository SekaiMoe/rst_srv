# 登录流程逆向报告（静态分析 libil2cpp.so + global-metadata.dat）

> 方法: capstone 反汇编 ARM64 + RELA 重定位 → Il2CppMetadataUsage 解码
> (encoded = type<<29 | index*2+1; type5=StringLiteral)
> 无需设备/抓包，纯静态还原

## 三段式登录协议

```
┌─────────────────────────────────────────────────────────────┐
│ POST /account/login1st                                      │
│   HTTP头:  UUID: <设备UUID>                                  │
│   表单:    device_token=<推送token>                          │
│   返回:    {code, message, token, ...}                       │
├─────────────────────────────────────────────────────────────┤
│ POST /account/login2nd                                      │
│   表单:    token=<login1st返回的token>                       │
│   返回:    {code, message, user基础信息...}                   │
├─────────────────────────────────────────────────────────────┤
│ POST /account/login3rd                                      │
│   表单:    token=<token>                                    │
│   返回:    {code, message, 完整玩家数据/资源版本同步...}        │
└─────────────────────────────────────────────────────────────┘
```

## 证据 (反汇编 @ RVA)

| 函数 | RVA | 关键调用 |
|---|---|---|
| AccountLogin1st | 0x1FCE748 | URL槽→"account/login1st" (MethodDef 3558 = get_URL) |
| Login1st lambda | 0x1FDFD54 | AddHeader("UUID", uuid) @0x1c3f084; AddData("device_token", devToken[0x20]) @0x1c3f16c |
| AccountLogin2nd | 0x1FCE840 | URL槽→"account/login2nd" |
| Login2nd lambda | 0x1FDF6A4 | AddData("token", SessionManager[0xb8]) |
| AccountLogin3rd | 0x1FCE8F8 | URL槽→"account/login3rd" |
| Login3rd lambda | 0x1FDF8E4 | AddData("token", ...) |

## 网络层结构 (dump.cs)

- `Connection` 基类: WWWForm 表单 + Headers 字典 + Get/Post(UnityWebRequest)
- `AbstractApiManager : Connection`: AddHeader/AddData/AddData(dict)/Filter(UnityWebRequest)
- 认证: HTTP 头 `UUID` + 表单 `token` (会话)
- 每个端点类覆写 `get_URL()` 返回路径 (如 "account/login1st")

## 其他端点字段推断 (同模式可批量解码)

AccountCreate(uuid), AccountHandover(handover_player_id, handover_code, newUUID) 等参数已从 dump.cs 方法签名直接确认。全部 116 端点的字段可按本方法继续解码（每个 ApiManager 的 b__0 lambda 中收集 AddData 调用）。

## 抓包验证方案 (可选, 用户提供设备时)

1. 安装 mitmproxy: `pip install mitmproxy && mitmproxy --listen-port 8080`
2. 改 APK 信任用户证书: 反编译 `apktool d app.apk`，AndroidManifest.xml 加
   `android:networkSecurityConfig="@xml/nsc"`，res/xml/nsc.xml:
   ```xml
   <network-security-config>
     <base-config cleartextTrafficPermitted="true">
       <trust-anchors><certificates src="user"/></trust-anchors>
     </base-config>
   </network-security-config>
   ```
3. 重打包签名安装，WiFi 代理指向 mitmproxy，启动游戏抓 login1st/2nd/3rd
4. mitmproxy 会话可直接对照本报告验证字段名
