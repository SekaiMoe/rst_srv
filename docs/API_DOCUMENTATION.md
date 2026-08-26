# Re:ステージ！プリズムステップ API 接口文档（逆向提取）

> 来源: global-metadata.dat 字符串池 (18182 条字面量) + dump.cs 类型签名 + api.rst-game.com 在线探测 (2026-08-26)

## 服务器状态

| 域名 | 状态 | 说明 |
|---|---|---|
| `api.rst-game.com` | **在线** | 全局返回停服响应, 路由/handler 仍在运行 |
| `rs.rst-game.com` | **在线** | CDN 静态资源, 已完整镜像 (见 mirror/) |

- 未登录端点: `{"code":404,"message":"サービス終了しました"}`
- 业务端点: `{"code":400,"message":"Re:ステージ！プリズムステップはサービスを終了しました。\n約9年..."}` (带完整停服公告文案, 证明真实 handler 在运行)

## 协议

- **传输**: HTTPS, POST 为主 (`Connection` 基类: UnityWebRequest + WWWForm)
- **请求**: `application/x-www-form-urlencoded` 表单 (字段见下)
- **响应**: JSON (`{code, message, ...数据}`), `AbstractApiManager.ProcessResponseData()` 解析
- **认证**: 登录后持有 `user_token` (`m_userToken` 字段), 请求头/表单携带
- **签名**: 字符串池存在 `MMSignature` / `Signature` / `Signature2` / `NO_SIGN` —— 请求签名机制
  （具体算法在 libil2cpp.so 机器码中, dump.cs 仅有签名, 需 Ghidra/IDA 级逆向才能确定;
  搭私服时可在服务端宽松校验绕过）

## 端点清单 (116 个, 从字符串池全量提取)

### 账号 (account/)
```
account/create              创建账号
account/login               登录 (返回404: 已被全局拦截)
account/login1st / login2nd / login3rd   三段式登录 (1st返回真实停服公告)
account/delete              删除账号
account/delete_info         删除信息
account/handover            继承(引继)
account/handover_code       生成继承码
account/handover_new_code   重新生成继承码
account/info                账号信息
```

### 卡牌/编成 (card/ deck/)
```
card/acce  card/batch  card/grow  card/lock  card/sell  card//acce_skill
deck/batch  deck/card  deck/disband  deck/leader  deck/rename
```

### 抽卡 (gacha/)
```
gacha/list  gacha/play  gacha/execute
```

### 好友 (friend/)
```
friend/follow  friend/unfollow  friend/unfollower
friend/followlist  friend/followerlist  friend/search
friend/livestagelist
```

### 演出/关卡 (livestage/ event/ background/)
```
livestage/playgame  livestage/finishgame  livestage/finishpractice
livestage/gameover  livestage/retrygame
livestage/adscontinue    (看广告续命)
livestage/stonecontinue  (石头续命)
event/battle_start  event/battle_finish  event/finishgame  event/gameover
event/info  event/playgame  event/friendsearch
background/retry/finishgame  background/retry/event_finishgame
```

### 商店 (shop/) / 课金
```
shop/itemlist  shop/purchaselist  shop/buy  shop/buy_movie  shop/buy_music
shop/exchange  shop/exchange_list  shop/log
shop/receiptcheckandroid2  shop/receiptcheckios2   (Google/Apple 收据校验)
repayment/create   (停服后退款申请)
```

### 谱面编辑器 (maker/) —— Re:Stage 特有的自制谱系统
```
maker/download  maker/upload  maker/list  maker/info  maker/playerlist
maker/playgame  maker/finishgame  maker/gameover  maker/retrygame  maker/continue
maker/favorite  maker/save_slot_unlock
```

### 其他
```
achieve/get  achieve/get_all  achieve/list  achieve/stock   (成就)
ap/healing   (体力恢复)
item/list  item/use  item/friendpt  item/livestage
loginbonus/get   (登录奖励)
presentbox/list  presentbox/get  presentbox/get_all  presentbox/log
profile/setname  profile/setprofile  profile/settitle  profile/titlelist
profile/favorite_card  profile/publish_card
ranking/event  ranking/livestage  ranking/past  ranking/power  ranking/private
story/read  story/status  story/unlock_event
vote/info  vote/decision   (投票活动)
acce/grow  acce/lock  acce/sell   (饰品)
```

## 请求字段名 (296 个蛇形字段, 摘录关键)

认证/设备: `uuid` `user_token` `device_token` `app_version` `os_version` `device_model` `client_device` `handover_code` `handover_player_id` `nonce` `signature`
游戏玩法: `card_id(s)` `deck_id` `deck_pos` `gacha_id` `event_id` `stage_id` `difficulty_id` `item_id` `item_kind` `item_num`
成绩: `hit_ranks_counts` `full_combo` `score` `max_combo` `continue_jewel` `use_stone`
支付: `purchase_token` `product_id` `receipt` `transaction_id`
系统: `cri_audio_version_list` (资源版本同步!) `assetbundle_versions.txt` `local_version` `is_updated` `created_at`

## 响应数据结构线索

- `AbstractApiManager` 泛型: `Collection` 后缀类 = 列表查询, `Model` 后缀类 = 单体查询 (73 个 ApiManager 类)
- masterdata 表 (93 张, 见 masterdata.json) 通过 `maker/download`-类似的同步端点下发
- `EndOfServiceInfoModelApiManager` —— 停服信息专用模型(停服前更新加入)

## 私服搭建要点

1. **静态资源**: `mirror/` 已就绪 (rs.rst-game.com 指向它)
2. **API 服务器**: 需要实现上述端点; 响应格式 `{code, message}` 外加各 Model/Collection JSON
3. **签名**: 服务端可选择不校验 `signature`/`MMSignature` (客户端会按代码逻辑生成, 但服务端自己说了算)
4. **HTTPS**: 客户端硬编码 `https://api.rst-game.com/` 与 `https://rs.rst-game.com/`, 需要 DNS 劫持 + 证书方案
   (安卓 7+ 不信用户 CA, 需改 APK 网络安全配置或用 magisk 模块)
5. **三段式登录**: login1st/2nd/3rd 流程需抓包确认每段字段 (服务器已拒绝请求, 无法在线观察成功响应)
