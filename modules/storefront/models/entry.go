package models

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
	BannerOverride       string             `json:"bannerOverride"`
	DisplayAssetPath     string             `json:"displayAssetPath"`
	NewDisplayAssetPath  string             `json:"NewDisplayAssetPath"`
	Refundable           bool               `json:"refundable"`
	Title                string             `json:"title"`
	Description          string             `json:"description"`
	ShortDescription     string             `json:"shortDescription"`
	AppStoreId           []string           `json:"appStoreId"`
	FulfillmentIds       []interface{}      `json:"fulfillmentIds"`
	DailyLimit           int                `json:"dailyLimit"`
	WeeklyLimit          int                `json:"weeklyLimit"`
	DynamicBundleInfo    *DynamicBundleInfo `json:"dynamicBundleInfo,omitempty"`
	MonthlyLimit         int                `json:"monthlyLimit"`
	SortPriority         int                `json:"sortPriority"`
	CatalogGroupPriority int                `json:"catalogGroupPriority"`
	FilterWeight         int                `json:"filterWeight"`
	MatchFilter          string             `json:"matchFilter"`
	AdditionalGrants     []ItemGrant        `json:"additionalGrants"`
}
