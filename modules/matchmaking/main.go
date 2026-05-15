package matchmaking

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/remixfn/xenon/modules/database/buckets/fortnite"
	matchmaking_socket "github.com/remixfn/xenon/modules/matchmaking/socket"
	"github.com/remixfn/xenon/modules/matchmaking/types"
	"github.com/remixfn/xenon/utilities"
)

func Init(router *gin.Engine) {
	types.Upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	types.Clients = make(map[*types.Client]bool)
	types.ClientM = sync.RWMutex{}
	if types.Sessions == nil {
		types.Sessions = make(map[string]*types.Server)
	}

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			<-ticker.C
			types.ClientM.Lock()
			sessionsToDelete := make([]string, 0)

			for key, session := range types.Sessions {
				if session.Conn == nil {
					sessionsToDelete = append(sessionsToDelete, key)
					continue
				}

				if time.Since(session.CreatedAt) > 7*time.Minute && !session.IsAssigned {
					sessionsToDelete = append(sessionsToDelete, key)
					continue
				}

				session.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
				err := session.Conn.WriteMessage(websocket.PingMessage, []byte{})
				if err != nil {
					sessionsToDelete = append(sessionsToDelete, key)
				}
			}

			for _, sessionID := range sessionsToDelete {
				if session, exists := types.Sessions[sessionID]; exists {
					if session.Conn != nil {
						if time.Since(session.CreatedAt) > 7*time.Minute && !session.IsAssigned {
							if err := session.Conn.WriteMessage(websocket.TextMessage, []byte(`{"name":"QueuedForBackfill","payload":{}}`)); err != nil {
								session.Conn.Close()
								return
							}
						}
						session.Conn.Close()
					}
					delete(types.Sessions, sessionID)

					sessions, _ := odin.FindWhere("GameSessions", map[string]interface{}{
						"session_id": sessionID,
					}, func() interface{} {
						return &fortnite.Sessions{}
					})
					if len(sessions) > 0 {
						if sesh, ok := sessions[0].(*fortnite.Sessions); ok {
							sesh.Bucket.Delete(sesh)
						}
					}
				}
			}

			dbSessions, _ := odin.FindAll("GameSessions", func() interface{} {
				return &fortnite.Sessions{}
			})
			for _, dbSession := range dbSessions {
				if sesh, ok := dbSession.(*fortnite.Sessions); ok {
					if _, exists := types.Sessions[sesh.SessionId]; !exists {
						sesh.Bucket.Delete(sesh)
					}
				}
			}
			types.ClientM.Unlock()
		}
	}()

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			<-ticker.C
			types.ClientM.Lock()

			for _, session := range types.Sessions {
				if !session.IsAssigned && time.Since(session.CreatedAt) >= 7*time.Minute {
					if session.Conn != nil {
						if err := session.Conn.WriteMessage(websocket.TextMessage, []byte(`{"name":"QueuedForBackfill","payload":{}}`)); err != nil {
							session.Conn.Close()
							return
						}
					}
				}
			}
			types.ClientM.Unlock()
		}
	}()

	router.GET("/xenon/matchmaking/base", matchmaking_socket.HandleSocket)
	router.GET("/xenon/matchmaking/session", matchmaking_socket.HandleSessionSocket)
	router.GET("/xenon/matchmaking/data", func(c *gin.Context) {
		clients := make([]map[string]interface{}, 0)
		sessions := make([]map[string]interface{}, 0)
		types.ClientM.RLock()
		for client := range types.Clients {
			clientInfo := map[string]interface{}{
				"account_id": client.Payload.AccountID,
				"region":     client.Payload.Region,
			}
			clients = append(clients, clientInfo)
		}
		types.ClientM.RUnlock()

		ip := utilities.Get[string]("ip")
		resp, err := http.Get(fmt.Sprintf("http://%s:2087/nxa/echo/metrics/clients", ip))
		externalClients := []map[string]interface{}{}
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			if err := json.NewDecoder(resp.Body).Decode(&externalClients); err != nil {
				externalClients = []map[string]interface{}{}
			}
		}
		for _, extClient := range externalClients {
			clientInfo := map[string]interface{}{
				"account_id": extClient["account_id"],
				"region":     extClient["region"],
			}
			clients = append(clients, clientInfo)
		}

		resp, err = http.Get(fmt.Sprintf("http://%s:2087/nxa/echo/metrics/sessions", ip))
		externalSessions := []map[string]interface{}{}
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			if err := json.NewDecoder(resp.Body).Decode(&externalSessions); err != nil {
				externalSessions = []map[string]interface{}{}
			}
		}

		sessions = make([]map[string]interface{}, 0)
		for _, ext := range externalSessions {
			createdAt, _ := time.Parse(time.RFC3339Nano, fmt.Sprintf("%v", ext["created_at"]))
			timeRemaining := 7*time.Minute - time.Since(createdAt)
			if timeRemaining < 0 {
				timeRemaining = 0
			}
			serverInfo := map[string]interface{}{
				"session_id": ext["id"],
				"region":     ext["ServerRegion"],
				"playlist":   ext["Playlist"],
				"count":      ext["ActivePlayers"],
				"assigned":   ext["Joinable"],
				"created_at": ext["created_at"],
				"remaining":  int(timeRemaining.Seconds()),
			}
			sessions = append(sessions, serverInfo)
		}

		c.JSON(http.StatusOK, gin.H{
			"clients":  clients,
			"sessions": sessions,
		})
	})
}
