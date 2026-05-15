package generation

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/remixfn/xenon/modules/database/buckets/fortnite"
	"github.com/remixfn/xenon/modules/storefront/declarations"
	"github.com/remixfn/xenon/modules/storefront/generation/displayassets"
	"github.com/remixfn/xenon/modules/storefront/models"
	"github.com/remixfn/xenon/modules/storefront/scraper"
	"github.com/remixfn/xenon/modules/storefront/utils"
	"github.com/remixfn/xenon/utilities"
)

var categoryTracker = struct {
	sync.Mutex
	data               map[string][]string
	lastPickaxeRemoval time.Time
}{data: make(map[string][]string)}

var sectionLayoutIDs = struct {
	sync.Mutex
	layouts    map[string]string
	layoutInts map[string]int
}{layouts: make(map[string]string), layoutInts: make(map[string]int)}

func getOrCreateLayoutID(section string) string {
	sectionLayoutIDs.Lock()
	defer sectionLayoutIDs.Unlock()
	if layoutID, exists := sectionLayoutIDs.layouts[section]; exists {
		return layoutID
	}
	uniqueNum := rand.Intn(90) + 10
	layoutID := fmt.Sprintf("%s.%d", strings.ReplaceAll(section, " ", ""), uniqueNum)
	sectionLayoutIDs.layouts[section] = layoutID
	sectionLayoutIDs.layoutInts[section] = uniqueNum
	return layoutID
}

func getOrCreateLayoutIDInt(section string) int {
	sectionLayoutIDs.Lock()
	defer sectionLayoutIDs.Unlock()
	if layoutInt, exists := sectionLayoutIDs.layoutInts[section]; exists {
		return layoutInt
	}
	uniqueNum := rand.Intn(90) + 10
	sectionLayoutIDs.layoutInts[section] = uniqueNum
	sectionLayoutIDs.layouts[section] = fmt.Sprintf("%s.%d", strings.ReplaceAll(section, " ", ""), uniqueNum)
	return uniqueNum
}

func determineItemTileSize(item scraper.Item, section string) string {
	if item.IsBundle {
		return "DoubleWide"
	}
	if section == "Featured" {
		return "Normal"
	}
	// Daily section: outfits get Normal, everything else Small
	itemID := strings.ToLower(item.ID)
	itemType := strings.ToLower(item.Type)
	if strings.Contains(itemID, "cid_") || strings.Contains(itemType, "outfit") || strings.Contains(itemType, "character") {
		return "Normal"
	}
	return "Small"
}

