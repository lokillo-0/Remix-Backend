package models

type GiftInfo struct {
	BIsEnabled              bool          `json:"bIsEnabled"`
	ForcedGiftBoxTemplateId string        `json:"forcedGiftBoxTemplateId"`
	PurchaseRequirements    []Requirement `json:"purchaseRequirements"`
	GiftRecordIds           []string      `json:"giftRecordIds"`
}
