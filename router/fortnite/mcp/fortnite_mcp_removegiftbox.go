package fortnite_mcp

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/classes/mcp"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
)

func POSTRemoveGiftBox(c *gin.Context) {
	accountId := c.Param("accountId")
	profileId := c.DefaultQuery("profileId", "")
	now := time.Now().UTC().Format("2006-01-02T15:04:05.999Z")

	var response mcp.DefaultMCPResponse

	if profileId == "" {
		utilities.MCP.ProfileNotFound().
			WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	db, err := odin.GetNamed("xenon_profiles")
	if err != nil {
		utilities.Internal.ServerError().WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	profileKey := fmt.Sprintf("%s:%s", accountId, profileId)
	var profile accounts.Profile
	profileFound := true
	if err := db.Get("Accounts_Profiles", profileKey, &profile); err != nil {
		profileFound = false
	}

	athenaKey := fmt.Sprintf("%s:athena", accountId)
	var athenaProfile accounts.Profile
	athenaProfileFound := true
	if err := db.Get("Accounts_Profiles", athenaKey, &athenaProfile); err != nil {
		athenaProfileFound = false
	}

	if !profileFound || !athenaProfileFound {
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
						AccountId:  accountId,
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
		body := struct {
			GiftBoxItemId  string   `json:"giftBoxItemId"`
			GiftBoxItemIds []string `json:"giftBoxItemIds"`
		}{}

		profileChanges := []map[string]interface{}{}

		if err := c.BindJSON(&body); err != nil {
			utilities.Internal.ValidationFailed().Apply(c.Writer)
			log.Printf("Failed to bind JSON: %v", err)
			return
		}

		if body.GiftBoxItemId != "" {
			for itemId, itemData := range profile.Items {
				if itemMap, ok := itemData.(map[string]interface{}); ok {
					if itemTemplateId, exists := itemMap["templateId"]; exists {
						if itemTemplateId == body.GiftBoxItemId {
							delete(profile.Items, itemId)
							profileChanges = append(profileChanges, map[string]interface{}{
								"changeType": "itemRemoved",
								"itemId":     itemId,
							})
						}
					}
				}
			}

			for itemId, itemData := range athenaProfile.Items {
				if itemMap, ok := itemData.(map[string]interface{}); ok {
					if itemTemplateId, exists := itemMap["templateId"]; exists {
						if itemTemplateId == body.GiftBoxItemId {
							delete(athenaProfile.Items, itemId)
							profileChanges = append(profileChanges, map[string]interface{}{
								"changeType": "itemRemoved",
								"itemId":     itemId,
							})
						}
					}
				}
			}
		}

		if len(body.GiftBoxItemIds) > 0 {
			for _, templateId := range body.GiftBoxItemIds {
				for itemId, itemData := range profile.Items {
					if itemMap, ok := itemData.(map[string]interface{}); ok {
						if itemTemplateId, exists := itemMap["templateId"]; exists {
							if itemTemplateId == templateId {
								delete(profile.Items, itemId)
								profileChanges = append(profileChanges, map[string]interface{}{
									"changeType": "itemRemoved",
									"itemId":     itemId,
								})
							}
						}
					}
				}

				for itemId, itemData := range athenaProfile.Items {
					if itemMap, ok := itemData.(map[string]interface{}); ok {
						if itemTemplateId, exists := itemMap["templateId"]; exists {
							if itemTemplateId == templateId {
								delete(athenaProfile.Items, itemId)
								profileChanges = append(profileChanges, map[string]interface{}{
									"changeType": "itemRemoved",
									"itemId":     itemId,
								})
							}
						}
					}
				}
			}
		}

		if profile.Bucket.ID != profileKey {
			profile.Bucket.ID = profileKey
		}

		if athenaProfile.Bucket.ID != athenaKey {
			athenaProfile.Bucket.ID = athenaKey
		}

		profile.Revision++
		profile.Bucket.Save(profile)
		athenaProfile.Revision++
		athenaProfile.Bucket.Save(athenaProfile)

		response = mcp.DefaultMCPResponse{
			ProfileChanges:             profileChanges,
			ProfileId:                  profileId,
			ProfileRevision:            profile.Revision,
			ProfileChangesBaseRevision: profile.Revision - 1,
			ProfileCommandRevision:     profile.Revision,
			ServerTime:                 now,
			ResponseVersion:            1,
		}
	}

	c.JSON(http.StatusOK, response)
}
