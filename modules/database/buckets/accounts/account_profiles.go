package accounts

import "github.com/andr1ww/odin"

type Profile struct {
	odin.Bucket `bucket:"Accounts_Profiles" database:"xenon_profiles"`
	AccountID   string                 `json:"account_id"`
	ProfileID   string                 `json:"profile_id"`
	Revision    int                    `json:"revision"`
	Items       map[string]interface{} `json:"items"`
	Stats       map[string]interface{} `json:"stats"`
}
