package remix_server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
)

type Woah struct {
	Status     string                 `json:"Status"`
	Properties map[string]interface{} `json:"Properties"`
}

type Presence struct {
	Online bool        `json:"online"`
	Status interface{} `json:"status"`
}

func getPresenceStatus(accountId string) ([]Presence, error) {
	resp, err := http.Get("http://127.0.0.1:4040/friends/status/" + accountId)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result []Presence
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func getDisplayName(p Presence) string {
	statusArr, ok := p.Status.([]interface{})
	if !ok || len(statusArr) == 0 {
		return ""
	}

	obj, ok := statusArr[0].(map[string]interface{})
	if !ok {
		return ""
	}

	props, ok := obj["Properties"].(map[string]interface{})
	if !ok {
		return ""
	}

	for key, val := range props {
		if strings.HasPrefix(key, "party.joininfodata") {
			if data, ok := val.(map[string]interface{}); ok {
				if name, ok := data["sourceDisplayName"].(string); ok {
					return name
				}
			}
		}
	}

	return ""
}

func GETLauncherFriends(c *gin.Context) {
	accountId := c.Param("accountId")

	allFriends, err := odin.FindWhere("Accounts_Friends", map[string]interface{}{
		"accountId": accountId,
	}, func() interface{} { return &accounts.Friends{} })

	if err != nil {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}

	allPresences, err := getPresenceStatus(accountId)

	presenceMap := map[string]Presence{}

	for _, p := range allPresences {
		name := getDisplayName(p)
		if name != "" {
			presenceMap[strings.ToLower(name)] = p
		}
	}

	result := []gin.H{}
	for _, f := range allFriends {
		friend, ok := f.(*accounts.Friends)
		if !ok || friend.Status != "ACCEPTED" {
			continue
		}

		var friendAccount accounts.Account
		if err := odin.Find("Accounts", friend.FriendId, &friendAccount); err != nil {
			continue
		}

		presence := presenceMap[strings.ToLower(friendAccount.DisplayName)]

		statusStr := ""
		switch v := presence.Status.(type) {

		case string:
			statusStr = v

		case []interface{}:
			if len(v) > 0 {
				if obj, ok := v[0].(map[string]interface{}); ok {
					if s, ok := obj["Status"].(string); ok {
						statusStr = s
					}
				}
			}
		}

		result = append(result, gin.H{
			"accountId":       friendAccount.ID,
			"displayName":     friendAccount.DisplayName,
			"profile_picture": friendAccount.ProfilePicture,
			"online":          presence.Online,
			"status":          statusStr,
		})
	}

	c.JSON(http.StatusOK, result)
}
