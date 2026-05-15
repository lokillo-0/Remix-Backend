package discovery

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/utilities"
)

var versionPlaylistMap = map[string][]string{
	"27.11": {"playlist_durian"},
	"32.11": {"playlist_quail"},
	"35.20": {"playlist_ripehoneydew"},
	"37.51": {"playlist_limerock"},
	"38.11": {"playlist_skymango"},
}

func parseBuildFloat(build string) float64 {
	f, _ := strconv.ParseFloat(build, 64)
	return f
}

func activateVersionPlaylists(discovery []map[string]interface{}, build string) {
	buildF := parseBuildFloat(build)

	activate := func(mnemonic string) {
		for _, item := range discovery {
			if item["mnemonic"] == mnemonic {
				item["active"] = true
				item["disabled"] = false
				break
			}
		}
	}

	if buildF >= 32.11 {
		activate("set_blastberry_playlists")
	}
	if buildF >= 33.00 {
		activate("set_figment_playlists")
	}
	if buildF >= 36.10 {
		activate("set_forbiddenfruit_nobuild_playlists")
	}
	if buildF >= 37.31 {
		activate("playlist_stridemice")
	}

	if playlists, ok := versionPlaylistMap[build]; ok {
		for _, p := range playlists {
			activate(p)
		}
	}
}

func buildSurfacePanels(build string) gin.H {
	buildF := parseBuildFloat(build)

	makeResult := func(linkCode string) gin.H {
		ccu := 0
		if playlists, ok := linkCodePlaylists[linkCode]; ok {
			ccu = playlistCCU(playlists...)
		}
		return gin.H{
			"lastVisited":      nil,
			"linkCode":         linkCode,
			"isFavorite":       false,
			"globalCCU":        ccu,
			"lockStatus":       "UNLOCKED",
			"lockStatusReason": "NONE",
			"isVisible":        true,
		}
	}

	byEpicResults := []gin.H{
		makeResult("set_br_playlists"),
	}
	if buildF >= 30.10 {
		byEpicResults = append(byEpicResults, makeResult("set_blastberry_playlists"))
	}
	if buildF >= 33.00 {
		byEpicResults = append(byEpicResults, makeResult("playlist_pilgrimquickplay"))
		byEpicResults = append(byEpicResults, makeResult("playlist_juno"))
		byEpicResults = append(byEpicResults, makeResult("playlist_beanstalk"))
		byEpicResults = append(byEpicResults, makeResult("playlist_papaya"))
	}

	panels := []gin.H{
		{
			"panelName":        "Homebar",
			"panelDisplayName": "Homebar",
			"panelSubtitle":    nil,
			"featureTags":      []string{"col:5", "homebar"},
			"firstPage": gin.H{
				"results": []gin.H{{
					"lastVisited":      nil,
					"linkCode":         "ref_panel_byepicfeeder_1",
					"isFavorite":       false,
					"globalCCU":        -1,
					"lockStatus":       "UNLOCKED",
					"lockStatusReason": "NONE",
					"isVisible":        true,
				}},
				"hasMore":         true,
				"panelTargetName": nil,
				"pageMarker":      nil,
			},
			"panelType":       "CuratedList",
			"playHistoryType": nil,
		},
		{
			"panelName":              "ByEpicFeeder",
			"panelNativeDisplayName": "By Epic",
			"panelDisplayName":       "By Epic",
			"panelSubtitle":          nil,
			"featureTags":            []string{"ForReferenceViewOnly", "col:5", "hasViewAll:true", "horizontalScroll:false"},
			"firstPage": gin.H{
				"results":         byEpicResults,
				"hasMore":         true,
				"panelTargetName": nil,
				"pageMarker":      nil,
			},
			"panelType":       "AnalyticsList",
			"playHistoryType": nil,
		},
	}

	if buildF >= 30.20 {
		convergenceResults := []gin.H{
			//makeResult("campaign"),
		}
		if buildF >= 33.00 {
			convergenceResults = append(convergenceResults, makeResult("set_figment_playlists"))
		}

		//if buildF >= 30.10 {
		//convergenceResults = append(convergenceResults, makeResult("set_blastberry_playlists"))
		//}
		if buildF >= 36.10 {
			convergenceResults = append(convergenceResults, makeResult("set_forbiddenfruit_nobuild_playlists"))
		}
		panels = append(panels, gin.H{
			"panelName":              "ByEpicConvergenceBlastberry",
			"panelNativeDisplayName": "Other Modes By Epic",
			"panelDisplayName":       "Other Modes By Epic",
			"panelSubtitle":          nil,
			"featureTags":            []string{"col:7", "horizontalScroll:false", "hasViewAll:true", "squareTiles:false", "grid:4"},
			"firstPage": gin.H{
				"results":         convergenceResults,
				"hasMore":         true,
				"panelTargetName": nil,
				"pageMarker":      nil,
			},
			"panelType":       "AnalyticsList",
			"playHistoryType": nil,
		})
	}

	type ltmPanel struct {
		minBuild float64
		maxBuild float64
		name     string
		display  string
		linkCode string
	}

	ltmPanels := []ltmPanel{
		//{32.00, 33.00, "Quail", "Remix: The Finale", "playlist_quail"},
		{35.20, 36.00, "RipeHoneyDew", "Death Star Sabotage", "playlist_ripehoneydew"},
		{37.51, 38.11, "LimeRock", "Welcome, Our Alien Overlords", "playlist_limerock"},
		{38.11, 39.00, "SkyMango", "Chapter Finale Zero Hour", "playlist_skymango"},
		{37.31, 38.11, "StrideMice", "The Daft Punk Experience", "playlist_stridemice"},
	}

	for _, lp := range ltmPanels {
		if buildF >= lp.minBuild && buildF < lp.maxBuild {
			r := makeResult(lp.linkCode)
			r["globalCCU"] = -1
			r["lockStatusReason"] = "RATING_THRESHOLD"
			r["favoriteStatus"] = "NONE"
			panels = append(panels, gin.H{
				"panelName":        lp.name,
				"panelDisplayName": lp.display,
				"panelSubtitle":    lp.display,
				"featureTags":      []string{"bannerItemRow"},
				"firstPage": gin.H{
					"results":         []gin.H{r},
					"hasMore":         false,
					"panelTargetName": nil,
					"pageMarker":      nil,
				},
				"panelType":       "CuratedList",
				"playHistoryType": nil,
				"panelContexts":   gin.H{},
			})
		}
	}

	return gin.H{"panels": panels}
}

