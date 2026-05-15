package remix_server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
	"golang.org/x/crypto/bcrypt"
)

func POSTRemixServerChangeDisplayName(c *gin.Context) {
	if !checkAdminAuth(c) {
		return
	}
	accountID := c.Param("accountid")
	var request struct {
		NewDisplayName string `json:"displayName" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	var account accounts.Account
	if err := odin.Find("Accounts", accountID, &account); err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	if account.Banned {
		utilities.Account.DisabledAccount().Apply(c.Writer)
		return
	}

	if account.DisplayName == request.NewDisplayName {
		utilities.Account.InvalidAccountIdCount().Apply(c.Writer)
		return
	}

	account.DisplayName = request.NewDisplayName
	account.LastDisplayNameChange = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	account.DisplayNameChanges++
	account.Bucket.Save(account)

	c.JSON(200, nil)
}

func POSTRemixServerChangeEmail(c *gin.Context) {
	if !checkAdminAuth(c) {
		return
	}
	accountID := c.Param("accountid")
	var request struct {
		NewEmail string `json:"new_email" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	var account accounts.Account
	if err := odin.Find("Accounts", accountID, &account); err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	if account.Banned {
		utilities.Account.DisabledAccount().Apply(c.Writer)
		return
	}

	account.Email = request.NewEmail
	account.Bucket.Save(account)

	c.JSON(200, nil)
}

func POSTRemixServerChangePassword(c *gin.Context) {
	if !checkAdminAuth(c) {
		return
	}
	accountID := c.Param("accountid")
	var request struct {
		NewPassword string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	var account accounts.Account
	if err := odin.Find("Accounts", accountID, &account); err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	if account.Banned {
		utilities.Account.DisabledAccount().Apply(c.Writer)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.String(500, "failed to hash password!")
		return
	}

	account.Password = string(hashedPassword)
	account.Bucket.Save(account)

	c.JSON(200, nil)
}

func POSTRemixServerBanAccount(c *gin.Context) {
	if !checkAdminAuth(c) {
		return
	}

	identifier := c.Param("accountid")
	if identifier == "" {
		c.JSON(400, gin.H{"error": "missing accountId"})
		return
	}

	accountID, err := resolveAccountID(identifier)
	if err != nil || accountID == "" {
		c.JSON(404, gin.H{"error": "Account not found"})
		return
	}

	var request struct {
		Reason string `json:"reason"`
	}

	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body", "details": err.Error()})
			return
		}
	}

	var account accounts.Account
	if err := odin.Find("Accounts", accountID, &account); err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	if account.Banned {
		c.JSON(200, gin.H{
			"message":   "Account already banned",
			"accountId": account.ID,
			"banned":    true,
		})
		return
	}

	reason := strings.TrimSpace(request.Reason)
	if reason == "" {
		reason = "Banned by admin panel"
	}

	now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

	account.Banned = true
	account.MatchmakingBannedSince = now
	account.MatchmakingBannedUntil = "PERMANENT"
	account.MatchmakingBannedReason = reason
	account.BanHistory = append(account.BanHistory, map[string]string{
		"reason": reason,
		"at":     now,
		"source": "admin_panel",
	})

	if err := account.Bucket.Save(account); err != nil {
		c.JSON(500, gin.H{"error": "Failed to save account", "details": err.Error()})
		return
	}

	anticheatURL := fmt.Sprintf("https://dev-anticheat-v1.arc-services.dev/router/v1/anticheat/public/deactivate/%s", account.ID)
	anticheatReq, err := http.NewRequest(http.MethodPost, anticheatURL, nil)
	if err == nil {
		anticheatReq.Header.Set("X-Arc-Auth", "iRyJgy-9owCM9vByUQn9ffjTZkza1kK0usZUlId-e4i5q3t83REPOwBjy672l_EEzzy17rt8judAUQDOKOjAKr3bD2_5qf2W5sJ2Ub4rUeWx4dwt06RWRhqy4EY7VWDKiB39ugJTIl9vs-cjsgi3Zj8oNYkpdZVKBOsL4sfLBvv8acYy3_BKuaigXNl0W85bNzJidDNGKYD-jFbzai2mLtGJhjzfujIK-WrMGx5t-gsf")
		anticheatReq.Header.Set("X-Arc-Client", "zszxfhpnjgzxzvpiiopxqrfpivjqhxbh")
		anticheatResp, anticheatErr := http.DefaultClient.Do(anticheatReq)
		if anticheatErr == nil {
			anticheatResp.Body.Close()
		}
	}

	sessions, _ := odin.FindWhere("Accounts_Sessions", map[string]interface{}{
		"accountId": account.ID,
	}, func() interface{} {
		return &accounts.Session{}
	})

	for _, sessionData := range sessions {
		session, ok := sessionData.(*accounts.Session)
		if !ok {
			continue
		}
		session.Delete(session)
	}

	c.JSON(200, gin.H{
		"message":   "Account banned successfully",
		"accountId": account.ID,
		"banned":    true,
		"reason":    reason,
	})
}

func POSTRemixServerUnbanAccount(c *gin.Context) {
	if !checkAdminAuth(c) {
		return
	}

	identifier := c.Param("accountid")
	if identifier == "" {
		c.JSON(400, gin.H{"error": "missing accountId"})
		return
	}

	accountID, err := resolveAccountID(identifier)
	if err != nil || accountID == "" {
		c.JSON(404, gin.H{"error": "Account not found"})
		return
	}

	var account accounts.Account
	if err := odin.Find("Accounts", accountID, &account); err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	if !account.Banned {
		c.JSON(200, gin.H{
			"message":   "Account is not banned",
			"accountId": account.ID,
			"banned":    false,
		})
		return
	}

	now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

	account.Banned = false
	account.MatchmakingBannedSince = ""
	account.MatchmakingBannedUntil = ""
	account.MatchmakingBannedReason = ""
	account.BanHistory = append(account.BanHistory, map[string]string{
		"action": "unban",
		"at":     now,
		"source": "admin_panel",
	})

	if err := account.Bucket.Save(account); err != nil {
		c.JSON(500, gin.H{"error": "Failed to save account", "details": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message":   "Account unbanned successfully",
		"accountId": account.ID,
		"banned":    false,
	})
}
