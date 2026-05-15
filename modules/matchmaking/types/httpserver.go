package types

import (
	"sync"
	"time"
)

type HTTPServer struct {
	SessionID    string
	IP           string
	Port         int
	Region       string
	Playlist     string
	RegisteredAt time.Time
}

var (
	HTTPServers = make(map[string]*HTTPServer)
	HTTPServerM sync.RWMutex
)
