package managers

import (
	"fmt"
	"time"

	"github.com/andr1ww/odin"
	"github.com/andr1ww/odin/bucket"
	"github.com/remixfn/xenon/classes/mcp"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
)

func CreateProfile(profileType, accountId string, displayName string) (*accounts.Profile, error) {
	items := make(map[string]interface{})
	stats := make(map[string]interface{})

	switch profileType {
	case mcp.Athena:
		items, stats = createAthenaData()
	case mcp.CommonCore:
		items, stats = createCommonCoreData()
	case mcp.Creative:
		items, stats = createCreativeData(displayName)
	}

	newProfile := accounts.Profile{
		Bucket:    odin.Bucket{ID: accountId + ":" + profileType},
		AccountID: accountId,
		ProfileID: profileType,
		Revision:  0,
		Items:     items,
		Stats:     stats,
	}

	if err := bucket.CreateInDatabase("xenon_profiles", &newProfile); err != nil {
		return nil, fmt.Errorf("failed to create profile: %v", err)
	}

	return &newProfile, nil
}

func createAthenaData() (map[string]interface{}, map[string]interface{}) {
	items := make(map[string]interface{})
	stats := make(map[string]interface{})

	items["AthenaPickaxe:DefaultPickaxe"] = createItemData("AthenaPickaxe:DefaultPickaxe", nil, 1)
	items["AthenaGlider:DefaultGlider"] = createItemData("AthenaGlider:DefaultGlider", nil, 1)
	items["AthenaDance:EID_DanceMoves"] = createItemData("AthenaDance:EID_DanceMoves", nil, 1)
	items["loadout1"] = createLoadoutData()

	stats["use_random_loadout"] = false
	stats["past_seasons"] = []interface{}{}
	stats["season_match_boost"] = 0
	stats["loadouts"] = []string{"loadout1"}
	stats["mfa_reward_claimed"] = false
	stats["rested_xp_overflow"] = 0
	stats["current_mtx_platform"] = "Epic"
	stats["last_xp_interaction"] = time.Now().UTC().Format(time.RFC3339)
	stats["quest_manager"] = map[string]interface{}{
		"dailyLoginInterval": time.Time{}.Format(time.RFC3339),
		"dailyQuestRerolls":  1,
	}
	stats["season_num"] = 0
	stats["party_assist_quest"] = ""
	stats["battlestars"] = 0
	stats["battlestars_season_total"] = 0
	stats["rested_xp"] = 2500
	stats["rested_xp_mult"] = 4.0
	stats["accountLevel"] = 1
	stats["style_points"] = 0
	stats["alien_style_points"] = 0
	stats["active_loadout_index"] = 0
	stats["favorite_character"] = ""
	stats["favorite_backpack"] = ""
	stats["favorite_pickaxe"] = "AthenaPickaxe:DefaultPickaxe"
	stats["favorite_glider"] = "AthenaGlider:DefaultGlider"
	stats["favorite_skydivecontrail"] = ""
	stats["favorite_dance"] = []string{"", "", "", "", "", ""}
	stats["favorite_itemwraps"] = []string{"", "", "", "", "", "", ""}
	stats["favorite_loadingscreen"] = ""
	stats["favorite_musicpack"] = ""
	stats["banner_icon"] = "StandardBanner31"
	stats["banner_color"] = "DefaultColor1"

	return items, stats
}

func createCreativeData(displayName string) (map[string]interface{}, map[string]interface{}) {
	items := make(map[string]interface{})
	stats := make(map[string]interface{})

	stats["max_creative_plots"] = 30
	stats["support_code"] = displayName
	stats["creator_name"] = displayName

	return items, stats
}

