package api

import (
	"log"
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ---- livestage 端点: 开局/结算/续命 ----
//
// 客户端字段 (LiveStageResultData + 字符串池):
//
//	stage_id, difficulty_id, score, max_combo, full_combo,
//	hit_ranks_counts (数组), continue_jewel (用石续命次数), use_stone

// livestagePlaygame — 开局扣体力
func (s *Server) livestagePlaygame(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	stageID, _ := strconv.Atoi(c.PostForm("stage_id"))
	st := s.stageRow(stageID)
	if st == nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "stage not found"})
		return
	}
	s.apRegen(pl)
	cost := rowInt(st, "stamina")
	if pl.Ap < cost {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "ap not enough", "ap": pl.Ap, "need": cost})
		return
	}
	pl.Ap -= cost
	if cost > 0 {
		pl.ApUpdatedAt = nowUTC()
	}
	_ = s.Players.Save(pl)
	log.Printf("[play] uuid=%s stage=%d ap=%d-%d", pl.UUID, stageID, pl.Ap+cost, pl.Ap)
	// ResponseLivestagePlaygame: status, power, stamina, stamina_updated_at
	c.JSON(http.StatusOK, gin.H{
		"code": 200, "message": "ok",
		"status":             1,
		"stage_id":           stageID,
		"power":              s.deckPower(pl),
		"stamina":            pl.Ap,
		"stamina_max":        pl.ApMax,
		"stamina_updated_at": pl.ApUpdatedAt.Format("2006-01-02 15:04:05"),
	})
}

// livestageFinishgame — 结算
//
// 评级: stageGoal.score_s/a/b/c → S/A/B/C
// 奖励: stageGoalStageGoalReward 12 梯度 (score/combo/clear 各 S/A/B/C) 首次达成发放
// 记录: BestScores[stage_id] 高分更新; event_point 累计
func (s *Server) livestageFinishgame(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	stageID, _ := strconv.Atoi(c.PostForm("stage_id"))
	score, _ := strconv.Atoi(c.PostForm("total_score")) // 客户端实际字段 (REQUEST_FIELDS.md)
	if score == 0 {
		score, _ = strconv.Atoi(c.PostForm("score")) // 兼容
	}
	maxCombo, _ := strconv.Atoi(c.PostForm("max_combo"))
	fullCombo, _ := strconv.Atoi(c.PostForm("full_combo"))
	if fullCombo == 0 && c.PostForm("is_full_combo") == "1" {
		fullCombo = 1
	}
	ranks := hitRanks(c.PostForm("hit_ranks_counts"))
	eventID, _ := strconv.Atoi(c.PostForm("event_id"))
	_ = c.PostForm("unit_id") // 编成上下文 (客户端发送, 纪念服不强校验)
	_ = c.PostForm("friend_id")

	st := s.stageRow(stageID)
	goal := s.stageGoalOf(stageID)
	if st == nil || goal == nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "stage/goal not found"})
		return
	}

	// ---- 评级 ----
	rank := calcRank(goal, score)
	isClear := rank > 0
	if cl := c.PostForm("is_clear"); cl == "1" {
		isClear = true
	} else if cl == "0" {
		isClear = false
	}

	// ---- 最佳成绩 & 梯度奖励 ----
	prev, had := pl.BestScores[stageID]
	bs := prev
	bs.Plays++
	newRecord := false
	if score > bs.Score {
		bs.Score = score
		newRecord = true
	}
	if maxCombo > bs.MaxCombo {
		bs.MaxCombo = maxCombo
	}
	if fullCombo == 1 {
		bs.FullCombo = 1
	}
	if rank > bs.Rank {
		bs.Rank = rank
	}

	rewards := []gin.H{}
	if !had || newRecord {
		g := s.goalRewardsOf(stageID)
		// score 梯度: 达标且未领过 → 发放 (高级别补齐低级别)
		tiers := []struct {
			need int
			idx  int
		}{
			{4, 0}, {3, 1}, {2, 2}, {1, 3}, // score S/A/B/C
		}
		for _, t := range tiers {
			if rank >= t.need && !bs.Got[t.idx] {
				bs.Got[t.idx] = true
				if r := g[t.idx]; r != nil {
					s.grantItem(pl, rowInt(r, "item_kind"), rowInt(r, "item_id"), rowInt(r, "num"))
					rewards = append(rewards, gin.H{"type": "score", "tier": t.need,
						"item_kind": rowInt(r, "item_kind"), "item_id": rowInt(r, "item_id"), "num": rowInt(r, "num")})
				}
			}
		}
		// combo 梯度
		comboTiers := []struct {
			need int
			key  string
			idx  int
		}{
			{4, "combo_s", 4}, {3, "combo_a", 5}, {2, "combo_b", 6}, {1, "combo_c", 7},
		}
		for _, t := range comboTiers {
			if maxCombo >= rowInt(goal, t.key) && !bs.Got[t.idx] {
				bs.Got[t.idx] = true
				if r := g[t.idx]; r != nil {
					s.grantItem(pl, rowInt(r, "item_kind"), rowInt(r, "item_id"), rowInt(r, "num"))
					rewards = append(rewards, gin.H{"type": "combo", "tier": t.need,
						"item_kind": rowInt(r, "item_kind"), "item_id": rowInt(r, "item_id"), "num": rowInt(r, "num")})
				}
			}
		}
		// clear 梯度 (累计通关次数, 简化为本次 clear 即达成 C; S=10 A=7 B=5 C=1)
		clearTiers := []struct {
			need int
			idx  int
		}{{10, 8}, {7, 9}, {5, 10}, {1, 11}}
		for _, t := range clearTiers {
			if isClear && bs.Plays >= t.need && !bs.Got[t.idx] {
				bs.Got[t.idx] = true
				if r := g[t.idx]; r != nil {
					s.grantItem(pl, rowInt(r, "item_kind"), rowInt(r, "item_id"), rowInt(r, "num"))
					rewards = append(rewards, gin.H{"type": "clear", "tier": t.need,
						"item_kind": rowInt(r, "item_kind"), "item_id": rowInt(r, "item_id"), "num": rowInt(r, "num")})
				}
			}
		}
	}
	pl.BestScores[stageID] = bs

	// ---- 经验 & 活动pt ----
	pl.Exp += score / 100 // 简单经验公式: score/100
	s.playerLevelUp(pl)
	if eventID > 0 {
		pl.EventPoints[eventID] += score / 1000
	}

	if err := s.Players.Save(pl); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	log.Printf("[finish] uuid=%s stage=%d score=%d rank=%d fc=%d combo=%d newRecord=%v 奖励%d项",
		pl.UUID, stageID, score, rank, fullCombo, maxCombo, newRecord, len(rewards))

	// ResponseLivestageFinishgame: rewards, eventRewards, eventPoint
	body := walletJSON(pl)
	body["code"] = 200
	body["message"] = "ok"
	body["stage_id"] = stageID
	body["score"] = score
	body["rank"] = rank
	body["is_clear"] = isClear
	body["max_combo"] = maxCombo
	body["full_combo"] = fullCombo
	body["hit_ranks_counts"] = ranks
	body["is_new_record"] = newRecord
	body["best_score"] = bs.Score
	body["best_rank"] = bs.Rank
	body["rewards"] = rewards
	body["eventPoint"] = pl.EventPoints[eventID] // 无活动时为 0
	body["eventRewards"] = []gin.H{}
	c.JSON(http.StatusOK, body)
}

