package fortnite

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
)

type ReportRequest struct {
	Reason  string `json:"reason"`
	Details string `json:"details"`
}

type DiscordWebhookPayload struct {
	Embeds []DiscordEmbed `json:"embeds"`
}

type DiscordEmbed struct {
	Title  string              `json:"title"`
	Color  int                 `json:"color"`
	Fields []DiscordEmbedField `json:"fields"`
}

type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

func ReportPlayer(c *gin.Context) {
	unsafeReporter := c.Param("unsafeReporter")
	reportedPlayer := c.Param("reportedPlayer")

	var reportRequest ReportRequest
	if err := c.ShouldBindJSON(&reportRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report data"})
		return
	}

	var reportedPlayerUser accounts.Account
	if err := odin.Find("Accounts", reportedPlayer, &reportedPlayerUser); err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	var reporterPlayerUser accounts.Account
	if err := odin.Find("Accounts", unsafeReporter, &reporterPlayerUser); err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	reason := reportRequest.Reason
	if reason == "" {
		reason = "No reason provided"
	}

	details := reportRequest.Details
	if details == "" {
		details = "No details provided"
	}

	webhookURL := "https://discord.com/api/webhooks/1398667162675318885/iuuwzPfj06VA8dptMGC_3b2r5IjZ_wC599YelOHkyDvHx2sBv5C8xSaViqlHhRZLNfHm"

	payload := DiscordWebhookPayload{
		Embeds: []DiscordEmbed{
			{
				Title: "Remix Report",
				Color: 16711680,
				Fields: []DiscordEmbedField{
					{Name: "Reporter", Value: reporterPlayerUser.DisplayName, Inline: true},
					{Name: "Reported Player", Value: reportedPlayerUser.DisplayName, Inline: true},
					{Name: "Reason", Value: reason, Inline: false},
					{Name: "Details", Value: details, Inline: false},
				},
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create report"})
		return
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send report"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Discord webhook failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Report submitted successfully"})
}
