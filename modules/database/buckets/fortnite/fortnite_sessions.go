package fortnite

import (
	"github.com/andr1ww/odin"
)

type Sessions struct {
	odin.Bucket                     `bucket:"GameSessions" database:"xenon" json:"-"`
	SessionId                       string   `json:"session_id"`
	PlaylistName                    string   `json:"playlist_name"`
	ServerAddress                   string   `json:"server_address"`
	LastUpdated                     string   `json:"last_updated"`
	OwnerId                         string   `json:"owner_id"`
	OwnerName                       string   `json:"owner_name"`
	ServerName                      string   `json:"server_name"`
	MaxPublicPlayers                int      `json:"max_public_players"`
	MaxPrivatePlayers               int      `json:"max_private_players"`
	ShouldAdvertise                 bool     `json:"should_advertise"`
	AllowJoinInProgress             bool     `json:"allow_join_in_progress"`
	IsDedicated                     bool     `json:"is_dedicated"`
	UsesStats                       bool     `json:"uses_stats"`
	AllowInvites                    bool     `json:"allow_invites"`
	UsesPresence                    bool     `json:"uses_presence"`
	AllowJoinViaPresence            bool     `json:"allow_join_via_presence"`
	AllowJoinViaPresenceFriendsOnly bool     `json:"allow_join_via_presence_friends_only"`
	BuildUniqueId                   string   `json:"build_unique_id"`
	Attributes                      string   `json:"attributes"`
	ServerPort                      int      `json:"server_port"`
	OpenPublicPlayers               int      `json:"open_public_players"`
	OpenPrivatePlayers              int      `json:"open_private_players"`
	SortWeight                      int      `json:"sort_weight"`
	Started                         bool     `json:"started"`
	PublicPlayers                   []string `json:"public_players"`
	PrivatePlayers                  []string `json:"private_players"`
	Stopped                         bool     `json:"stopped"`
	Region                          string   `json:"region"`
}
