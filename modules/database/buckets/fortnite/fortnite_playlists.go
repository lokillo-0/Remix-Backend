package fortnite

import "github.com/andr1ww/odin"

type Playlist struct {
	odin.Bucket          `bucket:"Playlists" database:"xenon"`
	PlaylistName         string `json:"playlist_name"`
	Enabled              bool   `json:"enabled"`
	IsDefault            bool   `json:"is_default"`
	VisibleWhenDisabled  bool   `json:"visible_when_disabled"`
	DisplayAsNew         bool   `json:"display_as_new"`
	CategoryIndex        int    `json:"category_index"`
	DisplayAsLimitedTime bool   `json:"display_as_limited_time"`
	DisplayPriority      int    `json:"display_priority"`
	EnabledTournament    bool   `json:"enabled_tournament"`
}
