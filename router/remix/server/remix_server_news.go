package remix_server

import (
	"net/http"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/remixfn/xenon/modules/database/buckets/remix"
	"github.com/remixfn/xenon/utilities"
)

func POSTRemixServerNews(c *gin.Context) {
	var request struct {
		Title           string          `json:"title" binding:"required"`
		Subtitle        string          `json:"subtitle"`
		Description     string          `json:"description" binding:"required"`
		BackgroundImage string          `json:"backgroundImage"`
		LogoImage       string          `json:"logoImage"`
		ShowProgress    bool            `json:"showProgress"`
		NewsCards       []remix.NewsCard `json:"newsCards"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	newNews := &remix.News{
		Bucket:          odin.Bucket{ID: uuid.New().String()},
		Title:           request.Title,
		Subtitle:        request.Subtitle,
		Description:     request.Description,
		BackgroundImage: request.BackgroundImage,
		LogoImage:       request.LogoImage,
		ShowProgress:    request.ShowProgress,
		NewsCards:       request.NewsCards,
	}

	if err := odin.Create(newNews); err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	c.JSON(http.StatusCreated, newNews)
}
