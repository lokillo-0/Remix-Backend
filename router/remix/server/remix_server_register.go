package remix_server

import (
	"strings"
	"sync"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
	"golang.org/x/crypto/bcrypt"
)

type RateLimiterType struct {
	mu    sync.Mutex
	store map[string]rateLimitEntry
}

type rateLimitEntry struct {
	Count     int
	ExpiresAt time.Time
}

func NewRateLimiter() *RateLimiterType {
	return &RateLimiterType{
		store: make(map[string]rateLimitEntry),
	}
}

func (rl *RateLimiterType) Get(key string) (interface{}, bool) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	entry, ok := rl.store[key]
	if !ok || time.Now().After(entry.ExpiresAt) {
		return 0, false
	}
	return entry.Count, true
}

func (rl *RateLimiterType) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	count := value.(int)
	rl.store[key] = rateLimitEntry{
		Count:     count,
		ExpiresAt: time.Now().Add(ttl),
	}
}

var RateLimiter = NewRateLimiter()

func POSTRemixServerRegister(c *gin.Context) {
	var request struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
		Username string `json:"username" binding:"required"`
		Server   bool   `json:"server"`
	}

	ip := c.ClientIP()
	key := "attempt:" + ip

	attempts, found := RateLimiter.Get(key)
	if found && attempts.(int) >= 5 {
		c.JSON(429, gin.H{
			"error": "Too many registration attempts from this IP. Please try again later.",
		})
		c.Abort()
		return
	}
	RateLimiter.SetWithTTL(key, func() interface{} {
		if found {
			return attempts.(int) + 1
		}
		return 1
	}(), 10*time.Second)

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader != utilities.GetConfig().JWTSecret+":"+"server" {
		c.JSON(401, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	if !strings.Contains(request.Email, "@")  {
		c.JSON(400, gin.H{
			"error": "Invalid email address",
		})
		return
	}

	existingAccounts, err := odin.FindWhere("Accounts", map[string]interface{}{
		"email": request.Email,
	}, func() interface{} {
		return &accounts.Account{}
	})

	if err == nil && len(existingAccounts) > 0 {
		c.JSON(409, gin.H{
			"error": "Account with this email already exists",
		})
		return
	}

	existingUsernames, err := odin.FindWhere("Accounts", map[string]interface{}{
		"display_name": request.Username,
	}, func() interface{} {
		return &accounts.Account{}
	})

	if err == nil && len(existingUsernames) > 0 {
		c.JSON(409, gin.H{
			"error": "Account with this username already exists",
		})
		return
	}

	accountID := uuid.New().String()
	accountID = strings.ReplaceAll(accountID, "-", "")

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(500, gin.H{
			"error": "Failed to hash password",
		})
		return
	}

	newAccount := &accounts.Account{
		Bucket:                  odin.Bucket{ID: accountID},
		Created:                 time.Now(),
		Email:                   request.Email,
		Password:                string(hashedPassword),
		DisplayName:             request.Username,
		Username:                request.Username,
		Banned:                  false,
		Roles:                   []string{"-1"},
		BanHistory:              []map[string]string{},
		IsServer:                request.Server,
		LastLoginTime:           time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		LastLoginIP:             "",
		ProfilePicture:          "https://cdn.discordapp.com/avatars//",
		DisplayNameChanges:      0,
		MatchmakingBannedUntil:  "",
		MatchmakingBannedSince:  "",
		MatchmakingBannedReason: "",
	}

	err = odin.Create(newAccount)
	if err != nil {
		c.JSON(500, gin.H{
			"error":   "Failed to create account",
			"details": err.Error(),
		})
		return
	}

	c.JSON(200, newAccount)
}

func POSTMassDeleteAccountsByEmailPrefix(c *gin.Context) {
	if !checkAdminAuth(c) { return }
	var request struct {
		Prefix string `json:"prefix" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utilities.Basic.BadRequest().Apply(c.Writer)
		return
	}

	accountsToDelete, err := odin.FindWhere("Accounts", map[string]interface{}{}, func() interface{} {
		return &accounts.Account{}
	})

	if err != nil {
		utilities.Basic.BadRequest().Apply(c.Writer)
		return
	}

	var deletedIDs []string
	for _, acc := range accountsToDelete {
		account := acc.(*accounts.Account)
		email := strings.ToLower(account.Email)
		prefix := strings.ToLower(request.Prefix)
		if strings.HasPrefix(email, prefix) {
			if err := account.Bucket.Delete(account); err == nil {
				deletedIDs = append(deletedIDs, account.Bucket.ID)
			}
		}
	}

	c.JSON(200, gin.H{
		"ids":   deletedIDs,
		"count": len(deletedIDs),
	})
}

func POSTMassDeleteAccountsByDomain(c *gin.Context) {
	if !checkAdminAuth(c) { return }
	var request struct {
		Domain string `json:"domain" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utilities.Basic.BadRequest().Apply(c.Writer)
		return
	}

	accountsToDelete, err := odin.FindWhere("Accounts", map[string]interface{}{}, func() interface{} {
		return &accounts.Account{}
	})

	if err != nil {
		utilities.Basic.BadRequest().Apply(c.Writer)
		return
	}

	var deletedIDs []string
	for _, acc := range accountsToDelete {
		account := acc.(*accounts.Account)
		email := strings.ToLower(account.Email)
		domain := "@" + strings.ToLower(request.Domain)
		if strings.HasSuffix(email, domain) || !strings.Contains(email, "@") {
			if err := account.Bucket.Delete(account); err == nil {
				deletedIDs = append(deletedIDs, account.Bucket.ID)
			}
		}
	}

	c.JSON(200, gin.H{
		"ids":   deletedIDs,
		"count": len(deletedIDs),
	})
}

