package fortnite_mcp

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/modules/database/buckets/fortnite"
	"github.com/remixfn/xenon/modules/synapse"
	"github.com/remixfn/xenon/utilities"
)

func POSTGiftCatalogEntry(c *gin.Context) {
	sm := synapse.GetStartedInstance()
	if sm == nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	accountId := c.Param("accountId")

	if !IsAthenaCacheLoaded() {
		utilities.Internal.ServerError().
			WithIntent(utilities.Prod).
			Apply(c.Writer)
		return
	}

	profileId := c.Query("profileId")
	if profileId == "" {
		utilities.MCP.ProfileNotFound().
			WithIntent(utilities.Prod).
			Apply(c.Writer)
		return
	}

	var body struct {
		OfferId            string   `json:"offerId"`
		ReceiverAccountIds []string `json:"receiverAccountIds"`
		PersonalMessage    string   `json:"personalMessage"`
		GiftWrapTemplateId string   `json:"giftWrapTemplateId"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		utilities.MCP.InvalidPayload().
			WithIntent(utilities.Prod).
			Apply(c.Writer)
		return
	}

	if len(body.ReceiverAccountIds) == 0 {
		utilities.MCP.InvalidPayload().
			WithIntent(utilities.Prod).
			Apply(c.Writer)
		return
	}

	profileKey := fmt.Sprintf("%s:%s", accountId, profileId)
	var profile accounts.Profile
	if err := odin.Find("Accounts_Profiles", profileKey, &profile); err != nil {
		utilities.MCP.ProfileNotFound().WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	athenaKey := fmt.Sprintf("%s:athena", accountId)
	var athena accounts.Profile
	if err := odin.Find("Accounts_Profiles", athenaKey, &athena); err != nil {
		utilities.MCP.ProfileNotFound().WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	if profile.Bucket.ID != profileKey {
		profile.Bucket.ID = profileKey
	}

	if athena.Bucket.ID != athenaKey {
		athena.Bucket.ID = athenaKey
	}

	catalogs, catErr := odin.FindWhere("Catalog", map[string]interface{}{
		"offerId": body.OfferId,
	}, func() interface{} { return &fortnite.Catalog{} })

	var catalog *fortnite.Catalog
	if catErr != nil || len(catalogs) == 0 {
		utilities.Storefront.InvalidItem().Apply(c.Writer)
		return
	}
	catalog = catalogs[0].(*fortnite.Catalog)

	offerData := OfferPool.Get().(*CatalogOffer)
	defer OfferPool.Put(offerData)

	if err := json.Unmarshal([]byte(catalog.Data), offerData); err != nil {
		utilities.Storefront.InvalidItem().Apply(c.Writer)
		return
	}

	currencyItem, exists := profile.Items["Currency:MtxPurchased"]
	if !exists {
		currencyItem = map[string]interface{}{
			"templateId": "Currency:MtxPurchased",
			"attributes": map[string]interface{}{
				"level":    1,
				"platform": "EpicPC",
			},
			"quantity": 0,
		}
		profile.Items["Currency:MtxPurchased"] = currencyItem
	}

	currency := currencyItem.(map[string]interface{})
	currentQuantity := int(currency["quantity"].(float64))

	var finalPrice int
	var currencyType string
	var itemsToGrant []struct {
		TemplateID string
		Quantity   int
		Attributes map[string]interface{}
	}

	if offerData.OfferType == "DynamicBundle" && offerData.DynamicBundleInfo != nil {
		bundleInfo := offerData.DynamicBundleInfo
		totalPrice := bundleInfo.RegularBasePrice + bundleInfo.DiscountedBasePrice
		ownedItemsValue := 0

		for _, bundleItem := range bundleInfo.BundleItems {
			if _, exists := athena.Items[bundleItem.Item.TemplateID]; exists {
				ownedItemsValue += bundleItem.AlreadyOwnedPriceReduction
			} else {
				itemsToGrant = append(itemsToGrant, struct {
					TemplateID string
					Quantity   int
					Attributes map[string]interface{}
				}{
					TemplateID: bundleItem.Item.TemplateID,
					Quantity:   bundleItem.Item.Quantity,
					Attributes: bundleItem.Item.Attributes,
				})
			}
		}

		finalPrice = totalPrice - ownedItemsValue
		if finalPrice < bundleInfo.FloorPrice {
			finalPrice = bundleInfo.FloorPrice
		}

		if len(itemsToGrant) == 0 {
			utilities.Storefront.AlreadyOwned().Apply(c.Writer)
			return
		}

		currencyType = bundleInfo.CurrencyType
	} else {
		if len(offerData.ItemGrants) == 0 {
			utilities.Storefront.InvalidItem().Apply(c.Writer)
			return
		}

		for _, itemGrant := range offerData.ItemGrants {
			if _, exists := athena.Items[itemGrant.TemplateID]; exists {
				utilities.Storefront.AlreadyOwned().Apply(c.Writer)
				return
			}
			itemsToGrant = append(itemsToGrant, struct {
				TemplateID string
				Quantity   int
				Attributes map[string]interface{}
			}{
				TemplateID: itemGrant.TemplateID,
				Quantity:   itemGrant.Quantity,
				Attributes: itemGrant.Attributes,
			})
		}

		finalPrice = int(offerData.Prices[0].FinalPrice)
		currencyType = offerData.Prices[0].CurrencyType
	}

	if currencyType == "MtxCurrency" {
		if currentQuantity < finalPrice {
			utilities.Storefront.CurrencyInsufficient().Apply(c.Writer)
			return
		}
		currency["quantity"] = currentQuantity - finalPrice
		profile.Items["Currency:MtxPurchased"] = currency
	} else if finalPrice > 0 {
		utilities.Storefront.CurrencyInsufficient().Apply(c.Writer)
		return
	}

	profileChanges := make([]map[string]interface{}, 0, 1)
	lootItems := make([]map[string]interface{}, 0, len(itemsToGrant))

	for _, receiverAccountId := range body.ReceiverAccountIds {
		if receiverAccountId != accountId {
			receiverAthenaKey := fmt.Sprintf("%s:athena", receiverAccountId)
			var receiverAthena accounts.Profile
			if err := odin.Find("Accounts_Profiles", receiverAthenaKey, &receiverAthena); err != nil {
				utilities.MCP.ProfileNotFound().WithIntent(utilities.Prod).Apply(c.Writer)
				return
			}

			receiverCommonKey := fmt.Sprintf("%s:common_core", receiverAccountId)
			var receiverCommon accounts.Profile
			if err := odin.Find("Accounts_Profiles", receiverCommonKey, &receiverCommon); err != nil {
				utilities.MCP.ProfileNotFound().WithIntent(utilities.Prod).Apply(c.Writer)
				return
			}

			for _, itemGrant := range itemsToGrant {
				if len(itemGrant.TemplateID) > 6 && itemGrant.TemplateID[:6] == "Athena" {
					newItem := map[string]interface{}{
						"templateId": itemGrant.TemplateID,
						"attributes": map[string]interface{}{
							"max_level_bonus": 0,
							"level":           1,
							"xp":              0,
							"item_seen":       false,
							"variants":        []interface{}{},
							"favorite":        false,
						},
						"quantity": itemGrant.Quantity,
					}

					if itemGrant.Attributes != nil {
						for key, value := range itemGrant.Attributes {
							newItem["attributes"].(map[string]interface{})[key] = value
						}
					}

					receiverAthena.Items[itemGrant.TemplateID] = newItem

					lootItems = append(lootItems, map[string]interface{}{
						"itemType":    itemGrant.TemplateID,
						"itemGuid":    itemGrant.TemplateID,
						"itemProfile": "athena",
						"quantity":    itemGrant.Quantity,
					})
				}
			}

			newItem := map[string]interface{}{
				"templateId": body.GiftWrapTemplateId,
				"attributes": map[string]interface{}{
					"max_level_bonus": 0,
					"fromAccountId":   accountId,
					"lootList":        lootItems,
					"params": gin.H{
						"userMessage": body.PersonalMessage,
					},
					"giftedOn": time.Now().UTC().Format(time.RFC3339),
				},
				"quantity": 1,
			}

			receiverCommon.Items[body.GiftWrapTemplateId] = newItem
			receiverCommon.Revision++
			receiverCommon.Bucket.Save(receiverCommon)
			receiverAthena.Revision++
			receiverAthena.Bucket.Save(receiverAthena)

			payload := map[string]interface{}{
				"type":      "com.epicgames.gift.received",
				"payload":   map[string]interface{}{},
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			if err := sm.SendMessage(receiverAccountId, payload); err != nil {
				log.Printf("Failed to send message: %v\n", err)
			}
		}
	}

	if currencyType == "MtxCurrency" {
		profileChanges = append(profileChanges, map[string]interface{}{
			"changeType": "itemQuantityChanged",
			"itemId":     "Currency:MtxPurchased",
			"quantity":   currency["quantity"],
		})
	}

	profile.Revision++
	athena.Revision++

	profile.Bucket.Save(profile)
	athena.Bucket.Save(athena)

	c.JSON(http.StatusOK, gin.H{
		"profileRevision":            profile.Revision,
		"profileId":                  profileId,
		"profileChangesBaseRevision": profile.Revision - 1,
		"profileChanges":             profileChanges,
		"profileCommandRevision":     profile.Revision,
		"serverTime":                 time.Now().UTC().Format("2006-01-02T15:04:05.999Z"),
		"responseVersion":            1,
	})
}
