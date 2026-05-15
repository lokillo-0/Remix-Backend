package account_public

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
)

func PostV3DeviceAuth(c *gin.Context) {
	var body map[string]interface{}
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		utilities.Basic.BadRequest().Apply(c.Writer)
		return
	}

	id := body["deviceId"].(string)

	var account accounts.Account
	if err := odin.Find("Accounts", id, &account); err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	account.DeviceID = uuid.NewString()
	account.Bucket.Save(account)

	ip := c.ClientIP()
	locationResp, err := http.Get("http://ip-api.com/json/" + ip)
	var locationData map[string]interface{}
	if err == nil && locationResp.StatusCode == http.StatusOK {
		defer locationResp.Body.Close()
		_ = json.NewDecoder(locationResp.Body).Decode(&locationData)
	}

	city, _ := locationData["city"].(string)
	country, _ := locationData["country"].(string)
	location := city
	if location != "" && country != "" {
		location = city + ", " + country
	} else if country != "" {
		location = country
	}

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
