package utilities

import (
	"log"
	"strconv"
	"strings"
)

type SeasonInfo struct {
	OS     string `json:"os"`
	Season int    `json:"season"`
	Build  string `json:"build"`
	CL     string `json:"cl"`
	Lobby  string `json:"lobby"`
}

type UserAgent struct {
	VersionId  string `json:"versionId"`
	VersionStr string `json:"versionStr"`
}

func Parse(userAgent string) *SeasonInfo {
	buildID := getBuildID(userAgent)
	buildString := getBuildString(userAgent)

	if buildID != "" && buildString != "" {
		return handleValidBuild(&UserAgent{
			VersionId:  buildID,
			VersionStr: buildString,
		}, getPlatform(userAgent))
	}

	log.Printf("Failed to parse Build ID and Build String from user agent: %s", userAgent)
	return nil
}

func getBuildID(userAgent string) string {
	var parts []string
	if strings.Contains(userAgent, "CL-") {
		parts = strings.Split(userAgent, "CL-")
	} else if strings.Contains(userAgent, "-") {
		parts = strings.Split(userAgent, "-")
	}

	if len(parts) > 1 {
		subParts := strings.Split(parts[1], " ")
		return strings.Split(subParts[0], "+")[0]
	}

	return ""
}

func getPlatform(userAgent string) string {
	lastSpaceIndex := strings.LastIndex(userAgent, " ")
	if lastSpaceIndex != -1 && lastSpaceIndex < len(userAgent)-1 {
		return userAgent[lastSpaceIndex+1:]
	}
	return ""
}

func getBuildString(userAgent string) string {
	parts := strings.Split(userAgent, "Release-")
	if len(parts) <= 1 || len(parts[1]) == 0 {
		return ""
	}

	subParts := strings.Split(parts[1], "-")
	if len(subParts) == 0 {
		return ""
	}

	version := strings.TrimSpace(subParts[0])
	return version
}

func parseNetCL(versionId string) float64 {
	netcl, err := strconv.ParseFloat(versionId, 64)
	if err != nil {
		netcl = 0
	}
	return netcl
}

func handleValidBuild(userAgentInfo *UserAgent, os string) *SeasonInfo {
	netcl := parseNetCL(userAgentInfo.VersionId)
	season, buildVersion := parseVersion(userAgentInfo.VersionStr)

	lobby := getLobby(netcl, season)

	result := &SeasonInfo{
		Season: season,
		Build:  buildVersion,
		CL:     strconv.FormatFloat(netcl, 'f', -1, 64),
		OS:     os,
		Lobby:  lobby,
	}
	return result
}

func parseVersion(versionStr string) (int, string) {
	if versionStr == "" {
		return 0, ""
	}

	versionStr = strings.TrimPrefix(versionStr, "Release-")

	versionParts := strings.Split(versionStr, ".")
	season := 0
	buildVersion := versionStr

	if len(versionParts) > 0 {
		majorVersion, err := strconv.Atoi(versionParts[0])
		if err == nil {
			season = majorVersion
			buildVersion = versionStr
		}
	}

	return season, buildVersion
}

func getLobby(netcl float64, season int) string {
	switch {
	case netcl == 0:
		return "LobbySeason0"
	case netcl < 3724489:
		return "Season0"
	case netcl <= 3790078:
		return "LobbySeason1"
	case season == 6:
		return "Lobby6"
	case season == 10:
		return "Lobby10"
	default:
		return "Lobby" + strconv.Itoa(season)
	}
}
