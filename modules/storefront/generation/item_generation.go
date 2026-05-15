package generation

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/andr1ww/odin"
	"github.com/fatih/color"
	"github.com/remixfn/xenon/modules/database/buckets/fortnite"
	"github.com/remixfn/xenon/modules/storefront/declarations"
	"github.com/remixfn/xenon/modules/storefront/models"
	"github.com/remixfn/xenon/modules/storefront/utils"
	"github.com/remixfn/xenon/utilities"
)

var dbMutex sync.Mutex

func generateDeterministicID(storefront, templateId, name string) string {
	data := fmt.Sprintf("%s:%s:%s", storefront, templateId, name)
	hasher := md5.New()
	hasher.Write([]byte(data))
	return hex.EncodeToString(hasher.Sum(nil))
}

func generateDeterministicOfferID(templateId, name string) string {
	data := fmt.Sprintf("%s:%s", templateId, name)
	hasher := md5.New()
	hasher.Write([]byte(data))
	hash := hex.EncodeToString(hasher.Sum(nil))
	return fmt.Sprintf("v2:/%s", hash)
}

func FillDailyStorefront(dailySection *models.Storefront) error {
	utils.Mu.Lock()
	defer utils.Mu.Unlock()

	dailyItems, exists := utils.StorefrontData["Daily"]
	if !exists || len(dailyItems) == 0 {
		return fmt.Errorf("no daily items found")
	}

	utilities.LogWithTimestamp(color.GreenString, "Generating up to 6 daily items for the daily storefront")

	seenItems := make(map[string]bool)
	processedCount := 0

	for _, item := range dailyItems {
		if processedCount >= 6 {
			break
		}

		itemKey := fmt.Sprintf("%s:%s", item.Type, item.Name)
		if seenItems[itemKey] {
			utilities.LogWithTimestamp(color.YellowString, fmt.Sprintf("Skipping duplicate in current run: %s", item.Name))
			continue
		}

		backendValue := declarations.DetermineItemBackendValue(item)
		templateId := fmt.Sprintf("%s:%s", backendValue, item.ID)

		currentVersion := utilities.GetConfig().CURRENT_VERSION
		entry, err := CreateShopEntry(item, "Daily", currentVersion >= 14)
		if err != nil {
			utilities.LogWithTimestamp(color.RedString, fmt.Sprintf("Error creating shop entry: %v", err))
			continue
		}

		deterministicID := generateDeterministicID("BRDailyStorefront", templateId, item.Name)
		deterministicOfferID := generateDeterministicOfferID(templateId, item.Name)

		entry.OfferId = deterministicOfferID

		catalogEntry := &fortnite.Catalog{
			Bucket:     odin.Bucket{ID: deterministicID},
			Created:    time.Now().UTC().Format("2006-01-02T15:04:05.999Z"),
			OfferId:    entry.OfferId,
			Storefront: "BRDailyStorefront",
			Name:       item.Name,
			Category:   "",
			Data:       string(utilities.MustMarshal(entry)),
			TemplateId: templateId,
		}

		if err := upsertCatalogEntry(catalogEntry); err != nil {
			utilities.LogWithTimestamp(color.RedString, fmt.Sprintf("Error upserting catalog entry: %v", err))
			continue
		}

		dailySection.CatalogEntries = append(dailySection.CatalogEntries, entry)
		seenItems[itemKey] = true
		processedCount++
	}

	utilities.LogWithTimestamp(color.GreenString, fmt.Sprintf("Generated %d daily items for the storefront", len(dailySection.CatalogEntries)))
	return nil
}

var nonBRSections = map[string]bool{
	"Gear For Festival": true,
	"Festival":          true,
	"LEGO":              true,
	"LEGO Fortnite":     true,
	"Rocket Racing":     true,
	"Racing":            true,
	"Ballistic":         true,
}

