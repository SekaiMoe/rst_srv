package api

import (
	"log"
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ---- shop — 商店 (金币购物, 排除氪金项) ----
//
// masterdata:
//
//	shopItemPack: 商品包 (name/price/limit_count/is_paid 0=金币 1=氪金)
//	shopItem:     包内明细 (item_kind/item_id/num)
//
// 纪念服语义: is_paid=0 的商品可用金币购买; is_paid=1 (氪金) 直接标记为已购(免费)

func (s *Server) shopItemlist(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "player not found"})
		return
	}
	packs := s.Master["shopItemPack"]
	itemsOf := map[int][]map[string]interface{}{} // pack_id -> contents
	for _, r := range s.Master["shopItem"] {
		itemsOf[rowInt(r, "shop_item_pack_id")] = append(itemsOf[rowInt(r, "shop_item_pack_id")], r)
	}
	type pack struct {
		ID         int     `json:"id"`
		Name       string  `json:"name"`
		Price      int     `json:"price"`
		IsPaid     int     `json:"is_paid"`
		LimitCount int     `json:"limit_count"`
		Contents   []gin.H `json:"contents"`
	}
	sort.Slice(packs, func(i, j int) bool { return rowInt(packs[i], "order") < rowInt(packs[j], "order") })
	out := []pack{}
	for _, p := range packs {
		id := rowInt(p, "id")
		contents := []gin.H{}
		for _, it := range itemsOf[id] {
			contents = append(contents, gin.H{
				"item_kind": rowInt(it, "item_kind"), "item_id": rowInt(it, "item_id"),
				"num": rowInt(it, "num"),
			})
		}
		out = append(out, pack{
			ID: id, Name: rowStr(p, "name"), Price: rowInt(p, "price"),
			IsPaid: rowInt(p, "is_paid"), LimitCount: rowInt(p, "limit_count"),
			Contents: contents,
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok",
		"packs": out, "count": len(out), "coin": pl.Coin, "jewel": pl.Jewel})
}

func (s *Server) shopPurchaselist(c *gin.Context) { s.shopItemlist(c) }

// shopBuy — 购买
func (s *Server) shopBuy(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "player not found"})
		return
	}
	packID, _ := strconv.Atoi(c.PostForm("shop_item_pack_id"))
	var pack map[string]interface{}
	for _, p := range s.Master["shopItemPack"] {
		if rowInt(p, "id") == packID {
			pack = p
			break
		}
	}
	if pack == nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "pack not found"})
		return
	}
	price := rowInt(pack, "price")
	isPaid := rowInt(pack, "is_paid") == 1

	// 氪金商品: 纪念服直接免费发放
	if !isPaid {
		if pl.Coin < price {
			c.JSON(http.StatusOK, gin.H{"code": 400, "message": "coin not enough",
				"coin": pl.Coin, "need": price})
			return
		}
		pl.Coin -= price
	}
	// 发放包内全部内容
	granted := []gin.H{}
	for _, it := range s.Master["shopItem"] {
		if rowInt(it, "shop_item_pack_id") == packID {
			kind, itemID, num := rowInt(it, "item_kind"), rowInt(it, "item_id"), rowInt(it, "num")
			s.grantItem(pl, kind, itemID, num)
			granted = append(granted, gin.H{"item_kind": kind, "item_id": itemID, "num": num})
		}
	}
	_ = s.Players.Save(pl)
	log.Printf("[shop] uuid=%s pack=%d price=%d paid=%v 获得%d项", pl.UUID, packID, price, isPaid, len(granted))
	// ResponseShopBuy: point_purchased, point_free
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok",
		"shop_item_pack_id": packID, "granted": granted,
		"point_purchased": 0, "point_free": pl.Jewel, "money": pl.Coin})
}

func (s *Server) shopBuyMusic(c *gin.Context) { s.shopBuy(c) }
func (s *Server) shopBuyMovie(c *gin.Context) { s.shopBuy(c) }
func (s *Server) shopLog(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "logs": []string{}})
}

// shopReceiptcheck — 氪金收据校验 (纪念服: 直接通过)
func (s *Server) shopReceiptcheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "valid": true})
}

// repayment/create — 停服退款 (纪念服: 记录后返回受理)
func (s *Server) repaymentCreate(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "accepted"})
}

