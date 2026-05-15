package messages

import (
	"github.com/remixfn/xenon/modules/matchmaking/types"
	"github.com/remixfn/xenon/modules/matchmaking/utils"
)

func SendQueued(c *types.Client, ticketID string, clients int) error {
	if c.SendQueue {
		msg := map[string]interface{}{
			"payload": map[string]interface{}{
				"ticketId":         ticketID,
				"queuedPlayers":    clients,
				"estimatedWaitSec": 0,
				"status":           map[string]interface{}{},
				"state":            "Queued",
			},
			"name": "StatusUpdate",
		}
		return utils.SendMessage(c.Conn, msg)
	}
	return nil
}
