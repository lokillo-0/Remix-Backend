package fortnite

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

var keychain interface{}

func Keychain(c *gin.Context) {
	if keychain == nil {
		data, err := os.ReadFile("static/storefront/keychain.json")
		if err != nil {
			panic(err)
		}
		if err := json.Unmarshal(data, &keychain); err != nil {
			panic(err)
		}
	}
	c.JSON(http.StatusOK, keychain)
}
