package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/piholster/piholster/apps/piholsterd/internal/store"
)

const defaultPasswordFile = "/run/piholster/initial-password"

func BootstrapAdminIfNeeded(ctx context.Context, st *store.Store) error {
	count, err := st.UserCount()
	if err != nil {
		return fmt.Errorf("auth: bootstrap: %w", err)
	}
	if count > 0 {
		return nil
	}

	passwordFile := os.Getenv("INITIAL_PASSWORD_FILE")
	if passwordFile == "" {
		passwordFile = defaultPasswordFile
	}

	raw, err := os.ReadFile(passwordFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("auth: bootstrap: no users exist and %s is missing — cannot create admin account", passwordFile)
		}
		return fmt.Errorf("auth: bootstrap: read password file: %w", err)
	}

	password := strings.TrimSpace(string(raw))
	if password == "" {
		return fmt.Errorf("auth: bootstrap: password file %s is empty", passwordFile)
	}

	hash, err := HashPassword(password)
	if err != nil {
		return fmt.Errorf("auth: bootstrap: %w", err)
	}

	if err := st.CreateUser("admin", hash); err != nil {
		return fmt.Errorf("auth: bootstrap: %w", err)
	}

	slog.Info("admin account created from initial-password file")

	if err := os.Remove(passwordFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Warn("auth: bootstrap: could not remove password file", "path", passwordFile, "err", err)
	}

	return nil
}
