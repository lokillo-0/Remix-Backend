package models

type Storefront struct {
	Name           string         `json:"name"`
	CatalogEntries []CatalogEntry `json:"catalogEntries"`
}
