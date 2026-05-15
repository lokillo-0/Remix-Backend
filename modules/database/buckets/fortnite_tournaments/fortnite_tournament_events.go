package fortnite_tournaments

import (
	"time"

	"github.com/andr1ww/odin"
)

type Events struct {
	odin.Bucket      `bucket:"Fortnite_Tournament_Events" database:"xenon_comp"`
	AnnouncementTime time.Time `json:"announcementTime"`
	AppId            *string   `json:"appId"`
	BeginTime        time.Time `json:"beginTime"`
	DisplayDataId    string    `json:"displayDataId"`
	EndTime          time.Time `json:"endTime"`
	Environment      *string   `json:"environment"`
	EventGroup       string    `json:"eventGroup"`
	EventId          string    `json:"eventId"`
	EventWindows     []struct {
		AdditionalRequirements []interface{} `json:"additionalRequirements"`
		BeginTime              time.Time     `json:"beginTime"`
		BlackoutPeriods        []interface{} `json:"blackoutPeriods"`
		CanLiveSpectate        bool          `json:"canLiveSpectate"`
		CountdownBeginTime     time.Time     `json:"countdownBeginTime"`
		EndTime                time.Time     `json:"endTime"`
		EventTemplateId        string        `json:"eventTemplateId"`
		EventWindowId          string        `json:"eventWindowId"`
		IsTBD                  bool          `json:"isTBD"`
		Metadata               struct {
			RoundType                  string `json:"RoundType"`
			ThresholdToAdvanceDivision int64  `json:"ThresholdToAdvanceDivision"`
			DivisionRank               int    `json:"divisionRank"`
		} `json:"metadata"`
		PayoutDelay             int      `json:"payoutDelay"`
		RequireAllTokens        []string `json:"requireAllTokens"`
		RequireAllTokensCaller  []string `json:"requireAllTokensCaller"`
		RequireAnyTokens        []string `json:"requireAnyTokens"`
		RequireAnyTokensCaller  []string `json:"requireAnyTokensCaller"`
		RequireNoneTokensCaller []string `json:"requireNoneTokensCaller"`
		Round                   int      `json:"round"`
		ScoreLocations          []struct {
			ScoreId   string `json:"scoreId"`
			ScoreMode string `json:"scoreMode"`
		} `json:"scoreLocations"`
		TeammateEligibility string `json:"teammateEligibility"`
		Visibility          string `json:"visibility"`
	} `json:"eventWindows"`
	Link struct {
		Type    string `json:"type"`
		Code    string `json:"code"`
		Version int    `json:"version"`
	} `json:"link"`
	GameId   string `json:"gameId"`
	Metadata struct {
		TrackedStats        []string `json:"TrackedStats"`
		MinimumAccountLevel int      `json:"minimumAccountLevel"`
	} `json:"metadata"`
	PlatformMappings map[string]interface{} `json:"platformMappings"`
	Platforms        []string               `json:"platforms"`
	RegionMappings   map[string]interface{} `json:"regionMappings"`
	Regions          []string               `json:"regions"`
}
