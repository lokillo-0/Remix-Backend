package fortnite

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func VersionCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"type": "NO_UPDATE",
	})
}

func EnabledFeatures(c *gin.Context) {
	c.JSON(http.StatusOK, []gin.H{})
}

func TryPlayOnPlatform(c *gin.Context) {
	c.String(http.StatusOK, "true")
}

func LightswitchBulk(c *gin.Context) {
	c.JSON(http.StatusOK, []gin.H{
		{
			"serviceInstanceId":  "fortnite",
			"status":             "UP",
			"message":            "fortnite is up.",
			"maintenanceUri":     nil,
			"overrideCatalogIds": []string{"a7f138b2e51945ffbfdacc1af0541053"},
			"allowedActions":     []string{"PLAY", "DOWNLOAD"},
			"banned":             false,
			"launcherInfoDTO": gin.H{
				"appName":       "Fortnite",
				"catalogItemId": "4fe75bbc5a674f4f9b356b5c90567da5",
				"namespace":     "fn",
			},
		},
	})
}

func Ret204(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func RetJson(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}

func GetContentControls(c *gin.Context) {
	accountId := c.Param("accountId")
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"ageGate":               0,
			"controlsEnabled":       false,
			"maxEpicProfilePrivacy": "none",
			"principalId":           accountId,
		},
	})
}
