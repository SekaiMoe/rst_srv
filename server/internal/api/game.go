package api

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

func nowUTC() time.Time { return time.Now().UTC() }

func todayStr() string { return time.Now().Format("2006-01-02") }

// ---- 通用游戏逻辑: 奖励发放 / AP / 玩家等级 ----

// grantItem 发放奖励 (itemKind 参照 masterdata.itemKind)
//
//	1=角色卡 2=饰品卡 3~6=素材卡 7=所持枠 8=ジュエル 9=体力回复 10=コイン
//	11=経験値 12=ガチャチケット 13=イベント 14=楽曲 ...
func (s *Server) grantItem(pl *Player, kind, itemID, num int) string {
	switch kind {
	case 8: // ジュエル (item_id 901/902)
		pl.Jewel += num
		return "jewel"
	case 10: // コイン
		pl.Coin += num
		return "coin"
	case 11: // 経験値
		pl.Exp += num
		return "exp"
	case 1, 2, 3, 4, 5, 6: // 卡类
		for i := 0; i < num; i++ {
			pl.Cards = append(pl.Cards, Card{ID: s.nextCardID(pl), MasterID: itemID, Level: 1})
		}
		return "card"
	default:
		key := strconv.Itoa(itemID)
		pl.Items[key] += num
		return "item"
	}
}

// apRegen AP 自然恢复 (每 AP_REGEN_MIN 回 1 点)
const AP_REGEN_MIN = 5

func (s *Server) apRegen(pl *Player) {
	if pl.Ap >= pl.ApMax || pl.ApUpdatedAt.IsZero() {
		pl.ApUpdatedAt = time.Now()
		return
	}
	elapsed := time.Since(pl.ApUpdatedAt)
	regen := int(elapsed.Minutes() / AP_REGEN_MIN)
	if regen > 0 {
		pl.Ap += regen
		if pl.Ap > pl.ApMax {
			pl.Ap = pl.ApMax
		}
		pl.ApUpdatedAt = pl.ApUpdatedAt.Add(time.Duration(regen*AP_REGEN_MIN) * time.Minute)
	}
}

// playerLevelUp 按经验升级 (masterdata.levelTable: lv1..400, ap 为升级后体力上限)
func (s *Server) playerLevelUp(pl *Player) {
	lt := s.Master["levelTable"]
	// levelTable 按 lv 排序
	sort.Slice(lt, func(i, j int) bool { return rowInt(lt[i], "lv") < rowInt(lt[j], "lv") })
	for _, r := range lt {
		expNeed := rowInt(r, "exp")
		if pl.Exp >= expNeed {
			lv := rowInt(r, "lv")
			if lv > pl.Level {
				pl.Level = lv
				if ap := rowInt(r, "ap"); ap > pl.ApMax {
					pl.ApMax = ap
					pl.Ap = ap // 升级回满
				}
			}
		} else {
			break
		}
	}
}

// ---- stage 索引 ----

func (s *Server) stageRow(stageID int) map[string]interface{} {
	for _, r := range s.Master["stage"] {
		if rowInt(r, "id") == stageID {
			return r
		}
	}
	return nil
}

func (s *Server) stageGoalOf(stageID int) map[string]interface{} {
	for _, r := range s.Master["stageGoal"] {
		if rowInt(r, "stage_id") == stageID {
			return r
		}
	}
	return nil
}

// goalRewardsOf — stageGoalStageGoalReward: 每关 12 梯度奖励 id → stageGoalReward 行
// 顺序: score S/A/B/C, combo S/A/B/C, clear S/A/B/C
func (s *Server) goalRewardsOf(stageID int) [12]map[string]interface{} {
	var out [12]map[string]interface{}
	for _, r := range s.Master["stageGoalStageGoalReward"] {
		if rowInt(r, "stage_id") != stageID {
			continue
		}
		keys := []string{"score_s", "score_a", "score_b", "score_c",
			"combo_s", "combo_a", "combo_b", "combo_c",
			"clear_s", "clear_a", "clear_b", "clear_c"}
		for i, k := range keys {
			rid := rowInt(r, k+"_reward_id")
			for _, rr := range s.Master["stageGoalReward"] {
				if rowInt(rr, "id") == rid {
					out[i] = rr
					break
				}
			}
		}
		break
	}
	return out
}

// calcRank 按 score_s/a/b/c 阈值评级: 4=S 3=A 2=B 1=C 0=miss
func calcRank(goal map[string]interface{}, score int) int {
	switch {
	case score >= rowInt(goal, "score_s"):
		return 4
	case score >= rowInt(goal, "score_a"):
		return 3
	case score >= rowInt(goal, "score_b"):
		return 2
	case score >= rowInt(goal, "score_c"):
		return 1
	}
	return 0
}

// hitRanks 解析客户端 hit_ranks_counts 字段 ("n1,n2,n3,n4,n5,m" 或 JSON 数组)
func hitRanks(s string) []int {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "[]")
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		v, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return nil
		}
		out = append(out, v)
	}
	return out
}

// ---- 结算评级与奖励工具 (HTTP handler 见 game_handlers.go) ----
