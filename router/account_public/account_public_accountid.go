package account_public

import (
	"net/http"
	"strings"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
)

func GETAccountPublicAccountID(c *gin.Context) {
	accountID := c.Param("accountId")

	var account accounts.Account
	if err := odin.Find("Accounts", accountID, &account); err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	if account.Banned {
		utilities.Account.DisabledAccount().Apply(c.Writer)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":            account.ID,
		"displayName":   account.DisplayName,
		"externalAuths": gin.H{},
		"cabinedMode":   false,
		"minorVerified": false,
		"minorExpected": false,
		"minorStatus":   "NOT_MINOR",
	})
}

func GetAvatarIds(c *gin.Context) {
	accountIdsParam := c.Query("accountIds")
	if accountIdsParam == "" {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}

	accountIds := strings.Split(accountIdsParam, ",")
	response := []gin.H{}

	for _, accountId := range accountIds {
		accountId = strings.TrimSpace(accountId)

		var locker accounts.Locker
		lockerKey := accountId + ":locker:v4"
		if err := odin.Find("Accounts_LockerV4", lockerKey, &locker); err != nil {
			continue
		}

		charLoadout, ok := locker.ActiveLoadoutGroup.Loadouts["CosmeticLoadout:LoadoutSchema_Character"]
		if !ok {
			continue
		}

		equippedId := ""
		for _, slot := range charLoadout.LoadoutSlots {
			if slot.SlotTemplate == "CosmeticLoadoutSlotTemplate:LoadoutSlot_Character" {
				equippedId = slot.EquippedItemId
				break
			}
		}

		if equippedId == "" {
			continue
		}

		response = append(response, gin.H{
			"accountId": accountId,
			"namespace": "fortnite",
			"avatarId":  strings.ToUpper(equippedId),
		})
	}

	c.JSON(http.StatusOK, response)
}

func GetPlayerByDisplayName(c *gin.Context) {
	displayName := c.Param("displayName")
	if displayName == "" {
		utilities.Internal.ServerError().WithMessage("displayName query parameter is required").Apply(c.Writer)
		return
	}

	acc, err := odin.FindWhere("Accounts", map[string]interface{}{
		"display_name": displayName,
	}, func() interface{} {
		return &accounts.Account{}
	})
	if err != nil || len(acc) == 0 {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}
	account := acc[0].(*accounts.Account)

	c.JSON(http.StatusOK, gin.H{
		"id":            account.ID,
		"displayName":   account.DisplayName,
		"externalAuths": gin.H{},
		"cabinedMode":   false,
		"minorVerified": false,
		"minorExpected": false,
		"minorStatus":   "NOT_MINOR",
	})
}
