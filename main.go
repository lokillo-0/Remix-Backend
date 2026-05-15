package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"math"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/andr1ww/odin"
	"github.com/fatih/color"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/joho/godotenv"
	"github.com/remixfn/xenon/modules/database"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/modules/matchmaking"
	"github.com/remixfn/xenon/modules/storefront"
	s "github.com/remixfn/xenon/modules/synapse"
	"github.com/remixfn/xenon/router/account_public"
	"github.com/remixfn/xenon/router/cloudstorage"
	"github.com/remixfn/xenon/router/discovery"
	"github.com/remixfn/xenon/router/eos"
	"github.com/remixfn/xenon/router/fortnite"
	fortnite_dedicated_server "github.com/remixfn/xenon/router/fortnite/dedicated_server"
	fortnite_mcp "github.com/remixfn/xenon/router/fortnite/mcp"
	"github.com/remixfn/xenon/router/iridium"
	matchmaking_router "github.com/remixfn/xenon/router/matchmaking"
	"github.com/remixfn/xenon/router/mercury"
	"github.com/remixfn/xenon/router/oauth"
	"github.com/remixfn/xenon/router/party"
	"github.com/remixfn/xenon/router/public"
	"github.com/remixfn/xenon/router/remix"
	remix_launcher "github.com/remixfn/xenon/router/remix/launcher"
	remix_server "github.com/remixfn/xenon/router/remix/server"
	remix_server_admin "github.com/remixfn/xenon/router/remix/server/admin"
	remix_server_fortnite "github.com/remixfn/xenon/router/remix/server/fortnite"
	"github.com/remixfn/xenon/router/synapse"
	"github.com/remixfn/xenon/utilities"
	"github.com/remixfn/xenon/utilities/middleware"
	"golang.org/x/net/http2"
)

var (
	clientCount        int = 0
	clientsLastFetched time.Time
)

