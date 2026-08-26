package api

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// card/grow — 卡牌强化
//
// masterdata 模型:
//
//	card.enhance_exp    素材卡提供的经验
//	card.enhance_price  强化费用基准
//	card.rarity 1..5    → levelTableCardR1..R5 升级经验曲线 (lv99 满)
//
// 表单: card_id=<实例id>, material_card_ids=<逗号分隔实例id列表>
func (s *Server) cardGrow(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	cardID, _ := strconv.Atoi(c.PostForm("card_id"))
	materialIDs := parseIntList(c.PostForm("material_card_ids"))

	// 找到目标卡
	var target *Card
	for i := range pl.Cards {
		if pl.Cards[i].ID == cardID {
			target = &pl.Cards[i]
			break
		}
	}
	if target == nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "card not found"})
		return
	}
	master := s.Gacha.Card[target.MasterID]
	if master == nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "master card not found"})
		return
	}
	rarity := rowInt(master, "rarity")

	// 素材: 校验持有 + 锁定, 累计经验
	materialSet := map[int]bool{}
	gainExp := 0
	for _, mid := range materialIDs {
		for i := range pl.Cards {
			if pl.Cards[i].ID == mid && mid != cardID {
				if pl.Cards[i].Lock == 1 {
					c.JSON(http.StatusOK, gin.H{"code": 400, "message": "material is locked", "card_id": mid})
					return
				}
				m := s.Gacha.Card[pl.Cards[i].MasterID]
				if m != nil {
					gainExp += rowInt(m, "enhance_exp")
				}
				materialSet[mid] = true
			}
		}
	}
	if len(materialSet) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "no materials"})
		return
	}

	// 费用: 每素材 enhance_price
	cost := rowInt(master, "enhance_price") * len(materialSet)
	if pl.Coin < cost {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "coin not enough", "coin": pl.Coin, "need": cost})
		return
	}
	pl.Coin -= cost

	// 移除素材
	kept := pl.Cards[:0]
	for _, cc := range pl.Cards {
		if !materialSet[cc.ID] {
			kept = append(kept, cc)
		}
	}
	pl.Cards = kept

	// 升级: levelTableCardR{rarity}
	table := s.Master["levelTableCardR"+strconv.Itoa(rarity)]
	beforeLv := target.Level
	target.Exp += gainExp
	for _, r := range table {
		if target.Exp >= rowInt(r, "exp") {
			lv := rowInt(r, "lv")
			if lv > target.Level {
				target.Level = lv
			}
		}
	}

	_ = s.Players.Save(pl)
	log.Printf("[grow] uuid=%s card#%d(rarity%d) +%dexp 素材%d张 Lv.%d→%d coin-%d",
		pl.UUID, cardID, rarity, gainExp, len(materialSet), beforeLv, target.Level, cost)

	// ResponseGrow: playerCardModel (PlayerCardModel 字段)
	card := cardJSON(*target)
	card["before_level"] = beforeLv
	card["gain_exp"] = gainExp
	c.JSON(http.StatusOK, gin.H{
		"code": 200, "message": "ok",
		"playerCardModel": card,
		"card":            card,
		"materials_used":  len(materialSet),
		"money":           pl.Coin,
	})
}

// card/lock — 锁定/解锁 (保护不被误合成/出售)
func (s *Server) cardLock(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	cardID, _ := strconv.Atoi(c.PostForm("card_id"))
	for i := range pl.Cards {
		if pl.Cards[i].ID == cardID {
			pl.Cards[i].Lock = 1 - pl.Cards[i].Lock
			lock := pl.Cards[i].Lock
			_ = s.Players.Save(pl)
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "card_id": cardID, "lock": lock})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 404, "message": "card not found"})
}

// card/sell — 出售 (sell_price 入手, 锁定卡不可)
func (s *Server) cardSell(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	ids := parseIntList(c.PostForm("card_ids"))
	gain := 0
	sold := 0
	idSet := map[int]bool{}
	for _, id := range ids {
		for i := range pl.Cards {
			if pl.Cards[i].ID == id && pl.Cards[i].Lock == 0 {
				if m := s.Gacha.Card[pl.Cards[i].MasterID]; m != nil {
					gain += rowInt(m, "sell_price")
					idSet[id] = true
					sold++
				}
			}
		}
	}
	if sold == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "no sellable cards"})
		return
	}
	kept := pl.Cards[:0]
	for _, cc := range pl.Cards {
		if !idSet[cc.ID] {
			kept = append(kept, cc)
		}
	}
	pl.Cards = kept
	pl.Coin += gain
	_ = s.Players.Save(pl)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "status": 1,
		"sold": sold, "gain_coin": gain, "coin": pl.Coin, "money": pl.Coin})
}

// deck/leader — 编成队长
func (s *Server) deckLeader(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	deckID, _ := strconv.Atoi(c.PostForm("deck_id"))
	// 客户端实际发送: deck_id + pos (REQUEST_FIELDS.md); pos=要设为队长的卡实例
	posID, _ := strconv.Atoi(c.PostForm("pos"))
	cardID, _ := strconv.Atoi(c.PostForm("card_id"))
	if cardID == 0 {
		cardID = posID
	}
	if deckID < 0 || deckID >= len(pl.Decks) {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "deck not found"})
		return
	}
	for len(pl.Decks[deckID]) < 5 {
		pl.Decks[deckID] = append(pl.Decks[deckID], 0)
	}
	pl.Decks[deckID][0] = cardID
	_ = s.Players.Save(pl)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "deck_id": deckID, "cards": pl.Decks[deckID]})
}

// deck/card — 编成卡位
func (s *Server) deckCard(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	deckID, _ := strconv.Atoi(c.PostForm("deck_id"))
	pos, _ := strconv.Atoi(c.PostForm("pos")) // 客户端实际字段 (REQUEST_FIELDS.md)
	if pos == 0 {
		pos, _ = strconv.Atoi(c.PostForm("deck_pos")) // 兼容
	}
	cardID, _ := strconv.Atoi(c.PostForm("card_id"))
	if deckID < 0 || deckID >= len(pl.Decks) {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "deck not found"})
		return
	}
	for len(pl.Decks[deckID]) < 5 {
		pl.Decks[deckID] = append(pl.Decks[deckID], 0)
	}
	if pos < 0 || pos >= 5 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "deck_pos out of range"})
		return
	}
	pl.Decks[deckID][pos] = cardID
	_ = s.Players.Save(pl)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "deck_id": deckID, "cards": pl.Decks[deckID]})
}

// ---- 工具 ----

func parseIntList(s string) []int {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "[]")
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		v, err := strconv.Atoi(strings.TrimSpace(p))
		if err == nil {
			out = append(out, v)
		}
	}
	return out
}
