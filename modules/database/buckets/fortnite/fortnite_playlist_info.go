package fortnite

import "github.com/andr1ww/odin"

type PlaylistInfo struct {
	odin.Bucket   `bucket:"PlaylistInfo" database:"xenon"`
	Type          string `json:"type"`
	Image         string `json:"image"`
	PlaylistName  string `json:"playlist_name"`
	Hidden        bool   `json:"hidden"`
	Description   string `json:"description"`
	SpecialBorder string `json:"special_border"`
	Violator      string `json:"violator"`
	DisplayName   string `json:"display_name"`
}
