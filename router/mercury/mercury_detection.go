package mercury

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
)

func POSTMercuryDetection(c *gin.Context) {
	var body map[string]string
	if err := c.BindJSON(&body); err != nil {
		utilities.Internal.ValidationFailed().
			WithMessage("Invalid request body").
			WithIntent(utilities.Prod).Apply(c.Writer)

		return
	}

	var account accounts.Account
	if err := odin.Find("Accounts", body["ID"], &account); err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	if account.Banned {
		utilities.Account.DisabledAccount().Apply(c.Writer)
		return
	}

	autoBan := true
	if strings.Contains(body["message"], "Unsigned dll loaded") {
		autoBan = false
	}

	if autoBan {
		account.Banned = true
		account.MatchmakingBannedReason = body["message"]
		account.Bucket.Save(account)
	}

	reqBody := map[string]string{
		"DisplayName": account.DisplayName,
		"Message":     body["message"],
		"AutoBanned":  strconv.FormatBool(autoBan),
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	resp, err := http.Post("http://127.0.0.1:3000/bot/discord/mercury/detection", "application/json", bytes.NewBuffer(jsonBody))

	if err != nil || resp.StatusCode != 200 {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	c.Status(200)
}