func CreateShopEntry(item scraper.Item, section string, s14above bool) (models.CatalogEntry, error) {
	if item.Item.BackendValue == "AthenaEmoji" || item.Item.BackendValue == "AthenaMusicPack" {
		return models.CatalogEntry{}, fmt.Errorf("this backendValue is not supported")
	}
	if (item.Price == 0 || item.Price < 0 || item.Name == "") && item.Name != "Heartspan" && item.ID != "MusicPack_LavaChicken" {
		return models.CatalogEntry{}, fmt.Errorf("invalid item data")
	}
	entry := createBaseShopEntry(item)
	uniqueLayoutID := getOrCreateLayoutID(section)
	tileSize := determineItemTileSize(item, section)

	entry.DisplayAssetPath = displayassets.SetDisplayAsset("DA_Featured_" + item.ID)
	entry.NewDisplayAssetPath = displayassets.SetNewDisplayAssetPath(item.ID)

	if section == "Featured" && !s14above {
		entry.Meta.DisplayAssetPath = entry.DisplayAssetPath
		entry.Meta.NewDisplayAssetPath = entry.NewDisplayAssetPath
		entry.Meta.SectionId = "Featured"
		entry.Meta.TileSize = tileSize
		entry.Meta.LayoutId = uniqueLayoutID
		entry.Meta.AnalyticOfferGroupId = "Featured"
		categoryOverrides := map[string]string{"Crystal": "Dino Guard"}
		if fortniteIsRetarded, exists := categoryOverrides[item.Name]; exists {
			entry.Categories = []string{fortniteIsRetarded}
		} else {
			entry.Categories = []string{item.Category}
		}
	} else {
		entry.Meta.DisplayAssetPath = entry.DisplayAssetPath
		entry.Meta.NewDisplayAssetPath = entry.NewDisplayAssetPath
		entry.Meta.SectionId = section
		entry.Meta.TileSize = tileSize
		entry.Meta.LayoutId = uniqueLayoutID
		entry.Meta.AnalyticOfferGroupId = section
		categoryOverrides := map[string]string{"Crystal": "Dino Guard"}
		if fortniteIsRetarded, exists := categoryOverrides[item.Name]; exists {
			entry.Categories = []string{fortniteIsRetarded}
		} else {
			if section != "Daily" {
				entry.Categories = []string{item.Category}
			} else {
				entry.Categories = []string{}
			}
		}
	}

	isPickaxe := strings.Contains(strings.ToLower(item.ID), "pickaxe_")
	isCharacter := strings.Contains(strings.ToLower(item.ID), "cid_")

	categoryTracker.Lock()
	if isCharacter {
		categoryTracker.data[item.ID] = entry.Categories
	}
	categoryTracker.Unlock()

	lastRemovalTime := categoryTracker.lastPickaxeRemoval
	oneWeek := 7 * 24 * time.Hour
	now := time.Now()

	if isPickaxe {
		categoryTracker.Lock()
		if now.Sub(lastRemovalTime) >= oneWeek {
			categoryTracker.lastPickaxeRemoval = now
			if item.Set.Value == "Goalbound" {
				entry.Categories = []string{item.Set.Value}
			} else {
				for _, charCategories := range categoryTracker.data {
					if utils.StringSlicesEqual(charCategories, entry.Categories) {
						entry.Categories = []string{}
						break
					}
				}
			}
		}
		categoryTracker.Unlock()
	}

	entry.MetaInfo = append(entry.MetaInfo, models.MetaInfo{Key: "NewDisplayAssetPath", Value: entry.NewDisplayAssetPath})
	entry.MetaInfo = append(entry.MetaInfo, models.MetaInfo{Key: "SectionId", Value: section})
	entry.MetaInfo = append(entry.MetaInfo, models.MetaInfo{Key: "LayoutId", Value: uniqueLayoutID})
	entry.MetaInfo = append(entry.MetaInfo, models.MetaInfo{Key: "TileSize", Value: tileSize})
	entry.MetaInfo = append(entry.MetaInfo, models.MetaInfo{Key: "AnalyticOfferGroupId", Value: section})
	entry.GiftInfo.PurchaseRequirements = append(entry.GiftInfo.PurchaseRequirements, entry.Requirements...)
	return entry, nil
}

func ValidateBundleEntry(entry *models.CatalogEntry) error {
	if entry.OfferType != "DynamicBundle" {
		return fmt.Errorf("bundle must have OfferType 'DynamicBundle'")
	}
	if entry.DynamicBundleInfo == nil {
		return fmt.Errorf("bundle must have DynamicBundleInfo")
	}
	if len(entry.Prices) > 0 {
		return fmt.Errorf("bundles should have empty Prices array")
	}
	if len(entry.DynamicBundleInfo.BundleItems) == 0 {
		return fmt.Errorf("bundle must have at least one bundle item")
	}
	return nil
}

func createBaseBundleEntry(item scraper.Item) models.CatalogEntry {
	return models.CatalogEntry{
		OfferId:      fmt.Sprintf("v2:/%s", uuid.New().String()),
		OfferType:    "DynamicBundle",
		DevName:      fmt.Sprintf("[VIRTUAL] Bundle: %s", item.Name),
		ItemGrants:   []models.ItemGrant{},
		Requirements: []models.Requirement{},
		Categories:   []string{},
		MetaInfo:     []models.MetaInfo{},
		Prices:       []models.Price{},
		GiftInfo: models.GiftInfo{
			BIsEnabled:              true,
			ForcedGiftBoxTemplateId: "",
			PurchaseRequirements:    []models.Requirement{},
			GiftRecordIds:           []string{},
		},
		Meta: models.Meta{
			NewDisplayAssetPath:  "",
			DisplayAssetPath:     "",
			SectionId:            "",
			TileSize:             "DoubleWide",
			LayoutId:             "",
			AnalyticOfferGroupId: "",
		},
		DisplayAssetPath:     "",
		NewDisplayAssetPath:  "",
		Refundable:           true,
		Title:                item.Name,
		Description:          "",
		ShortDescription:     "",
		AppStoreId:           []string{},
		FulfillmentIds:       []interface{}{},
		DailyLimit:           -1,
		WeeklyLimit:          -1,
		MonthlyLimit:         -1,
		SortPriority:         -1,
		CatalogGroupPriority: 0,
		FilterWeight:         0.0,
		BannerOverride:       "",
		MatchFilter:          "",
		AdditionalGrants:     []models.ItemGrant{},
		DynamicBundleInfo:    nil,
	}
}

