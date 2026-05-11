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
	"github.com/piholster/piholster/apps/piholsterd/internal/wealth"
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
		arpSock = "/run/piholster/arp.sock"
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

	wealthEngine := wealth.NewEngine()
	go wealthEngine.Run(appCtx)

	poller := alerts.NewTelegramPoller(tgClient, db, wealthEngine)
	go poller.Run(appCtx)

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
	srv := internaldns.NewServer(bl, upstream, db)

	if err := srv.Start(); err != nil {
		slog.Error("DNS server failed to start", "err", err)
		os.Exit(1)
	}

	router := api.NewRouter(appCtx, db, wealthEngine)

	// TLS configuration — read cert/key paths from environment.
	// On Pi: set by piholsterd.service (TLS_CERT, TLS_KEY, HTTPS_PORT, HTTP_PORT).
	// In dev/Docker: leave unset → plain HTTP on DEV_HTTP_PORT.
	tlsCert := os.Getenv("TLS_CERT")
	tlsKey := os.Getenv("TLS_KEY")

	var httpsServer, redirectServer *http.Server
	var shutdownErr error

	if tlsCert != "" && tlsKey != "" {
		// Production path: HTTPS on HTTPS_PORT, HTTP redirect on HTTP_PORT.
		httpsPort := os.Getenv("HTTPS_PORT")
		if httpsPort == "" {
			httpsPort = "443"
		}
		redirectPort := os.Getenv("HTTP_PORT")
		if redirectPort == "" {
			redirectPort = "80"
		}

		httpsServer = &http.Server{
			Addr:         ":" + httpsPort,
			Handler:      router,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}
		go func() {
			slog.Info("HTTPS server listening", "port", httpsPort)
			if err := httpsServer.ListenAndServeTLS(tlsCert, tlsKey); err != nil && err != http.ErrServerClosed {
				slog.Error("HTTPS server error", "err", err)
			}
		}()

		// HTTP → HTTPS redirect (US-24).
		redirectServer = &http.Server{
			Addr: ":" + redirectPort,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				target := "https://" + r.Host + r.URL.RequestURI()
				http.Redirect(w, r, target, http.StatusMovedPermanently)
			}),
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		}
		go func() {
			slog.Info("HTTP redirect server listening", "port", redirectPort)
			if err := redirectServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Warn("HTTP redirect server error", "err", err)
			}
		}()
	} else {
		// Development path: plain HTTP. TLS cert will be generated on first Pi boot.
		devPort := os.Getenv("DEV_HTTP_PORT")
		if devPort == "" {
			devPort = "8080"
		}
		httpsServer = &http.Server{
			Addr:         ":" + devPort,
			Handler:      router,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}
		go func() {
			slog.Info("HTTP server listening (dev mode, no TLS)", "port", devPort)
			if err := httpsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("HTTP server error", "err", err)
			}
		}()
	}

	<-appCtx.Done()
	slog.Info("shutting down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if redirectServer != nil {
		if err := redirectServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("HTTP redirect server shutdown error", "err", err)
			shutdownErr = err
		}
	}

	if err := httpsServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP(S) server shutdown error", "err", err)
		shutdownErr = err
	}

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("DNS server shutdown error", "err", err)
		shutdownErr = err
	}

	if shutdownErr != nil {
		os.Exit(1)
	}
	slog.Info("PiHolster stopped cleanly")
}
