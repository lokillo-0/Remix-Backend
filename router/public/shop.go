package public

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/classes/storefront"
	"github.com/remixfn/xenon/modules/database/buckets/fortnite"
)

type ShopItem struct {
	Name            string `json:"name"`
	Type            string `json:"type"`
	Price           int    `json:"price"`
	Banner          string `json:"banner"`
	ImageId         string `json:"imageId"`
	FeaturedImageId string `json:"featuredImageId"`
	OfferId         string `json:"offerId"`
	TileSize        string `json:"tileSize"`
}

type ShopSection struct {
	Name  string     `json:"name"`
	Items []ShopItem `json:"items"`
}

var (
	nameCache     = map[string]string{}
	nameCacheMu   sync.RWMutex
	nameCacheTime time.Time

)

func fetchDisplayNames(ids []string) map[string]string {
	if len(ids) == 0 {
		return nil
	}

	nameCacheMu.RLock()
	cacheAge := time.Since(nameCacheTime)
	nameCacheMu.RUnlock()

	if cacheAge < 10*time.Minute && len(nameCache) > 0 {
		nameCacheMu.RLock()
		result := make(map[string]string, len(ids))
		for _, id := range ids {
			if name, ok := nameCache[strings.ToLower(id)]; ok {
				result[strings.ToLower(id)] = name
			}
		}
		nameCacheMu.RUnlock()
		return result
	}

	params := url.Values{}
	params.Set("language", "en")
	seen := map[string]bool{}
	for _, id := range ids {
		low := strings.ToLower(id)
		if !seen[low] {
			params.Add("id", id)
			seen[low] = true
		}
	}

	reqURL := fmt.Sprintf("https://fortnite-api.com/v2/cosmetics/br/search/ids?%s", params.Encode())
	resp, err := http.Get(reqURL)
	if err != nil || resp.StatusCode != 200 {
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var apiResp struct {
		Data []struct {
			Id   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil
	}

	result := make(map[string]string)
	nameCacheMu.Lock()
	for _, item := range apiResp.Data {
		key := strings.ToLower(item.Id)
		nameCache[key] = item.Name
		result[key] = item.Name
	}
	nameCacheTime = time.Now()
	nameCacheMu.Unlock()

	return result
}

func itemTypeFromTemplate(prefix string) string {
	switch prefix {
	case "AthenaCharacter":
		return "Outfit"
	case "AthenaBackpack":
		return "Back Bling"
	case "AthenaPickaxe":
		return "Harvesting Tool"
	case "AthenaGlider":
		return "Glider"
	case "AthenaDance":
		return "Emote"
	case "AthenaItemWrap":
		return "Wrap"
	case "AthenaMusicPack":
		return "Music"
	case "AthenaLoadingScreen":
		return "Loading Screen"
	case "AthenaSkyDiveContrail":
		return "Contrail"
	case "AthenaSprint":
		return "Sprint Trail"
	}
	return "Item"
}

func GETPublicShop(c *gin.Context) {
	fetchEntries := func(storefrontName string) []storefront.CatalogEntry {
		found, err := odin.FindWhere("Catalog", map[string]interface{}{
			"storefront": storefrontName,
		}, func() interface{} { return &fortnite.Catalog{} })
		if err != nil || found == nil {
			return nil
		}
		var entries []storefront.CatalogEntry
		seen := map[string]bool{}
		for _, v := range found {
			catalog := v.(*fortnite.Catalog)
			var entry storefront.CatalogEntry
			if err := json.Unmarshal([]byte(catalog.Data), &entry); err != nil {
				continue
			}
			if seen[entry.OfferId] {
				continue
			}
			seen[entry.OfferId] = true
			entries = append(entries, entry)
		}
		return entries
	}

	daily := fetchEntries("BRDailyStorefront")
	weekly := fetchEntries("BRWeeklyStorefront")

	sort.Slice(daily, func(i, j int) bool { return daily[i].OfferId < daily[j].OfferId })
	sort.Slice(weekly, func(i, j int) bool { return weekly[i].OfferId < weekly[j].OfferId })

	all := append(daily, weekly...)

	var cosmeticIds []string
	for _, entry := range all {
		if len(entry.ItemGrants) > 0 {
			parts := strings.SplitN(entry.ItemGrants[0].TemplateId, ":", 2)
			if len(parts) == 2 {
				cosmeticIds = append(cosmeticIds, parts[1])
			}
		}
	}
	displayNames := fetchDisplayNames(cosmeticIds)

	sectionMap := map[string][]ShopItem{}
	sectionOrder := []string{}

	processEntries := func(entries []storefront.CatalogEntry) {
		for _, entry := range entries {
			sectionName := "Featured"
			if entry.Meta.SectionId != "" {
				sectionName = entry.Meta.SectionId
			} else if len(entry.Categories) > 0 {
				sectionName = entry.Categories[0]
			}

			price := 0
			if len(entry.Prices) > 0 {
				price = entry.Prices[0].FinalPrice
			}

			if price <= 0 {
				continue
			}

			name := ""
			itemType := "Item"
			imageId := ""

			if len(entry.ItemGrants) > 0 {
				tplId := entry.ItemGrants[0].TemplateId
				parts := strings.SplitN(tplId, ":", 2)
				if len(parts) == 2 {
					rawId := parts[1]
					imageId = rawId
					itemType = itemTypeFromTemplate(parts[0])

					if displayNames != nil {
						if n, ok := displayNames[strings.ToLower(rawId)]; ok {
							name = n
						}
					}
					if name == "" {
						name = rawId
					}
				} else {
					imageId = tplId
					name = tplId
				}
			}

			if name == "" {
				name = entry.DevName
			}

			tileSize := "Normal"
			if entry.Meta.TileSize != "" {
				tileSize = entry.Meta.TileSize
			}

			featuredImageId := ""
			if entry.Meta.NewDisplayAssetPath != "" {
				parts := strings.Split(entry.Meta.NewDisplayAssetPath, ".")
				last := parts[len(parts)-1]
				last = strings.TrimPrefix(last, "DAv2_Featured_")
				last = strings.TrimPrefix(last, "DAv2_")
				if last != "" && last != imageId {
					featuredImageId = last
				}
			}

			item := ShopItem{
				Name:            name,
				Type:            itemType,
				Price:           price,
				Banner:          entry.BannerOverride,
				ImageId:         imageId,
				FeaturedImageId: featuredImageId,
				OfferId:         entry.OfferId,
				TileSize:        tileSize,
			}

			if _, exists := sectionMap[sectionName]; !exists {
				sectionOrder = append(sectionOrder, sectionName)
			}
			sectionMap[sectionName] = append(sectionMap[sectionName], item)
		}
	}

	processEntries(daily)
	processEntries(weekly)

	sort.Strings(sectionOrder)

	sections := make([]ShopSection, 0, len(sectionOrder))
	for _, name := range sectionOrder {
		items := sectionMap[name]
		sort.Slice(items, func(i, j int) bool {
			return items[i].OfferId < items[j].OfferId
		})
		sections = append(sections, ShopSection{Name: name, Items: items})
	}

	c.JSON(http.StatusOK, sections)
}
