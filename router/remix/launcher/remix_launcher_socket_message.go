package remix_launcher

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/andr1ww/odin"
	"github.com/google/uuid"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/modules/database/buckets/fortnite"
	"github.com/remixfn/xenon/modules/database/buckets/remix"
	"github.com/remixfn/xenon/utilities"
)

func HandleLauncherWebsocketMessage(client *LauncherSocketClient, messageType int, baseMsg []byte) bool {
	var message map[string]interface{}
	if err := json.Unmarshal(baseMsg, &message); err != nil {
		return false
	}
	switch message["name"] {
	case "ping":
		client.Conn.WriteMessage(messageType, []byte(`{"name":"pong"}`))
	case "request_count":
		var (
			updated time.Time
			ctotal  int
		)

		if !updated.IsZero() && time.Since(updated) < time.Minute {
			response := map[string]interface{}{
				"name":  "count",
				"count": ctotal,
			}
			responseBytes, _ := json.Marshal(response)
			client.Conn.WriteMessage(messageType, responseBytes)
			return true
		}

		res, err := http.Get("http://localhost:3000/bot/synapse/clients")
		if err != nil {
			break
		}
		defer res.Body.Close()

		var data map[string]struct {
			Data  []string `json:"data"`
			Count int      `json:"count"`
		}
		if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
			break
		}

		total := 0
		for _, platform := range data {
			total += platform.Count
		}

		updated = time.Now()
		ctotal = total

		response := map[string]interface{}{
			"name":  "count",
			"count": total,
		}
		responseBytes, _ := json.Marshal(response)
		client.Conn.WriteMessage(messageType, responseBytes)
	case "request_servers":
		ip := utilities.Get[string]("ip")
		resp, err := http.Get("http://" + ip + ":2087/nxa/echo/metrics/sessions")
		if err != nil {
			break
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			break
		}

		var sessions []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
			break
		}

		var servers []map[string]interface{}
		for _, s := range sessions {
			activePlayers, _ := s["ActivePlayers"].(float64)
			allPlayers, _ := s["AllPlayers"].(float64)
			id, _ := s["id"].(string)
			playlist, _ := s["Playlist"].(string)
			serverRegion, _ := s["ServerRegion"].(string)
			joinable, _ := s["Joinable"].(bool)
			available, _ := s["Available"].(bool)

			if playlist == "playlist_vamp_duo" {
				playlist = "playlist_vamp_duos"
			}

			started := activePlayers != 0 && !joinable && !available

			var status string
			switch {
			case activePlayers == 0 && !joinable && !available:
				status = "LOADING"
			case available && !joinable:
				status = "PREPARING"
			case joinable && available:
				status = "WARMUP"
			default:
				status = "STARTED"
			}

			maxPlayers := int(allPlayers)
			if attrs, ok := s["Attributes"].(map[string]interface{}); ok {
				if mp, ok := attrs["MaxPlayers"].(float64); ok {
					maxPlayers = int(mp)
				}
			}
			server := map[string]interface{}{
				"players":    int(activePlayers),
				"maxplayers": maxPlayers,
				"sessionid":  id,
				"playlist":   playlist,
				"started":    started,
				"region":     serverRegion,
				"status":     status,
			}
			servers = append(servers, server)
		}

		if len(servers) == 0 {
			servers = []map[string]interface{}{}
		}

		response := map[string]interface{}{
			"name":    "servers",
			"servers": servers,
		}
		responseBytes, err := json.Marshal(response)
		if err != nil {
			break
		}
		client.Conn.WriteMessage(messageType, responseBytes)
	case "request_pak_update":
		resp, err := http.Get("https://saturn.nxa.app/Remix/Content/pakchunkNxa1261/pakchunkNxa1261.manifest")
		if err != nil {
			response := map[string]interface{}{
				"name":  "pak_update",
				"error": "failed to fetch manifest",
			}
			responseBytes, _ := json.Marshal(response)
			client.Conn.WriteMessage(messageType, responseBytes)
			return false
		}
		defer resp.Body.Close()

		var manifest struct {
			Size int64 `json:"Size"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
			response := map[string]interface{}{
				"name":  "pak_update",
				"error": "failed to decode manifest",
			}
			responseBytes, _ := json.Marshal(response)
			client.Conn.WriteMessage(messageType, responseBytes)
			return false
		}

		response := map[string]interface{}{
			"name": "pak_update",
			"size": manifest.Size,
		}
		responseBytes, err := json.Marshal(response)
		if err != nil {
			errorResponse := map[string]interface{}{
				"name":  "error",
				"error": "failed to marshal pak update date",
			}
			errorBytes, _ := json.Marshal(errorResponse)
			client.Conn.WriteMessage(messageType, errorBytes)
			return false
		}
		client.Conn.WriteMessage(messageType, responseBytes)
	case "request_account":
		response := map[string]interface{}{
			"name":    "account",
			"account": client.Account,
		}
		responseBytes, err := json.Marshal(response)
		if err != nil {
			return false
		}
		client.Conn.WriteMessage(messageType, responseBytes)
	case "request_banners":
		banners, err := odin.FindAll("Remix_News", func() interface{} {
			return &remix.News{}
		})
		if err != nil {
			errorResponse := map[string]interface{}{
				"name":  "error",
				"error": "failed to fetch banners",
			}
			errorBytes, _ := json.Marshal(errorResponse)
			client.Conn.WriteMessage(messageType, errorBytes)
		}

		var newsItems []interface{}
		newsItems = append(newsItems, banners...)

		response := map[string]interface{}{
			"name":    "banners",
			"banners": newsItems,
		}
		responseBytes, err := json.Marshal(response)
		if err != nil {
			errorResponse := map[string]interface{}{
				"name":  "error",
				"error": "failed to marshal banners",
			}
			errorBytes, _ := json.Marshal(errorResponse)
			client.Conn.WriteMessage(messageType, errorBytes)

		}
		client.Conn.WriteMessage(messageType, responseBytes)
	case "request_posts":
		posts, err := odin.FindAll("Remix_Posts", func() interface{} {
			return &remix.Posts{}
		})
		if err != nil {
			errorResponse := map[string]interface{}{
				"name":  "error",
				"error": "failed to fetch posts",
			}
			errorBytes, _ := json.Marshal(errorResponse)
			client.Conn.WriteMessage(messageType, errorBytes)
			return false
		}
		var postsItems []interface{}
		postsItems = append(postsItems, posts...)

		response := map[string]interface{}{
			"name":  "posts",
			"posts": postsItems,
		}
		responseBytes, err := json.Marshal(response)
		if err != nil {
			errorResponse := map[string]interface{}{
				"name":  "error",
				"error": "failed to marshal posts",
			}
			errorBytes, _ := json.Marshal(errorResponse)
			client.Conn.WriteMessage(messageType, errorBytes)
			return false
		}
		client.Conn.WriteMessage(messageType, responseBytes)
	case "request_exchange":
		code := fortnite.Exchange{
			Bucket: odin.Bucket{
				ID: uuid.New().String(),
			},
			Code:      uuid.New().String(),
			AccountID: client.Account.ID,
			Created:   time.Now().Format(time.RFC3339),
		}
		if err := odin.Create(&code); err != nil {
			errorResponse := map[string]interface{}{
				"name":  "error",
				"error": "failed to create exchange code",
			}
			errorBytes, _ := json.Marshal(errorResponse)
			client.Conn.WriteMessage(messageType, errorBytes)
			return false
		}
		response := map[string]interface{}{
			"name": "exchange",
			"code": code.Code,
		}
		responseBytes, err := json.Marshal(response)
		if err != nil {
			errorResponse := map[string]interface{}{
				"name":  "error",
				"error": "failed to marshal exchange code",
			}
			errorBytes, _ := json.Marshal(errorResponse)
			client.Conn.WriteMessage(messageType, errorBytes)
			return false
		}

		client.Conn.WriteMessage(messageType, responseBytes)
	case "display_name_change":
		newDisplayName, ok := message["display_name"].(string)
		if !ok || newDisplayName == "" {
			errorResponse := map[string]interface{}{
				"name":  "error",
				"error": "invalid display name",
			}
			errorBytes, _ := json.Marshal(errorResponse)
			client.Conn.WriteMessage(messageType, errorBytes)
			return false
		}
		if newDisplayName == client.Account.DisplayName || len(newDisplayName) < 3 || len(newDisplayName) > 36 {
			errorResponse := map[string]interface{}{
				"name":  "error",
				"error": "display name must be between 3 and 16 characters and cannot be the same as the current display name",
			}
			errorBytes, _ := json.Marshal(errorResponse)
			client.Conn.WriteMessage(messageType, errorBytes)
			return false
		}

		existingUsernames, err := odin.FindWhere("Accounts", map[string]interface{}{
			"display_name": newDisplayName,
		}, func() interface{} {
			return &accounts.Account{}
		})
		if err == nil && len(existingUsernames) > 0 {
			errorResponse := map[string]interface{}{
				"name":  "error",
				"error": "account with this username already exists",
			}
			errorBytes, _ := json.Marshal(errorResponse)
			client.Conn.WriteMessage(messageType, errorBytes)
			return false
		}

		lastChangeTime, err := time.Parse(time.RFC3339, client.Account.LastDisplayNameChange)
		if err == nil && time.Since(lastChangeTime) < 7*24*time.Hour {
			errorResponse := map[string]interface{}{
				"name":  "error",
				"error": "display name can only be changed once every 7 days",
			}
			errorBytes, _ := json.Marshal(errorResponse)
			client.Conn.WriteMessage(messageType, errorBytes)
			return false
		}

		client.Account.DisplayName = newDisplayName
		client.Account.LastDisplayNameChange = time.Now().Format(time.RFC3339)
		client.Account.DisplayNameChanges++
		if err := client.Account.Bucket.Save(client.Account); err != nil {
			errorResponse := map[string]interface{}{
				"name":  "error",
				"error": "failed to save account",
			}
			errorBytes, _ := json.Marshal(errorResponse)
			client.Conn.WriteMessage(messageType, errorBytes)
			return false
		}
	}
	return true
}