func CreateBundleEntry(item scraper.Item, section string, s14above bool) (models.CatalogEntry, error) {
	if !item.IsBundle || len(item.BundleItems) == 0 {
		return models.CatalogEntry{}, fmt.Errorf("item is not a bundle or has no bundle items")
	}
	utilities.LogWithTimestamp(color.GreenString, fmt.Sprintf("Creating bundle entry for: %s with %d items", item.Name, len(item.BundleItems)))

	FreeShops := utilities.Get[bool]("free")
	entry := createBaseBundleEntry(item)
	uniqueLayoutID := getOrCreateLayoutID(section)

	var primaryItem scraper.Item
	if len(item.BundleItems) > 0 {
		for i := range item.BundleItems {
			bundleItem := &item.BundleItems[i]
			parts := strings.Split(bundleItem.ID, ":")
			if len(parts) < 2 {
				backendValue := declarations.DetermineItemBackendValue(*bundleItem)
				if backendValue != "" {
					bundleItem.ID = fmt.Sprintf("%s:%s", backendValue, bundleItem.ID)
				}
			}
		}
		for _, bundleItem := range item.BundleItems {
			if isCharacterItem(bundleItem) {
				primaryItem = bundleItem
				break
			}
		}
		if primaryItem.ID == "" {
			for _, bundleItem := range item.BundleItems {
				if calculateIndividualItemPrice(bundleItem) > calculateIndividualItemPrice(primaryItem) {
					primaryItem = bundleItem
				}
			}
		}
	}

	if primaryItem.ID != "" {
		idParts := strings.Split(primaryItem.ID, ":")
		var displayID string
		if len(idParts) > 1 {
			displayID = idParts[1]
		} else {
			displayID = primaryItem.ID
		}
		entry.DisplayAssetPath = displayassets.SetDisplayAsset("DA_Featured_" + displayID)
		entry.NewDisplayAssetPath = displayassets.SetNewDisplayAssetPath(displayID)
	} else if len(item.BundleItems) > 0 {
		idParts := strings.Split(item.BundleItems[0].ID, ":")
		var displayID string
		if len(idParts) > 1 {
			displayID = idParts[1]
		} else {
			displayID = item.BundleItems[0].ID
		}
		entry.DisplayAssetPath = displayassets.SetDisplayAsset("DA_Featured_" + displayID)
		entry.NewDisplayAssetPath = displayassets.SetNewDisplayAssetPath(displayID)
	}

	entry.Meta.DisplayAssetPath = entry.DisplayAssetPath
	entry.Meta.NewDisplayAssetPath = entry.NewDisplayAssetPath
	entry.Meta.SectionId = section
	entry.Meta.TileSize = "DoubleWide"
	entry.Meta.LayoutId = uniqueLayoutID
	entry.Meta.AnalyticOfferGroupId = section

	entry.MetaInfo = []models.MetaInfo{
		{Key: "NewDisplayAssetPath", Value: entry.NewDisplayAssetPath},
		{Key: "SectionId", Value: section},
		{Key: "TileSize", Value: "DoubleWide"},
		{Key: "AnalyticOfferGroupId", Value: section},
	}

	totalRegularPrice := 0
	bundleItems := []models.DynamicBundleItem{}

	for _, bundleItem := range item.BundleItems {
		itemPrice := calculateIndividualItemPrice(bundleItem)
		if strings.HasPrefix(bundleItem.ID, ":") {
			utilities.LogWithTimestamp(color.RedString, fmt.Sprintf("Skipping item with invalid template ID: %s", bundleItem.ID))
			continue
		}
		totalRegularPrice += itemPrice
		discountedPrice := itemPrice
		if FreeShops {
			discountedPrice = 0
		}
		bundleItems = append(bundleItems, models.DynamicBundleItem{
			BCanOwnMultiple:            false,
			RegularPrice:               itemPrice,
			DiscountedPrice:            discountedPrice,
			AlreadyOwnedPriceReduction: itemPrice / 2,
			Item: models.ItemGrant{
				TemplateId: bundleItem.ID,
				Quantity:   1,
			},
		})
		attributes := make(map[string]interface{})
		if itemPrice == 0 || isBackpackItem(bundleItem) {
			attributes["extra_grant"] = true
		}
		itemGrant := models.ItemGrant{TemplateId: bundleItem.ID, Quantity: 1}
		if len(attributes) > 0 {
			itemGrant.Attributes = attributes
		}
		entry.ItemGrants = append(entry.ItemGrants, itemGrant)
	}

	var discountPercent float64
	switch {
	case totalRegularPrice >= 5000:
		discountPercent = 0.30
	case totalRegularPrice >= 3000:
		discountPercent = 0.25
	case totalRegularPrice >= 2000:
		discountPercent = 0.20
	case totalRegularPrice >= 1000:
		discountPercent = 0.15
	default:
		discountPercent = 0.10
	}

	discountAmount := int(float64(totalRegularPrice) * discountPercent)
	if discountAmount < 100 && totalRegularPrice > 800 {
		discountAmount = 100
	}
	if discountAmount < 50 && totalRegularPrice > 300 {
		discountAmount = 50
	}

	floorPrice := totalRegularPrice - discountAmount
	discountedBasePrice := -discountAmount

	if FreeShops {
		floorPrice = 0
		discountedBasePrice = -totalRegularPrice
	} else if floorPrice < 200 && totalRegularPrice > 300 {
		floorPrice = 200
	}

	entry.DynamicBundleInfo = &models.DynamicBundleInfo{
		DiscountedBasePrice: discountedBasePrice,
		RegularBasePrice:    totalRegularPrice,
		FloorPrice:          floorPrice,
		CurrencyType:        "MtxCurrency",
		CurrencySubType:     "",
		DisplayType:         "AmountOff",
		BundleItems:         bundleItems,
	}

	entry.Prices = []models.Price{}
	finalPriceForDevName := -1
	if FreeShops {
		finalPriceForDevName = 0
	}
	entry.DevName = fmt.Sprintf("[VIRTUAL]%s for %d MtxCurrency", generateBundleDevName(item.BundleItems), finalPriceForDevName)
	entry.SortPriority = -1
	return entry, nil
}

