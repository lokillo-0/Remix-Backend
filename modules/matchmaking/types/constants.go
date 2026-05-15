package types

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	Upgrader             websocket.Upgrader
	Clients              map[*Client]bool
	ClientM              sync.RWMutex
	ServerM              sync.RWMutex
	Sessions             = make(map[string]*Server)
	WriteWait            = 10 * time.Second
	PongWait             = 60 * time.Second
	PingPeriod           = (PongWait * 9) / 10
	MaxMessageSize       = 512
	LastSelectedPlaylist = make(map[string]string)
	PlaylistMutex        sync.RWMutex
)
