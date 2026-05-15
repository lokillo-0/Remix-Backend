package models

type MetaInfo struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Meta struct {
	NewDisplayAssetPath  string `json:"NewDisplayAssetPath"`
	LayoutId             string `json:"LayoutId"`
	TileSize             string `json:"TileSize"`
	AnalyticOfferGroupId string `json:"AnalyticOfferGroupId"`
	SectionId            string `json:"SectionId"`
	TemplateId           string `json:"templateId"`
	DisplayAssetPath     string `json:"displayAssetPath"`
}
