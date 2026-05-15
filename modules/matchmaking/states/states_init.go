package states

import (
	"fmt"
	"time"

	"github.com/remixfn/xenon/modules/matchmaking/messages"
	"github.com/remixfn/xenon/modules/matchmaking/types"
)

func SendInitMessages(c *types.Client, ticketID string, count int) error {
	if c.Conn == nil {
		return fmt.Errorf("client connection is nil")
	}

	if c.Conn.LocalAddr() == nil || c.Conn.RemoteAddr() == nil {
		return fmt.Errorf("websocket connection is not properly established")
	}

	if err := messages.SendConnecting(c.Conn); err != nil {
		return fmt.Errorf("failed to send connecting message: %w", err)
	}

	time.Sleep(400 * time.Millisecond)

	if c.Conn.LocalAddr() == nil || c.Conn.RemoteAddr() == nil {
		return fmt.Errorf("connection lost after connecting message")
	}

	if err := messages.SendWaiting(c.Conn); err != nil {
		return fmt.Errorf("failed to send waiting message: %w", err)
	}

	time.Sleep(500 * time.Millisecond)

	if c.Conn.LocalAddr() == nil || c.Conn.RemoteAddr() == nil {
		return fmt.Errorf("connection lost after waiting message")
	}

	if err := messages.SendQueued(c, ticketID, count); err != nil {
		return fmt.Errorf("failed to send queued message: %w", err)
	}

	return nil
}
