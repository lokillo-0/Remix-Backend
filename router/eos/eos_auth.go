package eos

import (
	"encoding/base64"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/remixfn/xenon/utilities"
)

func getForm(v url.Values, key, fallback string) string {
	val := v.Get(key)
	if val == "" {
		return fallback
	}
	return val
}

func GETEOSAuth(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		c.String(400, "invalid form")
		return
	}

	body := c.Request.PostForm

	grantType := body.Get("grant_type")
	deploymentID := getForm(body, "deployment_id", "62a9473a2dca46b29ccf17577fcf42d7")
	nonce := getForm(body, "nonce", "")

	var features = []string{
		"AntiCheat", "Connect", "ContentService", "Ecom",
		"EpicConnect", "Inventories", "LockerService",
		"Matchmaking Service", "ExchangeCodeCreation",
		"Achievements", "Leaderboards", "Matchmaking",
		"Metrics", "PlayerReports", "Sanctions",
		"Stats", "TitleStorage", "Voice",
		"CommerceService", "FNResonanceService",
		"MagpieService", "PCBService", "QuestService",
	}

	sign := func(claims gin.H) string {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(claims))
		token.Header["kid"] = "2022-06-14T06:17:57.047928700Z"
		signed, _ := token.SignedString([]byte(utilities.GetConfig().JWTSecret))
		return signed
	}

	switch grantType {

	case "client_credentials":
		token := sign(gin.H{
			"clientId":       "3e13c5c57f594a578abe516eecb673fe",
			"productId":      "3fd15bc288014f698cca1a3d1f01c7af",
			"iss":            "eos",
			"env":            "prod",
			"organizationId": "o-aa83a0a9bc45e98c80c1b1c9d92e9e",
			"features":       features,
			"deploymentId":   deploymentID,
			"sandboxId":      "183b8244f3d84e71a4d4af08a17f7d9a",
			"tokenType":      "clientToken",
			"exp":            2147483647,
			"iat":            time.Now().Unix(),
			"jti":            uuid.NewString(),
		})

		c.JSON(200, gin.H{
			"access_token":    token,
			"token_type":      "bearer",
			"expires_at":      "9999-12-31T23:59:59.999Z",
			"features":        features,
			"organization_id": "o-aa83a0a9bc45e98c80c1b1c9d92e9e",
			"product_id":      "prod-fn",
			"sandbox_id":      "fn",
			"deployment_id":   deploymentID,
			"expires_in":      3599,
		})
		return

	case "external_auth":
		extToken := getForm(body, "external_auth_token", "")

		var displayName, sub string

		if extToken != "" {
			token, _, err := new(jwt.Parser).ParseUnverified(extToken, jwt.MapClaims{})
			if err == nil && token != nil {
				if claims, ok := token.Claims.(jwt.MapClaims); ok {
					displayName, _ = claims["dn"].(string)
					sub, _ = claims["sub"].(string)
				}
			}
		}

		accessToken := sign(gin.H{
			"clientId":           "ec684b8c687f479fadea3cb2ad83f5c6",
			"role":               "GameClient",
			"productId":          "prod-fn",
			"iss":                "eos",
			"env":                "prod",
			"nonce":              nonce,
			"organizationId":     "o-aa83a0a9bc45e98c80c1b1c9d92e9e",
			"features":           features,
			"productUserId":      sub,
			"organizationUserId": "000185f80b9a4dc3aaf1ca83611c2bf5",
			"clientIp":           c.ClientIP(),
			"deploymentId":       deploymentID,
			"sandboxId":          "fn",
			"tokenType":          "userToken",
			"exp":                2147483647,
			"iat":                time.Now().Unix(),
			"jti":                uuid.NewString(),

			"account": gin.H{
				"idp":         "epicgames",
				"displayName": displayName,
				"id":          sub,
				"plf":         "other",
			},
		})

		idToken := sign(gin.H{
			"aud":   "ec684b8c687f479fadea3cb2ad83f5c6",
			"sub":   sub,
			"pfsid": "fn",
			"act": gin.H{
				"pltfm": "other",
				"eaid":  displayName,
				"eat":   "epicgames",
			},
			"pfdid":     deploymentID,
			"iss":       "http://remix/auth/v1/oauth",
			"exp":       2147483647,
			"iat":       time.Now().Unix(),
			"tokenType": "idToken",
			"pfpid":     "prod-fn",
		})

		c.JSON(200, gin.H{
			"access_token":            accessToken,
			"token_type":              "bearer",
			"expires_at":              "9999-12-31T23:59:59.999Z",
			"nonce":                   nonce,
			"features":                features,
			"organization_id":         "o-aa83a0a9bc45e98c80c1b1c9d92e9e",
			"product_id":              "prod-fn",
			"sandbox_id":              "fn",
			"deployment_id":           deploymentID,
			"organization_user_id":    "000185f80b9a4dc3aaf1ca83611c2bf5",
			"product_user_id":         sub,
			"product_user_id_created": false,
			"id_token":                idToken,
			"expires_in":              3599,
		})
		return

	case "refresh_token":
		refresh := getForm(body, "refresh_token", "")

		token, _, _ := new(jwt.Parser).ParseUnverified(refresh, jwt.MapClaims{})
		claims := token.Claims.(jwt.MapClaims)

		displayName, _ := claims["dn"].(string)
		sub, _ := claims["sub"].(string)

		newToken := sign(gin.H{
			"sub":   sub,
			"pfsid": "fn",
			"iss":   "http://remix/epic/oauth/v2",
			"dn":    displayName,
			"pfpid": "prod-fn",
			"aud":   "ec684b8c687f479fadea3cb2ad83f5c6",
			"pfdid": deploymentID,
			"t":     "epic_id_r",
			"appid": "fghi4567FNFBKFz3E4TROb0bmPS8h1GW",
			"scope": "basic_profile friends_list openid offline_access presence",
			"iat":   time.Now().Unix(),
			"exp":   2147483647,
			"jti":   uuid.NewString(),
		})

		idToken := sign(gin.H{
			"aud":   "ec684b8c687f479fadea3cb2ad83f5c6",
			"sub":   sub,
			"pfsid": "fn",
			"act": gin.H{
				"pltfm": "other",
				"eaid":  displayName,
				"eat":   "epicgames",
			},
			"pfdid":     deploymentID,
			"iss":       "http://remix/auth/v1/oauth",
			"exp":       2147483647,
			"iat":       time.Now().Unix(),
			"tokenType": "idToken",
			"pfpid":     "prod-fn",
		})

		c.JSON(200, gin.H{
			"access_token":       newToken,
			"expires_in":         15552000,
			"expires_at":         "9999-12-31T23:59:59.999Z",
			"token_type":         "bearer",
			"refresh_token":      newToken,
			"refresh_expires":    15552000,
			"refresh_expires_at": "9999-12-31T23:59:59.999Z",
			"account_id":         sub,
			"client_id":          "3e13c5c57f594a578abe516eecb673fe",
			"internal_client":    true,
			"client_service":     "3fd15bc288014f698cca1a3d1f01c7af",
			"scope": []string{
				"basic_profile", "friends_list", "openid",
				"offline_access", "presence",
			},
			"displayName":    displayName,
			"app":            "3fd15bc288014f698cca1a3d1f01c7af",
			"in_app_id":      sub,
			"device_id":      "Remix",
			"product_id":     "3fd15bc288014f698cca1a3d1f01c7af",
			"sandbox_id":     "fn",
			"deployment_id":  deploymentID,
			"application_id": "fghi4567UG3ZXlhvevzKJI65wfTUoYBC",
			"acr":            "urn:epic:loa:aal1",
			"auth_time":      time.Now().Format(time.RFC3339),
			"id_token":       idToken,
		})
		return
	}

	c.String(400, "unsupported grant_type")
}

