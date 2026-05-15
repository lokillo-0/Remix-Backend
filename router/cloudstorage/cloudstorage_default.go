package cloudstorage

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/modules/database/buckets/fortnite"
	"github.com/remixfn/xenon/utilities"
)

func GetCloudstorage(c *gin.Context) {
	c.Header("Content-Type", "application/json")

	hotfixes, err := odin.FindWhere("Hotfixes", map[string]interface{}{
		"enabled": true,
	}, func() interface{} {
		return &fortnite.Hotfixes{}
	})

	if err != nil {
		utilities.MCP.InvalidPayload().
			WithIntent(utilities.Prod).
			Apply(c.Writer)
		return
	}

	response := make([]gin.H, 0, len(hotfixes))
	for _, hotfix := range hotfixes {
		hf, ok := hotfix.(*fortnite.Hotfixes)
		if !ok {
			continue
		}
		valueBytes := []byte(hf.Value)

		sha1Hasher := sha1.New()
		sha1Hasher.Write(valueBytes)
		sha1Hash := hex.EncodeToString(sha1Hasher.Sum(nil))

		sha256Hasher := sha256.New()
		sha256Hasher.Write(valueBytes)
		sha256Hash := hex.EncodeToString(sha256Hasher.Sum(nil))

		item := gin.H{
			"uniqueFilename": hf.Name,
			"filename":       hf.Name,
			"hash":           sha1Hash,
			"hash256":        sha256Hash,
			"length":         len(valueBytes),
			"contentType":    "application/octet-stream",
			"uploaded":       time.Now().UTC(),
			"storageType":    "S3",
			"storageIds":     gin.H{},
			"doNotCache":     true,
		}

		response = append(response, item)
	}

	c.JSON(http.StatusOK, response)
}

func GetCloudstorageFile(c *gin.Context) {
	filename := c.Param("filename")
	hotfixes, err := odin.FindWhere("Hotfixes", map[string]interface{}{
		"name": filename,
	}, func() interface{} {
		return &fortnite.Hotfixes{}
	})
	if err != nil || len(hotfixes) == 0 {
		utilities.MCP.InvalidPayload().
			WithIntent(utilities.Prod).
			Apply(c.Writer)
		return
	}
	hf, ok := hotfixes[0].(*fortnite.Hotfixes)
	if !ok {
		c.Status(http.StatusInternalServerError)
		return
	}

	if filename == "DefaultGame.ini" {
		lines := strings.Split(hf.Value, "\n")
		var filteredLines []string
		for _, line := range lines {
			if !strings.Contains(line, "+FrontEndPlaylistData=") {
				filteredLines = append(filteredLines, line)
			}
		}
		var result []string
		playlistInserted := false
		for _, line := range filteredLines {
			result = append(result, line)
			if strings.Contains(line, "!FrontEndPlaylistData=ClearArray") && !playlistInserted {
				playlists, err := odin.FindWhere("Playlists", map[string]interface{}{}, func() interface{} {
					return &fortnite.Playlist{}
				})
				if err == nil {
					for _, playlistInterface := range playlists {
						if playlist, ok := playlistInterface.(*fortnite.Playlist); ok {
							entry := fmt.Sprintf("+FrontEndPlaylistData=(PlaylistName=%s, PlaylistAccess=(bEnabled=%t, bIsDefaultPlaylist=%t, bVisibleWhenDisabled=%t, bDisplayAsNew=%t, CategoryIndex=%d, bDisplayAsLimitedTime=%t, DisplayPriority=%d))",
								playlist.PlaylistName,
								playlist.Enabled,
								playlist.IsDefault,
								playlist.VisibleWhenDisabled,
								playlist.DisplayAsNew,
								playlist.CategoryIndex,
								playlist.DisplayAsLimitedTime,
								playlist.DisplayPriority)
							result = append(result, entry)
						}
					}
				}
				playlistInserted = true
			}
		}
		c.String(http.StatusOK, strings.Join(result, "\n"))
		return
	}

	if filename == "DefaultEngine.ini" {
		beaconPatch := "\n[ConsoleVariables]\nFortMatchmakingV2.ContentBeaconFailureCancelsMatchmaking=0\nFort.ShutdownWhenContentBeaconFails=0\nFortMatchmakingV2.EnableContentBeacon=0\n"
		c.String(http.StatusOK, hf.Value+beaconPatch)
		return
	}

	c.String(http.StatusOK, hf.Value)
}