// livestageGameover — 失败结算 (无奖励, 不扣记录)
func (s *Server) livestageGameover(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	stageID, _ := strconv.Atoi(c.PostForm("stage_id"))
	if bs, ok := pl.BestScores[stageID]; ok {
		bs.Plays++
		pl.BestScores[stageID] = bs
		_ = s.Players.Save(pl)
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "stage_id": stageID})
}

// livestageRetrygame — 重试 (体力已扣, 免费)
func (s *Server) livestageRetrygame(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok"})
}

// livestageStonecontinue — 石头续命 (player 配置 continue_jewel=10/次)
func (s *Server) livestageStonecontinue(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	cost := 10
	for _, r := range s.Master["player"] {
		if rowStr(r, "id") == "continue_jewel" {
			cost = rowInt(r, "value")
			break
		}
	}
	if pl.Jewel < cost {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "jewel not enough", "jewel": pl.Jewel, "need": cost})
		return
	}
	pl.Jewel -= cost
	_ = s.Players.Save(pl)
	// ResponseLivestageStonecontinue: status, jewel(消耗), freeJewel(余量)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "status": 1,
		"jewel": cost, "freeJewel": pl.Jewel, "point_free": pl.Jewel})
}

// livestageAdscontinue — 广告续命 (纪念服直接免费)
func (s *Server) livestageAdscontinue(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok"})
}

// livestageFinishpractice — 练习结算 (无记录)
func (s *Server) livestageFinishpractice(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok"})
}

// ---- ranking ----

// rankingPrivate — 自己的高分排行 (music_id 聚合, 取每曲最高难度最佳)
func (s *Server) rankingPrivate(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	stage2music := map[int]int{}
	for _, r := range s.Master["stage"] {
		stage2music[rowInt(r, "id")] = rowInt(r, "music_id")
	}
	musicName := map[int]string{}
	for _, r := range s.Master["music"] {
		musicName[rowInt(r, "id")] = rowStr(r, "name")
	}
	type row struct {
		MusicID int    `json:"music_id"`
		Name    string `json:"name"`
		StageID int    `json:"stage_id"`
		Score   int    `json:"score"`
		Rank    int    `json:"rank"`
		FC      int    `json:"full_combo"`
		Plays   int    `json:"plays"`
	}
	rows := []row{}
	for sid, bs := range pl.BestScores {
		mid := stage2music[sid]
		rows = append(rows, row{mid, musicName[mid], sid, bs.Score, bs.Rank, bs.FullCombo, bs.Plays})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Score > rows[j].Score })
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "rankings": rows, "count": len(rows)})
}

// rankingEvent / rankingLivestage / rankingPast / rankingPower — 单机化: 返回自己成绩
func (s *Server) rankingSelf(c *gin.Context) { s.rankingPrivate(c) }

// apHealing — 体力回复道具 (item_kind 9); 简化: 石头回满 (healing_jewel=50)
func (s *Server) apHealing(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		return
	}
	s.apRegen(pl)
	if pl.Ap >= pl.ApMax {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "ap is full", "ap": pl.Ap})
		return
	}
	pl.Ap = pl.ApMax
	pl.ApUpdatedAt = nowUTC()
	_ = s.Players.Save(pl)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "ap": pl.Ap, "ap_max": pl.ApMax})
}
