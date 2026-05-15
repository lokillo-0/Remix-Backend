package fortnite

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/modules/database/buckets/fortnite"
	"github.com/remixfn/xenon/utilities"
)

func FortniteContentPagesHandler(c *gin.Context) {
	if c.Param("any") == "/media-events-v2" {
		FortniteMediaEventsV2Handler(c)
		return
	}
	ua := utilities.Parse(c.GetHeader("User-Agent"))

	if ua == nil {
		utilities.Basic.BadRequest().Apply(c.Writer)
		return
	}
	seasonStr := strconv.Itoa(ua.Season)
	backgroundStage := "season" + seasonStr
	backgroundImage := ""

	switch ua.Season {
	case 10:
		if ua.Build == "10.40" {
			backgroundStage = "blackmonday"
		} else {
			backgroundStage = "seasonx"
		}
	case 11:
		switch ua.Build {
		case "11.10":
			backgroundStage = "fortnitemares"
		case "11.30":
			backgroundStage = "Galileo"
		case "11.31", "11.40":
			backgroundStage = "Winter19"
		default:
			backgroundStage = "season11"
		}
	case 12:
		backgroundStage = "season12"
	case 13:
		backgroundStage = "season13"
	case 14:
		backgroundStage = "season14"
	case 15:
		if ua.Build == "15.10" {
			backgroundStage = "season15xmas"
		} else {
			backgroundStage = "season15"
		}
	case 16:
		backgroundStage = "season16"
	case 17:
		switch ua.Build {
		case "17.10":
			backgroundStage = "defaultnotris"
			backgroundImage = "https://cdn2.unrealengine.com/t-bp-17-lobby-summer-2048x1024-709fa99e6be0.png"
		case "17.21":
			backgroundStage = "defaultnotris"
			backgroundImage = "https://cdn2.unrealengine.com/t-bp17-21-lobby-2048x1024-f6027bf109de.png"
		case "17.40":
			backgroundStage = "defaultnotris"
			backgroundImage = "https://cdn2.unrealengine.com/t-bp17-40-lobby-2048x1024-f742fc604aae.png"
		}
	case 18:
		backgroundStage = "season18"
	case 19:
		switch ua.Build {
		case "19.01":
			backgroundStage = "winter2021"
			backgroundImage = "https://cdn2.unrealengine.com/t-bp19-lobby-xmas-2048x1024-f85d2684b4af.png"
		case "19.10":
			backgroundStage = "defaultnotris"
			backgroundImage = "https://cdn2.unrealengine.com/fortnite-tilted-towers-1920x1080-ad94e5f0b016.jpg"
		default:
			backgroundStage = "season19"
		}
	case 20:
		switch ua.Build {
		case "20.10":
			backgroundStage = "season20"
			backgroundImage = "https://cdn2.unrealengine.com/t-bp20-lobby-2048x1024-d89eb522746c.png"
		case "20.40":
			backgroundStage = "season20"
			backgroundImage = "https://cdn2.unrealengine.com/t-bp20-40-armadillo-glowup-lobby-2048x2048-2048x2048-3b83b887cc7f.jpg"
		default:
			backgroundStage = "season20"
			backgroundImage = "https://cdn2.unrealengine.com/s20-landscapev4-2048x1024-2494a103ae6c.png"
		}
	case 21:
		if ua.Build == "21.30" {
			backgroundStage = "season2130"
			backgroundImage = "https://cdn2.unrealengine.com/nss-lobbybackground-2048x1024-f74a14565061.jpg"
		} else {
			backgroundStage = "season2100"
			backgroundImage = "https://cdn2.unrealengine.com/s21-lobby-background-2048x1024-2e7112b25dc3.jpg"
		}
	case 22:
		if ua.Build == "22.20" {
			backgroundStage = "season2220"
			backgroundImage = "https://cdn2.unrealengine.com/t-bp22-fortnitemares-lobby-square-2048x2048-2048x2048-3b7cda3aa517.jpg"
		} else {
			backgroundStage = "season2200"
			backgroundImage = "https://cdn2.unrealengine.com/t-bp22-lobby-square-2048x2048-2048x2048-e4e90c6e8018.jpg"
		}
	case 23:
		switch ua.Build {
		case "23.10":
			backgroundStage = "season2310"
			backgroundImage = "https://cdn2.unrealengine.com/t-bp23-winterfest-lobby-square-2048x2048-2048x2048-277a476e5ca6.png"
		case "23.40":
			backgroundStage = "defaultnotris"
			backgroundImage = "https://cdn2.unrealengine.com/mostwanted-final-v2-2048x2048-2048x2048-39f2b5041a40.jpg"
		default:
			backgroundStage = "season2300"
			backgroundImage = "https://cdn2.unrealengine.com/t-bp23-lobby-2048x1024-2048x1024-26f2c1b27f63.png"
		}
	case 24:
		if ua.Build == "24.30" {
			backgroundStage = "defaultnotris"
			backgroundImage = "https://cdn2.unrealengine.com/ch4s2-lobbyupdate-4-20-2022-lifted-copy-3840x2160-d3a138f5f9e7.jpg"
		} else {
			backgroundStage = "defaultnotris"
			backgroundImage = "https://cdn2.unrealengine.com/t-ch4s2-bp-lobby-4096x2048-edde08d15f7e.jpg"
		}
	case 25:
		if ua.Build == "25.11" {
			backgroundStage = "season2500"
			backgroundImage = "https://cdn2.unrealengine.com/t-s25-14dos-lobby-4096x2048-2be24969eee3.jpg"
		} else {
			backgroundStage = "season2500"
			backgroundImage = "https://cdn2.unrealengine.com/s25-lobby-4k-4096x2048-4a832928e11f.jpg"
		}
	case 26:
		if ua.Build == "26.30" {
			backgroundStage = "season2630"
			backgroundImage = "https://cdn2.unrealengine.com/s26-lobby-timemachine-final-2560x1440-a3ce0018e3fa.jpg"
		} else {
			backgroundStage = "season2600"
			backgroundImage = "https://cdn2.unrealengine.com/0814-ch4s4-lobby-2048x1024-2048x1024-e3c2cf8d342d.png"
		}
	case 27:
		if ua.Build == "27.11" {
			backgroundStage = "defaultnotris"
			backgroundImage = "https://cdn2.unrealengine.com/durianlobby2-4096x2048-242a51b6a8ee.jpg"
		} else {
			backgroundStage = "rufus"
		}
	case 28:
		switch ua.Build {
		case "28.01":
			backgroundStage = "defaultnotris"
			backgroundImage = "https://cdn2.unrealengine.com/winterfest2023-lobby-2048x1024-a8853c3a6f59.jpg"
		case "28.20":
			backgroundStage = "defaultnotris"
			backgroundImage = "https://cdn2.unrealengine.com/s28-tmnt-lobby-4096x2048-e6c06a310c05.jpg"
		default:
			backgroundImage = "https://cdn2.unrealengine.com/ch5s1-lobbybg-3640x2048-0974e0c3333c.jpg"
			backgroundStage = "defaultnotris"
		}
	case 29:
		switch ua.Build {
		case "29.20":
			backgroundStage = "season2920"
			backgroundImage = "https://cdn2.unrealengine.com/iceberg-lobby-3840x2160-217bb6ea8af9.jpg"
		case "29.40":
			backgroundStage = "defaultnotris"
			backgroundImage = "https://cdn2.unrealengine.com/mkart-2940-sw-fnbr-lobby-3840x2160-4f1f1486a54a.jpg"
		default:
			backgroundStage = "defaultnotris"
		}
	case 32:
		backgroundStage = "defaultnotris"
		backgroundImage = "https://cdn2.unrealengine.com/mkart-fnbr-quail-lobby-3264x1836-b157b2252db6.jpg"
	}

	shopSections, _ := odin.FindAll("ShopSections", func() interface{} {
		return &fortnite.ShopSections{}
	})

	mpItemShopSections, _ := odin.FindAll("MPItemShop", func() interface{} {
		return &fortnite.MpItemShop{}
	})

	filteredTournaments := []interface{}{}
	tournaments := []map[string]interface{}{
		{
			"_type":                  "Tournament Display Info",
			"title_color":            "",
			"loading_screen_image":   "https://saturn.nxa.app/Fortnite_fortnite-game_tournaments_12BR_Arena_Solo_ModeTile-1024x512-f0ecee555f69c65e8a0eace05372371bebcb050f.jpg",
			"background_text_color":  "",
			"background_right_color": "",
			"poster_back_image":      "https://saturn.nxa.app/Fortnite_fortnite-game_tournaments_12BR_Arena_Solo_ModeTile-1024x512-f0ecee555f69c65e8a0eace05372371bebcb050f.jpg",
			"type":                   "",
			"pin_score_requirement":  100,
			"pin_earned_text":        "",
			"tournament_display_id":  "remixarenasolo",
			"event_id":               "epicgames_Arena_S" + seasonStr + "_Solo",
			"highlight_color":        "",
			"schedule_info":          "",
			"primary_color":          "",
			"flavor_description":     "",
			"poster_front_image":     "https://saturn.nxa.app/Fortnite_fortnite-game_tournaments_12BR_Arena_Solo_ModeTile-1024x512-f0ecee555f69c65e8a0eace05372371bebcb050f.jpg",
			"short_format_title":     "",
			"title_line_2":           "",
			"title_line_1":           "Arena",
			"shadow_color":           "",
			"details_description":    "",
			"background_left_color":  "",
			"long_format_title":      "",
			"poster_fade_color":      "",
			"secondary_color":        "",
			"playlist_tile_image":    "https://saturn.nxa.app/Fortnite_fortnite-game_tournaments_12BR_Arena_Solo_ModeTile-1024x512-f0ecee555f69c65e8a0eace05372371bebcb050f.jpg",
			"base_color":             "",
		},
		{
			"_type":                  "Tournament Display Info",
			"title_color":            "",
			"loading_screen_image":   "https://saturn.nxa.app/CH4S1Data/Playlist_ShowdownAlt_Solo_LG.Png",
			"background_text_color":  "",
			"background_right_color": "",
			"poster_back_image":      "https://saturn.nxa.app/CH4S1Data/Playlist_ShowdownAlt_Solo_LG.Png",
			"type":                   "",
			"pin_score_requirement":  100,
			"pin_earned_text":        "",
			"tournament_display_id":  "remixarenalgduo",
			"event_id":               "epicgames_Arena_S" + seasonStr + "_Duos",
			"highlight_color":        "",
			"schedule_info":          "",
			"primary_color":          "",
			"flavor_description":     "",
			"poster_front_image":     "https://saturn.nxa.app/CH4S1Data/Playlist_ShowdownAlt_Solo_LG.Png",
			"short_format_title":     "",
			"title_line_2":           "Arena",
			"title_line_1":           "Lategame",
			"shadow_color":           "",
			"details_description":    "",
			"background_left_color":  "",
			"long_format_title":      "",
			"poster_fade_color":      "",
			"secondary_color":        "",
			"playlist_tile_image":    "https://saturn.nxa.app/CH4S1Data/Playlist_ShowdownAlt_Solo_LG.Png",
			"base_color":             "",
		},
	}

	for _, t := range tournaments {
		filteredTournaments = append(filteredTournaments, t)
	}

	filteredShopSections := []interface{}{}
	for _, s := range shopSections {
		shopSection, ok := s.(*fortnite.ShopSections)
		if !ok {
			continue
		}
		section := map[string]interface{}{
			"_type":                           "ShopSection",
			"bEnableToastNotification":        true,
			"bHidden":                         false,
			"bShowIneligibleOffers":           true,
			"bShowIneligibleOffersIfGiftable": true,
			"bShowTimer":                      true,
			"bSortOffersByOwnership":          false,
			"background": map[string]interface{}{
				"_type": "DynamicBackground",
				"key":   "vault",
				"stage": "default",
			},
			"landingPriority":    2,
			"sectionDisplayName": shopSection.Section,
			"sectionId":          shopSection.Section,
		}
		filteredShopSections = append(filteredShopSections, section)
	}

	filteredMpItemShopSections := []map[string]interface{}{}
	for _, section := range mpItemShopSections {
		s, ok := section.(*fortnite.MpItemShop)
		if !ok {
			continue
		}

		offerGroups := []map[string]interface{}{}
		for _, og := range s.Metadata.OfferGroups {
			stackRanks := []map[string]interface{}{}
			for _, sr := range og.StackRanks {
				stackRanks = append(stackRanks, map[string]interface{}{
					"stackRankValue": sr.StackRankValue,
					"_type":          sr.Type,
					"context":        sr.Context,
					"startDate":      sr.StartDate,
				})
			}

			offerGroups = append(offerGroups, map[string]interface{}{
				"bUseWidePreview": og.BUseWidePreview,
				"_type":           og.Type,
				"offerGroupId":    og.OfferGroupID,
				"stackRanks":      stackRanks,
			})
		}

		metadataStackRanks := []map[string]interface{}{}
		for _, sr := range s.Metadata.StackRanks {
			metadataStackRanks = append(metadataStackRanks, map[string]interface{}{
				"stackRankValue": sr.StackRankValue,
				"_type":          sr.Type,
				"context":        sr.Context,
				"startDate":      sr.StartDate,
			})
		}

		mpSection := map[string]interface{}{
			"metadata": map[string]interface{}{
				"offerGroups": offerGroups,
				"background": map[string]interface{}{
					"_type": s.Metadata.Background.Type,
				},
				"_type":                s.Metadata.Type,
				"showIneligibleOffers": s.Metadata.ShowIneligibleOffers,
				"stackRanks":           metadataStackRanks,
			},
			"displayName": s.DisplayName,
			"_type":       s.Type,
			"sectionID":   strings.ReplaceAll(s.SectionID, " ", ""),
			"category":    "Battle Royale",
		}
		filteredMpItemShopSections = append(filteredMpItemShopSections, mpSection)
	}

	playlistInfoRaw, _ := odin.FindAll("PlaylistInfo", func() interface{} {
		return &fortnite.PlaylistInfo{}
	})

	var playlistInfo []fortnite.PlaylistInfo
	for _, p := range playlistInfoRaw {
		if pi, ok := p.(*fortnite.PlaylistInfo); ok {
			playlistInfo = append(playlistInfo, *pi)
		}
	}

	filteredPlaylists := []interface{}{}
	for _, p := range playlistInfo {
		playlist := map[string]interface{}{
			"_type":          "FortPlaylistInfo",
			"image":          p.Image,
			"playlist_name":  p.PlaylistName,
			"special_border": p.SpecialBorder,
			"display_name":   p.DisplayName,
		}
		if p.Description != "" {
			playlist["description"] = p.Description
		}
		if p.Violator != "" {
			playlist["violator"] = p.Violator
		}
		if p.Hidden {
			playlist["hidden"] = p.Hidden
		}
		filteredPlaylists = append(filteredPlaylists, playlist)
	}

	contentPages := map[string]interface{}{
		"_title":       "Fortnite Game",
		"_activeDate":  "0001-01-01T00:00:00",
		"lastModified": "0001-01-01T00:00:00",
		"_locale":      "en-US",
		"battleroyalenews": map[string]interface{}{
			"_activeDate":  "0001-01-01T00:00:00",
			"_locale":      "en-US",
			"_title":       "battleroyalenews",
			"lastModified": "0001-01-01T00:00:00",
			"news": map[string]interface{}{
				"_type":       "Battle Royale News",
				"messages":    []map[string]interface{}{},
				"motds":       []map[string]interface{}{},
				"backgrounds": nil,
			},
		},
		"shopSections": map[string]interface{}{
			"_title": "shop-sections",
			"sectionList": map[string]interface{}{
				"_type":    "ShopSectionList",
				"sections": filteredShopSections,
			},
			"_noIndex":      false,
			"_activeDate":   "0001-12-01T21:00:00.000Z",
			"lastModified":  "0001-12-01T21:00:00.089Z",
			"_locale":       "en-US",
			"_templateName": "FortniteGameShopSections",
		},
		"mpItemShop": map[string]interface{}{
			"shopData": map[string]interface{}{
				"_type":    "MP Item Shop - Data Root",
				"sections": filteredMpItemShopSections,
			},
			"_title":        "mpItemShop",
			"_noIndex":      false,
			"_activeDate":   "2024-12-26T00:00:00.000Z",
			"lastModified":  "2024-12-26T00:00:00.000Z",
			"_locale":       "en-US",
			"_templateName": "FortniteGameMPItemShop",
		},
		"subgameinfo": map[string]interface{}{
			"_activeDate": "0001-01-01T00:00:00",
			"_locale":     "en-US",
			"_title":      "SubgameInfo",
			"battleroyale": map[string]interface{}{
				"_type":       "Subgame Info",
				"color":       "5b2569",
				"description": "100 Player PvP",
				"image":       "",
				"subgame":     "battleroyale",
				"title":       "Battle Royale",
			},
			"creative": map[string]interface{}{
				"_type":       "Subgame Info",
				"color":       "0658b9",
				"description": "Your Islands. Your Friends. Your Rules.",
				"image":       "",
				"subgame":     "creative",
				"title":       "Creative",
			},
			"lastModified": "0001-01-01T00:00:00",
			"savetheworld": map[string]interface{}{
				"_type":       "Subgame Info",
				"color":       "7615E9FF",
				"description": "Cooperative PvE Adventure",
				"image":       "",
				"subgame":     "savetheworld",
				"title":       "Save The World",
			},
		},
		"dynamicbackgrounds": map[string]interface{}{
			"backgrounds": map[string]interface{}{
				"backgrounds": []map[string]interface{}{
					{
						"stage":           backgroundStage,
						"backgroundimage": backgroundImage,
						"_type":           "DynamicBackground",
						"key":             "lobby",
					},
					{
						"stage": func() string {
							if ua.Build == "14.40" {
								return "Cyclone"
							}
							return backgroundStage
						}(),
						"_type": "DynamicBackground",
						"key":   "vault",
					},
				},
				"_type": "DynamicBackgroundList",
			},
			"_title":       "dynamicbackgrounds",
			"_noIndex":     false,
			"_activeDate":  "2019-08-21T15:59:59.342Z",
			"lastModified": "0001-01-01T00:00:00",
			"_locale":      "en-US",
		},
		"lobby": map[string]interface{}{
			"backgroundimage": backgroundImage,
			"stage":           "seasonx",
			"_title":          "lobby",
			"_activeDate":     "2019-08-21T15:59:59.342Z",
			"lastModified":    "0001-01-01T00:00:00",
			"_locale":         "en-US",
		},
		"playlistinformation": map[string]interface{}{
			"is_tile_hidden":                    true,
			"frontend_matchmaking_header_style": "Basic",
			"conversion_config": map[string]interface{}{
				"containerName":    "playlist_info",
				"_type":            "Conversion Config",
				"enableReferences": true,
				"contentName":      "playlists",
			},
			"show_ad_violator": false,
			"_title":           "playlistinformation",
			"playlist_info": map[string]interface{}{
				"_type":     "Playlist Information",
				"playlists": filteredPlaylists,
			},
			"_noIndex":     false,
			"_activeDate":  "0001-01-01T00:00:00",
			"lastModified": "0001-01-01T00:00:00",
			"_locale":      "en-US",
		},
		"tournamentinformation": map[string]interface{}{
			"conversion_config": map[string]interface{}{
				"containerName":    "tournament_info",
				"_type":            "Conversion Config",
				"enableReferences": true,
				"contentName":      "tournaments",
			},
			"tournament_info": map[string]interface{}{
				"tournaments": filteredTournaments,
				"_type":       "Tournaments Info",
			},
			"_title":       "tournamentinformation",
			"_noIndex":     false,
			"_activeDate":  "2020-01-01T00:00:00.000Z",
			"lastModified": "2024-01-01T00:00:00.000Z",
			"_locale":      "en-US",
		},
		"emergencynotice": map[string]interface{}{
			"news": map[string]interface{}{
				"_type":    "Battle Royale News",
				"messages": []map[string]interface{}{},
			},
			"_title":       "emergencynotice",
			"_noIndex":     false,
			"alwaysShow":   false,
			"_activeDate":  "2018-08-06T19:00:26.217Z",
			"lastModified": "2019-10-29T22:32:52.686Z",
			"_locale":      "en-US",
		},
		"scoringrulesinformation": map[string]interface{}{
			"scoring_rules_info": map[string]interface{}{
				"_type": "Scoring Rules Info",
				"scoring_rules": []map[string]interface{}{
					{
						"poster_description":             "Loot Island Captured",
						"rule_name":                      "MMO_LootIsland",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "/Game/UI/Foundation/Textures/Icons/Quest/T-Icon-Trophy-32.T-Icon-Trophy-32",
						"description":                    "Loot Island Captured",
						"hide_score_toast_notifications": false,
						"poster_interval_description":    "Loot Island Captured",
					},
					{
						"poster_description":             "{0}|plural(one=Each Forcast Tower Captured,other=Every {0} Forcast Tower Captures)",
						"rule_name":                      "EACH_MMO_RadioTower",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "/Game/UI/Foundation/Textures/Icons/Quest/T-Icon-Trophy-32.T-Icon-Trophy-32",
						"description":                    "{0}|plural(one=Each Forcast Tower Captured,other=Every {0} Forcast Tower Captures)",
						"hide_score_toast_notifications": false,
						"poster_interval_description":    "{0} - {1} Forcast Towers Captured",
					},
					{
						"poster_description":             "{0}|plural(one=Each Vault Captured,other=Every {0} Vault Captures)",
						"rule_name":                      "EACH_MMO_Vault",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "/Game/UI/Foundation/Textures/Icons/Quest/T-Icon-Trophy-32.T-Icon-Trophy-32",
						"description":                    "{0}|plural(one=Each Vault Captured,other=Every {0} Vault Captures)",
						"hide_score_toast_notifications": false,
						"poster_interval_description":    "{0} - {1} Vault Captured",
					},
					{
						"poster_description":             "{0}|plural(one=Each Cache Captured,other=Every {0} Cache Captures)",
						"rule_name":                      "EACH_MMO_Cache",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "/Game/UI/Foundation/Textures/Icons/Quest/T-Icon-Trophy-32.T-Icon-Trophy-32",
						"description":                    "{0}|plural(one=Each Cache Captured,other=Every {0} Cache Captures)",
						"hide_score_toast_notifications": false,
						"poster_interval_description":    "{0} - {1} Cache Captured",
					},
					{
						"poster_description":             "Victory Royale",
						"rule_name":                      "VICTORY_ROYALE_STAT",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "/Game/UI/Foundation/Textures/Icons/Quest/T-Icon-Trophy-32.T-Icon-Trophy-32",
						"description":                    "Victory Royale",
						"hide_score_toast_notifications": false,
						"poster_interval_description":    "Victory Royale",
					},
					{
						"poster_description":             "{0} {0}|plural(one=Elimination,other=Eliminations)",
						"rule_name":                      "Eliminations",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "/Game/Athena/HUD/Art/killsico_alt.KillsIco_Alt",
						"description":                    "{0} {0}|plural(one=Elimination,other=Eliminations)",
						"hide_score_toast_notifications": false,
						"poster_interval_description":    "{0} - {1} Eliminations",
					},
					{
						"poster_description":             "{0} {0}|plural(one=Elimination,other=Eliminations)",
						"rule_name":                      "TEAM_ELIMS_STAT_INDEX",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "/Game/Athena/HUD/Art/killsico_alt.KillsIco_Alt",
						"description":                    "{0} {0}|plural(one=Elimination,other=Eliminations)",
						"hide_score_toast_notifications": false,
						"poster_interval_description":    "{0} - {1} Eliminations",
					},
					{
						"poster_description":             "{0}{0}|ordinal(one=st,two=nd,few=rd,other=th) Place",
						"rule_name":                      "Placement",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "/Game/UI/Foundation/Textures/Icons/Quest/T-Icon-Trophy-32.T-Icon-Trophy-32",
						"description":                    "Reach Top {0}",
						"hide_score_toast_notifications": false,
						"poster_interval_description":    "{0}{0}|ordinal(one=st,two=nd,few=rd,other=th) - {1}{1}|ordinal(one=st,two=nd,few=rd,other=th) Place",
					},
					{
						"poster_description":             "{0}|plural(one=Each Elimination,other=Every {0} Eliminations)",
						"rule_name":                      "EachElimination",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "/Game/Athena/HUD/Art/killsico_alt.KillsIco_Alt",
						"description":                    "{0}|plural(one=Each Elimination,other=Every {0} Eliminations)",
						"hide_score_toast_notifications": false,
						"poster_interval_description":    "",
					},
					{
						"poster_description":             "{0}|plural(one=Each Elimination,other=Every {0} Eliminations)",
						"rule_name":                      "EACH_CREATIVE_ELIMINATIONS_STAT",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "/Game/Athena/HUD/Art/killsico_alt.KillsIco_Alt",
						"description":                    "{0}|plural(one=Each Elimination,other=Every {0} Eliminations)",
						"hide_score_toast_notifications": false,
						"poster_interval_description":    "{0} - {1} Eliminations",
					},
					{
						"poster_description":             "Bus Fare",
						"rule_name":                      "MatchEntryFee",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "/Game/UI/Events/Icons/T-Icon-Bus-Fare-Flat.T-Icon-Bus-Fare-Flat",
						"description":                    "Bus Fare",
						"hide_score_toast_notifications": false,
						"poster_interval_description":    nil,
					},
					{
						"poster_description":             "{0} {0}|plural(one=Bow Elimination,other=Bow Eliminations)",
						"rule_name":                      "BowKills",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "/Game/Athena/HUD/Art/killsico_alt.KillsIco_Alt",
						"description":                    "{0} {0}|plural(one=Bow Elimination,other=Bow Eliminations)",
						"hide_score_toast_notifications": false,
						"poster_interval_description":    "{0} - {1} Bow Eliminations",
					},
					{
						"poster_description":             "Each score of {0}",
						"rule_name":                      "EACH_CREATIVE_SCORE_STAT",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "/Game/UI/Foundation/Textures/Icons/Quest/T-Icon-Trophy-32.T-Icon-Trophy-32",
						"description":                    "Each score of {0}",
						"hide_score_toast_notifications": true,
						"poster_interval_description":    "",
					},
					{
						"poster_description":             "Score of {0}",
						"rule_name":                      "Scores",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "/Game/UI/Foundation/Textures/Icons/Quest/T-Icon-Trophy-32.T-Icon-Trophy-32",
						"description":                    "Score of {0}",
						"hide_score_toast_notifications": false,
						"poster_interval_description":    "Score of {0} - {1}",
					},
					{
						"poster_description":             "{0} {0}|plural(one=Time Eliminated,other=Times Eliminated)",
						"rule_name":                      "EACH_CREATIVE_ELIMINATED_STAT",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "/Game/Athena/HUD/Art/killsico_alt.KillsIco_Alt",
						"description":                    "{0} {0}|plural(one=Time Eliminated,other=Times Eliminated)",
						"hide_score_toast_notifications": false,
						"poster_interval_description":    "{0} - {1} Times Eliminated",
					},
					{
						"poster_description":             "{0}|plural(one=Each Damage Dealt,other=Every {0} Damage Dealt)",
						"rule_name":                      "CREATIVE_DAMAGE_DEALT_STAT",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "/Game/Athena/HUD/Art/icon_health.icon_health",
						"description":                    "{0}|plural(one=Each Damage Dealt,other=Every {0} Damage Dealt)",
						"hide_score_toast_notifications": true,
						"poster_interval_description":    nil,
					},
					{
						"poster_description":             "{0}|plural(one=Each Damage Taken,other=Every {0} Damage Taken)",
						"rule_name":                      "CREATIVE_DAMAGE_TAKEN_STAT",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "/Game/Athena/HUD/Art/icon_health.icon_health",
						"description":                    "{0}|plural(one=Each Damage Taken,other=Every {0} Damage Taken)",
						"hide_score_toast_notifications": false,
						"poster_interval_description":    nil,
					},
					{
						"poster_description":             "{0}|plural(one=Each Item Collected,other=Every {0} Items Collected)",
						"rule_name":                      "EACH_CREATIVE_COLLECT_ITEMS_STAT",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "/Game/UI/Foundation/Textures/Icons/Quest/T-Icon-Trophy-32.T-Icon-Trophy-32",
						"description":                    "{0}|plural(one=Each Item Collected,other=Every {0} Items Collected)",
						"hide_score_toast_notifications": false,
						"poster_interval_description":    "{0} - {1} Items Collected",
					},
					{
						"poster_description":             "Every {0} Remaining Spawn",
						"rule_name":                      "CREATIVE_SPAWNS_LEFT_STAT",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "/Game/UI/Foundation/Textures/Icons/Quest/T-Icon-Trophy-32.T-Icon-Trophy-32",
						"description":                    "Every {0} Remaining Spawn",
						"hide_score_toast_notifications": false,
						"poster_interval_description":    nil,
					},
					{
						"poster_description":             "Every {0} Health Remaining",
						"rule_name":                      "CREATIVE_REMAINING_HEALTH_STAT",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "/Game/Athena/HUD/Art/icon_health.icon_health",
						"description":                    "Every {0} Health Remaining",
						"hide_score_toast_notifications": false,
						"poster_interval_description":    nil,
					},
					{
						"poster_description":             "{0}|plural(one=Each Objective Completed,other=Every {0} Objectives Completed)",
						"rule_name":                      "EACH_CREATIVE_OBJECTIVES_STAT",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "/Game/UI/Foundation/Textures/Icons/Quest/T-Icon-Trophy-32.T-Icon-Trophy-32",
						"description":                    "{0}|plural(one=Each Objective Completed,other=Every {0} Objectives Completed)",
						"hide_score_toast_notifications": false,
						"poster_interval_description":    nil,
					},
					{
						"poster_description":             "{0}|plural(one=AI Elimination,other=Every {0} AI Eliminations)",
						"rule_name":                      "EACH_CREATIVE_AI_ELIMINATIONS_STAT",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "/Game/Athena/HUD/Art/killsico_alt.KillsIco_Alt",
						"description":                    "{0}|plural(one=AI Elimination,other=Every {0} AI Eliminations)",
						"hide_score_toast_notifications": false,
						"poster_interval_description":    "{0} - {1} AI Eliminated",
					},
					{
						"poster_description":             "{0}|plural(one=Each Assist,other=Assists)",
						"rule_name":                      "EACH_CREATIVE_ASSISTS_STAT",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "/Game/Athena/HUD/Art/killsico_alt.KillsIco_Alt",
						"description":                    "{0}|plural(one=Each Assist,other=Assists)",
						"hide_score_toast_notifications": false,
						"poster_interval_description":    "{0} - {1} Assists",
					},
					{
						"poster_description":             "{0}|plural(one=Each millisecond Alive,other=Every {0} Milliseconds Alive)",
						"rule_name":                      "EACH_CREATIVE_TIME_ALIVE_STAT",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "/Game/Athena/HUD/Art/icon_health.icon_health",
						"description":                    "{0}|plural(one=Each millisecond Alive,other=Every {0} Milliseconds Alive)",
						"hide_score_toast_notifications": true,
						"poster_interval_description":    nil,
					},
					{
						"poster_description":             "For every lap under {0} milliseconds",
						"rule_name":                      "CREATIVE_RACE_TIME_STAT",
						"_type":                          "Scoring Rules Display Info",
						"icon":                           "ns/Quest/T-Icon-Trophy-32.T-Icon-Trophy-32",
						"description":                    "For every lap under {0} milliseconds",
						"hide_score_toast_notifications": false,
						"poster_interval_description":    nil,
					},
				},
			},
			"_title":        "scoringrulesinformation",
			"_noIndex":      false,
			"_activeDate":   "2021-10-12T19:24:18.962Z",
			"lastModified":  "2023-11-16T19:22:34.542Z",
			"_locale":       "en-US",
			"_templateName": "FortniteGameScoringRulesInfo",
		},
		"emergencynoticev2": map[string]interface{}{
			"jcr_isCheckedOut": true,
			"_title":           "emergencynoticev2",
			"_noIndex":         false,
			"jcr_baseVersion":  "a7ca237317f1e7da533b38-74ee-468b-8c63-a7c3c256b313",
			"emergencynotices": map[string]interface{}{
				"_type": "Emergency Notices",
				"emergencynotices": []map[string]interface{}{
					{
						"hidden": false,
						"_type":  "CommonUI Emergency Notice Base",
						"title":  "Remix",
						"body":   "Made by Ploosh, Jeremy, and Andrew.\n\nhttps://discord.gg/remixfn",
					},
				},
			},
			"_activeDate":  "2018-08-06T19:00:26.217Z",
			"lastModified": "2021-06-22T08:27:47.969Z",
			"_locale":      "en-US",
		},
	}

	c.JSON(http.StatusOK, contentPages)
}

