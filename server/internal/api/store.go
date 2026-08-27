// Package api — Re:ステージ！プリズムステップ 私服 API 实现
//
// 协议依据: LOGIN_FLOW_ANALYSIS.md / API_DOCUMENTATION.md
//
//	请求: POST application/x-www-form-urlencoded (WWWForm)
//	认证: HTTP 头 "UUID" + 表单 "token" (login1st 发放)
//	响应: JSON {"code":200,"message":"...","token":"...", ...}
package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ---- 通用响应 ----

type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Token   string      `json:"token,omitempty"`
	Data    interface{} `json:"-"`
	// 其余业务字段通过 gin.H 附加, 见各 handler
}

// ---- 会话管理 ----

type Session struct {
	Token     string    `json:"token"`
	UUID      string    `json:"uuid"`
	CreatedAt time.Time `json:"created_at"`
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session // token -> session
	lastUUID string             // 最近一次 login1st 的 uuid (宽容模式用)
}

func NewSessionStore() *SessionStore {
	return &SessionStore{sessions: make(map[string]*Session)}
}

func (s *SessionStore) Issue(uuid string) string {
	b := make([]byte, 24)
	rand.Read(b)
	tok := hex.EncodeToString(b)
	s.mu.Lock()
	s.sessions[tok] = &Session{Token: tok, UUID: uuid, CreatedAt: time.Now()}
	s.lastUUID = uuid
	s.mu.Unlock()
	return tok
}

func (s *SessionStore) Get(token string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[token]
	return sess, ok
}

// Adopt — 宽容模式: 未知 token (客户端 SessionManager 本地生成的) 绑定到最近登录的玩家
func (s *SessionStore) Adopt(token string) (*Session, bool) {
	if token == "" || s.lastUUID == "" {
		return nil, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.sessions[token]; ok {
		return sess, true
	}
	sess := &Session{Token: token, UUID: s.lastUUID, CreatedAt: time.Now()}
	s.sessions[token] = sess
	return sess, true
}

// ---- 玩家数据 (JSON 文件存储) ----

type Player struct {
	ID          int               `json:"id"` // 客户端 UserModel.id (int)
	UUID        string            `json:"uuid"`
	Name        string            `json:"name"`
	CreatedAt   time.Time         `json:"created_at"`
	Jewel       int               `json:"jewel"`
	Coin        int               `json:"coin"`
	FriendPoint int               `json:"friend_point"`
	Level       int               `json:"level"`
	Exp         int               `json:"exp"`
	Ap          int               `json:"ap"`
	ApMax       int               `json:"ap_max"`
	Cards       []Card            `json:"cards"`         // 卡实例列表
	Decks       [][]int           `json:"decks"`         // 每编成位的卡实例 id
	Items       map[string]int    `json:"items"`         // 道具 (item_id -> num), 901=无偿石 902=有偿石
	GachaFree   map[string]string `json:"gacha_free"`    // gacha_id -> 最后免费日 (YYYY-MM-DD)
	BestScores  map[int]BestScore `json:"best_scores"`   // stage_id -> 最佳成绩
	ReadStories map[int]bool      `json:"read_stories"`  // story_id -> 已读
	EventPoints map[int]int       `json:"event_points"`  // event_id -> pt
	ApUpdatedAt time.Time         `json:"ap_updated_at"` // 体力回复基准
	// 档案
	ProfileText    string `json:"profile_text"`
	TitleID        int    `json:"title_id"`
	FavoriteCardID int    `json:"favorite_card_id"`
	PublishCardID  int    `json:"publish_card_id"`
	// 编成
	AcceSlots map[int][]int `json:"acce_slots"` // 角色卡实例id -> 饰品实例id列表
	DeckNames []string      `json:"deck_names"`
}

// BestScore — 单曲最佳 (S=4 A=3 B=2 C=1 miss=0); 12 梯度奖励首次达成发放
type BestScore struct {
	Score     int      `json:"score"`
	MaxCombo  int      `json:"max_combo"`
	FullCombo int      `json:"full_combo"`
	Rank      int      `json:"rank"`
	Plays     int      `json:"plays"`
	Got       [12]bool `json:"got"` // score S/A/B/C, combo S/A/B/C, clear S/A/B/C
}

// Card — 玩家持有的卡实例
// masterdata 中 card.id 为卡模板 id; 实例 id 服务器发放
// rarity: 1..5; level 初始 1

type Card struct {
	ID       int `json:"id"`
	MasterID int `json:"master_id"`
	Level    int `json:"level"`
	Exp      int `json:"exp"`
	Lock     int `json:"lock"`
}

type PlayerStore struct {
	mu  sync.RWMutex
	dir string
}

func NewPlayerStore(dir string) (*PlayerStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &PlayerStore{dir: dir}, nil
}

func (p *PlayerStore) path(uuid string) string {
	// 防 path traversal: 只保留 hex 字符
	safe := make([]rune, 0, len(uuid))
	for _, r := range uuid {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			safe = append(safe, r)
		}
	}
	return filepath.Join(p.dir, fmt.Sprintf("%s.json", string(safe)))
}

func (p *PlayerStore) Exists(uuid string) bool {
	_, err := os.Stat(p.path(uuid))
	return err == nil
}

func (p *PlayerStore) Create(uuid string) (*Player, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	pl := &Player{
		ID:        1,
		UUID:      uuid,
		Name:      "プリズムステップ",
		CreatedAt: time.Now(),
		Level:     1,
		Ap:        100, ApMax: 100,
		Jewel: 100000, Coin: 1000000, FriendPoint: 10000,
		Decks:       make([][]int, 5),
		// 初始: KiReRe 五人组 [On:STAGE!!] SR (master 1-5), 卡组1
		Cards: []Card{
			{ID: 1, MasterID: 1, Level: 30, Exp: 0},
			{ID: 2, MasterID: 2, Level: 30, Exp: 0},
			{ID: 3, MasterID: 3, Level: 30, Exp: 0},
			{ID: 4, MasterID: 4, Level: 30, Exp: 0},
			{ID: 5, MasterID: 5, Level: 30, Exp: 0},
		},
		FavoriteCardID: 1,
		Items:         map[string]int{},
		GachaFree:     map[string]string{},
		BestScores:    map[int]BestScore{},
		ReadStories:   map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true}, // 序章默认已读(跳过教程重演)
		EventPoints:   map[int]int{},
		AcceSlots:     map[int][]int{},
		DeckNames:     make([]string, 5),
		ApUpdatedAt:   time.Now(),
	}
	pl.Decks[0] = []int{1, 2, 3, 4, 5} // 卡组1 = KiReRe
	f, err := os.Create(p.path(uuid))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return pl, enc.Encode(pl)
}

func (p *PlayerStore) Load(uuid string) (*Player, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	b, err := os.ReadFile(p.path(uuid))
	if err != nil {
		return nil, err
	}
	pl := &Player{}
	return pl, json.Unmarshal(b, pl)
}

func (p *PlayerStore) Save(pl *Player) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	b, err := json.MarshalIndent(pl, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p.path(pl.UUID), b, 0o644)
}
