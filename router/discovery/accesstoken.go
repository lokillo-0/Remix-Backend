package discovery

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"

	"github.com/gin-gonic/gin"
)

func DiscoveryAccessToken(c *gin.Context) {
	userAgent := c.GetHeader("User-Agent")
	
	regex := regexp.MustCompile(`\+\+Fortnite\+Release-\d+\.\d+`)
	match := regex.FindString(userAgent)
	
	tokenBytes := make([]byte, 10)
	rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes) + "="
	
	c.JSON(200, gin.H{
		"branchName": match,
		"appId":      "Fortnite",
		"token":      token,
	})
}