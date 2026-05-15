package fortnite_mcp

import (
	"fmt"
	"net/http"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/classes/mcp"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
)


// FortniteEndpointsDocumentation is sus 69420 67 ohiogamer
func POSTSetItemFavoriteStatus(c *gin.Context) { // i have no idea if this works 
	accountID := c.Param("accountId")

	if !IsAthenaCacheLoaded() {
		utilities.Internal.ServerError().WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	profileID := c.Query("profileId")
	if profileID == "" {
		utilities.MCP.ProfileNotFound().WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	var body struct {
		TargetItemID string `json:"targetItemId" binding:"required"`
		BFavorite    bool   `json:"bFavorite"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		utilities.MCP.InvalidPayload().WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	var user accounts.Account
	userErr := odin.Find("Accounts", accountID, &user)

	if userErr != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	profileKey := fmt.Sprintf("%s:%s", accountID, profileID)
	var profile accounts.Profile
	profileFound := true
	if err := odin.Find("Accounts_Profiles", profileKey, &profile); err != nil {
		profileFound = false
	}

	if !profileFound {
		if profileID == "athena" {
			c.JSON(http.StatusOK, mcp.DefaultMCPResponse{
				ProfileChanges:             []map[string]interface{}{},
				ProfileId:                  profileID,
				ProfileRevision:            1,
				ProfileChangesBaseRevision: 0,
				ProfileCommandRevision:     1,
				ResponseVersion:            1,
				ServerTime:                 time.Now().UTC().Format("2006-01-02T15:04:05.999Z"),
			})
			return
		} else {
			utilities.MCP.ProfileNotFound().WithIntent(utilities.Prod).Apply(c.Writer)
			return
		}
	}

	hasFullLocker := HasFullLockerReward(accountID)

	profileChanges := make([]map[string]interface{}, 0, 1)

	item, exists := profile.Items[body.TargetItemID]
	if !exists {
		if !hasFullLocker {
			utilities.MCP.ItemNotFound().WithIntent(utilities.Prod).Apply(c.Writer)
			return
		}

		itemValue, ok := GetAthenaCachedItems()[body.TargetItemID]
		if !ok {
			utilities.MCP.ItemNotFound().WithIntent(utilities.Prod).Apply(c.Writer)
			return
		}

		item = map[string]interface{}{
			"attributes": itemValue.Attributes,
			"templateId": body.TargetItemID,
			"quantity":   itemValue.Quantity,
		}
		profile.Items[body.TargetItemID] = item
	}

	itemData := item.(map[string]interface{})

	attributes, ok := itemData["attributes"].(map[string]interface{})
	if !ok {
		attributes = make(map[string]interface{})
		itemData["attributes"] = attributes
	}

	attributes["favorite"] = body.BFavorite
	profile.Items[body.TargetItemID] = itemData

	profileChanges = append(profileChanges, map[string]interface{}{
		"changeType":     "itemAttrChanged",
		"itemId":         body.TargetItemID,
		"attributeName":  "favorite",
		"attributeValue": body.BFavorite,
	})

	if len(profileChanges) > 0 {
		profile.Revision++
	}

	response := mcp.DefaultMCPResponse{
		ProfileRevision:            profile.Revision,
		ProfileId:                  profileID,
		ProfileChangesBaseRevision: profile.Revision - len(profileChanges),
		ProfileCommandRevision:     profile.Revision,
		ServerTime:                 time.Now().UTC().Format("2006-01-02T15:04:05.999Z"),
		ResponseVersion:            1,
		ProfileChanges:             profileChanges,
	}

	profile.Bucket.Save(profile)

	c.JSON(http.StatusOK, response)
}