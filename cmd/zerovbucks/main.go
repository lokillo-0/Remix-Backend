package main

import (
	"encoding/json"
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

	username := "jeremyud"

	// Find account by display name
	found, err := odin.FindWhere("Accounts", map[string]interface{}{
		"displayName": username,
	}, func() interface{} { return &accounts.Account{} })
	if err != nil || len(found) == 0 {
		log.Fatalf("Account not found: %v", err)
	}

	account := found[0].(*accounts.Account)
	fmt.Printf("Found account: %s (%s)\n", account.DisplayName, account.ID)

	profileKey := account.ID + ":common_core"
	var profile accounts.Profile
	if err := odin.Find("Accounts_Profiles", profileKey, &profile); err != nil {
		log.Fatalf("Profile not found: %v", err)
	}

	if profile.Items == nil {
		profile.Items = make(map[string]interface{})
	}

	// Check current balance
	if curr, ok := profile.Items["Currency:MtxPurchased"]; ok {
		b, _ := json.Marshal(curr)
		fmt.Printf("Current: %s\n", b)
	}

	profile.Items["Currency:MtxPurchased"] = map[string]interface{}{
		"templateId": "Currency:MtxPurchased",
		"attributes": map[string]interface{}{
			"platform": "EpicPC",
			"level":    1,
		},
		"quantity": 0,
	}

	profile.Bucket.Save(profile)
	fmt.Println("V-Bucks set to 0.")
}
