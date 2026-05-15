package utils

import (
	"encoding/json"

	"github.com/gorilla/websocket"
)

func SendMessage(ws *websocket.Conn, msg map[string]interface{}) error {
	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	err = ws.WriteMessage(websocket.TextMessage, jsonMsg)
	if err != nil {
		return err
	}

	return nil
}
