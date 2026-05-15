package account_public

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
)

var cache sync.Map

func getLoc(ip string) string {
	if v, ok := cache.Load(ip); ok {
		return v.(string)
	}
	go func() {
		resp, err := http.Get("http://ip-api.com/json/" + ip)
		if err != nil {
			return
		}
		defer resp.Body.Close()
		var d map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&d)
		city, _ := d["city"].(string)
		country, _ := d["country"].(string)
		loc := city
		if loc != "" && country != "" {
			loc += ", " + country
		} else {
			loc = country
		}
		cache.Store(ip, loc)
	}()
	return ""
}

func POSTPublicDeviceAuth(c *gin.Context) {
	id := c.Param("accountId")

	var account accounts.Account
	if err := odin.Find("Accounts", id, &account); err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	account.DeviceID = uuid.NewString()
	account.Bucket.Save(account)

	ip := c.ClientIP()
	location := getLoc(ip)

	c.JSON(http.StatusOK, gin.H{
		"deviceId":  account.DeviceID,
		"accountId": account.ID,
		"secret":    account.DeviceID,
		"userAgent": c.Request.UserAgent(),
		"created": gin.H{
			"location":  location,
			"ipAddress": ip,
			"dateTime":  time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		},
	})
}
