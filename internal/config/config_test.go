package config

import (
	"os"
	"path/filepath"
	"testing"
)

// withTempDir redirects config storage to a throwaway directory.
func withTempDir(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	old := Path
	Path = func() (string, error) { return filepath.Join(tmp, "thinx", "config.json"), nil }
	t.Cleanup(func() { Path = old })
}

// TestLoadMissingReturnsZero verifies a missing file is not an error.
func TestLoadMissingReturnsZero(t *testing.T) {
	withTempDir(t)
	got, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.IsConfigured() {
		t.Fatalf("expected unconfigured, got %#v", got)
	}
}

// TestSaveLoadRoundTrip verifies persistence and that the file is private.
func TestSaveLoadRoundTrip(t *testing.T) {
	withTempDir(t)
	want := Config{Provider: ProviderThingsCloud, Username: "u@example.com", Password: "secret"}
	if err := Save(want); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got != want || !got.IsConfigured() {
		t.Fatalf("round trip = %#v, want %#v", got, want)
	}

	path, _ := Path()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("file mode = %v, want 0600", perm)
	}
}

// TestIsConfigured covers the partial-credential cases.
func TestIsConfigured(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		want bool
	}{
		{"empty", Config{}, false},
		{"no password", Config{Provider: ProviderThingsCloud, Username: "u"}, false},
		{"no provider", Config{Username: "u", Password: "p"}, false},
		{"complete", Config{Provider: ProviderThingsCloud, Username: "u", Password: "p"}, true},
	}
	for _, c := range cases {
		if got := c.cfg.IsConfigured(); got != c.want {
			t.Errorf("%s: IsConfigured = %v, want %v", c.name, got, c.want)
		}
	}
}