func FillWeeklyStorefront(weeklySection *models.Storefront) error {
	utils.Mu.Lock()
	defer utils.Mu.Unlock()
	utilities.LogWithTimestamp(color.GreenString, "Generating items for weekly storefront")

	seenItems := make(map[string]bool)

	for section, items := range utils.StorefrontData {
		if section == "Daily" {
			continue
		}
		if len(items) == 0 {
			continue
		}
		if nonBRSections[section] {
			utilities.LogWithTimestamp(color.YellowString, fmt.Sprintf("Skipping non-BR section: %s", section))
			continue
		}
		utilities.LogWithTimestamp(color.GreenString, fmt.Sprintf("Generating %d items for section %s", len(items), section))

		for _, item := range items {
			itemKey := fmt.Sprintf("%s:%s:%s", section, item.Type, item.Name)
			if seenItems[itemKey] {
				utilities.LogWithTimestamp(color.YellowString, fmt.Sprintf("Skipping duplicate in current run: %s", item.Name))
				continue
			}

			currentVersion := utilities.GetConfig().CURRENT_VERSION

			var entry models.CatalogEntry
			var err error
			var templateId string
			var deterministicID string
			var deterministicOfferID string

			if item.IsBundle && len(item.BundleItems) > 0 {
				templateId = fmt.Sprintf("Bundle:%s", strings.ReplaceAll(item.Name, " ", "_"))

				entry, err = CreateBundleEntry(item, section, currentVersion >= 14)
				if err == nil {
					if validateErr := ValidateBundleEntry(&entry); validateErr != nil {
						utilities.LogWithTimestamp(color.RedString, fmt.Sprintf("Bundle validation failed for %s: %v", item.Name, validateErr))
						continue
					}
				}

				deterministicID = generateDeterministicID("BRWeeklyStorefront", templateId, item.Name)
				deterministicOfferID = generateDeterministicOfferID(templateId, item.Name)
			} else {
				backendValue := declarations.DetermineItemBackendValue(item)
				templateId = fmt.Sprintf("%s:%s", backendValue, item.ID)

				entry, err = CreateShopEntry(item, section, currentVersion >= 14)

				deterministicID = generateDeterministicID("BRWeeklyStorefront", templateId, item.Name)
				deterministicOfferID = generateDeterministicOfferID(templateId, item.Name)
			}

			if err != nil {
				utilities.LogWithTimestamp(color.RedString, fmt.Sprintf("Skipping item due to error: %s - Error: %v", item.ID, err))
				continue
			}

			entry.OfferId = deterministicOfferID

			var category string
			if entry.OfferType == "DynamicBundle" {
				if len(entry.ItemGrants) > 0 {
					templateParts := strings.Split(entry.ItemGrants[0].TemplateId, ":")
					if len(templateParts) > 0 {
					}
				}
				if len(entry.Categories) > 0 {
					category = entry.Categories[0]
				}
			} else {
				if len(entry.Categories) > 0 {
					category = entry.Categories[0]
				}
			}

			catalogEntry := &fortnite.Catalog{
				Bucket:     odin.Bucket{ID: deterministicID},
				Created:    time.Now().UTC().Format("2006-01-02T15:04:05.999Z"),
				OfferId:    entry.OfferId,
				Storefront: "BRWeeklyStorefront",
				Name:       item.Name,
				Category:   category,
				Data:       string(utilities.MustMarshal(entry)),
				TemplateId: templateId,
			}

			if err := upsertCatalogEntry(catalogEntry); err != nil {
				utilities.LogWithTimestamp(color.RedString, fmt.Sprintf("Error upserting catalog entry: %v", err))
				continue
			}

			weeklySection.CatalogEntries = append(weeklySection.CatalogEntries, entry)
			seenItems[itemKey] = true
		}
	}

	return nil
}

func upsertCatalogEntry(catalogEntry *fortnite.Catalog) error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	return odin.Create(catalogEntry)
}

func FillAllSectionsInDatabase() error {
	existingSections := getExistingSections()

	for section, items := range utils.StorefrontData {
		if len(items) == 0 {
			continue
		}
		if nonBRSections[section] {
			continue
		}

		utilities.LogWithTimestamp(color.GreenString, fmt.Sprintf("Generating section %s", section))

		sectionID := generateDeterministicID("ShopSections", section, section)

		shopSection := &fortnite.ShopSections{
			Bucket:  odin.Bucket{ID: sectionID},
			Date:    time.Now().UTC(),
			Section: section,
		}

		if err := odin.Create(shopSection); err != nil {
			utilities.LogWithTimestamp(color.YellowString, fmt.Sprintf("Section %s may already exist, continuing: %v", section, err))
		}

		stackRank := getOrCreateLayoutIDInt(section)

		mpID := generateDeterministicID("MpItemShop", section, section)

		mpItemShop := &fortnite.MpItemShop{
			Bucket:  odin.Bucket{ID: mpID},
			Date:    time.Now().UTC(),
			Section: section,
			Metadata: fortnite.SectionMetadata{
				Type: "MP Item Shop - Section Metadata",
				OfferGroups: []fortnite.OfferGroup{
					{
						BUseWidePreview: false,
						Type:            "MP Item Shop - Row",
						OfferGroupID:    fmt.Sprintf("%d", stackRank),
						StackRanks: []fortnite.StackRank{
							{
								StackRankValue: stackRank,
								Type:           "MP Item Shop - Stack Rank",
								Context:        "battleRoyale",
								StartDate:      "2023-01-01T00:00:00.000Z",
							},
						},
					},
					{
						BUseWidePreview: false,
						Type:            "MP Item Shop - Row",
						OfferGroupID:    fmt.Sprintf("%d", stackRank-1),
						StackRanks: []fortnite.StackRank{
							{
								StackRankValue: stackRank - 1,
								Type:           "MP Item Shop - Stack Rank",
								Context:        "battleRoyale",
								StartDate:      "2023-01-01T00:00:00.000Z",
							},
						},
					},
				},
				Background: fortnite.Background{
					Type: "MP Item Shop - Background",
				},
				ShowIneligibleOffers: "Always",
				StackRanks: []fortnite.StackRank{
					{
						StackRankValue: stackRank,
						Type:           "MP Item Shop - Stack Rank",
						Context:        "battleRoyale",
						StartDate:      "2023-01-01T00:00:00.000Z",
					},
				},
			},
			DisplayName: section,
			Type:        "MP Item Shop - Section",
			SectionID:   section,
		}

		if err := odin.Create(mpItemShop); err != nil {
			utilities.LogWithTimestamp(color.YellowString, fmt.Sprintf("MP Item Shop section %s may already exist, continuing: %v", section, err))
		}

		existingSections[section] = true
	}

	return nil
}

func getExistingSections() map[string]bool {
	existing := make(map[string]bool)

	db, err := odin.Get()
	if err != nil {
		utilities.LogWithTimestamp(color.RedString, "Failed to get database for section check: "+err.Error())
		return existing
	}

	err = db.ForEach("ShopSections", func(k, v []byte) error {
		var section fortnite.ShopSections
		if err := json.Unmarshal(v, &section); err == nil {
			existing[section.Section] = true
		}
		return nil
	})

	if err != nil {
		utilities.LogWithTimestamp(color.RedString, "Error checking existing sections: "+err.Error())
	}

	return existing
}
