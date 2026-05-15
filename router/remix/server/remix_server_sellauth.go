package remix_server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
)

type sellAuthWebhook struct {
	ID       int    `json:"id"`
	Event    string `json:"event"`
	Customer struct {
		DiscordID       string `json:"discord_id"`
		DiscordUsername string `json:"discord_username"`
		Email           string `json:"email"`
	} `json:"customer"`
}

func parseSellAuthWebhook(c *gin.Context) (*sellAuthWebhook, string, bool) {
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return nil, "", false
	}

	os.WriteFile("/root/sellauth_payload.json", raw, 0644)

	var payload sellAuthWebhook
	if err := json.Unmarshal(raw, &payload); err != nil {
		c.Status(http.StatusBadRequest)
		return nil, "", false
	}

	if payload.Customer.DiscordID == "" {
		c.Status(http.StatusBadRequest)
		return nil, "", false
	}

	results, err := odin.FindWhere("Accounts", map[string]interface{}{
		"discord_id": payload.Customer.DiscordID,
	}, func() interface{} { return &accounts.Account{} })

	if err != nil || len(results) == 0 {
		accountID := strings.ReplaceAll(uuid.New().String(), "-", "")
		displayName := payload.Customer.DiscordUsername
		if displayName == "" {
			displayName = "user_" + payload.Customer.DiscordID
		}
		newAccount := &accounts.Account{
			Bucket:                  odin.Bucket{ID: accountID},
			Created:                 time.Now(),
			Email:                   fmt.Sprintf("discord_%s@remix.gg", payload.Customer.DiscordID),
			Password:                "",
			DisplayName:             displayName,
			Username:                displayName,
			Banned:                  false,
			Roles:                   []string{"-1"},
			BanHistory:              []map[string]string{},
			IsServer:                false,
			LastLoginTime:           time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
			ProfilePicture:          "https://cdn.discordapp.com/avatars//",
			DisplayNameChanges:      0,
			MatchmakingBannedUntil:  "",
			MatchmakingBannedSince:  "",
			MatchmakingBannedReason: "",
			DiscordID:               payload.Customer.DiscordID,
		}
		if err := odin.Create(newAccount); err != nil {
			c.Status(http.StatusInternalServerError)
			return nil, "", false
		}
		fmt.Printf("[SellAuth] Auto-created account %s for Discord %s\n", accountID, payload.Customer.DiscordID)
		return &payload, accountID, true
	}

	accountID := results[0].(*accounts.Account).ID
	return &payload, accountID, true
}

func grantFullLocker(c *gin.Context, payload *sellAuthWebhook, accountID string) {
	existing, _ := odin.FindWhere("Accounts_Rewards", map[string]interface{}{
		"account_id": accountID,
	}, func() interface{} { return &accounts.AccountReward{} })
	for _, r := range existing {
		if reward, ok := r.(*accounts.AccountReward); ok && reward.Redeemed {
			for _, item := range reward.Rewards {
				if item == "Full Locker" {
					c.Status(http.StatusOK)
					return
				}
			}
		}
	}

	reward := &accounts.AccountReward{
		Bucket:    odin.Bucket{ID: uuid.New().String()},
		AccountID: accountID,
		CodeID:    fmt.Sprintf("sellauth-%d", payload.ID),
		Rewards:   []string{"Full Locker"},
		Redeemed:  true,
	}
	if err := odin.Create(reward); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	fmt.Printf("[SellAuth] Full Locker granted to account %s (Discord: %s)\n", accountID, payload.Customer.DiscordID)
	c.Status(http.StatusOK)
}

func POSTSellAuthWebhook(c *gin.Context) {
	payload, accountID, ok := parseSellAuthWebhook(c)
	if !ok {
		return
	}
	grantFullLocker(c, payload, accountID)
}
