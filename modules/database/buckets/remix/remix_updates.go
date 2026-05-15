package remix

import "github.com/andr1ww/odin"

type Update struct {
	odin.Bucket `bucket:"Remix_Updates" database:"xenon"`
	Version     string `json:"version"`
	PublishDate string `json:"pub_date"`
	Url         string `json:"url"`
	Signature   string `json:"signature"`
	Notes       string `json:"notes"`
}
