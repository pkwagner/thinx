// Package config persists thinx settings (provider, credentials, and — later —
// user customizations) as JSON under the OS config directory.
package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// ProviderThingsCloud is the only backend provider supported today.
const ProviderThingsCloud = "thingscloud"

// Config is the on-disk configuration. It is intentionally a plain struct so new
// fields (customizations, preferences) can be added without migration logic.
type Config struct {
	Provider string `json:"provider"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// IsConfigured reports whether an account has been set up.
func (c Config) IsConfigured() bool {
	return c.Provider != "" && c.Username != "" && c.Password != ""
}

// Path returns the absolute path of the config file (~/.config/thinx/config.json
// on Linux, the OS equivalent elsewhere). It is a var so tests can redirect
// storage to a temp directory.
var Path = func() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "thinx", "config.json"), nil
}

// Load reads the config, returning a zero Config (not an error) when none exists.
func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, err
	}
	return c, nil
}

// Save writes the config, creating the directory with private permissions. The
// file is written 0600 because it holds credentials.
func Save(c Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
