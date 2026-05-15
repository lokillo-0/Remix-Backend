package fortnite

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/classes/storefront"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/modules/database/buckets/fortnite"
	fortnite_mcp "github.com/remixfn/xenon/router/fortnite/mcp"
	"github.com/remixfn/xenon/utilities"
)

var (
	catalogCache     []byte
	catalogCacheMu   sync.RWMutex
	catalogCacheTime time.Time
	catalogCacheTTL  = 5 * time.Minute
)

func invalidateCatalog() {
	catalogCacheMu.Lock()
	catalogCacheTime = time.Time{}
	catalogCacheMu.Unlock()
}

var bpCatalogData []storefront.CatalogEntry
var bpCatalogOnce sync.Once

func getBPCatalog() []storefront.CatalogEntry {
	bpCatalogOnce.Do(func() {
		fileContent, err := os.ReadFile("static/storefront/bp-catalog.json")
		if err != nil {
			return
		}
		json.Unmarshal(fileContent, &bpCatalogData)
	})
	return bpCatalogData
}

func Catalog(c *gin.Context) {
	catalogCacheMu.RLock()
	fresh := time.Since(catalogCacheTime) < catalogCacheTTL
	cached := catalogCache
	catalogCacheMu.RUnlock()
	if fresh && cached != nil {
		c.Data(http.StatusOK, "application/json", cached)
		return
	} else {
		invalidateCatalog()
	}

	var weeklyStorefront, dailyStorefront []fortnite.Catalog
	var mu sync.Mutex
	var fetchErr error

	var athenaProfile *accounts.Profile

	tokenHeader := c.GetHeader("Authorization")
	if tokenHeader != "" {
		token := strings.ReplaceAll(tokenHeader, "bearer ", "")
		session, _ := odin.FindWhere("Accounts_Sessions", map[string]interface{}{
			"token": token,
		}, func() interface{} {
			return &accounts.Session{}
		})

		if len(session) > 0 {
			sessionData := session[0].(*accounts.Session)
			var account accounts.Account
			if err := odin.Find("Accounts", sessionData.AccountID, &account); err == nil {
				athenaKey := fmt.Sprintf("%s:athena", sessionData.AccountID)
				odin.Find("Accounts_Profiles", athenaKey, &athenaProfile)
			}
		}
	}

	fetchStorefront := func(storefrontName string, result *[]fortnite.Catalog) {
		found, err := odin.FindWhere("Catalog", map[string]interface{}{
			"storefront": storefrontName,
		}, func() interface{} {
			return &fortnite.Catalog{}
		})
		if err != nil {
			mu.Lock()
			if fetchErr == nil {
				fetchErr = fmt.Errorf("failed to fetch %s: %w", storefrontName, err)
			}
			mu.Unlock()
			return
		}
		if found != nil {
			mu.Lock()
			for _, v := range found {
				if catalog, ok := v.(*fortnite.Catalog); ok {
					*result = append(*result, *catalog)
				}
			}
			mu.Unlock()
		}
	}

	fetchStorefront("BRWeeklyStorefront", &weeklyStorefront)
	fetchStorefront("BRDailyStorefront", &dailyStorefront)

	if fetchErr != nil {
		utilities.Internal.ServerError().
			WithIntent(utilities.Prod).
			Apply(c.Writer)
		return
	}

	now := time.Now().UTC()
	currentDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	nextDayMidnight := currentDay.AddDate(0, 0, 1).Format(time.RFC3339)

	globalSeenOffers := make(map[string]bool)

	convertEntries := func(entries []fortnite.Catalog, seenOffers map[string]bool) []gin.H {
		result := make([]gin.H, 0, len(entries))

		for _, entry := range entries {
			var catalogEntry storefront.CatalogEntry
			if err := json.Unmarshal([]byte(entry.Data), &catalogEntry); err != nil {
				continue
			}

			var uniqueID string
			switch {
			case catalogEntry.OfferId != "":
				uniqueID = catalogEntry.OfferId
			case catalogEntry.DevName != "":
				uniqueID = catalogEntry.DevName
			case len(catalogEntry.ItemGrants) > 0:
				uniqueID = catalogEntry.ItemGrants[0].TemplateId
			default:
				jsonData, _ := json.Marshal(catalogEntry)
				uniqueID = fmt.Sprintf("hash_%x", sha256.Sum256(jsonData))
			}

			if seenOffers[uniqueID] {
				continue
			}
			seenOffers[uniqueID] = true

			if catalogEntry.OfferType == "DynamicBundle" && catalogEntry.DynamicBundleInfo != nil && athenaProfile != nil {
				bundleInfo := catalogEntry.DynamicBundleInfo
				totalPrice := bundleInfo.RegularBasePrice + bundleInfo.DiscountedBasePrice
				ownedItemsValue := 0
				newBundleItems := make([]storefront.DynamicBundleItem, 0, len(bundleInfo.BundleItems))
				_newItemGrants := make([]storefront.ItemGrant, 0, len(catalogEntry.ItemGrants))

				for _, bundleItem := range bundleInfo.BundleItems {
					newBundleItem := bundleItem
					if _, exists := athenaProfile.Items[bundleItem.Item.TemplateId]; exists {
						ownedItemsValue += bundleItem.AlreadyOwnedPriceReduction
						newBundleItem.RegularPrice = 0
						newBundleItem.DiscountedPrice = 0
					}
					newBundleItems = append(newBundleItems, newBundleItem)
				}

				_newItemGrants = catalogEntry.ItemGrants

				finalPrice := totalPrice - ownedItemsValue
				if finalPrice < bundleInfo.FloorPrice {
					finalPrice = bundleInfo.FloorPrice
				}

				catalogEntry.DynamicBundleInfo.BundleItems = newBundleItems
				catalogEntry.DynamicBundleInfo.DiscountedBasePrice = -finalPrice
				catalogEntry.DynamicBundleInfo.FloorPrice = totalPrice - ownedItemsValue
				catalogEntry.ItemGrants = _newItemGrants

				allItems := []string{}
				for _, bundleItem := range bundleInfo.BundleItems {
					parts := strings.Split(bundleItem.Item.TemplateId, ":")
					if len(parts) > 1 {
						allItems = append(allItems, "1 x "+parts[1])
					} else {
						allItems = append(allItems, "1 x "+bundleItem.Item.TemplateId)
					}
				}
				if len(allItems) > 0 {
					catalogEntry.DevName = fmt.Sprintf("[VIRTUAL]%s for %d MtxCurrency", strings.Join(allItems, ", "), finalPrice)
				}
			}

			jsonData, err := json.Marshal(catalogEntry)
			if err != nil {
				continue
			}
			var entryMap gin.H
			if err := json.Unmarshal(jsonData, &entryMap); err != nil {
				continue
			}
			result = append(result, entryMap)
		}
		return result
	}

	catalogData := getBPCatalog()

	response := gin.H{
		"refreshIntervalHrs": 24,
		"dailyPurchaseHrs":   24,
		"expiration":         nextDayMidnight,
		"storefronts": []gin.H{
			{
				"name":           "BRDailyStorefront",
				"catalogEntries": convertEntries(dailyStorefront, globalSeenOffers),
			},
			{
				"name":           "BRWeeklyStorefront",
				"catalogEntries": convertEntries(weeklyStorefront, globalSeenOffers),
			},
			{
				"name":           "BRSeason32",
				"catalogEntries": catalogData,
			},
		},
	}

	data, _ := json.Marshal(response)
	catalogCacheMu.Lock()
	catalogCache = data
	catalogCacheTime = time.Now()
	catalogCacheMu.Unlock()

	c.JSON(http.StatusOK, response)
}

