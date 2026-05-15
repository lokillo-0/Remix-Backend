package matchmaking_router

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/remixfn/xenon/modules/matchmaking/types"
	"github.com/remixfn/xenon/utilities"
)

type serverReadyBody struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Region   string `json:"region"`
	Playlist string `json:"playlist"`
}

type serverUnreadyBody struct {
	SessionID string `json:"session_id"`
}

func serverAuth(c *gin.Context) bool {
	if c.GetHeader("Authorization") != utilities.GetConfig().ADMIN_KEY {
		c.Status(http.StatusUnauthorized)
		return false
	}
	return true
}

func ServerReady(c *gin.Context) {
	if !serverAuth(c) { return }
	var body serverReadyBody
	if err := c.ShouldBindJSON(&body); err != nil || body.IP == "" || body.Region == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	sessionID := strings.ReplaceAll(uuid.New().String(), "-", "")

	server := &types.HTTPServer{
		SessionID:    sessionID,
		IP:           body.IP,
		Port:         body.Port,
		Region:       body.Region,
		Playlist:     body.Playlist,
		RegisteredAt: time.Now(),
	}

	types.HTTPServerM.Lock()
	types.HTTPServers[sessionID] = server
	types.HTTPServerM.Unlock()

	c.JSON(http.StatusOK, gin.H{"session_id": sessionID})
}

func ServerUnready(c *gin.Context) {
	if !serverAuth(c) { return }
	var body serverUnreadyBody
	if err := c.ShouldBindJSON(&body); err != nil || body.SessionID == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	types.HTTPServerM.Lock()
	delete(types.HTTPServers, body.SessionID)
	types.HTTPServerM.Unlock()

	c.Status(http.StatusNoContent)
}
