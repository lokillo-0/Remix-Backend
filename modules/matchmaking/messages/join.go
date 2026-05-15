package messages

import (
	"github.com/gorilla/websocket"
	"github.com/remixfn/xenon/modules/matchmaking/utils"
)

func SendJoin(ws *websocket.Conn, matchID, sessionID string) error {
	msg := map[string]interface{}{
		"payload": map[string]interface{}{
			"matchId":      matchID,
			"sessionId":    sessionID,
			"joinDelaySec": 1,
		},
		"name": "Play",
	}
	return utils.SendMessage(ws, msg)
}
