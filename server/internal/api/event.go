package api

import (
	"log"
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ---- vote — 投票活动 (voteInfo 6 届 / voteItem 116 项) ----
//
// 纪念服: 已结束的历史投票可重新开票 (本地统计), 票券 vote_ticket (item kind 19)

// voteInfoHandler — 投票列表/详情
func (s *Server) voteInfoHandler(c *gin.Context) {
	vid, _ := strconv.Atoi(c.PostForm("vote_id"))
	itemsOf := map[int][]gin.H{}
	for _, r := range s.Master["voteItem"] {
		itemsOf[rowInt(r, "vote_id")] = append(itemsOf[rowInt(r, "vote_id")], gin.H{
			"id": rowInt(r, "id"), "name": rowStr(r, "name"),
			"description": rowStr(r, "description"), "order": rowInt(r, "order"),
		})
	}
	type v struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	out := []v{}
	for _, r := range s.Master["voteInfo"] {
		id := rowInt(r, "id")
		if vid != 0 && id != vid {
			continue
		}
		out = append(out, v{id, rowStr(r, "name")})
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "votes": out, "items": itemsOf})
}

// voteDecision — 投票
//
// 表单: vote_item_id=<选项>
func (s *Server) voteDecision(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	itemID, _ := strconv.Atoi(c.PostForm("vote_item_id"))
	var item map[string]interface{}
	for _, r := range s.Master["voteItem"] {
		if rowInt(r, "id") == itemID {
			item = r
			break
		}
	}
	if item == nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "vote item not found"})
		return
	}
	// 消耗票券: kind 19 投票券 (任意 id); 没有票也放行 (纪念服宽容)
	key := ""
	for k := range pl.Items {
		if k != "" && k[0] >= '0' && k[0] <= '9' {
			id, _ := strconv.Atoi(k)
			for _, r := range s.Master["item"] {
				if rowInt(r, "id") == id && rowInt(r, "kind_id") == 19 && pl.Items[k] > 0 {
					key = k
					break
				}
			}
		}
		if key != "" {
			break
		}
	}
	usedTicket := 0
	if key != "" {
		pl.Items[key]--
		if pl.Items[key] == 0 {
			delete(pl.Items, key)
		}
		usedTicket = 1
	}
	// 本地计票
	vk := "vote_" + strconv.Itoa(itemID)
	pl.Items[vk]++
	_ = s.Players.Save(pl)
	log.Printf("[vote] uuid=%s item=%d (%s) 票券=%d", pl.UUID, itemID, rowStr(item, "name"), usedTicket)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "status": 1,
		"vote_item_id": itemID, "name": rowStr(item, "name"),
		"local_count": pl.Items[vk], "used_ticket": usedTicket})
}

// ---- event — 活动系统 ----

// eventInfoHandler — 活动列表 (eventInfo 471 个, 纪念服全部可见)
func (s *Server) eventInfoHandler(c *gin.Context) {
	pl := s.playerOf(c)
	type e struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Kind int    `json:"kind"`
	}
	out := []e{}
	for _, r := range s.Master["eventInfo"] {
		out = append(out, e{rowInt(r, "id"), rowStr(r, "name"), rowInt(r, "kind")})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	pts := gin.H{}
	if pl != nil {
		for k, v := range pl.EventPoints {
			pts[strconv.Itoa(k)] = v
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok",
		"events": out, "count": len(out), "event_points": pts})
}

// eventPlaygame — 活动曲开局 (复用 livestage 扣体力逻辑)
func (s *Server) eventPlaygame(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	eventID, _ := strconv.Atoi(c.PostForm("event_id"))
	c.Set("event_id", eventID)
	s.livestagePlaygame(c)
}

// eventFinishgame — 活动曲结算 (复用结算 + event_pt 加成)
func (s *Server) eventFinishgame(c *gin.Context) {
	s.livestageFinishgame(c) // finishgame 内部已处理 event_id 表单
}

// eventGameover — 失败
func (s *Server) eventGameover(c *gin.Context) { s.livestageGameover(c) }

// eventFriendsearch — 活动好友搜索 (NPC 化: 角色)
func (s *Server) eventFriendsearch(c *gin.Context) { s.friendSearch(c) }

// ---- event Battle — 活动对战 (eventBattle 4 届) ----

// eventBattleStart — 对战开局: 返回敌方 unit 数据
func (s *Server) eventBattleStart(c *gin.Context) {
	battleID, _ := strconv.Atoi(c.PostForm("event_battle_id"))
	if battleID == 0 {
		for _, r := range s.Master["eventBattle"] {
			battleID = rowInt(r, "id") // 默认第一届
			break
		}
	}
	var battle map[string]interface{}
	for _, r := range s.Master["eventBattle"] {
		if rowInt(r, "id") == battleID {
			battle = r
			break
		}
	}
	if battle == nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "battle not found"})
		return
	}
	// 敌方单位 (eventBattleEnemy → eventBattleUnit 数值)
	enemyLv := 1
	units := []gin.H{}
	unitIDs := []string{}
	for _, r := range s.Master["eventBattleEnemy"] {
		if rowInt(r, "id") == rowInt(battle, "enemy_unit_id") { // 简化: 取同 id 组
			for i := 0; i < 7; i++ {
				uid := rowInt(r, "battle_unit_id_"+strconv.Itoa(i))
				if uid == 0 {
					continue
				}
				for _, u := range s.Master["eventBattleUnit"] {
					if rowInt(u, "id") == uid {
						units = append(units, gin.H{
							"unit_id": uid, "card_id": rowInt(u, "master_card_id"),
							"life": rowInt(u, "life_max"), "power1": rowInt(u, "power_1_max"),
							"power2": rowInt(u, "power_2_max"), "power3": rowInt(u, "power_3_max"),
						})
						break
					}
				}
			}
			enemyLv = rowInt(r, "lv_max")
			break
		}
	}
	_ = unitIDs
	c.JSON(http.StatusOK, gin.H{
		"code": 200, "message": "ok",
		"battle_id": battleID, "enemy_lv": enemyLv,
		"attack_coefficient": rowInt(battle, "bonus_status"),
		"enemy_units":        units,
	})
}

// eventBattleFinish — 对战结算 (简化: 按 score 判胜, 发活动pt)
func (s *Server) eventBattleFinish(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	eventID, _ := strconv.Atoi(c.PostForm("event_id"))
	score, _ := strconv.Atoi(c.PostForm("score"))
	win := score > 0
	gain := 0
	if win {
		gain = score / 1000
		if eventID > 0 {
			pl.EventPoints[eventID] += gain
		}
		_ = s.Players.Save(pl)
	}
	// ResponseEventBattleFinish: rewards, eventRewards
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "status": 1,
		"win": win, "score": score, "event_point_gained": gain,
		"rewards": []gin.H{}, "eventRewards": []gin.H{},
		"eventPoint":  pl.EventPoints[eventID],
		"event_point": map[int]int{eventID: pl.EventPoints[eventID]}})
}

// ---- background/retry — 后台结算重试 (断线重连) ----

func (s *Server) backgroundRetryFinishgame(c *gin.Context) {
	// 客户端掉线后重试: 与正常结算同参, 直接复用
	s.livestageFinishgame(c)
}

func (s *Server) backgroundRetryEventFinishgame(c *gin.Context) {
	s.livestageFinishgame(c)
}
