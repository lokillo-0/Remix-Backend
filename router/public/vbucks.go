package public

import (
	"net/http"
	"strings"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	fortnite_mcp "github.com/remixfn/xenon/router/fortnite/mcp"
)

func resolveSession(c *gin.Context) (*accounts.Session, bool) {
	tokenHeader := c.GetHeader("Authorization")
	if tokenHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization"})
		return nil, false
	}
	token := strings.TrimPrefix(strings.TrimPrefix(tokenHeader, "Bearer "), "bearer ")

	sessions, err := odin.FindWhere("Accounts_Sessions", map[string]interface{}{
		"token": token,
	}, func() interface{} { return &accounts.Session{} })
	if err != nil || len(sessions) == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return nil, false
	}
	return sessions[0].(*accounts.Session), true
}

func GETPublicVbucks(c *gin.Context) {
	sess, ok := resolveSession(c)
	if !ok {
		return
	}

	profileKey := sess.AccountID + ":common_core"
	var profile accounts.Profile
	if err := odin.Find("Accounts_Profiles", profileKey, &profile); err != nil {
		c.JSON(http.StatusOK, gin.H{"vbucks": 0})
		return
	}

	vbucks := 0
	if profile.Items != nil {
		if vbRaw, ok := profile.Items["Currency:MtxPurchased"]; ok {
			if vbMap, ok := vbRaw.(map[string]interface{}); ok {
				if qty, ok := vbMap["quantity"]; ok {
					switch v := qty.(type) {
					case float64:
						vbucks = int(v)
					case int:
						vbucks = v
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"vbucks": vbucks})
}

func GETPublicLocker(c *gin.Context) {
	sess, ok := resolveSession(c)
	if !ok {
		return
	}

	var athena accounts.Profile
	athenaKey := sess.AccountID + ":athena"
	if err := odin.Find("Accounts_Profiles", athenaKey, &athena); err != nil {
		c.JSON(http.StatusOK, gin.H{"items": []string{}})
		return
	}

	owned := make([]string, 0, len(athena.Items))
	for templateID := range athena.Items {
		owned = append(owned, templateID)
	}

	if fortnite_mcp.HasFullLockerReward(sess.AccountID) {
		cacheItems := fortnite_mcp.GetAthenaCacheKeys()
		seen := make(map[string]bool, len(owned))
		for _, id := range owned {
			seen[id] = true
		}
		for _, id := range cacheItems {
			if !seen[id] {
				owned = append(owned, id)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"items": owned})
}
