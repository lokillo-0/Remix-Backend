package messages

import (
	"github.com/remixfn/xenon/modules/matchmaking/types"
	"github.com/remixfn/xenon/modules/matchmaking/utils"
)

func SendSessionAssignment(c *types.Client, matchID string) error {
	types.ClientM.Lock()
	c.SendQueue = false
	types.Clients[c] = true
	types.ClientM.Unlock()

	msg := map[string]interface{}{
		"payload": map[string]interface{}{
			"matchId": matchID,
			"state":   "SessionAssignment",
		},
		"name": "StatusUpdate",
	}
	return utils.SendMessage(c.Conn, msg)
}