func CheckEligibility(c *gin.Context) {
	recipientId := c.Param("recipientId")
	offerId := strings.TrimPrefix(c.Param("offerId"), "/")

	if recipientId == "" || offerId == "" {
		utilities.Basic.NotAcceptable().
			WithIntent(utilities.Prod).
			Apply(c.Writer)
		return
	}

	catalogs, catErr := odin.FindWhere("Catalog", map[string]interface{}{
		"offerId": offerId,
	}, func() interface{} { return &fortnite.Catalog{} })

	if catErr != nil || len(catalogs) == 0 {
		utilities.Storefront.InvalidItem().
			WithIntent(utilities.Prod).
			Apply(c.Writer)
		return
	}
	catalog := catalogs[0].(*fortnite.Catalog)

	offerData := fortnite_mcp.OfferPool.Get().(*fortnite_mcp.CatalogOffer)
	defer fortnite_mcp.OfferPool.Put(offerData)

	if err := json.Unmarshal([]byte(catalog.Data), offerData); err != nil {
		utilities.Storefront.InvalidItem().
			WithIntent(utilities.Prod).
			Apply(c.Writer)
		return
	}

	recipientAthenaKey := fmt.Sprintf("%s:athena", recipientId)
	var recipientProfile accounts.Profile
	if err := odin.Find("Accounts_Profiles", recipientAthenaKey, &recipientProfile); err != nil {
		utilities.MCP.ProfileNotFound().
			WithIntent(utilities.Prod).
			Apply(c.Writer)
		return
	}

	var itemsToCheck []struct {
		TemplateID string
		Quantity   int
		Attributes map[string]interface{}
	}

	if offerData.OfferType == "DynamicBundle" && offerData.DynamicBundleInfo != nil {
		for _, bundleItem := range offerData.DynamicBundleInfo.BundleItems {
			itemsToCheck = append(itemsToCheck, struct {
				TemplateID string
				Quantity   int
				Attributes map[string]interface{}
			}{
				TemplateID: bundleItem.Item.TemplateID,
				Quantity:   bundleItem.Item.Quantity,
				Attributes: bundleItem.Item.Attributes,
			})
		}
	} else {
		for _, itemGrant := range offerData.ItemGrants {
			itemsToCheck = append(itemsToCheck, struct {
				TemplateID string
				Quantity   int
				Attributes map[string]interface{}
			}{
				TemplateID: itemGrant.TemplateID,
				Quantity:   itemGrant.Quantity,
				Attributes: itemGrant.Attributes,
			})
		}
	}

	for _, item := range itemsToCheck {
		if _, exists := recipientProfile.Items[item.TemplateID]; exists {
			utilities.Storefront.AlreadyOwned().
				WithIntent(utilities.Prod).
				Apply(c.Writer)
			return
		}
	}

	var priceInfo interface{}
	var itemGrants interface{}

	if len(offerData.Prices) > 0 {
		priceInfo = offerData.Prices[0]
	}

	if offerData.OfferType == "DynamicBundle" && offerData.DynamicBundleInfo != nil {
		var availableItems []interface{}
		for _, bundleItem := range offerData.DynamicBundleInfo.BundleItems {
			if _, exists := recipientProfile.Items[bundleItem.Item.TemplateID]; !exists {
				availableItems = append(availableItems, bundleItem.Item)
			}
		}
		itemGrants = availableItems
	} else {
		itemGrants = offerData.ItemGrants
	}

	c.JSON(http.StatusOK, gin.H{
		"price": priceInfo,
		"items": itemGrants,
	})
}
