package api

import "github.com/gin-gonic/gin"

// ---- 客户端字段映射助手 (docs/RESPONSE_FIELDS.md) ----
//
// IL2CPP DTO 属性名 = JSON 字段名直映:
//
//	UserModel:      lv/money/point_free/point_purchased/stamina (不是 level/coin/jewel/ap)
//	PlayerCardModel: card_id/lv (不是 master_id/level)

// userJSON — UserModel 扁平结构 (login2nd/3rd 响应主体)
func userJSON(pl *Player) gin.H {
	return gin.H{
		"id":                          pl.UUID,
		"name":                        pl.Name,
		"lv":                          pl.Level,
		"exp":                         pl.Exp,
		"stamina":                     pl.Ap,
		"stamina_max":                 pl.ApMax,
		"stamina_updated_at":          pl.ApUpdatedAt.Format("2006-01-02 15:04:05"),
		"chara_box_length":            len(pl.Cards),
		"acce_box_length":             250,
		"tutorial":                    1,
		"point_purchased":             0, // 有偿石 (纪念服不区分)
		"point_free":                  pl.Jewel,
		"point":                       pl.FriendPoint,
		"money":                       pl.Coin,
		"favorite_card_id":            pl.FavoriteCardID,
		"publish_card_id":             pl.PublishCardID,
		"title_id":                    pl.TitleID,
		"profile":                     pl.ProfileText,
		"unlock_event_story_item_num": 99,
	}
}

// cardJSON — PlayerCardModel (卡实例)
func cardJSON(c Card) gin.H {
	return gin.H{
		"id":       c.ID,
		"card_id":  c.MasterID, // master id
		"lv":       c.Level,
		"exp":      c.Exp,
		"favorite": 0,
	}
}

// playerCardsJSON — 卡箱数组
func playerCardsJSON(pl *Player) []gin.H {
	out := make([]gin.H, 0, len(pl.Cards))
	for _, c := range pl.Cards {
		out = append(out, cardJSON(c))
	}
	return out
}

// walletJSON — 钱包字段 (多数响应附带)
func walletJSON(pl *Player) gin.H {
	return gin.H{
		"point_free":      pl.Jewel,
		"point_purchased": 0,
		"money":           pl.Coin,
		"point":           pl.FriendPoint,
		"stamina":         pl.Ap,
		"stamina_max":     pl.ApMax,
		"lv":              pl.Level,
		"exp":             pl.Exp,
	}
}

// deckPower — 编成战力 (编成1的卡组 power 合计, 简化)
func (s *Server) deckPower(pl *Player) int {
	total := 0
	if len(pl.Decks) == 0 {
		return total
	}
	for _, cid := range pl.Decks[0] {
		if cid == 0 {
			continue
		}
		for _, c := range pl.Cards {
			if c.ID == cid {
				if m := s.Gacha.Card[c.MasterID]; m != nil {
					total += rowInt(m, "power1_max") + rowInt(m, "power2_max") + rowInt(m, "power3_max")
				}
				break
			}
		}
	}
	return total
}
