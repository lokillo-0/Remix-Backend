package mercury

import (
	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
)

func POSTGetMercuryUser(c *gin.Context) {
	var body struct {
		Token string `json:"token"`
		HWID  string `json:"hwid"`
	}
	if err := c.BindJSON(&body); err != nil {
		utilities.Internal.ValidationFailed().
			WithMessage("Invalid request body").
			WithIntent(utilities.Prod).Apply(c.Writer)

		return
	}

	token, err := jwt.Parse(body.Token, func(token *jwt.Token) (interface{}, error) {
		return []byte(utilities.Get[string]("jwt")), nil
	})
	if err != nil {
		utilities.Authentication.InvalidToken().Apply(c.Writer)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		utilities.Authentication.InvalidToken().Apply(c.Writer)
		return
	}

	accountId, ok := claims["sub"].(string)
	if !ok || accountId == "" {
		utilities.Authentication.InvalidToken().Apply(c.Writer)
		return
	}

	var account accounts.Account
	if err := odin.Find("Accounts", accountId, &account); err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	if account.Banned {
		utilities.Account.DisabledAccount().Apply(c.Writer)
		return
	}

	account.HWID = body.HWID
	account.Bucket.Save(account)

	matchingHwids, err := odin.FindWhere("Accounts", map[string]interface{}{
		"hwid": body.HWID,
	}, func() interface{} {
		return &accounts.Account{}
	})

	for _, acc := range matchingHwids {
		accountPtr, ok := acc.(*accounts.Account)
		if ok && accountPtr.Banned {
			utilities.Account.DisabledAccount().Apply(c.Writer)
			return
		}
	}

	c.JSON(200, gin.H{
		"id":          account.ID,
		"displayName": account.DisplayName,
		"server":      account.IsServer,
		"banned":      account.Banned,
	})
}
