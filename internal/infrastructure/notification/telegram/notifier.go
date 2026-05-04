// Package telegram implements application.Notifier using the Telegram Bot API.
// Docs: https://core.telegram.org/bots/api#sendmessage
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ducminhgd/weather-bot/internal/application"
)

// NotifierType is the key used in config.notifications[*].type to select this notifier.
const NotifierType = "telegram"

const apiBase = "https://api.telegram.org"

var _ application.Notifier = (*Notifier)(nil)

// Notifier sends messages to a Telegram chat via a bot.
type Notifier struct {
	botToken string
	chatID   string
	client   *http.Client
}

// New creates a Notifier from the params map.
// Required keys: "bot_token", "chat_id".
func New(params map[string]string) (*Notifier, error) {
	token := params["bot_token"]
	chatID := params["chat_id"]
	if token == "" {
		return nil, fmt.Errorf("telegram: bot_token is required")
	}
	if chatID == "" {
		return nil, fmt.Errorf("telegram: chat_id is required")
	}
	return &Notifier{
		botToken: token,
		chatID:   chatID,
		client:   &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// Send delivers message to the configured Telegram chat.
// The message is converted from CommonMark (**bold**) to Telegram Markdown (*bold*).
func (n *Notifier) Send(ctx context.Context, message string) error {
	payload, err := json.Marshal(map[string]string{
		"chat_id":    n.chatID,
		"text":       toTelegramMarkdown(message),
		"parse_mode": "Markdown",
	})
	if err != nil {
		return fmt.Errorf("telegram: marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", apiBase, n.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("telegram: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Description string `json:"description"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return fmt.Errorf("telegram: HTTP %d: %s", resp.StatusCode, apiErr.Description)
	}

	return nil
}

// toTelegramMarkdown converts CommonMark bold markers (**) to Telegram Markdown (*).
func toTelegramMarkdown(msg string) string {
	return strings.ReplaceAll(msg, "**", "*")
}
