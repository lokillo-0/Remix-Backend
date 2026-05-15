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

type FeedbackRequest struct {
	FeedbackType string `form:"feedbacktype"`
	AccountID    string `form:"accountid"`
	Subject      string `form:"subject"`
	FeedbackBody string `form:"feedbackbody"`
}

func SubmitBugFeedback(c *gin.Context) {
	var feedback FeedbackRequest
	if err := c.ShouldBind(&feedback); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid feedback data"})
		return
	}

	var user accounts.Account
	if err := odin.Find("Accounts", feedback.AccountID, &user); err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	webhookURL := "https://discord.com/api/webhooks/1398667162675318885/iuuwzPfj06VA8dptMGC_3b2r5IjZ_wC599YelOHkyDvHx2sBv5C8xSaViqlHhRZLNfHm"

	payload := DiscordWebhookPayload{
		Embeds: []DiscordEmbed{
			{
				Title: "Remix Bug Feedback",
				Color: 16711935,
				Fields: []DiscordEmbedField{
					{Name: "Feedback Type", Value: feedback.FeedbackType, Inline: true},
					{Name: "Username", Value: user.DisplayName, Inline: true},
					{Name: "Subject", Value: feedback.Subject, Inline: false},
					{Name: "Details", Value: feedback.FeedbackBody, Inline: false},
				},
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create feedback"})
		return
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send feedback"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Discord webhook failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Bug feedback submitted successfully"})
}

func SubmitCommentFeedback(c *gin.Context) {
	var feedback FeedbackRequest
	if err := c.ShouldBind(&feedback); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid feedback data"})
		return
	}

	var user accounts.Account
	if err := odin.Find("Accounts", feedback.AccountID, &user); err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	webhookURL := "https://discord.com/api/webhooks/1398667162675318885/iuuwzPfj06VA8dptMGC_3b2r5IjZ_wC599YelOHkyDvHx2sBv5C8xSaViqlHhRZLNfHm"

	payload := DiscordWebhookPayload{
		Embeds: []DiscordEmbed{
			{
				Title: "Remix Feedback", // idfk what to call this
				Color: 16711935,
				Fields: []DiscordEmbedField{
					{Name: "Feedback Type", Value: feedback.FeedbackType, Inline: true},
					{Name: "Username", Value: user.DisplayName, Inline: true},
					{Name: "Subject", Value: feedback.Subject, Inline: false},
					{Name: "Details", Value: feedback.FeedbackBody, Inline: false},
				},
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create feedback"})
		return
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send feedback"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Discord webhook failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Comment feedback submitted successfully"})
}