func isCharacterItem(item scraper.Item) bool {
	itemType := strings.ToLower(item.Type)
	itemID := strings.ToLower(item.ID)
	return strings.Contains(itemType, "character") || strings.Contains(itemID, "cid") || strings.Contains(itemType, "outfit")
}

func isBackpackItem(item scraper.Item) bool {
	itemType := strings.ToLower(item.Type)
	itemID := strings.ToLower(item.ID)
	return strings.Contains(itemType, "backpack") || strings.Contains(itemID, "bid") || strings.Contains(itemType, "petcarrier")
}

func calculateIndividualItemPrice(item scraper.Item) int {
	if item.Price > 0 {
		utilities.LogWithTimestamp(color.MagentaString, fmt.Sprintf("Using scraped price for %s: %d", item.Name, item.Price))
		return item.Price
	}
	itemType := strings.ToLower(item.Type)
	itemID := strings.ToLower(item.ID)
	itemName := strings.ToLower(item.Name)

	if strings.Contains(itemName, "battle pass") || strings.Contains(itemID, "battlepass") {
		return 950
	}

	switch {
	case isCharacterItem(item):
		if strings.Contains(itemType, "legendary") || strings.Contains(itemName, "legendary") {
			return 2000
		} else if strings.Contains(itemType, "epic") || strings.Contains(itemName, "epic") {
			return 1500
		} else if strings.Contains(itemType, "rare") || strings.Contains(itemName, "rare") {
			return 1200
		}
		return 1500
	case isBackpackItem(item):
		if strings.Contains(itemType, "legendary") || strings.Contains(itemName, "legendary") {
			return 400
		} else if strings.Contains(itemType, "epic") || strings.Contains(itemName, "epic") {
			return 200
		}
		return 0
	case strings.Contains(itemType, "pickaxe") || strings.Contains(itemID, "pickaxe"):
		if strings.Contains(itemType, "legendary") || strings.Contains(itemName, "legendary") {
			return 1200
		} else if strings.Contains(itemType, "epic") || strings.Contains(itemName, "epic") {
			return 800
		}
		return 800
	case strings.Contains(itemType, "glider") || strings.Contains(itemID, "glider"):
		if strings.Contains(itemType, "legendary") || strings.Contains(itemName, "legendary") {
			return 1500
		} else if strings.Contains(itemType, "epic") || strings.Contains(itemName, "epic") {
			return 1200
		}
		return 1200
	case strings.Contains(itemType, "dance") || strings.Contains(itemType, "emote") || strings.Contains(itemID, "eid"):
		if strings.Contains(itemType, "legendary") || strings.Contains(itemName, "legendary") {
			return 800
		} else if strings.Contains(itemType, "epic") || strings.Contains(itemName, "epic") {
			return 500
		} else if strings.Contains(itemType, "rare") || strings.Contains(itemName, "rare") {
			return 300
		}
		return 300
	case strings.Contains(itemType, "wrap") || strings.Contains(itemID, "wrap"):
		if strings.Contains(itemType, "epic") || strings.Contains(itemName, "epic") {
			return 600
		}
		return 500
	case strings.Contains(itemType, "contrail") || strings.Contains(itemID, "trails"):
		if strings.Contains(itemType, "epic") || strings.Contains(itemName, "epic") {
			return 500
		}
		return 400
	case strings.Contains(itemType, "loading") || strings.Contains(itemID, "lsid"):
		return 200
	case strings.Contains(itemType, "music") || strings.Contains(itemID, "music"):
		return 200
	default:
		if strings.Contains(itemName, "skin") || strings.Contains(itemName, "outfit") {
			return 1500
		}
		if strings.Contains(itemName, "back bling") || strings.Contains(itemName, "backpack") {
			return 0
		}
		if strings.Contains(itemName, "harvesting tool") || strings.Contains(itemName, "pickaxe") {
			return 800
		}
		return 500
	}
}

