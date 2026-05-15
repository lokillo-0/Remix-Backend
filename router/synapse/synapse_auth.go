package synapse

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
)

func Auth(c *gin.Context) {
	id := c.Param("id")
	password := c.Query("password")

	var account accounts.Account
	if err := odin.Find("Accounts", id, &account); err != nil {
	} else {
		c.Status(http.StatusNoContent)
		return
	}

	if password == "" {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	parts := strings.Split(password, ".")
	if len(parts) < 2 {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	payload := parts[1]
	payload += strings.Repeat("=", (4-len(payload)%4)%4)
	payload = strings.ReplaceAll(payload, "-", "+")
	payload = strings.ReplaceAll(payload, "_", "/")

	decodedBytes, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	dataStr := string(decodedBytes)

	var displayName string
	if idx := strings.Index(dataStr, "display_name"); idx != -1 {

		start := idx + len("display_name")
		for start < len(dataStr) && (dataStr[start] == 'm' || dataStr[start] == 0 || dataStr[start] < 32) {
			start++
		}

		if start < len(dataStr) {
			end := start
			for end < len(dataStr) && dataStr[end] != 0 && dataStr[end] >= 32 && dataStr[end] <= 126 {
				end++
			}

			if end > start {
				displayName = dataStr[start:end]
				displayName = strings.TrimSuffix(displayName, ".d")
				displayName = strings.TrimSpace(displayName)
			}
		}
	}

	if displayName == "" {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	users, err := odin.FindWhere("Accounts", map[string]interface{}{
		"display_name": displayName,
	}, func() interface{} {
		return &accounts.Account{}
	})

	if err != nil || len(users) == 0 {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	c.Status(http.StatusNoContent)
}

func GetUsernameViaAccountID(c *gin.Context) {
	id := c.Param("id")

	var account accounts.Account
	if err := odin.Find("Accounts", id, &account); err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	c.JSON(http.StatusOK, gin.H{"username": account.DisplayName})
}
