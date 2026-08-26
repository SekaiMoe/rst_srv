package api

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ---- acce — 饰品卡 (card.kind=2, 978 张) ----
//
// 饰品与角色卡同用 Card 实例模型; "装备" = 角色卡实例上记录饰品实例 id 列表.
// Player 新增: AcceSlots map[int][]int (角色卡实例id -> 已装备饰品实例id, 上限4)

const ACCE_SLOT_MAX = 4

// acceEquip(card/acce) — 饰品装备/更换
//
// 表单: card_id=<角色卡实例> acce_ids=<逗号分隔饰品实例列表, 空=卸下>
func (s *Server) cardAcce(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "player not found"})
		return
	}
	cardID, _ := strconv.Atoi(c.PostForm("card_id"))
	acceIDs := parseIntList(c.PostForm("acce_ids"))
	if !pl.ownsCard(cardID) {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "card not found"})
		return
	}
	if len(acceIDs) > ACCE_SLOT_MAX {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "too many acce", "max": ACCE_SLOT_MAX})
		return
	}
	// 校验: 饰品实例必须是 kind=2 且未被其他卡装备
	for _, aid := range acceIDs {
		var inst *Card
		for i := range pl.Cards {
			if pl.Cards[i].ID == aid {
				inst = &pl.Cards[i]
				break
			}
		}
		if inst == nil {
			c.JSON(http.StatusOK, gin.H{"code": 404, "message": "acce not found", "acce_id": aid})
			return
		}
		m := s.Gacha.Card[inst.MasterID]
		if m == nil || rowInt(m, "kind") != 2 {
			c.JSON(http.StatusOK, gin.H{"code": 400, "message": "not an acce card", "acce_id": aid})
			return
		}
	}
	// 从其他卡卸下
	for cid, slots := range pl.AcceSlots {
		if cid == cardID {
			continue
		}
		kept := slots[:0]
		for _, aid := range slots {
			used := false
			for _, want := range acceIDs {
				if aid == want {
					used = true
					break
				}
			}
			if !used {
				kept = append(kept, aid)
			}
		}
		pl.AcceSlots[cid] = kept
	}
	pl.AcceSlots[cardID] = acceIDs
	_ = s.Players.Save(pl)
	log.Printf("[acce] uuid=%s card#%d 装备%v", pl.UUID, cardID, acceIDs)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok",
		"card_id": cardID, "acce_ids": acceIDs})
}

// acceGrow — 饰品强化 (复用卡牌强化: kind=2 曲线同 R1..R5)
func (s *Server) acceGrow(c *gin.Context) { s.cardGrow(c) }

// acceLock — 饰品锁定
func (s *Server) acceLock(c *gin.Context) { s.cardLock(c) }

// acceSell — 饰品出售
func (s *Server) acceSell(c *gin.Context) { s.cardSell(c) }

// cardAcceSkill(card//acce_skill) — 饰品技能查看 (静态数据)
func (s *Server) cardAcceSkill(c *gin.Context) {
	cardID, _ := strconv.Atoi(c.PostForm("card_id"))
	m := s.Gacha.Card[cardID]
	if m == nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "card not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok",
		"card_id": cardID, "name": rowStr(m, "name"),
		"description": rowStr(m, "description")})
}

// ---- card/batch + deck/batch + deck/disband + deck/rename — 批量编成 ----

// deckBatch — 整卡组保存
//
// 表单: deck_id=<0..4> card_ids=<逗号分隔5个实例id>
func (s *Server) deckBatch(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "player not found"})
		return
	}
	deckID, _ := strconv.Atoi(c.PostForm("deck_id"))
	cardIDs := parseIntList(c.PostForm("card_ids"))
	if deckID < 0 || deckID >= len(pl.Decks) {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "deck not found"})
		return
	}
	if len(cardIDs) > 5 {
		cardIDs = cardIDs[:5]
	}
	for len(cardIDs) < 5 {
		cardIDs = append(cardIDs, 0)
	}
	for _, cid := range cardIDs {
		if cid != 0 && !pl.ownsCard(cid) {
			c.JSON(http.StatusOK, gin.H{"code": 404, "message": "card not owned", "card_id": cid})
			return
		}
	}
	pl.Decks[deckID] = cardIDs
	_ = s.Players.Save(pl)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "deck_id": deckID, "cards": cardIDs})
}

// cardBatch — 批量卡操作结果查询 (返回卡箱)
func (s *Server) cardBatch(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "player not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok",
		"cards": pl.Cards, "count": len(pl.Cards), "acce_slots": pl.AcceSlots})
}

// deckDisband — 清空卡组
func (s *Server) deckDisband(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "player not found"})
		return
	}
	deckID, _ := strconv.Atoi(c.PostForm("deck_id"))
	if deckID < 0 || deckID >= len(pl.Decks) {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "deck not found"})
		return
	}
	pl.Decks[deckID] = []int{0, 0, 0, 0, 0}
	_ = s.Players.Save(pl)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "deck_id": deckID})
}

// deckRename — 卡组命名 (纪念服: 存 items 元数据)
func (s *Server) deckRename(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "player not found"})
		return
	}
	deckID, _ := strconv.Atoi(c.PostForm("deck_id"))
	name := c.PostForm("name")
	if deckID < 0 || deckID >= len(pl.Decks) {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "deck not found"})
		return
	}
	pl.Items["deck_name_"+strconv.Itoa(deckID)] = 0
	pl.DeckNames[deckID] = name
	_ = s.Players.Save(pl)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "deck_id": deckID, "name": name})
}
