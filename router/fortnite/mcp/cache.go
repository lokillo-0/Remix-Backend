package fortnite_mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/remixfn/xenon/classes/mcp"
)

type Item struct {
	AccountID string                 `json:"accountId"`
	Profile   string                 `json:"profileid"`
	Template  string                 `json:"templateid"`
	Value     map[string]interface{} `json:"value"`
	Quantity  int                    `json:"quantity"`
	IsStat    bool                   `json:"isStat"`
}

type BPOffer struct {
	AssetPathName       string `json:"AssetPathName"`
	SubPathString       string `json:"SubPathString"`
	OfferPriceRowHandle struct {
		DataTable interface{} `json:"DataTable"`
		RowName   interface{} `json:"RowName"`
	} `json:"OfferPriceRowHandle"`
	Quantity       int `json:"Quantity"`
	ChainedRewards []struct {
		AssetPathName string `json:"AssetPathName"`
		SubPathString string `json:"SubPathString"`
		Quantity      int    `json:"Quantity"`
	} `json:"ChainedRewards"`
}

type XPLevel struct {
	XpToNextLevel int    `json:"xpToNextLevel"`
	RewardItem    string `json:"rewardItem"`
}

type XPData map[string]XPLevel

type BattlePassRewards struct {
	Rewards []map[string]int `json:"rewards"`
}

type SeasonXP struct {
	Level         int    `json:"Level"`
	XpToNextLevel int    `json:"XpToNextLevel"`
	XpTotal       int    `json:"XpTotal"`
	RowId         string `json:"RowId"`
}

var (
	xpDataCache                 XPData
	athenaCache                 map[string]mcp.BaseItem
	isAthenaLoaded              atomic.Bool
	allCVT                      []mcp.CosmeticVariantToken
	allCVTMap                   map[string]mcp.CosmeticVariantToken
	battlePassCache             map[int]map[string]BPOffer
	battlePassCacheMutex        sync.RWMutex
	dailyQuestCache             map[int]map[string]DailyQuest
	dailyQuestCacheMutex        sync.RWMutex
	battlePassRewardsCache      map[string]BattlePassRewards
	battlePassRewardsCacheMutex sync.RWMutex

	seasonXPCache      map[string][]SeasonXP
	seasonXPCacheMutex sync.RWMutex
)

type DailyQuest struct {
	Rewards    map[string]int `json:"Rewards"`
	Objectives map[string]int `json:"Objectives"`
	Count      int            `json:"Count"`
}

func LoadBattlePassData(season string) (*BattlePassRewards, error) {
	battlePassRewardsCacheMutex.RLock()
	if data, exists := battlePassRewardsCache[season]; exists {
		battlePassRewardsCacheMutex.RUnlock()
		return &data, nil
	}
	battlePassRewardsCacheMutex.RUnlock()

	if battlePassRewardsCache == nil {
		battlePassRewardsCacheMutex.Lock()
		if battlePassRewardsCache == nil {
			battlePassRewardsCache = make(map[string]BattlePassRewards)
		}
		battlePassRewardsCacheMutex.Unlock()
	}

	filePath := filepath.Join("static", "battlepass", season, "bp.json")

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read battle pass file %s: %v", filePath, err)
	}

	var rewards BattlePassRewards
	if err := json.Unmarshal(fileData, &rewards); err != nil {
		return nil, fmt.Errorf("failed to parse battle pass data %s: %v", filePath, err)
	}

	battlePassRewardsCacheMutex.Lock()
	battlePassRewardsCache[season] = rewards
	battlePassRewardsCacheMutex.Unlock()

	return &rewards, nil
}

