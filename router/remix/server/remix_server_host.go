package remix_server

import (
	"strings"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
	"golang.org/x/crypto/bcrypt"
)

func POSTCreateHostAccount(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader != utilities.GetConfig().JWTSecret+":"+"server" {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	var request struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	request.Username = strings.TrimSpace(request.Username)
	if request.Username == "" {
		c.JSON(400, gin.H{"error": "username cannot be blank"})
		return
	}
	if len(request.Password) < 6 {
		c.JSON(400, gin.H{"error": "password must be at least 6 characters"})
		return
	}

	internalEmail := strings.ToLower(request.Username) + "@host.server"

	existingByName, err := odin.FindWhere("Accounts", map[string]interface{}{
		"display_name": request.Username,
	}, func() interface{} { return &accounts.Account{} })
	if err == nil && len(existingByName) > 0 {
		c.JSON(409, gin.H{"error": "A host account with that username already exists"})
		return
	}

	existingByEmail, err := odin.FindWhere("Accounts", map[string]interface{}{
		"email": internalEmail,
	}, func() interface{} { return &accounts.Account{} })
	if err == nil && len(existingByEmail) > 0 {
		c.JSON(409, gin.H{"error": "A host account with that username already exists"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to hash password"})
		return
	}

	accountID := strings.ReplaceAll(uuid.New().String(), "-", "")

	newAccount := &accounts.Account{
		Bucket:                  odin.Bucket{ID: accountID},
		Created:                 time.Now(),
		Email:                   internalEmail,
		Password:                string(hashedPassword),
		DisplayName:             request.Username,
		Banned:                  false,
		Roles:                   []string{"-1"},
		BanHistory:              []map[string]string{},
		IsServer:                true, // <-- this is what makes it a host/server account
		LastLoginTime:           time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		LastLoginIP:             "",
		ProfilePicture:          "",
		DisplayNameChanges:      0,
		MatchmakingBannedUntil:  "",
		MatchmakingBannedSince:  "",
		MatchmakingBannedReason: "",
	}

	if err := odin.Create(newAccount); err != nil {
		c.JSON(500, gin.H{
			"error":   "Failed to create host account",
			"details": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"account_id":   newAccount.ID,
		"display_name": newAccount.DisplayName,
		"email":        newAccount.Email,
		"is_server":    newAccount.IsServer,
		"created":      newAccount.Created,
	})
}

func POSTHostAccountLogin(c *gin.Context) {
	var request struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	internalEmail := strings.ToLower(strings.TrimSpace(request.Username)) + "@host.server"

	found, err := odin.FindWhere("Accounts", map[string]interface{}{
		"email": internalEmail,
	}, func() interface{} { return &accounts.Account{} })

	if err != nil || len(found) == 0 {
		c.JSON(401, gin.H{"error": "Invalid username or password"})
		return
	}

	account := found[0].(*accounts.Account)

	if account.Banned {
		c.JSON(403, gin.H{"error": "This host account has been disabled"})
		return
	}

	if !account.IsServer {
		// Safety guard — shouldn't happen, but reject non-server accounts here
		c.JSON(403, gin.H{"error": "Account is not a host account"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(request.Password)); err != nil {
		c.JSON(401, gin.H{"error": "Invalid username or password"})
		return
	}

	// Update last login info
	account.LastLoginTime = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	account.LastLoginIP = c.ClientIP()
	_ = account.Bucket.Save(account)

	c.JSON(200, gin.H{
		"account_id":   account.ID,
		"display_name": account.DisplayName,
		"email":        account.Email,
		"is_server":    account.IsServer,
		"hint": "Use grant_type=password with email and password against /account/api/oauth/token to obtain an access token",
	})
}

func DELETEHostAccount(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader != utilities.GetConfig().JWTSecret+":"+"server" {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	username := c.Param("username")
	if username == "" {
		c.JSON(400, gin.H{"error": "username param is required"})
		return
	}

	internalEmail := strings.ToLower(username) + "@host.server"

	found, err := odin.FindWhere("Accounts", map[string]interface{}{
		"email": internalEmail,
	}, func() interface{} { return &accounts.Account{} })

	if err != nil || len(found) == 0 {
		c.JSON(404, gin.H{"error": "Host account not found"})
		return
	}

	account := found[0].(*accounts.Account)

	if !account.IsServer {
		c.JSON(400, gin.H{"error": "That account is not a host account"})
		return
	}

	if err := account.Bucket.Delete(account); err != nil {
		c.JSON(500, gin.H{"error": "Failed to delete host account", "details": err.Error()})
		return
	}

	c.JSON(200, gin.H{"deleted": account.DisplayName, "account_id": account.ID})
}

func GETListHostAccounts(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader != utilities.GetConfig().JWTSecret+":"+"server" {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	all, err := odin.FindWhere("Accounts", map[string]interface{}{
		"is_server": true,
	}, func() interface{} { return &accounts.Account{} })

	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to query accounts", "details": err.Error()})
		return
	}

	type hostInfo struct {
		AccountID   string `json:"account_id"`
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
		Banned      bool   `json:"banned"`
		Created     string `json:"created"`
	}

	result := make([]hostInfo, 0, len(all))
	for _, a := range all {
		acc := a.(*accounts.Account)
		result = append(result, hostInfo{
			AccountID:   acc.ID,
			DisplayName: acc.DisplayName,
			Email:       acc.Email,
			Banned:      acc.Banned,
			Created:     acc.Created.Format(time.RFC3339),
		})
	}

	c.JSON(200, result)
}
