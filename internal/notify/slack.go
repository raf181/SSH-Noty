package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"ssh-noty/internal/config"
	"ssh-noty/internal/logging"
	"ssh-noty/internal/model"
)

type Slack struct {
	webhook string
	client  *http.Client
}

func NewSlack(cfg *config.Config) *Slack {
	return &Slack{
		webhook: cfg.SlackWebhook,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

type SlackMessage struct {
	Text   string        `json:"text,omitempty"`
	Blocks []interface{} `json:"blocks,omitempty"`
}

func (s *Slack) Send(ctx context.Context, msg *SlackMessage) error {
	if s.webhook == "" {
		return nil
	}
	b, _ := json.Marshal(msg)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.webhook, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("slack webhook status %d", resp.StatusCode)
	}
	return nil
}

// Minimal event to Slack text for now; can be upgraded to rich blocks later.
func (s *Slack) SendEvent(ctx context.Context, ev *model.Event) error {
	// mrkdwn formatted message using Slack blocks
	header := "🔐 SSH LOGIN SUCCESS"
	if ev.Type != "login_success" {
		header = "🚨 SSH FAILED/INVALID LOGIN"
	}

	fields := []map[string]any{
		{"type": "mrkdwn", "text": fmt.Sprintf("*User*: `%s`", safe(ev.Username))},
		{"type": "mrkdwn", "text": fmt.Sprintf("*Source*: `%s`:`%d`", safe(ev.SourceIP), ev.Port)},
		{"type": "mrkdwn", "text": fmt.Sprintf("*Method*: `%s`", safe(ev.Method))},
		{"type": "mrkdwn", "text": fmt.Sprintf("*Host*: `%s`", safe(ev.Hostname))},
		{"type": "mrkdwn", "text": fmt.Sprintf("*Time*: `%s`", ev.Timestamp.Format(time.RFC3339))},
	}
	blocks := []interface{}{
		map[string]any{"type": "header", "text": map[string]any{"type": "plain_text", "text": header}},
		map[string]any{"type": "section", "fields": fields},
	}
	return s.Send(ctx, &SlackMessage{Blocks: blocks})
}

func safe(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func TestMessage() *SlackMessage {
	return &SlackMessage{Text: "ssh-noti installed and configured ✅"}
}

var log = logging.L
