package oauth

import (
	crand "crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/andr1ww/odin"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/modules/database/buckets/fortnite"
	"github.com/remixfn/xenon/utilities"
)

func PostOAuthToken(c *gin.Context) {
	tokenHeader := c.GetHeader("Authorization")
	if tokenHeader == "" {
		utilities.Authentication.InvalidHeader().Apply(c.Writer)
		return
	}

	client := strings.Split(tokenHeader, " ")

	err := c.Request.ParseForm()
	if err != nil {
		utilities.Authentication.InvalidRequest().Apply(c.Writer)
		return
	}

	body := make(map[string]string)
	for key, values := range c.Request.PostForm {
		if key != "" {
			body[key] = values[0]
		}
	}

	grantType := body["grant_type"]
	if grantType == "" {
		utilities.Authentication.OAuth.UnsupportedGrant().Apply(c.Writer)
		return
	}

	var user *accounts.Account
	switch grantType {
	case "device_auth":
		accountId := body["account_id"]
		secret := body["secret"]
		deviceId := body["device_id"]
		if accountId == "" {
			utilities.Authentication.OAuth.InvalidBody().Apply(c.Writer)
			return
		}

		var foundUser accounts.Account
		err := odin.Find("Accounts", accountId, &foundUser)
		if err != nil {
			utilities.Account.AccountNotFound().Apply(c.Writer)
			return
		}

		user = &foundUser
		if user.Banned {
			utilities.Account.DisabledAccount().Apply(c.Writer)
			return
		}

		if secret != user.DeviceID {
			utilities.Authentication.NotOwnSessionRemoval().Apply(c.Writer)
			return
		}

		if deviceId != user.DeviceID {
			utilities.Authentication.NotOwnSessionRemoval().Apply(c.Writer)
			return
		}

	case "authorization_code":
		code := body["code"]
		if code == "" {
			utilities.Authentication.OAuth.InvalidExchange().Apply(c.Writer)
			return
		}

		exchangeCodes, err := odin.FindWhere("ExchangeCodes", map[string]interface{}{
			"code": code,
		}, func() interface{} {
			return &fortnite.Exchange{}
		})

		if err != nil || len(exchangeCodes) == 0 {
			utilities.Authentication.OAuth.InvalidExchange().Apply(c.Writer)
			return
		}
		exchange := exchangeCodes[0].(*fortnite.Exchange)
		if exchange.AccountID == "" {
			utilities.Authentication.OAuth.InvalidExchange().Apply(c.Writer)
			return
		}
		created, parseErr := time.Parse(time.RFC3339, exchange.Created)
		if parseErr != nil || time.Since(created) > 5*time.Minute {
			exchange.Delete(exchange)
			utilities.Authentication.OAuth.InvalidExchange().Apply(c.Writer)
			return
		}
		var foundUser accounts.Account
		err = odin.Find("Accounts", exchange.AccountID, &foundUser)
		if err != nil {
			utilities.Account.AccountNotFound().Apply(c.Writer)
			return
		}
		user = &foundUser
		if user.Banned {
			utilities.Account.DisabledAccount().Apply(c.Writer)
			return
		}
		exchange.Delete(exchange)
	case "exchange_code":
		code := body["exchange_code"]
		if code == "" {
			utilities.Authentication.OAuth.InvalidExchange().Apply(c.Writer)
			return
		}

		exchangeCodes, err := odin.FindWhere("ExchangeCodes", map[string]interface{}{
			"code": code,
		}, func() interface{} {
			return &fortnite.Exchange{}
		})

		if err != nil || len(exchangeCodes) == 0 {
			utilities.Authentication.OAuth.InvalidExchange().Apply(c.Writer)
			return
		}
		exchange := exchangeCodes[0].(*fortnite.Exchange)
		if exchange.AccountID == "" {
			utilities.Authentication.OAuth.InvalidExchange().Apply(c.Writer)
			return
		}
		created, parseErr := time.Parse(time.RFC3339, exchange.Created)
		if parseErr != nil || time.Since(created) > 5*time.Minute {
			exchange.Delete(exchange)
			utilities.Authentication.OAuth.InvalidExchange().Apply(c.Writer)
			return
		}
		var foundUser accounts.Account
		err = odin.Find("Accounts", exchange.AccountID, &foundUser)
		if err != nil {
			utilities.Account.AccountNotFound().Apply(c.Writer)
			return
		}
		user = &foundUser
		if user.Banned {
			utilities.Account.DisabledAccount().Apply(c.Writer)
			return
		}
		exchange.Delete(exchange)

	case "password":
		username := body["username"]
		password := body["password"]

		if username == "" {
			utilities.Authentication.OAuth.InvalidAccountCredentials().Apply(c.Writer)
			return
		}

		if strings.Contains(username, "@project.remix") {
			displayName := username[:strings.Index(username, "@")]
			accs, err := odin.FindWhere("Accounts", map[string]interface{}{
				"username": displayName,
			}, func() interface{} { return &accounts.Account{} })
			if err != nil || len(accs) == 0 {
				accs, err = odin.FindWhere("Accounts", map[string]interface{}{
					"display_name": displayName,
				}, func() interface{} { return &accounts.Account{} })
			}
			if err != nil || len(accs) == 0 {
				utilities.Authentication.OAuth.InvalidAccountCredentials().Apply(c.Writer)
				return
			}
			found := accs[0].(*accounts.Account)
			if found.Username == "" {
				found.Username = found.DisplayName
				found.Bucket.Save(*found)
			}
			if found.Banned {
				utilities.Account.DisabledAccount().Apply(c.Writer)
				return
			}
			user = found

			if !user.IsServer {
				// beta role gate
				cfg := utilities.GetConfig()
				if len(cfg.BETA_ROLE_IDS) > 0 && found.DiscordID != "" {
					client2 := &http.Client{Timeout: 5 * 1000000000}
					req2, _ := http.NewRequest("GET",
						fmt.Sprintf("https://discord.com/api/v10/guilds/%s/members/%s", cfg.GUILD_ID, found.DiscordID), nil)
					req2.Header.Set("Authorization", "Bot "+cfg.DISCORD_BotToken)
					resp2, err2 := client2.Do(req2)
					hasRole := false
					if err2 == nil && resp2.StatusCode == 200 {
						body2, _ := io.ReadAll(resp2.Body)
						resp2.Body.Close()
						var mem struct {
							Roles []string `json:"roles"`
						}
						json.Unmarshal(body2, &mem)
						for _, br := range cfg.BETA_ROLE_IDS {
							for _, ur := range mem.Roles {
								if br == ur {
									hasRole = true
								}
							}
						}
					}
					if !hasRole {
						c.Status(http.StatusNotFound)
						return
					}
				}
			}

			break
		}

		users, err := odin.FindWhere("Accounts", map[string]interface{}{
			"email": username,
		}, func() interface{} {
			return &accounts.Account{}
		})
		if err != nil || len(users) == 0 {
			utilities.Authentication.OAuth.InvalidAccountCredentials().Apply(c.Writer)
			return
		}

		foundUser := users[0].(*accounts.Account)
		user = foundUser

		if user.Banned {
			utilities.Account.DisabledAccount().Apply(c.Writer)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
			utilities.Authentication.OAuth.InvalidAccountCredentials().Apply(c.Writer)
			return
		}

		if !user.IsServer {
			utilities.Authentication.OAuth.InvalidAccountCredentials().Apply(c.Writer)
			return
		}

	case "client_credentials":
		clientID, err := base64.StdEncoding.DecodeString(client[1])
		if err != nil {
			utilities.Authentication.InvalidRequest().Apply(c.Writer)
			return
		}

		randomBytes128 := make([]byte, 128)
		randomBytes32 := make([]byte, 32)
		_, err = crand.Read(randomBytes128)
		if err != nil {
			utilities.Internal.ServerError().Apply(c.Writer)
			return
		}
		_, err = crand.Read(randomBytes32)
		if err != nil {
			utilities.Internal.ServerError().Apply(c.Writer)
			return
		}

		p := base64.StdEncoding.EncodeToString(randomBytes128)
		jti := hex.EncodeToString(randomBytes32)
		now := time.Now().UTC()
		expirationTime := now.Add(240 * time.Minute)

		payload := jwt.MapClaims{
			"p":             p,
			"clsvc":         "fortnite",
			"t":             "s",
			"mver":          false,
			"clid":          string(clientID),
			"ic":            true,
			"exp":           expirationTime.Unix(),
			"am":            "client_credentials",
			"iat":           now.Unix(),
			"jti":           jti,
			"creation_date": now.Format(time.RFC3339),
			"expires_in":    14400,
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
		signedToken, err := token.SignedString([]byte(utilities.Get[string]("jwt")))
		if err != nil {
			utilities.Internal.ServerError().Apply(c.Writer)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"access_token":    signedToken,
			"expires_in":      14400,
			"expires_at":      time.Now().Add(240 * time.Minute).Format(time.RFC3339),
			"token_type":      "bearer",
			"client_id":       string(clientID),
			"internal_client": true,
			"client_service":  "fortnite",
		})
		return

	case "refresh_token":
		refreshToken := body["refresh_token"]
		if refreshToken == "" {
			utilities.Authentication.OAuth.InvalidRefresh().Apply(c.Writer)
			return
		}

		token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
			return []byte(utilities.Get[string]("jwt")), nil
		})
		if err != nil {
			utilities.Authentication.InvalidToken().Apply(c.Writer)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			utilities.Authentication.InvalidToken().Apply(c.Writer)
			return
		}
		accountID := claims["sub"].(string)

		var foundUser accounts.Account
		err = odin.Find("Accounts", accountID, &foundUser)
		if err != nil {
			utilities.Account.AccountNotFound().Apply(c.Writer)
			return
		}
		user = &foundUser

		if user.Banned {
			utilities.Account.DisabledAccount().Apply(c.Writer)
			return
		}

		fixedClientID := "ec684b8c687f479fadea3cb2ad83f5c6"

		accessToken, err := createAccessToken(fixedClientID, *user)
		if err != nil {
			utilities.Internal.ServerError().Apply(c.Writer)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"access_token":       accessToken,
			"expires_in":         3600,
			"expires_at":         time.Now().Add(time.Hour).Format(time.RFC3339),
			"token_type":         "bearer",
			"refresh_token":      body["refresh_token"],
			"refresh_expires":    86400,
			"refresh_expires_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			"account_id":         user.ID,
			"client_id":          fixedClientID,
			"internal_client":    true,
			"client_service":     "fortnite",
			"display_name":       user.DisplayName,
			"app":                "fortnite",
			"in_app_id":          user.ID,
			"device_id":          c.GetHeader("X-Epic-Device-Id"),
		})
		return

	case "epic_credentials":
		username := body["username"]
		password := body["password"]

		var foundUser *accounts.Account
		if password != "" {
			tok, err := jwt.ParseWithClaims(password, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte(utilities.Get[string]("jwt")), nil
			})
			if err == nil && tok.Valid {
				if claims, ok := tok.Claims.(jwt.MapClaims); ok {
					if sub, ok := claims["sub"].(string); ok && sub != "" {
						var acc accounts.Account
						if err := odin.Find("Accounts", sub, &acc); err == nil {
							foundUser = &acc
						}
					}
				}
			}
		}

		if foundUser == nil && username != "" {
			displayName := username
			if atIdx := strings.Index(username, "@"); atIdx >= 0 {
				displayName = username[:atIdx]
			}
			accs, err := odin.FindWhere("Accounts", map[string]interface{}{
				"username": displayName,
			}, func() interface{} { return &accounts.Account{} })
			if err == nil && len(accs) > 0 {
				foundUser = accs[0].(*accounts.Account)
			}
		}

		if foundUser == nil {
			utilities.Authentication.OAuth.InvalidAccountCredentials().Apply(c.Writer)
			return
		}
		user = foundUser

	default:

		username := body["username"]
		password := body["password"]
		var foundUser *accounts.Account

		if password != "" {
			tok, err := jwt.ParseWithClaims(password, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte(utilities.Get[string]("jwt")), nil
			})
			if err == nil && tok.Valid {
				if claims, ok := tok.Claims.(jwt.MapClaims); ok {
					if sub, ok := claims["sub"].(string); ok && sub != "" {
						var acc accounts.Account
						if err := odin.Find("Accounts", sub, &acc); err == nil {
							foundUser = &acc
						}
					}
				}
			}
		}

		if foundUser == nil && username != "" {
			displayName := username
			if atIdx := strings.Index(username, "@"); atIdx >= 0 {
				displayName = username[:atIdx]
			}
			accs, err := odin.FindWhere("Accounts", map[string]interface{}{
				"username": displayName,
			}, func() interface{} { return &accounts.Account{} })
			if err == nil && len(accs) > 0 {
				foundUser = accs[0].(*accounts.Account)
			}
		}

		if foundUser == nil {
			utilities.Authentication.OAuth.GrantNotImplemented().Apply(c.Writer)
			return
		}
		user = foundUser
	}

	if user.ID == "" {
		utilities.Authentication.OAuth.InvalidAccountCredentials().Apply(c.Writer)
		return
	}

	if !user.IsServer {
		sessions, _ := odin.FindWhere("Accounts_Sessions", map[string]interface{}{
			"type":      "access",
			"accountId": user.ID,
		}, func() interface{} { return &accounts.Session{} })
		for _, session := range sessions {
			s := session.(*accounts.Session)
			s.Delete(s)
		}

		refreshSessions, _ := odin.FindWhere("Accounts_Sessions", map[string]interface{}{
			"type":      "refresh",
			"accountId": user.ID,
		}, func() interface{} {
			return &accounts.Session{}
		})
		for _, session := range refreshSessions {
			s := session.(*accounts.Session)
			if s.ID == user.ID {
				s.Delete(s)
			}
		}
	}

	var clientIDStr string
	if len(client) > 1 {
		if decoded, decErr := base64.StdEncoding.DecodeString(client[1]); decErr == nil {
			clientIDStr = string(decoded)
		} else {
			clientIDStr = client[1]
		}
	}
	if clientIDStr == "" {
		clientIDStr = "ec684b8c687f479fadea3cb2ad83f5c6"
	}

	accessToken, err := createAccessToken(clientIDStr, *user)
	if err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	refreshToken, err := createRefreshToken(clientIDStr, *user)
	if err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":       accessToken,
		"expires_in":         86400,
		"expires_at":         time.Now().Add(time.Hour).Format(time.RFC3339),
		"token_type":         "bearer",
		"account_id":         user.ID,
		"client_id":          clientIDStr,
		"internal_client":    true,
		"client_service":     "fortnite",
		"refresh_token":      refreshToken,
		"refresh_expires":    86400,
		"refresh_expires_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"display_name":       user.DisplayName,
		"app":                "fortnite",
		"in_app_id":          user.ID,
		"device_id":          c.GetHeader("X-Epic-Device-Id"),
	})
}

