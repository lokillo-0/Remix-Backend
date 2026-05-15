package fortnite

import "github.com/andr1ww/odin"

type Hotfixes struct {
	odin.Bucket `bucket:"Hotfixes" database:"xenon"`
	Name        string `json:"name"`
	Value       string `json:"value"`
	Enabled     bool   `json:"enabled"`
}
