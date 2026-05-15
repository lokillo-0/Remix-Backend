package fortnite_dedicated_server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/remixfn/xenon/modules/database/buckets/fortnite"
	"github.com/remixfn/xenon/utilities"
)

func CreateSession(c *gin.Context) {
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		utilities.Basic.BadRequest().Apply(c.Writer)
		return
	}

	id := uuid.New().String()
	id = id[0:8] + id[9:13] + id[14:18] + id[19:23] + id[24:]

	var publicPlayers []string
	var privatePlayers []string

	if pubPlayers, ok := body["publicPlayers"].([]interface{}); ok {
		for _, player := range pubPlayers {
			if playerStr, ok := player.(string); ok {
				publicPlayers = append(publicPlayers, playerStr)
			}
		}
	}

	if privPlayers, ok := body["privatePlayers"].([]interface{}); ok {
		for _, player := range privPlayers {
			if playerStr, ok := player.(string); ok {
				privatePlayers = append(privatePlayers, playerStr)
			}
		}
	}

	var attributesStr string
	if attrs, ok := body["attributes"]; ok {
		attributesBytes, err := json.Marshal(attrs)
		if err == nil {
			attributesStr = string(attributesBytes)
		}
	}

	resp, err := http.Get("http://ipwho.is/" + c.ClientIP())
	if err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}
	defer resp.Body.Close()

	var ipRes struct {
		ContinentCode string `json:"continent_code"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ipRes); err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	if ipRes.ContinentCode == "NA" {
		ipRes.ContinentCode = "NAC"
	}

	session := fortnite.Sessions{
		Bucket:                          odin.Bucket{ID: id},
		SessionId:                       id,
		ServerAddress:                   c.ClientIP(),
		LastUpdated:                     time.Now().UTC().Format(time.RFC3339),
		PlaylistName:                    "",
		OwnerId:                         body["ownerId"].(string),
		OwnerName:                       body["ownerName"].(string),
		ServerName:                      body["serverName"].(string),
		MaxPublicPlayers:                int(body["maxPublicPlayers"].(float64)),
		MaxPrivatePlayers:               int(body["maxPrivatePlayers"].(float64)),
		ShouldAdvertise:                 body["shouldAdvertise"].(bool),
		AllowJoinInProgress:             body["allowJoinInProgress"].(bool),
		IsDedicated:                     body["isDedicated"].(bool),
		UsesStats:                       body["usesStats"].(bool),
		AllowInvites:                    body["allowInvites"].(bool),
		UsesPresence:                    body["usesPresence"].(bool),
		AllowJoinViaPresence:            body["allowJoinViaPresence"].(bool),
		AllowJoinViaPresenceFriendsOnly: body["allowJoinViaPresenceFriendsOnly"].(bool),
		BuildUniqueId:                   body["buildUniqueId"].(string),
		Attributes:                      attributesStr,
		ServerPort:                      int(body["serverPort"].(float64)),
		OpenPublicPlayers:               int(body["openPublicPlayers"].(float64)),
		OpenPrivatePlayers:              int(body["openPrivatePlayers"].(float64)),
		SortWeight:                      int(body["sortWeight"].(float64)),
		Started:                         body["started"].(bool),
		PublicPlayers:                   publicPlayers,
		PrivatePlayers:                  privatePlayers,
		Stopped:                         false,
		Region:                          ipRes.ContinentCode,
	}

	if err := odin.Create(&session); err != nil {
		utilities.Matchmaking.UnknownSession().Apply(c.Writer)
		return
	}

	attributes := make(map[string]interface{})
	if session.Attributes != "" {
		json.Unmarshal([]byte(session.Attributes), &attributes)
	}

	response := map[string]interface{}{
		"id":                              session.SessionId,
		"serverAddress":                   session.ServerAddress,
		"lastUpdated":                     session.LastUpdated,
		"ownerId":                         session.OwnerId,
		"ownerName":                       session.OwnerName,
		"serverName":                      session.ServerName,
		"maxPublicPlayers":                session.MaxPublicPlayers,
		"maxPrivatePlayers":               session.MaxPrivatePlayers,
		"shouldAdvertise":                 session.ShouldAdvertise,
		"allowJoinInProgress":             session.AllowJoinInProgress,
		"isDedicated":                     session.IsDedicated,
		"usesStats":                       session.UsesStats,
		"allowInvites":                    session.AllowInvites,
		"usesPresence":                    session.UsesPresence,
		"allowJoinViaPresence":            session.AllowJoinViaPresence,
		"allowJoinViaPresenceFriendsOnly": session.AllowJoinViaPresenceFriendsOnly,
		"buildUniqueId":                   session.BuildUniqueId,
		"attributes":                      attributes,
		"serverPort":                      session.ServerPort,
		"openPublicPlayers":               session.OpenPublicPlayers,
		"openPrivatePlayers":              session.OpenPrivatePlayers,
		"sortWeight":                      session.SortWeight,
		"started":                         session.Started,
		"publicPlayers":                   session.PublicPlayers,
		"privatePlayers":                  session.PrivatePlayers,
	}

	c.JSON(http.StatusOK, response)
}

func UpdateSessionPlayers(c *gin.Context) {
	sessionId := c.Param("sessionId")
	if sessionId == "" {
		utilities.Basic.BadRequest().Apply(c.Writer)
		return
	}

	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		utilities.Basic.BadRequest().Apply(c.Writer)
		return
	}

	var session fortnite.Sessions
	if err := odin.Find("GameSessions", sessionId, &session); err != nil {
		utilities.Matchmaking.UnknownSession().Apply(c.Writer)
		return
	}

	if session.Bucket.ID == "" {
		session.Bucket.ID = sessionId
	}

	if publicPlayers, ok := body["publicPlayers"].([]any); ok {
		var formattedPublicPlayers []string
		for _, player := range publicPlayers {
			if playerStr, ok := player.(string); ok {
				formattedPublicPlayers = append(formattedPublicPlayers, playerStr)
			}
		}
		if len(formattedPublicPlayers) > 0 {
			session.PublicPlayers = formattedPublicPlayers
		} else {
			session.PublicPlayers = []string{}
		}
	}

	if privatePlayers, ok := body["privatePlayers"].([]any); ok {
		var formattedPrivatePlayers []string
		for _, player := range privatePlayers {
			if playerStr, ok := player.(string); ok {
				formattedPrivatePlayers = append(formattedPrivatePlayers, playerStr)
			}
		}
		if len(formattedPrivatePlayers) > 0 {
			session.PrivatePlayers = formattedPrivatePlayers
		} else {
			session.PrivatePlayers = []string{}
		}
	}

	session.LastUpdated = time.Now().UTC().Format(time.RFC3339)

	if err := session.Bucket.Save(&session); err != nil {
		log.Printf("Error: %v", err)
		utilities.Matchmaking.NotAllowedIngame().Apply(c.Writer)
		return
	}

	attributes := make(map[string]interface{})
	if session.Attributes != "" {
		json.Unmarshal([]byte(session.Attributes), &attributes)
	}

	response := map[string]interface{}{
		"sessionId":                       session.SessionId,
		"serverAddress":                   session.ServerAddress,
		"lastUpdated":                     session.LastUpdated,
		"ownerId":                         session.OwnerId,
		"ownerName":                       session.OwnerName,
		"serverName":                      session.ServerName,
		"maxPublicPlayers":                session.MaxPublicPlayers,
		"maxPrivatePlayers":               session.MaxPrivatePlayers,
		"shouldAdvertise":                 session.ShouldAdvertise,
		"allowJoinInProgress":             session.AllowJoinInProgress,
		"isDedicated":                     session.IsDedicated,
		"usesStats":                       session.UsesStats,
		"allowInvites":                    session.AllowInvites,
		"usesPresence":                    session.UsesPresence,
		"allowJoinViaPresence":            session.AllowJoinViaPresence,
		"allowJoinViaPresenceFriendsOnly": session.AllowJoinViaPresenceFriendsOnly,
		"buildUniqueId":                   session.BuildUniqueId,
		"attributes":                      attributes,
		"serverPort":                      session.ServerPort,
		"openPublicPlayers":               session.OpenPublicPlayers,
		"openPrivatePlayers":              session.OpenPrivatePlayers,
		"sortWeight":                      session.SortWeight,
		"started":                         session.Started,
		"publicPlayers":                   session.PublicPlayers,
		"privatePlayers":                  session.PrivatePlayers,
	}

	c.JSON(http.StatusOK, response)
}

func CreateMatchmakingServiceTicket(c *gin.Context) {
	sessionId := c.Param("sessionId")
	if sessionId == "" {
		utilities.Basic.BadRequest().Apply(c.Writer)
		return
	}

	bucketIds := c.Query("bucketIds")
	if bucketIds == "" {
		utilities.Basic.BadRequest().Apply(c.Writer)
		return
	}

	bucketId := strings.Split(bucketIds, ":")
	if len(bucketId) < 4 {
		utilities.Basic.BadRequest().Apply(c.Writer)
		return
	}

	var session fortnite.Sessions
	if err := odin.Find("GameSessions", sessionId, &session); err != nil {
		utilities.Matchmaking.UnknownSession().Apply(c.Writer)
		return
	}

	if session.Bucket.ID == "" {
		session.Bucket.ID = sessionId
	}

	session.Region = bucketId[2]
	if err := session.Bucket.Save(session); err != nil {
		log.Printf("Error: %v", err)
		utilities.Matchmaking.NotAllowedIngame().Apply(c.Writer)
		return
	}

	finalBucketIds := strings.Join(bucketId[:4], ":")

	randomBytes128 := make([]byte, 128)
	randomBytes32 := make([]byte, 32)

	if _, err := rand.Read(randomBytes128); err != nil {
		return
	}

	if _, err := rand.Read(randomBytes32); err != nil {
		return
	}

	jti := hex.EncodeToString(randomBytes32)

	now := time.Now().UTC()
	expirationTime := now.Add(240 * time.Minute)

	ua := utilities.Parse(c.GetHeader("User-Agent"))

	region := strings.TrimSpace(bucketId[2])
	if strings.Contains(strings.ToLower(region), "none") || region == "" {
		region = session.Region
	}

	payload := jwt.MapClaims{
		"bucketId":      finalBucketIds,
		"region":        region,
		"version":       ua.Build,
		"buildUniqueId": bucketId[0],
		"exp":           expirationTime.Unix(),
		"iat":           now.Unix(),
		"jti":           jti,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	tokenString, err := token.SignedString([]byte(utilities.Get[string]("jwt")))
	if err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	response := gin.H{
		"serviceUrl": utilities.Get[string]("sesh_socket"),
		"ticketType": "Xenon-Sessions",
		"payload":    sessionId,
		"signature":  tokenString,
	}

	c.JSON(http.StatusOK, response)
}

func UpdateSession(c *gin.Context) {
	sessionId := c.Param("sessionId")
	if sessionId == "" {
		utilities.Basic.BadRequest().Apply(c.Writer)
		return
	}

	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		utilities.Basic.BadRequest().Apply(c.Writer)
		return
	}

	var session fortnite.Sessions
	if err := odin.Find("GameSessions", sessionId, &session); err != nil {
		utilities.Matchmaking.UnknownSession().Apply(c.Writer)
		return
	}
	if session.Bucket.ID == "" {
		session.Bucket.ID = sessionId
	}

	session.LastUpdated = time.Now().UTC().Format(time.RFC3339)

	if v, ok := body["ownerId"].(string); ok {
		session.OwnerId = v
	}
	if v, ok := body["ownerName"].(string); ok {
		session.OwnerName = v
	}
	if v, ok := body["serverName"].(string); ok {
		session.ServerName = v
	}
	if v, ok := body["maxPublicPlayers"].(float64); ok {
		session.MaxPublicPlayers = int(v)
	}
	if v, ok := body["maxPrivatePlayers"].(float64); ok {
		session.MaxPrivatePlayers = int(v)
	}
	if v, ok := body["shouldAdvertise"].(bool); ok {
		session.ShouldAdvertise = v
	}
	if v, ok := body["allowJoinInProgress"].(bool); ok {
		session.AllowJoinInProgress = v
	}
	if v, ok := body["isDedicated"].(bool); ok {
		session.IsDedicated = v
	}
	if v, ok := body["usesStats"].(bool); ok {
		session.UsesStats = v
	}
	if v, ok := body["allowInvites"].(bool); ok {
		session.AllowInvites = v
	}
	if v, ok := body["usesPresence"].(bool); ok {
		session.UsesPresence = v
	}
	if v, ok := body["allowJoinViaPresence"].(bool); ok {
		session.AllowJoinViaPresence = v
	}
	if v, ok := body["allowJoinViaPresenceFriendsOnly"].(bool); ok {
		session.AllowJoinViaPresenceFriendsOnly = v
	}
	if v, ok := body["buildUniqueId"].(string); ok {
		session.BuildUniqueId = v
	}
	if attrs, ok := body["attributes"]; ok {
		if attributesBytes, err := json.Marshal(attrs); err == nil {
			session.Attributes = string(attributesBytes)
		}
	}
	if v, ok := body["serverPort"].(float64); ok {
		session.ServerPort = int(v)
	}
	if v, ok := body["openPublicPlayers"].(float64); ok {
		session.OpenPublicPlayers = int(v)
	}
	if v, ok := body["openPrivatePlayers"].(float64); ok {
		session.OpenPrivatePlayers = int(v)
	}
	if v, ok := body["sortWeight"].(float64); ok {
		session.SortWeight = int(v)
	}
	if v, ok := body["started"].(bool); ok {
		session.Started = v
	}
	if publicPlayers, ok := body["publicPlayers"].([]interface{}); ok {
		var formattedPublicPlayers []string
		for _, player := range publicPlayers {
			if playerStr, ok := player.(string); ok {
				formattedPublicPlayers = append(formattedPublicPlayers, playerStr)
			}
		}
		session.PublicPlayers = formattedPublicPlayers
	}
	if privatePlayers, ok := body["privatePlayers"].([]interface{}); ok {
		var formattedPrivatePlayers []string
		for _, player := range privatePlayers {
			if playerStr, ok := player.(string); ok {
				formattedPrivatePlayers = append(formattedPrivatePlayers, playerStr)
			}
		}
		session.PrivatePlayers = formattedPrivatePlayers
	}

	if err := session.Bucket.Save(session); err != nil {
		log.Printf("Error: %v", err)
		utilities.Matchmaking.NotAllowedIngame().Apply(c.Writer)
		return
	}

	attributes := make(map[string]interface{})
	if session.Attributes != "" {
		json.Unmarshal([]byte(session.Attributes), &attributes)
	}

	response := map[string]interface{}{
		"sessionId":                       session.SessionId,
		"serverAddress":                   session.ServerAddress,
		"lastUpdated":                     session.LastUpdated,
		"ownerId":                         session.OwnerId,
		"ownerName":                       session.OwnerName,
		"serverName":                      session.ServerName,
		"maxPublicPlayers":                session.MaxPublicPlayers,
		"maxPrivatePlayers":               session.MaxPrivatePlayers,
		"shouldAdvertise":                 session.ShouldAdvertise,
		"allowJoinInProgress":             session.AllowJoinInProgress,
		"isDedicated":                     session.IsDedicated,
		"usesStats":                       session.UsesStats,
		"allowInvites":                    session.AllowInvites,
		"usesPresence":                    session.UsesPresence,
		"allowJoinViaPresence":            session.AllowJoinViaPresence,
		"allowJoinViaPresenceFriendsOnly": session.AllowJoinViaPresenceFriendsOnly,
		"buildUniqueId":                   session.BuildUniqueId,
		"attributes":                      attributes,
		"serverPort":                      session.ServerPort,
		"openPublicPlayers":               session.OpenPublicPlayers,
		"openPrivatePlayers":              session.OpenPrivatePlayers,
		"sortWeight":                      session.SortWeight,
		"started":                         session.Started,
		"publicPlayers":                   session.PublicPlayers,
		"privatePlayers":                  session.PrivatePlayers,
	}

	c.JSON(http.StatusOK, response)
}

func GameSessionHeartbeat(c *gin.Context) {
	sessionId := c.Param("sessionId")
	if sessionId == "" {
		utilities.Basic.BadRequest().Apply(c.Writer)
		return
	}

	var session fortnite.Sessions
	if err := odin.Find("GameSessions", sessionId, &session); err != nil {
		utilities.Matchmaking.UnknownSession().Apply(c.Writer)
		return
	}

	if session.Bucket.ID == "" {
		session.Bucket.ID = sessionId
	}

	session.LastUpdated = time.Now().UTC().Format(time.RFC3339)

	if err := session.Bucket.Save(&session); err != nil {
		log.Printf("Error: %v", err)
		utilities.Matchmaking.NotAllowedIngame().Apply(c.Writer)
		return
	}

	attributes := make(map[string]interface{})
	if session.Attributes != "" {
		json.Unmarshal([]byte(session.Attributes), &attributes)
	}

	response := map[string]interface{}{
		"sessionId":                       session.SessionId,
		"serverAddress":                   session.ServerAddress,
		"lastUpdated":                     session.LastUpdated,
		"ownerId":                         session.OwnerId,
		"ownerName":                       session.OwnerName,
		"serverName":                      session.ServerName,
		"maxPublicPlayers":                session.MaxPublicPlayers,
		"maxPrivatePlayers":               session.MaxPrivatePlayers,
		"shouldAdvertise":                 session.ShouldAdvertise,
		"allowJoinInProgress":             session.AllowJoinInProgress,
		"isDedicated":                     session.IsDedicated,
		"usesStats":                       session.UsesStats,
		"allowInvites":                    session.AllowInvites,
		"usesPresence":                    session.UsesPresence,
		"allowJoinViaPresence":            session.AllowJoinViaPresence,
		"allowJoinViaPresenceFriendsOnly": session.AllowJoinViaPresenceFriendsOnly,
		"buildUniqueId":                   session.BuildUniqueId,
		"attributes":                      attributes,
		"serverPort":                      session.ServerPort,
		"openPublicPlayers":               session.OpenPublicPlayers,
		"openPrivatePlayers":              session.OpenPrivatePlayers,
		"sortWeight":                      session.SortWeight,
		"started":                         session.Started,
		"publicPlayers":                   session.PublicPlayers,
		"privatePlayers":                  session.PrivatePlayers,
	}

	c.JSON(http.StatusOK, response)
}

func StartGameSession(c *gin.Context) {
	sessionId := c.Param("sessionId")
	if sessionId == "" {
		utilities.Basic.BadRequest().Apply(c.Writer)
		return
	}

	var session fortnite.Sessions
	if err := odin.Find("GameSessions", sessionId, &session); err != nil {
		utilities.Matchmaking.UnknownSession().Apply(c.Writer)
		return
	}

	if session.Bucket.ID == "" {
		session.Bucket.ID = sessionId
	}

	session.Started = true

	if err := session.Bucket.Save(&session); err != nil {
		log.Printf("Error: %v", err)
		utilities.Matchmaking.NotAllowedIngame().Apply(c.Writer)
		return
	}

	attributes := make(map[string]interface{})
	if session.Attributes != "" {
		json.Unmarshal([]byte(session.Attributes), &attributes)
	}

	response := map[string]interface{}{
		"sessionId":                       session.SessionId,
		"serverAddress":                   session.ServerAddress,
		"lastUpdated":                     session.LastUpdated,
		"ownerId":                         session.OwnerId,
		"ownerName":                       session.OwnerName,
		"serverName":                      session.ServerName,
		"maxPublicPlayers":                session.MaxPublicPlayers,
		"maxPrivatePlayers":               session.MaxPrivatePlayers,
		"shouldAdvertise":                 session.ShouldAdvertise,
		"allowJoinInProgress":             session.AllowJoinInProgress,
		"isDedicated":                     session.IsDedicated,
		"usesStats":                       session.UsesStats,
		"allowInvites":                    session.AllowInvites,
		"usesPresence":                    session.UsesPresence,
		"allowJoinViaPresence":            session.AllowJoinViaPresence,
		"allowJoinViaPresenceFriendsOnly": session.AllowJoinViaPresenceFriendsOnly,
		"buildUniqueId":                   session.BuildUniqueId,
		"attributes":                      attributes,
		"serverPort":                      session.ServerPort,
		"openPublicPlayers":               session.OpenPublicPlayers,
		"openPrivatePlayers":              session.OpenPrivatePlayers,
		"sortWeight":                      session.SortWeight,
		"started":                         session.Started,
		"publicPlayers":                   session.PublicPlayers,
		"privatePlayers":                  session.PrivatePlayers,
	}

	c.JSON(http.StatusOK, response)
}

func StopGameSession(c *gin.Context) {
	sessionId := c.Param("sessionId")
	if sessionId == "" {
		utilities.Basic.BadRequest().Apply(c.Writer)
		return
	}

	var session fortnite.Sessions
	if err := odin.Find("GameSessions", sessionId, &session); err != nil {
		utilities.Matchmaking.UnknownSession().Apply(c.Writer)
		return
	}

	if session.Bucket.ID == "" {
		session.Bucket.ID = sessionId
	}

	session.Bucket.Delete(session)

	attributes := make(map[string]interface{})
	if session.Attributes != "" {
		json.Unmarshal([]byte(session.Attributes), &attributes)
	}

	response := map[string]interface{}{
		"sessionId":                       session.SessionId,
		"serverAddress":                   session.ServerAddress,
		"lastUpdated":                     session.LastUpdated,
		"ownerId":                         session.OwnerId,
		"ownerName":                       session.OwnerName,
		"serverName":                      session.ServerName,
		"maxPublicPlayers":                session.MaxPublicPlayers,
		"maxPrivatePlayers":               session.MaxPrivatePlayers,
		"shouldAdvertise":                 session.ShouldAdvertise,
		"allowJoinInProgress":             session.AllowJoinInProgress,
		"isDedicated":                     session.IsDedicated,
		"usesStats":                       session.UsesStats,
		"allowInvites":                    session.AllowInvites,
		"usesPresence":                    session.UsesPresence,
		"allowJoinViaPresence":            session.AllowJoinViaPresence,
		"allowJoinViaPresenceFriendsOnly": session.AllowJoinViaPresenceFriendsOnly,
		"buildUniqueId":                   session.BuildUniqueId,
		"attributes":                      attributes,
		"serverPort":                      session.ServerPort,
		"openPublicPlayers":               session.OpenPublicPlayers,
		"openPrivatePlayers":              session.OpenPrivatePlayers,
		"sortWeight":                      session.SortWeight,
		"started":                         session.Started,
		"publicPlayers":                   session.PublicPlayers,
		"privatePlayers":                  session.PrivatePlayers,
	}

	c.JSON(http.StatusOK, response)
}
