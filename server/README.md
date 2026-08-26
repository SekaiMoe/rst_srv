# rstserver — Re:ステージ！プリズムステップ 私服

Go + gin 实现的游戏 API 服务器 + CDN 镜像服务，单二进制同时替代
`api.rst-game.com` 和 `rs.rst-game.com`。

## 运行

```bash
cd server
go build -o rstserver .
./rstserver -addr :8443 -mirror ../mirror -data data
```

参数:
- `-addr` 监听地址 (默认 :8443)
- `-mirror` CDN 镜像目录 (默认 ../mirror，含 Android2/ CRI/)
- `-data` 玩家数据目录 (默认 data)

## 已实现

### API (api.rst-game.com 替代)

| 端点 | 状态 | 说明 |
|---|---|---|
| `POST /account/login1st` | ✅ 完整 | 头 `UUID` + 表单 `device_token` → 发放 token |
| `POST /account/login2nd` | ✅ 完整 | 表单 `token` → 玩家基础信息 (首次自动建档) |
| `POST /account/login3rd` | ✅ 完整 | 表单 `token` → 完整玩家数据 + 资源版本 |
| `POST /account/login` | ✅ 完整 | 旧版单段登录 |
| `POST /account/create` | ✅ 完整 | 建号 (头 UUID) |
| `POST /account/delete` | ✅ | 删号确认 |
| `POST /gacha/list` | ✅ 完整 | 271 个卡池 (masterdata 驱动, 纪念服全部常驻) |
| `POST /gacha/play` | ✅ 完整 | 池明细 (单抽/十连消耗 + 每日免费状态) |
| `POST /gacha/execute` | ✅ 完整 | 真实抽卡: gachaLot 加权 → block 均匀选卡, 扣石头/免费日次 |
| `POST /livestage/playgame` | ✅ 完整 | 开局扣体力 (stage.stamina, 免费曲为 0) |
| `POST /livestage/finishgame` | ✅ 完整 | 结算: stageGoal 评级 S/A/B/C + 12梯度奖励首发 + 高分记录 + 经验升级 + 活动pt |
| `POST /livestage/stonecontinue` | ✅ | 石头续命 (player 表 continue_jewel=10/次) |
| `POST /card/grow` | ✅ 完整 | 强化: 素材 enhance_exp + levelTableCardR1~R5 升级, 扣 enhance_price |
| `POST /card/lock` `card/sell` | ✅ | 锁定/出售 (sell_price 入手, 锁定卡保护) |
| `POST /deck/card` `deck/leader` | ✅ | 编成卡位/队长 |
| `POST /story/status` | ✅ 完整 | 主线按等级解锁 (story.lv), 卡面剧情持卡解锁, 活动全开 |
| `POST /story/read` | ✅ 完整 | 标记已读 + storyReward 初读奖励 (不重复) |
| `POST /ranking/private` 等 5 个 | ✅ | 单机化: 自己的高分/成绩排行 (music 聚合) |
| `POST /loginbonus/get` | ✅ | 每日登录奖 50 石 (防重复) |
| `POST /item/use` `item/list` | ✅ | 道具使用/列表 (体力回复类回满) |
| `POST /ap/healing` | ✅ | 体力回满 |
| `POST /friend/follow` 等 7 个 | ✅ 完整 | NPC 化: 73 名角色作为好友, 关注送友点/搜索/列表 |
| `POST /shop/itemlist` `shop/buy` 等 10 个 | ✅ 完整 | 1371 商品包 (金币购买; 氪金项免费发放), 收据校验直接通过 |
| `POST /shop/exchange` `exchange_list` | ✅ 完整 | 47201 条兑换 (忽略时间窗全开, 货币校验+发放) |
| `POST /presentbox/*` 4 个 | ✅ | 礼物箱 (present_ 前缀, 单件/一键领取) |
| `POST /achieve/*` 4 个 | ✅ 完整 | 8732 成就 (演出次数/剧情/卡数判定), 一键领取 |
| `POST /maker/*` 11 个 | ✅ 完整 | 自制谱上传/列表/下载/游玩/收藏 (data/makers/) |
| `GET /debug/masterdata` | ✅ | 93 张表查询 (客户端本体走 CDN bundle, 此处供调试) |
| `POST /profile/*` 6 个 | ✅ 完整 | 档案: 改名/简介/称号(417项)/最爱卡/公开卡 |
| `POST /card/acce` `acce/*` 9 个 | ✅ 完整 | 饰品装备(4槽位/占用互斥)/强化/锁定/出售 + 批量编成/改名/解散 |
| `POST /vote/*` 2 个 | ✅ 完整 | 6 届历史投票重开 (116 选项, 本地计票, 票券自动消耗) |
| `POST /event/*` 9 个 | ✅ 完整 | 471 活动全开 + Battle 对战 (eventBattle 敌方数值配置下发/结算) + 后台重试结算 |

