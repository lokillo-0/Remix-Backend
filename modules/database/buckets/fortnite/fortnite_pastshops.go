package fortnite

import (
	"encoding/json"
	"time"

	"github.com/andr1ww/odin"
	"github.com/google/uuid"
)

type PastShops struct {
	odin.Bucket `bucket:"PastShops" database:"xenon"`
	Date        time.Time `gorm:"column:date"`
}

func GetLastProcessedShop() time.Time {
	db, err := odin.Get()
	if err != nil {
		return time.Time{}
	}

	var latestShop *PastShops
	var latestDate time.Time

	err = db.ForEach("PastShops", func(k, v []byte) error {
		var shop PastShops
		if err := json.Unmarshal(v, &shop); err != nil {
			return nil
		}

		if latestShop == nil || shop.Date.After(latestDate) {
			latestShop = &shop
			latestDate = shop.Date
		}

		return nil
	})

	if err != nil || latestShop == nil {
		return time.Time{}
	}

	return latestDate
}

func SaveProcessedShop(date time.Time) {
	odin.Create(&PastShops{Bucket: odin.Bucket{ID: uuid.New().String()}, Date: date})
}

func IsShopProcessed(date time.Time) bool {
	conditions := map[string]interface{}{
		"date": date.Format("2006-01-02"),
	}
	shops, err := odin.FindWhere("PastShops", conditions, func() interface{} { return &PastShops{} })
	if err != nil || len(shops) == 0 {
		return false
	}
	return true
}
