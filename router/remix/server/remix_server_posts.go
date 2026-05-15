package remix_server

import (
	"net/http"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/remixfn/xenon/modules/database/buckets/fortnite"
	"github.com/remixfn/xenon/modules/database/buckets/remix"
	"github.com/remixfn/xenon/utilities"
)

func POSTRemixServerPosts(c *gin.Context) {
	var request struct {
		Title string `json:"title" binding:"required"`
		Date  string `json:"date" binding:"required"`
		Image string `json:"image" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	newPost := &remix.Posts{
		Bucket: odin.Bucket{ID: uuid.New().String()},
		Title:  request.Title,
		Date:   request.Date,
		Image:  []string{request.Image},
	}

	if err := odin.Create(newPost); err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	c.JSON(http.StatusCreated, newPost)
}

func POSTCreatePlaylistInfo(c *gin.Context) {
	var body struct {
		Type          string `json:"_type"`
		Image         string `json:"image"`
		PlaylistName  string `json:"playlist_name"`
		Hidden        bool   `json:"hidden"`
		Description   string `json:"description"`
		SpecialBorder string `json:"special_border"`
		Violator      string `json:"violator"`
		DisplayName   string `json:"display_name"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	newPlaylist := &fortnite.PlaylistInfo{
		Bucket:        odin.Bucket{ID: uuid.New().String()},
		Type:          body.Type,
		Image:         body.Image,
		PlaylistName:  body.PlaylistName,
		Hidden:        body.Hidden,
		Description:   body.Description,
		SpecialBorder: body.SpecialBorder,
		Violator:      body.Violator,
		DisplayName:   body.DisplayName,
	}

	if err := odin.Create(newPlaylist); err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	c.JSON(http.StatusCreated, newPlaylist)
}