func DELETEAccountByDisplayName(c *gin.Context) {
	if !checkAdminAuth(c) { return }
	name := c.Param("name")
	found, err := odin.FindWhere("Accounts", map[string]interface{}{
		"display_name": name,
	}, func() interface{} { return &accounts.Account{} })
	if err != nil || len(found) == 0 {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	account := found[0].(*accounts.Account)
	account.Bucket.Delete(account)
	c.JSON(200, gin.H{"deleted": account.ID})
}

func POSTRemixServerPasswordReset(c *gin.Context) {
	if !checkAdminAuth(c) { return }
	var request struct {
		Email       string `json:"email" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	accountsFound, err := odin.FindWhere("Accounts", map[string]interface{}{
		"email": request.Email,
	}, func() interface{} {
		return &accounts.Account{}
	})

	if err != nil || len(accountsFound) == 0 {
		c.JSON(404, gin.H{
			"error": "Account not found",
		})
		return
	}

	account := accountsFound[0].(*accounts.Account)

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(500, gin.H{
			"error": "Failed to hash new password",
		})
		return
	}

	account.Password = string(hashedPassword)

	if err := account.Bucket.Save(account); err != nil {
		c.JSON(500, gin.H{
			"error":   "Failed to update password",
			"details": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"message": "Password reset successful",
	})
}

func checkAdminAuth(c *gin.Context) bool {
	authHeader := c.GetHeader("Authorization")
	if authHeader != utilities.GetConfig().ADMIN_KEY {
		c.JSON(401, gin.H{"error": "Unauthorized: Invalid Admin Key"})
		return false
	}
	return true
}

func GETAdminAccountsAll(c *gin.Context) {
	if !checkAdminAuth(c) { return }

	allAccounts, err := odin.FindAll("Accounts", func() interface{} {
		return &accounts.Account{}
	})
	
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch accounts"})
		return
	}

	var safeAccounts []map[string]interface{}
	for _, accData := range allAccounts {
		acc, ok := accData.(*accounts.Account)
		if !ok { continue }
		
		safeAccounts = append(safeAccounts, map[string]interface{}{
			"id":          acc.ID,
			"avatar":      acc.ProfilePicture,
			"displayName": acc.DisplayName,
			"email":       acc.Email,
			"banned":      acc.Banned,
		})
	}

	c.JSON(200, safeAccounts)
}

func GETAdminAccountLookup(c *gin.Context) {
	if !checkAdminAuth(c) { return }

	identifier := c.Param("identifier")
	if identifier == "" {
		c.JSON(400, gin.H{"error": "missing identifier"})
		return
	}

	accountId, err := resolveAccountID(identifier)
	if err != nil || accountId == "" {
		c.JSON(404, gin.H{"error": "Account not found"})
		return
	}

	c.JSON(200, gin.H{"id": accountId})
}

func resolveAccountID(identifier string) (string, error) {
	if len(identifier) == 32 || len(identifier) == 34 {
		return identifier, nil
	}
	var acc accounts.Account
	err := odin.Find("Accounts", identifier, &acc)
	if err == nil && acc.ID != "" {
		return acc.ID, nil
	}
	found, err := odin.FindWhere("Accounts", map[string]interface{}{
		"display_name": identifier,
	}, func() interface{} { return &accounts.Account{} })
	
	if err != nil || len(found) == 0 {
		return "", err
	}
	return found[0].(*accounts.Account).ID, nil
}

func POSTGrantVBucks(c *gin.Context) {
	if !checkAdminAuth(c) { return }

	identifier := c.Param("accountId")
	amountStr := c.Param("amount")
	if identifier == "" {
		c.JSON(400, gin.H{"error": "missing accountId"})
		return
	}

	accountId, err := resolveAccountID(identifier)
	if err != nil || accountId == "" {
		c.JSON(404, gin.H{"error": "Account not found by that Username"})
		return
	}

	amount := 0
	for _, ch := range amountStr {
		if ch >= '0' && ch <= '9' {
			amount = amount*10 + int(ch-'0')
		}
	}
	if amount < 0 {
		amount = 13500
	}

	profileKey := accountId + ":common_core"
	var profile accounts.Profile
	if err := odin.Find("Accounts_Profiles", profileKey, &profile); err != nil {
		profile = accounts.Profile{
			Bucket: odin.Bucket{ID: profileKey},
			Items: make(map[string]interface{}),
		}
	}

	if profile.Items == nil {
		profile.Items = make(map[string]interface{})
	}

	profile.Items["Currency:MtxPurchased"] = map[string]interface{}{
		"templateId": "Currency:MtxPurchased",
		"attributes": map[string]interface{}{
			"platform": "EpicPC",
			"level":    1,
		},
		"quantity": amount,
	}

	profile.Bucket.Save(profile)
	c.JSON(200, gin.H{"message": "V-Bucks granted", "amount": amount})
}

func DELETEFullLocker(c *gin.Context) {
	if !checkAdminAuth(c) { return }

	identifier := c.Param("accountId")
	accountId, err := resolveAccountID(identifier)
	if err != nil || accountId == "" {
		c.JSON(404, gin.H{"error": "Account not found by that Username"})
		return
	}

	rewards, err := odin.FindWhere("Accounts_Rewards", map[string]interface{}{
		"account_id": accountId,
	}, func() interface{} { return &accounts.AccountReward{} })
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	removed := 0
	for _, r := range rewards {
		reward := r.(*accounts.AccountReward)
		for _, item := range reward.Rewards {
			if item == "Full Locker" {
				reward.Bucket.Delete(reward)
				removed++
				break
			}
		}
	}
	c.JSON(200, gin.H{"removed": removed})
}

func POSTGrantFullLocker(c *gin.Context) {
	if !checkAdminAuth(c) { return }

	identifier := c.Param("accountId")
	if identifier == "" {
		c.JSON(400, gin.H{"error": "missing accountId"})
		return
	}

	accountId, err := resolveAccountID(identifier)
	if err != nil || accountId == "" {
		c.JSON(404, gin.H{"error": "account not found"})
		return
	}

	reward := &accounts.AccountReward{
		Bucket:    odin.Bucket{ID: uuid.New().String()},
		AccountID: accountId,
		CodeID:    "admin-fulllocker",
		Rewards:   []string{"Full Locker"},
		Redeemed:  true,
	}

	if err := odin.Create(reward); err != nil {
		c.JSON(500, gin.H{"error": "failed to grant"})
		return
	}

	c.JSON(200, gin.H{"message": "Full locker granted", "account": accountId})
}

func POSTGrantItem(c *gin.Context) {
	if !checkAdminAuth(c) { return }

	identifier := c.Param("accountId")
	templateId := c.Param("templateId")
	if identifier == "" || templateId == "" {
		c.JSON(400, gin.H{"error": "missing accountId or templateId"})
		return
	}
	
	accountId, err := resolveAccountID(identifier)
	if err != nil || accountId == "" {
		c.JSON(404, gin.H{"error": "Account not found by that Username"})
		return
	}

	athenaKey := accountId + ":athena"
	var athena accounts.Profile
	if err := odin.Find("Accounts_Profiles", athenaKey, &athena); err != nil {
		athena = accounts.Profile{
			Bucket: odin.Bucket{ID: athenaKey},
			Items: make(map[string]interface{}),
		}
	}
	
	if athena.Items == nil {
		athena.Items = make(map[string]interface{})
	}
	athena.Items[templateId] = map[string]interface{}{
		"templateId": templateId,
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
	athena.Bucket.Save(athena)
	c.JSON(200, gin.H{"message": "Item granted", "templateId": templateId})
}


func DELETEAllItems(c *gin.Context) {
	if !checkAdminAuth(c) { return }

	identifier := c.Param("accountId")
	accountId, err := resolveAccountID(identifier)
	if err != nil || accountId == "" {
		c.JSON(404, gin.H{"error": "Account not found"})
		return
	}

	rewards, _ := odin.FindWhere("Accounts_Rewards", map[string]interface{}{
		"account_id": accountId,
	}, func() interface{} { return &accounts.AccountReward{} })
	for _, r := range rewards {
		reward := r.(*accounts.AccountReward)
		for _, item := range reward.Rewards {
			if item == "Full Locker" {
				reward.Bucket.Delete(reward)
				break
			}
		}
	}

	athenaKey := accountId + ":athena"
	var athena accounts.Profile
	if err := odin.Find("Accounts_Profiles", athenaKey, &athena); err != nil {
		c.JSON(404, gin.H{"error": "Athena profile not found"})
		return
	}

	count := 0
	newItems := make(map[string]interface{})
	for k, v := range athena.Items {
		if k == "CosmeticLocker:cosmeticlocker_athena" {
			newItems[k] = v
		} else {
			count++
		}
	}
	athena.Items = newItems
	athena.Bucket.Save(athena)

	c.JSON(200, gin.H{"message": "All items wiped", "removed": count})
}