func main() {
	log.SetFlags(0)
	gin.SetMode(gin.ReleaseMode)
	runtime.GOMAXPROCS(0)
	debug.SetGCPercent(20)
	debug.SetMemoryLimit(math.MaxInt64)

	if err := odin.Connect("xenon", "databases/xenon.db"); err != nil {
		log.Fatal(err)
	}
	defer odin.Close("xenon")

	if err := odin.Connect("xenon_profiles", "databases/xenon_profiles.db"); err != nil {
		log.Fatal(err)
	}
	defer odin.Close("xenon_profiles")

	if err := odin.Connect("xenon_comp", "databases/xenon_comp.db"); err != nil {
		log.Fatal(err)
	}
	defer odin.Close("xenon_comp")

	if err := odin.Connect("xenon_redeem", "databases/xenon_redeem.db"); err != nil {
		log.Fatal(err)
	}
	defer odin.Close("xenon_redeem")
	if err := godotenv.Load(); err != nil {
		color.Yellow("Warning: Could not load .env file, using system environment variables only.")
	}

	utilities.Load()

	fortnite_mcp.LoadAthenaCache()
	storefront.Init()
	cloudstorage.Init()
	manager := s.NewSynapseManager()
	err := manager.Start()
	if err != nil {
		utilities.LogWithTimestamp(color.RedString, "Failed to start Synapse module: %v", err)
	} else {
		utilities.LogWithTimestamp(color.GreenString, "Synapse module started & connected")
	}

	dbNames := []string{"xenon", "xenon_profiles", "xenon_comp"}
	for _, name := range dbNames {
		db, err := odin.GetNamed(name)
		if err != nil {
			utilities.LogWithTimestamp(color.RedString, "Failed to get %s database: %v", name, err)
			continue
		}
		db.Compact()
	}

	db, err := odin.Get()
	if err != nil {
		log.Fatalf("Failed to get database connection: %v", err)
	}
	if err := db.Clear("GameSessions"); err != nil {
		utilities.LogWithTimestamp(color.RedString, "Failed to clear sessions: "+err.Error())
		return
	}

	var router *gin.Engine
	if utilities.GetConfig().Prod {
		router = gin.New()
	} else {
		router = gin.Default()
	}

	router.Use(gin.Recovery())

	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "issuer", "Issuer", "authorization"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
	}))

	matchmaking.Init(router)

	router.POST("/api/v1/server/ready", matchmaking_router.ServerReady)
	router.POST("/api/v1/server/unready", matchmaking_router.ServerUnready)

	router.GET("/fortnite/api/cloudstorage/system", cloudstorage.GetCloudstorage)
	router.GET("/fortnite/api/cloudstorage/system/config", cloudstorage.GetCloudstorage)
	router.GET("/fortnite/api/cloudstorage/system/:filename", cloudstorage.GetCloudstorageFile)
	router.POST("/fortnite/api/game/v2/tryPlayOnPlatform/account/:accountId", fortnite.TryPlayOnPlatform)
	router.GET("/fortnite/api/game/v2/enabled_features", fortnite.EnabledFeatures)
	Lightswitch := router.Group("/lightswitch/api/service")
	{
		Lightswitch.Use(middleware.ClientAuthMiddleware())
		Lightswitch.GET("/bulk/status", fortnite.LightswitchBulk)
	}
	router.GET("/fortnite/api/v2/versioncheck/:platform", fortnite.VersionCheck)
	router.GET("/waitingroom/api/waitingroom", fortnite.Ret204)
	router.POST("/fortnite/api/game/v2/toxicity/account/:unsafeReporter/report/:reportedPlayer", fortnite.ReportPlayer)

	router.GET("/fortnite/api/game/v2/privacy/account/:accountdId", fortnite.Ret204)
	// router.POST("/datarouter/api/v1/public/data", func(c *gin.Context) {
	// 	body, err := io.ReadAll(c.Request.Body)
	// 	if err != nil {
	// 		utilities.LogWithTimestamp(color.RedString, "Failed to read request body: %v", err)
	// 	} else {
	// 		utilities.LogWithTimestamp(color.YellowString, "datarouter request body: %s", string(body))
	// 	}
	// 	fortnite.Ret204(c)
	// })
	router.POST("/api/v1/:hi/channel/motd/target", fortnite.GETMotdTarget)
	router.POST("/api/v1/:hi/surfaces/:gameMode/target", fortnite.GETMotdTarget)
	router.POST("/api/v1/:hi/interactions/contentHash", fortnite.Ret204)
	router.POST("/datarouter/api/v1/public/data", fortnite.Ret204)
	router.POST("/datarouter/api/v1/public/data/clients", fortnite.Ret204)
	router.POST("/telemetry/data/datarouter/api/v1/public/data", fortnite.Ret204)
	router.GET("/account/api/oauth/verify", fortnite.RetJson)
	router.DELETE("/account/api/oauth/sessions/kill", fortnite.Ret204)
	router.POST("/auth/v1/turn/credentials", fortnite.RetJson)
	router.GET("/hotconfigs/v2/livefn.json", fortnite.RetJson)
	router.POST("/api/v1/fortnite-br/interactions", fortnite.RetJson)
	router.GET("/api/v2/interactions/latest/:game/:accountId", func(c *gin.Context) {
		c.JSON(200, gin.H{"interactions": []interface{}{}})
	})
	router.GET("/presence/api/v1/_/:accountId/last-online", fortnite.RetJson)
	router.GET("/eulatracking/api/public/agreements/:agreementId/account/:accountId", fortnite.Ret204)
	router.GET("//api/content/v2/launch-data", fortnite.Ret204)
	router.POST("/publickey/v2/publickey", func(c *gin.Context) {
		if err := c.Request.ParseForm(); err != nil {
			c.JSON(400, gin.H{"error": "invalid form"})
			return
		}

		body := c.Request.PostForm

		accountId := body.Get("account_id")
		if accountId == "" {
			accountId = body.Get("accountId")
		}
		if accountId == "" {
			accountId = body.Get("username")
		}

		key := body.Get("key")

		claims := jwt.MapClaims{
			"account_id": accountId,
			"generated":  1731795408,
			"key_guid":   "2e57bba7-4a7a-423c-b4b4-853acfcf019c",
			"kid":        "20230621",
			"key":        key,
			"expiration": "9999-12-31T23:59:59.999Z",
			"type":       "legacy",
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		token.Header["kid"] = "20230621"

		signed, err := token.SignedString([]byte("NexaKey"))
		if err != nil {
			c.JSON(500, gin.H{"error": "failed to sign jwt"})
			return
		}

		c.JSON(200, gin.H{
			"key":        key,
			"account_id": accountId,
			"key_guid":   "2e57bba7-4a7a-423c-b4b4-853acfcf019c",
			"kid":        "20230621",
			"expiration": "9999-12-31T23:59:59.999Z",
			"jwt":        signed,
			"type":       "legacy",
		})
	})
	router.GET("/api/v1/public/accounts", func(c *gin.Context) {
		accountId := c.Query("accountId")

		c.JSON(200, gin.H{
			"accounts": []gin.H{
				{
					"accountId": accountId,
					"tags":      []string{},
				},
			},
		})
	})
	router.Any("/fortnite/api/game/v2/profileToken/verify/:accountId", func(c *gin.Context) {
		if c.Request.Method != "POST" {
			c.Status(405)
			return
		}

		c.Status(204)
	})
	router.POST("/auth/v1/oauth/token", eos.GETEOSAuth)
	router.POST("/epic/oauth/v2/tokenInfo", eos.GetEpicOAuthV2TokenInfo)
	router.POST("/epic/oauth/v2/token", eos.PostEpicOAuthV2Token)
	router.GET("/epic/id/v2/sdk/accounts", eos.GETEOSSDKAccounts)
	router.GET("/sdk/v1/default", eos.GETEOSSdk)
	router.GET("/sdk/v1/product/prod-fn", eos.GETEOSSdk)
	router.GET("/epic/friends/v1/:accountId/blocklist", fortnite.EnabledFeatures)
	router.Any("/v1/epic-settings/public/users/:accountId/values", eos.GETEOSSettings)
	router.GET("/content/api/pages/fortnite-game", fortnite.FortniteContentPagesHandler)
	router.GET("/content/api/pages/fortnite-game/*any", fortnite.FortniteContentPagesHandler)
	router.GET("/content-controls/:accountId", fortnite.GetContentControls)
	router.GET("/content-controls/:accountId/rules/namespaces/fn", fortnite.Ret204)
	router.GET("/socialban/api/public/v1/:accountId", fortnite.RetJson)
	router.GET("/launcher/api/public/assets/:platform/:catalogItemId/:appName", func(c *gin.Context) {
		manifestPath := "Builds/Fortnite/Content/CloudDir/Nexa.manifest"
		distribution := "http://localhost:80/"
		if c.Param("platform") == "Android" || c.Param("platform") == "android" {
			manifestPath = "Builds/Fortnite/Content/CloudDir/9O7dGkaFewI7qGElsE2rjSDu5u6jeg.manifest"
			distribution = "https://epicgames-download1.akamaized.net/"
		}
		c.JSON(200, gin.H{
			"appName":       c.Param("appName"),
			"labelName":     c.Query("label") + "-" + c.Param("platform"),
			"buildVersion":  "remix",
			"catalogItemId": c.Param("catalogItemId"),
			"expires":       "9988-09-23T23:59:59.999Z",
			"items": gin.H{
				"MANIFEST": gin.H{
					"signature": "remix", "distribution": distribution,
					"path": manifestPath, "additionalDistributions": []string{},
				},
			},
			"assetId": c.Param("appName"),
		})
	})
	router.GET("/launcher/api/public/distributionpoints", func(c *gin.Context) {
		c.JSON(200, gin.H{"distributions": []string{
			"https://epicgames-download1.akamaized.net/",
			"https://download.epicgames.com/",
			"https://download2.epicgames.com/",
			"https://download3.epicgames.com/",
			"https://download4.epicgames.com/",
			"https://remix.ol.epicgames.com/",
		}})
	})
	router.GET("/Builds/Fortnite/Content/CloudDir/ChunksV4/:chunknum/:filename", func(c *gin.Context) {
		client := &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				DisableKeepAlives:   false,
			},
		}

		url := "https://epicgames-download1.akamaized.net/Builds/Fortnite/Content/CloudDir/ChunksV4/" +
			c.Param("chunknum") + "/" + c.Param("filename")

		ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		resp, err := client.Do(req)
		if err != nil {
			c.Status(http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		c.Header("Content-Type", "application/octet-stream")
		c.Status(resp.StatusCode)

		_, err = io.Copy(c.Writer, resp.Body)
		if err != nil {
			utilities.LogWithTimestamp(color.RedString, "Error copying response body: %v", err)
		}
	})
	router.GET("/Builds/Fortnite/Content/CloudDir/Nexa/:filename", func(c *gin.Context) {
		filename := c.Param("filename")
		if len(filename) >= 4 && filename[len(filename)-4:] == ".ini" {
			c.FileAttachment("static/assets/stuff.ini", "stuff.ini")
		} else if filename == "Nexa.manifest" {
			c.Header("Content-Type", "application/octet-stream")
			c.File("static/assets/Nexa.manifest")
		} else {
			c.Status(404)
		}
	})

	router.GET("/Builds/Fortnite/Content/CloudDir/:filename", func(c *gin.Context) {
		filename := c.Param("filename")
		if filename == "LtARIUI_xpy6J5oNsEmwKeKv3Q1lmg.manifest" {
			c.Header("Content-Type", "application/octet-stream")
			c.File("static/assets/LtARIUI_xpy6J5oNsEmwKeKv3Q1lmg.manifest")
		}
		if len(filename) >= 4 && filename[len(filename)-4:] == ".ini" {
			c.FileAttachment("static/assets/stuff.ini", "stuff.ini")
		} else if filename == "Nexa.manifest" {
			c.Header("Content-Type", "application/octet-stream")
			c.File("static/assets/Nexa.manifest")
		} else {
			c.Status(404)
		}
	})

	router.GET("/fortnite/api/discovery/accessToken/*path", discovery.DiscoveryAccessToken)

	router.POST("/api/v2/discovery/surface/*path", discovery.HandleDiscoverySurface)
	router.POST("/api/v1/assets/Fortnite/:ver/:cl", discovery.HandleAssets)
	router.GET("/api/v1/assets/Fortnite/:ver/:cl/FortPlaylistAthena/:playlist", discovery.GETAssetsFortPlaylistAthena)
	router.GET("/fortnite/api/matchmaking/session/findPlayer/:id", fortnite.EnabledFeatures)
	router.POST("/fortnite/api/game/v2/creative/discovery/surface/*path", discovery.HandleCreativeDiscoverySurface)
	router.POST("/api/v1/discovery/surface/*path", discovery.HandleDiscoverySurfaceV1)
	router.POST("/links/api/fn/mnemonic", discovery.HandleMnemonic)
	router.POST("/api/v1/links/lock-status/:accountId/check", discovery.HandleLockStatusCheck)
	router.GET("/api/v1/search/:accid", fortnite.SearchUsersByPrefix)
	router.GET("/region", discovery.GETRegion)
	router.GET("/links/api/fn/mnemonic/:playlistId", discovery.GETLinksMnemonicPlaylist)
	router.GET("/links/api/fn/mnemonic/:playlistId/related", discovery.HandleRelatedPlaylist)
	router.POST("/fortnite/api/matchmaking/session/:sessionId/join", fortnite.PostJoinMatchmakingSession)
	router.GET("/v1/avatar/fortnite/ids", account_public.GetAvatarIds)
	router.GET("/fortnite/api/game/v2/matchmaking/account/:accountId/session/:sessionId", fortnite.GetMatchmakingEncryptionKey)

	router.GET("/friends/api/v1/:accountId/friends/:friendId/mutual", func(c *gin.Context) {
		c.JSON(200, gin.H{})
	})
	router.GET("/api/v2/interactions/aggregated/Fortnite/:accountId", fortnite.Ret204)

	router.GET("/mobile/install", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "itms-services://?action=download-manifest&url=https://r2.ploosh.dev/remix/RemixPresigned-1776574883131.plist")
	})

	AccountOAuth := router.Group("/account/api/oauth")
	{
		AccountOAuth.POST("/token", oauth.PostOAuthToken)
		AccountOAuth.POST("/token/switch", oauth.PostOAuthTokenSwitch)
		AccountOAuth.GET("/exchange", oauth.POSTCreateOAUTHExchange)
		AccountOAuth.POST("/exchange", oauth.POSTCreateOAUTHExchange)
	}

	FeedBack := router.Group("/fortnite/api/feedback")
	{

		FeedBack.POST("/Bug", fortnite.SubmitBugFeedback)
		FeedBack.POST("/Comment", fortnite.SubmitCommentFeedback)
	}

	router.Static("/assets", "./assets")
	router.Static("/admin", "./admin")

	router.StaticFile("/shop", "./admin/shop.html")
	router.GET("/api/public/shop", public.GETPublicShop)
	router.GET("/api/public/vbucks", public.GETPublicVbucks)
	router.GET("/api/public/locker", public.GETPublicLocker)

	NexaLauncher := router.Group("/rmx/launcher/v1")
	{
		NexaLauncher.GET("/clients/bot/discord", func(c *gin.Context) {
			ids := make([]string, 0, len(remix_launcher.LauncherClients))
			for _, client := range remix_launcher.LauncherClients {
				if client.Account != nil {
					ids = append(ids, client.Account.ID)
				}
			}
			c.JSON(200, ids)
		})

		NexaLauncher.GET("/updater/:platform", remix_launcher.GETLauncherUpdater)
		NexaLauncher.GET("/socket", remix_launcher.HandleLauncherSocketConnection)
		NexaLauncher.GET("/banners", remix_launcher.GETLauncherBanners)
		NexaLauncher.POST("/banners/upload/:name", remix_launcher.POSTAdminUploadBanner)
		NexaLauncher.GET("/dlls", remix_launcher.GETLauncherDlls)
		NexaLauncher.POST("/dlls/upload/:name", remix_launcher.POSTAdminUploadDll)
	}

	RemixMainAPI := router.Group("/rmx/api/v1")
	{
		RemixMainAPI.POST("/upload/profile/:accountid", remix.POSTRemixUploadProfile)
	}

	RemixServer := router.Group("/rmx/server/api/v1")
	{
		RemixServer.GET("/clients", func(ctx *gin.Context) {
			if clientCount == 0 || time.Since(clientsLastFetched) > 30*time.Second {
				resp, err := http.Get("http://127.0.0.1:4040/connections")
				if err != nil {
					ctx.String(500, "0")
					return
				}
				defer resp.Body.Close()
				var result struct {
					Count int `json:"count"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					ctx.String(500, "0")
					return
				}
				clientCount = result.Count
				clientsLastFetched = time.Now()
			}

			ctx.String(200, "%d", clientCount)
		})
		RemixServer.POST("/playlist", remix_server.POSTCreatePlaylistInfo)
		RemixServer.POST("/massdel/ugh/prefix", remix_server.POSTMassDeleteAccountsByEmailPrefix)
		RemixServer.POST("/massdel/ugh", remix_server.POSTMassDeleteAccountsByDomain)
		RemixServer.GET("/massun/ughhh/wow", remix_server_admin.POSTUnbanAll)
		RemixServer.POST("/register", remix_server.POSTRemixServerRegister)
		RemixServer.POST("/login", remix_server.POSTRemixServerLogin)
		RemixServer.GET("/discord/auth", remix_server.GETDiscordAuth)
		RemixServer.GET("/discord/callback", remix_server.GETDiscordCallback)
		RemixServer.GET("/discord/pending/:state", remix_server.GETDiscordPending)
		router.GET("/mobile/login", remix_server.GETMobileLogin)
		RemixServer.PUT("/hotfixes", remix_server.PUTRemixServerHotfixes)
		RemixServer.PUT("/codes/create", remix_server.PUTRemixServerCreateCode)
		RemixServer.POST("/codes/redeem/:accountId/:code", remix_server.POSTRemixServerCodesRedeem)
		RemixServer.POST("/news", remix_server.POSTRemixServerNews)
		RemixServer.POST("/posts", remix_server.POSTRemixServerPosts)
		RemixServer.POST("/account/management/displayName/:accountid", remix_server.POSTRemixServerChangeDisplayName)
		RemixServer.POST("/account/management/email/:accountid", remix_server.POSTRemixServerChangeEmail)
		RemixServer.POST("/account/management/password/:accountid", remix_server.POSTRemixServerChangePassword)
		RemixServer.POST("/create/event", remix_server_fortnite.POSTRemixServerCreateEvent)
		RemixServer.DELETE("/delete/event/:eventId", remix_server_fortnite.DELETERemixServerDeleteEvent)
		RemixServer.POST("/admin/check", remix_server_admin.POSTRemixServerAdminCheck)
		RemixServer.GET("/admin/account/lookup/:identifier", remix_server.GETAdminAccountLookup)
		RemixServer.POST("/admin/reset", remix_server.POSTRemixServerPasswordReset)
		RemixServer.POST("/admin/ban/:accountid", remix_server.POSTRemixServerBanAccount)
		RemixServer.POST("/admin/unban/:accountid", remix_server.POSTRemixServerUnbanAccount)
		RemixServer.DELETE("/admin/account/:name", remix_server.DELETEAccountByDisplayName)
		RemixServer.POST("/admin/fulllocker/:accountId", remix_server.POSTGrantFullLocker)
		RemixServer.POST("/webhooks/sellauth/full-locker", remix_server.POSTSellAuthWebhook)
		RemixServer.DELETE("/admin/fulllocker/:accountId", remix_server.DELETEFullLocker)
		RemixServer.POST("/admin/vbucks/:accountId/:amount", remix_server.POSTGrantVBucks)
		RemixServer.POST("/admin/grant/:accountId/:templateId", remix_server.POSTGrantItem)
		RemixServer.DELETE("/admin/items/:accountId", remix_server.DELETEAllItems)
		RemixServer.POST("/admin/shop/regenerate", func(c *gin.Context) {
			go storefront.ForceRegenerate()
			c.JSON(200, gin.H{"message": "Shop regeneration started"})
		})
		RemixServer.GET("/admin/shop/config", remix_server_admin.GETAdminShopConfig)
		RemixServer.PUT("/admin/shop/config", remix_server_admin.PUTAdminShopConfig)
		RemixServer.GET("/friends/:accountId", remix_server.GETLauncherFriends)
		RemixServer.POST("/admin/register", remix_server_admin.POSTRemixServerAdminRegister)
		RemixServer.POST("/host/create", remix_server.POSTCreateHostAccount)
		RemixServer.POST("/host/login", remix_server.POSTHostAccountLogin)
		RemixServer.GET("/host", remix_server.GETListHostAccounts)
		RemixServer.DELETE("/host/:username", remix_server.DELETEHostAccount)
		RemixServer.POST("/admin/banners", remix_server_admin.POSTAdminCreateBanner)
		RemixServer.DELETE("/admin/banners/:id", remix_server_admin.DELETEAdminBanner)
		RemixServer.GET("/beta/check", remix_server.GETBetaCheck)
		RemixServer.PUT("/admin/playlist", remix_server_admin.POSTRemixAdminCreatePlaylist)
		RemixServer.POST("/admin/playlist/disable", remix_server_admin.POSTRemixAdminDisablePlaylist)
		RemixServer.POST("/admin/playlist/enable", remix_server_admin.POSTRemixAdminEnablePlaylist)
		RemixServer.GET("/admin/accounts/all", remix_server.GETAdminAccountsAll)
		RemixServer.DELETE("/admin/battlepass/:accountId/:season", remix_server_admin.DELETEAdminBattlePass)

		//RemixServer.Use(middleware.NexaServerAuthMiddleware())
		RemixServer.POST("/:sessionId/BroadcastMatchRewards", remix_server_fortnite.POSTBroadcastMatchResults)
		RemixServer.POST("/kill/:accountId", remix_server_fortnite.POSTVbucksPerKill)
		RemixServer.POST("/win/:accountId", remix_server_fortnite.POSTWonGame)
	}

	AccountPublic := router.Group("/account/api/public/account")
	{
		AccountPublic.GET("/", account_public.GETAccountPublicAccountIDQuery)
		AccountPublic.Use(middleware.ClientAuthMiddleware())
		AccountPublic.GET("/:accountId", account_public.GETAccountPublicAccountID)
		AccountPublic.GET("/displayName/:displayName", account_public.GetPlayerByDisplayName)
		AccountPublic.GET("/:accountId/externalAuths", account_public.GETAccountPublicAccountID)
		AccountPublic.POST("/:accountId/deviceAuth", account_public.POSTPublicDeviceAuth)
	}

	Devices := router.Group("/api/v3/external/devices")
	{
		Devices.Use(middleware.ClientAuthMiddleware())
		Devices.POST("/", account_public.PostV3DeviceAuth)
	}

	Matchmaking := router.Group("/fortnite/api/matchmaking")
	{
		Matchmaking.GET("/session/:sessionId", fortnite.GetMatchmakingSession)
		Matchmaking.POST("/session", fortnite_dedicated_server.CreateSession)
		Matchmaking.POST("/session/:sessionId/players", fortnite_dedicated_server.UpdateSessionPlayers)
		Matchmaking.POST("/session/:sessionId", fortnite_dedicated_server.UpdateSession)
		Matchmaking.POST("/session/:sessionId/heartbeat", fortnite_dedicated_server.GameSessionHeartbeat)
		Matchmaking.PUT("/session/:sessionId", fortnite_dedicated_server.UpdateSession)
		Matchmaking.POST("/session/:sessionId/start", fortnite_dedicated_server.StartGameSession)
		Matchmaking.POST("/session/:sessionId/stop", fortnite_dedicated_server.StopGameSession)
	}

	EventsService := router.Group("/api/v1/events/Fortnite")
	{
		EventsService.GET("/download/:accountId", fortnite.DownloadEventsViaAccountId)
		EventsService.POST("/bulk/team", fortnite.BulkTeam)
		EventsService.GET("/data/:eventId/:windowId", fortnite.GetEventWindowData)
	}

	MCP := router.Group("/fortnite/api/game/v2/profile/:accountId")
	{
		MCP.Use(middleware.ClientAuthMiddleware())
		MCP.POST("/client/QueryProfile", fortnite_mcp.POSTQueryProfile)
		MCP.POST("/client/SetHardcoreModifier", fortnite_mcp.POSTQueryProfile)
		MCP.POST("/client/PurchaseCatalogEntry", fortnite_mcp.POSTPurchaseCatalogEntry)
		MCP.POST("/client/PurchaseMultipleCatalogEntries", fortnite_mcp.POSTPurchaseMultipleCatalogEntries)
		MCP.POST("/client/GiftCatalogEntry", fortnite_mcp.POSTGiftCatalogEntry)
		MCP.POST("/client/RedeemRealMoneyPurchases", fortnite_mcp.POSTQueryProfile)
		MCP.POST("/client/SetMtxPlatform", fortnite_mcp.POSTQueryProfile)
		MCP.POST("/client/ClientQuestLogin", fortnite_mcp.POSTClientQuestLogin)
		MCP.POST("/client/SetCosmeticLockerSlot", fortnite_mcp.POSTSetCosmeticLockerSlot)
		MCP.POST("/client/MarkItemSeen", fortnite_mcp.POSTMarkItemSeen)
		MCP.POST("/client/SetCosmeticLockerBanner", fortnite_mcp.POSTSetCosmeticLockerBanner)
		MCP.POST("/client/SetItemFavoriteStatusBatch", fortnite_mcp.POSTSetItemFavoriteStatusBatch)
		MCP.POST("/client/ExchangeGameCurrencyForBattlePassOffer", fortnite_mcp.POSTExchangeGameCurrencyForBattlePassOffer)
		MCP.POST("/client/RemoveGiftBox", fortnite_mcp.POSTRemoveGiftBox)
		MCP.POST("/dedicated_server/QueryProfile", fortnite_mcp.POSTQueryProfile)
		MCP.POST("/client/SetItemFavoriteStatus", fortnite_mcp.POSTSetItemFavoriteStatus)
		//MCP.POST("/client/CopyCosmeticLoadout", fortnite_mcp.POSTCopyCosmeticLoadout)
	}

	Locker := router.Group("/api/locker/v4")
	{
		//Locker.Use(middleware.ClientAuthMiddleware())
		Locker.GET(":deploymentId/account/:accountId/items", fortnite.GETLockerItems)
		Locker.PUT(":deploymentId/account/:accountId/active-loadout-group", fortnite.PUTActiveLoadoutGroup)
	}

	Synapse := router.Group("/synapse/api/v1")
	{
		Synapse.POST("/:id/auth", synapse.Auth)
		Synapse.GET("/:id/friends", synapse.Friends)
		Synapse.GET("/account/:id/username", synapse.GetUsernameViaAccountID)
	}

	Calendar := router.Group("/fortnite/api/calendar/v1")
	{
		Calendar.Use(middleware.ClientAuthMiddleware())
		Calendar.GET("/timeline", fortnite.Timeline)
	}

	Storefront := router.Group("/fortnite/api/storefront/v2")
	{
		Storefront.GET("/keychain", fortnite.Keychain)
		Storefront.GET("/catalog", fortnite.Catalog)
		Storefront.GET("/gift/check_eligibility/recipient/:recipientId/offer/*offerId", fortnite.CheckEligibility)
	}

	CloudstorageUser := router.Group("/fortnite/api/cloudstorage/user")
	{
		CloudstorageUser.GET("/:accountId", cloudstorage.GetUsersCloudstorageFiles2)
		CloudstorageUser.GET("/:accountId/:filename", cloudstorage.GetUserCloudstorageFile)
		CloudstorageUser.PUT("/:accountId/:filename", cloudstorage.SaveUsersCloudstorageFile)
	}

	router.GET("/api/v1/access/fortnite/cloudstorage/Live/user/:accountId/*filename", cloudstorage.GetUserCloudstorageFileViaAccess)

	IridiumZinc := router.Group("/i/zinc/api/v1")
	{
		IridiumZinc.GET("/account/:accountid", iridium.GETIridiumGetUser)
		IridiumZinc.POST("/account/:accountid/:action", iridium.POSTIridiumMain)
	}

	Iridium := router.Group("/i/api/v1")
	{
		Iridium.POST("/auth/create/:accountid", iridium.POSTIridiumCreateAuth)
	}

	MercurySolana := router.Group("/m/solana/api/v1/")
	{
		MercurySolana.POST("/auth", mercury.POSTGetMercuryUser)
		MercurySolana.POST("/detection", mercury.POSTMercuryDetection)
	}

	MatchmakingService := router.Group("/fortnite/api/game/v2/matchmakingservice")
	{
		MatchmakingService.GET("/ticket/session/:sessionId", fortnite_dedicated_server.CreateMatchmakingServiceTicket)
		MatchmakingService.Use(middleware.ClientAuthMiddleware())
		MatchmakingService.GET("/ticket/player/:accountId", fortnite.CreateMatchmakingServiceTicket)
	}

	Friends := router.Group("/friends/api")
	{

		Friends.GET("/v1/:accountId/summary", fortnite.GetPublicFriendsV2)
		Friends.PUT("/v1/:accountId/friends/:friendId/alias", fortnite.Ret204)
		Friends.DELETE("/v1/:accountId/friends/:friendId/alias", fortnite.DeleteFriend)
		Friends.GET("/public/friends/:accountId", fortnite.GetPublicFriends)
		Friends.POST("/public/friends/:accountId/:friendId", fortnite.AddPublicFriend)
		Friends.POST("/v1/:accountId/friends/:friendId", fortnite.AddPublicFriend)
		Friends.DELETE("/v1/:accountId/friends/:friendId", fortnite.DeleteFriend)
		Friends.DELETE("/public/friends/:accountId/:friendId", fortnite.DeleteFriend)
	}

	DBProxy := database.NewProxy()
	DBProxy.Init(router)

	Party := router.Group("/party/api/v1/Fortnite")
	{
		Party.Use(middleware.ClientAuthMiddleware())
		Party.GET("/user/:accountId/notifications/undelivered/count", party.GETNotificationsCount)
		Party.GET("/user/:accountId", party.GETUserParty)
		Party.POST("/parties", party.POSTCreateParty)
		Party.GET("/parties/:partyId", party.GETParty)
		Party.PATCH("/parties/:partyId", party.PATCHParty)
		Party.PATCH("/parties/:partyId/members/:accountId/meta", party.PATCHMemberMeta)
		Party.POST("/parties/:partyId/members/:accountId/join", party.POSTJoinParty)
		Party.DELETE("/parties/:partyId/members/:accountId", party.DELETEPartyMember)
		Party.POST("/user/:accountId/pings/:pingerId", party.POSTSendPing)
		Party.DELETE("/user/:accountId/pings/:pingerId", party.DELETEPing)
		Party.GET("/user/:accountId/pings/:pingerId/parties", party.GETUserPingerParties)
		Party.POST("/user/:accountId/pings/:pingerId/join", party.POSTJoinViaPing)
		Party.POST("/parties/:partyId/invites", party.POSTPartyInvite)
		Party.POST("/parties/:partyId/invites/:accountId", party.POSTPartyInviteToAccount)
		Party.DELETE("/parties/:partyId/invites/:accountId", party.DELETEPartyInvite)
		Party.POST("/parties/:partyId/invites/:accountId/decline", party.POSTDeclineInvite)
		Party.POST("/parties/:partyId/members/:accountId/promote", party.POSTPromoteMember)
		Party.POST("/members/:accountId/intentions/:senderId", party.POSTIntention)
	}

	router.Use(func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				utilities.LogWithTimestamp(color.RedString, "Error! %v", e.Err.Error())
			}

			if !c.Writer.Written() {
				utilities.Internal.ServerError().Apply(c.Writer)
			}
		}
	})

	router.NoRoute(func(c *gin.Context) {
		utilities.Basic.NotFound().Apply(c.Writer)
	})

	router.HandleMethodNotAllowed = true
	router.NoMethod(func(c *gin.Context) {
		utilities.Basic.MethodNotAllowed().Apply(c.Writer)
	})

	httpServer := &http.Server{
		Addr:              ":80",
		Handler:           router,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	httpsServer := &http.Server{
		Addr:              ":443",
		Handler:           router,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	http2Server := &http2.Server{
		MaxConcurrentStreams:         100,
		MaxReadFrameSize:             16384,
		IdleTimeout:                  120 * time.Second,
		MaxUploadBufferPerConnection: 32768,
		MaxUploadBufferPerStream:     16384,
	}

	http2.ConfigureServer(httpServer, http2Server)
	http2.ConfigureServer(httpsServer, http2Server)

	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			numGoroutines := runtime.NumGoroutine()

			if numGoroutines > 1000 {
				utilities.LogWithTimestamp(color.YellowString, "WARNING: High goroutine count: %d", numGoroutines)
			}
		}
	}()

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to run HTTP server: %v", err)
		}
	}()

	go func() {
		ticker := time.NewTicker(3 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			db, err := odin.Get()
			if err != nil {
				continue
			}

			sessions, err := odin.FindAll("Accounts_Sessions", func() interface{} {
				return &accounts.Session{}
			})
			if err != nil {
				continue
			}

			now := time.Now()
			deletedCount := 0

			for _, sessionData := range sessions {
				session, ok := sessionData.(*accounts.Session)
				if !ok {
					continue
				}

				if now.Sub(session.CreatedAt) > 24*time.Hour {
					err = db.Delete("Accounts_Sessions", session.ID)
					if err != nil {
						continue
					}
					deletedCount++
				}
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := httpsServer.ListenAndServeTLS("static/certs/server.crt", "static/certs/server.key"); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to run HTTPS server: %v", err)
		}
	}()

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		utilities.LogWithTimestamp(color.RedString, "HTTP server shutdown error: %v", err)
	}

	if err := httpsServer.Shutdown(ctx); err != nil {
		utilities.LogWithTimestamp(color.RedString, "HTTPS server shutdown error: %v", err)
	}

	odin.Close("xenon")
	odin.Close("xenon_profiles")
	odin.Close("xenon_comp")
	odin.Close("xenon_redeem")
}