// ---- shop/exchange — 活动兑换商店 ----
//
// masterdata:
//
//	campaignExchangeInfo:      兑换活动 (title/order/时间窗)
//	campaignExchangeItem:      可兑换条目 (campaign_id/item_kind/item_id/limit_count)
//	                             item_group_id → campaignExchangeItemGroup = 消耗的货币
//	campaignExchangeItemGroup: (item_kind=12チケット等, item_id, item_num)
//
// 纪念服: 忽略时间窗全部开放

func (s *Server) shopExchangeList(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "player not found"})
		return
	}
	// 货币组: group_id -> 消耗
	groupCost := map[int]map[string]int{}
	for _, r := range s.Master["campaignExchangeItemGroup"] {
		groupCost[rowInt(r, "group_id")] = map[string]int{
			"item_kind": rowInt(r, "item_kind"), "item_id": rowInt(r, "item_id"),
			"item_num": rowInt(r, "item_num"),
		}
	}
	itemName := map[int]string{}
	for _, r := range s.Master["item"] {
		itemName[rowInt(r, "id")] = rowStr(r, "name")
	}
	type ex struct {
		ID       int    `json:"id"`
		Campaign int    `json:"campaign_id"`
		Title    string `json:"campaign_title"`
		ItemKind int    `json:"item_kind"`
		ItemID   int    `json:"item_id"`
		ItemName string `json:"item_name"`
		ItemNum  int    `json:"item_num"`
		CostKind int    `json:"cost_item_kind"`
		CostID   int    `json:"cost_item_id"`
		CostName string `json:"cost_item_name"`
		CostNum  int    `json:"cost_item_num"`
		Limit    int    `json:"limit_count"`
	}
	infos := map[int]map[string]interface{}{}
	for _, r := range s.Master["campaignExchangeInfo"] {
		infos[rowInt(r, "id")] = r
	}
	out := []ex{}
	for _, r := range s.Master["campaignExchangeItem"] {
		cid := rowInt(r, "campaign_id")
		g := groupCost[rowInt(r, "item_group_id")]
		if g == nil {
			continue
		}
		title := ""
		if info := infos[cid]; info != nil {
			title = rowStr(info, "title")
		}
		out = append(out, ex{
			ID: rowInt(r, "id"), Campaign: cid, Title: title,
			ItemKind: rowInt(r, "item_kind"), ItemID: rowInt(r, "item_id"),
			ItemName: itemName[rowInt(r, "item_id")], ItemNum: rowInt(r, "item_num"),
			CostKind: g["item_kind"], CostID: g["item_id"],
			CostName: itemName[g["item_id"]], CostNum: g["item_num"],
			Limit: rowInt(r, "limit_count"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Campaign < out[j].Campaign })
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok",
		"exchanges": out, "count": len(out), "items": pl.Items})
}

// shopExchange — 兑换
func (s *Server) shopExchange(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "player not found"})
		return
	}
	exID, _ := strconv.Atoi(c.PostForm("campaign_exchange_item_id"))
	num, _ := strconv.Atoi(c.PostForm("num"))
	if num <= 0 {
		num = 1
	}
	var item map[string]interface{}
	for _, r := range s.Master["campaignExchangeItem"] {
		if rowInt(r, "id") == exID {
			item = r
			break
		}
	}
	if item == nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "exchange item not found"})
		return
	}
	// 查消耗
	var cost map[string]interface{}
	gid := rowInt(item, "item_group_id")
	for _, r := range s.Master["campaignExchangeItemGroup"] {
		if rowInt(r, "group_id") == gid {
			cost = r
			break
		}
	}
	if cost == nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "cost group missing"})
		return
	}
	costID, costNum := rowInt(cost, "item_id"), rowInt(cost, "item_num")*num
	key := strconv.Itoa(costID)
	if pl.Items[key] < costNum {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "currency not enough",
			"have": pl.Items[key], "need": costNum})
		return
	}
	pl.Items[key] -= costNum
	if pl.Items[key] == 0 {
		delete(pl.Items, key)
	}
	s.grantItem(pl, rowInt(item, "item_kind"), rowInt(item, "item_id"), rowInt(item, "item_num")*num)
	_ = s.Players.Save(pl)
	log.Printf("[exchange] uuid=%s ex=%d x%d 消耗item%d x%d", pl.UUID, exID, num, costID, costNum)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok",
		"got_kind": rowInt(item, "item_kind"), "got_id": rowInt(item, "item_id"),
		"got_num":      rowInt(item, "item_num") * num,
		"cost_item_id": costID, "cost_num": costNum, "items": pl.Items})
}

