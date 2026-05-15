package matchmaking_handlers

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/andr1ww/odin"
	"github.com/remixfn/xenon/modules/database/buckets/fortnite"
	"github.com/remixfn/xenon/modules/matchmaking/messages"
	"github.com/remixfn/xenon/modules/matchmaking/types"
)

func SelectPlaylist(sessionID string, region string) (string, string, error) {
	if sessionID == "" {
		return "", "WAITING", fmt.Errorf("empty session ID")
	}

	types.ClientM.RLock()
	server := types.Sessions[sessionID]
	types.ClientM.RUnlock()

	if server == nil {
		return "", "WAITING", fmt.Errorf("session not found in memory: %s", sessionID)
	}

	if server.IsAssigning || server.IsSending || server.IsAssigned {
		return "", "WAITING", nil
	}

	var session *fortnite.Sessions
	var err error
	sessions, err := odin.FindWhere("GameSessions", map[string]interface{}{
		"session_id": sessionID,
	}, func() interface{} {
		return &fortnite.Sessions{}
	})
	if err != nil {
		return "", "WAITING", fmt.Errorf("failed to get session: %w", err)
	}
	if len(sessions) == 0 {
		return "", "WAITING", fmt.Errorf("no session found for session_id %s", sessionID)
	}
	session, _ = sessions[0].(*fortnite.Sessions)

	if region == "" {
		region = session.Region
	}

	dbsessions, err := odin.FindWhere("GameSessions", map[string]interface{}{
		"region": region,
	}, func() interface{} {
		return &fortnite.Sessions{}
	})

	if err != nil {
		return "", "WAITING", fmt.Errorf("failed to get sessions for region: %w", err)
	}

	types.PlaylistMutex.Lock()
	if _, exists := types.LastSelectedPlaylist[region]; !exists {
		types.LastSelectedPlaylist[region] = "playlist_showdownalt_solo"
	}
	if types.LastSelectedPlaylist[region] == "playlist_showdownalt_duos" {
		types.LastSelectedPlaylist[region] = "playlist_showdownalt_solo"
	}
	types.PlaylistMutex.Unlock()

	types.ClientM.RLock()
	playerCounts := make(map[string]int)
	players := make(map[string]*types.Client)
	for client := range types.Clients {
		if client.Payload.Region == region {
			playerCounts[client.Payload.Playlist]++
			players[client.Payload.AccountID] = client
		}
	}
	types.ClientM.RUnlock()

	if len(playerCounts) == 0 {
		return "", "WAITING", nil
	}

	serverCounts := make(map[string]int)
	for _, s := range dbsessions {
		if sessionObj, ok := s.(*fortnite.Sessions); ok {
			serverCounts[sessionObj.PlaylistName]++
		}
	}

	var attributes map[string]interface{}
	if err := json.Unmarshal([]byte(session.Attributes), &attributes); err != nil {
		return "", "WAITING", fmt.Errorf("failed to get session attributes: %w", err)
	}

	maxPlayersPerServer := 50
	if maxPlayers, ok := attributes["MaxPlayers"].(float64); ok {
		maxPlayersPerServer = int(maxPlayers)
	}

	type Metric struct {
		Playlist         string
		PlayerCount      int
		ServerCount      int
		PlayersPerServer float64
		NeedsServer      bool
	}

	var metrics []Metric

	for playlist, playerCount := range playerCounts {
		serverCount := serverCounts[playlist]
		playersPerServer := float64(playerCount)
		if serverCount > 0 {
			playersPerServer = float64(playerCount) / float64(serverCount)
		}

		needsServer := playersPerServer >= float64(maxPlayersPerServer) || serverCount == 0

		min := 1

		if playerCount >= min && needsServer {
			metric := Metric{
				Playlist:         playlist,
				PlayerCount:      playerCount,
				ServerCount:      serverCount,
				PlayersPerServer: playersPerServer,
				NeedsServer:      needsServer,
			}
			metrics = append(metrics, metric)
		}
	}

	if len(metrics) == 0 {
		return "", "WAITING", nil
	}

	sort.SliceStable(metrics, func(i, j int) bool {
		if metrics[i].NeedsServer != metrics[j].NeedsServer {
			return metrics[i].NeedsServer
		}
		return metrics[i].PlayerCount > metrics[j].PlayerCount
	})

	metric := metrics[0]

	types.PlaylistMutex.Lock()
	types.LastSelectedPlaylist[region] = metric.Playlist
	types.PlaylistMutex.Unlock()

	session.PlaylistName = metric.Playlist

	types.ClientM.Lock()
	if types.Sessions[session.SessionId] != nil {
		types.Sessions[session.SessionId].Playlist = metric.Playlist
		types.Sessions[session.SessionId].IsAssigning = true
		types.Sessions[session.SessionId].IsSending = true
	}
	types.ClientM.Unlock()

	session.Bucket.Save(session)

	types.ClientM.RLock()
	currentSession := types.Sessions[session.SessionId]
	types.ClientM.RUnlock()

	if currentSession != nil {
		if currentSession.Teams == nil {
			currentSession.Teams = make([][][]string, 0)
		}

		for _, player := range players {
			ids := strings.Split(player.Payload.PartyPlayerIDs, ",")

			teamIndex := -1
			for i, team := range currentSession.Teams {
				for _, playerEntry := range team {
					for _, existingId := range playerEntry {
						for _, newId := range ids {
							if existingId == newId {
								teamIndex = i
								break
							}
						}
						if teamIndex != -1 {
							break
						}
					}
					if teamIndex != -1 {
						break
					}
				}
				if teamIndex != -1 {
					break
				}
			}

			if teamIndex == -1 {
				teamIndex = len(currentSession.Teams)
				currentSession.Teams = append(currentSession.Teams, make([][]string, 0))
			}

			for _, id := range ids {
				exists := false
				for _, playerEntry := range currentSession.Teams[teamIndex] {
					for _, existingId := range playerEntry {
						if existingId == id {
							exists = true
							break
						}
					}
					if exists {
						break
					}
				}

				if !exists {
					playerEntry := []string{id}
					currentSession.Teams[teamIndex] = append(currentSession.Teams[teamIndex], playerEntry)
				}
			}
		}

		types.ClientM.Lock()
		types.Sessions[session.SessionId] = currentSession
		types.ClientM.Unlock()
	}

	time.Sleep(2000 * time.Millisecond)

	types.ClientM.RLock()
	finalSession := types.Sessions[sessionID]
	types.ClientM.RUnlock()

	if finalSession != nil && !finalSession.AssignMatchSent {
		payload := types.AssignMatchPayload{
			Name: "AssignMatch",
			Payload: types.AssignMatchPayloadData{
				Spectators:     make([]interface{}, 0),
				Teams:          finalSession.Teams,
				BucketId:       fmt.Sprintf("Fortnite:Fortnite:%s:0:%s:%s", session.BuildUniqueId, region, metric.Playlist),
				MatchId:        finalSession.MatchId,
				MatchOptions:   "",
				MatchOptionsV2: make(map[string]interface{}),
			},
		}

		msg, err := json.Marshal(payload)
		if err != nil {
			return "", "WAITING", fmt.Errorf("failed to marshal AssignMatch payload: %v", err)
		}

		if err := finalSession.Conn.WriteMessage(1, msg); err != nil {
			if strings.Contains(err.Error(), "i/o timeout") {
				finalSession.Conn.Close()
				types.ClientM.Lock()
				delete(types.Sessions, sessionID)
				types.ClientM.Unlock()
			}
			return "", "WAITING", fmt.Errorf("failed to send AssignMatch message: %v", err)
		}

		types.ClientM.Lock()
		if types.Sessions[sessionID] != nil {
			types.Sessions[sessionID].AssignMatchSent = true
		}
		types.ClientM.Unlock()
	}

	count := 0
	if finalSession != nil {
		for _, client := range GetAllClientsViaData(finalSession.Payload.Version, metric.Playlist, region) {
			if count >= 17 {
				break
			}
			client.SendQueue = false
			messages.SendSessionAssignment(client, sessionID)
			count++
		}
	}

	return metric.Playlist, "OK", nil
}
