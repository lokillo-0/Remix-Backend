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

func POSTSetItemFavoriteStatusBatch(c *gin.Context) {
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
		ItemIds       []string `json:"itemIds" binding:"required"`
		ItemFavStatus []bool   `json:"itemFavStatus" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		utilities.MCP.InvalidPayload().WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	if len(body.ItemIds) != len(body.ItemFavStatus) {
		utilities.MCP.InvalidPayload().WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	if len(body.ItemIds) == 0 {
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

	profileChanges := make([]map[string]interface{}, 0, len(body.ItemIds))
	failedItems := make([]string, 0)

	for i, itemID := range body.ItemIds {
		isFavorite := body.ItemFavStatus[i]

		item, exists := profile.Items[itemID]
		if !exists {
			if !hasFullLocker {
				utilities.MCP.ItemNotFound().WithIntent(utilities.Prod).Apply(c.Writer)
				return
			}

			itemValue, ok := GetAthenaCachedItems()[itemID]
			if !ok {
				failedItems = append(failedItems, itemID)
				continue
			}

			item = map[string]interface{}{
				"attributes": itemValue.Attributes,
				"templateId": itemID,
				"quantity":   itemValue.Quantity,
			}
			profile.Items[itemID] = item
		}

		itemData := item.(map[string]interface{})

		attributes, ok := itemData["attributes"].(map[string]interface{})
		if !ok {
			attributes = make(map[string]interface{})
			itemData["attributes"] = attributes
		}

		attributes["favorite"] = isFavorite
		profile.Items[itemID] = itemData

		profileChanges = append(profileChanges, map[string]interface{}{
			"changeType":     "itemAttrChanged",
			"itemId":         itemID,
			"attributeName":  "favorite",
			"attributeValue": isFavorite,
		})
	}

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
