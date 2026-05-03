package alerts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

const telegramAPIBase = "https://api.telegram.org"

type TelegramClient struct {
	token  string
	chatID string
	http   *http.Client
}

// NewTelegramClient returns a client configured with a 10s timeout.
// If token or chatID is empty it logs a warning once and all subsequent
// Send calls become no-ops so missing config never crashes the daemon.
func NewTelegramClient(token, chatID string) *TelegramClient {
	if token == "" || chatID == "" {
		slog.Warn("alerts: TELEGRAM_TOKEN or TELEGRAM_CHAT_ID not set — Telegram notifications disabled")
	}
	return &TelegramClient{
		token:  token,
		chatID: chatID,
		http:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (t *TelegramClient) Send(ctx context.Context, message string) error {
	if t.token == "" || t.chatID == "" {
		return nil
	}

	payload := struct {
		ChatID    string `json:"chat_id"`
		Text      string `json:"text"`
		ParseMode string `json:"parse_mode"`
	}{
		ChatID:    t.chatID,
		Text:      message,
		ParseMode: "HTML",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("alerts: failed to marshal Telegram payload", "err", err)
		return nil
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", telegramAPIBase, t.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		slog.Error("alerts: failed to build Telegram request", "err", err)
		return nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.http.Do(req)
	if err != nil {
		slog.Error("alerts: Telegram send failed", "err", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.Error("alerts: Telegram API returned non-2xx", "status", resp.StatusCode)
	}
	return nil
}
