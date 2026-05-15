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

func POSTMarkItemSeen(c *gin.Context) {
	accountID := c.Param("accountId")
	if accountID == "" {
		utilities.MCP.ProfileNotFound().WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	profileId := c.Query("profileId")
	if profileId == "" {
		utilities.Internal.ValidationFailed().WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	var body struct {
		ItemIds []string `json:"itemIds" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		utilities.MCP.InvalidPayload().WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	if len(body.ItemIds) == 0 {
		now := time.Now().UTC().Format("2006-01-02T15:04:05.999Z")
		c.JSON(http.StatusOK, mcp.DefaultMCPResponse{
			ProfileRevision:            0,
			ProfileId:                  profileId,
			ProfileChangesBaseRevision: 0,
			ProfileCommandRevision:     0,
			ServerTime:                 now,
			ResponseVersion:            1,
			ProfileChanges:             []map[string]interface{}{},
		})
		return
	}

	var user accounts.Account
	userErr := odin.Find("Accounts", accountID, &user)

	if userErr != nil {
		utilities.MCP.ProfileNotFound().
			WithMessageVar([]string{accountID}).
			WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	profileKey := fmt.Sprintf("%s:%s", accountID, profileId)
	var profile accounts.Profile
	profErr := odin.Find("Accounts_Profiles", profileKey, &profile)

	now := time.Now().UTC().Format("2006-01-02T15:04:05.999Z")
	var response mcp.DefaultMCPResponse
	updatedCount := 0

	if profErr != nil {
		response = mcp.DefaultMCPResponse{
			ProfileRevision:            0,
			ProfileId:                  profileId,
			ProfileChangesBaseRevision: 0,
			ProfileCommandRevision:     0,
			ServerTime:                 now,
			ResponseVersion:            1,
			ProfileChanges: []map[string]interface{}{
				{
					"changeType": "fullProfileUpdate",
					"profile": mcp.Profile{
						Created:    now,
						Updated:    now,
						Rvn:        0,
						WipeNumber: 1,
						AccountId:  accountID,
						ProfileId:  profileId,
						Version:    "no_version",
						Items:      make(map[string]interface{}),
						Stats: mcp.Stats{
							Attributes: make(map[string]interface{}),
						},
						CommandRevision: 0,
					},
				},
			},
		}
	} else {
		profileChanges := make([]map[string]interface{}, 0, len(body.ItemIds))

		for _, itemId := range body.ItemIds {
			item, exists := profile.Items[itemId]
			if !exists {
				continue
			}

			itemData, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			attributes, ok := itemData["attributes"].(map[string]interface{})
			if !ok {
				attributes = make(map[string]interface{})
				itemData["attributes"] = attributes
			}

			if seen, exists := attributes["item_seen"]; exists && seen == true {
				continue
			}

			attributes["item_seen"] = true
			profile.Items[itemId] = itemData
			updatedCount++

			profileChanges = append(profileChanges, map[string]interface{}{
				"changeType":     "itemAttrChanged",
				"itemId":         itemId,
				"attributeName":  "item_seen",
				"attributeValue": true,
			})
		}

		if updatedCount > 0 {
			profile.Revision++
		}

		response = mcp.DefaultMCPResponse{
			ProfileRevision:            profile.Revision,
			ProfileId:                  profileId,
			ProfileChangesBaseRevision: profile.Revision - 1,
			ProfileCommandRevision:     profile.Revision,
			ServerTime:                 now,
			ResponseVersion:            1,
			ProfileChanges:             profileChanges,
		}
	}

	if profErr == nil && updatedCount > 0 {
		if profile.Bucket.ID != profileKey {
			profile.Bucket.ID = profileKey
		}
		profile.Bucket.Save(profile)
	}

	c.JSON(http.StatusOK, response)
}