func PostOAuthTokenSwitch(c *gin.Context) {
	tokenHeader := c.GetHeader("Authorization")
	if tokenHeader == "" {
		utilities.Authentication.InvalidHeader().Apply(c.Writer)
		return
	}

	client := strings.Split(tokenHeader, " ")
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		utilities.Authentication.InvalidRequest().Apply(c.Writer)
		return
	}

	var user *accounts.Account

	email := body.Email
	password := body.Password

	if email == "" || password == "" {
		utilities.Authentication.OAuth.InvalidAccountCredentials().Apply(c.Writer)
		return
	}

	users, err := odin.FindWhere("Accounts", map[string]interface{}{
		"email": email,
	}, func() interface{} {
		return &accounts.Account{}
	})
	if err != nil || len(users) == 0 {
		utilities.Authentication.OAuth.InvalidAccountCredentials().Apply(c.Writer)
		return
	}

	foundUser := users[0].(*accounts.Account)
	user = foundUser

	if user.Banned {
		utilities.Account.DisabledAccount().Apply(c.Writer)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		utilities.Authentication.OAuth.InvalidAccountCredentials().Apply(c.Writer)
		return
	}

	if !user.IsServer {
		sessions, _ := odin.FindWhere("Accounts_Sessions", map[string]interface{}{
			"type": "access",
		}, func() interface{} {
			return &accounts.Session{}
		})
		for _, session := range sessions {
			s := session.(*accounts.Session)
			if s.ID == user.ID {
				s.Delete(s)
			}
		}

		refreshSessions, _ := odin.FindWhere("Accounts_Sessions", map[string]interface{}{
			"type": "refresh",
		}, func() interface{} {
			return &accounts.Session{}
		})
		for _, session := range refreshSessions {
			s := session.(*accounts.Session)
			if s.ID == user.ID {
				s.Delete(s)
			}
		}
	}

	clientID, err := base64.StdEncoding.DecodeString(client[1])
	if err != nil {
		utilities.Authentication.InvalidRequest().Apply(c.Writer)
		return
	}

	accessToken, err := createAccessToken(string(clientID), *user)
	if err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	refreshToken, err := createRefreshToken(string(clientID), *user)
	if err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":       accessToken,
		"expires_in":         86400,
		"expires_at":         time.Now().Add(time.Hour).Format(time.RFC3339),
		"token_type":         "bearer",
		"account_id":         user.ID,
		"client_id":          string(clientID),
		"internal_client":    true,
		"client_service":     "fortnite",
		"refresh_token":      refreshToken,
		"refresh_expires":    86400,
		"refresh_expires_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"display_name":       user.DisplayName,
		"app":                "fortnite",
		"in_app_id":          user.ID,
		"device_id":          c.GetHeader("X-Epic-Device-Id"),
	})
}