type Panel struct {
	PanelName        string      `json:"panelName"`
	PanelDisplayName string      `json:"panelDisplayName"`
	FeatureTags      []string    `json:"featureTags"`
	FirstPage        FirstPage   `json:"firstPage"`
	PanelType        string      `json:"panelType"`
	PlayHistoryType  interface{} `json:"playHistoryType"`
}

type FirstPage struct {
	Results         []Result    `json:"results"`
	HasMore         bool        `json:"hasMore"`
	PanelTargetName interface{} `json:"panelTargetName"`
}

type Result struct {
	LastVisited interface{} `json:"lastVisited"`
	LinkCode    string      `json:"linkCode"`
	IsFavorite  bool        `json:"isFavorite"`
	GlobalCCU   int         `json:"globalCCU"`
}

type SurfaceResponse struct {
	Panels []Panel `json:"panels"`
}

type AssetsResponse struct {
	FortCreativeDiscoverySurface AssetGroup `json:"FortCreativeDiscoverySurface"`
}

type AssetGroup struct {
	Meta   AssetMeta           `json:"meta"`
	Assets map[string]AssetSet `json:"assets"`
}

type AssetMeta struct {
	Promotion int `json:"promotion"`
}

type AssetSet struct {
	Meta      AssetSetMeta `json:"meta"`
	AssetData AssetData    `json:"assetData"`
}

type AssetSetMeta struct {
	Revision     int    `json:"revision"`
	HeadRevision int    `json:"headRevision"`
	RevisedAt    string `json:"revisedAt"`
	Promotion    int    `json:"promotion"`
	PromotedAt   string `json:"promotedAt"`
}

type AssetData struct {
	AnalyticsId             string       `json:"AnalyticsId"`
	TestCohorts             []TestCohort `json:"TestCohorts"`
	GlobalLinkCodeBlacklist []string     `json:"GlobalLinkCodeBlacklist"`
	SurfaceName             string       `json:"SurfaceName"`
	TestName                string       `json:"TestName"`
	PrimaryAssetId          string       `json:"primaryAssetId"`
	GlobalLinkCodeWhitelist []string     `json:"GlobalLinkCodeWhitelist"`
}

