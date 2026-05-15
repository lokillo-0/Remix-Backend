package models

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
