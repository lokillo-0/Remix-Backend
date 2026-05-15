package models

type Price struct {
	CurrencyType        string `json:"currencyType"`
	CurrencySubType     string `json:"currencySubType"`
	RegularPrice        int    `json:"regularPrice"`
	DynamicRegularPrice int    `json:"dynamicRegularPrice"`
	FinalPrice          int    `json:"finalPrice"`
	SaleExpiration      string `json:"saleExpiration"`
	SaleType            string `json:"saleType"`
	BasePrice           int    `json:"basePrice"`
}
