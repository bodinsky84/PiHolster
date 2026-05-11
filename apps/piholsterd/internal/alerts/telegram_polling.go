package alerts

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/piholster/piholster/apps/piholsterd/internal/store"
	"github.com/piholster/piholster/apps/piholsterd/internal/wealth"
)

type TelegramPoller struct {
	client *TelegramClient
	store  *store.Store
	wealth *wealth.Engine
	offset int
}

func NewTelegramPoller(client *TelegramClient, st *store.Store, w *wealth.Engine) *TelegramPoller {
	return &TelegramPoller{
		client: client,
		store:  st,
		wealth: w,
	}
}

func (p *TelegramPoller) Run(ctx context.Context) {
	if p.client.token == "" || p.client.chatID == "" {
		slog.Info("alerts: Telegram polling disabled (no config)")
		return
	}

	slog.Info("alerts: Telegram polling started")
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

func (p *TelegramPoller) poll(ctx context.Context) {
	url := fmt.Sprintf("%s/bot%s/getUpdates?offset=%d&timeout=10", telegramAPIBase, p.client.token, p.offset)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return
	}

	resp, err := p.client.http.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var result struct {
		Ok     bool `json:"ok"`
		Result []struct {
			UpdateID int `json:"update_id"`
			Message  struct {
				Text string `json:"text"`
				Chat struct {
					ID int64 `json:"id"`
				} `json:"chat"`
			} `json:"message"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return
	}

	for _, update := range result.Result {
		p.offset = update.UpdateID + 1

		chatIDStr := strconv.FormatInt(update.Message.Chat.ID, 10)
		if chatIDStr != p.client.chatID {
			continue
		}

		switch update.Message.Text {
		case "/wealth", "/market":
			p.handleMarket(ctx)
		case "/signals":
			p.handleSignals(ctx)
		}
	}
}

func (p *TelegramPoller) handleMarket(ctx context.Context) {
	m := p.wealth.GetMarket()
	msg := fmt.Sprintf("💹 <b>Wealth Market Report</b>\n\nBTC: $%.2f\nETH: $%.2f\nSOL: $%.2f\n\n<i>Updated: %s</i>",
		m.Bitcoin, m.Ethereum, m.Solana, m.LastUpdate.Format("15:04:05"))
	p.client.Send(ctx, msg)
}

func (p *TelegramPoller) handleSignals(ctx context.Context) {
	signals := p.wealth.GetSignals()
	if len(signals) == 0 {
		p.client.Send(ctx, "⏳ Scanning for wealth opportunities...")
		return
	}

	msg := "🚨 <b>Alpha Signals Detected</b>\n\n"
	for i, s := range signals {
		if i >= 5 { break }
		msg += fmt.Sprintf("<b>[%s]</b> %s (Prob: %.0f%%)\n\n", s.Type, s.Description, s.Probability*100)
	}
	p.client.Send(ctx, msg)
}
