package mcp

type PastSeasons struct {
	SeasonNumber     int  `json:"seasonNumber"`
	NumWins          int  `json:"numWins"`
	NumHighBracket   int  `json:"numHighBracket"`
	NumLowBracket    int  `json:"numLowBracket"`
	SeasonXp         int  `json:"seasonXp"`
	SeasonLevel      int  `json:"seasonLevel"`
	BookXp           int  `json:"bookXp"`
	BookLevel        int  `json:"bookLevel"`
	PurchasedVIP     bool `json:"purchasedVIP"`
	NumRoyalRoyales  int  `json:"numRoyalRoyales"`
	SurvivorTier     int  `json:"survivorTier"`
	SurvivorPrestige int  `json:"survivorPrestige"`
}
