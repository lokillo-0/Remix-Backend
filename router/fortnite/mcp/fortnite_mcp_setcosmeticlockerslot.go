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

func POSTSetCosmeticLockerSlot(c *gin.Context) {
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
		LockerItem     string `json:"lockerItem" binding:"required"`
		Category       string `json:"category" binding:"required"`
		ItemToSlot     string `json:"itemToSlot"`
		SlotIndex      int    `json:"slotIndex"`
		VariantUpdates []struct {
			Channel string   `json:"channel"`
			Active  string   `json:"active"`
			Owned   []string `json:"owned,omitempty"`
		} `json:"variantUpdates,omitempty"`
		OptLockerUseCountOverride int `json:"optLockerUseCountOverride,omitempty"`
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
	if err := odin.Find("Accounts_Profiles", profileKey, &profile); err != nil {
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

	loadoutItem, exists := profile.Items[body.LockerItem]
	if !exists {
		utilities.MCP.ItemNotFound().WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	loadout := loadoutItem.(map[string]interface{})

	attributes, ok := loadout["attributes"].(map[string]interface{})
	if !ok {
		attributes = make(map[string]interface{})
		loadout["attributes"] = attributes
	}

	lockerSlotsData, ok := attributes["locker_slots_data"].(map[string]interface{})
	if !ok {
		lockerSlotsData = make(map[string]interface{})
		attributes["locker_slots_data"] = lockerSlotsData
	}

	slots, ok := lockerSlotsData["slots"].(map[string]interface{})
	if !ok {
		slots = make(map[string]interface{})
		lockerSlotsData["slots"] = slots
	}

	categoryData, ok := slots[body.Category].(map[string]interface{})
	if !ok {
		categoryData = make(map[string]interface{})
	}

	profileChanges := make([]map[string]interface{}, 0, 2)

	if body.ItemToSlot == "" || (len(body.ItemToSlot) >= 7 && body.ItemToSlot[len(body.ItemToSlot)-7:] == "_random") {
		switch body.Category {
		case "Dance":
			items, ok := categoryData["items"].([]interface{})
			if !ok {
				items = make([]interface{}, 6)
				for i := range items {
					items[i] = ""
				}
			}

			if len(items) <= body.SlotIndex {
				newItems := make([]interface{}, body.SlotIndex+1)
				copy(newItems, items)
				for i := len(items); i < body.SlotIndex; i++ {
					newItems[i] = ""
				}
				items = newItems
			}

			items[body.SlotIndex] = body.ItemToSlot
			categoryData["items"] = items

		case "ItemWrap":
			items, ok := categoryData["items"].([]interface{})
			if !ok {
				items = make([]interface{}, 7)
				for i := range items {
					items[i] = ""
				}
			} else if len(items) < 7 {
				newItems := make([]interface{}, 7)
				copy(newItems, items)
				for i := len(items); i < 7; i++ {
					newItems[i] = ""
				}
				items = newItems
			}

			if body.SlotIndex == -1 {
				for i := range items {
					items[i] = body.ItemToSlot
				}
			} else if body.SlotIndex >= 0 && body.SlotIndex < 7 {
				items[body.SlotIndex] = body.ItemToSlot
			}

			categoryData["items"] = items

		default:
			if body.ItemToSlot != "" {
				categoryData["items"] = []string{body.ItemToSlot}
				if body.Category == "Character" {
					profile.Stats["favorite_character"] = body.ItemToSlot
				}
			} else {
				categoryData["items"] = []string{""}
				if body.Category == "Character" {
					profile.Stats["favorite_character"] = ""
				}
			}
		}
	} else {
		item, exists := profile.Items[body.ItemToSlot]
		if !exists {
			if !HasFullLockerReward(accountID) {
				utilities.MCP.ItemNotFound().WithIntent(utilities.Prod).Apply(c.Writer)
				return
			}

			itemValue, ok := GetAthenaCachedItems()[body.ItemToSlot]
			if !ok {
				utilities.MCP.ItemNotFound().WithIntent(utilities.Prod).Apply(c.Writer)
				return
			}

			item = map[string]interface{}{
				"attributes": itemValue.Attributes,
				"templateId": body.ItemToSlot,
				"quantity":   itemValue.Quantity,
			}
			profile.Items[body.ItemToSlot] = item
		}

		itemData := item.(map[string]interface{})

		switch body.Category {
		case "Dance":
			items, ok := categoryData["items"].([]interface{})
			if !ok {
				items = make([]interface{}, 6)
				for i := range items {
					items[i] = ""
				}
			}

			if len(items) <= body.SlotIndex {
				newItems := make([]interface{}, body.SlotIndex+1)
				copy(newItems, items)
				for i := len(items); i < body.SlotIndex; i++ {
					newItems[i] = ""
				}
				items = newItems
			}

			items[body.SlotIndex] = body.ItemToSlot
			categoryData["items"] = items

		case "ItemWrap":
			items, ok := categoryData["items"].([]interface{})
			if !ok {
				items = make([]interface{}, 7)
				for i := range items {
					items[i] = ""
				}
			} else if len(items) < 7 {
				newItems := make([]interface{}, 7)
				copy(newItems, items)
				for i := len(items); i < 7; i++ {
				}
				items = newItems
			}

			if body.SlotIndex == -1 {
				for i := range items {
					items[i] = body.ItemToSlot
				}
			} else if body.SlotIndex >= 0 && body.SlotIndex < 7 {
				items[body.SlotIndex] = body.ItemToSlot
			}

			categoryData["items"] = items

		default:
			categoryData["items"] = []string{body.ItemToSlot}
			if body.Category == "Character" {
				profile.Stats["favorite_character"] = body.ItemToSlot
			}
		}

		if len(body.VariantUpdates) > 0 {
			itemAttributes, ok := itemData["attributes"].(map[string]interface{})
			if !ok {
				itemAttributes = make(map[string]interface{})
				itemData["attributes"] = itemAttributes
			}
			variants, ok := itemAttributes["variants"].([]interface{})
			if !ok {
				variants = make([]interface{}, 0, len(body.VariantUpdates))
			}
			variantMap := make(map[string]int, len(variants))
			for i, variant := range variants {
				if variantData, ok := variant.(map[string]interface{}); ok {
					if channel, ok := variantData["channel"].(string); ok {
						variantMap[channel] = i
					}
				}
			}

			for _, update := range body.VariantUpdates {
				if index, exists := variantMap[update.Channel]; exists {
					if existingVariant, ok := variants[index].(map[string]interface{}); ok {
						existingVariant["active"] = update.Active
						if len(update.Owned) > 0 {
							ownedSet := make(map[string]bool)
							var existingOwned []interface{}
							if owned, ok := existingVariant["owned"].([]interface{}); ok {
								for _, item := range owned {
									if itemStr, ok := item.(string); ok {
										ownedSet[itemStr] = true
										existingOwned = append(existingOwned, item)
									}
								}
							}
							for _, newItem := range update.Owned {
								if !ownedSet[newItem] {
									existingOwned = append(existingOwned, newItem)
									ownedSet[newItem] = true
								}
							}
							existingVariant["owned"] = existingOwned
						}
					}
				} else {
					variantData := map[string]interface{}{
						"channel": update.Channel,
						"active":  update.Active,
					}
					if len(update.Owned) > 0 {
						owned := make([]interface{}, len(update.Owned))
						for i, item := range update.Owned {
							owned[i] = item
						}
						variantData["owned"] = owned
					}
					variants = append(variants, variantData)
				}
			}
			itemAttributes["variants"] = variants
			profile.Items[body.ItemToSlot] = itemData
			profileChanges = append(profileChanges, map[string]interface{}{
				"changeType":     "itemAttrChanged",
				"itemId":         body.ItemToSlot,
				"attributeName":  "variants",
				"attributeValue": variants,
			})
		}
	}

	slots[body.Category] = categoryData
	lockerSlotsData["slots"] = slots
	attributes["locker_slots_data"] = lockerSlotsData
	loadout["attributes"] = attributes
	profile.Items[body.LockerItem] = loadout

	profileChanges = append(profileChanges, map[string]interface{}{
		"changeType":     "itemAttrChanged",
		"itemId":         body.LockerItem,
		"attributeName":  "locker_slots_data",
		"attributeValue": lockerSlotsData,
	})

	profile.Revision++
	profile.Bucket.Save(profile)

	c.JSON(http.StatusOK, mcp.DefaultMCPResponse{
		ProfileChanges:             profileChanges,
		ProfileId:                  profileID,
		ProfileRevision:            profile.Revision,
		ProfileChangesBaseRevision: profile.Revision - 1,
		ProfileCommandRevision:     profile.Revision,
		ResponseVersion:            1,
		ServerTime:                 time.Now().UTC().Format("2006-01-02T15:04:05.999Z"),
	})
}
