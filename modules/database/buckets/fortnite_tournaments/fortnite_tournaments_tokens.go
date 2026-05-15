package fortnite_tournaments

import "github.com/andr1ww/odin"

type Tokens struct {
	odin.Bucket `bucket:"Fortnite_Tournament_Tokens" database:"xenon_comp"`
	AccountId   string `json:"account_id"`
	Token       string `json:"token"`
	Season      int    `json:"season"`
}
