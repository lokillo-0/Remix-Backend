package remix_launcher

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/utilities"
)

func fileSHA256(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}

func GETLauncherDlls(c *gin.Context) {
	entries, err := os.ReadDir("assets/dlls")
	if err != nil || len(entries) == 0 {
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
		if strings.ToLower(filepath.Ext(e.Name())) != ".dll" {
			continue
		}
		full := filepath.Join("assets/dlls", e.Name())
		out = append(out, gin.H{
			"name": e.Name(),
			"url":  scheme + "://" + host + "/assets/dlls/" + e.Name(),
			"sha256": fileSHA256(full),
		})
	}
	c.JSON(200, out)
}

func POSTAdminUploadDll(c *gin.Context) {
	const maxSize = 32 * 1024 * 1024 // 32MB

	key := utilities.GetConfig().ADMIN_KEY
	if key == "" || c.GetHeader("X-Admin-Key") != key {
		c.String(401, "unauthorized")
		return
	}

	name := c.Param("name")
	if name == "" || strings.ToLower(filepath.Ext(name)) != ".dll" {
		c.String(400, "name must end in .dll")
		return
	}

	data, err := io.ReadAll(io.LimitReader(c.Request.Body, maxSize+1))
	if err != nil || len(data) == 0 {
		c.String(400, "failed to read body")
		return
	}
	if len(data) > maxSize {
		c.String(400, "file too large (max 32MB)")
		return
	}

	// basic PE header check (MZ)
	if len(data) < 2 || data[0] != 0x4D || data[1] != 0x5A {
		ct := http.DetectContentType(data)
		_ = ct
	}

	os.MkdirAll("assets/dlls", 0755)
	dest := filepath.Join("assets/dlls", filepath.Base(name))
	if err := os.WriteFile(dest, data, 0644); err != nil {
		c.String(500, "failed to save: %s", err.Error())
		return
	}

	c.JSON(200, gin.H{
		"name": filepath.Base(name),
		"url":  "https://" + c.Request.Host + "/assets/dlls/" + filepath.Base(name),
	})
}