func createCommonCoreData() (map[string]interface{}, map[string]interface{}) {
	items := make(map[string]interface{})
	stats := make(map[string]interface{})

	items["Currency:MtxPurchased"] = createCurrencyData()

	stats["survey_data"] = map[string]interface{}{}
	stats["personal_offers"] = map[string]interface{}{}
	stats["intro_game_played"] = true
	stats["import_friends_claimed"] = map[string]interface{}{}
	stats["mtx_purchase_history"] = map[string]interface{}{
		"purchases":     []interface{}{},
		"refundCredits": 3,
		"refundsUsed":   0,
	}
	stats["undo_cooldowns"] = []interface{}{}
	stats["mtx_affiliate_set_time"] = "None"
	stats["inventory_limit_bonus"] = 0
	stats["current_mtx_platform"] = "EpicPC"
	stats["mtx_affiliate"] = "None"
	stats["weekly_purchases"] = map[string]interface{}{}
	stats["daily_purchases"] = map[string]interface{}{}
	stats["ban_history"] = map[string]interface{}{}
	stats["in_app_purchases"] = map[string]interface{}{}
	stats["permissions"] = []interface{}{}
	stats["mfa_enabled"] = true
	stats["allowed_to_send_gifts"] = true
	stats["allowed_to_receive_gifts"] = true
	stats["gift_history"] = map[string]interface{}{}
	stats["banner_icon"] = "StandardBanner31"
	stats["banner_color"] = "DefaultColor1"
	stats["homebase_name"] = ""

	for i := 1; i <= 21; i++ {
		templateId := fmt.Sprintf("HomebaseBannerColor:DefaultColor%d", i)
		items[templateId] = createItemData(templateId, map[string]interface{}{"item_seen": false}, 1)
	}

	for i := 1; i <= 31; i++ {
		templateId := fmt.Sprintf("HomebaseBannerIcon:StandardBanner%d", i)
		items[templateId] = createItemData(templateId, map[string]interface{}{"item_seen": false}, 1)
	}

	return items, stats
}

func createItemData(templateId string, value interface{}, quantity int) mcp.BaseItem {
	if value == nil {
		value = mcp.BaseItemAttributes{
			XP:       0,
			Level:    1,
			Variants: []mcp.Variants{},
			ItemSeen: false,
		}
	}

	if templateId == "Currency:MtxPurchased" {
		quantity = 0
	}

	return mcp.BaseItem{
		Attributes: value,
		TemplateId: templateId,
		Quantity:   quantity,
	}
}

func createLoadoutData() mcp.BaseItem {
	loadoutData := map[string]interface{}{
		"favorite":    false,
		"item_seen":   false,
		"use_count":   0,
		"locker_name": "",
		"locker_slots_data": map[string]interface{}{
			"slots": map[string]interface{}{
				"Dance": map[string]interface{}{
					"items": []string{"AthenaDance:EID_DanceMoves", "", "", "", "", ""},
				},
				"Glider": map[string]interface{}{
					"items": []string{"AthenaGlider:DefaultGlider"},
				},
				"Pickaxe": map[string]interface{}{
					"items":          []string{"AthenaPickaxe:DefaultPickaxe"},
					"activeVariants": []interface{}{},
				},
				"Character": map[string]interface{}{
					"items": []string{"AthenaCharacter:CID_001_Athena_Commando_F_Default"},
					"activeVariants": []map[string]interface{}{
						{"variants": []interface{}{}},
					},
				},
				"Backpack": map[string]interface{}{
					"items": []string{""},
					"activeVariants": []map[string]interface{}{
						{"variants": []interface{}{}},
					},
				},
				"ItemWrap": map[string]interface{}{
					"items":          []string{"", "", "", "", "", "", ""},
					"activeVariants": []interface{}{nil, nil, nil, nil, nil, nil, nil},
				},
				"MusicPack": map[string]interface{}{
					"items":          []string{""},
					"activeVariants": []interface{}{nil},
				},
				"LoadingScreen": map[string]interface{}{
					"items":          []string{""},
					"activeVariants": []interface{}{nil},
				},
				"SkyDiveContrail": map[string]interface{}{
					"items":          []string{""},
					"activeVariants": []interface{}{nil},
				},
			},
		},
		"banner_icon_template":  "",
		"banner_color_template": "",
	}

	return mcp.BaseItem{
		Attributes: loadoutData,
		TemplateId: "CosmeticLocker:cosmeticlocker_athena",
		Quantity:   1,
	}
}

func createCurrencyData() mcp.BaseItem {
	value := map[string]interface{}{
		"platform": "EpicPC",
		"level":    1,
	}

	return mcp.BaseItem{
		Attributes: value,
		TemplateId: "Currency:MtxPurchased",
		Quantity:   0,
	}
}