func generateBundleDevName(items []scraper.Item) string {
	var parts []string
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("1 x %s", item.Name))
	}
	return strings.Join(parts, ", ")
}

func CreateShopEntryWithDBItem(item *fortnite.Catalog, section string, s14above bool) (models.CatalogEntry, error) {
	if item == nil || item.TemplateId == "" || item.Storefront == "" {
		return models.CatalogEntry{}, fmt.Errorf("invalid item data")
	}
	if section == "BRDailyStorefront" {
		section = "Daily"
	}
	if section == "BRWeeklyStorefront" {
		section = "Featured"
	}
	var itemData struct {
		Prices []struct {
			BasePrice           int       `json:"basePrice"`
			FinalPrice          int       `json:"finalPrice"`
			RegularPrice        int       `json:"regularPrice"`
			CurrencySubType     string    `json:"currencySubType"`
			CurrencyType        string    `json:"currencyType"`
			DynamicRegularPrice int       `json:"dynamicRegularPrice"`
			SaleExpiration      time.Time `json:"saleExpiration"`
			SaleType            string    `json:"saleType"`
		} `json:"prices"`
	}
	if err := json.Unmarshal([]byte(item.Data), &itemData); err != nil {
		utilities.LogWithTimestamp(color.RedString, fmt.Sprintf("Error unmarshalling item data for %s: %v", item.TemplateId, err))
	}
	scrapedItem := scraper.Item{
		ID:       item.TemplateId,
		Name:     item.Name,
		Price:    itemData.Prices[0].RegularPrice,
		Type:     strings.Split(item.TemplateId, ":")[0],
		Category: item.Category,
		Set: struct {
			Value        string `json:"value"`
			Text         string `json:"text"`
			BackendValue string `json:"backendValue"`
		}{
			Value:        item.Category,
			Text:         item.Category,
			BackendValue: item.Category,
		},
	}
	return CreateShopEntry(scrapedItem, section, s14above)
}

