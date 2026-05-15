package utilities

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Config struct {
	JWTSecret            string
	Port                 string
	Prod                 bool
	DISCORD_BotToken     string
	Maintenance          bool
	Rewards              bool
	MatchMaker           string
	SeshSocket           string
	CFRegion             string
	CFAccessID           string
	CFSecretKey          string
	CFEndpoint           string
	SPassword            string
	SBaseURL             string
	SAPIToken            string
	CURRENT_VERSION      int
	DISCORD_CLIENTID     string
	DISCORD_CLIENTSECRET string
	DISCORD_REDIRECT_URI string
	GUILD_ID             string
	Current_QuestWeek    int
	LAUNCHER_SOCKET      string
	BACKEND_IP           string
	GAMESESSION_SECRET   string
	FREE_SHOP            bool
	ADMIN_KEY            string
	BETA_ROLE_IDS        []string
	VBUCKS_PER_KILL      int
	VBUCKS_PER_WIN       int
	SELLAUTH_SECRET      string
}

var config Config
var configMap map[string]interface{}
var CC *s3.Client

func Load() {
	config = Config{
		JWTSecret:            os.Getenv("JWTSECRET"),
		Port:                 os.Getenv("PORT"),
		DISCORD_BotToken:     os.Getenv("DISCORD_BOT_TOKEN"),
		MatchMaker:           os.Getenv("MATCHMAKER"),
		SeshSocket:           os.Getenv("MM_SESSION_SOCKET"),
		CFRegion:             os.Getenv("CF_REGION"),
		CFAccessID:           os.Getenv("CF_ACCESS_ID"),
		CFSecretKey:          os.Getenv("CF_SECRET_KEY"),
		CFEndpoint:           os.Getenv("CF_ENDPOINT"),
		SPassword:            os.Getenv("S_PASSWORD"),
		SBaseURL:             os.Getenv("S_BASEURL"),
		SAPIToken:            os.Getenv("S_APITOKEN"),
		DISCORD_CLIENTID:     os.Getenv("DISCORD_CLIENTID"),
		DISCORD_CLIENTSECRET: os.Getenv("DISCORD_CLIENTSECRET"),
		DISCORD_REDIRECT_URI: os.Getenv("DISCORD_REDIRECT_URI"),
		GUILD_ID:             os.Getenv("GUILD_ID"),
		LAUNCHER_SOCKET:      os.Getenv("LAUNCHER_SOCKET"),
		BACKEND_IP:           os.Getenv("BACKEND_IP"),
		GAMESESSION_SECRET:   os.Getenv("GAMESESSION_SECRET"),
		ADMIN_KEY:            os.Getenv("ADMIN_KEY"),
		SELLAUTH_SECRET:      os.Getenv("SELLAUTH_SECRET"),
	}

	if betaRoles := os.Getenv("BETA_ROLE_IDS"); betaRoles != "" {
		for _, r := range strings.Split(betaRoles, ",") {
			r = strings.TrimSpace(r)
			if r != "" {
				config.BETA_ROLE_IDS = append(config.BETA_ROLE_IDS, r)
			}
		}
	}

	config.Prod, _ = strconv.ParseBool(os.Getenv("PROD"))
	config.Maintenance, _ = strconv.ParseBool(os.Getenv("MAINTENANCE"))
	config.FREE_SHOP, _ = strconv.ParseBool(os.Getenv("FREE_SHOP"))
	config.Rewards, _ = strconv.ParseBool(os.Getenv("ALLOWREWARDS"))

	vbucksPerKillStr := os.Getenv("VBUCKS_PER_KILL")
	if vbucksPerKillStr == "" {
		config.VBUCKS_PER_KILL = 0
	} else {
		vpk, err := strconv.Atoi(vbucksPerKillStr)
		if err != nil {
			log.Fatalf("Error parsing VBUCKS_PER_KILL: %v", err)
		}
		config.VBUCKS_PER_KILL = vpk
	}

	vbucksPerWinStr := os.Getenv("VBUCKS_PER_WIN")
	if vbucksPerWinStr == "" {
		config.VBUCKS_PER_WIN = 0
	} else {
		vpk, err := strconv.Atoi(vbucksPerWinStr)
		if err != nil {
			log.Fatalf("Error parsing VBUCKS_PER_WIN: %v", err)
		}
		config.VBUCKS_PER_WIN = vpk
	}

	currentVersionStr := os.Getenv("CURRENT_VERSION")
	if currentVersionStr == "" {
		config.CURRENT_VERSION = 10
	} else {
		currentVersion, err := strconv.Atoi(currentVersionStr)
		if err != nil {
			log.Fatalf("Error parsing CURRENT_VERSION: %v", err)
		}
		config.CURRENT_VERSION = currentVersion
	}

	currentQuestWeekStr := os.Getenv("Current_QuestWeek")
	if currentQuestWeekStr == "" {
		config.Current_QuestWeek = 10
	} else {
		currentQuestWeek, err := strconv.Atoi(currentQuestWeekStr)
		if err != nil {
			log.Fatalf("Error parsing Current_QuestWeek: %v", err)
		}
		config.Current_QuestWeek = currentQuestWeek
	}

	configMap = map[string]interface{}{
		"jwt":           config.JWTSecret,
		"port":          config.Port,
		"prod":          config.Prod,
		"bot":           config.DISCORD_BotToken,
		"maintenance":   config.Maintenance,
		"matchmaker":    config.MatchMaker,
		"sesh_socket":   config.SeshSocket,
		"cf_region":     config.CFRegion,
		"cf_access_id":  config.CFAccessID,
		"cf_secret_key": config.CFSecretKey,
		"cf_endpoint":   config.CFEndpoint,
		"s_password":    config.SPassword,
		"s_baseurl":     config.SBaseURL,
		"s_apitoken":    config.SAPIToken,
		"client_id":     config.DISCORD_CLIENTID,
		"client_secret": config.DISCORD_CLIENTSECRET,
		"guild_id":      config.GUILD_ID,
		"socket":        config.LAUNCHER_SOCKET,
		"ip":            config.BACKEND_IP,
		"free":          config.FREE_SHOP,
		"gs_s":          config.GAMESESSION_SECRET,
	}
}

func Get[T any](key string) T {
	var zero T

	val, exists := configMap[key]
	if !exists {
		return zero
	}

	typedVal, ok := val.(T)
	if ok {
		return typedVal
	}

	return zero
}

func GetConfig() Config {
	return config
}
