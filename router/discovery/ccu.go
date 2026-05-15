package discovery

import (
	matchmaking_types "github.com/remixfn/xenon/modules/matchmaking/types"
)

var linkCodePlaylists = map[string][]string{
	"set_br_playlists":         {"playlist_defaultsolo", "playlist_defaultduo", "playlist_trios", "playlist_defaultsquad"},
	"set_blastberry_playlists": {"playlist_sunflowersolo"},
}

func playlistCCU(playlists ...string) int {
	matchmaking_types.ClientM.RLock()
	defer matchmaking_types.ClientM.RUnlock()
	count := 0
	for client := range matchmaking_types.Clients {
		for _, p := range playlists {
			if client.Payload.Playlist == p {
				count++
				break
			}
		}
	}
	return count
}
