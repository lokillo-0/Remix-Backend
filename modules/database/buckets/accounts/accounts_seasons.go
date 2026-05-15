package accounts

import "github.com/andr1ww/odin"

type Season struct {
	odin.Bucket  `bucket:"Accounts_Seasons" database:"xenon"`
	BookLevel    int  `json:"bookLevel"`
	BookXp       int  `json:"bookXp"`
	Level        int  `json:"level"`
	Xp           int  `json:"xp"`
	AllXpGained  int  `json:"allXpGained"`
	PurchasedVip bool `json:"purchasedVip"`
	Wins         int  `json:"wins"`
}
