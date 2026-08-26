package api

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// gacha 实现 — 依据 masterdata 数据模型:
//
//	gacha          卡池 (banner): id/name/group_id/gacha_kind/day_free_count/is_hide/start_date/end_date
//	gachaDetail    每池明细: 单抽(gacha_count=1) / 十连(=10), item_id_1/2 + item_num 为消耗
//	gachaLot       概率表: gacha_detail_id -> {ratio, gacha_block_id} 按 ratio 加权
//	gachaBlockDetail  block -> card_id 列表 (block 内均匀分布)
//	gachaGroup     分组展示: 1ジュエル 2チケット 3期間限定 4無料 5コラボ 6無料10連 7おさらい
//
// 纪念服语义: 忽略 start/end_date (全部卡池常驻可玩), is_hide=1 仍隐藏

// ---- gacha/list ----

func (s *Server) gachaList(c *gin.Context) {
	gi := s.Gacha
	groups := map[int]string{}
	for _, r := range s.Master["gachaGroup"] {
		groups[rowInt(r, "id")] = rowStr(r, "name")
	}

	type banner struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		SubName   string `json:"sub_name"`
		GroupID   int    `json:"group_id"`
		GroupName string `json:"group_name"`
		GachaKind int    `json:"gacha_kind"`
		DayFree   int    `json:"day_free_count"`
		Order     int    `json:"order"`
	}
	out := make([]banner, 0, len(gi.Gacha))
	for id, r := range gi.Gacha {
		if rowInt(r, "is_hide") == 1 {
			continue
		}
		out = append(out, banner{
			ID: id, Name: rowStr(r, "name"), SubName: rowStr(r, "sub_name"),
			GroupID: rowInt(r, "group_id"), GroupName: groups[rowInt(r, "group_id")],
			GachaKind: rowInt(r, "gacha_kind"), DayFree: rowInt(r, "day_free_count"),
			Order: rowInt(r, "order"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Order < out[j].Order })
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "gachas": out})
}

// ---- gacha/play: 池详情 (消耗/免费状态) ----

func (s *Server) gachaPlay(c *gin.Context) {
	gachaID, _ := strconv.Atoi(c.PostForm("gacha_id"))
	gi := s.Gacha
	g, ok := gi.Gacha[gachaID]
	if !ok {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "gacha not found"})
		return
	}
	pl := s.playerOf(c)

	type detail struct {
		ID          int  `json:"id"`
		GachaCount  int  `json:"gacha_count"`
		ItemID1     int  `json:"item_id_1"`
		ItemID2     int  `json:"item_id_2"`
		ItemNum     int  `json:"item_num"`
		IsFreeToday bool `json:"is_free_today"`
	}
	details := make([]detail, 0)
	for _, d := range gi.Details[gachaID] {
		dd := detail{
			ID: rowInt(d, "id"), GachaCount: rowInt(d, "gacha_count"),
			ItemID1: rowInt(d, "item_id_1"), ItemID2: rowInt(d, "item_id_2"),
			ItemNum:     rowInt(d, "item_num"),
			IsFreeToday: s.freeAvailable(pl, gachaID, rowInt(d, "gacha_count")),
		}
		details = append(details, dd)
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 200, "message": "ok",
		"gacha_id": gachaID, "name": rowStr(g, "name"),
		"details": details,
		"jewel":   pl.Jewel,
	})
}

// ---- gacha/execute: 真实抽卡 ----
//
// 表单: gacha_detail_id=<明细id>
// 逻辑: 免费日次可用且当日未用 -> 免费; 否则扣 item_num 石头 (item_id_2=901 无偿优先)
// 概率: gachaLot 按 ratio 加权 -> block 内均匀选卡 -> 发放卡实例