func FortniteMediaEventsV2Handler(c *gin.Context) {
	ua := utilities.Parse(c.GetHeader("User-Agent"))
	season := 32
	if ua != nil && ua.Season > 0 {
		season = ua.Season
	}
	seasonStr := strconv.Itoa(season)

	soloKey := "arenaS" + seasonStr + "Solo"
	duosKey := "arenaS" + seasonStr + "Duos"
	soloEventId := "epicgames_Arena_S" + seasonStr + "_Solo"
	duosEventId := "epicgames_Arena_S" + seasonStr + "_Duos"

	response := map[string]interface{}{
		"_title":             "media-events-v2",
		"_noIndex":           false,
		"_activeDate":        "2022-01-12T01:43:01.626Z",
		"lastModified":       "2023-06-30T21:59:54.608Z",
		"_locale":            "en-US",
		"_templateName":      "FortniteMediaEvents",
		"_suggestedPrefetch": []interface{}{},
	}
	response[soloKey] = map[string]interface{}{
		"_type":        "Fortnite - Media Event",
		"_title":       soloKey,
		"_noIndex":     false,
		"_activeDate":  "2020-01-01T00:00:00.000Z",
		"lastModified": "2024-01-01T00:00:00.000Z",
		"_locale":      "en-US",
		"eventId":      soloEventId,
		"event_id":     soloEventId,
	}
	response[duosKey] = map[string]interface{}{
		"_type":        "Fortnite - Media Event",
		"_title":       duosKey,
		"_noIndex":     false,
		"_activeDate":  "2020-01-01T00:00:00.000Z",
		"lastModified": "2024-01-01T00:00:00.000Z",
		"_locale":      "en-US",
		"eventId":      duosEventId,
		"event_id":     duosEventId,
	}

	c.JSON(http.StatusOK, response)
}

