package accounts

import "github.com/andr1ww/odin"

type Friends struct {
	odin.Bucket `bucket:"Accounts_Friends" database:"xenon"`
	Created     string `json:"created"`
	AccountId   string `json:"accountId"`
	FriendId    string `json:"friendId"`
	Alias       string `json:"alias"`
	Direction   string `json:"direction"`
	Status      string `json:"status"`
}
