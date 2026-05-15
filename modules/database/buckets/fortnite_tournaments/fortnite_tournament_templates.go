package fortnite_tournaments

import "github.com/andr1ww/odin"

type Templates struct {
	odin.Bucket       `bucket:"Fortnite_Tournament_Templates" database:"xenon_comp"`
	EventTemplateId   string `json:"eventTemplateId"`
	GameId            string `json:"gameId"`
	MatchCap          int    `json:"matchCap"`
	PersistentScoreId string `json:"persistentScoreId"`
	PlaylistId        string `json:"playlistId"`
	ScoringRules      []struct {
		MatchRule   string `json:"matchRule"`
		RewardTiers []struct {
			KeyValue       int  `json:"keyValue"`
			Multiplicative bool `json:"multiplicative"`
			PointsEarned   int  `json:"pointsEarned"`
		} `json:"rewardTiers"`
		TrackedStat string `json:"trackedStat"`
	} `json:"scoringRules"`
}
