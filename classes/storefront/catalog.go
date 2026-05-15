package storefront

type Catalog struct {
	RefreshIntervalHrs int                 `json:"refreshIntervalHrs"`
	DailyPurchaseHrs   int                 `json:"dailyPurchaseHrs"`
	Expiration         string              `json:"expiration"`
	Storefronts        []DefaultStorefront `json:"storefronts"`
}

type CatalogEntry struct {
	OfferId              string             `json:"offerId"`
	OfferType            string             `json:"offerType"`
	DevName              string             `json:"devName"`
	ItemGrants           []ItemGrant        `json:"itemGrants"`
	Requirements         []Requirement      `json:"requirements"`
	Categories           []string           `json:"categories"`
	MetaInfo             []MetaInfo         `json:"metaInfo"`
	Meta                 Meta               `json:"meta"`
	GiftInfo             GiftInfo           `json:"giftInfo"`
	Prices               []Price            `json:"prices"`
	DynamicBundleInfo    *DynamicBundleInfo `json:"dynamicBundleInfo,omitempty"`
	DisplayAssetPath     string             `json:"displayAssetPath"`
	NewDisplayAssetPath  string             `json:"newDisplayAssetPath"`
	Refundable           bool               `json:"refundable"`
	Title                string             `json:"title"`
	Description          string             `json:"description"`
	ShortDescription     string             `json:"shortDescription"`
	AppStoreId           []string           `json:"appStoreId"`
	FulfillmentIds       []interface{}      `json:"fulfillmentIds"`
	DailyLimit           int                `json:"dailyLimit"`
	WeeklyLimit          int                `json:"weeklyLimit"`
	MonthlyLimit         int                `json:"monthlyLimit"`
	SortPriority         int                `json:"sortPriority"`
	CatalogGroupPriority int                `json:"catalogGroupPriority"`
	FilterWeight         float64            `json:"filterWeight"`
	BannerOverride       string             `json:"bannerOverride"`
	MatchFilter          string             `json:"matchFilter"`
	AdditionalGrants     []ItemGrant        `json:"additionalGrants"`
}

type DynamicBundleInfo struct {
	DiscountedBasePrice int                 `json:"discountedBasePrice"`
	RegularBasePrice    int                 `json:"regularBasePrice"`
	FloorPrice          int                 `json:"floorPrice"`
	CurrencyType        string              `json:"currencyType"`
	CurrencySubType     string              `json:"currencySubType"`
	DisplayType         string              `json:"displayType"`
	BundleItems         []DynamicBundleItem `json:"bundleItems"`
}

type DynamicBundleItem struct {
	BCanOwnMultiple            bool      `json:"bCanOwnMultiple"`
	RegularPrice               int       `json:"regularPrice"`
	DiscountedPrice            int       `json:"discountedPrice"`
	AlreadyOwnedPriceReduction int       `json:"alreadyOwnedPriceReduction"`
	Item                       ItemGrant `json:"item"`
}

type DefaultStorefront struct {
	Name           string         `json:"name"`
	CatalogEntries []CatalogEntry `json:"catalogEntries"`
}

type GiftInfo struct {
	BIsEnabled              bool          `json:"bIsEnabled"`
	ForcedGiftBoxTemplateId string        `json:"forcedGiftBoxTemplateId"`
	PurchaseRequirements    []Requirement `json:"purchaseRequirements"`
	GiftRecordIds           []string      `json:"giftRecordIds"`
}

type ItemGrant struct {
	TemplateId string                 `json:"templateId"`
	Quantity   int                    `json:"quantity"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

type Meta struct {
	NewDisplayAssetPath  string `json:"NewDisplayAssetPath"`
	DisplayAssetPath     string `json:"displayAssetPath"`
	SectionId            string `json:"SectionId"`
	TileSize             string `json:"TileSize"`
	LayoutId             string `json:"LayoutId"`
	AnalyticOfferGroupId string `json:"AnalyticOfferGroupId"`
}

type MetaInfo struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Price struct {
	CurrencyType        string `json:"currencyType"`
	CurrencySubType     string `json:"currencySubType"`
	RegularPrice        int    `json:"regularPrice"`
	FinalPrice          int    `json:"finalPrice"`
	BasePrice           int    `json:"basePrice"`
	SaleExpiration      string `json:"saleExpiration"`
	DynamicRegularPrice int    `json:"dynamicRegularPrice"`
}

type Requirement struct {
	RequirementType string `json:"requirementType"`
	RequiredId      string `json:"requiredId"`
	MinQuantity     int    `json:"minQuantity"`
}
