package declarations

import (
	"strings"

	"github.com/remixfn/xenon/modules/storefront/models"
	"github.com/remixfn/xenon/modules/storefront/scraper"
)

func CreateSection(section string) models.Storefront {
	return models.Storefront{
		Name:           section,
		CatalogEntries: []models.CatalogEntry{},
	}
}

func DetermineItemBackendValue(item scraper.Item) string {
	typeToBackend := map[string]string{
		"emote":                      "AthenaDance",
		"emoji":                      "AthenaDance",
		"toy":                        "AthenaDance",
		"loading":                    "AthenaLoadingScreen",
		"trails":                     "AthenaSkyDiveContrail",
		"glider":                     "AthenaGlider",
		"backpack":                   "AthenaBackpack",
		"petcarrier":                 "AthenaBackpack",
		"musicpack":                  "AthenaMusicPack",
		"pickaxe":                    "AthenaPickaxe",
		"wrap":                       "AthenaItemWrap",
		"outfit":                     "AthenaCharacter",
		"EID":                        "AthenaDance",
		"Emoji":                      "AthenaDance",
		"SPID":                       "AthenaDance",
		"TOY":                        "AthenaDance",
		"LSID":                       "AthenaLoadingScreen",
		"Trails":                     "AthenaSkyDiveContrail",
		"MtxGiveaway":                "Currency",
		"Glider":                     "AthenaGlider",
		"BID":                        "AthenaBackpack",
		"Backpack":                   "AthenaBackpack",
		"PetCarrier":                 "AthenaBackpack",
		"MusicPack":                  "AthenaMusicPack",
		"Pickaxe":                    "AthenaPickaxe",
		"AthenaSeasonXpBoost":        "Token",
		"AthenaSeasonFriendXpBoost":  "Token",
		"CID":                        "AthenaCharacter",
		"Character":                  "AthenaCharacter",
		"AthenaSeasonMergedXpBoosts": "Token",
		"Wrap":                       "AthenaItemWrap",
		"AthenaSeasonalXP":           "Token",
		"AthenaNextSeasonTierBoost":  "Token",
		"AthenaNextSeasonXPBoost":    "Token",
		"AthenaBattlePassTier":       "Token",
		"VTID":                       "CosmeticVariantToken",
	}

	lowered := strings.ToLower(item.Type)
	if backend, ok := typeToBackend[lowered]; ok {
		return backend
	}

	for prefix, backendValue := range typeToBackend {
		if strings.Contains(item.ID, prefix) {
			return backendValue
		}
	}

	return ""
}

func DetermineItemBackendValueWithStr(itemId string) string {
	typeToBackend := map[string]string{
		"emote":                      "AthenaDance",
		"eid":                        "AthenaDance",
		"emoji":                      "AthenaDance",
		"toy":                        "AthenaDance",
		"loading":                    "AthenaLoadingScreen",
		"trails":                     "AthenaSkyDiveContrail",
		"glider":                     "AthenaGlider",
		"backpack":                   "AthenaBackpack",
		"petcarrier":                 "AthenaBackpack",
		"musicpack":                  "AthenaMusicPack",
		"pickaxe":                    "AthenaPickaxe",
		"wrap":                       "AthenaItemWrap",
		"outfit":                     "AthenaCharacter",
		"EID":                        "AthenaDance",
		"Emoji":                      "AthenaDance",
		"SPID":                       "AthenaDance",
		"TOY":                        "AthenaDance",
		"LSID":                       "AthenaLoadingScreen",
		"Trails":                     "AthenaSkyDiveContrail",
		"MtxGiveaway":                "Currency",
		"Glider":                     "AthenaGlider",
		"BID":                        "AthenaBackpack",
		"Backpack":                   "AthenaBackpack",
		"PetCarrier":                 "AthenaBackpack",
		"MusicPack":                  "AthenaMusicPack",
		"Pickaxe":                    "AthenaPickaxe",
		"AthenaSeasonXpBoost":        "Token",
		"AthenaSeasonFriendXpBoost":  "Token",
		"CID":                        "AthenaCharacter",
		"character":                  "AthenaCharacter",
		"athenaSeasonMergedXpBoosts": "Token",
		"AthenaSeasonalXP":           "Token",
		"AthenaNextSeasonTierBoost":  "Token",
		"AthenaNextSeasonXPBoost":    "Token",
		"AthenaBattlePassTier":       "Token",
		"VTID":                       "CosmeticVariantToken",
	}

	for prefix, backendValue := range typeToBackend {
		if strings.Contains(strings.ToLower(itemId), prefix) {
			return backendValue
		}
	}

	return ""
}
