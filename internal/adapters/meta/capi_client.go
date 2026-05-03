package meta

import (
	"bytes"
	"context"
	"crypto/sha256"
	"dental-app/internal/core/ports"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type capiClient struct {
	pixelID         string
	accessToken     string
	testEventCode   string
	httpClient      *http.Client
}

func NewCAPIClient() ports.MetaCAPIClient {
	return &capiClient{
		pixelID:       os.Getenv("META_PIXEL_ID"),
		accessToken:   os.Getenv("META_ACCESS_TOKEN"),
		testEventCode: os.Getenv("META_TEST_EVENT_CODE"),
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}
}

func sha256hex(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}

func (c *capiClient) SendLeadEvent(ctx context.Context, ev ports.MetaLeadEvent) error {
	if c.pixelID == "" || c.accessToken == "" {
		log.Printf("[Meta CAPI] modo simulación — fbclid=%s", ev.Fbclid)
		return nil
	}

	userData := map[string]interface{}{}
	if ev.Phone != "" {
		userData["ph"] = []string{sha256hex(ev.Phone)}
	}
	if ev.Email != "" {
		userData["em"] = []string{sha256hex(ev.Email)}
	}
	if ev.ExternalID != "" {
		userData["external_id"] = []string{sha256hex(ev.ExternalID)}
	}
	if ev.Fbclid != "" {
		userData["fbc"] = ev.Fbclid
	}

	eventData := map[string]interface{}{
		"event_name":       "Lead",
		"event_time":       ev.EventTime,
		"action_source":    "website",
		"event_source_url": ev.EventSourceURL,
		"user_data":        userData,
	}

	payload := map[string]interface{}{
		"data": []interface{}{eventData},
	}
	if c.testEventCode != "" {
		payload["test_event_code"] = c.testEventCode
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("meta capi: marshal: %w", err)
	}

	url := fmt.Sprintf("https://graph.facebook.com/v19.0/%s/events?access_token=%s", c.pixelID, c.accessToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("meta capi: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("meta capi: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("meta capi: status %d", resp.StatusCode)
	}

	log.Printf("[Meta CAPI] Lead enviado — pixel=%s fbclid=%s status=%d", c.pixelID, ev.Fbclid, resp.StatusCode)
	return nil
}
