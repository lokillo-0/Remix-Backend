package fortnite

import (
	"time"

	"github.com/andr1ww/odin"
)

type ShopSections struct {
	odin.Bucket `bucket:"ShopSections" database:"xenon"`
	Date        time.Time `gorm:"column:date"`
	Section     string    `gorm:"column:section"`
}
