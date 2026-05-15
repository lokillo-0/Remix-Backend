package types

import (
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn      *websocket.Conn
	SendQueue bool
	Payload   struct {
		AccountID      string `json:"accountId"`
		BucketID       string `json:"bucketId"`
		BuildUniqueID  string `json:"buildUniqueId"`
		Exp            int64  `json:"exp"`
		FillTeam       string `json:"fillTeam"`
		Iat            int64  `json:"iat"`
		Jti            string `json:"jti"`
		PartyPlayerIDs string `json:"partyPlayerIds"`
		Playlist       string `json:"playlist"`
		Region         string `json:"region"`
		Version        string `json:"version"`
	}
}

type Server struct {
	Conn    *websocket.Conn
	Payload struct {
		BucketID      interface{} `json:"bucketId"`
		Region        string      `json:"region"`
		Version       string      `json:"version"`
		BuildUniqueID string      `json:"buildUniqueId"`
		Exp           int64       `json:"exp"`
		Iat           int64       `json:"iat"`
		Jti           string      `json:"jti"`
	}
	CreatedAt               time.Time    `json:"createdAt"`
	MatchId                 string       `json:"matchId"`
	SessionId               string       `json:"sessionId"`
	IsAssigned              bool         `json:"isAssigned"`
	IsAssigning             bool         `json:"isAssigning"`
	StopAllowingConnections bool         `json:"stopAllowingConnections"`
	Playlist                string       `json:"playlist"`
	Teams                   [][][]string `json:"teams"`
	IsSending               bool         `json:"isSending"`
	AssignMatchSent         bool         `json:"assignMatchSent"`
	MinPlayers              int          `json:"minPlayers"`
	MaxPlayers              int          `json:"maxPlayers"`
}
