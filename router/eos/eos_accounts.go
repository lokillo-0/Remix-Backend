package eos

import "github.com/gin-gonic/gin"

func GETEOSSDKAccounts(c *gin.Context) {
	accountId := c.Query("accountId")
	c.JSON(200, []gin.H{
		{
			"accountId":         accountId,
			"displayName":       accountId,
			"preferredLanguage": "en",
			"cabinedMode":       false,
			"empty":             false,
		},
	})
}
