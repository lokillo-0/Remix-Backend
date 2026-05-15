package fortnite

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
)

var buildUniqueIds = make(map[string]string)
var buildUniqueIdsMutex = sync.RWMutex{}

func CreateMatchmakingServiceTicket(c *gin.Context) {
	accountId := c.Param("accountId")
	bucketId := c.Query("bucketId")
	partyPlayerIds := c.Query("partyPlayerIds")
	fillTeam := c.DefaultQuery("player.option.fillTeam", "")

	if accountId == "" || bucketId == "" {
		utilities.Internal.ValidationFailed().Apply(c.Writer)
		return
	}

	decodedBucketId, err := url.QueryUnescape(bucketId)
	if err != nil {
		utilities.Internal.ValidationFailed().Apply(c.Writer)
		return
	}

	decodedBucketIds := strings.Split(decodedBucketId, ":")

	var account accounts.Account
	if err := odin.Find("Accounts", accountId, &account); err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	/*randomBytes32 := make([]byte, 32)

	if _, err := rand.Read(randomBytes32); err != nil {
		return
	}

	jti := hex.EncodeToString(randomBytes32)*/

	now := time.Now().UTC()
	expirationTime := now.Add(5 * time.Minute)

	//ua := utilities.Parse(c.GetHeader("User-Agent"))

	playlist := decodedBucketIds[3]

	buildUniqueIdsMutex.Lock()
	buildUniqueIds[accountId] = decodedBucketIds[0]
	buildUniqueIdsMutex.Unlock()

	payloadData := map[string]interface{}{
		"accountId":      accountId,
		"displayName":    account.DisplayName,
		"bucketId":       bucketId,
		"partyPlayerIds": partyPlayerIds,
		"fillTeam":       fillTeam,
		"playlist":       playlist,
		"region":         decodedBucketIds[2],
		"buildUniqueId":  decodedBucketIds[0],
		"exp":            expirationTime.Unix(),
	}
	payloadJSON, err := json.Marshal(payloadData)
	if err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	mac := hmac.New(sha256.New, []byte("ed14ba700b1aeb4103b457c2a43028e0"))
	mac.Write([]byte(payloadB64))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	response := gin.H{
		"serviceUrl": utilities.Get[string]("matchmaker"),
		"ticketType": "mms-player",
		"payload":    payloadB64,
		"signature":  signature,
	}
	c.JSON(http.StatusOK, response)
}

var sessions = make(map[string]struct {
	ID             string                 `json:"id"`
	CreatedAt      string                 `json:"created_at"`
	UpdatedAt      string                 `json:"updated_at"`
	Playlist       string                 `json:"Playlist"`
	ServerAddr     string                 `json:"ServerAddr"`
	ServerPort     int                    `json:"ServerPort"`
	ServerRegion   string                 `json:"ServerRegion"`
	ActivePlayers  int                    `json:"ActivePlayers"`
	AllPlayers     int                    `json:"AllPlayers"`
	Secret         string                 `json:"Secret"`
	Teams          interface{}            `json:"Teams"`
	Attributes     map[string]interface{} `json:"Attributes"`
	JoinInProgress bool                   `json:"JoinInProgress"`
	Version        string                 `json:"Version"`
	Joinable       bool                   `json:"Joinable"`
	Available      bool                   `json:"Available"`
})

