package messages

import (
	"github.com/gorilla/websocket"
	"github.com/remixfn/xenon/modules/matchmaking/utils"
)

func SendWaiting(ws *websocket.Conn) error {
	msg := map[string]interface{}{
		"payload": map[string]interface{}{
			"totalPlayers":     1,
			"connectedPlayers": 1,
			"state":            "Waiting",
		},
		"name": "StatusUpdate",
	}
	return utils.SendMessage(ws, msg)
}