type TestCohort struct {
	AnalyticsId          string         `json:"AnalyticsId"`
	CohortSelector       string         `json:"CohortSelector"`
	PlatformBlacklist    []string       `json:"PlatformBlacklist"`
	CountryCodeBlocklist []string       `json:"CountryCodeBlocklist"`
	ContentPanels        []ContentPanel `json:"ContentPanels"`
	PlatformWhitelist    []string       `json:"PlatformWhitelist"`
	SelectionChance      float64        `json:"SelectionChance"`
	TestName             string         `json:"TestName"`
}

type ContentPanel struct {
	NumPages               int             `json:"NumPages"`
	AnalyticsId            string          `json:"AnalyticsId"`
	PanelType              string          `json:"PanelType"`
	AnalyticsListName      string          `json:"AnalyticsListName"`
	CuratedListOfLinkCodes []string        `json:"CuratedListOfLinkCodes"`
	ModelName              string          `json:"ModelName"`
	PageSize               int             `json:"PageSize"`
	PlatformBlacklist      []string        `json:"PlatformBlacklist"`
	PanelName              string          `json:"PanelName"`
	MetricInterval         string          `json:"MetricInterval"`
	CountryCodeBlocklist   []string        `json:"CountryCodeBlocklist"`
	SkippedEntriesCount    int             `json:"SkippedEntriesCount"`
	SkippedEntriesPercent  int             `json:"SkippedEntriesPercent"`
	SplicedEntries         []interface{}   `json:"SplicedEntries"`
	PlatformWhitelist      []string        `json:"PlatformWhitelist"`
	MMRegionBlocklist      []string        `json:"MMRegionBlocklist"`
	EntrySkippingMethod    string          `json:"EntrySkippingMethod"`
	PanelDisplayName       LocalizedString `json:"PanelDisplayName"`
	PlayHistoryType        string          `json:"PlayHistoryType"`
	BLowestToHighest       bool            `json:"bLowestToHighest"`
	PanelLinkCodeBlacklist []string        `json:"PanelLinkCodeBlacklist"`
	CountryCodeAllowlist   []string        `json:"CountryCodeAllowlist"`
	PanelLinkCodeWhitelist []string        `json:"PanelLinkCodeWhitelist"`
	FeatureTags            []string        `json:"FeatureTags"`
	MMRegionAllowlist      []string        `json:"MMRegionAllowlist"`
	MetricName             string          `json:"MetricName"`
}

type LocalizedString struct {
	Category         string        `json:"Category"`
	NativeCulture    string        `json:"NativeCulture"`
	Namespace        string        `json:"Namespace"`
	LocalizedStrings []interface{} `json:"LocalizedStrings"`
	BIsMinimalPatch  bool          `json:"bIsMinimalPatch"`
	NativeString     string        `json:"NativeString"`
	Key              string        `json:"Key"`
}

type MnemonicResponse struct {
	ParentLinks []interface{}           `json:"parentLinks"`
	Links       map[string]PlaylistLink `json:"links"`
}

type PlaylistLink struct {
	Namespace        string           `json:"namespace"`
	AccountId        string           `json:"accountId"`
	CreatorName      string           `json:"creatorName"`
	Mnemonic         string           `json:"mnemonic"`
	LinkType         string           `json:"linkType"`
	Metadata         PlaylistMetadata `json:"metadata"`
	Version          int              `json:"version"`
	Active           bool             `json:"active"`
	Disabled         bool             `json:"disabled"`
	Created          string           `json:"created"`
	Published        string           `json:"published"`
	DescriptionTags  []string         `json:"descriptionTags"`
	ModerationStatus string           `json:"moderationStatus"`
}

type PlaylistMetadata struct {
	ImageUrl    string      `json:"image_url"`
	ImageUrls   ImageUrls   `json:"image_urls"`
	Matchmaking Matchmaking `json:"matchmaking"`
}

type ImageUrls struct {
	UrlS  string `json:"url_s"`
	UrlXs string `json:"url_xs"`
	UrlM  string `json:"url_m"`
	Url   string `json:"url"`
}

type Matchmaking struct {
	OverridePlaylist string `json:"override_playlist"`
}

