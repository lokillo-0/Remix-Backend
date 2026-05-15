package fortnite

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/utilities"
)

type Event struct {
	EventType   string `json:"eventType"`
	ActiveUntil string `json:"activeUntil"`
	ActiveSince string `json:"activeSince"`
}

var seasonEvents = map[int][]Event{
	3: {
		{EventType: "EventFlag.Spring2018Phase1", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
	},
	4: {
		{EventType: "EventFlag.Blockbuster2018", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
		{EventType: "EventFlag.Blockbuster2018Phase1", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
	},
	6: {
		{EventType: "EventFlag.LTM_Fortnitemares", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
		{EventType: "EventFlag.FortnitemaresPhase1", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
		{EventType: "EventFlag.LTM_LilKevin", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
		{EventType: "EventFlag.LobbySeason6Halloween", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
	},
	8: {
		{EventType: "EventFlag.Spring2019", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
		{EventType: "EventFlag.Spring2019.Phase1", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
		{EventType: "EventFlag.LTM_Ashton", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
		{EventType: "EventFlag.LTM_Goose", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
		{EventType: "EventFlag.LTM_HighStakes", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
		{EventType: "EventFlag.LTE_BootyBay", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
		{EventType: "EventFlag.Spring2019.Phase2", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
	},
	10: {
		{EventType: "EventFlag.Mayday", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
		{EventType: "EventFlag.Season10.Phase2", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
		{EventType: "EventFlag.Season10.Phase3", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
		{EventType: "EventFlag.LTE_BlackMonday", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
		{EventType: "EventFlag.LTE_SharpShooter", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
		{EventType: "EventFlag.LTE_Fortnitemares2020", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
	},
	11: {
		{EventType: "EventFlag.Fortnitemares2020", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
		{EventType: "EventFlag.Halloween2020", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
		{EventType: "EventFlag.LobbyHalloween2020", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
		{EventType: "EventFlag.LobbySeason11Halloween", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
	},
	14: {
		{EventType: "EventFlag.Fortnitemares2021", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
		{EventType: "EventFlag.LobbyHalloween2021", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
	},
	15: {
		{EventType: "EventFlag.Season15", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
		{EventType: "EventFlag.Winterfest2021", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
		{EventType: "EventFlag.LobbyWinterfest2021", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
	},
	16: {
		{EventType: "EventFlag.Season16", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
	},
	17: {},
	18: {},
	19: {},
	32: {
		{EventType: "Week4", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
		{EventType: "KL2", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
		{EventType: "ClydeSeason3Part1", ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: "0001-01-01T00:00:00.000Z"},
	},
}

func Timeline(c *gin.Context) {
	ua := utilities.Parse(c.GetHeader("User-Agent"))

	if ua == nil {
		utilities.Basic.BadRequest().Apply(c.Writer)
		return
	}

	currentDateTime := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	seasonStr := strconv.Itoa(ua.Season)
	events := []Event{
		{EventType: "EventFlag.Season" + seasonStr, ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: currentDateTime},
		{EventType: "EventFlag.LobbySeason" + seasonStr, ActiveUntil: "9999-12-31T23:59:59.999Z", ActiveSince: currentDateTime},
	}

	if seasonEvents, ok := seasonEvents[ua.Season]; ok {
		events = append(events, seasonEvents...)
	}

	now := time.Now().UTC()
	currentDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	tomorrowStr := currentDay.AddDate(0, 0, 1).Format(time.RFC3339)

	c.JSON(http.StatusOK, gin.H{
		"channels": gin.H{
			"client-matchmaking": []gin.H{
				{
					"states":      []gin.H{},
					"cacheExpire": tomorrowStr,
				},
			},
			"client-events": gin.H{
				"states": []gin.H{
					{
						"validFrom":    "2020-01-01T00:00:00.000Z",
						"activeEvents": events,
						"state": gin.H{
							"activeStorefronts":  []gin.H{},
							"eventNamedWeights":  gin.H{},
							"seasonNumber":       ua.Season,
							"seasonTemplateId":   "AthenaSeason:athenaseason" + seasonStr,
							"matchXpBonusPoints": 0,
							"seasonBegin":        "2020-01-01T00:00:00.000Z",
							"seasonEnd":          "9999-12-31T23:59:59.999Z",
							"seasonDisplayedEnd": "9999-12-31T23:59:59.999Z",
							"weeklyStoreEnd":     tomorrowStr,
							"stwEventStoreEnd":   tomorrowStr,
							"stwWeeklyStoreEnd":  tomorrowStr,
							"sectionstoreEnds": gin.H{
								"Featured": tomorrowStr,
							},
							"dailyStoreEnd": tomorrowStr,
						},
					},
				},
				"cacheExpire": tomorrowStr,
			},
		},
		"eventsTimeOffsetHrs": 0,
		"cacheIntervalMins":   100,
		"currentTime":         currentDateTime,
	})
}
