package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pkwagner/thinx/internal/adapters/thingscloud"
	"github.com/pkwagner/thinx/internal/config"
	"github.com/pkwagner/thinx/internal/onboarding"
	"github.com/pkwagner/thinx/internal/tui"
)

// errOnboardingAborted signals that the user quit onboarding before finishing,
// which is a clean exit rather than a startup failure.
var errOnboardingAborted = errors.New("onboarding aborted")

// main runs the thinx command and reports startup errors.
func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run dispatches CLI arguments or starts the TUI.
func run(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("unknown command %q", args[0])
	}

	dir, err := userDataDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	dbPath := filepath.Join(dir, "things.db")

	email, password, err := resolveAccount(dbPath)
	if errors.Is(err, errOnboardingAborted) {
		return nil
	}
	if err != nil {
		return err
	}

	store, err := thingscloud.NewStore(dbPath, email, password)
	if err != nil {
		return err
	}
	defer store.Close()

	return tui.Run(store)
}

// resolveAccount returns the Things Cloud credentials to use, running first-time
// onboarding when none are configured yet. Returns errOnboardingAborted if the
// user quit onboarding before finishing.
func resolveAccount(dbPath string) (email, password string, err error) {
	cfg, err := config.Load()
	if err != nil {
		return "", "", err
	}

	if !cfg.IsConfigured() {
		if err := onboarding.Run(onboarding.Deps{
			Verify: func(u, p string) error {
				if err := thingscloud.Verify(u, p); errors.Is(err, thingscloud.ErrInvalidCredentials) {
					return onboarding.ErrInvalidCredentials
				} else if err != nil {
					return err
				}
				return nil
			},
			Save: func(u, p string) error {
				return config.Save(config.Config{
					Provider: config.ProviderThingsCloud,
					Username: u,
					Password: p,
				})
			},
			FirstSync: func(u, p string) error {
				return thingscloud.FirstSync(dbPath, u, p)
			},
		}); err != nil {
			return "", "", err
		}

		// Reload to see whether onboarding completed (saved) or was aborted.
		if cfg, err = config.Load(); err != nil {
			return "", "", err
		}
	}

	if !cfg.IsConfigured() {
		return "", "", errOnboardingAborted
	}
	return cfg.Username, cfg.Password, nil
}

func userDataDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		if dir := os.Getenv("LOCALAPPDATA"); dir != "" {
			return filepath.Join(dir, "thinx"), nil
		}
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "thinx"), nil
	default:
		if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
			return filepath.Join(dir, "thinx"), nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "thinx"), nil
}