func LoadSeasonXPData(season string) ([]SeasonXP, error) {
	seasonXPCacheMutex.RLock()
	if data, exists := seasonXPCache[season]; exists {
		seasonXPCacheMutex.RUnlock()
		return data, nil
	}
	seasonXPCacheMutex.RUnlock()

	if seasonXPCache == nil {
		seasonXPCacheMutex.Lock()
		if seasonXPCache == nil {
			seasonXPCache = make(map[string][]SeasonXP)
		}
		seasonXPCacheMutex.Unlock()
	}

	filePath := filepath.Join("static", "battlepass", season, "xp.json")

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read xp file %s: %v", filePath, err)
	}

	var xp []SeasonXP
	if err := json.Unmarshal(fileData, &xp); err != nil {
		return nil, fmt.Errorf("failed to parse xp data %s: %v", filePath, err)
	}

	seasonXPCacheMutex.Lock()
	seasonXPCache[season] = xp
	seasonXPCacheMutex.Unlock()

	return xp, nil
}

type CatalogOffer struct {
	OfferID    string `json:"offerId"`
	OfferType  string `json:"offerType"`
	ItemGrants []struct {
		TemplateID string                 `json:"templateId"`
		Quantity   int                    `json:"quantity"`
		Attributes map[string]interface{} `json:"attributes,omitempty"`
	} `json:"itemGrants"`
	Prices []struct {
		CurrencyType string  `json:"currencyType"`
		FinalPrice   float64 `json:"finalPrice"`
	} `json:"prices"`
	DynamicBundleInfo *struct {
		DiscountedBasePrice int    `json:"discountedBasePrice"`
		RegularBasePrice    int    `json:"regularBasePrice"`
		FloorPrice          int    `json:"floorPrice"`
		CurrencyType        string `json:"currencyType"`
		BundleItems         []struct {
			BCanOwnMultiple            bool `json:"bCanOwnMultiple"`
			RegularPrice               int  `json:"regularPrice"`
			DiscountedPrice            int  `json:"discountedPrice"`
			AlreadyOwnedPriceReduction int  `json:"alreadyOwnedPriceReduction"`
			Item                       struct {
				TemplateID string                 `json:"templateId"`
				Quantity   int                    `json:"quantity"`
				Attributes map[string]interface{} `json:"attributes,omitempty"`
			} `json:"item"`
		} `json:"bundleItems"`
	} `json:"dynamicBundleInfo,omitempty"`
}

var OfferPool = sync.Pool{
	New: func() interface{} {
		return &CatalogOffer{}
	},
}

func LoadDailyQuestData(season int) (map[string]DailyQuest, error) {
	dailyQuestCacheMutex.RLock()
	if data, exists := dailyQuestCache[season]; exists {
		dailyQuestCacheMutex.RUnlock()
		return data, nil
	}
	dailyQuestCacheMutex.RUnlock()

	if dailyQuestCache == nil {
		dailyQuestCacheMutex.Lock()
		if dailyQuestCache == nil {
			dailyQuestCache = make(map[int]map[string]DailyQuest)
		}
		dailyQuestCacheMutex.Unlock()
	}

	filename := fmt.Sprintf("daily_s%d.json", season)
	filePath := filepath.Join("static", "athena", "quests", filename)

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read daily quest file for season %d: %v", season, err)
	}

	var quests map[string]DailyQuest
	if err := json.Unmarshal(fileData, &quests); err != nil {
		return nil, fmt.Errorf("failed to parse daily quest data for season %d: %v", season, err)
	}

	dailyQuestCacheMutex.Lock()
	dailyQuestCache[season] = quests
	dailyQuestCacheMutex.Unlock()

	return quests, nil
}

func LoadXPData() error {
	if xpDataCache != nil {
		return nil
	}

	data, err := os.ReadFile("static/battlepass/xp.json")
	if err != nil {
		return fmt.Errorf("failed to read XP data: %v", err)
	}

	var xpFile XPData

	if err := json.Unmarshal(data, &xpFile); err != nil {
		return fmt.Errorf("failed to parse XP data: %v", err)
	}

	xpDataCache = xpFile
	return nil
}