func createBaseShopEntry(item scraper.Item) models.CatalogEntry {
	backendValue := strings.Split(item.ID, ":")[0]
	if backendValue == "" || backendValue != item.Type {
		backendValue = declarations.DetermineItemBackendValue(item)
		item.ID = fmt.Sprintf("%s:%s", backendValue, item.ID)
	}
	FreeShops := utilities.Get[bool]("free")
	finalPrice := item.Price
	saleType := ""
	if FreeShops {
		finalPrice = 0
		saleType = "PercentOff"
	}
	return models.CatalogEntry{
		OfferId:   fmt.Sprintf("v2:/%s", uuid.New().String()),
		OfferType: "StaticPrice",
		DevName:   fmt.Sprintf("[VIRTUAL] 1x %s for %d MtxCurrency", item.ID, finalPrice),
		ItemGrants: []models.ItemGrant{
			{
				TemplateId: item.ID,
				Quantity:   1,
			},
		},
		Requirements: []models.Requirement{
			{
				RequirementType: "DenyOnItemOwnership",
				RequiredId:      item.ID,
				MinQuantity:     1,
			},
		},
		Categories: []string{},
		MetaInfo: []models.MetaInfo{
			{
				Key:   "BannerOverride",
				Value: "",
			},
		},
		Prices: []models.Price{
			{
				CurrencyType:        "MtxCurrency",
				CurrencySubType:     "",
				RegularPrice:        item.Price,
				DynamicRegularPrice: item.Price,
				FinalPrice:          finalPrice,
				SaleType:            saleType,
				SaleExpiration:      "9999-12-31T23:59:59.999Z",
				BasePrice:           item.Price,
			},
		},
		GiftInfo: models.GiftInfo{
			BIsEnabled:              true,
			ForcedGiftBoxTemplateId: "",
			PurchaseRequirements:    []models.Requirement{},
			GiftRecordIds:           []string{},
		},
		Meta: models.Meta{
			NewDisplayAssetPath:  "",
			DisplayAssetPath:     "",
			SectionId:            "Daily",
			TileSize:             "Small",
			LayoutId:             getOrCreateLayoutID("Daily"),
			AnalyticOfferGroupId: "Daily",
		},
		DisplayAssetPath:     "",
		NewDisplayAssetPath:  "",
		Refundable:           true,
		Title:                "",
		Description:          "",
		ShortDescription:     "",
		AppStoreId:           []string{},
		FulfillmentIds:       []interface{}{},
		DailyLimit:           -1,
		WeeklyLimit:          -1,
		MonthlyLimit:         -1,
		SortPriority:         0,
		CatalogGroupPriority: 0,
		FilterWeight:         0,
		BannerOverride:       "",
		MatchFilter:          "",
		AdditionalGrants:     []models.ItemGrant{},
	}
}

func CreateCatalogBattlePassEntry(section models.Storefront, offerId string, devName string, title string, shortDescription string, description string, regularPrice int, finalPrice int, displayAssetPath string, requirements []models.Requirement) models.CatalogEntry {
	FreeShops := utilities.Get[bool]("free")
	actualFinalPrice := finalPrice
	saleType := "PercentOff"
	if FreeShops {
		actualFinalPrice = 0
		saleType = "AmountOff"
	}
	entry := models.CatalogEntry{
		OfferId:   offerId,
		DevName:   devName,
		OfferType: "StaticPrice",
		Prices: []models.Price{
			{
				CurrencyType:        "MtxCurrency",
				CurrencySubType:     "",
				RegularPrice:        regularPrice,
				DynamicRegularPrice: -1,
				FinalPrice:          actualFinalPrice,
				SaleType:            saleType,
				SaleExpiration:      "9999-12-31T23:59:59.999Z",
				BasePrice:           actualFinalPrice,
			},
		},
		Categories:   []string{},
		DailyLimit:   -1,
		WeeklyLimit:  -1,
		MonthlyLimit: -1,
		Refundable:   false,
		AppStoreId: []string{
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
		},
		Requirements:         requirements,
		MetaInfo:             []models.MetaInfo{},
		CatalogGroupPriority: 0,
		SortPriority:         1,
		Description:          description,
		ShortDescription:     shortDescription,
		Title:                title,
		DisplayAssetPath:     displayAssetPath,
		ItemGrants:           []models.ItemGrant{},
	}
	if strings.Contains(strings.ToLower(title), "bundle") {
		entry.SortPriority = 0
		return entry
	}
	return entry
}
