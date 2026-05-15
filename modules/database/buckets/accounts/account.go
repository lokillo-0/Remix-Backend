package accounts

import (
	"time"

	"github.com/andr1ww/odin"
)

type Account struct {
	odin.Bucket             `bucket:"Accounts" database:"xenon"`
	Created                 time.Time           `json:"created"`
	Email                   string              `json:"email"`
	Password                string              `json:"password"`
	DisplayName             string              `json:"display_name"`
	Username                string              `json:"username"`
	Banned                  bool                `json:"banned"`
	Roles                   []string            `json:"roles"`
	BanHistory              []map[string]string `json:"ban_history"`
	IsServer                bool                `json:"is_server"`
	LastLoginTime           string              `json:"last_login_time"`
	LastLoginIP             string              `json:"last_login_ip"`
	LastDisplayNameChange   string              `json:"last_display_name_change"`
	ProfilePicture          string              `json:"profile_picture"`
	DisplayNameChanges      int                 `json:"display_name_changes"`
	MatchmakingBannedUntil  string              `json:"matchmaking_banned_until"`
	MatchmakingBannedSince  string              `json:"matchmaking_banned_since"`
	MatchmakingBannedReason string              `json:"matchmaking_banned_reason"`
	DeviceID                string              `json:"device_id"`
	HWID                    string              `json:"hwid"`
	DiscordID               string              `json:"discord_id"`
}
