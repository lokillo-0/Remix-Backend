package matchmaking_socket

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	matchmaking_handlers "github.com/remixfn/xenon/modules/matchmaking/handlers"
	"github.com/remixfn/xenon/modules/matchmaking/messages"
	"github.com/remixfn/xenon/modules/matchmaking/states"
	"github.com/remixfn/xenon/modules/matchmaking/types"
	"github.com/remixfn/xenon/modules/matchmaking/utils"
	"github.com/remixfn/xenon/utilities"
)

func HandleSocket(c *gin.Context) {
	if c.Request.ProtoMajor == 2 {
		c.Request.Proto = "HTTP/1.1"
		c.Request.ProtoMajor = 1
		c.Request.ProtoMinor = 1
	}

	w, r := c.Writer, c.Request

	ws, err := types.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	defer func() {
		if ws != nil {
			ws.Close()
		}
	}()

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	authParts := strings.SplitN(authHeader, " ", 4)
	if len(authParts) != 4 || authParts[0] != "Epic-Signed" || authParts[1] != "mms-player" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	token := authParts[2] + "." + strings.SplitN(authParts[3], " ", 2)[0]

	payload, err := utils.VerifyJWT(token, utilities.Get[string]("jwt"))
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	client := &types.Client{
		Conn: ws,
	}

	if bucketID, ok := payload["bucketId"].(string); ok {
		client.Payload.BucketID = bucketID
	}
	if buildUniqueID, ok := payload["buildUniqueId"].(string); ok {
		client.Payload.BuildUniqueID = buildUniqueID
	}
	if exp, ok := payload["exp"].(float64); ok {
		client.Payload.Exp = int64(exp)
	}
	if fillTeam, ok := payload["fillTeam"].(string); ok {
		client.Payload.FillTeam = fillTeam
	}
	if iat, ok := payload["iat"].(float64); ok {
		client.Payload.Iat = int64(iat)
	}
	if jti, ok := payload["jti"].(string); ok {
		client.Payload.Jti = jti
	}
	if partyPlayerIDs, ok := payload["partyPlayerIds"].(string); ok {
		client.Payload.PartyPlayerIDs = partyPlayerIDs
	}
	if playlist, ok := payload["playlist"].(string); ok {
		client.Payload.Playlist = playlist
	}
	if region, ok := payload["region"].(string); ok {
		client.Payload.Region = region
	}
	if version, ok := payload["version"].(string); ok {
		client.Payload.Version = version
	}
	if accountID, ok := payload["accountId"].(string); ok {
		client.Payload.AccountID = accountID
	}

	client.SendQueue = true

	ws.SetReadLimit(512)
	ws.SetReadDeadline(time.Now().Add(120 * time.Second))
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(120 * time.Second))
		return nil
	})

	types.ClientM.Lock()
	types.Clients[client] = true
	types.ClientM.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	cleanup := func() {
		cancel()
		types.ClientM.Lock()
		delete(types.Clients, client)
		types.ClientM.Unlock()
		if ws != nil {
			ws.Close()
			ws = nil
		}
	}

	ticketID := strings.ReplaceAll(uuid.New().String(), "-", "")
	count := matchmaking_handlers.GetAllClientsViaDataLen(client.Payload.Version, client.Payload.Playlist, client.Payload.Region)

	if err := states.SendInitMessages(client, ticketID, count); err != nil {
		cleanup()
		return
	}

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	queueTicker := time.NewTicker(100 * time.Millisecond)
	defer queueTicker.Stop()

	readErrors := make(chan error, 1)

	go func() {
		defer close(readErrors)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				client.Conn.SetReadDeadline(time.Now().Add(120 * time.Second))
				_, msg, err := client.Conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						readErrors <- err
					}
					return
				}
				if string(msg) == "ping" {
					client.Conn.SetWriteDeadline(time.Now().Add(types.WriteWait))
					client.Conn.WriteMessage(websocket.TextMessage, []byte("pong"))
				}
			}
		}
	}()

	defer cleanup()

	lastSentCount := count

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-readErrors:
			if err != nil {
			}
			return
		case <-pingTicker.C:
			select {
			case <-ctx.Done():
				return
			default:
				if err := client.Conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
					return
				}
			}
		case <-queueTicker.C:
			select {
			case <-ctx.Done():
				return
			default:
			}

			if client.SendQueue {
				types.HTTPServerM.RLock()
				var httpServer *types.HTTPServer
				for _, s := range types.HTTPServers {
					if s.Region == client.Payload.Region {
						httpServer = s
						break
					}
				}
				types.HTTPServerM.RUnlock()

				if httpServer != nil {
					client.SendQueue = false
					go func(srv *types.HTTPServer) {
						messages.SendSessionAssignment(client, srv.SessionID)
						time.Sleep(100 * time.Millisecond)
						messages.SendJoin(client.Conn, srv.SessionID, srv.SessionID)
					}(httpServer)
				} else {
					currentCount := matchmaking_handlers.GetAllClientsViaDataLen(
						client.Payload.Version,
						client.Payload.Playlist,
						client.Payload.Region,
					)

					if currentCount != lastSentCount {
						lastSentCount = currentCount

						var matchedSession *types.Server
						for _, session := range types.Sessions {
							if session != nil &&
								session.Payload.Region == client.Payload.Region &&
								!session.IsSending &&
								!session.IsAssigning {
								matchedSession = session
								break
							}
						}

						if matchedSession != nil {
							client.SendQueue = false
						} else {
							if err := messages.SendQueued(client, ticketID, currentCount); err != nil {
							}
						}
					}
				}
			}
		}
	}
}
