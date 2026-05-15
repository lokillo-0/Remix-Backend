package fortnite

import (
	"time"

	"github.com/andr1ww/odin"
)

type StackRank struct {
	StackRankValue int    `json:"stackRankValue"`
	Type           string `json:"_type"`
	Context        string `json:"context"`
	StartDate      string `json:"startDate"`
}

type OfferGroup struct {
	BUseWidePreview bool        `json:"bUseWidePreview"`
	Type            string      `json:"_type"`
	OfferGroupID    string      `json:"offerGroupId"`
	StackRanks      []StackRank `json:"stackRanks"`
}

type Background struct {
	Type string `json:"_type"`
}

type SectionMetadata struct {
	OfferGroups          []OfferGroup `json:"offerGroups"`
	Background           Background   `json:"background"`
	Type                 string       `json:"_type"`
	ShowIneligibleOffers string       `json:"showIneligibleOffers"`
	StackRanks           []StackRank  `json:"stackRanks"`
}

type MpItemShop struct {
	odin.Bucket `bucket:"MPItemShop" database:"xenon"`
	Date        time.Time       `json:"date"`
	Section     string          `json:"section"`
	Metadata    SectionMetadata `json:"metadata"`
	DisplayName string          `json:"displayName"`
	Type        string          `json:"_type"`
	SectionID   string          `json:"sectionID"`
}
