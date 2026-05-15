package fortnite_mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
)

func POSTPurchaseMultipleCatalogEntries(c *gin.Context) {
	accountID := c.Param("accountId")
	profileID := c.Query("profileId")

	if profileID == "" {
		utilities.Basic.NotAcceptable().Apply(c.Writer)
		return
	}

	var body map[string]interface{}
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		utilities.Basic.NotAcceptable().Apply(c.Writer)
		return
	}

	// Extract from purchaseInfoList[0]
	var offerID string
	purchaseQuantity := 1

	if pil, ok := body["purchaseInfoList"].([]interface{}); ok && len(pil) > 0 {
		if entry, ok := pil[0].(map[string]interface{}); ok {
			if oid, ok := entry["offerId"].(string); ok {
				offerID = oid
			}
			if q, ok := entry["purchaseQuantity"].(float64); ok && q > 0 {
				purchaseQuantity = int(q)
			}
		}
	}

	if offerID == "" {
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

	bpOffer, err := loadBPCatalogOffer(offerID)
	if err != nil {
		utilities.Storefront.InvalidItem().Apply(c.Writer)
		return
	}

	if len(bpOffer.Prices) == 0 || bpOffer.Prices[0].CurrencyType != "MtxCurrency" {
		utilities.Storefront.InvalidItem().Apply(c.Writer)
		return
	}

	finalPrice := bpOffer.Prices[0].FinalPrice * purchaseQuantity

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
	grantBP := false

	switch offerID {
	case "E25779A44C8313E402485FBF9793F075":
		grantBP = true
	case "2384AFC64CFB17EDDC8FBEA7F95F5443":
		grantBP = true
		levelsToGrant = 25 * purchaseQuantity
	case "6AE70FFA47481618E7F570855213C6E4":
		levelsToGrant = 1 * purchaseQuantity
	case "98EBDE5A4E2C1266D78B97ADC559BA14":
		levelsToGrant = 25 * purchaseQuantity
	default:
		utilities.Storefront.InvalidItem().Apply(c.Writer)
		return
	}

	currency["quantity"] = currentQty - finalPrice
	profile.Items["Currency:MtxPurchased"] = currency

	if grantBP {
		setBattlePassPurchased(accountID, seasonNum)
	}
	newLevel := 0
	if levelsToGrant > 0 {
		newLevel, _ = grantLevels(accountID, seasonNum, levelsToGrant)
	}

	// Grant battle stars (5 per level) into athena profile stats
	newStars := 0
	if levelsToGrant > 0 {
		currentStars := 0
		if s, ok := athena.Stats["battlestars"].(float64); ok {
			currentStars = int(s)
		}
		newStars = currentStars + 5*levelsToGrant
		athena.Stats["battlestars"] = float64(newStars)
		if total, ok := athena.Stats["battlestars_season_total"].(float64); ok {
			athena.Stats["battlestars_season_total"] = total + float64(5*levelsToGrant)
		} else {
			athena.Stats["battlestars_season_total"] = float64(5 * levelsToGrant)
		}
	}

	profile.Revision++
	athena.Revision++
	profile.Bucket.Save(profile)
	athena.Bucket.Save(athena)

	athenaChanges := []map[string]interface{}{}
	if grantBP {
		athenaChanges = append(athenaChanges, map[string]interface{}{
			"changeType": "statModified", "name": "book_purchased", "value": true,
		})
	}
	if levelsToGrant > 0 {
		athenaChanges = append(athenaChanges, map[string]interface{}{
			"changeType": "statModified", "name": "level", "value": newLevel,
		})
		athenaChanges = append(athenaChanges, map[string]interface{}{
			"changeType": "statModified", "name": "battlestars", "value": float64(newStars),
		})
	}

	profileChanges := []map[string]interface{}{{
		"changeType": "itemQuantityChanged",
		"itemId":     "Currency:MtxPurchased",
		"quantity":   currency["quantity"],
	}}

	c.JSON(http.StatusOK, gin.H{
		"profileRevision":            profile.Revision,
		"profileId":                  profileID,
		"profileChangesBaseRevision": profile.Revision - 1,
		"profileChanges":             profileChanges,
		"notifications": []gin.H{{
			"type": "CatalogPurchase", "primary": true,
			"lootResult": gin.H{"items": []interface{}{}},
		}},
		"profileCommandRevision": profile.Revision,
		"serverTime":             time.Now().UTC().Format("2006-01-02T15:04:05.999Z"),
		"multiUpdate": []gin.H{{
			"profileRevision":            athena.Revision,
			"profileId":                  "athena",
			"profileChangesBaseRevision": athena.Revision - 1,
			"profileChanges":             athenaChanges,
			"profileCommandRevision":     athena.Revision,
		}},
		"responseVersion": 1,
	})
}
