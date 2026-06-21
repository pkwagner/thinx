package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"thinx/internal/adapters/thingscloud"
	"thinx/internal/tui"
)

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

	email := os.Getenv("THINGS_USERNAME")
	password := os.Getenv("THINGS_PASSWORD")
	if email == "" || password == "" {
		return fmt.Errorf("THINGS_USERNAME and THINGS_PASSWORD are required")
	}

	dir, err := userDataDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	store, err := thingscloud.NewStore(filepath.Join(dir, "things.db"), email, password)
	if err != nil {
		return err
	}
	defer store.Close()

	return tui.Run(store)
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
