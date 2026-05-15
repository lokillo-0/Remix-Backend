package fortnite

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	"github.com/remixfn/xenon/modules/synapse"
	"github.com/remixfn/xenon/utilities"
)

var (
	userCache     []accounts.Account
	userCacheMu   sync.RWMutex
	userCacheTime time.Time
)

func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func refreshUserCache() {
	all, err := odin.FindAll("Accounts", func() interface{} { return &accounts.Account{} })
	if err != nil {
		return
	}
	userCacheMu.Lock()
	defer userCacheMu.Unlock()
	userCache = userCache[:0]
	for _, u := range all {
		if acc, ok := u.(*accounts.Account); ok && !acc.Banned {
			userCache = append(userCache, *acc)
		}
	}
	userCacheTime = time.Now()
}

func SearchUsersByPrefix(c *gin.Context) {
	prefix := strings.ToLower(c.Query("prefix"))
	if prefix == "" {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}
	userCacheMu.RLock()
	stale := time.Since(userCacheTime) > 30*time.Second
	userCacheMu.RUnlock()
	if stale {
		go refreshUserCache()
	}
	userCacheMu.RLock()
	defer userCacheMu.RUnlock()
	matches := make([]gin.H, 0, 20)
	for i := range userCache {
		if strings.HasPrefix(strings.ToLower(userCache[i].DisplayName), prefix) {
			matches = append(matches, gin.H{"accountId": userCache[i].ID, "displayName": userCache[i].DisplayName})
			if len(matches) >= 100 {
				break
			}
		}
	}
	c.JSON(http.StatusOK, matches)
}

func loadFriends(accountId string) ([]*accounts.Friends, error) {
	results, err := odin.FindWhere("Accounts_Friends", map[string]interface{}{
		"accountId": accountId,
	}, func() interface{} { return &accounts.Friends{} })
	if err != nil {
		return nil, err
	}
	friends := make([]*accounts.Friends, 0, len(results))
	for _, r := range results {
		if f, ok := r.(*accounts.Friends); ok {
			friends = append(friends, f)
		}
	}
	return friends, nil
}

func findFriendRecord(accountId, friendId string) (*accounts.Friends, error) {
	friends, err := loadFriends(accountId)
	if err != nil {
		return nil, err
	}
	for _, f := range friends {
		if f.FriendId == friendId {
			return f, nil
		}
	}
	return nil, nil
}

