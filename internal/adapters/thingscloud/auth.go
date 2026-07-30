package thingscloud

import (
	"errors"

	things "github.com/pkwagner/things-cloud-sdk"
)

// ErrInvalidCredentials is returned by Verify when the username/password pair is
// rejected by Things Cloud.
var ErrInvalidCredentials = errors.New("invalid username or password")

// Verify checks Things Cloud credentials without opening or persisting anything.
func Verify(email, password string) error {
	client := things.New(things.APIEndpoint, email, password)
	if _, err := client.Verify(); err != nil {
		if errors.Is(err, things.ErrUnauthorized) {
			return ErrInvalidCredentials
		}
		return err
	}
	return nil
}

// FirstSync opens the local store, performs the initial (potentially slow) full
// sync, and closes it again. The on-disk database is left populated for the main
// app to reopen.
func FirstSync(dbPath, email, password string) error {
	store, err := NewStore(dbPath, email, password)
	if err != nil {
		return err
	}
	defer store.Close()
	_, err = store.syncer.Sync()
	return err
}
