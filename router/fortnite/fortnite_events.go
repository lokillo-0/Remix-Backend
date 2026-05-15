package fortnite

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/remixfn/xenon/modules/database/buckets/fortnite"
	"github.com/remixfn/xenon/modules/database/buckets/fortnite_tournaments"
	"github.com/remixfn/xenon/utilities"
)

func stripBucketFields(v interface{}) map[string]interface{} {
	data, _ := json.Marshal(v)
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	delete(m, "id")
	delete(m, "created_at")
	delete(m, "updated_at")
	return m
}

func DownloadEventsViaAccountId(c *gin.Context) {
	accountId := c.Param("accountId")
	ua := utilities.Parse(c.Request.UserAgent())

	seasons := []int{}
	if ua == nil || ua.Season <= 0 {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	for i := 1; i <= ua.Season; i++ {
		seasons = append(seasons, i)
	}

	var event_tokens []fortnite_tournaments.Tokens
	ttokens, _ := odin.FindWhere(
		"Fortnite_Tournament_Tokens",
		map[string]interface{}{
			"account_id": accountId,
		},
		func() interface{} {
			return &fortnite_tournaments.Tokens{}
		},
	)

	existingTokens := make(map[int]bool)
	for _, p := range ttokens {
		t := *(p.(*fortnite_tournaments.Tokens))
		existingTokens[t.Season] = true
		event_tokens = append(event_tokens, t)
	}

	for _, season := range seasons {
		if !existingTokens[season] {
			newToken := fortnite_tournaments.Tokens{
				Bucket: odin.Bucket{
					ID: fmt.Sprintf("Token_%s", uuid.New().String()),
				},
				AccountId: accountId,
				Season:    season,
				Token:     fmt.Sprintf("ARENA_S%d_Division1", season),
			}
			if err := odin.Create(&newToken); err != nil {
			}
			event_tokens = append(event_tokens, newToken)
		}
	}

	var event_scores []fortnite_tournaments.Scores
	scores, _ := odin.FindWhere(
		"Fortnite_Tournament_Scores",
		map[string]interface{}{
			"account_id": accountId,
		},
		func() interface{} {
			return &fortnite_tournaments.Scores{}
		},
	)

	existingScores := make(map[int]bool)
	for _, s := range scores {
		sc := *(s.(*fortnite_tournaments.Scores))
		existingScores[sc.Season] = true
		event_scores = append(event_scores, sc)
	}

	for _, season := range seasons {
		if !existingScores[season] {
			newScore := fortnite_tournaments.Scores{
				Bucket:    odin.Bucket{ID: fmt.Sprintf("Score_%s", uuid.New().String())},
				AccountId: accountId,
				Season:    season,
				Type:      "Hype",
				Value:     0,
			}
			if err := odin.Create(&newScore); err != nil {
			}
			event_scores = append(event_scores, newScore)
		}
	}

	playlists, _ := odin.FindWhere("Playlists", map[string]interface{}{
		"enabled_tournament": true,
	}, func() interface{} {
		return &fortnite.Playlist{}
	})

	enabledPlaylists := make(map[string]bool)
	for _, playlist := range playlists {
		p := *(playlist.(*fortnite.Playlist))
		enabledPlaylists[strings.ToLower(p.Bucket.ID)] = true
	}
	noPlaylistFilter := len(enabledPlaylists) == 0

	allEvents, _ := odin.FindWhere(
		"Fortnite_Tournament_Events",
		map[string]interface{}{},
		func() interface{} {
			return &fortnite_tournaments.Events{}
		},
	)
	allTemplates, _ := odin.FindWhere(
		"Fortnite_Tournament_Templates",
		map[string]interface{}{},
		func() interface{} {
			return &fortnite_tournaments.Templates{}
		},
	)

	filteredTemplates := []fortnite_tournaments.Templates{}
	templatePlaylistMap := make(map[string]bool)

	for _, template := range allTemplates {
		templateObj := *(template.(*fortnite_tournaments.Templates))

		if strings.Contains(strings.ToLower(templateObj.EventTemplateId), "arena") {
			templateObj.EventTemplateId = strings.Replace(
				templateObj.EventTemplateId,
				"S8",
				fmt.Sprintf("S%d", ua.Season),
				-1,
			)
			templateObj.Bucket.ID = strings.Replace(templateObj.Bucket.ID, "S8", fmt.Sprintf("S%d", ua.Season), -1)
		}

		if noPlaylistFilter || enabledPlaylists[strings.ToLower(templateObj.PlaylistId)] {
			filteredTemplates = append(filteredTemplates, templateObj)
			templatePlaylistMap[templateObj.EventTemplateId] = true
		}
	}

	updatedEvents := []fortnite_tournaments.Events{}
	for _, event := range allEvents {
		eventObj := *(event.(*fortnite_tournaments.Events))

		if strings.Contains(strings.ToLower(eventObj.EventId), "arena") {
			eventObj.EventId = strings.Replace(eventObj.EventId, "S8", fmt.Sprintf("S%d", ua.Season), -1)
			eventObj.Bucket.ID = strings.Replace(eventObj.Bucket.ID, "S8", fmt.Sprintf("S%d", ua.Season), -1)
			eventObj.EventGroup = strings.Replace(eventObj.EventGroup, "S8", fmt.Sprintf("S%d", ua.Season), -1)

			for i := range eventObj.EventWindows {
				eventObj.EventWindows[i].EventTemplateId = strings.Replace(
					eventObj.EventWindows[i].EventTemplateId,
					"S8",
					fmt.Sprintf("S%d", ua.Season),
					-1,
				)
				eventObj.EventWindows[i].EventWindowId = strings.Replace(
					eventObj.EventWindows[i].EventWindowId,
					"S8",
					fmt.Sprintf("S%d", ua.Season),
					-1,
				)

				for j, token := range eventObj.EventWindows[i].RequireAllTokens {
					eventObj.EventWindows[i].RequireAllTokens[j] = strings.Replace(
						token,
						"S8",
						fmt.Sprintf("S%d", ua.Season),
						-1,
					)
				}

				for j, token := range eventObj.EventWindows[i].RequireNoneTokensCaller {
					eventObj.EventWindows[i].RequireNoneTokensCaller[j] = strings.Replace(
						token,
						"S8",
						fmt.Sprintf("S%d", ua.Season),
						-1,
					)
				}
			}
		}

		hasValidTemplate := noPlaylistFilter
		if !hasValidTemplate {
			for _, window := range eventObj.EventWindows {
				if templatePlaylistMap[window.EventTemplateId] {
					hasValidTemplate = true
					break
				}
			}
		}

		if hasValidTemplate {
			updatedEvents = append(updatedEvents, eventObj)
		}
	}

	persistentScores := make(map[string]int)
	for _, score := range event_scores {
		persistentScores[score.Type] = score.Value
	}

	tokens := make([]string, 0)
	for _, token := range event_tokens {
		tokens = append(tokens, token.Token)
	}

	teams := make(map[string][]string)
	for _, event := range updatedEvents {
		eventId := event.EventId

		for _, window := range event.EventWindows {
			windowId := window.EventWindowId
			teamKey := fmt.Sprintf("%s:%s", eventId, windowId)
			teams[teamKey] = []string{accountId}
		}
	}

	cleanEvents := make([]map[string]interface{}, 0, len(updatedEvents))
	for _, e := range updatedEvents {
		cleanEvents = append(cleanEvents, stripBucketFields(e))
	}

	cleanTemplates := make([]map[string]interface{}, 0, len(filteredTemplates))
	for _, t := range filteredTemplates {
		cleanTemplates = append(cleanTemplates, stripBucketFields(t))
	}

	resolvedWindowLocations := make(map[string][]string)
	for _, event := range updatedEvents {
		for _, window := range event.EventWindows {
			key := fmt.Sprintf("Fortnite:%s:%s", event.EventId, window.EventWindowId)
			resolvedWindowLocations[key] = []string{key}
		}
	}

	response := gin.H{
		"player": gin.H{
			"tokens":           tokens,
			"accountId":        accountId,
			"gameId":           "Fortnite",
			"teams":            teams,
			"pendingPayouts":   []interface{}{},
			"pendingPenalties": gin.H{},
			"persistentScores": persistentScores,
			"groupIdentity":    gin.H{},
		},
		"events":                  cleanEvents,
		"templates":               cleanTemplates,
		"scores":                  []interface{}{},
		"leaderboardDefs":         []interface{}{},
		"resolvedWindowLocations": resolvedWindowLocations,
	}

	c.JSON(200, response)
}

func BulkTeam(c *gin.Context) {
	c.JSON(200, gin.H{
		"teams": []interface{}{},
	})
}

func GetEventWindowData(c *gin.Context) {
	eventId := c.Param("eventId")
	windowId := c.Param("windowId")
	c.JSON(200, gin.H{
		"eventId":                eventId,
		"eventWindowId":          windowId,
		"entries":                []interface{}{},
		"liveSessionIDs":         []interface{}{},
		"updatedAt":              "2020-01-01T00:00:00.000Z",
		"liveLeaderboardVersion": 0,
	})
}
