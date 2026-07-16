// Package output renders events to the terminal: ANSI colors with automatic
// downgrade, and a single-writer printer that keeps concurrent updates from
// interleaving.
package output

import (
	"os"

	"golang.org/x/term"
)

type Style string

const (
	Reset  Style = "\x1b[0m"
	Bold   Style = "\x1b[1m"
	Dim    Style = "\x1b[2m"
	Red    Style = "\x1b[31m"
	Green  Style = "\x1b[32m"
	Yellow Style = "\x1b[33m"
	Cyan   Style = "\x1b[36m"
)

// ColorsEnabled reports whether stdout should receive ANSI codes:
// disabled when piped, when NO_COLOR is set, or on TERM=dumb.
func ColorsEnabled() bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return false
	}
	return enableVT() // no-op except legacy Windows consoles
}

// Colorize wraps s in the style when enabled is true.
func Colorize(enabled bool, style Style, s string) string {
	if !enabled || style == "" {
		return s
	}
	return string(style) + s + string(Reset)
}

// MethodStyle color-codes HTTP methods.
func MethodStyle(method string) Style {
	switch method {
	case "GET":
		return Cyan
	case "POST":
		return Green
	case "PUT", "PATCH":
		return Yellow
	case "DELETE":
		return Red
	default:
		return ""
	}
}
