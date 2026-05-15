package scraper

import (
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/remixfn/xenon/modules/storefront/models"
)

type Item struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	ID       string `json:"id"`
	Price    int    `json:"price"`
	Category string `json:"category"`
	Set      struct {
		Value        string `json:"value"`
		Text         string `json:"text"`
		BackendValue string `json:"backendValue"`
	} `json:"set"`
	Item struct {
		Value        string `json:"value"`
		Text         string `json:"text"`
		BackendValue string `json:"backendValue"`
	} `json:"item"`
	Images struct {
		Icon      string `json:"icon"`
		SmallIcon string `json:"smallIcon"`
	}
	Position    int    `json:"-"`
	IsBundle    bool   `json:"isBundle,omitempty"`
	BundleItems []Item `json:"bundleItems,omitempty"`
}

type Bundle struct {
	Name  string
	URL   string
	Items []Item
}

func GetStorefront(url string, items map[string]models.Item) map[string][]Item {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:137.0) Gecko/20100101 Firefox/137.0")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to fetch URL: %v", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatalf("Failed to parse HTML: %v", err)
	}

	shopData := map[string][]Item{}
	currentSection := "Featured"
	seenItems := map[string]bool{}
	position := 0
	bundlesToFetch := []Bundle{}

	doc.Find(".shop-section-title, .item-responsive").Each(func(i int, s *goquery.Selection) {
		if s.HasClass("shop-section-title") {
			currentSection = parseSectionTitle(s.Text())
			if _, ok := shopData[currentSection]; !ok {
				shopData[currentSection] = []Item{}
			}
			return
		}

		s.Find("a.item-display").Each(func(j int, itemEl *goquery.Selection) {
			item, bundle, valid := extractItem(itemEl, currentSection, items, seenItems, &position)
			if !valid {
				return
			}

			if bundle != nil {
				bundlesToFetch = append(bundlesToFetch, *bundle)
			}

			shopData[currentSection] = append(shopData[currentSection], item)
		})
	})

	if len(bundlesToFetch) > 0 {
		fetchBundleContents(client, bundlesToFetch, shopData, items)
	}

	for section := range shopData {
		sortItemsByPosition(shopData[section])
	}

	return shopData
}

func fetchBundleContents(client *http.Client, bundles []Bundle, shopData map[string][]Item, knownItems map[string]models.Item) {
	var wg sync.WaitGroup
	bundleResults := make(chan Bundle, len(bundles))

	semaphore := make(chan struct{}, 5)

	for _, bundle := range bundles {
		wg.Add(1)
		go func(b Bundle) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			bundleItems := fetchBundlePage(client, b.URL, knownItems)
			b.Items = bundleItems
			bundleResults <- b
		}(bundle)
	}

	go func() {
		wg.Wait()
		close(bundleResults)
	}()

	bundleMap := make(map[string][]Item)
	for result := range bundleResults {
		bundleMap[result.Name] = result.Items
	}

	for section, items := range shopData {
		for i, item := range items {
			if item.IsBundle {
				if bundleItems, exists := bundleMap[item.Name]; exists {
					shopData[section][i].BundleItems = bundleItems
				}
			}
		}
	}
}

func fetchBundlePage(client *http.Client, url string, knownItems map[string]models.Item) []Item {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:137.0) Gecko/20100101 Firefox/137.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil
	}

	var bundleItems []Item
	position := 0

	doc.Find(".items-row.bundle .item-responsive a.item-display").Each(func(i int, s *goquery.Selection) {
		if item, _, valid := extractBundleItem(s, knownItems, &position); valid {
			bundleItems = append(bundleItems, item)
		}
	})

	return bundleItems
}

func extractBundleItem(s *goquery.Selection, knownItems map[string]models.Item, pos *int) (Item, *Bundle, bool) {
	href, exists := s.Attr("href")
	if !exists || !strings.HasPrefix(href, "/") {
		return Item{}, nil, false
	}

	parts := strings.Split(strings.Trim(href, "/"), "/")
	if len(parts) < 2 {
		return Item{}, nil, false
	}
	itemType, itemName := parts[0], parts[1]

	name := convertURLToName(itemName)

	*pos++

	item := Item{
		Type:     itemType,
		Name:     name,
		Position: *pos,
	}

	var known models.Item
	var found bool

	if known, found = knownItems[name]; !found {
		if known, found = knownItems[itemName]; !found {
			titleCaseName := strings.Title(strings.ReplaceAll(itemName, "-", " "))
			if known, found = knownItems[titleCaseName]; !found {
				item.ID = itemType + "_" + itemName
				return item, nil, true
			}
		}
	}

	if found {
		if known.Images.Icon == "" {
			return Item{}, nil, false
		}
		item.ID = known.ID
		item.Category = known.Set.Value
		item.Set = known.Set
		item.Item = known.Item
		item.Images = known.Images
		item.Name = known.Name
	}

	return item, nil, true
}

