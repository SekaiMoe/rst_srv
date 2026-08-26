package api

import (
	"log"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
)

// Server 持有全局状态
type Server struct {
	Sessions *SessionStore
	Players  *PlayerStore
	Master   Master      // masterdata 93 张表
	Gacha    *GachaIndex // gacha 索引
	Makers   *MakerStore // 自制谱存储
}

// ---- 路由注册: 全部 116 个端点 (来自 API_DOCUMENTATION.md) ----

func RegisterRoutes(r *gin.Engine, dataDir, masterdataPath string) {
	s := &Server{
		Sessions: NewSessionStore(),
	}
	ps, err := NewPlayerStore(dataDir + "/players")
	if err != nil {
		panic(err)
	}
	s.Players = ps
	s.Makers, err = NewMakerStore(dataDir + "/makers")
	if err != nil {
		panic(err)
	}
	if masterdataPath != "" {
		m, err := LoadMaster(masterdataPath)
		if err != nil {
			log.Printf("[warn] masterdata 未加载: %v (gacha 端点将不可用)", err)
		} else {
			s.Master = m
			s.Gacha = BuildGachaIndex(m)
		}
	}

	// ===== 账号 (三段式登录, 见 LOGIN_FLOW_ANALYSIS.md) =====
	r.POST("/account/login1st", s.login1st)
	r.POST("/account/login2nd", s.login2nd)
	r.POST("/account/login3rd", s.login3rd)
	r.POST("/account/login", s.loginFull) // 旧版一键登录
	r.POST("/account/create", s.accountCreate)
	r.POST("/account/info", s.requireAuth, s.ok)
	r.POST("/account/delete", s.requireAuth, s.accountDelete)
	r.POST("/account/delete_info", s.requireAuth, s.ok)
	r.POST("/account/handover", s.requireAuth, s.ok)
	r.POST("/account/handover_code", s.requireAuth, s.accountHandoverCode)
	r.POST("/account/handover_new_code", s.requireAuth, s.accountHandoverCode)

	// ===== 抽卡 (真实实现, 见 gacha.go) =====
	r.POST("/gacha/list", s.requireAuth, s.gachaList)
	r.POST("/gacha/play", s.requireAuth, s.gachaPlay)
	r.POST("/gacha/execute", s.requireAuth, s.gachaExecute)

	// ===== masterdata 调试端点 (客户端本体走 CDN bundle, 此处供调试/第三方工具) =====
	if s.Master != nil {
		r.GET("/debug/masterdata", func(c *gin.Context) {
			table := c.Query("table")
			if table == "" {
				tables := make([]string, 0, len(s.Master))
				for k := range s.Master {
					tables = append(tables, k)
				}
				sort.Strings(tables)
				c.JSON(http.StatusOK, gin.H{"code": 200, "tables": tables})
				return
			}
			rows, ok := s.Master[table]
			if !ok {
				c.JSON(http.StatusOK, gin.H{"code": 404, "message": "table not found"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"code": 200, "table": table, "count": len(rows), "rows": rows})
		})
	}

	// ===== 直播/结算 (真实实现, 见 game_handlers.go) =====
	r.POST("/livestage/playgame", s.requireAuth, s.livestagePlaygame)
	r.POST("/livestage/finishgame", s.requireAuth, s.livestageFinishgame)
	r.POST("/livestage/finishpractice", s.requireAuth, s.livestageFinishpractice)
	r.POST("/livestage/gameover", s.requireAuth, s.livestageGameover)
	r.POST("/livestage/retrygame", s.requireAuth, s.livestageRetrygame)
	r.POST("/livestage/adscontinue", s.requireAuth, s.livestageAdscontinue)
	r.POST("/livestage/stonecontinue", s.requireAuth, s.livestageStonecontinue)

	// ===== 排行 =====
	r.POST("/ranking/private", s.requireAuth, s.rankingPrivate)
	r.POST("/ranking/event", s.requireAuth, s.rankingSelf)
	r.POST("/ranking/livestage", s.requireAuth, s.rankingSelf)
	r.POST("/ranking/past", s.requireAuth, s.rankingSelf)
	r.POST("/ranking/power", s.requireAuth, s.rankingSelf)

	// ===== 卡牌/编成 (真实实现, 见 card.go) =====
	r.POST("/card/grow", s.requireAuth, s.cardGrow)
	r.POST("/card/lock", s.requireAuth, s.cardLock)
	r.POST("/card/sell", s.requireAuth, s.cardSell)
	r.POST("/deck/leader", s.requireAuth, s.deckLeader)
	r.POST("/deck/card", s.requireAuth, s.deckCard)

	// ===== 剧情/道具/登录奖 (真实实现, 见 story.go) =====
	r.POST("/story/read", s.requireAuth, s.storyRead)
	r.POST("/story/status", s.requireAuth, s.storyStatus)
	r.POST("/story/unlock_event", s.requireAuth, s.storyUnlockEvent)
	r.POST("/loginbonus/get", s.requireAuth, s.loginbonusGet)
	r.POST("/item/list", s.requireAuth, s.itemList)
	r.POST("/item/use", s.requireAuth, s.itemUse)
	r.POST("/ap/healing", s.requireAuth, s.apHealing)

	// ===== 好友 (NPC 化, 见 friend.go) =====
	r.POST("/friend/follow", s.requireAuth, s.friendFollow)
	r.POST("/friend/unfollow", s.requireAuth, s.friendUnfollow)
	r.POST("/friend/unfollower", s.requireAuth, s.friendUnfollower)
	r.POST("/friend/followlist", s.requireAuth, s.friendFollowlist)
	r.POST("/friend/followerlist", s.requireAuth, s.friendFollowerlist)
	r.POST("/friend/search", s.requireAuth, s.friendSearch)
	r.POST("/friend/livestagelist", s.requireAuth, s.friendLivestagelist)

	// ===== 商店/兑换/退款 (见 shop.go) =====
	r.POST("/shop/itemlist", s.requireAuth, s.shopItemlist)
	r.POST("/shop/purchaselist", s.requireAuth, s.shopPurchaselist)
	r.POST("/shop/buy", s.requireAuth, s.shopBuy)
	r.POST("/shop/buy_movie", s.requireAuth, s.shopBuyMovie)
	r.POST("/shop/buy_music", s.requireAuth, s.shopBuyMusic)
	r.POST("/shop/exchange", s.requireAuth, s.shopExchange)
	r.POST("/shop/exchange_list", s.requireAuth, s.shopExchangeList)
	r.POST("/shop/log", s.requireAuth, s.shopLog)
	r.POST("/shop/receiptcheckandroid2", s.requireAuth, s.shopReceiptcheck)
	r.POST("/shop/receiptcheckios2", s.requireAuth, s.shopReceiptcheck)
	r.POST("/repayment/create", s.requireAuth, s.repaymentCreate)

	// ===== 礼物箱 (见 friend.go) =====
	r.POST("/presentbox/list", s.requireAuth, s.presentboxList)
	r.POST("/presentbox/get", s.requireAuth, s.presentboxGet)
	r.POST("/presentbox/get_all", s.requireAuth, s.presentboxGetAll)
	r.POST("/presentbox/log", s.requireAuth, s.presentboxLog)

	// ===== 成就 (见 shop.go) =====
	r.POST("/achieve/list", s.requireAuth, s.achieveList)
	r.POST("/achieve/stock", s.requireAuth, s.achieveStock)
	r.POST("/achieve/get", func(c *gin.Context) { s.achieveGet(c, false) })
	r.POST("/achieve/get_all", s.requireAuth, s.achieveGetAll)

	// ===== 谱面编辑器 (见 maker.go) =====
	r.POST("/maker/upload", s.requireAuth, s.makerUpload)
	r.POST("/maker/list", s.requireAuth, s.makerList)
	r.POST("/maker/info", s.requireAuth, s.makerInfo)
	r.POST("/maker/download", s.requireAuth, s.makerDownload)
	r.POST("/maker/playgame", s.requireAuth, s.makerPlaygame)
	r.POST("/maker/finishgame", s.requireAuth, s.makerFinishgame)
	r.POST("/maker/gameover", s.requireAuth, s.makerFinishgame)
	r.POST("/maker/retrygame", s.requireAuth, s.makerFinishgame)
	r.POST("/maker/continue", s.requireAuth, s.makerFinishgame)
	r.POST("/maker/favorite", s.requireAuth, s.makerFavorite)
	r.POST("/maker/playerlist", s.requireAuth, s.makerPlayerlist)
	r.POST("/maker/save_slot_unlock", s.requireAuth, s.makerSaveSlotUnlock)

	// ===== 通用 stub =====
	// ===== 真机协议行为对齐 (2026-08-26 在线探测确认) =====
	// 全部响应 HTTP 200, 错误在 JSON code 字段:
	//   404=未知路由  400=业务拒绝  700=token 无效(客户端重新登录)
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "サービス終了しました"})
	})
	r.HandleMethodNotAllowed = true
	r.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "サービス終了しました"})
	})

	// ===== 档案 (见 profile.go) =====
	r.POST("/profile/setname", s.requireAuth, s.profileSetName)
	// tutorial 推进 (UserModelApiManager.Tutorial, 客户端字面量 "tutorial")
	for _, p := range []string{"/tutorial", "/user/tutorial", "/profile/tutorial", "/account/tutorial"} {
		r.POST(p, s.requireAuth, s.tutorialAdvance)
	}
	r.POST("/profile/setprofile", s.requireAuth, s.profileSetProfile)
	r.POST("/profile/settitle", s.requireAuth, s.profileSetTitle)
	r.POST("/profile/titlelist", s.requireAuth, s.profileTitlelist)
	r.POST("/profile/favorite_card", s.requireAuth, s.profileFavoriteCard)
	r.POST("/profile/publish_card", s.requireAuth, s.profilePublishCard)

	// ===== 饰品/批量编成 (见 acce.go) =====
	r.POST("/card/acce", s.requireAuth, s.cardAcce)
	r.POST("/card//acce_skill", s.requireAuth, s.cardAcceSkill)
	r.POST("/card/batch", s.requireAuth, s.cardBatch)
	r.POST("/deck/batch", s.requireAuth, s.deckBatch)
	r.POST("/deck/disband", s.requireAuth, s.deckDisband)
	r.POST("/deck/rename", s.requireAuth, s.deckRename)
	r.POST("/acce/grow", s.requireAuth, s.acceGrow)
	r.POST("/acce/lock", s.requireAuth, s.acceLock)
	r.POST("/acce/sell", s.requireAuth, s.acceSell)

	// ===== 投票 (见 event.go) =====
	r.POST("/vote/info", s.requireAuth, s.voteInfoHandler)
	r.POST("/vote/decision", s.requireAuth, s.voteDecision)

	// ===== 活动/对战 (见 event.go) =====
	r.POST("/event/info", s.requireAuth, s.eventInfoHandler)
	r.POST("/event/playgame", s.requireAuth, s.eventPlaygame)
	r.POST("/event/finishgame", s.requireAuth, s.eventFinishgame)
	r.POST("/event/gameover", s.requireAuth, s.eventGameover)
	r.POST("/event/friendsearch", s.requireAuth, s.eventFriendsearch)
	r.POST("/event/battle_start", s.requireAuth, s.eventBattleStart)
	r.POST("/event/battle_finish", s.requireAuth, s.eventBattleFinish)
	r.POST("/background/retry/finishgame", s.requireAuth, s.backgroundRetryFinishgame)
	r.POST("/background/retry/event_finishgame", s.requireAuth, s.backgroundRetryEventFinishgame)

	// ===== 杂项 =====
	r.POST("/item/friendpt", s.requireAuth, s.itemFriendpt)
	r.POST("/item/livestage", s.requireAuth, s.itemLivestage)
}

// ---- 认证中间件: HTTP 头 UUID + 表单 token ----

func (s *Server) requireAuth(c *gin.Context) {
	uuid := c.GetHeader("UUID")
	token := c.PostForm("token")
	if token == "" {
		token = c.GetHeader("TOKEN") // 客户端反汇编确认: 头字段名为 TOKEN
	}
	if uuid == "" && token == "" {
		// 真机行为: 认证失败 code=700 (客户端触发重新登录)
		c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
		c.Abort()
		return
	}
	if token != "" {
		if sess, ok := s.Sessions.Get(token); ok {
			c.Set("uuid", sess.UUID)
			c.Next()
			return
		}
		if sess, ok := s.Sessions.Adopt(token); ok {
			c.Set("uuid", sess.UUID)
			c.Next()
			return
		}
	}
	if uuid != "" && s.Players.Exists(uuid) {
		c.Set("uuid", uuid)
		c.Next()
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 700, "message": "トークンが存在しない。ログインしなおしかな。"})
	c.Abort()
}

// ---- 通用 OK ----

func (s *Server) ok(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok"})
}
