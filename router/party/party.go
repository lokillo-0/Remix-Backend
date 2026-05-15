package party

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/remixfn/xenon/modules/database/buckets/accounts"
	s "github.com/remixfn/xenon/modules/synapse"
)

type Config struct {
	Type             string `json:"type"`
	Joinability      string `json:"joinability"`
	Discoverability  string `json:"discoverability"`
	SubType          string `json:"sub_type"`
	MaxSize          int    `json:"max_size"`
	InviteTTL        int    `json:"invite_ttl"`
	JoinConfirmation bool   `json:"join_confirmation"`
}

type Connection struct {
	ID              string                 `json:"id"`
	ConnectedAt     string                 `json:"connected_at"`
	UpdatedAt       string                 `json:"updated_at"`
	YieldLeadership bool                   `json:"yield_leadership"`
	Meta            map[string]interface{} `json:"meta"`
}

type Member struct {
	AccountID   string                 `json:"account_id"`
	Meta        map[string]interface{} `json:"meta"`
	Connections []Connection           `json:"connections"`
	Revision    int                    `json:"revision"`
	UpdatedAt   string                 `json:"updated_at"`
	JoinedAt    string                 `json:"joined_at"`
	Role        string                 `json:"role"`
}

type Invite struct {
	PartyID   string                 `json:"party_id"`
	SentBy    string                 `json:"sent_by"`
	Meta      map[string]interface{} `json:"meta"`
	SentTo    string                 `json:"sent_to"`
	SentAt    string                 `json:"sent_at"`
	UpdatedAt string                 `json:"updated_at"`
	ExpiresAt string                 `json:"expires_at"`
	Status    string                 `json:"status"`
}

type Ping struct {
	SentBy    string                 `json:"sent_by"`
	SentTo    string                 `json:"sent_to"`
	SentAt    string                 `json:"sent_at"`
	ExpiresAt string                 `json:"expires_at"`
	Meta      map[string]interface{} `json:"meta"`
}

type Intention struct {
	RequesterID   string                 `json:"requester_id"`
	RequesterDN   string                 `json:"requester_dn"`
	RequesterPL   string                 `json:"requester_pl"`
	RequesterPLDN string                 `json:"requester_pl_dn"`
	RequesteeID   string                 `json:"requestee_id"`
	Meta          map[string]interface{} `json:"meta"`
	ExpiresAt     string                 `json:"expires_at"`
	SentAt        string                 `json:"sent_at"`
}

type Party struct {
	ID         string                 `json:"id"`
	CreatedAt  string                 `json:"created_at"`
	UpdatedAt  string                 `json:"updated_at"`
	Config     Config                 `json:"config"`
	Members    []Member               `json:"members"`
	Applicants []interface{}          `json:"applicants"`
	Meta       map[string]interface{} `json:"meta"`
	Invites    []Invite               `json:"invites"`
	Revision   int                    `json:"revision"`
	Intentions []Intention            `json:"intentions"`
}

var (
	mu      sync.RWMutex
	parties = map[string]*Party{}
	pings   = []Ping{}
)

func nowStr() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.999Z")
}

func expiresStr() string {
	return time.Now().UTC().Add(1 * time.Hour).Format("2006-01-02T15:04:05.999Z")
}

func newID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}

func stripJID(id string) string {
	if idx := strings.Index(id, "@prod"); idx != -1 {
		return id[:idx]
	}
	return id
}

func getDisplayName(accountId string) string {
	var acc accounts.Account
	if err := odin.Find("Accounts", accountId, &acc); err != nil {
		return accountId
	}
	if acc.Username != "" {
		return acc.Username
	}
	return accountId
}

func sendXMPP(accountId string, payload interface{}) {
	sm := s.GetStartedInstance()
	if sm == nil {
		return
	}
	sm.SendMessage(accountId, payload)
}

func partyForUser(accountId string) *Party {
	for _, p := range parties {
		for _, m := range p.Members {
			if m.AccountID == accountId {
				return p
			}
		}
	}
	return nil
}

func removeFromParty(p *Party, accountId string) {
	newMembers := []Member{}
	for _, m := range p.Members {
		if m.AccountID != accountId {
			newMembers = append(newMembers, m)
		}
	}
	p.Members = newMembers
}

func getRSAKey(p *Party) string {
	if _, ok := p.Meta["Default:RawSquadAssignments_j"]; ok {
		return "Default:RawSquadAssignments_j"
	}
	return "RawSquadAssignments_j"
}