func Verify(c *gin.Context) {
	tokenHeader := c.GetHeader("Authorization")

	if tokenHeader == "" {
		utilities.Authentication.InvalidHeader().Apply(c.Writer)
		return
	}

	bearerPrefix := "bearer "
	if !strings.HasPrefix(tokenHeader, bearerPrefix) {
		utilities.Authentication.InvalidHeader().Apply(c.Writer)
		return
	}

	token := strings.TrimPrefix(tokenHeader, bearerPrefix)
	accessToken := strings.Replace(token, "eg1~", "", -1)

	if accessToken == "" {
		utilities.Authentication.InvalidToken().Apply(c.Writer)
		return
	}

	sessions, err := odin.FindWhere("Accounts_Sessions", map[string]interface{}{
		"token": accessToken,
	}, func() interface{} {
		return &accounts.Session{}
	})
	if err != nil || len(sessions) == 0 {
		utilities.Authentication.InvalidToken().Apply(c.Writer)
		return
	}

	session := sessions[0].(*accounts.Session)

	var user accounts.Account
	err = odin.Find("Accounts", session.ID, &user)
	if err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	if user.Banned {
		c.JSON(http.StatusBadRequest, gin.H{})
		return
	}

	deviceID := c.GetHeader("X-Epic-Device-Id")
	if deviceID == "" {
		deviceID = uuid.New().String()
	}

	claims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(session.Token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(utilities.Get[string]("jwt")), nil
	})
	if err != nil {
		utilities.Authentication.InvalidToken().Apply(c.Writer)
		return
	}

	exp := claims["expires_in"].(float64)
	expiresAt := time.Unix(int64(exp), 0)
	expiresIn := int(time.Until(expiresAt).Seconds())

	c.JSON(http.StatusOK, gin.H{
		"token":           session.Token,
		"session_id":      claims["jti"],
		"token_type":      "bearer",
		"client_id":       claims["clid"],
		"internal_client": true,
		"client_service":  "fortnite",
		"account_id":      user.ID,
		"expires_in":      expiresIn,
		"expires_at":      expiresAt.Format(time.RFC3339),
		"auth_method":     session.Type,
		"display_name":    user.DisplayName,
		"app":             "fortnite",
		"in_app_id":       user.ID,
		"device_id":       deviceID,
	})
}

