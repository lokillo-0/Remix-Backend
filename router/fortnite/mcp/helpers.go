package fortnite_mcp

import (
	"fmt"
	"strings"

	"github.com/andr1ww/odin"
	"github.com/remixfn/xenon/classes/mcp"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
)

func HasFullLockerReward(accountID string) bool {
	rewardsResults, err := odin.FindWhere("Accounts_Rewards", map[string]interface{}{
		"account_id": accountID,
	}, func() interface{} {
		return &accounts.AccountReward{}
	})

	if err != nil {
		return false
	}

	for _, result := range rewardsResults {
		if reward, ok := result.(*accounts.AccountReward); ok && reward.Redeemed {
			for _, rewardItem := range reward.Rewards {
				if rewardItem == "Full Locker" {
					return true
				}
			}
		}
	}
	return false
}

func HasAccess(accountID string) bool {
	if utilities.GetConfig().Rewards && utilities.GetConfig().Maintenance {
		rewardsResults, err := odin.FindWhere("Accounts_Rewards", map[string]interface{}{
			"account_id": accountID,
		}, func() interface{} {
			return &accounts.AccountReward{}
		})

		if err != nil {
			return false
		}

		for _, result := range rewardsResults {
			if reward, ok := result.(*accounts.AccountReward); ok && reward.Redeemed {
				for _, rewardItem := range reward.Rewards {
					if strings.Contains(rewardItem, "Beta") {
						return true
					}
				}
			}
		}
	} else if utilities.GetConfig().Maintenance {
		admins, err := odin.FindWhere("Remix_Admins", map[string]interface{}{
			"account_id": accountID,
		}, func() interface{} {
			return &accounts.AccountReward{}
		})
		if err != nil || len(admins) == 0 {
			return false
		}

		return true
	}

	return !utilities.GetConfig().Maintenance
}

func HasAthenaReward(accountID string) {
	if accountID == "" {
		return
	}

	athenaKey := fmt.Sprintf("%s:athena", accountID)
	var profile accounts.Profile
	if err := odin.Find("Accounts_Profiles", athenaKey, &profile); err != nil {
		return
	}

	rewardsResults, err := odin.FindWhere("Accounts_Rewards", map[string]interface{}{
		"account_id": accountID,
	}, func() interface{} {
		return &accounts.AccountReward{}
	})

	if err != nil {
		return
	}

	allAthena := GetAthenaCachedItems()
	athenaItems := make(map[string]*mcp.BaseItem)
	for _, item := range allAthena {
		if strings.Contains(strings.ToLower(item.TemplateId), "athena") {
			athenaItems[item.TemplateId] = &item
		}
	}

	if len(athenaItems) == 0 {
		return
	}

	var itemsToAdd []string

	for _, result := range rewardsResults {
		reward, ok := result.(*accounts.AccountReward)
		if !ok || !reward.Redeemed || len(reward.Rewards) == 0 {
			continue
		}

		var toRemove []string

		for _, rewardItem := range reward.Rewards {
			if athenaItem, exists := athenaItems[rewardItem]; exists {
				profile.Items[rewardItem] = map[string]interface{}{
					"templateId": athenaItem.TemplateId,
					"attributes": athenaItem.Attributes,
					"quantity":   athenaItem.Quantity,
				}
				toRemove = append(toRemove, rewardItem)
				itemsToAdd = append(itemsToAdd, rewardItem)
			}
		}

		if len(toRemove) > 0 {
			removeSet := make(map[string]struct{}, len(toRemove))
			for _, item := range toRemove {
				removeSet[item] = struct{}{}
			}

			newRewards := reward.Rewards[:0]
			for _, r := range reward.Rewards {
				if _, remove := removeSet[r]; !remove {
					newRewards = append(newRewards, r)
				}
			}
			reward.Rewards = newRewards
		}

		reward.Bucket.Save(reward)
	}

	var lootLists []map[string]interface{}
	for _, id := range itemsToAdd {
		lootLists = append(lootLists, map[string]interface{}{
			"itemType":    id,
			"itemProfile": "athena",
			"itemGuid":    id,
			"quantity":    1,
		})
	}

	profileKey := fmt.Sprintf("%s:%s", accountID, "common_core")
	var commonCoreProfile accounts.Profile
	if err := odin.Find("Accounts_Profiles", profileKey, &commonCoreProfile); err != nil {
		return
	}

	if err == nil && len(commonCoreProfile.Items) > 0 {
		commonCoreProfile.Items["GiftBox:GB_RMTOffer"] = map[string]interface{}{
			"templateId": "GiftBox:GB_RMTOffer",
			"attributes": map[string]interface{}{
				"max_level_bonus": 0,
				"fromAccountId":   "Epic Games",
				"lootList":        []map[string]interface{}{},
				"params": map[string]interface{}{
					"userMessage": "Thank you for playing Remix, Here's your donator perks!",
				},
			},
			"quantity": 1,
		}
		commonCoreProfile.Bucket.Save(commonCoreProfile)
	}

	profile.Bucket.Save(profile)
}
