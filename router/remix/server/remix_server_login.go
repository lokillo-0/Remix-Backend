package remix_server

import (
	"encoding/base64"
	"math/rand"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
	"golang.org/x/crypto/bcrypt"
)

func POSTRemixServerLogin(c *gin.Context) {
	var request struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.String(400, "Invalid request body: %s", err.Error())
		return
	}

	existingAccounts, err := odin.FindWhere("Accounts", map[string]interface{}{
		"email": request.Email,
	}, func() interface{} {
		return &accounts.Account{}
	})

	if err != nil || len(existingAccounts) == 0 {
		c.String(401, "failed to find account!")
		return
	}

	account := existingAccounts[0].(*accounts.Account)

	if account.Banned {
		c.String(403, "Account is banned!")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(request.Password))
	if err != nil {
		c.String(500, "failed to compare password hash!")
		return
	}

	account.LastLoginTime = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	account.LastLoginIP = c.ClientIP()

	err = account.Bucket.Save(account)
	if err != nil {
		c.String(499, "failed to update login session!")
		return
	}

	response := map[string]interface{}{
		"id":                        account.ID,
		"created":                   account.Created,
		"email":                     account.Email,
		"display_name":              account.DisplayName,
		"banned":                    account.Banned,
		"roles":                     account.Roles,
		"ban_history":               account.BanHistory,
		"is_server":                 account.IsServer,
		"last_login_time":           account.LastLoginTime,
		"last_login_ip":             account.LastLoginIP,
		"profile_picture":           account.ProfilePicture,
		"display_name_changes":      account.DisplayNameChanges,
		"matchmaking_banned_until":  account.MatchmakingBannedUntil,
		"matchmaking_banned_since":  account.MatchmakingBannedSince,
		"last_display_name_change":  account.LastDisplayNameChange,
		"matchmaking_banned_reason": account.MatchmakingBannedReason,
	}

	payload := jwt.MapClaims{
		"app":           "Remix",
		"sub":           account.ID,
		"dvid":          rand.Intn(1000000000),
		"mver":          false,
		"dn":            account.DisplayName,
		"am":            "access",
		"p":             base64.StdEncoding.EncodeToString([]byte(uuid.New().String())),
		"iai":           account.ID,
		"sec":           1,
		"clsvc":         "remix",
		"t":             "s",
		"ic":            true,
		"jti":           uuid.New().String(),
		"creation_date": time.Now().UTC().Format(time.RFC3339),
		"expires_in":    4 * 3600,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	signedToken, err := token.SignedString([]byte(utilities.Get[string]("jwt")))
	if err != nil {
		c.String(477, "failed to create access token!")
		return
	}

	session := &accounts.Session{
		Bucket:    odin.Bucket{ID: account.ID},
		AccountID: account.ID,
		Token:     signedToken,
		Type:      "login",
	}

	err = odin.Create(session)
	if err != nil {
		c.String(488, "failed to create session!")
		return
	}

	c.JSON(200, gin.H{
		"response": response,
		"auth":     signedToken,
	})
}