func createAccessToken(clientID string, user accounts.Account) (string, error) {
	payload := jwt.MapClaims{
		"app":           "fortnite",
		"sub":           user.ID,
		"dvid":          rand.Intn(1000000000),
		"mver":          false,
		"clid":          clientID,
		"dn":            user.DisplayName,
		"am":            "access",
		"p":             base64.StdEncoding.EncodeToString([]byte(uuid.New().String())),
		"iai":           user.ID,
		"sec":           1,
		"clsvc":         "fortnite",
		"t":             "s",
		"ic":            true,
		"jti":           uuid.New().String(),
		"creation_date": time.Now().UTC().Format(time.RFC3339),
		"expires_in":    4 * 3600,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	signedToken, err := token.SignedString([]byte(utilities.Get[string]("jwt")))
	if err != nil {
		return "", err
	}

	session := &accounts.Session{
		Bucket:    odin.Bucket{ID: uuid.NewString()},
		AccountID: user.ID,
		Token:     signedToken,
		Type:      "access",
	}

	err = odin.Create(session)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func createRefreshToken(clientID string, user accounts.Account) (string, error) {
	payload := jwt.MapClaims{
		"app":           "fortnite",
		"sub":           user.ID,
		"dvid":          rand.Intn(1000000000),
		"mver":          false,
		"clid":          clientID,
		"dn":            user.DisplayName,
		"am":            "refresh",
		"p":             base64.StdEncoding.EncodeToString([]byte(uuid.New().String())),
		"iai":           user.ID,
		"sec":           1,
		"clsvc":         "fortnite",
		"t":             "s",
		"ic":            true,
		"jti":           uuid.New().String(),
		"creation_date": time.Now().UTC().Format(time.RFC3339),
		"expires_in":    14 * 24 * 3600,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	signedToken, err := token.SignedString([]byte(utilities.Get[string]("jwt")))
	if err != nil {
		return "", err
	}

	session := &accounts.Session{
		Bucket:    odin.Bucket{ID: uuid.NewString()},
		AccountID: user.ID,
		Token:     signedToken,
		Type:      "refresh",
	}

	err = odin.Create(session)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}
