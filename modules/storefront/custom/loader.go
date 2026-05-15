package custom

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/remixfn/xenon/modules/storefront/scraper"
)

type ShopItem struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Price   int    `json:"price"`
	Section string `json:"section"`
}

func LoadShop(path string) (map[string][]scraper.Item, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var items []ShopItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	result := make(map[string][]scraper.Item)
	for _, item := range items {
		section := item.Section
		if section == "" {
			section = "Featured"
		}

		// Accept full templateId (e.g. "AthenaCharacter:CID_017...") or short id
		id := item.ID
		itemType := ""
		if idx := strings.Index(id, ":"); idx != -1 {
			itemType = id[:idx]
			id = id[idx+1:]
		}

		s := scraper.Item{
			ID:    id,
			Name:  item.Name,
			Price: item.Price,
			Type:  itemType,
		}
		result[section] = append(result[section], s)
	}
	return result, nil
}
