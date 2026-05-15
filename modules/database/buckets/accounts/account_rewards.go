package accounts

import "github.com/andr1ww/odin"

type AccountReward struct {
	odin.Bucket `bucket:"Accounts_Rewards" database:"xenon"`
	AccountID   string   `json:"account_id"`
	CodeID      string   `json:"code_id"`
	Rewards     []string `json:"rewards"`
	Redeemed    bool     `json:"redeemed"`
}