func configFromMap(m map[string]interface{}) Config {
	cfg := Config{
		Type:            "DEFAULT",
		Joinability:     "INVITE_AND_FORMER",
		Discoverability: "ALL",
		SubType:         "default",
		MaxSize:         16,
		InviteTTL:       14400,
	}
	if v, ok := m["type"].(string); ok {
		cfg.Type = v
	}
	if v, ok := m["joinability"].(string); ok {
		cfg.Joinability = v
	}
	if v, ok := m["discoverability"].(string); ok {
		cfg.Discoverability = v
	}
	if v, ok := m["sub_type"].(string); ok {
		cfg.SubType = v
	}
	if v, ok := m["max_size"].(float64); ok {
		cfg.MaxSize = int(v)
	}
	if v, ok := m["invite_ttl"].(float64); ok {
		cfg.InviteTTL = int(v)
	}
	if v, ok := m["join_confirmation"].(bool); ok {
		cfg.JoinConfirmation = v
	}
	return cfg
}

func GETNotificationsCount(c *gin.Context) {
	accountId := c.Param("accountId")
	mu.RLock()
	p := partyForUser(accountId)
	inviteCount := 0
	if p != nil {
		for _, inv := range p.Invites {
			if inv.SentTo == accountId {
				inviteCount++
			}
		}
	}
	pingCount := 0
	for _, pg := range pings {
		if pg.SentTo == accountId {
			pingCount++
		}
	}
	mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"pings": pingCount, "invites": inviteCount})
}

func GETUserParty(c *gin.Context) {
	accountId := c.Param("accountId")
	mu.RLock()
	var current []interface{}
	for _, p := range parties {
		for _, m := range p.Members {
			if m.AccountID == accountId {
				current = append(current, p)
				break
			}
		}
	}
	var userPings []map[string]interface{}
	for _, pg := range pings {
		if pg.SentTo == accountId {
			userPings = append(userPings, map[string]interface{}{
				"sent_by": pg.SentBy, "sent_to": pg.SentTo,
				"sent_at": pg.SentAt, "expires_at": pg.ExpiresAt, "meta": pg.Meta,
			})
		}
	}
	mu.RUnlock()
	if current == nil {
		current = []interface{}{}
	}
	if userPings == nil {
		userPings = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, gin.H{
		"current": current, "pending": []interface{}{},
		"invites": []interface{}{}, "pings": userPings,
	})
}

