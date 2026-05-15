package discovery

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/utilities"
)

func GETRegion(c *gin.Context) {
	ip := c.GetHeader("X-Forwarded-For")
	if ip == "" {
		ip = c.GetHeader("X-Real-IP")
	}
	if ip == "" {
		ip, _, _ = net.SplitHostPort(c.Request.RemoteAddr)
	}
	if strings.HasPrefix(ip, "127.") || strings.HasPrefix(ip, "192.168.") || strings.HasPrefix(ip, "10.") {
		ip = ""
	}

	resp, err := http.Get(fmt.Sprintf("http://ip-api.com/json/%s", ip))
	if err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var ipData map[string]interface{}
	json.Unmarshal(body, &ipData)

	if ipData["status"] == "fail" {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	countryCode := ipData["countryCode"].(string)
	country := ipData["country"].(string)
	region := ipData["regionName"].(string)
	regionCode := ipData["region"].(string)

	continentCode := "XX"
	continentName := "Unknown"
	if countryCode == "US" || countryCode == "CA" {
		continentCode = "NA"
		continentName = "North America"
	} else if countryCode == "BR" || countryCode == "AR" {
		continentCode = "SA"
		continentName = "South America"
	} else if countryCode == "DE" || countryCode == "FR" || countryCode == "ES" {
		continentCode = "EU"
		continentName = "Europe"
	} else if countryCode == "CN" || countryCode == "JP" || countryCode == "IN" {
		continentCode = "AS"
		continentName = "Asia"
	}

	isEU := countryCode == "AT" || countryCode == "BE" || countryCode == "BG" || countryCode == "HR" || countryCode == "CY" || countryCode == "CZ" || countryCode == "DK" || countryCode == "EE" || countryCode == "FI" || countryCode == "FR" || countryCode == "DE" || countryCode == "GR" || countryCode == "HU" || countryCode == "IE" || countryCode == "IT" || countryCode == "LV" || countryCode == "LT" || countryCode == "LU" || countryCode == "MT" || countryCode == "NL" || countryCode == "PL" || countryCode == "PT" || countryCode == "RO" || countryCode == "SK" || countryCode == "SI" || countryCode == "ES" || countryCode == "SE"

	c.JSON(200, gin.H{
		"continent": gin.H{
			"names": gin.H{
				"de":    continentName,
				"en":    continentName,
				"es":    continentName,
				"fr":    continentName,
				"ja":    continentName,
				"pt-BR": continentName,
				"ru":    continentName,
				"zh-CN": continentName,
			},
			"code":       continentCode,
			"geoname_id": 6255149,
		},
		"country": gin.H{
			"names": gin.H{
				"de":    country,
				"en":    country,
				"es":    country,
				"fr":    country,
				"ja":    country,
				"pt-BR": country,
				"ru":    country,
				"zh-CN": country,
			},
			"iso_code":             countryCode,
			"geoname_id":           6252001,
			"is_in_european_union": isEU,
		},
		"subdivisions": []gin.H{
			{
				"names": gin.H{
					"de":    region,
					"en":    region,
					"es":    region,
					"fr":    region,
					"ja":    region,
					"pt-BR": region,
					"ru":    region,
					"zh-CN": region,
				},
				"iso_code":   regionCode,
				"geoname_id": 5128638,
			},
		},
	})
}
