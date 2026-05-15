package remix_server_fortnite

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/modules/database/buckets/fortnite_tournaments"
	"github.com/remixfn/xenon/modules/synapse"
	fortnite_mcp "github.com/remixfn/xenon/router/fortnite/mcp"
	"github.com/remixfn/xenon/utilities"
)

func POSTBroadcastMatchResults(c *gin.Context) {
	var body struct {
		FortPlayerControllerAthena struct {
			XPComponent struct {
				TotalXpEarned int `json:"TotalXpEarned"`
			} `json:"XPComponent"`
			MatchReport struct {
				Place        int `json:"Place"`
				TotalPlayers int `json:"TotalPlayers"`
			} `json:"MatchReport"`
		} `json:"FortPlayerControllerAthena"`
		FortPlayerStateAthena struct {
			KillScore     int `json:"KillScore"`
			TeamKillScore int `json:"TeamKillScore"`
			DeathInfo     struct {
				DeathCause    int `json:"DeathCause"`
				DeathLocation struct {
					X float64 `json:"X"`
					Y float64 `json:"Y"`
					Z float64 `json:"Z"`
				} `json:"DeathLocation"`
			} `json:"DeathInfo"`
		} `json:"FortPlayerStateAthena"`
		PlaylistData struct {
			PlaylistName  string `json:"PlaylistName"`
			TournamentId  string `json:"TournamentId"`
			EventWindowId string `json:"EventWindowId"`
		} `json:"PlaylistData"`
		FortGameModeAthena struct {
			BSafeZonePaused bool `json:"bSafeZonePaused"`
		} `json:"FortGameModeAthena"`
		FortGameStateAthena struct {
			GamePhase string `json:"GamePhase"`
		} `json:"FortGameStateAthena"`
		Remix struct {
			StartingPlayers        int    `json:"StartingPlayers"`
			AccountID              string `json:"AccountId"`
			BroadcastQuestProgress []struct {
				BackendName string `json:"BackendName"`
				Count       int    `json:"Count"`
			} `json:"BroadcastQuestProgress"`
		} `json:"Remix"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		utilities.Internal.JsonParsingFailed().Apply(c.Writer)
		return
	}

	sm := synapse.GetStartedInstance()
	if sm == nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	log.Printf("body: %+v", body)

	var account accounts.Account
	if err := odin.Find("Accounts", body.Remix.AccountID, &account); err != nil {
		utilities.Internal.DataBaseError().Apply(c.Writer)
		return
	}

	athenaKey := fmt.Sprintf("%s:athena", account.ID)
	var athena accounts.Profile
	if err := odin.Find("Accounts_Profiles", athenaKey, &athena); err != nil {
		utilities.MCP.ProfileNotFound().WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	commonCoreKey := fmt.Sprintf("%s:common_core", account.ID)
	var commonCore accounts.Profile
	if err := odin.Find("Accounts_Profiles", commonCoreKey, &commonCore); err != nil {
		utilities.MCP.ProfileNotFound().WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	var season accounts.Season
	seasonKey := fmt.Sprintf("%s:%v", athena.AccountID, utilities.GetConfig().CURRENT_VERSION)
	if err := odin.Find("Accounts_Seasons", seasonKey, &season); err != nil {
		season = accounts.Season{
			Bucket:       odin.Bucket{ID: seasonKey},
			Level:        1,
			Xp:           0,
			BookLevel:    1,
			BookXp:       0,
			PurchasedVip: false,
			AllXpGained:  0,
		}
	}

	if athena.Stats == nil {
		athena.Stats = make(map[string]interface{})
	}

	athena.Stats["level"] = float64(season.Level)
	athena.Stats["xp"] = float64(season.Xp)
	athena.Stats["book_level"] = season.BookLevel
	athena.Stats["book_xp"] = season.BookXp
	athena.Stats["book_purchased"] = season.PurchasedVip

	if _, exists := athena.Stats["battlestars"]; !exists {
		athena.Stats["battlestars"] = float64(0)
	}
	if _, exists := athena.Stats["purchased_bp_offers"]; !exists {
		athena.Stats["purchased_bp_offers"] = []interface{}{}
	}

	vbRaw, exists := commonCore.Items["Currency:MtxPurchased"]
	if !exists {
		utilities.MCP.ProfileNotFound().Apply(c.Writer)
		return
	}

	vbucks, ok := vbRaw.(map[string]any)
	if !ok {
		utilities.Internal.DataBaseError().Apply(c.Writer)
		return
	}

	if qtyFloat, exists := vbucks["quantity"].(float64); exists {
		qty := int(qtyFloat)
		qty += int(body.FortPlayerStateAthena.KillScore * 75)

		if body.FortPlayerControllerAthena.MatchReport.Place == 1 {
			qty += int(250)
			reward := "AthenaGlider:Umbrella_Season_" + strconv.Itoa(utilities.GetConfig().CURRENT_VERSION)
			umbrella := "AthenaGlider:Solo_Umbrella"
			_, rewardExists := athena.Items[reward]
			_, umbrellaExists := athena.Items[umbrella]

			if !rewardExists {
				athena.Items[reward] = map[string]interface{}{
					"templateId": reward,
					"attributes": map[string]interface{}{
						"max_level_bonus": 0,
						"level":           1,
						"xp":              0,
						"item_seen":       false,
						"variants":        []interface{}{},
						"favorite":        false,
					},
					"quantity": 1,
				}

				lootList := []interface{}{
					map[string]interface{}{
						"itemType": reward,
						"itemGuid": reward,
						"quantity": 1,
					},
				}

				if !umbrellaExists {
					athena.Items[umbrella] = map[string]interface{}{
						"templateId": umbrella,
						"attributes": map[string]interface{}{
							"max_level_bonus": 0,
							"level":           1,
							"xp":              0,
							"item_seen":       false,
							"variants":        []interface{}{},
							"favorite":        false,
						},
						"quantity": 1,
					}
					lootList = append(lootList, map[string]interface{}{
						"itemType": umbrella,
						"itemGuid": umbrella,
						"quantity": 1,
					})
				}

				commonCore.Items["GiftBox:GB_SeasonFirstWin"] = map[string]interface{}{
					"templateId": "GiftBox:GB_SeasonFirstWin",
					"attributes": map[string]interface{}{
						"max_level_bonus": 0,
						"fromAccountId":   "",
						"lootList":        lootList,
					},
					"quantity": 1,
				}
			}
		}
		vbucks["quantity"] = qty
		commonCore.Items["Currency:MtxPurchased"] = vbucks
	}

	var levelUpChanges []map[string]interface{}

	if dailyQuests, err := fortnite_mcp.LoadDailyQuestData(utilities.GetConfig().CURRENT_VERSION); err == nil {
		for _, questProgress := range body.Remix.BroadcastQuestProgress {
			for questKey, quest := range dailyQuests {
				if _, exists := quest.Objectives[questProgress.BackendName]; exists {
					if questMap, ok := athena.Items[fmt.Sprintf("Quest:%s", questKey)].(map[string]interface{}); ok {
						if attributes, ok := questMap["attributes"].(map[string]interface{}); ok {
							key := fmt.Sprintf("completion_%s", questProgress.BackendName)
							prog := questProgress.Count
							if prog > quest.Count {
								prog = quest.Count
							}
							attributes[key] = prog

							complete := prog >= quest.Count

							if complete && attributes["quest_state"] != "Claimed" {
								for rewardType, rewardAmount := range quest.Rewards {
									if rewardType == "Currency:MtxPurchased" {
										if vb, ok := commonCore.Items["Currency:MtxPurchased"].(map[string]interface{}); ok {
											if qty, ok := vb["quantity"].(float64); ok {
												vb["quantity"] = int(qty) + rewardAmount
											}
										}
									}
								}
								delete(athena.Items, fmt.Sprintf("Quest:%s", questKey))
							}
						}
					}
					break
				}
			}
		}
	}

	bookPurchased := season.PurchasedVip

	if !bookPurchased {
		if utilities.GetConfig().CURRENT_VERSION >= 17 {
			data, err := fortnite_mcp.LoadPassData(utilities.GetConfig().CURRENT_VERSION)
			if err == nil {
				for id, offer := range data {
					if offer.OfferPriceRowHandle.RowName != nil {
						if rowName, ok := offer.OfferPriceRowHandle.RowName.(string); ok {
							if rowName == "Outfit_Included" {
								parts := strings.Split(offer.AssetPathName, ".")
								variantToken, _ := fortnite_mcp.GetCosmeticVariantToken(parts[1])
								if variantToken.TemplateID != "" {
									if len(parts) > 1 {
										athena.Items[variantToken.TemplateID] = map[string]interface{}{
											"templateId": variantToken.TemplateID,
											"attributes": map[string]interface{}{
												"item_seen": false,
												"variants":  []interface{}{},
											},
											"quantity": 1,
										}

										levelUpChanges = append(levelUpChanges, map[string]interface{}{
											"changeType": "itemAdded",
											"itemId":     variantToken.TemplateID,
											"item": map[string]interface{}{
												"templateId":  variantToken.TemplateID,
												"purchasedAt": time.Now().UTC().Format(time.RFC3339),
												"attributes": map[string]interface{}{
													"item_seen": false,
													"variants":  []interface{}{},
												},
												"quantity": 1,
											},
										})
									}

									season.PurchasedVip = true
									athena.Stats["book_purchased"] = true
									levelUpChanges = append(levelUpChanges, map[string]interface{}{
										"changeType": "statModified",
										"name":       "book_purchased",
										"value":      true,
									})

									var purchasedOffers []interface{}
									if offers, exists := athena.Stats["purchased_bp_offers"]; exists {
										if offersArr, ok := offers.([]interface{}); ok {
											purchasedOffers = offersArr
										}
									}

									purchaseEntry := map[string]interface{}{
										"offerId":           id,
										"bIsFreePassReward": false,
										"purchaseDate":      time.Now().Format(time.RFC3339),
										"lootResult": []interface{}{map[string]interface{}{
											"itemType":    variantToken.TemplateID,
											"itemGuid":    variantToken.TemplateID,
											"itemProfile": athena.ProfileID,
											"quantity":    offer.Quantity,
										}},
										"currencyType":      "battlestars",
										"totalCurrencyPaid": 0,
									}

									purchasedOffers = append(purchasedOffers, purchaseEntry)
									athena.Stats["purchased_bp_offers"] = purchasedOffers

									levelUpChanges = append(levelUpChanges, map[string]interface{}{
										"changeType": "statModified",
										"name":       "purchased_bp_offers",
										"value":      purchasedOffers,
									})

									giftBoxID := "GiftBox:gb_battlepasspurchased"
									commonCore.Items[giftBoxID] = map[string]interface{}{
										"templateId": giftBoxID,
										"attributes": map[string]interface{}{
											"max_level_bonus": 0,
											"fromAccountId":   "",
											"lootList": []interface{}{
												map[string]interface{}{
													"itemType":    variantToken.TemplateID,
													"itemGuid":    variantToken.TemplateID,
													"itemProfile": athena.ProfileID,
													"quantity":    offer.Quantity,
												},
												map[string]interface{}{
													"itemType":    "AccountResource:AthenaBattleStar",
													"itemGuid":    "AccountResource:AthenaBattleStar",
													"itemProfile": athena.ProfileID,
													"quantity":    5,
												},
											},
										},
										"quantity": 1,
									}

									currentBattleStars := float64(0)
									if stars, exists := athena.Stats["battlestars"]; exists {
										if starsFloat, ok := stars.(float64); ok {
											currentBattleStars = starsFloat
										}
									}

									newBattleStars := currentBattleStars + 5
									athena.Stats["battlestars"] = newBattleStars
									athena.Stats["battlestars_season_total"] = newBattleStars

									season.Level++
									athena.Stats["level"] = float64(season.Level)

									levelUpChanges = append(levelUpChanges, map[string]interface{}{
										"changeType": "statModified",
										"name":       "battlestars",
										"value":      newBattleStars,
									})
								}
								break
							}
						}
					}
				}
			}
		} else {
			bpData, err := fortnite_mcp.LoadBattlePassData(fmt.Sprintf("s%d", utilities.GetConfig().CURRENT_VERSION))
			if err == nil && len(bpData.Rewards) > 0 {
				firstReward := bpData.Rewards[0]

				giftBoxID := "GiftBox:gb_battlepasspurchased"
				commonCore.Items[giftBoxID] = map[string]interface{}{
					"templateId": giftBoxID,
					"attributes": map[string]interface{}{
						"max_level_bonus": 0,
						"fromAccountId":   "",
						"lootList":        []interface{}{},
					},
					"quantity": 1,
				}

				lootList := commonCore.Items[giftBoxID].(map[string]interface{})["attributes"].(map[string]interface{})["lootList"].([]interface{})

				for templateId, quantity := range firstReward {
					athena.Items[templateId] = map[string]interface{}{
						"templateId": templateId,
						"attributes": map[string]interface{}{
							"item_seen": false,
							"variants":  []interface{}{},
						},
						"quantity": quantity,
					}

					levelUpChanges = append(levelUpChanges, map[string]interface{}{
						"changeType": "itemAdded",
						"itemId":     templateId,
						"item": map[string]interface{}{
							"templateId":  templateId,
							"purchasedAt": time.Now().UTC().Format(time.RFC3339),
							"attributes": map[string]interface{}{
								"item_seen": false,
								"variants":  []interface{}{},
							},
							"quantity": quantity,
						},
					})

					lootList = append(lootList, map[string]interface{}{
						"itemType":    templateId,
						"itemGuid":    templateId,
						"itemProfile": athena.ProfileID,
						"quantity":    quantity,
					})
				}

				commonCore.Items[giftBoxID].(map[string]interface{})["attributes"].(map[string]interface{})["lootList"] = lootList

				season.PurchasedVip = true
				athena.Stats["book_purchased"] = true
				levelUpChanges = append(levelUpChanges, map[string]interface{}{
					"changeType": "statModified",
					"name":       "book_purchased",
					"value":      true,
				})
			}
		}
		bookPurchased = season.PurchasedVip
	}

	xpData, _ := fortnite_mcp.GetXPData()
	if xpData == nil {
		c.JSON(200, nil)
		return
	}

	var seasonXPData []fortnite_mcp.SeasonXP
	if utilities.GetConfig().CURRENT_VERSION < 17 {
		seasonStr := fmt.Sprintf("s%d", utilities.GetConfig().CURRENT_VERSION)
		var err error
		seasonXPData, err = fortnite_mcp.LoadSeasonXPData(seasonStr)
		if err != nil {
			log.Printf("Failed to load season XP data: %v", err)
			utilities.Internal.ServerError().Apply(c.Writer)
			return
		}
	}

	currentLevel := season.Level
	currentXP := season.Xp
	xpGained := body.FortPlayerControllerAthena.XPComponent.TotalXpEarned
	newTotalXP := currentXP + xpGained
	newLevel := currentLevel
	remainingXP := newTotalXP
	allLevelRewards := []interface{}{}

	if utilities.GetConfig().CURRENT_VERSION < 17 {
		for newLevel < len(seasonXPData) && newLevel > 0 {
			xpNeededForNextLevel := seasonXPData[newLevel-1].XpToNextLevel
			if remainingXP >= xpNeededForNextLevel {
				remainingXP -= xpNeededForNextLevel
				newLevel++

				if bookPurchased {
					bpData, err := fortnite_mcp.LoadBattlePassData(fmt.Sprintf("s%d", utilities.GetConfig().CURRENT_VERSION))
					if err == nil && newLevel-1 < len(bpData.Rewards) {
						levelRewards := bpData.Rewards[newLevel-1]

						for templateId, quantity := range levelRewards {
							if strings.Contains(templateId, "mtxgiveaway") {
								if vbucksItem, exists := commonCore.Items["Currency:MtxPurchased"]; exists {
									if vbucksMap, ok := vbucksItem.(map[string]interface{}); ok {
										if currentQuantity, ok := vbucksMap["quantity"].(int); ok {
											vbucksMap["quantity"] = currentQuantity + quantity
										}
									}
								}
							} else {
								if _, exists := athena.Items[templateId]; !exists {
									athena.Items[templateId] = map[string]interface{}{
										"templateId": templateId,
										"attributes": map[string]interface{}{
											"item_seen": false,
											"variants":  []interface{}{},
										},
										"quantity": quantity,
									}
									levelUpChanges = append(levelUpChanges, map[string]interface{}{
										"changeType": "itemAdded",
										"itemId":     templateId,
										"item": map[string]interface{}{
											"templateId":  templateId,
											"purchasedAt": time.Now().UTC().Format(time.RFC3339),
											"attributes": map[string]interface{}{
												"item_seen": false,
												"variants":  []interface{}{},
											},
											"quantity": quantity,
										},
									})
								}
							}

							allLevelRewards = append(allLevelRewards, map[string]interface{}{
								"itemType":    templateId,
								"itemGuid":    templateId,
								"itemProfile": athena.ProfileID,
								"quantity":    quantity,
							})
						}
					}
				}
			} else {
				break
			}
		}

		if bookPurchased && len(allLevelRewards) > 0 {
			commonCore.Items["GiftBox:gb_battlepass"] = map[string]interface{}{
				"templateId": "GiftBox:gb_battlepass",
				"attributes": map[string]interface{}{
					"max_level_bonus": 0,
					"fromAccountId":   "",
					"lootList":        allLevelRewards,
				},
				"quantity": 1,
			}
		}
	} else {
		for {
			levelStr := strconv.Itoa(newLevel)
			if xpLevel, exists := xpData[levelStr]; exists {
				if remainingXP >= xpLevel.XpToNextLevel {
					remainingXP -= xpLevel.XpToNextLevel
					newLevel++

					if bookPurchased && xpLevel.RewardItem == "AccountResource:AthenaBattleStar" {
						currentBattleStars := float64(0)
						if stars, exists := athena.Stats["battlestars"]; exists {
							if starsFloat, ok := stars.(float64); ok {
								currentBattleStars = starsFloat
							}
						}

						newBattleStars := currentBattleStars + 5
						athena.Stats["battlestars"] = newBattleStars

						levelUpChanges = append(levelUpChanges, map[string]interface{}{
							"changeType": "statModified",
							"name":       "battlestars",
							"value":      newBattleStars,
						})
					}
				} else {
					break
				}
			} else {
				break
			}
		}
	}

	season.Level = newLevel
	season.BookLevel = newLevel
	season.Xp = remainingXP
	season.AllXpGained += xpGained

	athena.Stats["level"] = float64(newLevel)
	athena.Stats["xp"] = float64(remainingXP)
	athena.Stats["book_level"] = float64(newLevel)

	levelUpChanges = append(levelUpChanges, map[string]interface{}{
		"changeType": "statModified",
		"name":       "level",
		"value":      float64(newLevel),
	})

	levelUpChanges = append(levelUpChanges, map[string]interface{}{
		"changeType": "statModified",
		"name":       "xp",
		"value":      float64(remainingXP),
	})

	if body.PlaylistData.EventWindowId != "" {
		var event_tokens []fortnite_tournaments.Tokens
		ttokens, err := odin.FindWhere(
			"Fortnite_Tournament_Tokens",
			map[string]interface{}{
				"account_id": body.Remix.AccountID,
			},
			func() interface{} {
				return &fortnite_tournaments.Tokens{}
			},
		)

		if err != nil || len(ttokens) == 0 {
			lategameToken := fortnite_tournaments.Tokens{
				Bucket: odin.Bucket{
					ID: fmt.Sprintf("Token_%s", uuid.New().String()),
				},
				AccountId: body.Remix.AccountID,
				Season:    utilities.GetConfig().CURRENT_VERSION,
				Token:     fmt.Sprintf("ARENA_S%s_Division1", strconv.Itoa(utilities.GetConfig().CURRENT_VERSION)),
			}
			if err := odin.Create(&lategameToken); err != nil {
				log.Printf("Failed to create tournament token: %v", err)
			} else {
				event_tokens = append(event_tokens, lategameToken)
			}
		} else {
			for _, p := range ttokens {
				event_tokens = append(event_tokens, *(p.(*fortnite_tournaments.Tokens)))
			}
		}

		var event_scores []fortnite_tournaments.Scores
		scores, err := odin.FindWhere(
			"Fortnite_Tournament_Scores",
			map[string]interface{}{
				"account_id": body.Remix.AccountID,
			},
			func() interface{} {
				return &fortnite_tournaments.Scores{}
			},
		)

		if err != nil || len(scores) == 0 {
			lategameScore := fortnite_tournaments.Scores{
				Bucket:    odin.Bucket{ID: fmt.Sprintf("Score_%s", uuid.New().String())},
				AccountId: body.Remix.AccountID,
				Season:    utilities.GetConfig().CURRENT_VERSION,
				Type:      "Hype",
				Value:     0,
			}
			if err := odin.Create(&lategameScore); err != nil {
				log.Printf("Failed to create tournament score: %v", err)
			} else {
				event_scores = append(event_scores, lategameScore)
			}
		} else {
			for _, s := range scores {
				event_scores = append(event_scores, *(s.(*fortnite_tournaments.Scores)))
			}
		}

		if strings.Contains(body.PlaylistData.PlaylistName, "Playlist_ShowdownAlt") || strings.Contains(body.PlaylistData.PlaylistName, "Vamp") {
			for i := range event_scores {
				score := &event_scores[i]
				if score.Type == "Hype" {
					season := utilities.GetConfig().CURRENT_VERSION
					accountId := body.Remix.AccountID

					var tokens []fortnite_tournaments.Tokens
					for _, t := range event_tokens {
						if t.AccountId == accountId && t.Season == season {
							tokens = append(tokens, t)
						}
					}

					divisions := []struct {
						Points int
						Token  string
					}{
						{Points: 275, Token: "ARENA_S" + strconv.Itoa(season) + "_Division2"},
						{Points: 855, Token: "ARENA_S" + strconv.Itoa(season) + "_Division3"},
						{Points: 1500, Token: "ARENA_S" + strconv.Itoa(season) + "_Division4"},
						{Points: 2000, Token: "ARENA_S" + strconv.Itoa(season) + "_Division5"},
						{Points: 3000, Token: "ARENA_S" + strconv.Itoa(season) + "_Division6"},
						{Points: 5000, Token: "ARENA_S" + strconv.Itoa(season) + "_Division7"},
						{Points: 8500, Token: "ARENA_S" + strconv.Itoa(season) + "_Division8"},
						{Points: 11500, Token: "ARENA_S" + strconv.Itoa(season) + "_Division9"},
						{Points: 15500, Token: "ARENA_S" + strconv.Itoa(season) + "_Division10"},
					}

					elimScore := body.FortPlayerStateAthena.KillScore * 5
					newScore := score.Value + elimScore

					placementPoints := []struct {
						Placement int
						Points    int
					}{
						{Placement: 1, Points: 30},
						{Placement: 2, Points: 25},
						{Placement: 5, Points: 15},
						{Placement: 25, Points: 10},
						{Placement: 50, Points: 5},
					}

					for _, pp := range placementPoints {
						if body.FortPlayerControllerAthena.MatchReport.Place <= pp.Placement &&
							body.Remix.StartingPlayers >= pp.Placement {
							newScore += pp.Points
							break
						}
					}

					for _, division := range divisions {
						if score.Value < division.Points && newScore >= division.Points {
							hasToken := false
							for _, token := range tokens {
								if strings.Contains(token.Token, division.Token) {
									hasToken = true
									break
								}
							}
							if !hasToken {
								newToken := fortnite_tournaments.Tokens{
									Bucket: odin.Bucket{
										ID: fmt.Sprintf("Token_%s", uuid.New().String()),
									},
									AccountId: accountId,
									Season:    season,
									Token:     division.Token,
								}
								if err := odin.Create(&newToken); err != nil {
									log.Printf("Failed to create division token: %v", err)
								} else {
									event_tokens = append(event_tokens, newToken)
								}
							}
						}
					}

					score.Value = newScore
					if err := score.Bucket.Save(score); err != nil {
						log.Printf("Failed to save tournament score: %v", err)
					}
				}
			}
		}
	}

	payload := map[string]interface{}{
		"type":      "com.epicgames.gift.received",
		"payload":   map[string]interface{}{},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	if err := sm.SendMessage(body.Remix.AccountID, payload); err != nil {
		log.Printf("Failed to send message: %v\n", err)
	}

	commonCore.Revision++
	athena.Revision++

	if err := commonCore.Bucket.Save(commonCore); err != nil {
		log.Printf("Failed to save common_core profile: %v", err)
		utilities.Internal.DataBaseError().Apply(c.Writer)
		return
	}

	if err := athena.Bucket.Save(athena); err != nil {
		log.Printf("Failed to save athena profile: %v", err)
		utilities.Internal.DataBaseError().Apply(c.Writer)
		return
	}

	if err := season.Bucket.Save(season); err != nil {
		log.Printf("Failed to save season profile: %v", err)
		utilities.Internal.DataBaseError().Apply(c.Writer)
		return
	}

	c.JSON(200, levelUpChanges)
}
