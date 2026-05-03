package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/piholster/piholster/apps/piholster-arpd/internal/arp"
	"github.com/piholster/piholster/apps/piholster-arpd/internal/ipc"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("piholster-arpd starting")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	iface := os.Getenv("ARP_INTERFACE")
	sockPath := os.Getenv("ARP_SOCK_PATH")
	if sockPath == "" {
		sockPath = "/run/piholster/arp.sock"
	}

	scanner, err := arp.NewScanner(iface)
	if err != nil {
		slog.Error("failed to initialise ARP scanner", "err", err)
		os.Exit(1)
	}

	srv, err := ipc.NewServer(sockPath, scanner.Devices())
	if err != nil {
		slog.Error("failed to create IPC server", "err", err)
		os.Exit(1)
	}

	go func() {
		if err := scanner.Run(ctx); err != nil && ctx.Err() == nil {
			slog.Error("ARP scanner exited", "err", err)
		}
	}()

	go func() {
		if err := srv.Serve(ctx); err != nil && ctx.Err() == nil {
			slog.Error("IPC server exited", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("piholster-arpd stopping")
}