func GetPublicFriends(c *gin.Context) {
	accountId := c.Param("accountId")

	var user accounts.Account
	if err := odin.Find("Accounts", accountId, &user); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	friends, err := loadFriends(accountId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	response := []gin.H{}

	for _, friend := range friends {
		createdTime, err := time.Parse(time.RFC3339, friend.Created)
		if err != nil {
			createdTime = time.Now()
		}
		ts := createdTime.UTC().Format(time.RFC3339)

		if friend.Status == "ACCEPTED" {
			response = append(response, gin.H{
				"accountId": friend.FriendId,
				"status":    "ACCEPTED",
				"direction": "OUTBOUND",
				"created":   ts,
				"favorite":  false,
			})
		} else if friend.Status == "PENDING" && friend.Direction == "OUTBOUND" {
			response = append(response, gin.H{
				"accountId": friend.FriendId,
				"status":    "PENDING",
				"direction": "OUTBOUND",
				"created":   ts,
				"favorite":  false,
			})
		} else if friend.Status == "PENDING" && friend.Direction == "INBOUND" {
			response = append(response, gin.H{
				"accountId": friend.FriendId,
				"status":    "PENDING",
				"direction": "INBOUND",
				"created":   ts,
				"favorite":  false,
			})
		}
	}

	c.JSON(http.StatusOK, response)
}

func GetPublicFriendsV2(c *gin.Context) {
	accountId := c.Param("accountId")

	response := gin.H{
		"friends":   []gin.H{},
		"incoming":  []gin.H{},
		"outgoing":  []gin.H{},
		"suggested": []gin.H{},
		"blocklist": []gin.H{},
		"settings": gin.H{
			"acceptInvites": "public",
		},
	}

	var user accounts.Account
	if err := odin.Find("Accounts", accountId, &user); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	friends, err := loadFriends(accountId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	for _, friend := range friends {
		createdTime, err := time.Parse(time.RFC3339, friend.Created)
		if err != nil {
			createdTime = time.Now()
		}
		ts := createdTime.UTC().Format(time.RFC3339)

		switch friend.Status {
		case "ACCEPTED":
			response["friends"] = append(response["friends"].([]gin.H), gin.H{
				"accountId": friend.FriendId,
				"groups":    []string{},
				"mutual":    0,
				"alias":     friend.Alias,
				"note":      "",
				"favorite":  false,
				"created":   ts,
			})
		case "PENDING":
			if friend.Direction == "INBOUND" {
				response["incoming"] = append(response["incoming"].([]gin.H), gin.H{
					"accountId": friend.FriendId,
					"mutual":    0,
					"favorite":  false,
					"created":   ts,
				})
			} else if friend.Direction == "OUTBOUND" {
				response["outgoing"] = append(response["outgoing"].([]gin.H), gin.H{
					"accountId": friend.FriendId,
					"favorite":  false,
					"created":   ts,
				})
			}
		}
	}

	c.JSON(http.StatusOK, response)
}

func AddPublicFriend(c *gin.Context) {
	accountId := c.Param("accountId")
	friendId := c.Param("friendId")

	if accountId == friendId {
		utilities.Friends.SelfFriend().Apply(c.Writer)
		return
	}

	var fromUser accounts.Account
	if err := odin.Find("Accounts", accountId, &fromUser); err != nil {
		utilities.Friends.AccountNotFound().Apply(c.Writer)
		return
	}

	var toUser accounts.Account
	if err := odin.Find("Accounts", friendId, &toUser); err != nil {
		utilities.Friends.AccountNotFound().Apply(c.Writer)
		return
	}
	_ = toUser

	senderRecord, err := findFriendRecord(accountId, friendId)
	if err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	receiverRecord, err := findFriendRecord(friendId, accountId)
	if err != nil {
		utilities.Internal.ServerError().Apply(c.Writer)
		return
	}

	if (senderRecord != nil && senderRecord.Status == "ACCEPTED") ||
		(receiverRecord != nil && receiverRecord.Status == "ACCEPTED") {
		utilities.Friends.RequestAlreadySent().Apply(c.Writer)
		return
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)

	if senderRecord != nil && senderRecord.Status == "PENDING" && senderRecord.Direction == "INBOUND" {
		senderRecord.Status = "ACCEPTED"
		if err := senderRecord.Bucket.Save(senderRecord); err != nil {
			utilities.Internal.ServerError().Apply(c.Writer)
			return
		}

		if receiverRecord != nil {
			receiverRecord.Status = "ACCEPTED"
			if err := receiverRecord.Bucket.Save(receiverRecord); err != nil {
				utilities.Internal.ServerError().Apply(c.Writer)
				return
			}
		} else {
			newRecord := &accounts.Friends{
				AccountId: friendId,
				FriendId:  accountId,
				Status:    "ACCEPTED",
				Direction: "OUTBOUND",
				Created:   timestamp,
			}
			newRecord.ID = generateID()
			if err := odin.Create(newRecord); err != nil {
				utilities.Internal.ServerError().Apply(c.Writer)
				return
			}
		}

		go func() {
			sendFriendXMPP(accountId, friendId, "ACCEPTED", "OUTBOUND")
			sendFriendXMPP(friendId, accountId, "ACCEPTED", "OUTBOUND")
			forwardPresence(accountId, friendId)
		}()

		c.JSON(http.StatusNoContent, nil)
		return
	}

	if senderRecord != nil && senderRecord.Status == "PENDING" && senderRecord.Direction == "OUTBOUND" {
		utilities.Friends.RequestAlreadySent().Apply(c.Writer)
		return
	}

	if senderRecord == nil && receiverRecord == nil {
		newOutbound := &accounts.Friends{
			AccountId: accountId,
			FriendId:  friendId,
			Status:    "PENDING",
			Direction: "OUTBOUND",
			Created:   timestamp,
		}
		newOutbound.ID = generateID()
		if err := odin.Create(newOutbound); err != nil {
			utilities.Internal.ServerError().Apply(c.Writer)
			return
		}

		newInbound := &accounts.Friends{
			AccountId: friendId,
			FriendId:  accountId,
			Status:    "PENDING",
			Direction: "INBOUND",
			Created:   timestamp,
		}
		newInbound.ID = generateID()
		if err := odin.Create(newInbound); err != nil {
			utilities.Internal.ServerError().Apply(c.Writer)
			return
		}

		go func() {
			sendFriendXMPP(accountId, friendId, "PENDING", "OUTBOUND")
			sendFriendXMPP(friendId, accountId, "PENDING", "INBOUND")
		}()

		c.JSON(http.StatusNoContent, nil)
		return
	}

	utilities.Internal.ServerError().Apply(c.Writer)
}

func sendFriendXMPP(toId, fromId, status, direction string) {
	sm := synapse.GetStartedInstance()
	if sm == nil {
		return
	}

	payload := struct {
		Payload struct {
			AccountId string `json:"accountId"`
			Status    string `json:"status"`
			Direction string `json:"direction"`
			Created   string `json:"created"`
			Favorite  bool   `json:"favorite"`
		} `json:"payload"`
		Type      string `json:"type"`
		Timestamp string `json:"timestamp"`
	}{
		Payload: struct {
			AccountId string `json:"accountId"`
			Status    string `json:"status"`
			Direction string `json:"direction"`
			Created   string `json:"created"`
			Favorite  bool   `json:"favorite"`
		}{
			AccountId: fromId,
			Status:    status,
			Direction: direction,
			Created:   time.Now().UTC().Format(time.RFC3339),
			Favorite:  false,
		},
		Type:      "com.epicgames.friends.core.apiobjects.Friend",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	sm.SendMessage(toId, payload)
}

func forwardPresence(user1, user2 string) {
	sm := synapse.GetStartedInstance()
	if sm == nil {
		return
	}
	sm.ForwardPresenceBothWays(user1, user2)
}

func SendFriendNotification(toId, fromId, status, direction string) error {
	sm := synapse.GetStartedInstance()
	if sm == nil {
		return fmt.Errorf("SynapseManager is not started")
	}
	sendFriendXMPP(toId, fromId, status, direction)
	if status == "ACCEPTED" {
		sm.ForwardPresenceBothWays(toId, fromId)
	}
	return nil
}

func DeleteFriend(c *gin.Context) {
	accountId := c.Param("accountId")
	friendId := c.Param("friendId")

	var user accounts.Account
	if err := odin.Find("Accounts", accountId, &user); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	var friendUser accounts.Account
	if err := odin.Find("Accounts", friendId, &friendUser); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Friend account not found"})
		return
	}
	_ = friendUser

	allFriends, err := odin.FindAll("Accounts_Friends", func() interface{} {
		return &accounts.Friends{}
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	for _, rawFriend := range allFriends {
		friend, ok := rawFriend.(*accounts.Friends)
		if !ok {
			continue
		}
		if (friend.AccountId == accountId && friend.FriendId == friendId) ||
			(friend.AccountId == friendId && friend.FriendId == accountId) {
			friend.Bucket.Delete(friend)
		}
	}

	go func() {
		sm := synapse.GetStartedInstance()
		if sm == nil {
			return
		}

		timestamp := time.Now().UTC().Format(time.RFC3339)

		removalPayload := struct {
			Payload struct {
				AccountId string `json:"accountId"`
				Reason    string `json:"reason"`
			} `json:"payload"`
			Type      string `json:"type"`
			Timestamp string `json:"timestamp"`
		}{
			Type:      "com.epicgames.friends.core.apiobjects.FriendRemoval",
			Timestamp: timestamp,
		}

		removalPayload.Payload.AccountId = friendId
		removalPayload.Payload.Reason = "DELETED"
		sm.SendMessage(accountId, removalPayload)

		removalPayload.Payload.AccountId = accountId
		sm.SendMessage(friendId, removalPayload)

		sm.ForwardOfflinePresenceBothWays(accountId, friendId)
	}()

	c.JSON(http.StatusNoContent, nil)
}
