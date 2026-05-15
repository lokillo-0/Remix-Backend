package remix_server

import (
	"net/http"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/remixfn/xenon/modules/database/buckets/fortnite"
)

func PUTRemixServerHotfixes(c *gin.Context) {
	var request struct {
		Name    string `json:"name" binding:"required"`
		Value   string `json:"value" binding:"required"`
		Enabled bool   `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	hotfixes, err := odin.FindWhere("Hotfixes", map[string]interface{}{
		"name": request.Name,
	}, func() interface{} {
		return &fortnite.Hotfixes{}
	})

	var hotfix *fortnite.Hotfixes
	var isUpdate bool

	if err == nil && len(hotfixes) > 0 {
		hotfix = hotfixes[0].(*fortnite.Hotfixes)
		hotfix.Value = request.Value
		hotfix.Enabled = request.Enabled
		isUpdate = true
	} else {
		hotfixID := uuid.New().String()
		hotfix = &fortnite.Hotfixes{
			Bucket:  odin.Bucket{ID: hotfixID},
			Name:    request.Name,
			Value:   request.Value,
			Enabled: request.Enabled,
		}
		isUpdate = false
	}

	if err := odin.Create(hotfix); err != nil {
		action := "create"
		if isUpdate {
			action = "update"
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to " + action + " hotfix",
			"details": err.Error(),
		})
		return
	}

	action := "created"
	if isUpdate {
		action = "updated"
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"action":  action,
	})
}
