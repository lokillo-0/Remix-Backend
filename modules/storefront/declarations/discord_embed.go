package declarations

type DiscordEmbed struct {
	Title     string    `json:"title"`
	Image     ImageData `json:"image"`
	Timestamp string    `json:"timestamp"`
}

type ImageData struct {
	URL string `json:"url"`
}

type DiscordWebhookPayload struct {
	Content  string         `json:"content"`
	Username string         `json:"username"`
	Embeds   []DiscordEmbed `json:"embeds"`
}
