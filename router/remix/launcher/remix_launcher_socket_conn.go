package remix_launcher

import (
	"context"
	"log"
	"net/url"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/modules/database/buckets/remix"
	fortnite_mcp "github.com/remixfn/xenon/router/fortnite/mcp"
	"github.com/remixfn/xenon/utilities"
	"golang.org/x/crypto/bcrypt"
)

func HandleLauncherSocketConnection(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			return
		}
	}()

	updates, err := odin.FindAll("Remix_Updates", func() any {
		return &remix.Update{}
	})

	if err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	if len(updates) == 0 {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	newest := updates[0].(*remix.Update)
	for _, u := range updates {
		update, ok := u.(*remix.Update)
		if !ok {
			continue
		}
		if update.Version >= newest.Version {
			newest = update
		}
	}

	ua := c.Request.Header.Get("User-Agent")
	var ver string
	const prefix = "RemixSocket/"
	if len(ua) > len(prefix) && ua[:len(prefix)] == prefix {
		ver = ua[len(prefix):]
	}

	if ver != newest.Version && !utilities.GetConfig().Maintenance {
		utilities.Internal.ServerError().WithMessage("You are using an outdated version of the Remix Launcher. Please update to the latest version.").Apply(c.Writer)
		return
	}

	email, _ := url.QueryUnescape(c.Query("email"))
	password, _ := url.QueryUnescape(c.Query("password"))

	if email == "" || password == "" {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	users, err := odin.FindWhere("Accounts", map[string]interface{}{
		"email": email,
	}, func() interface{} {
		return &accounts.Account{}
	})

	if err != nil || len(users) == 0 {
		utilities.Authentication.OAuth.InvalidAccountCredentials().Apply(c.Writer)
		return
	}

	account := users[0].(*accounts.Account)

	if account == nil {
		utilities.Authentication.OAuth.InvalidAccountCredentials().Apply(c.Writer)
		return
	}

	if account.Banned {
		utilities.Authentication.AuthenticationFailed().Apply(c.Writer)
		return
	}

	if !fortnite_mcp.HasAccess(account.ID) {
		utilities.Authentication.AuthenticationFailed().WithMessage("you must have beta to use the launcher currently.").Apply(c.Writer)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(password)); err != nil {
		utilities.Authentication.OAuth.InvalidAccountCredentials().Apply(c.Writer)
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	ClientsMutex.Lock()
	if LauncherClients == nil {
		LauncherClients = make(map[string]*LauncherSocketClient)
	}

	if existingClient, exists := LauncherClients[account.ID]; exists {
		if existingClient.Conn != nil {
			existingClient.Conn.Close()
		}
		if existingClient.Cancel != nil {
			existingClient.Cancel()
		}
		delete(LauncherClients, account.ID)
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &LauncherSocketClient{
		Conn:    conn,
		Account: account,
		Cancel:  cancel,
		Done:    make(chan bool, 1),
	}

	LauncherClients[account.ID] = client
	ClientsMutex.Unlock()

	account.LastLoginIP = c.ClientIP()
	account.LastLoginTime = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	account.Bucket.Save(account)

	conn.SetCloseHandler(func(code int, text string) error {
		cancel()
		ClientsMutex.Lock()
		delete(LauncherClients, account.ID)
		ClientsMutex.Unlock()
		select {
		case client.Done <- true:
		default:
		}
		return nil
	})

	go func() {
		defer func() {
			if r := recover(); r != nil {
			}

			cancel()
			conn.Close()

			ClientsMutex.Lock()
			delete(LauncherClients, account.ID)
			ClientsMutex.Unlock()

			select {
			case client.Done <- true:
			default:
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case <-client.Done:
				return
			default:
				messageType, message, err := conn.ReadMessage()
				if err != nil {
					if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
						continue
					}
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						log.Printf("socket error for account %s: %v", account.ID, err)
					}
					return
				}

				HandleLauncherWebsocketMessage(client, messageType, message)
			}
		}
	}()
}