> **全部 104 个端点已实现, 0 stub** (与 docs/API_DOCUMENTATION.md 完全对齐) |

协议依据: docs/LOGIN_FLOW_ANALYSIS.md + **docs/RESPONSE_FIELDS.md** (客户端 SetJson/DTO 静态逆向的权威字段表)。

响应字段已按客户端 DTO 对齐 (2026-08-26):
- `UserModel`: lv/money/point_free/point_purchased/stamina (IL2CPP 属性直映, 非 level/coin/jewel/ap)
- `PlayerCardModel`: card_id/lv; login3rd 附带 playerCards/units/cardEquipments 子集合
- gacha/execute → card_ids[]; playgame → status/power/stamina/stamina_updated_at
- finishgame → rewards/eventRewards/eventPoint + 钱包字段
- stonecontinue → status/jewel/freeJewel; presentbox → charaBoxLength 系
- shop/buy → point_purchased/point_free; achieve → achievement_id[]
- item/friendpt → friend_pt/total_friend_pt

### CDN (rs.rst-game.com 替代)

- `/Android2/*` → `mirror/Android2/`（20249 文件，Range/206 支持已验证）
- `/CRI/*` → `mirror/CRI/`（4704 音频）
- `/healthz` 健康检查

## 玩家数据

`data/players/<uuid>.json`：
- 钱包: jewel/coin/friend_point; 等级经验 (levelTable lv1..400, 升级回满 AP)
- 体力: ap/ap_max + 自然回复 (每5分钟1点)
- 卡箱: `cards[]` 卡实例 `{id, master_id, level, exp, lock}`
- `decks[5]` 编成、`items` 道具 (901=无偿石 902=有偿石)、`gacha_free` 每日免费/登录奖记录
- `best_scores` 单曲最佳 (score/combo/rank/plays + 12梯度奖励领取标记)
- `read_stories` 已读剧情、`event_points` 活动pt

新玩家初始：jewel=100000, coin=1000000。

## 结算数据模型 (masterdata.json)

```
stage (关卡) ──< stageGoal (score_s/a/b/c 评级线 + combo/clear 目标)
     └─< stageGoalStageGoalReward (12梯度奖励id) ──< stageGoalReward
levelTable (玩家等级→经验/体力上限, lv400)
levelTableCardR1..R5 (卡等级经验曲线, 按稀有度 R1..R5, lv99)
story ──< storyReward (初读奖励); lv 字段=解锁等级
```

测试验证: stage1 提交130000分→S评价→9项梯度奖励(coin+11000)→升级Lv11；
二周目不重复发奖；强化 #2(rarity3)+#3(2500exp)→Lv1→14；
story/read 首读+20石，再读不重发；登录奖防重复；
商店购买(AP糖x5扣100金币)/兑换(票→卡1016)/成就一键领取4256项(+44630石)/
maker上传→列表→下载→游玩/礼物箱一键领取(2件) 全部通过。

## 抽卡数据模型 (masterdata.json)

```
gacha (卡池) ──< gachaDetail (单抽/十连, item_num=消耗) ──< gachaLot (ratio 加权)
                                                          │
                                                          ▼
                                            gachaBlockDetail (block→卡列表, 均匀)
```

