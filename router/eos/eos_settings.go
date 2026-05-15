package eos

import (
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GETEOSSettings(c *gin.Context) {
	data, err := ioutil.ReadFile("./static/epic-settings.json")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read settings"})
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}