// ---- achieve — 成就 ----
//
// masterdata.achievement: condition_kind + condition_op + condition_value
// condition_kind: 1=? 2=? 3=? 4=ライブ成功回数(样例) ...
// 纪念服: 只自动检查可从本地统计得出的 (play 次数/剧情数/卡数), 其余手动领取?

func (s *Server) achieveList(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "player not found"})
		return
	}
	// 本地统计
	totalPlays, totalStories := 0, 0
	for _, bs := range pl.BestScores {
		totalPlays += bs.Plays
	}
	for range pl.ReadStories {
		totalStories++
	}
	stats := map[int]int{
		4: totalPlays, // ライブ成功回数
		1: totalStories,
		2: len(pl.Cards),
	}
	type ach struct {
		ID         int    `json:"id"`
		Name       string `json:"name"`
		Kind       int    `json:"achievement_kind"`
		CondKind   int    `json:"condition_kind"`
		CondValue  int    `json:"condition_value"`
		Current    int    `json:"current"`
		Achieved   bool   `json:"achieved"`
		RewardKind int    `json:"reward_item_kind"`
		RewardID   int    `json:"reward_item_id"`
		RewardNum  int    `json:"reward_item_num"`
	}
	out := []ach{}
	for _, r := range s.Master["achievement"] {
		ck := rowInt(r, "condition_kind")
		cur, ok := stats[ck]
		val := rowInt(r, "condition_value")
		achieved := ok && cur >= val
		out = append(out, ach{
			ID: rowInt(r, "id"), Name: rowStr(r, "name"),
			Kind: rowInt(r, "achievement_kind"), CondKind: ck, CondValue: val,
			Current: cur, Achieved: achieved,
			RewardKind: rowInt(r, "item_kind"), RewardID: rowInt(r, "item_id"), RewardNum: rowInt(r, "item_num"),
		})
	}
	achievedCount := 0
	for _, a := range out {
		if a.Achieved {
			achievedCount++
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok",
		"achievements": out, "count": len(out), "achieved": achievedCount})
}

func (s *Server) achieveGetAll(c *gin.Context) { s.achieveGet(c, true) }

func (s *Server) achieveGet(c *gin.Context, all bool) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "player not found"})
		return
	}
	achID, _ := strconv.Atoi(c.PostForm("achievement_id"))
	totalPlays := 0
	for _, bs := range pl.BestScores {
		totalPlays += bs.Plays
	}
	stats := map[int]int{4: totalPlays, 1: len(pl.ReadStories), 2: len(pl.Cards)}
	claimed := 0
	rewards := []gin.H{}
	for _, r := range s.Master["achievement"] {
		id := rowInt(r, "id")
		if !all && id != achID {
			continue
		}
		ck := rowInt(r, "condition_kind")
		cur, ok := stats[ck]
		if !ok || cur < rowInt(r, "condition_value") {
			continue
		}
		// 领取标记: items 里 achieve_<id>
		key := "achieve_" + strconv.Itoa(id)
		if pl.Items[key] == 1 {
			continue
		}
		pl.Items[key] = 1
		kind, itemID, num := rowInt(r, "item_kind"), rowInt(r, "item_id"), rowInt(r, "item_num")
		s.grantItem(pl, kind, itemID, num)
		rewards = append(rewards, gin.H{"achievement_id": id,
			"item_kind": kind, "item_id": itemID, "num": num})
		claimed++
	}
	_ = s.Players.Save(pl)
	log.Printf("[achieve] uuid=%s 领取%d项", pl.UUID, claimed)
	// ResponseAchiveGetAll: achievement_id[]
	ids := []int{}
	for _, r := range rewards {
		ids = append(ids, r["achievement_id"].(int))
	}
	body := walletJSON(pl)
	body["code"] = 200
	body["message"] = "ok"
	body["achievement_id"] = ids
	body["rewards"] = rewards
	c.JSON(http.StatusOK, body)
}

// achieveStock — 达成状态
func (s *Server) achieveStock(c *gin.Context) { s.achieveList(c) }
