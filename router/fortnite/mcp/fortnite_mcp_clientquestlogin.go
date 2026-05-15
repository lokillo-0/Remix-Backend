package fortnite_mcp

import (
	"fmt"
	"strings"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/classes/mcp"
	"github.com/remixfn/xenon/managers"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
)

func POSTClientQuestLogin(c *gin.Context) {
	accountID := c.Param("accountId")
	profileId := c.Query("profileId")

	if profileId == "" || accountID == "" {
		utilities.Internal.ValidationFailed().
			WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	ua := utilities.Parse(c.GetHeader("User-Agent"))
	if ua == nil {
		utilities.Internal.ValidationFailed().
			WithMessage("Invalid User-Agent").
			WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	var account accounts.Account
	if err := odin.Find("Accounts", accountID, &account); err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	profileKey := fmt.Sprintf("%s:%s", accountID, profileId)
	var profile accounts.Profile
	profileFound := true
	if err := odin.Find("Accounts_Profiles", profileKey, &profile); err != nil {
		profileFound = false
	}

	if !profileFound && profileId != "common_public" && profileId != "collections" && profileId != "profile0" {
		managers.CreateProfile(mcp.Athena, accountID, account.DisplayName)
		managers.CreateProfile(mcp.CommonCore, accountID, account.DisplayName)
		managers.CreateProfile(mcp.Creative, accountID, account.DisplayName)

		if err := odin.Find("Accounts_Profiles", profileKey, &profile); err != nil {
			profileFound = false
		} else {
			profileFound = true
		}
	}

	now := time.Now().UTC().Format("2006-01-02T15:04:05.999Z")

	if !profileFound {
		c.JSON(200, mcp.DefaultMCPResponse{
			ProfileRevision:            0,
			ProfileId:                  profileId,
			ProfileChangesBaseRevision: 0,
			ProfileCommandRevision:     0,
			ServerTime:                 now,
			ResponseVersion:            1,
			ProfileChanges: []map[string]interface{}{
				{
					"changeType": "fullProfileUpdate",
					"profile": mcp.Profile{
						Created:    now,
						Updated:    now,
						Rvn:        0,
						WipeNumber: 1,
						AccountId:  accountID,
						ProfileId:  profileId,
						Version:    "no_version",
						Items:      make(map[string]interface{}),
						Stats: mcp.Stats{
							Attributes: make(map[string]interface{}),
						},
						CommandRevision: 0,
					},
				},
			},
		})
		return
	}

	profileData := mcp.Profile{
		Created:         now,
		Updated:         now,
		Rvn:             profile.Revision,
		WipeNumber:      1,
		AccountId:       accountID,
		ProfileId:       profile.ProfileID,
		Version:         "no_version",
		CommandRevision: profile.Revision,
		Items:           profile.Items,
		Stats: mcp.Stats{
			Attributes: profile.Stats,
		},
	}

	if _, hasGBRMTOffer := profile.Items["GiftBox:GB_RMTOffer"]; hasGBRMTOffer {
		delete(profileData.Items, "GiftBox:GB_RMTOffer")
	}

	switch profileId {
	case "athena":
		profileData.Stats.Attributes["season_num"] = ua.Season
		var season accounts.Season
		seasonKey := fmt.Sprintf("%s:%v", accountID, ua.Season)
		if err := odin.Find("Accounts_Seasons", seasonKey, &season); err == nil {
			profileData.Stats.Attributes["level"] = season.Level
			profileData.Stats.Attributes["xp"] = season.Xp
			profileData.Stats.Attributes["book_level"] = season.BookLevel
			profileData.Stats.Attributes["book_xp"] = season.BookXp
			profileData.Stats.Attributes["book_purchased"] = season.PurchasedVip
		} else {
			season = accounts.Season{
				Bucket:       odin.Bucket{ID: seasonKey},
				Level:        1,
				Xp:           0,
				BookLevel:    1,
				BookXp:       0,
				AllXpGained:  0,
				PurchasedVip: false,
				Wins:         0,
			}
			odin.Create(&season)
		}

		profileData.Items["PlayerTech:PTID_SpyTeam_01"] = map[string]interface{}{
			"attributes": map[string]interface{}{
				"favorite":  false,
				"item_seen": true,
				"level":     2,
				"name":      "Athena.Faction.Ego",
				"tag":       "Athena.Faction.Ego",
				"variants":  []interface{}{},
				"xp":        0,
			},
			"quantity":   1,
			"templateId": "PlayerTech:PTID_SpyTeam_01",
		}
		profileData.Items["PlayerTech:PTID_SpyTeam_02"] = map[string]interface{}{
			"attributes": map[string]interface{}{
				"favorite":  false,
				"item_seen": true,
				"level":     2,
				"variants":  []interface{}{},
				"xp":        0,
			},
			"quantity":   1,
			"templateId": "PlayerTech:PTID_SpyTeam_02",
		}

		if IsAthenaCacheLoaded() {
			hasFullLocker := HasFullLockerReward(accountID)
			if hasFullLocker {
				ApplyAthenaCache(&profileData)
			}

			for key, item := range profileData.Items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if templateId, hasTemplate := itemMap["templateId"].(string); hasTemplate &&
						!strings.Contains(strings.ToLower(key), "loadout") &&
						!strings.Contains(strings.ToLower(templateId), "loadout") {
						itemMap["templateId"] = key
					}
				}
			}
		}

		profileData.Items = managers.InitLocationQuest(profileData.Items)

		if dailyQuests, err := LoadDailyQuestData(ua.Season); err == nil {
			questManager, exists := profileData.Stats.Attributes["quest_manager"].(map[string]interface{})
			questManager["dailyQuestRerolls"] = 0
			if !exists {
				questManager = map[string]interface{}{
					"dailyLoginInterval": "0001-01-01T00:00:00Z",
					"dailyQuestRerolls":  0,
				}
				profileData.Stats.Attributes["quest_manager"] = questManager
			}

			lastLoginStr, _ := questManager["dailyLoginInterval"].(string)
			lastLogin, err := time.Parse("2006-01-02T15:04:05Z", lastLoginStr)
			if err != nil {
				lastLogin = time.Time{}
			}

			currentTime := time.Now().UTC()
			y1, m1, d1 := lastLogin.Date()
			y2, m2, d2 := currentTime.Date()
			isNewDay := lastLogin.IsZero() || !(y1 == y2 && m1 == m2 && d1 == d2)

			existingDailyCount := 0
			existingQuests := make(map[string]bool)
			for itemKey := range profileData.Items {
				if strings.HasPrefix(itemKey, "Quest:quest_s") && strings.Contains(itemKey, "_daily_") {
					existingDailyCount++
					questKey := strings.TrimPrefix(itemKey, "Quest:")
					existingQuests[questKey] = true
				}
			}

			shouldGrantQuests := isNewDay || existingDailyCount < 3

			if shouldGrantQuests {
				if isNewDay {
					for itemKey := range profileData.Items {
						if strings.HasPrefix(itemKey, "Quest:quest_s") && strings.Contains(itemKey, "_daily_") {
							delete(profileData.Items, itemKey)
							delete(profile.Items, itemKey)
						}
					}
					existingDailyCount = 0
					existingQuests = make(map[string]bool)
				}

				var availableQuests []string
				for questKey := range dailyQuests {
					if !existingQuests[questKey] {
						availableQuests = append(availableQuests, questKey)
					}
				}

				questsToGrant := 3 - existingDailyCount
				if questsToGrant > len(availableQuests) {
					questsToGrant = len(availableQuests)
				}

				for i := 0; i < questsToGrant && i < len(availableQuests); i++ {
					randomIndex := int(currentTime.UnixNano()) % len(availableQuests)
					questKey := availableQuests[randomIndex]

					availableQuests = append(availableQuests[:randomIndex], availableQuests[randomIndex+1:]...)

					quest := dailyQuests[questKey]
					questTemplateId := fmt.Sprintf("Quest:%s", questKey)

					now := currentTime.Format("2006-01-02T15:04:05.999Z")
					objectiveStates := make(map[string]interface{})

					for objKey := range quest.Objectives {
						objectiveStates["completion_"+objKey] = 0
					}

					xpReward := 0
					if xp, exists := quest.Rewards["AccountResource:athenaseasonalxp"]; exists {
						xpReward = xp
					}

					combinedAttributes := map[string]interface{}{
						"creation_time":                 now,
						"level":                         -1,
						"item_seen":                     true,
						"playlists":                     []string{},
						"sent_new_notification":         true,
						"challenge_bundle_id":           "",
						"xp_reward_scalar":              1,
						"challenge_linked_quest_given":  "",
						"quest_pool":                    "",
						"quest_state":                   "Active",
						"bucket":                        "",
						"last_state_change_time":        now,
						"challenge_linked_quest_parent": "",
						"max_level_bonus":               0,
						"xp":                            xpReward,
						"quest_rarity":                  "uncommon",
						"favorite":                      false,
					}

					for key, value := range objectiveStates {
						combinedAttributes[key] = value
					}

					profileData.Items[questTemplateId] = mcp.BaseItem{
						TemplateId: questTemplateId,
						Attributes: combinedAttributes,
						Quantity:   1,
					}
					profile.Items[questTemplateId] = profileData.Items[questTemplateId]
				}

				if isNewDay {
					questManager["dailyLoginInterval"] = currentTime.Format("2006-01-02T15:04:05Z")
					profileData.Stats.Attributes["quest_manager"] = questManager
				}
			}
		}

	case "common_core":
		HasAthenaReward(accountID)

		if account.MatchmakingBannedSince != "" {
			banStartTime, _ := time.Parse(time.RFC3339, account.MatchmakingBannedSince)
			banEndTime, _ := time.Parse(time.RFC3339, account.MatchmakingBannedUntil)
			banDurationDays := int(banEndTime.Sub(banStartTime).Hours() / 24)

			profileData.Stats.Attributes["ban_status"] = gin.H{
				"bRequiresUserAck":     true,
				"bBanHasStarted":       true,
				"banStartTimeUtc":      account.MatchmakingBannedSince,
				"banDurationDays":      banDurationDays,
				"banReasons":           []string{account.MatchmakingBannedReason},
				"additionalInfo":       "",
				"exploitProgramName":   "",
				"competitiveBanReason": "None",
			}
		}
	}

	if profile.Bucket.ID != profileKey {
		profile.Bucket.ID = profileKey
	}

	profile.Revision++
	profile.Bucket.Save(profile)

	c.JSON(200, mcp.DefaultMCPResponse{
		ProfileRevision:            profile.Revision,
		ProfileId:                  profileId,
		ProfileChangesBaseRevision: profile.Revision - 1,
		ProfileCommandRevision:     profile.Revision,
		ServerTime:                 now,
		ResponseVersion:            1,
		ProfileChanges: []map[string]interface{}{
			{
				"changeType": "fullProfileUpdate",
				"profile":    profileData,
			},
		},
	})
}