func PostEpicOAuthV2Token(c *gin.Context) {
	var body struct {
		RefreshToken string `form:"refresh_token" json:"refresh_token"`
	}
	if err := c.ShouldBind(&body); err != nil {
		utilities.Internal.ValidationFailed().
			WithMessage("Invalid request body").
			WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	JWT := body.RefreshToken
	token, _, err := new(jwt.Parser).ParseUnverified(JWT, jwt.MapClaims{})
	if err != nil {
		utilities.Internal.ValidationFailed().
			WithMessage("Invalid refresh token").
			WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		utilities.Internal.ValidationFailed().
			WithMessage("Invalid token claims").
			WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	email, _ := claims["sub"].(string)
	scope := c.Query("scope")
	if scope == "" {
		scope = "basic_profile friends_list openid presence"
	}

	ua := c.GetHeader("User-Agent")
	var clid string
	if strings.Contains(strings.ToLower(ua), "switch") {
		clid = "5229dcd3ac3845208b496649092f251b"
	} else {
		clid = "ec684b8c687f479fadea3cb2ad83f5c6"
	}

	accessClaims := jwt.MapClaims{
		"sub":   email,
		"pfsid": "fn",
		"iss":   "https://api.epicgames.dev/epic/oauth/v1",
		"dn":    email,
		"nonce": "n-01/jkXYh/9P5JimUEpSisDyK3Xw=",
		"pfpid": "prod-fn",
		"sec":   1,
		"aud":   clid,
		"pfdid": "62a9473a2dca46b29ccf17577fcf42d7",
		"t":     "epic_id",
		"scope": scope,
		"appid": "fghi4567FNFBKFz3E4TROb0bmPS8h1GW",
		"exp":   9668536326,
		"iat":   1668529126,
		"jti":   "c01f29504dcd42f9b68cf55759392928",
	}
	refreshClaims := jwt.MapClaims{
		"sub":   email,
		"pfsid": "fn",
		"iss":   "https://api.epicgames.dev/epic/oauth/v1",
		"dn":    email,
		"pfpid": "prod-fn",
		"aud":   clid,
		"pfdid": "62a9473a2dca46b29ccf17577fcf42d7",
		"t":     "epic_id",
		"appid": "fghi4567FNFBKFz3E4TROb0bmPS8h1GW",
		"scope": scope,
		"exp":   9668557926,
		"iat":   1668529126,
		"jti":   "c01f29504dcd42f9b68cf55759392928",
	}
	idClaims := jwt.MapClaims{
		"sub":   email,
		"pfsid": "fn",
		"iss":   "https://api.epicgames.dev/epic/oauth/v1",
		"dn":    email,
		"nonce": "n-e3Kcqw0hulXkbebFRBL8o5AwL3M=",
		"pfpid": "prod-fn",
		"aud":   clid,
		"pfdid": "62a9473a2dca46b29ccf17577fcf42d7",
		"t":     "id_token",
		"appid": "fghi4567FNFBKFz3E4TROb0bmPS8h1GW",
		"exp":   9668536326,
		"iat":   1668529126,
		"jti":   "c01f29504dcd42f9b68cf55759392928",
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenStr, err := accessToken.SignedString([]byte(utilities.GetConfig().JWTSecret))
	if err != nil {
		utilities.Internal.ValidationFailed().
			WithMessage("Failed to sign access token").
			WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenStr, err := refreshToken.SignedString([]byte(utilities.GetConfig().JWTSecret))
	if err != nil {
		utilities.Internal.ValidationFailed().
			WithMessage("Failed to sign refresh token").
			WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}
	idToken := jwt.NewWithClaims(jwt.SigningMethodHS256, idClaims)
	idTokenStr, err := idToken.SignedString([]byte(utilities.GetConfig().JWTSecret))
	if err != nil {
		utilities.Internal.ValidationFailed().
			WithMessage("Failed to sign id token").
			WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	c.JSON(200, gin.H{
		"scope":               "basic_profile friends_list openid presence",
		"token_type":          "bearer",
		"acr":                 "AAL1",
		"access_token":        accessTokenStr,
		"expires_in":          7200,
		"expires_at":          "9999-12-31T23:59:59.999Z",
		"refresh_token":       refreshTokenStr,
		"refresh_expires_in":  28800,
		"refresh_expires_at":  "9999-12-31T23:59:59.999Z",
		"account_id":          email,
		"client_id":           clid,
		"application_id":      "fghi4567FNFBKFz3E4TROb0bmPS8h1GW",
		"selected_account_id": email,
		"id_token":            idTokenStr,
		"auth_time":           time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	})
}

func GetEpicOAuthV2TokenInfo(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(400, gin.H{"error": "Missing Authorization header"})
		return
	}

	base64String := strings.TrimPrefix(authHeader, "Basic ")
	if base64String == authHeader {
		c.JSON(400, gin.H{"error": "Invalid Authorization header format"})
		return
	}

	decodedBytes, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid base64 encoding"})
		return
	}

	credentials := strings.Split(string(decodedBytes), ":")
	if len(credentials) != 2 {
		c.JSON(400, gin.H{"error": "Invalid credentials format"})
		return
	}

	c.JSON(200, gin.H{
		"active":         true,
		"scope":          "basic_profile openid offline_access",
		"token_type":     "bearer",
		"expires_in":     2147483647,
		"expires_at":     "9999-12-31T23:59:59.999Z",
		"account_id":     "skid",
		"client_id":      credentials[0],
		"application_id": "fghi45672f0QV6b6B1KntLd7JR7RFLWc",
	})
}
