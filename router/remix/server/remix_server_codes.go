package remix_server

import (
	"net/http"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/modules/database/buckets/remix"
	"github.com/remixfn/xenon/utilities"
)

func PUTRemixServerCreateCode(c *gin.Context) {
	if !checkAdminAuth(c) { return }
	var body struct {
		Code    string   `json:"Code" binding:"required"`
		Package string   `json:"Package" binding:"required"`
		Rewards []string `json:"Rewards" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		utilities.MCP.InvalidPayload().WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	if body.Code == "" {
		utilities.MCP.InvalidPayload().
			WithMessage("Code cannot be empty").
			WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	if body.Package == "" {
		utilities.MCP.InvalidPayload().
			WithMessage("Package cannot be empty").
			WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	if len(body.Rewards) == 0 {
		utilities.MCP.InvalidPayload().
			WithMessage("Rewards cannot be empty").
			WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	var existingCode remix.Code
	if err := odin.Find("Remix_Codes", body.Code, &existingCode); err != nil {

	}

	if existingCode.Code == body.Code {
		utilities.MCP.InvalidPayload().
			WithMessage("Code already exists").
			WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	now := time.Now().Unix()
	expires := now + (365 * 24 * 60 * 60)

	newCode := &remix.Code{
		Bucket:  odin.Bucket{ID: body.Code},
		Code:    body.Code,
		Package: body.Package,
		Rewards: body.Rewards,
		Created: now,
		Expires: expires,
	}

	if err := newCode.Bucket.Save(newCode); err != nil {
		utilities.Internal.DataBaseError().
			WithIntent(utilities.Prod).
			WithMessage("Failed to create code").
			Apply(c.Writer)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    newCode.Code,
		"package": newCode.Package,
		"rewards": newCode.Rewards,
		"created": newCode.Created,
		"expires": newCode.Expires,
	})
}

func POSTRemixServerCodesRedeem(c *gin.Context) {
	accountID := c.Param("accountId")
	code := c.Param("code")
	if accountID == "" {
		utilities.MCP.InvalidPayload().
			WithMessage("Account ID is required").
			WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}
	if code == "" {
		utilities.MCP.InvalidPayload().
			WithMessage("Code is required").
			WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	var account accounts.Account
	if err := odin.Find("Accounts", accountID, &account); err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	var foundCode remix.Code
	if err := odin.Find("Remix_Codes", code, &foundCode); err != nil || foundCode.Code != code {
		utilities.MCP.InvalidPayload().
			WithMessage("Code not found").
			WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	now := time.Now().Unix()
	if foundCode.Expires < now {
		utilities.MCP.InvalidPayload().
			WithMessage("Code has expired").
			WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	existingRewards, err := odin.FindWhere("Accounts_Rewards", map[string]interface{}{
		"account_id": accountID,
		"code_id":    code,
	}, func() interface{} {
		return &accounts.AccountReward{}
	})
	if err == nil && len(existingRewards) > 0 {
		utilities.MCP.InvalidPayload().
			WithMessage("Code already redeemed").
			WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	accountRewards := make(map[string]bool)
	allRewards, err := odin.FindWhere("Accounts_Rewards", map[string]interface{}{
		"account_id": accountID,
	}, func() interface{} {
		return &accounts.AccountReward{}
	})
	if err == nil {
		for _, result := range allRewards {
			if reward, ok := result.(*accounts.AccountReward); ok && reward.Redeemed {
				for _, rewardItem := range reward.Rewards {
					accountRewards[rewardItem] = true
				}
			}
		}
	}

	for _, codeReward := range foundCode.Rewards {
		if !accountRewards[codeReward] {
			newReward := &accounts.AccountReward{
				Bucket:    odin.Bucket{ID: uuid.New().String()},
				AccountID: accountID,
				CodeID:    foundCode.Code,
				Rewards:   []string{codeReward},
				Redeemed:  true,
			}
			if err := newReward.Bucket.Save(newReward); err != nil {
				utilities.Internal.DataBaseError().
					WithIntent(utilities.Prod).
					WithMessage("Failed to save reward").
					Apply(c.Writer)
				return
			}
		}
	}

	if err := foundCode.Bucket.Delete(foundCode); err != nil {
		utilities.Internal.DataBaseError().
			WithIntent(utilities.Prod).
			WithMessage("Failed to delete code").
			Apply(c.Writer)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}
