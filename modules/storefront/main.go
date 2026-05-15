package storefront

import (
	"encoding/json"
	"time"

	"github.com/andr1ww/odin"
	"github.com/fatih/color"
	"github.com/remixfn/xenon/modules/database/buckets/fortnite"
	"github.com/remixfn/xenon/modules/storefront/custom"
	"github.com/remixfn/xenon/modules/storefront/declarations"
	"github.com/remixfn/xenon/modules/storefront/generation"
	"github.com/remixfn/xenon/modules/storefront/utils"
	"github.com/remixfn/xenon/utilities"
)

func Init() {
	for {
		if _, err := odin.Get(); err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	processStorefront()
	go func() {
		for {
			now := time.Now().UTC()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
			time.Sleep(time.Until(next))
			processStorefront()
		}
	}()
}

func ForceRegenerate() {
	db, err := odin.Get()
	if err == nil {
		db.Clear("PastShops")
	}
	processStorefront()
}

func processStorefront() {
	utilities.LogWithTimestamp(color.CyanString, "Loading custom shop from assets/shop.json")

	shopData, err := custom.LoadShop("assets/shop.json")
	if err != nil {
		utilities.LogWithTimestamp(color.RedString, "Failed to load shop.json: "+err.Error())
		return
	}

	db, err := odin.Get()
	if err != nil {
		utilities.LogWithTimestamp(color.RedString, "Failed to get database: "+err.Error())
		return
	}

	deleteCatalogByStorefront := func(sf string) {
		var keys []string
		db.ForEach("Catalog", func(k, v []byte) error {
			var c fortnite.Catalog
			if json.Unmarshal(v, &c) == nil && c.Storefront == sf {
				keys = append(keys, string(k))
			}
			return nil
		})
		for _, key := range keys {
			db.Delete("Catalog", key)
		}
	}

	deleteCatalogByStorefront("BRWeeklyStorefront")
	deleteCatalogByStorefront("BRDailyStorefront")
	db.Clear("ShopSections")
	db.Clear("MPItemShop")

	utils.Mu.Lock()
	utils.StorefrontData = shopData
	utils.Mu.Unlock()

	dailySection := declarations.CreateSection("BRDailyStorefront")
	weeklySection := declarations.CreateSection("BRWeeklyStorefront")
	generation.FillDailyStorefront(&dailySection)
	generation.FillWeeklyStorefront(&weeklySection)
	generation.FillAllSectionsInDatabase()

	utilities.LogWithTimestamp(color.GreenString, "Custom shop loaded successfully")
}
