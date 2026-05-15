package discovery

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/utilities"
)

func GETLinksMnemonicPlaylist(c *gin.Context) {
	playlistId := c.Param("playlistId")
	ua := utilities.Parse(c.GetHeader("User-Agent"))
	data, err := ioutil.ReadFile("static/discovery/latest/menu.json")
	if err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	var discoveryv2 []map[string]interface{}
	if err := json.Unmarshal(data, &discoveryv2); err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	if ua != nil {
		activateVersionPlaylists(discoveryv2, ua.Build)
	}

	for _, result := range discoveryv2 {
		if mnemonic, ok := result["mnemonic"].(string); ok && mnemonic == playlistId {
			c.JSON(http.StatusOK, result)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{})
}

func HandleLockStatusCheck(c *gin.Context) {
	accountId := c.Param("accountId")
	body, _ := io.ReadAll(c.Request.Body)

	var req map[string]interface{}
	var results []gin.H

	if json.Unmarshal(body, &req) == nil {
		for _, key := range []string{"linkCodes", "mnemonics", "link_codes"} {
			if list, ok := req[key].([]interface{}); ok {
				for _, v := range list {
					if code, ok := v.(string); ok {
						results = append(results, gin.H{
							"playerId":         accountId,
							"linkCode":         code,
							"lockStatus":       "UNLOCKED",
							"lockStatusReason": "NONE",
							"isVisible":        true,
						})
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"hasMore": false,
	})
}