func GETMotdTarget(c *gin.Context) {
	response := map[string]interface{}{
		"contentType": "collection",
		"contentId":   "fortnite-br-br-motd-collection",
		"tcId":        "6e8026f1-406b-49d7-8a8f-142d74c992ee",
		"contentItems": []interface{}{
			map[string]interface{}{
				"contentHash":       "1bb45225d9f4615bb88b09c400128b70",
				"contentSchemaName": "DynamicMotd",
				"contentFields": map[string]interface{}{
					"Buttons": []interface{}{
						map[string]interface{}{
							"Action": map[string]interface{}{
								"_type": "MotdVideoAction",
								"video": map[string]interface{}{
									"Autoplay":         false,
									"ShouldLoop":       false,
									"StreamingEnabled": true,
									"UID":              "EUovmsrjZxCuBFkuVI",
									"VideoString":      "Stainless_Ringer_Emote",
									"_type":            "Video",
								},
							},
							"Style": "1",
							"Text":  "Watch Video",
							"_type": "Button",
						},
					},
					"FullScreenBackground": map[string]interface{}{
						"Image": []interface{}{
							map[string]interface{}{
								"width":  1920,
								"height": 1080,
								"url":    "https://cdn2.unrealengine.com/fnbr-32-00-c5s5-kiln-discoverplaylist-tiles-br-480x270-480x270-5ea86e9c4723.jpg",
							},
							map[string]interface{}{
								"width":  960,
								"height": 540,
								"url":    "https://cdn2.unrealengine.com/fnbr-32-00-c5s5-kiln-discoverplaylist-tiles-br-480x270-480x270-5ea86e9c4723.jpg",
							},
						},
						"_type": "FullScreenBackground",
					},
					"FullScreenBody":  "Made by Ploosh, Jeremy, and Andrew.",
					"FullScreenTitle": "Remix",
					"TeaserBackground": map[string]interface{}{
						"Image": []interface{}{
							map[string]interface{}{
								"width":  1024,
								"height": 512,
								"url":    "https://cdn2.unrealengine.com/fnbr-32-00-c5s5-kiln-discoverplaylist-tiles-br-480x270-480x270-5ea86e9c4723.jpg",
							},
						},
						"_type": "TeaserBackground",
					},
					"TeaserTitle":        "Remix",
					"VerticalTextLayout": false,
				},
				"placements": []interface{}{
					map[string]interface{}{
						"trackingId": "ebb179c8-35ad-4d63-a261-1d2c72dce03b",
						"tag":        "Product.BR.Build",
						"position":   0,
					},
					map[string]interface{}{
						"trackingId": "ebb179c8-35ad-4d63-a261-1d2c72dce03b",
						"tag":        "Product.BR.Build.Solo",
						"position":   0,
					},
					map[string]interface{}{
						"trackingId": "ebb179c8-35ad-4d63-a261-1d2c72dce03b",
						"tag":        "Product.Juno",
						"position":   0,
					},
					map[string]interface{}{
						"trackingId": "ebb179c8-35ad-4d63-a261-1d2c72dce03b",
						"tag":        "Product.FNE.Beanstalk",
						"position":   0,
					},
					map[string]interface{}{
						"trackingId": "ebb179c8-35ad-4d63-a261-1d2c72dce03b",
						"tag":        "Product.Sparks.PilgrimQuickplay",
						"position":   0,
					},
					map[string]interface{}{
						"trackingId": "ebb179c8-35ad-4d63-a261-1d2c72dce03b",
						"tag":        "Product.FNE.Unknown",
						"position":   0,
					},
				},
			},
		},
	}

	c.JSON(http.StatusOK, response)
}
