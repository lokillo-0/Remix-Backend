package iridium

import (
	"log"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
)

func POSTIridiumMain(c *gin.Context) {
	var body map[string]interface{}

	if err := c.BindJSON(&body); err != nil {
		utilities.Internal.ValidationFailed().Apply(c.Writer)
		return
	}

	action := c.Param("action")
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

	switch action {
	case "detection":
		EnumFloat, enumOk := body["Enum"].(float64)
		CallerAddress, callerOk := body["CallerAddress"].(string)
		if !enumOk || !callerOk || CallerAddress == "" {
			log.Printf("Invalid request body: %v", body)
			log.Printf("Enum: %v, CallerAddress: %v", EnumFloat, CallerAddress)
			utilities.Internal.ValidationFailed().Apply(c.Writer)
			return
		}
		account.Banned = true
		account.Bucket.Save(account)
		c.JSON(200, nil)
	case "ban":
		if !account.Banned {
			account.Banned = true
			account.Bucket.Save(account)
			c.JSON(200, nil)
		}
	case "matchmakingban":
		if account.MatchmakingBannedReason == "" {
			account.MatchmakingBannedSince = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
			account.MatchmakingBannedUntil = time.Now().UTC().Add(24 * time.Hour).Format("2006-01-02T15:04:05.000Z")
			account.MatchmakingBannedReason = body["reason"].(string)
			account.Bucket.Save(account)
			c.JSON(200, nil)
		}
	case "unmatchmakingban":
		if account.MatchmakingBannedReason != "" {
			account.MatchmakingBannedSince = ""
			account.MatchmakingBannedUntil = ""
			account.MatchmakingBannedReason = ""
			account.Bucket.Save(account)
			c.JSON(200, nil)
		}
	case "unban":
		if account.Banned {
			account.Banned = false
			account.MatchmakingBannedReason = ""
			account.Bucket.Save(account)
			c.JSON(200, nil)
		}
	default:
		break
	}
}
