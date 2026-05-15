package fortnite_tournaments

import "github.com/andr1ww/odin"

type Scores struct {
	odin.Bucket `bucket:"Fortnite_Tournament_Scores" database:"xenon_comp"`
	AccountId   string `json:"account_id"`
	Type        string `json:"type"`
	Value       int    `json:"value"`
	Season      int    `json:"season"`
}
