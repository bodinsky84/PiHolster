package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/piholster/piholster/apps/piholsterd/internal/alerts"
	"github.com/piholster/piholster/apps/piholsterd/internal/api"
	"github.com/piholster/piholster/apps/piholsterd/internal/arp"
	"github.com/piholster/piholster/apps/piholsterd/internal/auth"
	internaldns "github.com/piholster/piholster/apps/piholsterd/internal/dns"
	"github.com/piholster/piholster/apps/piholsterd/internal/store"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("PiHolster starting")

	// appCtx lives for the entire daemon lifetime and is cancelled on SIGINT/SIGTERM.
	appCtx, appCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer appCancel()

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./piholster.db"
	}

	db, err := store.Open(dbPath)
	if err != nil {
		slog.Error("failed to open store", "path", dbPath, "err", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("store opened", "path", dbPath)

	if err := auth.BootstrapAdminIfNeeded(appCtx, db); err != nil {
		slog.Error("firstboot bootstrap failed", "err", err)
		os.Exit(1)
	}

	// ARP client — socket path defaults to the piholster-arpd IPC socket.
	arpSock := os.Getenv("ARP_SOCK")
	if arpSock == "" {
		arpSock = "/run/piholster/arpd.sock"
	}
	arpClient := arp.NewClient(arpSock)
	if err := arpClient.Connect(appCtx); err != nil {
		// Non-fatal: device discovery and alerts are degraded but DNS still works.
		slog.Warn("arp client: failed to connect, device alerts disabled", "err", err)
	}
	defer arpClient.Close()

	tgToken := os.Getenv("TELEGRAM_TOKEN")
	tgChatID := os.Getenv("TELEGRAM_CHAT_ID")
	tgClient := alerts.NewTelegramClient(tgToken, tgChatID)
	notifier := alerts.NewNotifier(tgClient, db)
	go notifier.Run(appCtx, arpClient.Devices())

	bl := internaldns.NewBlocklist()

	blocklistPath := os.Getenv("BLOCKLIST_PATH")
	if blocklistPath == "" {
		blocklistPath = "packages/blocklists/ads.txt"
	}
	if err := bl.LoadFromFile(blocklistPath); err != nil {
		// Non-fatal: the server operates with an empty blocklist when the file
		// is absent (e.g. first boot before lists are populated).
		slog.Warn("could not load blocklist", "path", blocklistPath, "err", err)
	}

	upstream := internaldns.NewDoHUpstream()
	srv := internaldns.NewServer(bl, upstream)

	if err := srv.Start(); err != nil {
		slog.Error("DNS server failed to start", "err", err)
		os.Exit(1)
	}

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	router := api.NewRouter(appCtx, db)
	httpServer := &http.Server{
		Addr:         ":" + httpPort,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("HTTP API listening", "port", httpPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "err", err)
		}
	}()

	<-appCtx.Done()
	slog.Info("shutting down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", "err", err)
	}

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("DNS server shutdown error", "err", err)
		os.Exit(1)
	}
	slog.Info("PiHolster stopped cleanly")
}
