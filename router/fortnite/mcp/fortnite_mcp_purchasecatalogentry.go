package fortnite_mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/modules/database/buckets/fortnite"
	"github.com/remixfn/xenon/utilities"
)

type bpStaticOffer struct {
	OfferID   string `json:"offerId"`
	OfferType string `json:"offerType"`
	Prices    []struct {
		CurrencyType string `json:"currencyType"`
		FinalPrice   int    `json:"finalPrice"`
	} `json:"prices"`
}

func grantLevels(accountID string, season int, levels int) (int, int) {
	seasonKey := fmt.Sprintf("%s:%v", accountID, season)
	var s accounts.Season
	odin.Find("Accounts_Seasons", seasonKey, &s)
	if s.Bucket.ID == "" {
		s.Bucket.ID = seasonKey
	}
	s.Level += levels
	s.BookXp += 5 * levels
	s.Bucket.Save(s)
	return s.Level, s.BookXp
}

func setBattlePassPurchased(accountID string, season int) {
	seasonKey := fmt.Sprintf("%s:%v", accountID, season)
	var s accounts.Season
	odin.Find("Accounts_Seasons", seasonKey, &s)
	if s.Bucket.ID == "" {
		s.Bucket.ID = seasonKey
	}
	s.PurchasedVip = true
	s.Bucket.Save(s)
}

func loadBPCatalogOffer(offerID string) (*bpStaticOffer, error) {
	data, err := os.ReadFile("static/storefront/bp-catalog.json")
	if err != nil {
		return nil, err
	}
	var offers []bpStaticOffer
	if err := json.Unmarshal(data, &offers); err != nil {
		return nil, err
	}
	for i := range offers {
		if offers[i].OfferID == offerID {
			return &offers[i], nil
		}
	}
	return nil, fmt.Errorf("offer not found")
}

