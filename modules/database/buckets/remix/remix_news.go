package remix

import "github.com/andr1ww/odin"

type NewsCard struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Date            string `json:"date"`
	BackgroundImage string `json:"backgroundImage"`
	ShowProgress    bool   `json:"showProgress"`
	ProgressValue   int    `json:"progressValue"`
}

type News struct {
	odin.Bucket     `bucket:"Remix_News" database:"xenon"`
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Subtitle        string     `json:"subtitle"`
	Description     string     `json:"description"`
	BackgroundImage string     `json:"backgroundImage"`
	LogoImage       string     `json:"logoImage"`
	ShowProgress    bool       `json:"showProgress"`
	NewsCards       []NewsCard `json:"newsCards"`
}
