package iridium

import (
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
)

func POSTIridiumCreateAuth(c *gin.Context) {
	accountId := c.Param("accountid")
	var account accounts.Account
	if err := odin.Find("Accounts", accountId, &account); err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	if account.Banned {
		utilities.Account.DisabledAccount().Apply(c.Writer)
		return
	}

	payload := jwt.MapClaims{
		"iss": "https://xenon-api-v1.nxa.app",
		"iat": int64(time.Now().Unix()),
		"exp": int64(time.Now().Add(time.Hour * 12).Unix()),
		"aud": "Iridium",
		"sub": accountId,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	str, err := token.SignedString([]byte(utilities.Get[string]("i_jwt")))
	if err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	c.String(200, str)
}

func GETIridiumGetUser(c *gin.Context) {
	accountId := c.Param("accountid")

	var account accounts.Account
	if err := odin.Find("Accounts", accountId, &account); err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	if account.Banned {
		utilities.Account.DisabledAccount().Apply(c.Writer)
		return
	}

	c.JSON(200, gin.H{
		"id":          account.ID,
		"displayName": account.DisplayName,
		"server":      account.IsServer,
		"banned":      account.Banned,
	})
}
