package middleware

import (
	"strings"
	"sync"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	fortnite_mcp "github.com/remixfn/xenon/router/fortnite/mcp"
	"github.com/remixfn/xenon/utilities"
)

type cachedSession struct {
	sessionData *accounts.Session
	account     accounts.Account
	expiry      time.Time
}

var (
	authCache      = make(map[string]*cachedSession)
	authCacheMutex sync.RWMutex
	cacheDuration  = 30 * time.Second
)

func ClientAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenHeader := c.GetHeader("Authorization")
		if tokenHeader == "" {
			utilities.Authentication.InvalidHeader().Apply(c.Writer)
			c.Abort()
			return
		}

		parts := strings.SplitN(tokenHeader, " ", 2)
		token := tokenHeader
		if len(parts) == 2 {
			token = parts[1]
		}

		authCacheMutex.RLock()
		cached, found := authCache[token]
		authCacheMutex.RUnlock()

		var sessionData *accounts.Session
		var account accounts.Account

		if found && cached.expiry.After(time.Now()) {
			account = cached.account
			if account.Banned {
				authCacheMutex.Lock()
				delete(authCache, token)
				authCacheMutex.Unlock()
				utilities.Account.DisabledAccount().Apply(c.Writer)
				c.Abort()
				return
			}
		} else {
			session, _ := odin.FindWhere("Accounts_Sessions", map[string]interface{}{
				"token": token,
			}, func() interface{} {
				return &accounts.Session{}
			})

			if len(session) == 0 {
				utilities.Authentication.InvalidToken().Apply(c.Writer)
				c.Abort()
				return
			}

			sessionData = session[0].(*accounts.Session)

			if err := odin.Find("Accounts", sessionData.AccountID, &account); err != nil {
				utilities.Account.AccountNotFound().Apply(c.Writer)
				return
			}

			authCacheMutex.Lock()
			authCache[token] = &cachedSession{
				sessionData: sessionData,
				account:     account,
				expiry:      time.Now().Add(cacheDuration),
			}
			authCacheMutex.Unlock()
		}

		if account.Banned {
			utilities.Account.DisabledAccount().Apply(c.Writer)
			c.Abort()
			return
		}

		if !account.IsServer {
			if !fortnite_mcp.HasAccess(account.ID) {
				utilities.Authentication.AuthenticationFailed().WithMessage("you must have beta to use remix currently.").Apply(c.Writer)
				return
			}
		}

		c.Set("accountId", account.ID)
		c.Next()
	}
}
