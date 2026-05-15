package remix_launcher

import (
	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/modules/database/buckets/remix"
	"github.com/remixfn/xenon/utilities"
)

func GETLauncherUpdater(c *gin.Context) {
	updates, err := odin.FindAll("Remix_Updates", func() any {
		return &remix.Update{}
	})

	if err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		c.Abort()
		return
	}

	if len(updates) == 0 {
		utilities.Internal.ServerError().Apply(c.Writer)
		c.Abort()
		return
	}

	newest := updates[0].(*remix.Update)
	for _, u := range updates {
		update, ok := u.(*remix.Update)
		if !ok {
			continue
		}
		if update.Version >= newest.Version {
			newest = update
		}
	}

	c.JSON(200, gin.H{
		"version":   newest.Version,
		"pub_date":  newest.PublishDate,
		"url":       newest.Url,
		"signature": newest.Signature,
		"notes":     newest.Notes,
	})
}