func POSTCreateParty(c *gin.Context) {
	var body struct {
		Config   map[string]interface{} `json:"config"`
		JoinInfo struct {
			Connection struct {
				ID              string                 `json:"id"`
				YieldLeadership bool                   `json:"yield_leadership"`
				Meta            map[string]interface{} `json:"meta"`
			} `json:"connection"`
			Meta map[string]interface{} `json:"meta"`
		} `json:"join_info"`
		Meta map[string]interface{} `json:"meta"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.JoinInfo.Connection.ID == "" {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	n := nowStr()
	connMeta := body.JoinInfo.Connection.Meta
	if connMeta == nil {
		connMeta = map[string]interface{}{}
	}
	memberMeta := body.JoinInfo.Meta
	if memberMeta == nil {
		memberMeta = map[string]interface{}{}
	}
	partyMeta := body.Meta
	if partyMeta == nil {
		partyMeta = map[string]interface{}{}
	}
	cfg := body.Config
	if cfg == nil {
		cfg = map[string]interface{}{}
	}

	id := newID()
	p := &Party{
		ID: id, CreatedAt: n, UpdatedAt: n,
		Config:     configFromMap(cfg),
		Meta:       partyMeta,
		Applicants: []interface{}{},
		Invites:    []Invite{},
		Intentions: []Intention{},
		Revision:   0,
		Members: []Member{{
			AccountID: stripJID(body.JoinInfo.Connection.ID),
			Meta:      memberMeta,
			Connections: []Connection{{
				ID: body.JoinInfo.Connection.ID, ConnectedAt: n, UpdatedAt: n,
				YieldLeadership: body.JoinInfo.Connection.YieldLeadership, Meta: connMeta,
			}},
			Revision: 0, UpdatedAt: n, JoinedAt: n, Role: "CAPTAIN",
		}},
	}
	mu.Lock()
	if old := partyForUser(p.Members[0].AccountID); old != nil {
		removeFromParty(old, p.Members[0].AccountID)
		if len(old.Members) == 0 {
			delete(parties, old.ID)
		}
	}
	parties[id] = p
	mu.Unlock()
	c.JSON(http.StatusOK, p)
}

func GETParty(c *gin.Context) {
	pid := c.Param("partyId")
	mu.RLock()
	p, ok := parties[pid]
	mu.RUnlock()
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"errorCode": "errors.com.epicgames.party.not_found"})
		return
	}
	c.JSON(http.StatusOK, p)
}

func PATCHParty(c *gin.Context) {
	pid := c.Param("partyId")
	userId := c.GetString("accountId")
	var body struct {
		Config map[string]interface{} `json:"config"`
		Meta   struct {
			Delete []string               `json:"delete"`
			Update map[string]interface{} `json:"update"`
		} `json:"meta"`
		Revision int `json:"revision"`
	}
	c.ShouldBindJSON(&body)

	mu.Lock()
	p, ok := parties[pid]
	if !ok {
		mu.Unlock()
		c.JSON(http.StatusNotFound, gin.H{"errorCode": "errors.com.epicgames.party.not_found"})
		return
	}
	for _, m := range p.Members {
		if m.AccountID == userId && m.Role != "CAPTAIN" {
			mu.Unlock()
			c.JSON(http.StatusForbidden, gin.H{"errorCode": "errors.com.epicgames.party.unauthorized"})
			return
		}
	}
	if body.Config != nil {
		for k, v := range body.Config {
			switch k {
			case "type":
				if s, ok := v.(string); ok {
					p.Config.Type = s
				}
			case "joinability":
				if s, ok := v.(string); ok {
					p.Config.Joinability = s
				}
			case "discoverability":
				if s, ok := v.(string); ok {
					p.Config.Discoverability = s
				}
			case "sub_type":
				if s, ok := v.(string); ok {
					p.Config.SubType = s
				}
			case "max_size":
				if f, ok := v.(float64); ok {
					p.Config.MaxSize = int(f)
				}
			case "invite_ttl":
				if f, ok := v.(float64); ok {
					p.Config.InviteTTL = int(f)
				}
			case "join_confirmation":
				if b, ok := v.(bool); ok {
					p.Config.JoinConfirmation = b
				}
			}
		}
	}
	for _, key := range body.Meta.Delete {
		delete(p.Meta, key)
	}
	for k, v := range body.Meta.Update {
		p.Meta[k] = v
	}
	p.Revision = body.Revision
	p.UpdatedAt = nowStr()

	captain := ""
	for _, m := range p.Members {
		if m.Role == "CAPTAIN" {
			captain = m.AccountID
			break
		}
	}
	members := make([]Member, len(p.Members))
	copy(members, p.Members)
	partyID := p.ID
	createdAt := p.CreatedAt
	maxSize := p.Config.MaxSize
	joinability := p.Config.Joinability
	subType, _ := p.Meta["urn:epic:cfg:party-type-id_s"].(string)
	revision := p.Revision
	stateRemoved := body.Meta.Delete
	if stateRemoved == nil {
		stateRemoved = []string{}
	}
	stateUpdated := body.Meta.Update
	if stateUpdated == nil {
		stateUpdated = map[string]interface{}{}
	}
	mu.Unlock()

	c.Status(http.StatusNoContent)
	go func() {
		n := nowStr()
		for _, m := range members {
			sendXMPP(m.AccountID, map[string]interface{}{
				"captain_id": captain, "created_at": createdAt,
				"invite_ttl_seconds": 14400, "max_number_of_members": maxSize,
				"ns": "Fortnite", "party_id": partyID,
				"party_privacy_type": joinability, "party_state_overriden": map[string]interface{}{},
				"party_state_removed": stateRemoved, "party_state_updated": stateUpdated,
				"party_sub_type": subType, "party_type": "DEFAULT",
				"revision": revision, "sent": n,
				"type": "com.epicgames.social.party.notification.v0.PARTY_UPDATED", "updated_at": n,
			})
		}
	}()
}

func PATCHMemberMeta(c *gin.Context) {
	pid := c.Param("partyId")
	accountId := c.Param("accountId")
	userId := c.GetString("accountId")
	if userId != accountId {
		c.JSON(http.StatusForbidden, gin.H{"errorCode": "errors.com.epicgames.party.unauthorized"})
		return
	}
	var body struct {
		Delete   map[string]interface{} `json:"delete"`
		Update   map[string]interface{} `json:"update"`
		Revision int                    `json:"revision"`
	}
	c.ShouldBindJSON(&body)

	mu.Lock()
	p, ok := parties[pid]
	if !ok {
		mu.Unlock()
		c.JSON(http.StatusNotFound, gin.H{"errorCode": "errors.com.epicgames.party.not_found"})
		return
	}
	mIdx := -1
	for i, m := range p.Members {
		if m.AccountID == accountId {
			mIdx = i
			break
		}
	}
	if mIdx == -1 {
		mu.Unlock()
		c.Status(http.StatusNotFound)
		return
	}
	for k := range body.Delete {
		delete(p.Members[mIdx].Meta, k)
	}
	for k, v := range body.Update {
		p.Members[mIdx].Meta[k] = v
	}
	p.Members[mIdx].Revision = body.Revision
	p.Members[mIdx].UpdatedAt = nowStr()
	p.UpdatedAt = nowStr()

	dn, _ := p.Members[mIdx].Meta["urn:epic:member:dn_s"].(string)
	revision := p.Members[mIdx].Revision
	members := make([]Member, len(p.Members))
	copy(members, p.Members)
	partyID := p.ID
	stateUpdated := body.Update
	if stateUpdated == nil {
		stateUpdated = map[string]interface{}{}
	}
	stateRemoved := body.Delete
	if stateRemoved == nil {
		stateRemoved = map[string]interface{}{}
	}
	mu.Unlock()

	c.Status(http.StatusNoContent)
	go func() {
		n := nowStr()
		for _, m := range members {
			sendXMPP(m.AccountID, map[string]interface{}{
				"account_id": accountId, "account_dn": dn,
				"member_state_updated": stateUpdated, "member_state_removed": stateRemoved,
				"member_state_overridden": map[string]interface{}{},
				"party_id":                partyID, "updated_at": n, "sent": n,
				"revision": revision, "ns": "Fortnite",
				"type": "com.epicgames.social.party.notification.v0.MEMBER_STATE_UPDATED",
			})
		}
	}()
}

func DELETEPartyMember(c *gin.Context) {
	pid := c.Param("partyId")
	accountId := c.Param("accountId")

	mu.Lock()
	p, ok := parties[pid]
	if !ok {
		mu.Unlock()
		c.JSON(http.StatusNotFound, gin.H{"errorCode": "errors.com.epicgames.party.not_found"})
		return
	}

	membersBefore := make([]Member, len(p.Members))
	copy(membersBefore, p.Members)

	removeFromParty(p, accountId)

	partyID := p.ID
	revision := p.Revision
	createdAt := p.CreatedAt
	maxSize := p.Config.MaxSize
	joinability := p.Config.Joinability
	subType, _ := p.Meta["urn:epic:cfg:party-type-id_s"].(string)

	if len(p.Members) == 0 {
		delete(parties, pid)
		mu.Unlock()
		c.Status(http.StatusNoContent)
		go func() {
			n := nowStr()
			for _, m := range membersBefore {
				sendXMPP(m.AccountID, map[string]interface{}{
					"account_id": accountId, "member_state_update": map[string]interface{}{},
					"ns": "Fortnite", "party_id": partyID,
					"revision": revision, "sent": n,
					"type": "com.epicgames.social.party.notification.v0.MEMBER_LEFT",
				})
			}
		}()
		return
	}

	hasCaptain := false
	for _, m := range p.Members {
		if m.Role == "CAPTAIN" {
			hasCaptain = true
			break
		}
	}
	if !hasCaptain {
		p.Members[0].Role = "CAPTAIN"
	}

	v := getRSAKey(p)
	var rsaStr string
	if rsaRaw, exists := p.Meta[v]; exists {
		if s, ok := rsaRaw.(string); ok {
			var rsa map[string]interface{}
			if err := json.Unmarshal([]byte(s), &rsa); err == nil {
				assignments, _ := rsa["RawSquadAssignments"].([]interface{})
				filtered := []interface{}{}
				for _, a := range assignments {
					if am, ok := a.(map[string]interface{}); ok {
						if mid, _ := am["memberId"].(string); mid != accountId {
							filtered = append(filtered, a)
						}
					}
				}
				rsa["RawSquadAssignments"] = filtered
				if updated, err := json.Marshal(rsa); err == nil {
					rsaStr = string(updated)
					p.Meta[v] = rsaStr
				}
			}
		}
	}

	captain := ""
	for _, m := range p.Members {
		if m.Role == "CAPTAIN" {
			captain = m.AccountID
			break
		}
	}
	p.UpdatedAt = nowStr()
	membersAfter := make([]Member, len(p.Members))
	copy(membersAfter, p.Members)
	mu.Unlock()

	c.Status(http.StatusNoContent)
	go func() {
		n := nowStr()
		for _, m := range membersBefore {
			sendXMPP(m.AccountID, map[string]interface{}{
				"account_id": accountId, "member_state_update": map[string]interface{}{},
				"ns": "Fortnite", "party_id": partyID,
				"revision": revision, "sent": n,
				"type": "com.epicgames.social.party.notification.v0.MEMBER_LEFT",
			})
		}
		if rsaStr != "" {
			for _, m := range membersAfter {
				sendXMPP(m.AccountID, map[string]interface{}{
					"captain_id": captain, "created_at": createdAt,
					"invite_ttl_seconds": 14400, "max_number_of_members": maxSize,
					"ns": "Fortnite", "party_id": partyID,
					"party_privacy_type": joinability, "party_state_overriden": map[string]interface{}{},
					"party_state_removed": []interface{}{},
					"party_state_updated": map[string]interface{}{v: rsaStr},
					"party_sub_type":      subType, "party_type": "DEFAULT",
					"revision": revision, "sent": n,
					"type": "com.epicgames.social.party.notification.v0.PARTY_UPDATED", "updated_at": n,
				})
			}
		}
	}()
}

func doJoin(c *gin.Context, p *Party, accountId string, connID string, connMeta map[string]interface{}, memberMeta map[string]interface{}, yieldLeadership bool) {
	n := nowStr()
	if connID == "" {
		connID = accountId
	}
	if connMeta == nil {
		connMeta = map[string]interface{}{}
	}
	if memberMeta == nil {
		memberMeta = map[string]interface{}{}
	}

	role := "MEMBER"
	if yieldLeadership {
		role = "CAPTAIN"
	}

	mem := Member{
		AccountID: stripJID(connID),
		Meta:      memberMeta,
		Connections: []Connection{{
			ID: connID, ConnectedAt: n, UpdatedAt: n,
			YieldLeadership: yieldLeadership, Meta: connMeta,
		}},
		Revision: 0, UpdatedAt: n, JoinedAt: n, Role: role,
	}
	p.Members = append(p.Members, mem)

	v := getRSAKey(p)
	var rsaStr string
	if rsaRaw, exists := p.Meta[v]; exists {
		if s, ok := rsaRaw.(string); ok {
			var rsa map[string]interface{}
			if err := json.Unmarshal([]byte(s), &rsa); err == nil {
				assignments, _ := rsa["RawSquadAssignments"].([]interface{})
				assignments = append(assignments, map[string]interface{}{
					"memberId":          stripJID(connID),
					"absoluteMemberIdx": len(p.Members) - 1,
				})
				rsa["RawSquadAssignments"] = assignments
				if updated, err := json.Marshal(rsa); err == nil {
					rsaStr = string(updated)
					p.Meta[v] = rsaStr
					p.Revision++
				}
			}
		}
	}
	p.UpdatedAt = n

	captain := ""
	for _, m := range p.Members {
		if m.Role == "CAPTAIN" {
			captain = m.AccountID
			break
		}
	}
	partyID := p.ID
	createdAt := p.CreatedAt
	maxSize := p.Config.MaxSize
	joinability := p.Config.Joinability
	subType, _ := p.Meta["urn:epic:cfg:party-type-id_s"].(string)
	revision := p.Revision
	members := make([]Member, len(p.Members))
	copy(members, p.Members)
	mu.Unlock()

	c.JSON(http.StatusOK, gin.H{"status": "JOINED", "party_id": partyID})

	go func() {
		n := nowStr()
		dn, _ := connMeta["urn:epic:member:dn_s"].(string)
		joinedPayload := map[string]interface{}{
			"account_dn": dn, "account_id": stripJID(connID),
			"connection": map[string]interface{}{
				"connected_at": n, "id": connID, "meta": connMeta, "updated_at": n,
			},
			"joined_at": n, "member_state_updated": memberMeta,
			"ns": "Fortnite", "party_id": partyID, "revision": 0, "sent": n,
			"type": "com.epicgames.social.party.notification.v0.MEMBER_JOINED", "updated_at": n,
		}
		updatedPayload := map[string]interface{}{
			"captain_id": captain, "created_at": createdAt,
			"invite_ttl_seconds": 14400, "max_number_of_members": maxSize,
			"ns": "Fortnite", "party_id": partyID,
			"party_privacy_type": joinability, "party_state_overriden": map[string]interface{}{},
			"party_state_removed": []interface{}{},
			"party_state_updated": map[string]interface{}{v: rsaStr},
			"party_sub_type":      subType, "party_type": "DEFAULT",
			"revision": revision, "sent": n,
			"type": "com.epicgames.social.party.notification.v0.PARTY_UPDATED", "updated_at": n,
		}
		for _, m := range members {
			sendXMPP(m.AccountID, joinedPayload)
			sendXMPP(m.AccountID, updatedPayload)
		}
	}()
}

func POSTJoinParty(c *gin.Context) {
	pid := c.Param("partyId")
	accountId := c.Param("accountId")
	var body struct {
		Connection struct {
			ID              string                 `json:"id"`
			YieldLeadership bool                   `json:"yield_leadership"`
			Meta            map[string]interface{} `json:"meta"`
		} `json:"connection"`
		Meta map[string]interface{} `json:"meta"`
	}
	c.ShouldBindJSON(&body)

	mu.Lock()
	p, ok := parties[pid]
	if !ok {
		mu.Unlock()
		c.JSON(http.StatusNotFound, gin.H{"errorCode": "errors.com.epicgames.party.not_found"})
		return
	}
	for _, m := range p.Members {
		if m.AccountID == accountId {
			mu.Unlock()
			c.JSON(http.StatusOK, gin.H{"status": "JOINED", "party_id": p.ID})
			return
		}
	}
	if old := partyForUser(accountId); old != nil && old.ID != pid {
		removeFromParty(old, accountId)
		if len(old.Members) == 0 {
			delete(parties, old.ID)
		}
	}
	doJoin(c, p, accountId, body.Connection.ID, body.Connection.Meta, body.Meta, body.Connection.YieldLeadership)
}

func POSTJoinViaPing(c *gin.Context) {
	accountId := c.Param("accountId")
	pingerId := c.Param("pingerId")
	var body struct {
		Connection struct {
			ID              string                 `json:"id"`
			YieldLeadership bool                   `json:"yield_leadership"`
			Meta            map[string]interface{} `json:"meta"`
		} `json:"connection"`
		Meta map[string]interface{} `json:"meta"`
	}
	c.ShouldBindJSON(&body)

	mu.Lock()
	var p *Party
	for _, party := range parties {
		for _, m := range party.Members {
			if m.AccountID == pingerId {
				p = party
				break
			}
		}
		if p != nil {
			break
		}
	}
	if p == nil {
		mu.Unlock()
		c.JSON(http.StatusNotFound, gin.H{"errorCode": "errors.com.epicgames.party.not_found"})
		return
	}
	for _, m := range p.Members {
		if m.AccountID == accountId {
			mu.Unlock()
			c.JSON(http.StatusOK, gin.H{"status": "JOINED", "party_id": p.ID})
			return
		}
	}
	if old := partyForUser(accountId); old != nil && old.ID != p.ID {
		removeFromParty(old, accountId)
		if len(old.Members) == 0 {
			delete(parties, old.ID)
		}
	}
	newPings := []Ping{}
	for _, pg := range pings {
		if !(pg.SentTo == accountId && pg.SentBy == pingerId) {
			newPings = append(newPings, pg)
		}
	}
	pings = newPings
	doJoin(c, p, accountId, body.Connection.ID, body.Connection.Meta, body.Meta, body.Connection.YieldLeadership)
}

func POSTPromoteMember(c *gin.Context) {
	pid := c.Param("partyId")
	accountId := c.Param("accountId")
	userId := c.GetString("accountId")

	mu.Lock()
	p, ok := parties[pid]
	if !ok {
		mu.Unlock()
		c.JSON(http.StatusNotFound, gin.H{"errorCode": "errors.com.epicgames.party.not_found"})
		return
	}
	captainIdx, newCaptainIdx := -1, -1
	for i, m := range p.Members {
		if m.Role == "CAPTAIN" {
			captainIdx = i
		}
		if m.AccountID == accountId {
			newCaptainIdx = i
		}
	}
	if captainIdx != -1 && p.Members[captainIdx].AccountID != userId {
		mu.Unlock()
		c.JSON(http.StatusForbidden, gin.H{"errorCode": "errors.com.epicgames.party.unauthorized"})
		return
	}
	if captainIdx != -1 {
		p.Members[captainIdx].Role = "MEMBER"
	}
	if newCaptainIdx != -1 {
		p.Members[newCaptainIdx].Role = "CAPTAIN"
	}
	p.UpdatedAt = nowStr()
	members := make([]Member, len(p.Members))
	copy(members, p.Members)
	partyID := p.ID
	revision := p.Revision
	mu.Unlock()

	c.Status(http.StatusNoContent)
	go func() {
		n := nowStr()
		for _, m := range members {
			sendXMPP(m.AccountID, map[string]interface{}{
				"account_id": accountId, "member_state_update": map[string]interface{}{},
				"ns": "Fortnite", "party_id": partyID, "revision": revision, "sent": n,
				"type": "com.epicgames.social.party.notification.v0.MEMBER_NEW_CAPTAIN",
			})
		}
	}()
}

func POSTSendPing(c *gin.Context) {
	accountId := c.Param("accountId")
	pingerId := c.Param("pingerId")
	var body struct {
		Meta map[string]interface{} `json:"meta"`
	}
	c.ShouldBindJSON(&body)

	mu.Lock()
	newPings := []Ping{}
	for _, pg := range pings {
		if !(pg.SentTo == accountId && pg.SentBy == pingerId) {
			newPings = append(newPings, pg)
		}
	}
	n, exp := nowStr(), expiresStr()
	ping := Ping{SentBy: pingerId, SentTo: accountId, SentAt: n, ExpiresAt: exp, Meta: body.Meta}
	pings = append(newPings, ping)
	mu.Unlock()

	c.JSON(http.StatusOK, ping)
	go func() {
		sendXMPP(accountId, map[string]interface{}{
			"expires": exp, "meta": body.Meta, "ns": "Fortnite",
			"pinger_dn": getDisplayName(pingerId), "pinger_id": pingerId, "sent": n,
			"type": "com.epicgames.social.party.notification.v0.PING",
		})
	}()
}

func DELETEPing(c *gin.Context) {
	accountId := c.Param("accountId")
	pingerId := c.Param("pingerId")
	mu.Lock()
	newPings := []Ping{}
	for _, pg := range pings {
		if !(pg.SentTo == accountId && pg.SentBy == pingerId) {
			newPings = append(newPings, pg)
		}
	}
	pings = newPings
	mu.Unlock()
	c.Status(http.StatusNoContent)
}

func GETUserPingerParties(c *gin.Context) {
	accountId := c.Param("accountId")
	pingerId := c.Param("pingerId")

	mu.RLock()
	var senderID string
	for _, pg := range pings {
		if pg.SentTo == accountId && pg.SentBy == pingerId {
			senderID = pg.SentBy
			break
		}
	}
	if senderID == "" {
		senderID = pingerId
	}

	result := []interface{}{}
	for _, p := range parties {
		for _, m := range p.Members {
			if m.AccountID == senderID {
				result = append(result, map[string]interface{}{
					"id": p.ID, "created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
					"config": p.Config, "members": p.Members, "applicants": []interface{}{},
					"meta": p.Meta, "invites": []interface{}{}, "revision": p.Revision,
				})
				break
			}
		}
	}
	mu.RUnlock()
	c.JSON(http.StatusOK, result)
}

func POSTPartyInvite(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func POSTPartyInviteToAccount(c *gin.Context) {
	pid := c.Param("partyId")
	accountId := c.Param("accountId")
	userId := c.GetString("accountId")
	sendPing := c.Query("sendPing") == "true"

	var body map[string]interface{}
	c.ShouldBindJSON(&body)
	if body == nil {
		body = map[string]interface{}{}
	}

	mu.Lock()
	p, ok := parties[pid]
	if !ok {
		mu.Unlock()
		c.JSON(http.StatusNotFound, gin.H{"errorCode": "errors.com.epicgames.party.not_found"})
		return
	}
	newInvites := []Invite{}
	for _, inv := range p.Invites {
		if !(inv.SentTo == accountId && inv.SentBy == userId) {
			newInvites = append(newInvites, inv)
		}
	}
	n, exp := nowStr(), expiresStr()
	invite := Invite{
		PartyID: p.ID, SentBy: userId, Meta: body, SentTo: accountId,
		SentAt: n, UpdatedAt: n, ExpiresAt: exp, Status: "SENT",
	}
	p.Invites = append(newInvites, invite)
	p.UpdatedAt = n

	var inviterMeta map[string]interface{}
	for _, m := range p.Members {
		if m.AccountID == userId {
			inviterMeta = m.Meta
			break
		}
	}
	partyID := p.ID
	membersCount := len(p.Members)

	if sendPing {
		newPings := []Ping{}
		for _, pg := range pings {
			if !(pg.SentTo == accountId && pg.SentBy == userId) {
				newPings = append(newPings, pg)
			}
		}
		pings = append(newPings, Ping{SentBy: userId, SentTo: accountId, SentAt: n, ExpiresAt: exp, Meta: body})
	}
	mu.Unlock()

	c.Status(http.StatusNoContent)
	go func() {
		inviterDN := ""
		if inviterMeta != nil {
			inviterDN, _ = inviterMeta["urn:epic:member:dn_s"].(string)
		}
		sendXMPP(accountId, map[string]interface{}{
			"expires": exp, "meta": body, "ns": "Fortnite", "party_id": partyID,
			"inviter_dn": inviterDN, "inviter_id": userId, "invitee_id": accountId,
			"members_count": membersCount, "sent_at": n, "updated_at": n,
			"friends_ids": []interface{}{}, "sent": n,
			"type": "com.epicgames.social.party.notification.v0.INITIAL_INVITE",
		})
		if sendPing {
			sendXMPP(accountId, map[string]interface{}{
				"expires": exp, "meta": body, "ns": "Fortnite",
				"pinger_dn": inviterDN, "pinger_id": userId, "sent": n,
				"type": "com.epicgames.social.party.notification.v0.PING",
			})
		}
	}()
}

func DELETEPartyInvite(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func POSTDeclineInvite(c *gin.Context) {
	pid := c.Param("partyId")
	accountId := c.Param("accountId")
	var body map[string]interface{}
	c.ShouldBindJSON(&body)

	mu.RLock()
	p, ok := parties[pid]
	if !ok {
		mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"errorCode": "errors.com.epicgames.party.not_found"})
		return
	}
	var invite *Invite
	for i := range p.Invites {
		if p.Invites[i].SentTo == accountId {
			invite = &p.Invites[i]
			break
		}
	}
	if invite == nil {
		mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"errorCode": "errors.com.epicgames.party.not_found"})
		return
	}
	sentBy := invite.SentBy
	sentAt := invite.SentAt
	updatedAt := invite.UpdatedAt
	expiresAt := invite.ExpiresAt
	partyID := p.ID
	var inviterMeta map[string]interface{}
	for _, m := range p.Members {
		if m.AccountID == sentBy {
			inviterMeta = m.Meta
			break
		}
	}
	mu.RUnlock()

	c.Status(http.StatusNoContent)
	go func() {
		if inviterMeta == nil {
			return
		}
		inviterDN, _ := inviterMeta["urn:epic:member:dn_s"].(string)
		sendXMPP(sentBy, map[string]interface{}{
			"expires": expiresAt, "meta": body, "ns": "Fortnite", "party_id": partyID,
			"inviter_dn": inviterDN, "inviter_id": sentBy, "invitee_id": accountId,
			"sent_at": sentAt, "updated_at": updatedAt, "sent": nowStr(),
			"type": "com.epicgames.social.party.notification.v0.INVITE_CANCELLED",
		})
	}()
}

func POSTIntention(c *gin.Context) {
	accountId := c.Param("accountId")
	senderId := c.Param("senderId")
	var body map[string]interface{}
	c.ShouldBindJSON(&body)
	if body == nil {
		body = map[string]interface{}{}
	}

	mu.Lock()
	var senderParty *Party
	for _, p := range parties {
		for _, m := range p.Members {
			if m.AccountID == senderId {
				senderParty = p
				break
			}
		}
		if senderParty != nil {
			break
		}
	}
	if senderParty == nil {
		mu.Unlock()
		c.JSON(http.StatusNotFound, gin.H{"errorCode": "errors.com.epicgames.party.not_found"})
		return
	}
	var senderMeta, captainMeta map[string]interface{}
	captainID := ""
	for _, m := range senderParty.Members {
		if m.AccountID == senderId {
			senderMeta = m.Meta
		}
		if m.Role == "CAPTAIN" {
			captainID = m.AccountID
			captainMeta = m.Meta
		}
	}

	n, exp := nowStr(), expiresStr()
	intention := Intention{
		RequesterID: senderId, RequesteeID: accountId,
		Meta: body, ExpiresAt: exp, SentAt: n,
	}
	if senderMeta != nil {
		intention.RequesterDN, _ = senderMeta["urn:epic:member:dn_s"].(string)
	}
	if captainMeta != nil {
		intention.RequesterPL = captainID
		intention.RequesterPLDN, _ = captainMeta["urn:epic:member:dn_s"].(string)
	}
	senderParty.Intentions = append(senderParty.Intentions, intention)
	senderPartyID := senderParty.ID
	membersCount := len(senderParty.Members)
	mu.Unlock()

	c.JSON(http.StatusOK, intention)
	go func() {
		sendXMPP(accountId, map[string]interface{}{
			"expires_at": exp, "requester_id": senderId,
			"requester_dn": intention.RequesterDN, "requester_pl": intention.RequesterPL,
			"requester_pl_dn": intention.RequesterPLDN, "requestee_id": accountId,
			"meta": body, "sent_at": n, "updated_at": n,
			"friends_ids": []interface{}{}, "members_count": membersCount,
			"party_id": senderPartyID, "ns": "Fortnite", "sent": n,
			"type": "com.epicgames.social.party.notification.v0.INITIAL_INTENTION",
		})
	}()
}
