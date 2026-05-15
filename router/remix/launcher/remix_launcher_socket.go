package remix_launcher

import (
	"context"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
)

type LauncherSocketClient struct {
	Conn    *websocket.Conn
	Account *accounts.Account
	Cancel  context.CancelFunc
	Done    chan bool
}

var (
	LauncherClients = make(map[string]*LauncherSocketClient)
	ClientsMutex    sync.RWMutex

	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		EnableCompression: false,
	}
)
