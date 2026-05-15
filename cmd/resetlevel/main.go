package main

import (
	"fmt"
	"log"

	"github.com/andr1ww/odin"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
)

func main() {
	if err := odin.Connect("xenon", "databases/xenon.db"); err != nil {
		log.Fatalf("xenon db: %v", err)
	}
	if err := odin.Connect("xenon_profiles", "databases/xenon_profiles.db"); err != nil {
		log.Fatalf("xenon_profiles db: %v", err)
	}

	accountID := "e9b6c36ae5284e3b8609f57bf6bfe624"
	season := 32

	// Reset season level
	seasonKey := fmt.Sprintf("%s:%d", accountID, season)
	var s accounts.Season
	odin.Find("Accounts_Seasons", seasonKey, &s)
	s.Bucket.ID = seasonKey
	s.Level = 1
	s.BookLevel = 1
	s.BookXp = 0
	s.Xp = 0
	s.Bucket.Save(s)
	fmt.Printf("Season reset: level=1 bookLevel=1 xp=0\n")

	// Reset battlestars in athena profile
	athenaKey := accountID + ":athena"
	var athena accounts.Profile
	if err := odin.Find("Accounts_Profiles", athenaKey, &athena); err != nil {
		log.Fatalf("athena profile not found: %v", err)
	}
	if athena.Stats == nil {
		athena.Stats = make(map[string]interface{})
	}
	athena.Stats["battlestars"] = float64(0)
	athena.Stats["battlestars_season_total"] = float64(0)
	athena.Bucket.Save(athena)
	fmt.Println("Battlestars reset to 0")
}
