package api

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ---- friend — 好友系统 (纪念服单机化: NPC = 角色) ----
//
// masterdata.character 73 名角色 (式宮舞菜/月坂紗由/KiRaRe...) 作为 NPC 好友。
// follow/unfollow 改为收藏; followlist = 已关注角色。

func (s *Server) friendFollow(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	friendID, _ := strconv.Atoi(c.PostForm("friend_id"))
	// friend_id 直接用 character_id
	found := false
	for _, r := range s.Master["character"] {
		if rowInt(r, "id") == friendID {
			found = true
			break
		}
	}
	if !found {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "character not found"})
		return
	}
	// 存到 items 里用特殊 key (不与道具冲突)
	key := "friend_" + strconv.Itoa(friendID)
	if pl.Items[key] == 1 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "already following"})
		return
	}
	pl.Items[key] = 1
	pl.FriendPoint += 10 // 关注送好友点
	_ = s.Players.Save(pl)
	name := ""
	for _, r := range s.Master["character"] {
		if rowInt(r, "id") == friendID {
			name = rowStr(r, "name")
			break
		}
	}
	// ResponseFollowFollower: status + 钱包
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "status": 1,
		"friend_id": friendID, "name": name,
		"is_follow": 1, "is_follower": 1,
		"friend_pt": pl.FriendPoint, "total_friend_pt": pl.FriendPoint})
}

func (s *Server) friendUnfollow(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	friendID, _ := strconv.Atoi(c.PostForm("friend_id"))
	key := "friend_" + strconv.Itoa(friendID)
	delete(pl.Items, key)
	_ = s.Players.Save(pl)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "friend_id": friendID})
}

// friendFollowlist — 已关注角色列表
func (s *Server) friendFollowlist(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	s.charList(c, pl, true)
}

// friendFollowerlist — 单机化: 全部角色 (视作互关)
func (s *Server) friendFollowerlist(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	s.charList(c, pl, false)
}

func (s *Server) charList(c *gin.Context, pl *Player, followingOnly bool) {
	unitName := map[int]string{}
	for _, r := range s.Master["unit"] {
		unitName[rowInt(r, "id")] = rowStr(r, "name")
	}
	// FriendModel: id,user_id,name,card_id,lv,title_id,life,power1..3,unit_power,skill_id,profile,is_follow,is_follower
	type fr struct {
		ID         int    `json:"id"`
		UserID     string `json:"user_id"`
		Name       string `json:"name"`
		CardID     int    `json:"card_id"`
		Lv         int    `json:"lv"`
		TitleID    int    `json:"title_id"`
		Life       int    `json:"life"`
		Power1     int    `json:"power1"`
		Power2     int    `json:"power2"`
		Power3     int    `json:"power3"`
		UnitPower  int    `json:"unit_power"`
		SkillID    int    `json:"skill_id"`
		Profile    string `json:"profile"`
		IsFollow   int    `json:"is_follow"`
		IsFollower int    `json:"is_follower"`
		UnitID     int    `json:"unit_id"`
		UnitName   string `json:"unit_name"`
	}
	// 每角色的代表卡 (masterdata 最强卡) 供展示
	bestCard := map[int]int{}
	cardLv := map[int]int{}
	for _, c := range s.Master["card"] {
		cid := rowInt(c, "character_id")
		if rowInt(c, "rarity") >= cardLv[cid] {
			cardLv[cid] = rowInt(c, "rarity")
			bestCard[cid] = rowInt(c, "id")
		}
	}
	out := []fr{}
	for _, r := range s.Master["character"] {
		id := rowInt(r, "id")
		key := "friend_" + strconv.Itoa(id)
		followed := pl.Items[key] == 1
		if followingOnly && !followed {
			continue
		}
		bc := bestCard[id]
		var m map[string]interface{}
		for _, c := range s.Master["card"] {
			if rowInt(c, "id") == bc {
				m = c
				break
			}
		}
		f := fr{
			ID: id, UserID: "npc-" + strconv.Itoa(id), Name: rowStr(r, "name"),
			CardID: bc, Lv: 99, Life: 99,
			UnitID: rowInt(r, "unit_id"), UnitName: unitName[rowInt(r, "unit_id")],
			Profile:  rowStr(r, "description"),
			IsFollow: boolInt(followed), IsFollower: 1,
		}
		if m != nil {
			f.Power1 = rowInt(m, "power1_max")
			f.Power2 = rowInt(m, "power2_max")
			f.Power3 = rowInt(m, "power3_max")
			f.UnitPower = f.Power1 + f.Power2 + f.Power3
		}
		out = append(out, f)
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok",
		"friends": out, "count": len(out), "follow_limit": 100})
}

