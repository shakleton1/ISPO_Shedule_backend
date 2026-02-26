package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const fcmScope = "https://www.googleapis.com/auth/firebase.messaging"

type fcmConfig struct {
	ProjectID       string
	CredentialsFile string
	RequestTimeout  time.Duration
	HTTPClient      *http.Client
}

type fcmNotifier struct {
	projectID string
	ts        oauth2.TokenSource
	hc        *http.Client
	timeout   time.Duration
}

func newFCMNotifier(ctx context.Context, cfg fcmConfig) (Notifier, error) {
	if cfg.ProjectID == "" {
		return nil, fmt.Errorf("push.fcm.project_id required")
	}
	if cfg.CredentialsFile == "" {
		return nil, fmt.Errorf("push.fcm.credentials_file required")
	}
	b, err := os.ReadFile(cfg.CredentialsFile)
	if err != nil {
		return nil, fmt.Errorf("read fcm credentials: %w", err)
	}
	creds, err := google.CredentialsFromJSON(ctx, b, fcmScope)
	if err != nil {
		return nil, fmt.Errorf("parse fcm credentials: %w", err)
	}

	hc := cfg.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}
	tout := cfg.RequestTimeout
	if tout <= 0 {
		tout = 5 * time.Second
	}

	return &fcmNotifier{projectID: cfg.ProjectID, ts: creds.TokenSource, hc: hc, timeout: tout}, nil
}

type fcmSendReq struct {
	Message fcmMessage `json:"message"`
}

type fcmMessage struct {
	Token string            `json:"token"`
	Data  map[string]string `json:"data,omitempty"`
}

func (n *fcmNotifier) Send(ctx context.Context, token string, data map[string]string) error {
	if token == "" {
		return fmt.Errorf("token empty")
	}

	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	oauthTok, err := n.ts.Token()
	if err != nil {
		return fmt.Errorf("fcm oauth token: %w", err)
	}

	body, err := json.Marshal(fcmSendReq{Message: fcmMessage{Token: token, Data: data}})
	if err != nil {
		return fmt.Errorf("marshal fcm request: %w", err)
	}

	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", n.projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+oauthTok.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.hc.Do(req)
	if err != nil {
		return fmt.Errorf("fcm request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		log.Warn().Int("status", resp.StatusCode).RawJSON("body", b).Msg("fcm send failed")
		return fmt.Errorf("fcm send failed: status %d", resp.StatusCode)
	}

	return nil
}