func HandleDiscoverySurface(c *gin.Context) {
	ua := utilities.Parse(c.GetHeader("User-Agent"))
	if ua != nil && parseBuildFloat(ua.Build) >= 23.00 {
		panels := buildSurfacePanels(ua.Build)
		if panelList, ok := panels["panels"].([]gin.H); ok {
			names := make([]string, 0, len(panelList))
			for _, p := range panelList {
				if name, ok := p["panelName"].(string); ok {
					names = append(names, name)
				}
			}
		}
		c.JSON(http.StatusOK, panels)
		return
	}

	data, err := ioutil.ReadFile("static/discovery/menu.json")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load menu data"})
		return
	}

	var menuData interface{}
	if err := json.Unmarshal(data, &menuData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse menu data"})
		return
	}

	c.JSON(http.StatusOK, menuData)
}

func buildPlaylistAssetEntry(name string) map[string]interface{} {
	data, err := ioutil.ReadFile("static/assets/playlists/" + name + ".json")
	if err != nil {
		return nil
	}
	var p map[string]interface{}
	if err := json.Unmarshal(data, &p); err != nil {
		return nil
	}
	assetData, _ := p["assetData"]
	return map[string]interface{}{
		"meta": map[string]interface{}{
			"revision":     2,
			"headRevision": 2,
			"revisedAt":    "2023-11-27T06:41:57.818Z",
			"promotion":    3,
			"promotedAt":   "2023-11-27T06:43:00.452Z",
		},
		"assetData": assetData,
	}
}