func (s *Server) gachaExecute(c *gin.Context) {
	detailID, _ := strconv.Atoi(c.PostForm("gacha_detail_id"))
	gi := s.Gacha
	d, ok := gi.Detail[detailID]
	if !ok {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "gacha_detail not found"})
		return
	}
	gachaID := rowInt(d, "gacha_id")
	count := rowInt(d, "gacha_count")
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}

	// ---- 消耗结算 ----
	isFree := s.freeAvailable(pl, gachaID, count)
	cost := rowInt(d, "item_num")
	if isFree {
		s.markFree(pl, gachaID)
	} else {
		if pl.Jewel < cost {
			c.JSON(http.StatusOK, gin.H{"code": 400, "message": "jewel not enough",
				"jewel": pl.Jewel, "need": cost})
			return
		}
		pl.Jewel -= cost
	}

	// ---- 抽取 ----
	newCards := s.rollGacha(pl, detailID, count)
	if err := s.Players.Save(pl); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	log.Printf("[gacha] uuid=%s detail=%d free=%v cost=%d 获得%v",
		pl.UUID, detailID, isFree, cost, cardIDs(newCards))

	// ResponseGachaExecute: card_ids + id + day_free_count
	c.JSON(http.StatusOK, gin.H{
		"code": 200, "message": "ok",
		"id":             detailID,
		"card_ids":       cardIDs(newCards), // master id 数组
		"day_free_count": dayFreeOf(s, gachaID),
		"playerCards":    cardsToJSON(newCards),
		"point_free":     pl.Jewel,
		"money":          pl.Coin,
	})
}

// ---- 内部: 概率引擎 ----

func (s *Server) rollGacha(pl *Player, detailID, count int) []Card {
	gi := s.Gacha
	lots := gi.Lots[detailID]
	total := 0
	for _, l := range lots {
		total += rowInt(l, "ratio")
	}
	newCards := make([]Card, 0, count)
	for i := 0; i < count; i++ {
		// 1) 加权选 block
		if total == 0 || len(lots) == 0 {
			break
		}
		pick := rand.Intn(total)
		var blockID int
		for _, l := range lots {
			pick -= rowInt(l, "ratio")
			if pick < 0 {
				blockID = rowInt(l, "gacha_block_id")
				break
			}
		}
		// 2) block 内均匀选卡
		cards := gi.Block[blockID]
		if len(cards) == 0 {
			continue
		}
		masterID := cards[rand.Intn(len(cards))]
		card := Card{ID: s.nextCardID(pl), MasterID: masterID, Level: 1}
		pl.Cards = append(pl.Cards, card)
		newCards = append(newCards, card)
	}
	return newCards
}

func (s *Server) nextCardID(pl *Player) int {
	max := 0
	for _, c := range pl.Cards {
		if c.ID > max {
			max = c.ID
		}
	}
	return max + 1
}

func cardIDs(cards []Card) []int {
	ids := make([]int, len(cards))
	for i, c := range cards {
		ids[i] = c.MasterID
	}
	return ids
}

// ---- 内部: 每日免费 ----

func (s *Server) freeAvailable(pl *Player, gachaID, count int) bool {
	if pl == nil {
		return false
	}
	g := s.Gacha.Gacha[gachaID]
	if g == nil || rowInt(g, "day_free_count") < 1 || count != 1 {
		return false
	}
	today := time.Now().Format("2006-01-02")
	return pl.GachaFree[fmt.Sprintf("%d", gachaID)] != today
}

func (s *Server) markFree(pl *Player, gachaID int) {
	pl.GachaFree[fmt.Sprintf("%d", gachaID)] = time.Now().Format("2006-01-02")
}

// ---- playerOf: 从请求上下文取玩家 ----

func (s *Server) playerOf(c *gin.Context) *Player {
	uuid, ok := c.Get("uuid")
	if !ok {
		return nil
	}
	pl, err := s.Players.Load(uuid.(string))
	if err != nil {
		return nil
	}
	return pl
}

func dayFreeOf(s *Server, gachaID int) int {
	if g := s.Gacha.Gacha[gachaID]; g != nil {
		return rowInt(g, "day_free_count")
	}
	return 0
}

func cardsToJSON(cards []Card) []gin.H {
	out := make([]gin.H, 0, len(cards))
	for _, c := range cards {
		out = append(out, cardJSON(c))
	}
	return out
}
