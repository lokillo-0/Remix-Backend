package fortnite_mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/classes/mcp"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
)

var pricesCache map[string]struct {
	Cost int `json:"Cost"`
}

func POSTExchangeGameCurrencyForBattlePassOffer(c *gin.Context) {
	var response mcp.DefaultMCPResponse
	accountID := c.Param("accountId")
	profileId := c.Query("profileId")

	now := time.Now().UTC().Format("2006-01-02T15:04:05.999Z")

	if profileId == "" || accountID == "" {
		utilities.Internal.ValidationFailed().
			WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	var account accounts.Account
	if err := odin.Find("Accounts", accountID, &account); err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	profileKey := fmt.Sprintf("%s:%s", accountID, profileId)
	var profile accounts.Profile
	profileFound := true
	if err := odin.Find("Accounts_Profiles", profileKey, &profile); err != nil {
		profileFound = false
	}

	var body struct {
		OfferItemIdList []string `json:"offerItemIdList"`
	}

	if err := c.BindJSON(&body); err != nil {
		utilities.Basic.BadRequest().Apply(c.Writer)
		return
	}

	ua := utilities.Parse(c.GetHeader("User-Agent"))
	if ua == nil {
		utilities.Internal.InvalidUserAgent().
			WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	data, err := LoadPassData(ua.Season)
	if err != nil {
		utilities.Basic.NotFound().Apply(c.Writer)
		return
	}

	if pricesCache == nil {
		priceData, err := os.ReadFile("static/battlepass/prices.json")
		if err != nil {
			utilities.Basic.BadRequest().Apply(c.Writer)
			return
		}
		json.Unmarshal(priceData, &pricesCache)
	}

	var Offers []BPOffer
	totalCost := 0
	for _, offerID := range body.OfferItemIdList {
		if offer, exists := data[offerID]; exists {
			if offer.OfferPriceRowHandle.RowName != nil {
				rowName := offer.OfferPriceRowHandle.RowName.(string)
				if priceData, priceExists := pricesCache[rowName]; priceExists {
					totalCost += priceData.Cost
				}
			}
			Offers = append(Offers, offer)
		}
	}

	if len(Offers) == 0 {
		utilities.Storefront.InvalidItem().Apply(c.Writer)
		return
	}

	if !profileFound {
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
		profileChanges := []map[string]interface{}{}
		battleStars := profile.Stats["battlestars"].(float64)

		if battleStars < float64(totalCost) {
			utilities.Storefront.CurrencyInsufficient().Apply(c.Writer)
			return
		}

		if battleStars < float64(totalCost) {
			utilities.Storefront.CurrencyInsufficient().Apply(c.Writer)
			return
		}

		stats := profile.Stats
		if _, ok := stats["purchased_bp_offers"]; !ok {
			stats["purchased_bp_offers"] = []interface{}{}
		}

		var purchasedOffers []interface{}
		if val, ok := stats["purchased_bp_offers"]; ok {
			if arr, ok := val.([]interface{}); ok {
				purchasedOffers = arr
			} else {
				purchasedOffers = []interface{}{}
			}
		} else {
			purchasedOffers = []interface{}{}
		}

		var unownedOffers []BPOffer

		for _, offer := range Offers {
			assetPath := offer.AssetPathName
			if assetPath[len(assetPath)-1] == '.' {
				assetPath = assetPath[:len(assetPath)-1]
			}
			parts := strings.Split(assetPath, "/")
			lastPart := parts[len(parts)-1]
			idParts := strings.Split(lastPart, ".")

			var itemID string
			if len(idParts) > 1 {
				itemID = idParts[1]
			} else {
				itemID = lastPart
			}

			if _, exists := profile.Items[itemID]; exists {
				utilities.Storefront.AlreadyOwned().Apply(c.Writer)
				return
			}
			offer.SubPathString = itemID
			unownedOffers = append(unownedOffers, offer)
		}

		newBattleStars := battleStars - float64(totalCost)
		profile.Stats["battlestars"] = newBattleStars

		profileChanges = append(profileChanges, map[string]interface{}{
			"changeType": "statModified",
			"name":       "battlestars",
			"value":      newBattleStars,
		})

		offerIndex := 0
		for _, offer := range unownedOffers {
			offerID := body.OfferItemIdList[offerIndex]
			offerIndex++

			offerCost := 0
			if offer.OfferPriceRowHandle.RowName != nil {
				rowName := offer.OfferPriceRowHandle.RowName.(string)
				if priceData, priceExists := pricesCache[rowName]; priceExists {
					offerCost = priceData.Cost
				}
			}

			var templateID string
			for cachedTemplateID := range GetAthenaCachedItems() {
				if strings.Contains(cachedTemplateID, offer.SubPathString) {
					templateID = cachedTemplateID
					break
				}
			}

			purchaseEntry := map[string]interface{}{
				"offerId":           offerID,
				"bIsFreePassReward": false,
				"purchaseDate":      time.Now().Format(time.RFC3339),
				"lootResult": []interface{}{map[string]interface{}{
					"itemType":    templateID,
					"itemGuid":    templateID,
					"itemProfile": profileId,
					"quantity":    offer.Quantity,
				}},
				"currencyType":      "battlestars",
				"totalCurrencyPaid": offerCost,
			}

			if templateID == "Currency:mtxgiveaway" {
				commonCoreKey := fmt.Sprintf("%s:common_core", accountID)
				var commonCore accounts.Profile
				if err := odin.Find("Accounts_Profiles", commonCoreKey, &commonCore); err == nil {
					if mtxPurchased, exists := commonCore.Items["Currency:MtxPurchased"]; exists {
						if mtxItem, ok := mtxPurchased.(map[string]interface{}); ok {
							if qty, ok := mtxItem["quantity"].(float64); ok {
								mtxItem["quantity"] = qty + float64(offer.Quantity)
							} else {
								mtxItem["quantity"] = offer.Quantity
							}
							commonCore.Items["Currency:MtxPurchased"] = mtxItem
							profileChanges = append(profileChanges, map[string]interface{}{
								"changeType": "itemQuantityChanged",
								"itemId":     "Currency:MtxPurchased",
								"quantity":   mtxItem["quantity"],
							})
							commonCore.Bucket.Save(commonCore)
						}
					}
				}
			} else if strings.Contains(templateID, "VTID") {
				parts := strings.Split(offer.AssetPathName, ".")
				variantToken, _ := GetCosmeticVariantToken(parts[1])
				if variantToken.TemplateID != "" {
					if _, exists := profile.Items[variantToken.TemplateID]; !exists {
						if item, exists := GetAthenaCachedItems()[variantToken.TemplateID]; exists {
							profile.Items[variantToken.TemplateID] = map[string]interface{}{
								"templateId":  variantToken.TemplateID,
								"purchasedAt": time.Now().UTC().Format(time.RFC3339),
								"attributes":  item.Attributes,
								"quantity":    offer.Quantity,
							}

							profileChanges = append(profileChanges, map[string]interface{}{
								"changeType": "itemAdded",
								"itemId":     variantToken.TemplateID,
								"item": map[string]interface{}{
									"templateId":  variantToken.TemplateID,
									"purchasedAt": time.Now().UTC().Format(time.RFC3339),
									"attributes":  item.Attributes,
									"quantity":    offer.Quantity,
								},
							})
						}
					}
				}
			} else {
				if item, exists := GetAthenaCachedItems()[templateID]; exists {
					profile.Items[templateID] = map[string]interface{}{
						"templateId":  templateID,
						"purchasedAt": time.Now().UTC().Format(time.RFC3339),
						"attributes":  item.Attributes,
						"quantity":    offer.Quantity,
					}

					profileChanges = append(profileChanges, map[string]interface{}{
						"changeType": "itemAdded",
						"itemId":     templateID,
						"item": map[string]interface{}{
							"templateId":  templateID,
							"purchasedAt": time.Now().UTC().Format(time.RFC3339),
							"attributes":  item.Attributes,
							"quantity":    offer.Quantity,
						},
					})

					for _, chainedReward := range offer.ChainedRewards {
						for chainTemplateID := range GetAthenaCachedItems() {
							if strings.Contains(chainTemplateID, chainedReward.SubPathString) {
								profile.Items[chainTemplateID] = map[string]interface{}{
									"templateId":  chainTemplateID,
									"purchasedAt": time.Now().UTC().Format(time.RFC3339),
									"attributes":  item.Attributes,
									"quantity":    chainedReward.Quantity,
								}

								profileChanges = append(profileChanges, map[string]interface{}{
									"changeType": "itemAdded",
									"itemId":     chainTemplateID,
									"item": map[string]interface{}{
										"templateId":  chainTemplateID,
										"purchasedAt": time.Now().UTC().Format(time.RFC3339),
										"attributes":  item.Attributes,
										"quantity":    chainedReward.Quantity,
									},
								})
								break
							}
						}
					}
				}
			}

			purchasedOffers = append(purchasedOffers, purchaseEntry)
		}

		stats["purchased_bp_offers"] = purchasedOffers
		profile.Stats = stats

		profileChanges = append(profileChanges, map[string]interface{}{
			"changeType": "statModified",
			"name":       "purchased_bp_offers",
			"value":      purchasedOffers,
		})

		if profile.Bucket.ID != profileKey {
			profile.Bucket.ID = profileKey
		}

		profile.Revision++
		profile.Bucket.Save(profile)

		var lootItems []map[string]interface{}
		for _, change := range profileChanges {
			if change["changeType"] == "itemAdded" {
				if item, ok := change["item"].(map[string]interface{}); ok {
					lootItems = append(lootItems, map[string]interface{}{
						"itemType":    item["templateId"],
						"itemGuid":    change["itemId"],
						"itemProfile": profileId,
						"quantity":    item["quantity"],
					})
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"profileRevision":            profile.Revision,
			"profileId":                  profileId,
			"profileChangesBaseRevision": profile.Revision - 1,
			"profileChanges":             profileChanges,
			"profileCommandRevision":     profile.Revision,
			"serverTime":                 now,
			"responseVersion":            1,
			"notifications": []gin.H{
				{
					"type":    "CatalogPurchase",
					"primary": true,
					"lootResult": gin.H{
						"items": lootItems,
					},
				},
			},
		})
		return
	}

	c.JSON(http.StatusOK, response)
}