func HandleAssets(c *gin.Context) {
	playlistAssets := map[string]interface{}{}
	for _, name := range []string{"Playlist_DefaultSolo", "Playlist_DefaultDuo", "Playlist_Trios", "Playlist_NoBuildBR_Solo", "Playlist_NoBuildBR_Squad"} {
		if entry := buildPlaylistAssetEntry(name); entry != nil {
			playlistAssets[name] = entry
		}
	}

	ua := utilities.Parse(c.GetHeader("User-Agent"))
	if ua != nil {
		if playlists, ok := versionPlaylistMap[ua.Build]; ok {
			for _, mnemonic := range playlists {
				assetName := ""
				switch mnemonic {
				case "playlist_quail":
					assetName = "Playlist_Quail"
				case "playlist_ripehoneydew":
					assetName = "Playlist_RipeHoneyDew"
				case "playlist_limerock":
					assetName = "Playlist_LimeRock"
				case "playlist_skymango":
					assetName = "Playlist_SkyMango"
				}
				if assetName != "" {
					if entry := buildPlaylistAssetEntry(assetName); entry != nil {
						playlistAssets[assetName] = entry
					}
				}
			}
		}
	}

	fortPlaylistAthena := map[string]interface{}{
		"meta":   map[string]interface{}{"promotion": 9},
		"assets": playlistAssets,
	}

	response := AssetsResponse{
		FortCreativeDiscoverySurface: AssetGroup{
			Meta: AssetMeta{
				Promotion: 26,
			},
			Assets: map[string]AssetSet{
				"CreativeDiscoverySurface_Frontend": {
					Meta: AssetSetMeta{
						Revision:     32,
						HeadRevision: 32,
						RevisedAt:    "2023-04-25T19:30:52.489Z",
						Promotion:    26,
						PromotedAt:   "2023-04-25T19:31:12.618Z",
					},
					AssetData: AssetData{
						AnalyticsId: "v538",
						TestCohorts: []TestCohort{
							{
								AnalyticsId:          "c-1v2_v2_c727",
								CohortSelector:       "PlayerDeterministic",
								PlatformBlacklist:    []string{},
								CountryCodeBlocklist: []string{},
								ContentPanels: []ContentPanel{
									{
										NumPages:               1,
										AnalyticsId:            "p1114",
										PanelType:              "AnalyticsList",
										AnalyticsListName:      "ByEpicNoBigBattle",
										CuratedListOfLinkCodes: []string{},
										ModelName:              "",
										PageSize:               7,
										PlatformBlacklist:      []string{},
										PanelName:              "ByEpicNoBigBattle6Col",
										MetricInterval:         "",
										CountryCodeBlocklist:   []string{},
										SkippedEntriesCount:    0,
										SkippedEntriesPercent:  0,
										SplicedEntries:         []interface{}{},
										PlatformWhitelist:      []string{},
										MMRegionBlocklist:      []string{},
										EntrySkippingMethod:    "None",
										PanelDisplayName: LocalizedString{
											Category:         "Game",
											NativeCulture:    "",
											Namespace:        "CreativeDiscoverySurface_Frontend",
											LocalizedStrings: []interface{}{},
											BIsMinimalPatch:  false,
											NativeString:     "LTMS",
											Key:              "ByEpicNoBigBattle6Col",
										},
										PlayHistoryType:        "RecentlyPlayed",
										BLowestToHighest:       false,
										PanelLinkCodeBlacklist: []string{},
										CountryCodeAllowlist:   []string{},
										PanelLinkCodeWhitelist: []string{},
										FeatureTags:            []string{"col:5"},
										MMRegionAllowlist:      []string{},
										MetricName:             "",
									},
									{
										NumPages:               2,
										AnalyticsId:            "p969|88dba0c4e2af76447df43d1e31331a3d",
										PanelType:              "AnalyticsList",
										AnalyticsListName:      "EventPanel",
										CuratedListOfLinkCodes: []string{},
										ModelName:              "",
										PageSize:               25,
										PlatformBlacklist:      []string{},
										PanelName:              "EventPanel",
										MetricInterval:         "",
										CountryCodeBlocklist:   []string{},
										SkippedEntriesCount:    0,
										SkippedEntriesPercent:  0,
										SplicedEntries:         []interface{}{},
										PlatformWhitelist:      []string{},
										MMRegionBlocklist:      []string{},
										EntrySkippingMethod:    "None",
										PanelDisplayName: LocalizedString{
											Category:         "Game",
											NativeCulture:    "",
											Namespace:        "CreativeDiscoverySurface_Frontend",
											LocalizedStrings: []interface{}{},
											BIsMinimalPatch:  false,
											NativeString:     "Event LTMS",
											Key:              "EventPanel",
										},
										PlayHistoryType:        "RecentlyPlayed",
										BLowestToHighest:       false,
										PanelLinkCodeBlacklist: []string{},
										CountryCodeAllowlist:   []string{},
										PanelLinkCodeWhitelist: []string{},
										FeatureTags:            []string{"col:6"},
										MMRegionAllowlist:      []string{},
										MetricName:             "",
									},
								},
								PlatformWhitelist: []string{},
								SelectionChance:   0.1,
								TestName:          "testing",
							},
						},
						GlobalLinkCodeBlacklist: []string{},
						SurfaceName:             "CreativeDiscoverySurface_Frontend",
						TestName:                "20.10_4/11/2022_hero_combat_popularConsole",
						PrimaryAssetId:          "FortCreativeDiscoverySurface:CreativeDiscoverySurface_Frontend",
						GlobalLinkCodeWhitelist: []string{},
					},
				},
			},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"FortCreativeDiscoverySurface": response.FortCreativeDiscoverySurface,
		"FortPlaylistAthena":           fortPlaylistAthena,
	})
}

func HandleCreativeDiscoverySurface(c *gin.Context) {
	ua := utilities.Parse(c.GetHeader("User-Agent"))
	if ua != nil && parseBuildFloat(ua.Build) >= 23.00 {
		c.JSON(http.StatusOK, buildSurfacePanels(ua.Build))
		return
	}

	data, err := ioutil.ReadFile("static/discovery/menu.json")
	if err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	var menuData interface{}
	if err := json.Unmarshal(data, &menuData); err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	c.JSON(http.StatusOK, menuData)
}

func HandleDiscoverySurfaceV1(c *gin.Context) {
	data, err := ioutil.ReadFile("static/discovery/menu.json")
	if err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	var menuData interface{}
	if err := json.Unmarshal(data, &menuData); err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	c.JSON(http.StatusOK, menuData)
}

