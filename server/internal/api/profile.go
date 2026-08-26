package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ---- profile — 玩家档案 ----
//
// 新增玩家字段: ProfileText, TitleID, FavoriteCardID, PublishCardID
// (存档 JSON 向后兼容, 旧档缺字段零值即默认)

// profileSetTitle — 设置称号 (title 表 417 项)
func (s *Server) profileSetTitle(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "player not found"})
		return
	}
	titleID, _ := strconv.Atoi(c.PostForm("title_id"))
	for _, r := range s.Master["title"] {
		if rowInt(r, "id") == titleID {
			pl.TitleID = titleID
			_ = s.Players.Save(pl)
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok",
				"title_id": titleID, "name": rowStr(r, "name")})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 404, "message": "title not found"})
}

// profileTitlelist — 已获得称号 (纪念服: 全部开放)
func (s *Server) profileTitlelist(c *gin.Context) {
	type t struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Desc  string `json:"description"`
		Order int    `json:"order"`
	}
	out := []t{}
	for _, r := range s.Master["title"] {
		out = append(out, t{rowInt(r, "id"), rowStr(r, "name"), rowStr(r, "description"), rowInt(r, "order")})
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok",
		"titles": out, "count": len(out), "current": c.GetInt("title_id")})
}

// profileSetName — 改名
func (s *Server) profileSetName(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "player not found"})
		return
	}
	name := c.PostForm("name")
	if name == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "name required"})
		return
	}
	pl.Name = name
	_ = s.Players.Save(pl)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "name": name})
}

// profileSetProfile — 简介文本
func (s *Server) profileSetProfile(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "player not found"})
		return
	}
	pl.ProfileText = c.PostForm("profile")
	_ = s.Players.Save(pl)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "profile": pl.ProfileText})
}

// profileFavoriteCard — 最爱卡展示
func (s *Server) profileFavoriteCard(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "player not found"})
		return
	}
	cardID, _ := strconv.Atoi(c.PostForm("card_id"))
	if !pl.ownsCard(cardID) {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "card not owned"})
		return
	}
	pl.FavoriteCardID = cardID
	_ = s.Players.Save(pl)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "card_id": cardID})
}

// profilePublishCard — 公开卡展示
func (s *Server) profilePublishCard(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "player not found"})
		return
	}
	cardID, _ := strconv.Atoi(c.PostForm("card_id"))
	if cardID != 0 && !pl.ownsCard(cardID) {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "card not owned"})
		return
	}
	pl.PublishCardID = cardID
	_ = s.Players.Save(pl)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "card_id": cardID})
}

func (p *Player) ownsCard(id int) bool {
	for _, c := range p.Cards {
		if c.ID == id {
			return true
		}
	}
	return false
}

// ---- item/friendpt — 好友点兑换 (100pt = 1抽, 对应免费池 1913) ----

func (s *Server) itemFriendpt(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "player not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok",
		"friend_point": pl.FriendPoint, "cost": 100})
}

// itemLivestage — 演出道具列表
func (s *Server) itemLivestage(c *gin.Context) { s.itemList(c) }
