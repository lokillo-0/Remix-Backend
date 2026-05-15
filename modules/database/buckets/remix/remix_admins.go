package remix

import "github.com/andr1ww/odin"

type Admins struct {
	odin.Bucket `bucket:"Remix_Admins" database:"xenon"`
	IPAddress   string `json:"ip_address"`
	AccountID   string `json:"account_id"`
}
