package discovery

import (
	"encoding/json"
	"os"
	"sync"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/utilities"
)

var (
	playlistCache     = make(map[string]map[string]any)
	playlistCacheMu   sync.RWMutex
	playlistCacheInit int32
)

func loadAllPlaylists() error {
	if atomic.LoadInt32(&playlistCacheInit) == 1 {
		return nil
	}

	playlistCacheMu.Lock()
	defer playlistCacheMu.Unlock()

	if atomic.LoadInt32(&playlistCacheInit) == 1 {
		return nil
	}

	files, err := os.ReadDir("static/assets/playlists")
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() || len(file.Name()) < 6 || file.Name()[len(file.Name())-5:] != ".json" {
			continue
		}
		name := file.Name()[:len(file.Name())-5]
		data, err := os.ReadFile("static/assets/playlists/" + file.Name())
		if err != nil {
			continue
		}
		var p map[string]any
		if err := json.Unmarshal(data, &p); err != nil {
			continue
		}
		playlistCache[name] = p
	}

	atomic.StoreInt32(&playlistCacheInit, 1)
	return nil
}

func GETAssetsFortPlaylistAthena(c *gin.Context) {
	if atomic.LoadInt32(&playlistCacheInit) == 0 {
		if err := loadAllPlaylists(); err != nil {
			utilities.Internal.ServerError().Apply(c.Writer)
			return
		}
	}

	playlist := c.Param("playlist")
	if playlist == "" {
		utilities.Internal.ValidationFailed().Apply(c.Writer)
		return
	}

	playlistCacheMu.RLock()
	p, ok := playlistCache[playlist]
	playlistCacheMu.RUnlock()

	if !ok {
		utilities.Internal.ValidationFailed().Apply(c.Writer)
		return
	}

	c.JSON(200, p)
}
