package remix_server_admin

import (
	"encoding/json"
	"os"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/modules/database/buckets/fortnite"
	"github.com/remixfn/xenon/modules/database/buckets/remix"
	"github.com/remixfn/xenon/utilities"
)

const shopConfigPath = "assets/shop.json"

func adminAuth(c *gin.Context) bool {
	if c.GetHeader("Authorization") != utilities.GetConfig().ADMIN_KEY {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return false
	}
	return true
}

func POSTAdminCreateBanner(c *gin.Context) {
	if !adminAuth(c) { return }
	var body struct {
		Name  string `json:"name" binding:"required"`
		URL   string `json:"url" binding:"required"`
		Order int    `json:"order"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.String(400, "invalid body: %s", err.Error())
		return
	}

	banner := &remix.Banner{
		Bucket: odin.Bucket{ID: uuid.New().String()},
		Name:   body.Name,
		URL:    body.URL,
		Order:  body.Order,
	}

	if err := odin.Create(banner); err != nil {
		c.String(500, "failed to create banner: %s", err.Error())
		return
	}

	c.JSON(201, banner)
}

func DELETEAdminBanner(c *gin.Context) {
	if !adminAuth(c) { return }
	id := c.Param("id")
	if id == "" {
		c.String(400, "id required")
		return
	}

	var banner remix.Banner
	if err := odin.Find("Remix_Banners", id, &banner); err != nil {
		c.String(404, "banner not found")
		return
	}

	if err := banner.Bucket.Delete(&banner); err != nil {
		c.String(500, "failed to delete banner: %s", err.Error())
		return
	}

	c.String(200, "deleted")
}

func POSTRemixServerAdminCheck(c *gin.Context) {
	if !adminAuth(c) { return }
	c.JSON(200, gin.H{"admin": true})
}

func POSTRemixServerAdminRegister(c *gin.Context) {
	if !adminAuth(c) { return }

	var request struct {
		AccountID string `json:"account_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.String(400, "invalid body: %s", err.Error())
		return
	}

	admin := &remix.Admins{
		Bucket:    odin.Bucket{ID: request.AccountID},
		AccountID: request.AccountID,
	}

	if err := admin.Bucket.Save(admin); err != nil {
		c.String(500, "failed to register admin: %s", err.Error())
		return
	}

	c.JSON(200, admin)
}

func POSTUnbanAll(c *gin.Context) {
	if !adminAuth(c) { return }
	accs, err := odin.FindWhere("Accounts", nil, func() interface{} {
		return &accounts.Account{}
	})

	if err != nil {
		c.String(500, "failed to get accounts: %s", err.Error())
		return
	}

	for _, account := range accs {
		account := account.(*accounts.Account)
		account.Banned = false
		account.Bucket.Save(account)
	}

	c.String(200, "success")
}

func POSTRemixAdminCreatePlaylist(c *gin.Context) {
	if !adminAuth(c) { return }
	var body struct {
		PlaylistName         string `json:"playlist_name"`
		Enabled              bool   `json:"enabled"`
		IsDefault            bool   `json:"is_default"`
		VisibleWhenDisabled  bool   `json:"visible_when_disabled"`
		DisplayAsNew         bool   `json:"display_as_new"`
		CategoryIndex        int    `json:"category_index"`
		DisplayAsLimitedTime bool   `json:"display_as_limited_time"`
		DisplayPriority      int    `json:"display_priority"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.String(400, "invalid body: %s", err.Error())
		return
	}

	if body.PlaylistName == "" {
		c.String(400, "playlist_name is required")
		return
	}

	playlist := &fortnite.Playlist{
		Bucket:               odin.Bucket{ID: body.PlaylistName},
		PlaylistName:         body.PlaylistName,
		Enabled:              body.Enabled,
		IsDefault:            body.IsDefault,
		VisibleWhenDisabled:  body.VisibleWhenDisabled,
		DisplayAsNew:         body.DisplayAsNew,
		CategoryIndex:        body.CategoryIndex,
		DisplayAsLimitedTime: body.DisplayAsLimitedTime,
		DisplayPriority:      body.DisplayPriority,
	}

	if err := playlist.Bucket.Save(playlist); err != nil {
		c.String(500, "failed to create playlist: %s", err.Error())
		return
	}

	c.JSON(200, playlist)
}

func POSTRemixAdminDisablePlaylist(c *gin.Context) {
	if !adminAuth(c) { return }
	var body struct {
		PlaylistName string `json:"playlist_name"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.String(400, "invalid body: %s", err.Error())
		return
	}

	if body.PlaylistName == "" {
		c.String(400, "playlist_name is required")
		return
	}

	var playlist fortnite.Playlist
	if err := odin.Find("Playlists", body.PlaylistName, &playlist); err != nil {
		c.String(500, "failed to get playlist: %s", err.Error())
		return
	}

	playlist.Enabled = false

	if err := playlist.Bucket.Save(&playlist); err != nil {
		c.String(500, "failed to disable playlist: %s", err.Error())
		return
	}

	c.JSON(200, playlist)
}

func POSTRemixAdminEnablePlaylist(c *gin.Context) {
	if !adminAuth(c) { return }
	var body struct {
		PlaylistName string `json:"playlist_name"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.String(400, "invalid body: %s", err.Error())
		return
	}

	if body.PlaylistName == "" {
		c.String(400, "playlist_name is required")
		return
	}

	var playlist fortnite.Playlist
	if err := odin.Find("Playlists", body.PlaylistName, &playlist); err != nil {
		c.String(500, "failed to get playlist: %s", err.Error())
		return
	}

	playlist.Enabled = true

	if err := playlist.Bucket.Save(&playlist); err != nil {
		c.String(500, "failed to enable playlist: %s", err.Error())
		return
	}

	c.JSON(200, playlist)
}

func DELETEAdminBattlePass(c *gin.Context) {
	if !adminAuth(c) { return }
	accountID := c.Param("accountId")
	season := c.Param("season")
	if accountID == "" || season == "" {
		c.String(400, "accountId and season required")
		return
	}

	seasonKey := accountID + ":" + season
	var s accounts.Season
	if err := odin.Find("Accounts_Seasons", seasonKey, &s); err != nil {
		c.String(404, "season not found")
		return
	}

	s.PurchasedVip = false
	s.Bucket.ID = seasonKey
	if err := s.Bucket.Save(&s); err != nil {
		c.String(500, "failed to save: %s", err.Error())
		return
	}

	c.JSON(200, gin.H{"success": true, "purchasedVip": false})
}

func GETAdminShopConfig(c *gin.Context) {
	if !adminAuth(c) {
		return
	}

	content, err := os.ReadFile(shopConfigPath)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to read shop config"})
		return
	}

	c.JSON(200, gin.H{
		"content": string(content),
		"path":    shopConfigPath,
	})
}

func PUTAdminShopConfig(c *gin.Context) {
	if !adminAuth(c) {
		return
	}

	var body struct {
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}

	if !json.Valid([]byte(body.Content)) {
		c.JSON(400, gin.H{"error": "content must be valid JSON"})
		return
	}

	if err := os.WriteFile(shopConfigPath, []byte(body.Content), 0644); err != nil {
		c.JSON(500, gin.H{"error": "failed to save shop config"})
		return
	}

	c.JSON(200, gin.H{"message": "shop config saved"})
}
