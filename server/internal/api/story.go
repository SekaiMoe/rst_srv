package api

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// story — 剧情
//
// masterdata 模型:
//
//	story: id, story_chapter_id, name, title, label(bundle名), lv(解锁等级), order
//	storyChapter: 章节 (1001..=主线, 2xxx=卡面, 3xxx=活动?)
//	storyChapterMain: 章节属于主线
//	storyChapterCard: card_id → story_chapter_id (卡面剧情, 持卡解锁)
//	storyReward: story_id → {item_kind, item_id, num} 初读奖励
//
// 纪念服语义: 主线剧情按玩家等级解锁 (story.lv); 卡面剧情持卡即解锁; 活动剧情全开

// storyStatus — 已读/解锁状态
func (s *Server) storyStatus(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "player not found"})
		return
	}
	// 拥有的卡 → 解锁的卡面章节
	ownedCards := map[int]bool{}
	for _, cc := range pl.Cards {
		ownedCards[cc.MasterID] = true
	}
	cardChapter := map[int][]int{} // card_id -> story_chapter_id
	for _, r := range s.Master["storyChapterCard"] {
		cid := rowInt(r, "card_id")
		cardChapter[cid] = append(cardChapter[cid], rowInt(r, "story_chapter_id"))
	}
	unlockedCardChapters := map[int]bool{}
	for cid := range ownedCards {
		for _, ch := range cardChapter[cid] {
			unlockedCardChapters[ch] = true
		}
	}

	// 主线章节 (storyChapterMain)
	mainChapters := map[int]bool{}
	for _, r := range s.Master["storyChapterMain"] {
		mainChapters[rowInt(r, "story_chapter_id")] = true
	}
	mainChapters2 := map[int]bool{}
	for _, r := range s.Master["storyChapterMain2"] {
		mainChapters2[rowInt(r, "story_chapter_id")] = true
	}

	read := []int{}
	unlocked := []int{}
	lockedByLv := []int{}
	for _, st := range s.Master["story"] {
		sid := rowInt(st, "id")
		ch := rowInt(st, "story_chapter_id")
		lv := rowInt(st, "lv")
		if pl.ReadStories[sid] {
			read = append(read, sid)
		}
		switch {
		case mainChapters[ch] || mainChapters2[ch]:
			if pl.Level >= lv {
				unlocked = append(unlocked, sid)
			} else {
				lockedByLv = append(lockedByLv, sid)
			}
		case unlockedCardChapters[ch]:
			unlocked = append(unlocked, sid)
		default: // 活动/其他章节: 纪念服全开
			unlocked = append(unlocked, sid)
		}
	}
	// ResponseStoryStatus: status
	c.JSON(http.StatusOK, gin.H{
		"code": 200, "message": "ok", "status": 1,
		"read": read, "read_count": len(read),
		"unlocked": unlocked, "locked_by_lv": lockedByLv,
		"lv": pl.Level,
	})
}

// storyRead — 标记已读 + 初读奖励 (storyReward)
func (s *Server) storyRead(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "player not found"})
		return
	}
	storyID, _ := strconv.Atoi(c.PostForm("story_id"))
	var st map[string]interface{}
	for _, r := range s.Master["story"] {
		if rowInt(r, "id") == storyID {
			st = r
			break
		}
	}
	if st == nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "story not found"})
		return
	}
	rewards := []gin.H{}
	if !pl.ReadStories[storyID] {
		pl.ReadStories[storyID] = true
		for _, r := range s.Master["storyReward"] {
			if rowInt(r, "story_id") == storyID {
				s.grantItem(pl, rowInt(r, "item_kind"), rowInt(r, "item_id"), rowInt(r, "num"))
				rewards = append(rewards, gin.H{
					"item_kind": rowInt(r, "item_kind"), "item_id": rowInt(r, "item_id"), "num": rowInt(r, "num"),
				})
			}
		}
		// 剧情经验
		pl.Exp += 10
		s.playerLevelUp(pl)
	}
	_ = s.Players.Save(pl)
	log.Printf("[story] uuid=%s read story=%d 奖励%d项", pl.UUID, storyID, len(rewards))
	// ResponseStoryRead: status, rewards
	body := walletJSON(pl)
	body["code"] = 200
	body["message"] = "ok"
	body["status"] = 1
	body["story_id"] = storyID
	body["label"] = rowStr(st, "label")
	body["read"] = true
	body["rewards"] = rewards
	c.JSON(http.StatusOK, body)
}

// storyUnlockEvent — 活动剧情解锁 (纪念服: 直接成功)
// ResponseStoryUnlockEvent: status
func (s *Server) storyUnlockEvent(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "status": 1, "unlocked": true})
}

// loginbonusGet — 登录奖励 (纪念服: 每日领取 50 石)
func (s *Server) loginbonusGet(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "player not found"})
		return
	}
	today := todayStr()
	if pl.GachaFree["loginbonus"] != today {
		pl.GachaFree["loginbonus"] = today
		pl.Jewel += 50
		_ = s.Players.Save(pl)
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "status": 1,
			"point_free": pl.Jewel, "got": 50})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "already got today",
		"point_free": pl.Jewel, "got": 0})
}

// itemUse — 使用道具 (体力回复/抽奖券等简化处理)
func (s *Server) itemUse(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "player not found"})
		return
	}
	itemID, _ := strconv.Atoi(c.PostForm("item_id"))
	num, _ := strconv.Atoi(c.PostForm("num"))
	if num <= 0 {
		num = 1
	}
	key := strconv.Itoa(itemID)
	if pl.Items[key] < num {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "item not enough", "have": pl.Items[key]})
		return
	}
	pl.Items[key] -= num
	if pl.Items[key] == 0 {
		delete(pl.Items, key)
	}
	// kind 9 = 体力回复: 回满
	if itemID >= 800 && itemID < 900 { // TODO: 按 itemKind 精确判断
		s.apRegen(pl)
		pl.Ap = pl.ApMax
		pl.ApUpdatedAt = nowUTC()
	}
	_ = s.Players.Save(pl)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "item_id": itemID, "used": num, "items": pl.Items, "ap": pl.Ap})
}

// itemList — 道具列表
func (s *Server) itemList(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "player not found"})
		return
	}
	items := []gin.H{}
	for id, n := range pl.Items {
		items = append(items, gin.H{"item_id": id, "num": n})
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "items": items,
		"jewel": pl.Jewel, "coin": pl.Coin, "friend_point": pl.FriendPoint})
}
