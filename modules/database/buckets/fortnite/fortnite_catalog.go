package fortnite

import "github.com/andr1ww/odin"

type Catalog struct {
	odin.Bucket `bucket:"Catalog" database:"xenon"`
	Created     string `json:"created"`
	Storefront  string `json:"storefront"`
	OfferId     string `json:"offerId"`
	Name        string `json:"name"`
	TemplateId  string `json:"templateId"`
	Data        string `json:"data"`
	Category    string `json:"category"`
}
