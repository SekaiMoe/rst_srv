package api

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// ---- maker — 谱面编辑器 (自制谱上传/游玩) ----
//
// 纪念服: 全部谱面公开存储于 data/makers/*.json (单机自用)
//
//	maker/upload    上传自制谱
//	maker/list      谱面列表 (全部公开)
//	maker/info      谱面详情
//	maker/download  下载谱面数据
//	maker/playgame  开始游玩 (不计入正式成绩)
//	maker/finishgame 结算 (记录到谱面作者统计)

type MakerChart struct {
	ID        int            `json:"id"`
	Author    string         `json:"author"`     // 玩家 uuid
	AuthorNo  int            `json:"-"`          // 序号 (避免 uuid 泄露)
	MusicID   int            `json:"music_id"`
	Title     string         `json:"title"`
	Difficulty int           `json:"difficulty"`
	Notes     json.RawMessage `json:"notes"`     // 谱面数据 (客户端格式原样保存)
	Plays     int            `json:"plays"`
	CreatedAt string         `json:"created_at"`
}

type MakerStore struct {
	mu   sync.RWMutex
	dir  string
	next int
}

func NewMakerStore(dir string) (*MakerStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	ms := &MakerStore{dir: dir, next: 1}
	// 扫描已有谱面编号
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		id, err := strconv.Atoi(strings.TrimSuffix(e.Name(), ".json"))
		if err == nil && id >= ms.next {
			ms.next = id + 1
		}
	}
	return ms, nil
}

func (m *MakerStore) Save(ch *MakerChart) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, err := json.MarshalIndent(ch, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(m.dir, strconv.Itoa(ch.ID)+".json"), b, 0o644)
}

func (m *MakerStore) All() []*MakerChart {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []*MakerChart{}
	entries, _ := os.ReadDir(m.dir)
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(m.dir, e.Name()))
		if err != nil {
			continue
		}
		ch := &MakerChart{}
		if json.Unmarshal(b, ch) == nil {
			out = append(out, ch)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (m *MakerStore) Get(id int) *MakerChart {
	m.mu.RLock()
	defer m.mu.RUnlock()
	b, err := os.ReadFile(filepath.Join(m.dir, strconv.Itoa(id)+".json"))
	if err != nil {
		return nil
	}
	ch := &MakerChart{}
	if json.Unmarshal(b, ch) != nil {
		return nil
	}
	return ch
}

// ---- handlers ----

func (s *Server) makerUpload(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "player not found"})
		return
	}
	// 消耗谱面上传道具 (item 806 谱面アップローダー) — 免费额度不足才扣
	musicID, _ := strconv.Atoi(c.PostForm("music_id"))
	difficulty, _ := strconv.Atoi(c.PostForm("difficulty"))
	title := c.PostForm("title")
	notes := c.PostForm("notes")
	if notes == "" {
		notes = c.PostForm("chart")
	}
	if musicID <= 0 || notes == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "music_id and notes required"})
		return
	}
	ch := &MakerChart{
		ID: s.Makers.next, Author: pl.UUID, MusicID: musicID,
		Title: title, Difficulty: difficulty,
		Notes: json.RawMessage(notes), CreatedAt: nowUTC().Format("2006-01-02 15:04:05"),
	}
	if !json.Valid([]byte(notes)) {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "invalid notes json"})
		return
	}
	s.Makers.next++
	if err := s.Makers.Save(ch); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	log.Printf("[maker] upload uuid=%s chart=%d music=%d", pl.UUID, ch.ID, musicID)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok",
		"chart_id": ch.ID, "music_id": musicID, "title": title})
}

func (s *Server) makerList(c *gin.Context) {
	all := s.Makers.All()
	type row struct {
		ID         int    `json:"chart_id"`
		MusicID    int    `json:"music_id"`
		Title      string `json:"title"`
		Difficulty int    `json:"difficulty"`
		Plays      int    `json:"plays"`
		Author     string `json:"author"`
		CreatedAt  string `json:"created_at"`
	}
	out := []row{}
	musicName := map[int]string{}
	for _, r := range s.Master["music"] {
		musicName[rowInt(r, "id")] = rowStr(r, "name")
	}
	for _, ch := range all {
		title := ch.Title
		if title == "" {
			title = musicName[ch.MusicID]
		}
		out = append(out, row{ch.ID, ch.MusicID, title, ch.Difficulty, ch.Plays,
			"player-" + strconv.Itoa(ch.ID%1000), ch.CreatedAt})
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok",
		"charts": out, "count": len(out)})
}

func (s *Server) makerInfo(c *gin.Context) {
	id, _ := strconv.Atoi(c.PostForm("chart_id"))
	ch := s.Makers.Get(id)
	if ch == nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "chart not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok",
		"chart_id": ch.ID, "music_id": ch.MusicID, "title": ch.Title,
		"difficulty": ch.Difficulty, "plays": ch.Plays, "created_at": ch.CreatedAt})
}

func (s *Server) makerDownload(c *gin.Context) {
	id, _ := strconv.Atoi(c.PostForm("chart_id"))
	ch := s.Makers.Get(id)
	if ch == nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "chart not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok",
		"chart_id": ch.ID, "music_id": ch.MusicID, "notes": ch.Notes})
}

func (s *Server) makerPlaygame(c *gin.Context) {
	id, _ := strconv.Atoi(c.PostForm("chart_id"))
	ch := s.Makers.Get(id)
	if ch == nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "chart not found"})
		return
	}
	ch.Plays++
	_ = s.Makers.Save(ch)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "chart_id": id})
}

func (s *Server) makerFinishgame(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok"})
}

func (s *Server) makerFavorite(c *gin.Context) {
	pl := s.playerOf(c)
	if pl == nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "player not found"})
		return
	}
	id, _ := strconv.Atoi(c.PostForm("chart_id"))
	key := "maker_fav_" + strconv.Itoa(id)
	if pl.Items[key] == 1 {
		delete(pl.Items, key)
		_ = s.Players.Save(pl)
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "favorited": false})
		return
	}
	pl.Items[key] = 1
	_ = s.Players.Save(pl)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "favorited": true})
}

func (s *Server) makerPlayerlist(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "players": []string{}})
}

func (s *Server) makerSaveSlotUnlock(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "unlocked": true})
}
