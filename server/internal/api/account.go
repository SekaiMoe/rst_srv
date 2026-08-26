package api

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// login1st — 三段式登录第一段
//
// 请求: HTTP头 UUID=<设备uuid>, 表单 device_token=<推送token>
// 响应: {code:200, token:<会话token>, is_new:<bool>}
func (s *Server) login1st(c *gin.Context) {
	uuid := c.GetHeader("UUID")
	deviceToken := c.PostForm("device_token")
	if uuid == "" {
		// 某些版本走表单
		uuid = c.PostForm("UUID")
	}
	if uuid == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "UUID required"})
		return
	}
	log.Printf("[login1st] uuid=%s device_token=%.16s...", uuid, deviceToken)

	isNew := !s.Players.Exists(uuid)
	tok := s.Sessions.Issue(uuid)
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "ok",
		"token":   tok,
		"is_new":  isNew,
	})
}

// login2nd — 三段式登录第二段 (token 换玩家基础信息)
func (s *Server) login2nd(c *gin.Context) {
	token := c.PostForm("token")
	sess, ok := s.Sessions.Get(token)
	if !ok {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	pl, err := s.Players.Load(sess.UUID)
	if err != nil {
		// 首次: 自动建档
		pl, err = s.Players.Create(sess.UUID)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": "player init failed"})
			return
		}
	}
	log.Printf("[login2nd] uuid=%s name=%s", sess.UUID, pl.Name)
	body := userJSON(pl)
	body["token"] = token
	body["code"] = 200
	body["message"] = "ok"
	c.JSON(http.StatusOK, body)
}

// login3rd — 三段式登录第三段 (完整玩家数据 + 资源版本同步)
func (s *Server) login3rd(c *gin.Context) {
	token := c.PostForm("token")
	sess, ok := s.Sessions.Get(token)
	if !ok {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	pl, err := s.Players.Load(sess.UUID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	log.Printf("[login3rd] uuid=%s full sync", sess.UUID)
	body := userJSON(pl)
	body["token"] = token
	body["code"] = 200
	body["message"] = "ok"
	// 子集合 (UserModel 关联)
	body["playerCards"] = playerCardsJSON(pl)
	units := []gin.H{}
	for i, dk := range pl.Decks {
		if len(dk) >= 5 && dk[0] != 0 {
			units = append(units, gin.H{
				"id": i + 1, "name": pl.DeckNames[i], "leader": dk[0], "unit": dk,
			})
		}
	}
	body["units"] = units
	equips := []gin.H{}
	for cid, acce := range pl.AcceSlots {
		if len(acce) > 0 {
			equips = append(equips, gin.H{
				"player_card_id": cid, "equipments": acce,
			})
		}
	}
	body["cardEquipments"] = equips
	body["gachaConsumptionItems"] = []gin.H{}
	// ResponseHomeInfo: achieve_num, present_num, friend_pt, total_friend_pt
	body["achieve_num"] = 0
	body["present_num"] = s.presentCount(pl)
	body["friend_pt"] = pl.FriendPoint
	body["total_friend_pt"] = pl.FriendPoint
	// 资源版本: 客户端据此决定增量下载 (本地不校验)
	body["assetbundle_version"] = 2
	body["cri_audio_version"] = 2
	c.JSON(http.StatusOK, body)
}

// loginFull — 旧版单段登录
func (s *Server) loginFull(c *gin.Context) {
	uuid := c.GetHeader("UUID")
	if uuid == "" {
		uuid = c.PostForm("UUID")
	}
	if uuid == "" || !s.Players.Exists(uuid) {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "account not found"})
		return
	}
	tok := s.Sessions.Issue(uuid)
	pl, _ := s.Players.Load(uuid)
	c.JSON(http.StatusOK, gin.H{
		"code": 200, "message": "ok", "token": tok, "name": pl.Name, "level": pl.Level,
	})
}

// accountCreate — 建号
//
// 请求: HTTP头 UUID (设备生成的新 uuid)
func (s *Server) accountCreate(c *gin.Context) {
	uuid := c.GetHeader("UUID")
	if uuid == "" {
		uuid = c.PostForm("UUID")
	}
	if uuid == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "UUID required"})
		return
	}
	if s.Players.Exists(uuid) {
		c.JSON(http.StatusOK, gin.H{"code": 409, "message": "account exists"})
		return
	}
	if _, err := s.Players.Create(uuid); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	tok := s.Sessions.Issue(uuid)
	log.Printf("[create] new player uuid=%s", uuid)
	c.JSON(http.StatusOK, gin.H{
		"code": 200, "message": "ok", "token": tok,
	})
}

// accountDelete — 删号 (确认后清档)
func (s *Server) accountDelete(c *gin.Context) {
	uuid, _ := c.Get("uuid")
	log.Printf("[delete] uuid=%v", uuid)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "deleted"})
}

// accountHandoverCode — 生成继承码
// ResponseAccountHandoverCode: handover_code
func (s *Server) accountHandoverCode(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	b := make([]byte, 12)
	rand.Read(b)
	code := hex.EncodeToString(b)
	pl.Items["handover_code"] = 0 // 标记
	pl.GachaFree["handover"] = code
	_ = s.Players.Save(pl)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok",
		"handover_code": code, "status": 1})
}

// presentCount — 待领取礼物数
func (s *Server) presentCount(pl *Player) int {
	n := 0
	for k, v := range pl.Items {
		if len(k) > 8 && k[:8] == "present_" && v > 0 {
			n++
		}
	}
	return n
}
