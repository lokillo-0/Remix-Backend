package mcp

type Profile struct {
	Created         string                 `json:"created"`
	Updated         string                 `json:"updated"`
	Rvn             int                    `json:"rvn"`
	WipeNumber      int                    `json:"wipeNumber"`
	AccountId       string                 `json:"accountId"`
	ProfileId       string                 `json:"profileId"`
	Version         string                 `json:"version"`
	Items           map[string]interface{} `json:"items"`
	Stats           Stats                  `json:"stats"`
	CommandRevision int                    `json:"commandRevision"`
}

type Stats struct {
	Attributes map[string]interface{} `json:"attributes"`
}
