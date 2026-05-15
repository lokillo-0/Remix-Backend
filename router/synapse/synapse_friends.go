package synapse

import (
	"net/http"
	"strings"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/utilities"
)

func Friends(c *gin.Context) {
	id := c.Param("id")

	var account accounts.Account
	if err := odin.Find("Accounts", id, &account); err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	friends, err := odin.FindWhere("Accounts_Friends", map[string]interface{}{
		"accountId": id,
	}, func() interface{} {
		return &accounts.Friends{}
	})

	if err != nil {
		utilities.Account.AccountNotFound().Apply(c.Writer)
		return
	}

	acceptedFriends := []string{}
	for _, friend := range friends {
		if f, ok := friend.(*accounts.Friends); ok {
			if strings.ToUpper(f.Status) == "ACCEPTED" {
				acceptedFriends = append(acceptedFriends, f.FriendId)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": acceptedFriends,
	})
}
