package matchmaking_socket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/remixfn/xenon/modules/database/buckets/fortnite"
	matchmaking_handlers "github.com/remixfn/xenon/modules/matchmaking/handlers"
	"github.com/remixfn/xenon/modules/matchmaking/messages"
	"github.com/remixfn/xenon/modules/matchmaking/types"
	"github.com/remixfn/xenon/modules/matchmaking/utils"
	"github.com/remixfn/xenon/utilities"
)

func findPlayerInTeams(teams [][][]string, targetAccountID string) bool {
	for _, team := range teams {
		for _, playerEntry := range team {
			for _, accountID := range playerEntry {
				if accountID == targetAccountID {
					return true
				}
			}
		}
	}
	return false
}

func HandleSessionSocket(c *gin.Context) {
	w, r := c.Writer, c.Request

	ws, err := types.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		ws.Close()
		return
	}

	authParts := strings.SplitN(authHeader, " ", 4)
	if len(authParts) != 4 || authParts[0] != "Epic-Signed" || authParts[1] != "Xenon-Sessions" {
		c.AbortWithStatus(http.StatusUnauthorized)
		ws.Close()
		return
	}

	jwt := utilities.Get[string]("jwt")
	payload, err := utils.VerifyJWT(authParts[3], jwt)
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		ws.Close()
		return
	}

	server := &types.Server{
		Conn: ws,
	}

	if bucketID, ok := payload["bucketId"].(string); ok {
		server.Payload.BucketID = bucketID
	}

	if region, ok := payload["region"].(string); ok {
		if region == "NONE" {
			resp, err := http.Get("http://ipwho.is/" + c.ClientIP())
			if err != nil {
				utilities.Internal.ServerError().Apply(c.Writer)
				ws.Close()
				return
			}
			defer resp.Body.Close()

			var ipRes struct {
				ContinentCode string `json:"continent_code"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&ipRes); err != nil {
				utilities.Internal.ServerError().Apply(c.Writer)
				ws.Close()
				return
			}

			if ipRes.ContinentCode == "NA" {
				ipRes.ContinentCode = "NAC"
			}

			server.Payload.Region = ipRes.ContinentCode
		} else {
			server.Payload.Region = region
		}
	}

	if version, ok := payload["version"].(string); ok {
		server.Payload.Version = version
	}
	if buildUniqueID, ok := payload["buildUniqueId"].(string); ok {
		server.Payload.BuildUniqueID = buildUniqueID
	}
	if exp, ok := payload["exp"].(float64); ok {
		server.Payload.Exp = int64(exp)
	}
	if iat, ok := payload["iat"].(float64); ok {
		server.Payload.Iat = int64(iat)
	}
	if jti, ok := payload["jti"].(string); ok {
		server.Payload.Jti = jti
	}

	server.MatchId = uuid.New().String()
	server.IsAssigned = false
	server.IsAssigning = false
	server.StopAllowingConnections = false
	server.Playlist = ""
	server.IsSending = false
	server.AssignMatchSent = false
	server.MinPlayers = 2
	server.MaxPlayers = 0
	server.SessionId = authParts[2]
	server.CreatedAt = time.Now()

	types.ClientM.Lock()
	types.Sessions[server.SessionId] = server
	types.ClientM.Unlock()

	ws.SetReadLimit(int64(types.MaxMessageSize))
	ws.SetPongHandler(func(string) error {
		return nil
	})

	if err := ws.WriteMessage(websocket.TextMessage, []byte(`{"name":"Registered","payload":{}}`)); err != nil {
		ws.Close()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	var cleanupOnce sync.Once

	cleanup := func() {
		cleanupOnce.Do(func() {
			cancel()
			ws.Close()
			types.ClientM.Lock()
			delete(types.Sessions, server.SessionId)
			types.ClientM.Unlock()

			sessions, _ := odin.FindWhere("GameSessions", map[string]interface{}{
				"session_id": server.SessionId,
			}, func() interface{} {
				return &fortnite.Sessions{}
			})
			if len(sessions) > 0 {
				if sesh, ok := sessions[0].(*fortnite.Sessions); ok {
					sesh.Bucket.Delete(sesh)
				}
			}

			wg.Wait()
		})
	}
	defer cleanup()

	ws.SetCloseHandler(func(code int, text string) error {
		cleanup()
		return nil
	})

	wg.Add(1)
	go func() {
		defer wg.Done()

		pingTicker := time.NewTicker(types.PingPeriod)
		playlistTicker := time.NewTicker(50 * time.Millisecond)
		defer pingTicker.Stop()
		defer playlistTicker.Stop()

		for {
			select {
			case <-pingTicker.C:
				ws.SetWriteDeadline(time.Now().Add(types.WriteWait))
				if err := ws.WriteMessage(websocket.PingMessage, nil); err != nil {
					cleanup()
					return
				}
			case <-playlistTicker.C:
				if (!server.IsSending && !server.IsAssigning) || !server.IsAssigned {
					_, _, err := matchmaking_handlers.SelectPlaylist(server.SessionId, server.Payload.Region)
					if err != nil {
						log.Printf("failed to select playlist: %v", err)
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		messageCount := 0
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			ws.SetReadDeadline(time.Now().Add(60 * time.Second))
			_, message, err := ws.ReadMessage()
			if err != nil {
				cleanup()
				return
			}

			messageCount++

			if string(message) == "ping" {
				ws.SetWriteDeadline(time.Now().Add(types.WriteWait))
				if err := ws.WriteMessage(websocket.TextMessage, []byte("pong")); err != nil {
					cleanup()
					return
				}
				continue
			}

			var data map[string]interface{}
			if err := json.Unmarshal(message, &data); err != nil {
				continue
			}

			if messageName, ok := data["name"].(string); ok {
				if messageName == "AssignMatchResult" {
					payload, ok := data["payload"].(map[string]interface{})
					if !ok {
						continue
					}
					if result, ok := payload["result"].(string); ok {
						if result == "failed" {
							cleanup()
							return
						} else if result == "ready" {
							wg.Add(1)
							go func() {
								defer wg.Done()

								select {
								case <-time.After(2 * time.Second):
									server.IsAssigned = true

									clients := matchmaking_handlers.GetAllClientsViaData(
										server.Payload.Version,
										server.Playlist,
										server.Payload.Region,
									)

									for _, client := range clients {
										if client.Payload.AccountID == "" || client.Conn == nil {
											continue
										}

										found := findPlayerInTeams(server.Teams, client.Payload.AccountID)

										if found {
											select {
											case <-time.After(100 * time.Millisecond):
												messages.SendJoin(client.Conn, server.SessionId, server.SessionId)
											case <-ctx.Done():
												return
											}
										}
									}

									select {
									case <-time.After(3 * time.Second):
										server.StopAllowingConnections = true
									case <-ctx.Done():
										return
									}
								case <-ctx.Done():
									return
								}
							}()
						}
					}
				}
			}
		}
	}()

	<-ctx.Done()
}