func InitLocationQuest(items map[string]interface{}) map[string]interface{} {
	items["Quest:quest_s11_discover_landmarks"] = map[string]interface{}{
		"templateId": "Quest:quest_s11_discover_landmarks",
		"attributes": map[string]interface{}{
			"creation_time":                 "2018-04-30T00:00:00.000Z",
			"level":                         -1,
			"item_seen":                     true,
			"playlists":                     []interface{}{},
			"sent_new_notification":         true,
			"challenge_bundle_id":           "",
			"xp_reward_scalar":              1,
			"challenge_linked_quest_given":  "",
			"quest_pool":                    "",
			"quest_state":                   "Active",
			"bucket":                        "",
			"last_state_change_time":        "2018-04-30T00:00:00.000Z",
			"challenge_linked_quest_parent": "",
			"max_level_bonus":               0,
			"xp":                            0,
			"quest_rarity":                  "uncommon",
			"favorite":                      false,
			"completion_visit_landmark_crashedairplane":                 1,
			"completion_visit_landmark_angryapples":                     1,
			"completion_visit_landmark_campcod":                         1,
			"completion_visit_landmark_coralcove":                       1,
			"completion_visit_landmark_lighthouse":                      1,
			"completion_visit_landmark_fortruin":                        1,
			"completion_visit_landmark_beachsidemansion":                1,
			"completion_visit_landmark_digsite":                         1,
			"completion_visit_landmark_bonfirecampsite":                 1,
			"completion_visit_landmark_riskyreels":                      1,
			"completion_visit_landmark_radiostation":                    1,
			"completion_visit_landmark_scrapyard":                       1,
			"completion_visit_landmark_powerdam":                        1,
			"completion_visit_landmark_weatherstation":                  1,
			"completion_visit_landmark_islandlodge":                     1,
			"completion_visit_landmark_waterfallgorge":                  1,
			"completion_visit_landmark_canoerentals":                    1,
			"completion_visit_landmark_mountainvault":                   1,
			"completion_visit_landmark_swampville":                      1,
			"completion_visit_landmark_cliffsideruinedhouses":           1,
			"completion_visit_landmark_sawmill":                         1,
			"completion_visit_landmark_pipeplayground":                  1,
			"completion_visit_landmark_pipeperson":                      1,
			"completion_visit_landmark_hayhillbilly":                    1,
			"completion_visit_landmark_lawnmowerraces":                  1,
			"completion_visit_landmark_shipwreckcove":                   1,
			"completion_visit_landmark_tallestmountain":                 1,
			"completion_visit_landmark_beachbus":                        1,
			"completion_visit_landmark_bobsbluff":                       1,
			"completion_visit_landmark_buoyboat":                        1,
			"completion_visit_landmark_chair":                           1,
			"completion_visit_landmark_forkknifetruck":                  1,
			"completion_visit_landmark_durrrburgertruck":                1,
			"completion_visit_landmark_snowconetruck":                   1,
			"completion_visit_landmark_pizzapetetruck":                  1,
			"completion_visit_landmark_beachrentals":                    1,
			"completion_visit_landmark_overgrownheads":                  1,
			"completion_visit_landmark_loverslookout":                   1,
			"completion_visit_landmark_toiletthrone":                    1,
			"completion_visit_landmark_hatch_a":                         1,
			"completion_visit_landmark_hatch_b":                         1,
			"completion_visit_landmark_hatch_c":                         1,
			"completion_visit_landmark_bigbridgeblue":                   1,
			"completion_visit_landmark_bigbridgered":                    1,
			"completion_visit_landmark_bigbridgeyellow":                 1,
			"completion_visit_landmark_bigbridgegreen":                  1,
			"completion_visit_landmark_bigbridgepurple":                 1,
			"completion_visit_landmark_captaincarpstruck":               1,
			"completion_visit_landmark_mountf8":                         1,
			"completion_visit_landmark_mounth7":                         1,
			"completion_visit_landmark_basecamphotel":                   1,
			"completion_visit_landmark_basecampfoxtrot":                 1,
			"completion_visit_landmark_basecampgolf":                    1,
			"completion_visit_landmark_lazylakeisland":                  1,
			"completion_visit_landmark_boatlaunch":                      1,
			"completion_visit_landmark_crashedcargo":                    1,
			"completion_visit_landmark_homelyhills":                     1,
			"completion_visit_landmark_flopperpond":                     1,
			"completion_visit_landmark_hilltophouse":                    1,
			"completion_visit_landmark_stackshack":                      1,
			"completion_visit_landmark_unremarkableshack":               1,
			"completion_visit_landmark_rapidsrest":                      1,
			"completion_visit_landmark_stumpyridge":                     1,
			"completion_visit_landmark_corruptedisland":                 1,
			"completion_visit_landmark_bushface":                        1,
			"completion_visit_landmark_mountaindanceclub":               1,
			"completion_visit_landmark_icechair":                        1,
			"completion_visit_landmark_icehotel":                        1,
			"completion_visit_landmark_toyfactory":                      1,
			"completion_visit_landmark_iceblockfactory":                 1,
			"completion_visit_landmark_crackshotscabin":                 1,
			"completion_visit_landmark_agencyhq":                        1,
			"completion_visit_landmark_spybase_undergroundsoccerfield":  1,
			"completion_visit_landmark_spybase_undergroundgasstation":   1,
			"completion_visit_landmark_suburban_abandonedhouse":         1,
			"completion_visit_landmark_remotehouse":                     1,
			"completion_visit_landmark_fishfarm":                        1,
			"completion_visit_landmark_sirenisland":                     1,
			"completion_visit_landmark_boatsales":                       1,
			"completion_visit_landmark_boatrace":                        1,
			"completion_visit_landmark_theyacht2":                       1,
			"completion_visit_landmark_toilettitan":                     1,
			"completion_visit_landmark_highhoop":                        1,
			"completion_visit_landmark_pizzapitbarge":                   1,
			"completion_visit_landmark_trophy":                          1,
			"completion_visit_namedpoi_spybase_oilrig":                  1,
			"completion_visit_namedpoi_spybase_yacht":                   1,
			"completion_visit_namedpoi_spybase_mountainbase":            1,
			"completion_visit_namedpoi_spybase_shark":                   1,
			"completion_visit_landmark_spybase_undergroundradiostation": 1,
			"completion_visit_landmark_spybase_boxfactory":              1,
			"completion_visit_landmark_spybase_teddybearspies":          1,
			"completion_visit_landmark_spybase_recruitmentofficeego":    1,
			"completion_visit_landmark_spybase_recruitmentofficealter":  1,
			"completion_visit_landmark_spybase_spyobstaclecourse":       1,
			"completion_visit_landmark_sharkremains":                    1,
			"completion_visit_landmark_restaurantgas":                   1,
			"completion_visit_landmark_carman":                          1,
			"completion_visit_landmark_spybase_grottoruins":             1,
			"completion_visit_landmark_sentinelgraveyard":               1,
			"completion_visit_landmark_tantorsphere":                    1,
			"completion_visit_landmark_bigdoghouse":                     1,
			"completion_visit_landmark_onyxsphere":                      1,
			"completion_visit_landmark_dateroom":                        1,
			"completion_visit_landmark_awakestatue":                     1,
			"completion_visit_landmark_datehouse":                       1,
			"completion_visit_landmark_throne":                          1,
			"completion_visit_landmark_lawoffice":                       1,
			"completion_visit_landmark_ruin":                            1,
			"completion_visit_landmark_friendmonument":                  1,
			"completion_visit_landmark_workshop":                        1,
			"completion_visit_landmark_heroespark":                      1,
			"completion_visit_landmark_turbo":                           1,
			"completion_visit_landmark_jetlanding_01":                   1,
			"completion_visit_landmark_jetlanding_02":                   1,
			"completion_visit_landmark_jetlanding_04":                   1,
			"completion_visit_landmark_jetlanding_05":                   1,
			"completion_visit_landmark_jetlanding_06":                   1,
			"completion_visit_landmark_jetlanding_07":                   1,
			"completion_visit_landmark_jetlanding_08":                   1,
			"completion_visit_landmark_jetlanding_09":                   1,
			"completion_visit_landmark_jetlanding_10":                   1,
			"completion_visit_landmark_jetlanding_11":                   1,
			"completion_visit_landmark_jetlanding_12":                   1,
			"completion_visit_landmark_jetlanding_13":                   1,
			"completion_visit_landmark_jetlanding_14":                   1,
			"completion_visit_landmark_jetlanding_15":                   1,
			"completion_visit_landmark_jetlanding_16":                   1,
			"completion_visit_landmark_jetlanding_17":                   1,
			"completion_visit_landmark_bstore":                          1,
			"completion_visit_landmark_cabin":                           1,
			"completion_visit_landmark_flushfactory":                    1,
			"completion_visit_landmark_steelfarm":                       1,
			"completion_visit_landmark_tomatotown":                      1,
			"completion_visit_landmark_greasygrove":                     1,
			"completion_visit_landmark_vikingvillage":                   1,
			"completion_visit_landmark_sheriffoffice":                   1,
			"completion_visit_landmark_cosmoscrashsite":                 1,
			"completion_visit_landmark_dustydepot":                      1,
			"completion_visit_landmark_butterbarn":                      1,
			"completion_visit_landmark_zpoint":                          1,
			"completion_visit_landmark_noahhouse":                       1,
			"completion_visit_landmark_kitskantina":                     1,
			"completion_visit_landmark_iohub_d1":                        1,
			"completion_visit_landmark_iohub_e6":                        1,
			"completion_visit_landmark_iohub_f4":                        1,
			"completion_visit_landmark_crashedhelicopter":               1,
			"completion_visit_landmark_arenafloor4":                     1,
			"completion_visit_landmark_arenafloor3":                     1,
			"completion_visit_landmark_arenafloor1":                     1,
			"completion_visit_landmark_arenafloor5":                     1,
			"completion_visit_landmark_holidaystore":                    1,
			"completion_visit_landmark_outpost_alpha":                   1,
			"completion_visit_landmark_outpost_beta":                    1,
			"completion_visit_landmark_outpost_charlie":                 1,
			"completion_visit_landmark_outpost_delta":                   1,
			"completion_visit_landmark_outpost_echo":                    1,
			"completion_visit_landmark_outpost_outsidetower1":           1,
			"completion_visit_landmark_outpost_outsidetower2":           1,
			"completion_visit_landmark_outpost_outsidetower3":           1,
			"completion_visit_landmark_outpost_outsidetower4":           1,
			"completion_visit_landmark_outpost_outsidetower5":           1,
			"completion_visit_landmark_outpost_outsidetower6":           1,
			"completion_visit_landmark_outpost_primalpond":              1,
			"completion_visit_landmark_outpost_rockface":                1,
			"completion_visit_landmark_outpost_cattycornergarage":       1,
			"completion_visit_landmark_outpost_goldenisland":            1,
			"completion_visit_landmark_radardish_01":                    1,
			"completion_visit_landmark_radardish_02":                    1,
			"completion_visit_landmark_radardish_03":                    1,
			"completion_visit_landmark_radardish_04":                    1,
			"completion_visit_landmark_radardish_05":                    1,
			"completion_visit_landmark_radardish_06":                    1,
			"completion_visit_landmark_radardish_07":                    1,
			"completion_visit_landmark_towerruins":                      1,
			"completion_visit_landmark_hiddenufo_01":                    1,
			"completion_visit_landmark_hiddenufo_02":                    1,
			"completion_visit_landmark_hiddenufo_03":                    1,
			"completion_visit_landmark_hiddenufo_04":                    1,
			"completion_visit_landmark_hiddenufo_05":                    1,
			"completion_visit_landmark_crashsite_01":                    1,
			"completion_visit_landmark_crashsite_02":                    1,
			"completion_visit_landmark_crashsite_03":                    1,
			"completion_visit_landmark_crashsite_04":                    1,
			"completion_visit_landmark_crashsite_05":                    1,
			"completion_visit_landmark_friendlycube":                    1,
			"completion_visit_landmark_outpost_01":                      1,
			"completion_visit_landmark_outpost_02":                      1,
			"completion_visit_landmark_outpost_03":                      1,
			"completion_visit_landmark_outpost_04":                      1,
			"completion_visit_landmark_outpost_05":                      1,
			"completion_visit_landmark_convoy_01":                       1,
			"completion_visit_landmark_convoy_02":                       1,
			"completion_visit_landmark_convoy_03":                       1,
			"completion_visit_landmark_convoy_04":                       1,
			"completion_visit_landmark_convoy_05":                       1,
			"completion_visit_landmark_convoy_06":                       1,
			"completion_visit_landmark_convoy_07":                       1,
			"completion_visit_landmark_convoy_08":                       1,
			"completion_visit_landmark_convoy_09":                       1,
			"completion_visit_landmark_convoy_10":                       1,
			"completion_visit_namedpoi_glasscases":                      1,
			"completion_visit_namedpoi_tomatosphere":                    1,
			"completion_visit_namedpoi_tomatowater":                     1,
			"completion_visit_namedpoi_tomatohouse":                     1,
			"completion_visit_namedpoi_riskyreels":                      1,
			"completion_visit_namedpoi_spybase_oilrig_reset_0":          1,
			"completion_visit_landmark_militarycamp_01":                 1,
			"completion_visit_landmark_militarycamp_02":                 1,
			"completion_visit_landmark_militarycamp_03":                 1,
			"completion_visit_landmark_militarycamp_04":                 1,
			"completion_visit_landmark_militarycamp_05":                 1,
			"completion_visit_landmark_galileosite1":                    1,
			"completion_visit_landmark_galileosite2":                    1,
			"completion_visit_landmark_galileosite3":                    1,
			"completion_visit_landmark_galileosite4":                    1,
			"completion_visit_landmark_galileosite5":                    1,
			"completion_visit_landmark_pirateradioboat":                 1,
			"completion_visit_landmark_arkbarge":                        1,
			"completion_visit_landmark_fishrestaurantbarge1":            1,
			"completion_visit_landmark_fishrestaurantbarge2":            1,
			"completion_visit_landmark_wagontrail":                      1,
			"completion_visit_landmark_pawnbarge":                       1,
			"completion_visit_landmark_partybarge":                      1,
			"completion_visit_landmark_piratebarge":                     1,
			"completion_visit_landmark_floodedweepingwoods":             1,
			"completion_visit_landmark_spybase_floodeddirtydocks":       1,
			"completion_visit_landmark_spybase_floodedcraggycliffs":     1,
			"completion_visit_landmark_witch1":                          1,
			"completion_visit_landmark_witch2":                          1,
			"completion_visit_landmark_witch3":                          1,
			"completion_visit_landmark_witch4":                          1,
			"completion_visit_landmark_witch5":                          1,
			"completion_visit_landmark_witch6":                          1,
			"completion_visit_landmark_witch7":                          1,
			"completion_visit_landmark_booshop":                         1,
			"completion_visit_papaya_theater":                           1,
			"completion_visit_papaya_thehub":                            1,
			"completion_visit_papaya_mainstage":                         1,
			"completion_visit_papaya_soccerfield":                       1,
			"completion_visit_papaya_fishingpond":                       1,
			"completion_visit_papaya_secretbeach":                       1,
			"completion_visit_papaya_giantskeleton":                     1,
			"completion_visit_papaya_boatramps":                         1,
			"completion_visit_papaya_piratecove":                        1,
			"completion_visit_papaya_boatraceeast":                      1,
			"completion_visit_papaya_boatracesouth":                     1,
			"completion_visit_papaya_gliderdrop":                        1,
			"completion_visit_papaya_obstaclecourse":                    1,
			"completion_visit_papaya_racecenter":                        1,
			"completion_visit_papaya_mountainpeak":                      1,
			"completion_visit_boogieboat":                               1,
		},
		"quantity": 1,
	}
	items["Quest:quest_s11_discover_namedlocations"] = map[string]interface{}{
		"templateId": "Quest:quest_s11_discover_namedlocations",
		"attributes": map[string]interface{}{
			"creation_time":                          "2018-04-30T00:00:00.000Z",
			"level":                                  -1,
			"item_seen":                              true,
			"playlists":                              []interface{}{},
			"sent_new_notification":                  true,
			"challenge_bundle_id":                    "",
			"xp_reward_scalar":                       1,
			"challenge_linked_quest_given":           "",
			"quest_pool":                             "",
			"quest_state":                            "Active",
			"bucket":                                 "",
			"last_state_change_time":                 "2018-04-30T00:00:00.000Z",
			"challenge_linked_quest_parent":          "",
			"max_level_bonus":                        0,
			"xp":                                     0,
			"quest_rarity":                           "uncommon",
			"favorite":                               false,
			"completion_visit_location_beachybluffs": 1,
			"completion_visit_location_dirtydocks":   1,
			"completion_visit_location_frenzyfarm":   1,
			"completion_visit_location_hollyhedges":  1,
			"completion_visit_location_hollyhedgesupdate": 1,
			"completion_visit_location_lazylake":          1,
			"completion_visit_location_mountainmeadow":    1,
			"completion_visit_location_powerplant":        1,
			"completion_visit_location_slurpyswamp":       1,
			"completion_visit_location_sunnyshores":       1,
			"completion_visit_location_weepingwoods":      1,
			"completion_visit_location_retailrow":         1,
			"completion_visit_location_saltysprings":      1,
			"completion_visit_location_pleasantpark":      1,
			"completion_visit_location_fortilla":          1,
			"completion_visit_location_oilrigislands":     1,
			"completion_visit_location_shadowagency":      1,
			"completion_visit_location_cattycorner":       1,
			"completion_visit_location_carl":              1,
			"completion_visit_location_tomatolab":         1,
			"completion_visit_location_nightmarejungle":   1,
			"completion_visit_location_thecoliseum":       1,
			"completion_visit_location_huntershaven":      1,
			"completion_visit_location_saltytowers":       1,
			"completion_visit_location_heroesharvest":     1,
			"completion_visit_location_maintower":         1,
			"completion_visit_location_governmentcomplex": 1,
			"completion_visit_location_theconvergence":    1,
		},
		"quantity": 1,
	}

	return items
}
