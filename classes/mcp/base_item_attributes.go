package mcp

type BaseItemAttributes struct {
	XP       int        `json:"xp"`
	Level    int        `json:"level"`
	Variants []Variants `json:"variants"`
	ItemSeen bool       `json:"item_seen"`
}
