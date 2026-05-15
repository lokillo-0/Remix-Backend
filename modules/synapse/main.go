package synapse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/remixfn/xenon/utilities"
	"gosrc.io/xmpp"
	"gosrc.io/xmpp/stanza"
)

type SynapseManager struct {
	client     *xmpp.Client
	apiClient  *http.Client
	isStarted  bool
	mu         sync.Mutex
	ctx        context.Context
	cancelFunc context.CancelFunc
}

var (
	startedInstance *SynapseManager
	instanceLock    sync.Mutex
)

func GetStartedInstance() *SynapseManager {
	instanceLock.Lock()
	defer instanceLock.Unlock()
	return startedInstance
}

func NewSynapseManager() *SynapseManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &SynapseManager{
		apiClient:  &http.Client{},
		isStarted:  false,
		ctx:        ctx,
		cancelFunc: cancel,
	}
}

func (sm *SynapseManager) Start() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.isStarted {
		return nil
	}

	service := "prod.ol.epicgames.com"
	username := "xmpp-admin"
	password := utilities.Get[string]("s_password")

	config := xmpp.Config{
		TransportConfiguration: xmpp.TransportConfiguration{
			Address: "127.0.0.1:5223",
		},
		Jid:        fmt.Sprintf("%s@%s", username, service),
		Credential: xmpp.Password(password),
		Insecure:   true,
	}

	router := xmpp.NewRouter()
	router.HandleFunc("presence", func(s xmpp.Sender, p stanza.Packet) {
		// don't need to do anything for this!
	})

	client, err := xmpp.NewClient(&config, router, errorHandler)
	if err != nil {
		return fmt.Errorf("failed to create XMPP client: %w", err)
	}
	sm.client = client

	if err := sm.client.Connect(); err != nil {
		return fmt.Errorf("failed to connect to XMPP server: %w", err)
	}

	presence := stanza.NewPresence(stanza.Attrs{
		From: fmt.Sprintf("%s@%s", username, service),
	})
	if err := sm.client.Send(presence); err != nil {
		return fmt.Errorf("failed to send initial presence: %w", err)
	}

	sm.isStarted = true
	startedInstance = sm
	return nil
}

func errorHandler(err error) {
	errStr := err.Error()
	if !strings.Contains(errStr, "Unsolicited response received") &&
		!strings.Contains(errStr, "HTTP/1.0 500") {
		fmt.Printf("XMPP error: %v\n", err)
	}
}

func (sm *SynapseManager) SendMessage(user string, content interface{}) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.isStarted {
		sm.mu.Unlock()
		if err := sm.Start(); err != nil {
			sm.mu.Lock()
			return fmt.Errorf("failed to start client: %w", err)
		}
		sm.mu.Lock()
	}

	var body string
	switch v := content.(type) {
	case string:
		body = v
	default:
		data, err := json.Marshal(content)
		if err != nil {
			return fmt.Errorf("failed to marshal content: %w", err)
		}
		body = string(data)
	}

	service := "prod.ol.epicgames.com"
	username := "xmpp-admin"

	messageID := uuid.New().String()
	msg := stanza.Message{
		Attrs: stanza.Attrs{
			From: fmt.Sprintf("%s@%s", username, service),
			To:   fmt.Sprintf("%s@%s", user, service),
			Id:   messageID,
			Type: "normal",
		},
		Body: body,
	}

	err := sm.client.Send(msg)
	if err != nil {
		if err.Error() == "client is not connected" ||
			strings.Contains(strings.ToLower(err.Error()), "connect") ||
			strings.Contains(strings.ToLower(err.Error()), "connection") {

			sm.isStarted = false

			sm.mu.Unlock()
			if startErr := sm.Start(); startErr != nil {
				sm.mu.Lock()
				return fmt.Errorf("failed to reconnect: %w", startErr)
			}
			sm.mu.Lock()

			if err := sm.client.Send(msg); err != nil {
				return fmt.Errorf("failed to send message after reconnect: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to send message: %w", err)
	}
	return nil
}

func (sm *SynapseManager) ForwardPresenceStanza(user1, user2 string) error {
	baseURL := utilities.Get[string]("s_baseurl")
	apiToken := utilities.Get[string]("s_apitoken")

	url := fmt.Sprintf("%s/forward_presence_stanza/%s/%s", baseURL, user1, user2)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte("")))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-synapse", apiToken)
	resp, err := sm.apiClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to forward presence stanza: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to forward presence stanza, status: %d", resp.StatusCode)
	}
	return nil
}

func (sm *SynapseManager) ForwardOfflinePresenceStanza(user1, user2 string) error {
	baseURL := utilities.Get[string]("s_baseurl")
	apiToken := utilities.Get[string]("s_apitoken")

	url := fmt.Sprintf("%s/forward_offline_presence_stanza/%s/%s", baseURL, user1, user2)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte("")))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-synapse", apiToken)
	resp, err := sm.apiClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to forward offline presence stanza: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to forward offline presence stanza, status: %d", resp.StatusCode)
	}
	return nil
}

func (sm *SynapseManager) ForwardPresenceBothWays(user1, user2 string) error {
	err := sm.ForwardPresenceStanza(user1, user2)
	if err != nil {
		return err
	}
	return sm.ForwardPresenceStanza(user2, user1)
}

func (sm *SynapseManager) ForwardOfflinePresenceBothWays(user1, user2 string) error {
	err := sm.ForwardOfflinePresenceStanza(user1, user2)
	if err != nil {
		return err
	}
	return sm.ForwardOfflinePresenceStanza(user2, user1)
}

func (sm *SynapseManager) Close() error {
	instanceLock.Lock()
	defer instanceLock.Unlock()

	if sm.isStarted && sm.client != nil {
		sm.cancelFunc()
		sm.client.Disconnect()
		sm.isStarted = false
	}

	if startedInstance == sm {
		startedInstance = nil
	}
	return nil
}
