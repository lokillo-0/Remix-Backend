package remix_server_fortnite

import (
	"fmt"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
)

func POSTWonGame(c *gin.Context) {
	if utilities.GetConfig().VBUCKS_PER_WIN <= 0 {
		c.JSON(200, gin.H{"granted": 0})
		return
	}

	accountId := c.Param("accountId")

	profileKey := fmt.Sprintf("%s:common_core", accountId)
	var profile accounts.Profile
	if err := odin.Find("Accounts_Profiles", profileKey, &profile); err != nil {
		c.JSON(404, gin.H{"error": "profile not found"})
		return
	}

	vbRaw, exists := profile.Items["Currency:MtxPurchased"]
	if !exists {
		c.JSON(404, gin.H{"error": "vbucks item not found"})
		return
	}

	vbucks, ok := vbRaw.(map[string]interface{})
	if !ok {
		c.JSON(500, gin.H{"error": "invalid vbucks item"})
		return
	}

	current := 0
	if qty, ok := vbucks["quantity"].(float64); ok {
		current = int(qty)
	} else if qty, ok := vbucks["quantity"].(int); ok {
		current = qty
	}

	vbucks["quantity"] = current + utilities.GetConfig().VBUCKS_PER_WIN
	profile.Items["Currency:MtxPurchased"] = vbucks
	profile.Revision++

	if err := profile.Bucket.Save(profile); err != nil {
		c.JSON(500, gin.H{"error": "failed to save profile"})
		return
	}

	c.JSON(200, gin.H{"granted": utilities.GetConfig().VBUCKS_PER_WIN})
}
