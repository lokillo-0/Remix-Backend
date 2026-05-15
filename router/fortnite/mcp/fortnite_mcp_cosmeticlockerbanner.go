package fortnite_mcp

import (
	"fmt"
	"net/http"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/classes/mcp"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
)

func POSTSetCosmeticLockerBanner(c *gin.Context) {
	accountID := c.Param("accountId")

	profileID := c.Query("profileId")
	if profileID == "" {
		profileID = "athena"
	}

	var body struct {
		LockerItem              string `json:"lockerItem" binding:"required"`
		BannerIconTemplateName  string `json:"bannerIconTemplateName" binding:"required"`
		BannerColorTemplateName string `json:"bannerColorTemplateName" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		utilities.MCP.InvalidPayload().WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	var user accounts.Account
	userErr := odin.Find("Accounts", accountID, &user)

	if userErr != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	profileKey := fmt.Sprintf("%s:%s", accountID, profileID)
	var profile accounts.Profile
	if err := odin.Find("Accounts_Profiles", profileKey, &profile); err != nil {
		utilities.MCP.ProfileNotFound().Apply(c.Writer)
		return
	}

	lockerItem, exists := profile.Items[body.LockerItem]
	if !exists {
		utilities.MCP.ItemNotFound().WithIntent(utilities.Prod).Apply(c.Writer)
		return
	}

	itemData := lockerItem.(map[string]interface{})

	attributes, ok := itemData["attributes"].(map[string]interface{})
	if !ok {
		attributes = make(map[string]interface{})
		itemData["attributes"] = attributes
	}

	baseRevision := profile.Revision

	attributes["banner_color_template"] = body.BannerColorTemplateName
	attributes["banner_icon_template"] = body.BannerIconTemplateName

	profile.Items[body.LockerItem] = itemData

	profileChanges := []map[string]interface{}{
		{
			"changeType":     "itemAttrChanged",
			"itemId":         body.LockerItem,
			"attributeName":  "banner_color_template",
			"attributeValue": body.BannerColorTemplateName,
		},
		{
			"changeType":     "itemAttrChanged",
			"itemId":         body.LockerItem,
			"attributeName":  "banner_icon_template",
			"attributeValue": body.BannerIconTemplateName,
		},
	}

	if profile.Bucket.ID != profileKey {
		profile.Bucket.ID = profileKey
	}

	profile.Revision++

	c.JSON(http.StatusOK, mcp.DefaultMCPResponse{
		ProfileChanges:             profileChanges,
		ProfileId:                  profileID,
		ProfileRevision:            profile.Revision,
		ProfileChangesBaseRevision: baseRevision,
		ProfileCommandRevision:     profile.Revision,
		ResponseVersion:            1,
		ServerTime:                 time.Now().UTC().Format("2006-01-02T15:04:05.999Z"),
	})
	profile.Bucket.Save(profile)
}
