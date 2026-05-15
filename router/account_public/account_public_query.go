package account_public

import (
	"net/http"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
)

func GETAccountPublicAccountIDQuery(c *gin.Context) {
	accountIDs := c.QueryArray("accountId")

	if len(accountIDs) == 0 {
		utilities.Account.InvalidAccountIdCount().Apply(c.Writer)
		return
	}

	response := make([]gin.H, 0)
	for _, accountID := range accountIDs {
		var account accounts.Account
		if err := odin.Find("Accounts", accountID, &account); err != nil {
			continue
		}

		response = append(response, gin.H{
			"id":            account.ID,
			"displayName":   account.DisplayName,
			"cabinedMode":   false,
			"externalAuth":  gin.H{},
			"minorVerified": false,
			"minorExpected": false,
			"minorStatus":   "NOT_MINOR",
		})
	}

	c.JSON(http.StatusOK, response)
}
