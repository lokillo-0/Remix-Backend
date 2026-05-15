package remix

import "github.com/andr1ww/odin"

type Posts struct {
	odin.Bucket `bucket:"Remix_Posts" database:"xenon"`
	Title       string   `json:"title"`
	Date        string   `json:"date"`
	Image       []string `json:"image"`
}
