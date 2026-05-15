package eos

import (
	_ "embed"

	"github.com/gin-gonic/gin"
)

//go:embed eos.json
var eosConfigRaw []byte

func GETEOSSdk(c *gin.Context) {
	c.Data(200, "application/json", eosConfigRaw)
}