// friendSearch — 角色名搜索
func (s *Server) friendSearch(c *gin.Context) {
	q := c.PostForm("name")
	type fr struct {
		ID   int    `json:"friend_id"`
		Name string `json:"name"`
		CV   string `json:"cv"`
	}
	out := []fr{}
	for _, r := range s.Master["character"] {
		name := rowStr(r, "name")
		if q == "" || containsFold(name, q) || containsFold(rowStr(r, "read"), q) {
			out = append(out, fr{rowInt(r, "id"), name, rowStr(r, "cv")})
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "friends": out, "count": len(out)})
}

func containsFold(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	sl, subl := lower(s), lower(sub)
	for i := 0; i+len(subl) <= len(sl); i++ {
		if sl[i:i+len(subl)] == subl {
			return true
		}
	}
	return false
}

func lower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 32
		}
	}
	return string(b)
}

// friendLivestagelist — NPC 的演出列表 (单机化: 返回自己的成绩曲)
func (s *Server) friendLivestagelist(c *gin.Context) { s.rankingPrivate(c) }

// friendUnfollower — 同 unfollow
func (s *Server) friendUnfollower(c *gin.Context) { s.friendUnfollow(c) }

// ---- presentbox — 礼物箱 ----
//
// 单机化: 服务器事件推送不可用, 把"待领取"存在 items 的 present_ 前缀.
// 提供 list/get/get_all; 运营补偿类礼物可手工注入 data/players/<uuid>.json

func (s *Server) presentboxList(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	// PresentBoxModel: id,item_kind,item_id,item_num,created_at,deadline,message
	presents := []gin.H{}
	idx := 0
	for key, n := range pl.Items {
		if len(key) > 8 && key[:8] == "present_" {
			itemID, _ := strconv.Atoi(key[8:])
			idx++
			kind := 0
			var m map[string]interface{}
			for _, r := range s.Master["item"] {
				if rowInt(r, "id") == itemID {
					m = r
					break
				}
			}
			if m != nil {
				kind = rowInt(m, "kind_id")
			}
			presents = append(presents, gin.H{
				"id": idx, "item_kind": kind, "item_id": itemID, "item_num": n,
				"created_at": pl.CreatedAt.Format("2006-01-02 15:04:05"),
				"deadline":   "2030-12-31 23:59:59",
				"message":    "運営からのプレゼントです",
			})
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "status": 1,
		"presents": presents, "count": len(presents)})
}

func (s *Server) presentboxGet(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	itemID, _ := strconv.Atoi(c.PostForm("item_id"))
	key := "present_" + strconv.Itoa(itemID)
	n, ok := pl.Items[key]
	if !ok || n <= 0 {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "present not found"})
		return
	}
	delete(pl.Items, key)
	// 转为真实道具
	pl.Items[strconv.Itoa(itemID)] += n
	_ = s.Players.Save(pl)
	log.Printf("[present] uuid=%s 领取 item=%d x%d", pl.UUID, itemID, n)
	// ResponsePresentBoxGet: box 长度 + isGetTitle/isGetMusic
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok",
		"item_id": itemID, "num": n,
		"charaBoxLengthBefore": len(pl.Cards) - 1, "charaBoxLength": len(pl.Cards),
		"acceBoxLengthBefore": 250, "acceBoxLength": 250,
		"isGetTitle": 0, "isGetMusic": 0, "exchangeMessage": "",
		"items": pl.Items})
}

func (s *Server) presentboxGetAll(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	got := 0
	for key, n := range pl.Items {
		if len(key) > 8 && key[:8] == "present_" && n > 0 {
			itemID, _ := strconv.Atoi(key[8:])
			delete(pl.Items, key)
			pl.Items[strconv.Itoa(itemID)] += n
			got++
		}
	}
	_ = s.Players.Save(pl)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "got": got, "items": pl.Items})
}

func (s *Server) presentboxLog(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "logs": []string{}})
}
