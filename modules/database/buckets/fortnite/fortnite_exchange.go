package fortnite

import "github.com/andr1ww/odin"

type Exchange struct {
	odin.Bucket `bucket:"ExchangeCodes" database:"xenon"`
	Code        string `json:"code"`
	AccountID   string `json:"account_id"`
	Created     string `json:"created"`
}
