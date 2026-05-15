package remix_launcher

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/modules/database/buckets/remix"
	"github.com/remixfn/xenon/utilities"
)

var imageExts = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}

func GETLauncherBanners(c *gin.Context) {
	results, err := odin.FindAll("Remix_Banners", func() any {
		return &remix.Banner{}
	})

	// if DB has entries, return them sorted by order
	if err == nil && len(results) > 0 {
		banners := make([]*remix.Banner, 0, len(results))
		for _, r := range results {
			if b, ok := r.(*remix.Banner); ok {
				banners = append(banners, b)
			}
		}
		sort.Slice(banners, func(i, j int) bool {
			return banners[i].Order < banners[j].Order
		})
		out := make([]gin.H, 0, len(banners))
		for _, b := range banners {
			out = append(out, gin.H{"name": b.Name, "url": b.URL})
		}
		c.JSON(200, out)
		return
	}

	// fallback: serve image files from assets/banners/ on disk
	entries, readErr := os.ReadDir("assets/banners")
	if readErr != nil || len(entries) == 0 {
		c.JSON(200, []gin.H{})
		return
	}

	scheme := "https"
	host := c.Request.Host
	out := make([]gin.H, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if !imageExts[ext] {
			continue
		}
		out = append(out, gin.H{
			"name": e.Name(),
			"url":  scheme + "://" + host + "/assets/banners/" + e.Name(),
		})
	}
	c.JSON(200, out)
}

func POSTAdminUploadBanner(c *gin.Context) {
	const maxSize = 10 * 1024 * 1024 // 10MB

	key := utilities.GetConfig().ADMIN_KEY
	if key == "" || c.GetHeader("X-Admin-Key") != key {
		c.String(401, "unauthorized")
		return
	}

	name := c.Param("name")
	if name == "" {
		c.String(400, "name required")
		return
	}
	ext := strings.ToLower(filepath.Ext(name))
	if !imageExts[ext] {
		c.String(400, "unsupported file type")
		return
	}

	data, err := io.ReadAll(io.LimitReader(c.Request.Body, maxSize+1))
	if err != nil || len(data) == 0 {
		c.String(400, "failed to read body")
		return
	}
	if len(data) > maxSize {
		c.String(400, "file too large (max 10MB)")
		return
	}

	ct := http.DetectContentType(data)
	if !strings.HasPrefix(ct, "image/") {
		c.String(400, "not an image")
		return
	}

	os.MkdirAll("assets/banners", 0755)
	dest := filepath.Join("assets/banners", filepath.Base(name))
	if err := os.WriteFile(dest, data, 0644); err != nil {
		c.String(500, "failed to save: %s", err.Error())
		return
	}

	scheme := "https"
	c.JSON(200, gin.H{"url": scheme + "://" + c.Request.Host + "/assets/banners/" + filepath.Base(name)})
}
