package api

import (
	"encoding/json"
	"log"
	"os"
)

// Master — masterdata.json (93 张表) 的内存表示
type Master map[string][]map[string]interface{}

// LoadMaster 从 masterdata.json 加载
func LoadMaster(path string) (Master, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Master
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	log.Printf("[master] 加载 %d 张表", len(m))
	return m, nil
}

// ---- 便捷取值 (masterdata 行均为 JSON 对象) ----

func rowInt(row map[string]interface{}, key string) int {
	if v, ok := row[key].(float64); ok {
		return int(v)
	}
	return 0
}

func rowStr(row map[string]interface{}, key string) string {
	if v, ok := row[key].(string); ok {
		return v
	}
	return ""
}

// ---- gacha 索引 ----

type GachaIndex struct {
	Gacha   map[int]map[string]interface{}   // gacha_id -> row
	Detail  map[int]map[string]interface{}   // gacha_detail_id -> row
	Details map[int][]map[string]interface{} // gacha_id -> details
	Lots    map[int][]map[string]interface{} // gacha_detail_id -> lots (ratio 加权)
	Block   map[int][]int                    // gacha_block_id -> card_ids
	Card    map[int]map[string]interface{}   // card_id -> row
}

func BuildGachaIndex(m Master) *GachaIndex {
	gi := &GachaIndex{
		Gacha:   map[int]map[string]interface{}{},
		Detail:  map[int]map[string]interface{}{},
		Details: map[int][]map[string]interface{}{},
		Lots:    map[int][]map[string]interface{}{},
		Block:   map[int][]int{},
		Card:    map[int]map[string]interface{}{},
	}
	for _, r := range m["gacha"] {
		gi.Gacha[rowInt(r, "id")] = r
	}
	for _, r := range m["gachaDetail"] {
		id := rowInt(r, "id")
		gi.Detail[id] = r
		gid := rowInt(r, "gacha_id")
		gi.Details[gid] = append(gi.Details[gid], r)
	}
	for _, r := range m["gachaLot"] {
		did := rowInt(r, "gacha_detail_id")
		gi.Lots[did] = append(gi.Lots[did], r)
	}
	for _, r := range m["gachaBlockDetail"] {
		bid := rowInt(r, "gacha_block_id")
		gi.Block[bid] = append(gi.Block[bid], rowInt(r, "card_id"))
	}
	for _, r := range m["card"] {
		gi.Card[rowInt(r, "id")] = r
	}
	log.Printf("[gacha] 索引: %d 池 %d 明细 %d lot组 %d block %d 卡",
		len(gi.Gacha), len(gi.Detail), len(gi.Lots), len(gi.Block), len(gi.Card))
	return gi
}