func GetMatchmakingSession(c *gin.Context) {
	tokenHeader := c.GetHeader("Authorization")
	if tokenHeader == "" {
		utilities.Authentication.InvalidHeader().Apply(c.Writer)
		c.Abort()
		return
	}

	token := strings.ReplaceAll(tokenHeader, "bearer ", "")
	s, _ := odin.FindWhere("Accounts_Sessions", map[string]interface{}{
		"token": token,
	}, func() interface{} {
		return &accounts.Session{}
	})

	if s == nil {
		utilities.Authentication.InvalidToken().Apply(c.Writer)
		c.Abort()
		return
	}

	sessionSlice := s
	if len(sessionSlice) == 0 {
		utilities.Authentication.InvalidToken().Apply(c.Writer)
		c.Abort()
		return
	}

	sessionData := sessionSlice[0].(*accounts.Session)

	var account accounts.Account
	if err := odin.Find("Accounts", sessionData.AccountID, &account); err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	if account.Banned {
		utilities.Account.DisabledAccount().Apply(c.Writer)
		c.Abort()
		return
	}

	sessionId := c.Param("sessionId")
	if sessionId == "" {
		utilities.Internal.JsonParsingFailed().Apply(c.Writer)
		return
	}
	mmUrl := fmt.Sprintf("http://127.0.0.1:6767/remix/api/v1/session/%s", sessionId)
	req, err := http.NewRequest("GET", mmUrl, nil)
	if err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}
	req.Header.Set("X-Api-Key", "74993bcef253a6eea767ab01621b0c81")

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}
	defer resp.Body.Close()

	var mmServer struct {
		IP       string `json:"ip"`
		Port     uint16 `json:"port"`
		Region   string `json:"region"`
		Playlist string `json:"playlist"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&mmServer); err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}
	buildUniqueIdsMutex.RLock()
	playerBuildUniqueId := buildUniqueIds[sessionData.AccountID]
	buildUniqueIdsMutex.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"id":                              sessionId,
		"ownerId":                         strings.ReplaceAll(uuid.New().String(), "-", ""),
		"ownerName":                       "[DS]fortnite-liveeugcec1c2e30ubrcore0a-z8hj-1968",
		"serverName":                      "[DS]fortnite-liveeugcec1c2e30ubrcore0a-z8hj-1968",
		"serverAddress":                   mmServer.IP,
		"serverPort":                      mmServer.Port,
		"maxPublicPlayers":                220,
		"openPublicPlayers":               175,
		"maxPrivatePlayers":               0,
		"openPrivatePlayers":              0,
		"attributes":                      map[string]interface{}{},
		"publicPlayers":                   []string{},
		"privatePlayers":                  []string{},
		"totalPlayers":                    1,
		"allowJoinInProgress":             true,
		"shouldAdvertise":                 true,
		"isDedicated":                     false,
		"usesStats":                       true,
		"allowInvites":                    true,
		"usesPresence":                    true,
		"allowJoinViaPresence":            false,
		"allowJoinViaPresenceFriendsOnly": false,
		"buildUniqueId":                   playerBuildUniqueId,
		"lastUpdated":                     time.Now().UTC().Format(time.RFC3339),
		"started":                         false,
	})
	return

	/*types.HTTPServerM.RLock()
	httpServer, isHTTPServer := types.HTTPServers[sessionId]
	types.HTTPServerM.RUnlock()

	if isHTTPServer {
		buildUniqueIdsMutex.RLock()
		playerBuildUniqueId := buildUniqueIds[sessionData.AccountID]
		buildUniqueIdsMutex.RUnlock()

		c.JSON(http.StatusOK, gin.H{
			"id":                              sessionId,
			"ownerId":                         strings.ReplaceAll(uuid.New().String(), "-", ""),
			"ownerName":                       "[DS]fortnite-liveeugcec1c2e30ubrcore0a-z8hj-1968",
			"serverName":                      "[DS]fortnite-liveeugcec1c2e30ubrcore0a-z8hj-1968",
			"serverAddress":                   httpServer.IP,
			"serverPort":                      httpServer.Port,
			"maxPublicPlayers":                220,
			"openPublicPlayers":               175,
			"maxPrivatePlayers":               0,
			"openPrivatePlayers":              0,
			"attributes":                      map[string]interface{}{},
			"publicPlayers":                   []string{},
			"privatePlayers":                  []string{},
			"totalPlayers":                    1,
			"allowJoinInProgress":             true,
			"shouldAdvertise":                 true,
			"isDedicated":                     false,
			"usesStats":                       true,
			"allowInvites":                    true,
			"usesPresence":                    true,
			"allowJoinViaPresence":            false,
			"allowJoinViaPresenceFriendsOnly": false,
			"buildUniqueId":                   playerBuildUniqueId,
			"lastUpdated":                     time.Now().UTC().Format(time.RFC3339),
			"started":                         false,
		})
		return
	}

	var session fortnite.Sessions
	if err := odin.Find("GameSessions", sessionId, &session); err != nil {
		mapdata, exists := sessions[sessionId]
		if exists {
			buildUniqueIdsMutex.RLock()
			storedBuildUniqueId := buildUniqueIds[sessionData.AccountID]
			buildUniqueIdsMutex.RUnlock()

			c.JSON(http.StatusOK, gin.H{
				"id":                              sessionId,
				"ownerId":                         strings.ReplaceAll(uuid.New().String(), "-", ""),
				"ownerName":                       "[DS]fortnite-liveeugcec1c2e30ubrcore0a-z8hj-1968",
				"serverName":                      "[DS]fortnite-liveeugcec1c2e30ubrcore0a-z8hj-1968",
				"serverAddress":                   mapdata.ServerAddr,
				"serverPort":                      mapdata.ServerPort,
				"maxPublicPlayers":                220,
				"openPublicPlayers":               175,
				"maxPrivatePlayers":               0,
				"openPrivatePlayers":              0,
				"attributes":                      map[string]interface{}{},
				"publicPlayers":                   []string{},
				"privatePlayers":                  []string{},
				"totalPlayers":                    45,
				"allowJoinInProgress":             true,
				"shouldAdvertise":                 true,
				"isDedicated":                     false,
				"usesStats":                       true,
				"allowInvites":                    true,
				"usesPresence":                    true,
				"allowJoinViaPresence":            false,
				"allowJoinViaPresenceFriendsOnly": false,
				"buildUniqueId":                   storedBuildUniqueId,
				"lastUpdated":                     time.Now().UTC().Format(time.RFC3339),
				"started":                         false,
			})

			return
		}

		ip := utilities.Get[string]("ip")
		resp, err := http.Get("http://" + ip + ":2087/nxa/echo/session/get/" + sessionId)
		if err != nil {
			utilities.Matchmaking.UnknownSession().Apply(c.Writer)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			utilities.Matchmaking.UnknownSession().Apply(c.Writer)
			return
		}

		var body struct {
			ID             string                 `json:"id"`
			CreatedAt      string                 `json:"created_at"`
			UpdatedAt      string                 `json:"updated_at"`
			Playlist       string                 `json:"Playlist"`
			ServerAddr     string                 `json:"ServerAddr"`
			ServerPort     int                    `json:"ServerPort"`
			ServerRegion   string                 `json:"ServerRegion"`
			ActivePlayers  int                    `json:"ActivePlayers"`
			AllPlayers     int                    `json:"AllPlayers"`
			Secret         string                 `json:"Secret"`
			Teams          interface{}            `json:"Teams"`
			Attributes     map[string]interface{} `json:"Attributes"`
			JoinInProgress bool                   `json:"JoinInProgress"`
			Version        string                 `json:"Version"`
			Joinable       bool                   `json:"Joinable"`
			Available      bool                   `json:"Available"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			utilities.Matchmaking.UnknownSession().Apply(c.Writer)
			return
		} else {
			buildUniqueIdsMutex.RLock()
			storedBuildUniqueId := buildUniqueIds[sessionData.AccountID]
			buildUniqueIdsMutex.RUnlock()

			c.JSON(http.StatusOK, gin.H{
				"id":                              sessionId,
				"ownerId":                         strings.ReplaceAll(uuid.New().String(), "-", ""),
				"ownerName":                       "[DS]fortnite-liveeugcec1c2e30ubrcore0a-z8hj-1968",
				"serverName":                      "[DS]fortnite-liveeugcec1c2e30ubrcore0a-z8hj-1968",
				"serverAddress":                   body.ServerAddr,
				"serverPort":                      body.ServerPort,
				"maxPublicPlayers":                220,
				"openPublicPlayers":               175,
				"maxPrivatePlayers":               0,
				"openPrivatePlayers":              0,
				"attributes":                      map[string]interface{}{},
				"publicPlayers":                   []string{},
				"privatePlayers":                  []string{},
				"totalPlayers":                    45,
				"allowJoinInProgress":             true,
				"shouldAdvertise":                 true,
				"isDedicated":                     false,
				"usesStats":                       true,
				"allowInvites":                    true,
				"usesPresence":                    true,
				"allowJoinViaPresence":            false,
				"allowJoinViaPresenceFriendsOnly": false,
				"buildUniqueId":                   storedBuildUniqueId,
				"lastUpdated":                     time.Now().UTC().Format(time.RFC3339),
				"started":                         false,
			})

			return
		}
	}

	buildUniqueIdsMutex.RLock()
	playerBuildUniqueId := buildUniqueIds[sessionData.AccountID]
	buildUniqueIdsMutex.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"id":                              sessionId,
		"ownerId":                         strings.ReplaceAll(uuid.New().String(), "-", ""),
		"ownerName":                       session.OwnerName,
		"serverName":                      session.ServerName,
		"serverAddress":                   session.ServerAddress,
		"serverPort":                      session.ServerPort,
		"maxPublicPlayers":                session.MaxPublicPlayers,
		"openPublicPlayers":               session.OpenPublicPlayers,
		"maxPrivatePlayers":               session.MaxPrivatePlayers,
		"openPrivatePlayers":              session.OpenPrivatePlayers,
		"attributes":                      map[string]interface{}{},
		"publicPlayers":                   session.PublicPlayers,
		"privatePlayers":                  session.PrivatePlayers,
		"totalPlayers":                    len(session.PublicPlayers),
		"allowJoinInProgress":             session.AllowJoinInProgress,
		"shouldAdvertise":                 session.ShouldAdvertise,
		"isDedicated":                     false,
		"usesStats":                       session.UsesStats,
		"allowInvites":                    session.AllowInvites,
		"usesPresence":                    session.UsesPresence,
		"allowJoinViaPresence":            session.AllowJoinViaPresence,
		"allowJoinViaPresenceFriendsOnly": session.AllowJoinViaPresenceFriendsOnly,
		"buildUniqueId":                   playerBuildUniqueId,
		"lastUpdated":                     time.Now().UTC().Format(time.RFC3339),
		"started":                         session.Started,
	})*/
}

func GetMatchmakingEncryptionKey(c *gin.Context) {
	accountId := c.Param("accountId")
	sessionId := c.Param("sessionId")

	if accountId == "" || sessionId == "" {
		utilities.Internal.JsonParsingFailed().Apply(c.Writer)
		return
	}

	key := "AOJEv8uTFmUh7XM2328kq9rlAzeQ5xzWzPIiyKn2s7s="

	c.JSON(http.StatusOK, gin.H{
		"accountId": accountId,
		"sessionId": sessionId,
		"key":       key,
	})
}

func PostJoinMatchmakingSession(c *gin.Context) {
	c.String(http.StatusOK, "")
}
