package alerts

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/piholster/piholster/apps/piholsterd/internal/arp"
)

// DeviceStore is the subset of store.Store used by the Notifier.
// Kept as an interface so tests can substitute a mock without a real DB.
type DeviceStore interface {
	UpsertDevice(mac, ip, hostname string) error
	IsDeviceTrusted(mac string) (bool, error)
	DeviceFirstSeen(mac string) (time.Time, error)
}

type Notifier struct {
	telegram *TelegramClient
	store    DeviceStore
}

func NewNotifier(telegram *TelegramClient, store DeviceStore) *Notifier {
	return &Notifier{telegram: telegram, store: store}
}

// Run consumes the devices channel until ctx is cancelled.
// For each device it upserts into the store and sends a Telegram alert when
// the device is newly discovered (first_seen < 5 min ago) and not trusted.
func (n *Notifier) Run(ctx context.Context, devices <-chan arp.Device) {
	for {
		select {
		case <-ctx.Done():
			return
		case dev, ok := <-devices:
			if !ok {
				return
			}
			n.handle(ctx, dev)
		}
	}
}

func (n *Notifier) handle(ctx context.Context, dev arp.Device) {
	if err := n.store.UpsertDevice(dev.MAC, dev.IP, dev.Hostname); err != nil {
		slog.Error("alerts: upsert device failed", "mac", dev.MAC, "err", err)
		return
	}

	trusted, err := n.store.IsDeviceTrusted(dev.MAC)
	if err != nil {
		slog.Error("alerts: could not check trusted status", "mac", dev.MAC, "err", err)
		return
	}
	if trusted {
		return
	}

	firstSeen, err := n.store.DeviceFirstSeen(dev.MAC)
	if err != nil {
		slog.Error("alerts: could not read first_seen", "mac", dev.MAC, "err", err)
		return
	}

	// Only alert once, in the window right after first discovery.
	if time.Since(firstSeen) >= 5*time.Minute {
		return
	}

	msg := buildMessage(dev)
	n.telegram.Send(ctx, msg) //nolint:errcheck — Send never returns a real error
}

func buildMessage(dev arp.Device) string {
	name := dev.Hostname
	if name == "" {
		name = "Okänd enhet"
	}
	return fmt.Sprintf(
		"🔔 <b>Ny enhet på nätverket</b>\n\n"+
			"📱 <b>Namn:</b> %s\n"+
			"🔌 <b>IP:</b> %s\n"+
			"🔑 <b>MAC:</b> %s\n\n"+
			"Gå till <a href=\"http://piholster.local\">piholster.local</a> för att godkänna eller blockera enheten.",
		name, dev.IP, dev.MAC,
	)
}
