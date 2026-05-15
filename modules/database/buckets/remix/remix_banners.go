package remix

import "github.com/andr1ww/odin"

type Banner struct {
	odin.Bucket `bucket:"Remix_Banners" database:"xenon"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Order       int    `json:"order"`
}