func convertURLToName(urlName string) string {
	words := strings.Split(urlName, "-")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

func parseSectionTitle(raw string) string {
	section := strings.TrimSpace(raw)
	section = strings.ReplaceAll(section, " Items", "")
	section = strings.ReplaceAll(section, " ITEMS", "")
	if section == "" {
		return "Featured"
	}
	return section
}

func extractItem(s *goquery.Selection, section string, knownItems map[string]models.Item, seen map[string]bool, pos *int) (Item, *Bundle, bool) {
	href, exists := s.Attr("href")
	if !exists || !strings.HasPrefix(href, "/") {
		return Item{}, nil, false
	}

	parts := strings.Split(strings.Trim(href, "/"), "/")
	if len(parts) < 2 {
		return Item{}, nil, false
	}
	itemType, _ := parts[0], parts[1]

	name := strings.TrimSpace(s.Find(".item-name span").Text())
	if name == "" || name == "Unknown Item" {
		return Item{}, nil, false
	}

	isBundle := strings.Contains(name, "(Bundle)") || itemType == "bundle"

	itemKey := section + "-" + name
	if seen[itemKey] && !strings.Contains(name, "Battle Pass") {
		return Item{}, nil, false
	}
	seen[itemKey] = true
	*pos++

	var price int
	if isBundle {
		price = extractBundlePrice(s)
	} else {
		price = extractPrice(s)
	}

	if price <= 0 {
		price = 0
	}

	item := Item{
		Type:     itemType,
		ID:       "",
		Name:     name,
		Price:    price,
		Position: *pos,
		IsBundle: isBundle,
	}

	var bundle *Bundle
	if isBundle {
		baseURL := "https://fnbr.co"
		fullURL := baseURL + href
		bundle = &Bundle{
			Name: name,
			URL:  fullURL,
		}

		if known, ok := knownItems[name]; ok {
			item.ID = known.ID
			item.Category = known.Set.Value
			item.Set = known.Set
			item.Item = known.Item
			item.Images = known.Images
		} else {
			item.ID = "bundle_" + strings.ToLower(strings.ReplaceAll(name, " ", "_"))
		}
	} else {
		if known, ok := knownItems[name]; ok {
			if known.Images.Icon == "" {
				return Item{}, nil, false
			}
			item.ID = known.ID
			item.Category = known.Set.Value
			item.Set = known.Set
			item.Item = known.Item
			item.Images = known.Images
		} else if name == "Battle Pass Tiers" {
			item.ID = "AthenaBattlePassTier"
		} else {
			return Item{}, nil, false
		}
	}

	return item, bundle, true
}

func extractBundlePrice(s *goquery.Selection) int {
	priceSelectors := []string{
		".item-price",
		"p.item-price",
		".price-tag",
		"[data-price]",
	}

	for _, selector := range priceSelectors {
		element := s.Find(selector)
		if element.Length() > 0 {
			priceText := strings.TrimSpace(element.Text())
			if priceText != "" {
				re := regexp.MustCompile(`[0-9,]+`)
				matches := re.FindAllString(priceText, -1)
				if len(matches) > 0 {
					priceDigits := strings.ReplaceAll(matches[0], ",", "")
					if price, err := strconv.Atoi(priceDigits); err == nil {
						return price
					}
				}
			}
		}
	}

	return 0
}

func extractPrice(s *goquery.Selection) int {
	var priceText string
	priceSelectors := []string{
		".item-price",
		".price-tag",
		"[data-price]",
		".item-overlay .content-box .item-price",
		".content-box .item-price",
	}

	for _, selector := range priceSelectors {
		element := s.Find(selector)
		if element.Length() > 0 {
			if price := element.AttrOr("data-price", ""); price != "" {
				priceText = price
				break
			}

			if text := element.Text(); text != "" {
				priceText = strings.TrimSpace(text)
				break
			}
		}
	}

	if priceText == "" {
		return 0
	}

	re := regexp.MustCompile(`[0-9,]+`)
	matches := re.FindAllString(priceText, -1)

	if len(matches) == 0 {
		return 0
	}

	priceDigits := strings.ReplaceAll(matches[0], ",", "")
	price, err := strconv.Atoi(priceDigits)
	if err != nil {
		return 0
	}

	return price
}

func sortItemsByPosition(items []Item) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].Position < items[j].Position
	})
}