func GetXPData() (XPData, error) {
	if xpDataCache == nil {
		if err := LoadXPData(); err != nil {
			return nil, err
		}
	}
	return xpDataCache, nil
}

func LoadAthenaCache() {
	data, _ := os.ReadFile("static/athena/allitems.json")
	var items []Item
	if err := json.Unmarshal(data, &items); err != nil {
		return
	}

	cache := make(map[string]mcp.BaseItem, len(items))
	for _, item := range items {
		cache[item.Template] = mcp.BaseItem{
			Attributes: item.Value,
			TemplateId: item.Template,
			Quantity:   item.Quantity,
		}
	}

	athenaCache = cache
	isAthenaLoaded.Store(true)
}

func LoadPassData(season int) (map[string]BPOffer, error) {
	battlePassCacheMutex.RLock()
	if data, exists := battlePassCache[season]; exists {
		battlePassCacheMutex.RUnlock()
		return data, nil
	}
	battlePassCacheMutex.RUnlock()

	if battlePassCache == nil {
		battlePassCacheMutex.Lock()
		if battlePassCache == nil {
			battlePassCache = make(map[int]map[string]BPOffer)
		}
		battlePassCacheMutex.Unlock()
	}

	filename := fmt.Sprintf("s%d.json", season)
	filePath := filepath.Join("static", "battlepass", filename)

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var offers map[string]BPOffer
	if err := json.Unmarshal(fileData, &offers); err != nil {
		return nil, err
	}

	battlePassCacheMutex.Lock()
	battlePassCache[season] = offers
	battlePassCacheMutex.Unlock()

	return offers, nil
}

func GetAthenaCachedItems() map[string]mcp.BaseItem {
	return athenaCache
}

func IsAthenaCacheLoaded() bool {
	return isAthenaLoaded.Load()
}

func ApplyAthenaCache(profileData *mcp.Profile) {
	if !IsAthenaCacheLoaded() {
		return
	}

	originalItems := profileData.Items
	profileData.Items = make(map[string]interface{})
	for k, v := range originalItems {
		profileData.Items[k] = v
	}

	cache := athenaCache
	for templateID, baseItem := range cache {
		if _, exists := profileData.Items[templateID]; !exists {
			profileData.Items[templateID] = baseItem
		}
	}
}

func LoadCosmeticVariantTokens() (*[]mcp.CosmeticVariantToken, error) {
	dirPath := "static/athena"
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory %s does not exist: %v", dirPath, err)
	}

	path := fmt.Sprintf("%s/cvt.json", dirPath)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("file %s does not exist: %v", path, err)
	}

	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %v", path, err)
	}

	file = bytes.TrimPrefix(file, []byte("\xef\xbb\xbf"))

	var CVT map[string]mcp.CosmeticVariantToken
	err = json.Unmarshal(file, &CVT)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal file %s: %v", path, err)
	}

	allCVT = make([]mcp.CosmeticVariantToken, 0, len(CVT))
	allCVTMap = make(map[string]mcp.CosmeticVariantToken, len(CVT))
	for key, token := range CVT {
		allCVT = append(allCVT, token)
		allCVTMap[key] = token
	}

	return &allCVT, nil
}

func GetCosmeticVariantToken(key string) (*mcp.CosmeticVariantToken, error) {
	if token, exists := allCVTMap[key]; exists {
		return &token, nil
	} else {
		if _, err := LoadCosmeticVariantTokens(); err == nil {
			if token, exists := allCVTMap[key]; exists {
				return &token, nil
			}
		}
	}

	return nil, fmt.Errorf("cosmetic variant token not found: %s", key)
}

func GetAthenaCacheKeys() []string {
	if !IsAthenaCacheLoaded() {
		return nil
	}
	keys := make([]string, 0, len(athenaCache))
	for k := range athenaCache {
		keys = append(keys, k)
	}
	return keys
}
