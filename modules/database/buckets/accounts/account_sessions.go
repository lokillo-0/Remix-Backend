package accounts

import "github.com/andr1ww/odin"

type Session struct {
	odin.Bucket `bucket:"Accounts_Sessions" database:"xenon"`
	Token       string `json:"token"`
	Type        string `json:"type"`
	AccountID   string `json:"accountId"`
}
