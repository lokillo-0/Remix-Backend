package oauth

import (
	"strings"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/modules/database/buckets/fortnite"
	"github.com/remixfn/xenon/utilities"
)

func POSTCreateOAUTHExchange(c *gin.Context) {
	tokenHeader := c.GetHeader("Authorization")
	if tokenHeader == "" {
		utilities.Authentication.InvalidHeader().Apply(c.Writer)
		c.Abort()
		return
	}

	token := strings.ReplaceAll(tokenHeader, "Bearer ", "")
	token = strings.ReplaceAll(token, "bearer ", "")

	var sessionData *accounts.Session
	var account accounts.Account

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
		if err := odin.Find("Accounts", sessionData.Bucket.ID, &account); err != nil {
			utilities.Account.AccountNotFound().Apply(c.Writer)
			return
		}
	}

	if account.Banned {
		utilities.Account.DisabledAccount().Apply(c.Writer)
		c.Abort()
		return
	}

	code := fortnite.Exchange{
		Bucket: odin.Bucket{
			ID: uuid.New().String(),
		},
		Code:      uuid.New().String(),
		AccountID: account.ID,
		Created:   time.Now().Format(time.RFC3339),
	}

	if err := odin.Create(&code); err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		c.Abort()
		return
	}

	c.JSON(200, gin.H{
		"code":             code,
		"creatingClientId": "ec684b8c687f479fadea3cb2ad83f5c6",
		"expiresInSeconds": 300,
	})
}
