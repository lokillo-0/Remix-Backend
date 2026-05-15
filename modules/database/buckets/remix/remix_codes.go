package remix

import "github.com/andr1ww/odin"

type Code struct {
	odin.Bucket `bucket:"Remix_Codes" database:"xenon_redeem"`
	Code        string   `json:"code"`
	Package     string   `json:"package"`
	Rewards     []string `json:"rewards"`
	Created     int64    `json:"created"`
	Expires     int64    `json:"expires"`
}
