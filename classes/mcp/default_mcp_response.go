package mcp

type DefaultMCPResponse struct {
	ProfileRevision            int         `json:"profileRevision"`
	ProfileId                  string      `json:"profileId"`
	ProfileChangesBaseRevision int         `json:"profileChangesBaseRevision"`
	ProfileChanges             interface{} `json:"profileChanges"`
	ProfileCommandRevision     int         `json:"profileCommandRevision"`
	ServerTime                 string      `json:"serverTime"`
	ResponseVersion            int         `json:"responseVersion"`
}
