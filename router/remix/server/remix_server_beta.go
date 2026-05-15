package remix_server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
)

func GETBetaCheck(c *gin.Context) {
	cfg := utilities.GetConfig()

	if len(cfg.BETA_ROLE_IDS) == 0 {
		c.JSON(http.StatusOK, gin.H{"access": true})
		return
	}

	token := c.GetHeader("Authorization")
	if len(token) > 7 && (token[:7] == "bearer " || token[:7] == "Bearer ") {
		token = token[7:]
	}
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"access": false})
		return
	}

	sess, _ := odin.FindWhere("Accounts_Sessions", map[string]interface{}{"token": token}, func() interface{} {
		return &accounts.Session{}
	})
	if len(sess) == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"access": false})
		return
	}

	var account accounts.Account
	if err := odin.Find("Accounts", sess[0].(*accounts.Session).AccountID, &account); err != nil || account.DiscordID == "" {
		c.JSON(http.StatusForbidden, gin.H{"access": false})
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET",
		fmt.Sprintf("https://discord.com/api/v10/guilds/%s/members/%s", cfg.GUILD_ID, account.DiscordID), nil)
	req.Header.Set("Authorization", "Bot "+cfg.DISCORD_BotToken)
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		c.JSON(http.StatusForbidden, gin.H{"access": false})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var member struct {
		Roles []string `json:"roles"`
	}
	json.Unmarshal(body, &member)

	for _, betaRole := range cfg.BETA_ROLE_IDS {
		for _, userRole := range member.Roles {
			if betaRole == userRole {
				c.JSON(http.StatusOK, gin.H{"access": true})
				return
			}
		}
	}

	c.JSON(http.StatusForbidden, gin.H{"access": false})
}
