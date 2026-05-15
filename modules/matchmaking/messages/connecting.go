package messages

import (
	"github.com/gorilla/websocket"
	"github.com/remixfn/xenon/modules/matchmaking/utils"
)

func SendConnecting(ws *websocket.Conn) error {
	msg := map[string]interface{}{
		"payload": map[string]interface{}{
			"state": "Connecting",
		},
		"name": "StatusUpdate",
	}
	return utils.SendMessage(ws, msg)
}