验证: 双葉詩穂池 (id 6442) 95%→☆3池3张 / 5%→☆4池10张，模拟10000抽分布 95.0%/5.0%。

## 接入客户端 (推荐: 无证书方案)

客户端硬编码 `https://api.rst-game.com/` + `https://rs.rst-game.com/`，
但这两个 URL 是 `global-metadata.dat` 的明文字面量，可直接原位替换：

```bash
# 上层目录工具 (≤24字符, 省略端口=80)
python3 patch_client_url.py global-metadata.dat --url http://192.168.1.55/
# 再给 APK 加 android:usesCleartextTraffic="true" 重打包重签名
./rstserver -addr :80 ...   # 私服监听 80
```

完整步骤 (含 apktool/签名/故障排查) 见 **../PATCH_CLIENT_HTTPS.md**。
备选: DNS 劫持 + 自签证书 (Magisk 系统CA / networkSecurityConfig), 较繁琐不推荐。

## masterdata 同步机制 (重要)

客户端**不通过 API** 拿 masterdata：
1. login3rd 返回版本号 → 客户端下载 CDN 的 `Android2/assetbundle_versions.txt`
2. 对比本地 `masterdata_encrypted` 大小 (当前 1734941) → 有差异则重新下载
3. bundle 内 TextAsset 是 Rijndael-256-CBC 加密的内层 UnityFS → 客户端自带密钥解密

因此 **mirror/ 的 CDN 服务就是 masterdata 下发端点**，版本一致性由 login3rd 的
`assetbundle_version` 字段 + 镜像文件本身保证。`/debug/masterdata` 仅供调试/工具使用。

## 字段对齐审计 (2026-08-26 最终)

27 组端点字段全量自动审计 **27/27 通过** (login3rd 全 UserModel 字段、
FriendModel/PresentBoxModel 内嵌结构、handover_code、receiptcheck、
buy_music/movie、battle_finish、finishgame eventPoint 等全部按客户端 DTO 对齐)。
附加发现: 认证头字段名为 `TOKEN` (反汇编确认), 已支持 表单 token + 头 TOKEN 双通道。
明细见 docs/RESPONSE_FIELDS.md。

## 原站在线探测 (2026-08-26)

`api.rst-game.com` 仍全功能运行 (104 路由存活, 认证层有效):
错误码分类学 **200/404/400/700** 已对齐 — 详见 docs/LIVE_API_PROBE.md。
code=700(无效token→客户端重登录) 为关键对齐项。

## 后续开发路线

- [x] ~~全部 104 端点实现完毕 (2026-08-26)~~
- [ ] 真机联调: 客户端 SetJson 字段校准 (需设备, 见 ../docs/PATCH_CLIENT_HTTPS.md 明文接入方案)
- [ ] 按客户端 SetJson 逐端点补全响应字段（抓包验证最快, 需设备）

## 结构

```
server/
├── main.go                 # 入口 (-masterdata 参数)
├── internal/api/
│   ├── router.go           # 端点路由 + 认证中间件 + masterdata 调试
│   ├── account.go          # 三段式登录/建号
│   ├── gacha.go            # 真实抽卡 (list/play/execute + 概率引擎)
│   ├── game.go             # 结算核心: 奖励/AP/等级/评级工具
│   ├── game_handlers.go    # livestage 开局/结算/续命 + ranking
│   ├── card.go             # 卡牌强化/锁定/出售 + 编成
│   ├── story.go            # 剧情/登录奖/道具
│   ├── friend.go           # 好友(NPC化) + 礼物箱
│   ├── shop.go             # 商店/兑换/退款 + 成就
│   ├── maker.go            # 自制谱编辑器
│   ├── master.go           # masterdata 加载 + gacha 索引
│   ├── store.go            # 会话 + 玩家 JSON 存储 (卡实例/最佳成绩模型)
│   └── cdn.go              # 静态镜像挂载
├── data/players/           # 玩家存档
└── data/makers/            # 自制谱
```
