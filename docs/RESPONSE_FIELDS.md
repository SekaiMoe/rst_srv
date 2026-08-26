# 客户端响应字段映射表 (SetJson/DTO 静态逆向)

> 来源: dump.cs DTO 属性 (IL2CPP 直映 JSON 字段) + SetJson 反汇编
> 本表是私服响应的**权威字段名**来源

## 通用信封

`{code, message, ...}` — code=200 成功 (AbstractApiManager.ProcessResponseData)

## account/login2nd, login3rd → UserModel (扁平)

```
id, name, lv, exp,
stamina, stamina_max, stamina_updated_at,        ← AP 系 (不是 ap!)
chara_box_length, acce_box_length,
tutorial, point_purchased, point_free,            ← 有偿石/无偿石 (不是 jewel!)
favorite_card_id, publish_card_id, point,         ← point=好友点
money,                                           ← 金币 (不是 coin!)
title_id, profile, purchase_price_month, unlock_event_story_item_num,
playerCards[]: PlayerCardModel                    ← 卡箱
units[]: UnitModel
cardEquipments[]: CardEquipmentModel
gachaConsumptionItems[]: GachaConsumptionItemModel
```

### PlayerCardModel (卡实例)
```
id, card_id (master id), lv, exp, acce_skill_pos, favorite,
created_at, created_at_int, break_limit_count
```

### UnitModel (编成)
```
id, name, leader, unit
```
### CardEquipmentModel (饰品装备)
```
player_card_id, equipments[], itemId
```

## gacha

**gacha/play → ResponseGachaPlay**: `gacha_id, day_free_count, quantity_limit_count, item_id_1, item_num_1, item_id_2, item_num_2`
**gacha/execute → ResponseGachaExecute**: `card_ids[] (master id 数组), id, day_free_count`

## livestage

**playgame → ResponseLivestagePlaygame**: `status, power, stamina, stamina_updated_at`
**finishgame → ResponseLivestageFinishgame**: `rewards, eventRewards, eventPoint, characterCollectionBefore`
**stonecontinue → ResponseLivestageStonecontinue**: `status, jewel, freeJewel`
**adscontinue → ResponseLivestageAdscontinue**: `status`

## story

**read → ResponseStoryRead**: `status, rewards`
**status → ResponseStoryStatus**: `status`
**unlock_event → ResponseStoryUnlockEvent**: `status`
**check_available → ResponseStoryCheckAvailable**: `is_ok`

## shop

**buy → ResponseShopBuy**: `point_purchased, point_free`
**buy_music → ResponseShopBuyMusic**: `point_purchased, point_free, music_id`
**buy_movie → ResponseShopBuyMovie**: `point_purchased, point_free, movie_id`
**receipt → ResponseShopReceipt**: `purchase_price_month, point_purchased, point_free`

## item

**friendpt → ResponseItemFriendpt**: `friend_pt, total_friend_pt`

## presentbox

**get → ResponsePresentBoxGet**: `charaBoxLengthBefore, charaBoxLength, acceBoxLengthBefore, acceBoxLength, isGetTitle, isGetMusic, exchangeMessage`
**get_all → ResponsePresentBoxGetAll**: 同上系

### PresentBoxModel (列表项)
```
id, item_kind, item_id, item_num, created_at, deadline, message
```

## achieve

**get_all → ResponseAchiveGetAll**: `achievement_id[]`
### AchievementModel: `id, progress`

## account

**handover_code → ResponseAccountHandoverCode**: `handover_code`
**delete → ResponseAccountDeletion**: `status`
**home → ResponseHomeInfo**: `achieve_num, present_num, friend_pt, total_friend_pt`

## event / ranking

**battle_finish → ResponseEventBattleFinish**: `rewards, eventRewards`
### EventStatusModel: `ranking, event_point, event_id, challenge_ticket`
### RankingPlayerModel: `music_id, score_easy/normal/hard/veryhard, score_tap_*`
### LiveStageModel (成绩): `id, stage_id, score, combo, clear`

## 其他 Model 字段速查

- FriendModel: id, user_id, name, card_id, lv, title_id, life, power1..3, unit_power, skill_id, profile, is_follow, is_follower
- StoryModel: isUnlock, isRead, kind, chapterId, isFirst, eventId, eventPoint
- GachaConsumptionItemModel: item_id, item_num