func POSTPurchaseCatalogEntry(c *gin.Context) {
	accountID := c.Param("accountId")
	profileID := c.Query("profileId")

	if profileID == "" {
		utilities.Basic.NotAcceptable().Apply(c.Writer)
		return
	}

	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		utilities.Basic.NotAcceptable().Apply(c.Writer)
		return
	}

	offerID, ok := body["offerId"].(string)
	if !ok {
		utilities.Basic.NotAcceptable().Apply(c.Writer)
		return
	}

	profileKey := fmt.Sprintf("%s:%s", accountID, profileID)
	var profile accounts.Profile
	if err := odin.Find("Accounts_Profiles", profileKey, &profile); err != nil {
		utilities.MCP.ProfileNotFound().WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	athenaKey := fmt.Sprintf("%s:athena", accountID)
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
		"offerId": offerID,
	}, func() interface{} { return &fortnite.Catalog{} })

	if catErr != nil || len(catalogs) == 0 {
		bpOffer, err := loadBPCatalogOffer(offerID)
		if err != nil {
			utilities.Storefront.InvalidItem().Apply(c.Writer)
			return
		}

		if len(bpOffer.Prices) == 0 || bpOffer.Prices[0].CurrencyType != "MtxCurrency" {
			utilities.Storefront.InvalidItem().Apply(c.Writer)
			return
		}

		finalPrice := bpOffer.Prices[0].FinalPrice

		currencyItem, exists := profile.Items["Currency:MtxPurchased"]
		if !exists {
			utilities.Storefront.CurrencyInsufficient().Apply(c.Writer)
			return
		}
		currency := currencyItem.(map[string]interface{})
		currentQty := int(currency["quantity"].(float64))
		if currentQty < finalPrice {
			utilities.Storefront.CurrencyInsufficient().Apply(c.Writer)
			return
		}

		ua := utilities.Parse(c.GetHeader("User-Agent"))
		seasonNum := 0
		if ua != nil {
			seasonNum = ua.Season
		}

		var levelsToGrant int
		grantPass := false

		switch offerID {
		case "E25779A44C8313E402485FBF9793F075":
			grantPass = true
		case "2384AFC64CFB17EDDC8FBEA7F95F5443":
			grantPass = true
			levelsToGrant = 25
		case "6AE70FFA47481618E7F570855213C6E4":
			levelsToGrant = 1
		case "98EBDE5A4E2C1266D78B97ADC559BA14":
			levelsToGrant = 25
		default:
			utilities.Storefront.InvalidItem().Apply(c.Writer)
			return
		}

		currency["quantity"] = currentQty - finalPrice
		profile.Items["Currency:MtxPurchased"] = currency

		if grantPass {
			setBattlePassPurchased(accountID, seasonNum)
		}
		newLevel, newBookXp := 0, 0
		if levelsToGrant > 0 {
			newLevel, newBookXp = grantLevels(accountID, seasonNum, levelsToGrant)
		}

		lootItems := []interface{}{}
		if grantPass {
			// Load season bp rewards if available (seasons with auto-grants)
			bpData, bpErr := LoadBattlePassData(fmt.Sprintf("s%d", seasonNum))
			if bpErr == nil && len(bpData.Rewards) > 0 {
				giftBoxID := "GiftBox:gb_battlepasspurchased"
				lootList := []interface{}{}
				for templateId, quantity := range bpData.Rewards[0] {
					athena.Items[templateId] = map[string]interface{}{
						"templateId": templateId,
						"attributes": map[string]interface{}{"item_seen": false, "variants": []interface{}{}},
						"quantity":   quantity,
					}
					entry := map[string]interface{}{
						"itemType": templateId, "itemGuid": templateId,
						"itemProfile": "athena", "quantity": quantity,
					}
					lootList = append(lootList, entry)
					lootItems = append(lootItems, entry)
				}
				profile.Items[giftBoxID] = map[string]interface{}{
					"templateId": giftBoxID,
					"attributes": map[string]interface{}{
						"max_level_bonus": 0, "fromAccountId": "", "lootList": lootList,
					},
					"quantity": 1,
				}
			}
		}

		profile.Revision++
		athena.Revision++
		profile.Bucket.Save(profile)
		athena.Bucket.Save(athena)

		athenaChanges := []map[string]interface{}{}
		if grantPass {
			athenaChanges = append(athenaChanges, map[string]interface{}{
				"changeType": "statModified",
				"name":       "book_purchased",
				"value":      true,
			})
		}
		if levelsToGrant > 0 {
			athenaChanges = append(athenaChanges, map[string]interface{}{
				"changeType": "statModified",
				"name":       "level",
				"value":      newLevel,
			})
			athenaChanges = append(athenaChanges, map[string]interface{}{
				"changeType": "statModified",
				"name":       "book_xp",
				"value":      newBookXp,
			})
		}

		profileChanges := []map[string]interface{}{
			{
				"changeType": "itemQuantityChanged",
				"itemId":     "Currency:MtxPurchased",
				"quantity":   currency["quantity"],
			},
		}
		if gb, exists := profile.Items["GiftBox:gb_battlepasspurchased"]; exists {
			profileChanges = append(profileChanges, map[string]interface{}{
				"changeType": "itemAdded",
				"itemId":     "GiftBox:gb_battlepasspurchased",
				"item":       gb,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"profileRevision":            profile.Revision,
			"profileId":                  profileID,
			"profileChangesBaseRevision": profile.Revision - 1,
			"profileChanges":             profileChanges,
			"notifications": []gin.H{
				{
					"type":       "CatalogPurchase",
					"primary":    true,
					"lootResult": gin.H{"items": lootItems},
				},
			},
			"profileCommandRevision": profile.Revision,
			"serverTime":             time.Now().UTC().Format("2006-01-02T15:04:05.999Z"),
			"multiUpdate": []gin.H{
				{
					"profileRevision":            athena.Revision,
					"profileId":                  "athena",
					"profileChangesBaseRevision": athena.Revision - 1,
					"profileChanges":             athenaChanges,
					"profileCommandRevision":     athena.Revision,
				},
			},
			"responseVersion": 1,
		})
		return
	}

	var catalog *fortnite.Catalog
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

		hasFullLocker := HasFullLockerReward(accountID)
		cacheKeys := GetAthenaCacheKeys()
		cacheSet := make(map[string]bool, len(cacheKeys))
		for _, k := range cacheKeys {
			cacheSet[k] = true
		}

		for _, itemGrant := range offerData.ItemGrants {
			_, inDB := athena.Items[itemGrant.TemplateID]
			inCache := hasFullLocker && cacheSet[itemGrant.TemplateID]
			if inDB || inCache {
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
	athenaChanges := make([]map[string]interface{}, 0, len(itemsToGrant))
	lootItems := make([]map[string]interface{}, 0, len(itemsToGrant))

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

			athena.Items[itemGrant.TemplateID] = newItem
			athenaChanges = append(athenaChanges, map[string]interface{}{
				"changeType": "itemAdded",
				"itemId":     itemGrant.TemplateID,
				"item": map[string]interface{}{
					"templateId": itemGrant.TemplateID,
					"attributes": newItem["attributes"],
					"quantity":   itemGrant.Quantity,
				},
			})

			lootItems = append(lootItems, map[string]interface{}{
				"itemType":    itemGrant.TemplateID,
				"itemGuid":    itemGrant.TemplateID,
				"itemProfile": "athena",
				"quantity":    itemGrant.Quantity,
			})
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
		"profileId":                  profileID,
		"profileChangesBaseRevision": profile.Revision - 1,
		"profileChanges":             profileChanges,
		"notifications": []gin.H{
			{
				"type":    "CatalogPurchase",
				"primary": true,
				"lootResult": gin.H{
					"items": lootItems,
				},
			},
		},
		"profileCommandRevision": profile.Revision,
		"serverTime":             time.Now().UTC().Format("2006-01-02T15:04:05.999Z"),
		"multiUpdate": []gin.H{
			{
				"profileRevision":            athena.Revision,
				"profileId":                  "athena",
				"profileChangesBaseRevision": athena.Revision - 1,
				"profileChanges":             athenaChanges,
				"profileCommandRevision":     athena.Revision,
			},
		},
		"responseVersion": 1,
	})
}
