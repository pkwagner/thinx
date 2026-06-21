// Package uihelp holds small formatting helpers shared across the TUI packages.
package uihelp

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
)

// FormatDate renders a date in the canonical yyyy-mm-dd form used throughout the UI.
func FormatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// BindingHelp renders a single binding as "key: description".
func BindingHelp(binding key.Binding) string {
	help := binding.Help()
	return help.Key + ": " + help.Desc
}

// MultiBindingHelp renders several bindings sharing one description as "k1/k2/k3: description".
func MultiBindingHelp(help string, bindings ...key.Binding) string {
	keys := make([]string, len(bindings))
	for i, b := range bindings {
		keys[i] = b.Help().Key
	}
	return strings.Join(keys, "/") + ": " + help
}
