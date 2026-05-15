package remix_server_fortnite

import (
	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/modules/database/buckets/fortnite_tournaments"
	"github.com/remixfn/xenon/utilities"
)

func POSTBroadcastEventRewards(c *gin.Context) {
	// TODO: implement logic for tournament payout
}

func POSTRemixServerCreateEvent(c *gin.Context) {
	var request struct {
		Event     fortnite_tournaments.Events      `json:"event" binding:"required"`
		Templates []fortnite_tournaments.Templates `json:"templates" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utilities.Internal.JsonParsingFailed().Apply(c.Writer)
		return
	}

	var existingEvent fortnite_tournaments.Events
	if err := odin.Find("Fortnite_Tournament_Events", request.Event.EventId, &existingEvent); err == nil {
		utilities.Internal.UnsupportedMediaType().Apply(c.Writer)
		return
	}

	createdTemplates := []string{}
	for _, template := range request.Templates {
		var existingTemplate fortnite_tournaments.Templates
		if err := odin.Find("Fortnite_Tournament_Templates", template.EventTemplateId, &existingTemplate); err == nil {
			utilities.Internal.UnknownRoute().Apply(c.Writer)
			return
		}

		template.Bucket.ID = template.EventTemplateId

		if err := template.Bucket.Save(template); err != nil {
			utilities.Internal.EosError().Apply(c.Writer)
			return
		}

		createdTemplates = append(createdTemplates, template.EventTemplateId)
	}

	request.Event.Bucket.ID = request.Event.EventId

	if err := request.Event.Bucket.Save(request.Event); err != nil {
		utilities.Internal.NotImplemented().Apply(c.Writer)
		return
	}

	c.JSON(200, gin.H{
		"eventId":     request.Event.EventId,
		"templateIds": createdTemplates,
	})
}

func DELETERemixServerDeleteEvent(c *gin.Context) {
	eventId := c.Param("eventId")

	var event fortnite_tournaments.Events
	if err := odin.Find("Fortnite_Tournament_Events", eventId, &event); err != nil {
		utilities.Internal.UnknownRoute().Apply(c.Writer)
		return
	}

	allTemplates, _ := odin.FindWhere("Fortnite_Tournament_Templates", map[string]interface{}{}, func() interface{} {
		return &fortnite_tournaments.Templates{}
	})
	for _, t := range allTemplates {
		tmpl := t.(*fortnite_tournaments.Templates)
		for _, w := range event.EventWindows {
			if tmpl.EventTemplateId == w.EventTemplateId {
				tmpl.Bucket.Delete(tmpl)
				break
			}
		}
	}

	event.Bucket.Delete(event)
	c.JSON(200, gin.H{"deleted": eventId})
}