func HandleMnemonic(c *gin.Context) {
	ua := utilities.Parse(c.GetHeader("User-Agent"))
	if ua == nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	buildF := parseBuildFloat(ua.Build)

	if buildF >= 23.00 {
		data, err := ioutil.ReadFile("static/discovery/latest/menu.json")
		if err != nil {
			utilities.Internal.ServerError().Apply(c.Writer)
			return
		}

		var discovery []map[string]interface{}
		if err := json.Unmarshal(data, &discovery); err != nil {
			utilities.Internal.ServerError().Apply(c.Writer)
			return
		}

		activateVersionPlaylists(discovery, ua.Build)
		c.JSON(http.StatusOK, discovery)
		return
	}

	data, err := ioutil.ReadFile("static/discovery/menu.json")
	if err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	var normal map[string]interface{}
	if err := json.Unmarshal(data, &normal); err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	if panels, ok := normal["Panels"].([]interface{}); ok && len(panels) > 0 {
		if panel, ok := panels[0].(map[string]interface{}); ok {
			if pages, ok := panel["Pages"].([]interface{}); ok && len(pages) > 0 {
				if page, ok := pages[0].(map[string]interface{}); ok {
					if results, ok := page["results"].([]interface{}); ok {
						linkData := make([]interface{}, 0, len(results))
						for _, result := range results {
							if resultMap, ok := result.(map[string]interface{}); ok {
								if linkDataItem, exists := resultMap["linkData"]; exists {
									linkData = append(linkData, linkDataItem)
								}
							}
						}
						c.JSON(http.StatusOK, linkData)
						return
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, []interface{}{})
}

func HandleRelatedPlaylist(c *gin.Context) {
	playlistId := c.Param("playlistId")

	ua := utilities.Parse(c.GetHeader("User-Agent"))
	if ua == nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	var filePath string
	if ua.Build >= "23.50" {
		filePath = "static/discovery/latest/menu.json"
	} else {
		filePath = "static/discovery/menu.json"
	}

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	var discovery []map[string]interface{}
	if err := json.Unmarshal(data, &discovery); err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	if ua != nil {
		activateVersionPlaylists(discovery, ua.Build)
	}

	rawResponse := gin.H{
		"parentLinks": []interface{}{},
		"links":       map[string]interface{}{},
	}
	links := rawResponse["links"].(map[string]interface{})
	parentLinks := rawResponse["parentLinks"].([]interface{})

	var current map[string]interface{}
	for _, item := range discovery {
		if item["mnemonic"] == playlistId {
			current = item
			break
		}
	}

	if current == nil {
		c.JSON(http.StatusOK, rawResponse)
		return
	}

	metadata, _ := current["metadata"].(map[string]interface{})

	if subLinks, ok := metadata["sub_link_codes"].([]interface{}); ok {
		parentLinks = append(parentLinks, current)

		for _, sub := range subLinks {
			code, _ := sub.(string)
			for _, item := range discovery {
				if item["mnemonic"] == code {
					links[code] = item
				}
			}
		}
	} else {
		code := current["mnemonic"].(string)
		links[code] = current

		if parentSetId, ok := metadata["parent_set"].(string); ok {
			var parent map[string]interface{}
			for _, item := range discovery {
				if item["mnemonic"] == parentSetId {
					parent = item
					break
				}
			}

			if parent != nil {
				parentLinks = append(parentLinks, parent)

				parentMeta, _ := parent["metadata"].(map[string]interface{})
				if subLinks, ok := parentMeta["sub_link_codes"].([]interface{}); ok {
					for _, sub := range subLinks {
						subCode, _ := sub.(string)
						if _, exists := links[subCode]; exists {
							continue
						}
						for _, item := range discovery {
							if item["mnemonic"] == subCode {
								links[subCode] = item
							}
						}
					}
				}
			}
		}
	}

	rawResponse["parentLinks"] = parentLinks
	rawResponse["links"] = links
	c.JSON(http.StatusOK, rawResponse)
}

func convertToPlaylistLink(item map[string]interface{}) PlaylistLink {
	mnemonic, _ := item["mnemonic"].(string)

	return PlaylistLink{
		Namespace:   "fn",
		AccountId:   "epic",
		CreatorName: "Epic",
		Mnemonic:    mnemonic,
		LinkType:    "BR:Playlist",
		Metadata: PlaylistMetadata{
			Matchmaking: Matchmaking{
				OverridePlaylist: mnemonic,
			},
		},
		Version:          1,
		Active:           true,
		Disabled:         false,
		Created:          "",
		Published:        "",
		DescriptionTags:  []string{},
		ModerationStatus: "Approved",
	}
}

func GenerateRandomHex(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)
}
