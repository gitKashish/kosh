package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"git.plutolab.org/plutolab/kosh/internal/logger"
)

const (
	ColorReset  = "\033[0m"
	ColorBold   = "\033[1m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[90m"
)

// Output writers. Swap these (via Pause) to silence or redirect logging,
// e.g. while a raw-mode TUI owns the terminal.
var (
	profile string    = "default"
	out     io.Writer = os.Stdout
	errOut  io.Writer = os.Stderr
)

// Pause silences all logger output and returns a function that restores the
// previous writers. Use it around interactive/raw-mode sessions that own the
// terminal cursor, so stray log lines don't corrupt the display:
//
//	defer ui.Pause()()
func PauseOutput() func() {
	prevOut, prevErr := out, errOut
	out, errOut = io.Discard, io.Discard
	enable := logger.Pause()
	return func() {
		out, errOut = prevOut, prevErr
		enable()
	}
}

func SetProfile(p string) {
	profile = p
}

// WithProfile temporarily overrides the profile shown in the output prefix and
// returns a function restoring the previous value. Use it in commands that act
// on a profile other than the active one:
//
//	defer ui.WithProfile(target)()
func WithProfile(p string) func() {
	prev := profile
	profile = p
	return func() { profile = prev }
}

// Error prints error messages
func Error(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(
		errOut, "%s %s %s\n",
		prf(), glyph("✗", ColorRed), message,
	)
}

// Warn prints warning messages
func Warn(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(
		errOut, "%s %s %s\n",
		prf(), glyph("!", ColorYellow), message,
	)
}

// Prompt prints a prompt for user input
func Prompt(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(
		out, "%s %s %s",
		prf(), glyph("?", ColorCyan), message,
	)
}

// Muted prints muted text messages
func Muted(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	muted := ColorGray + message + ColorReset
	fmt.Fprintf(
		errOut, "%s %s\n",
		glyph("•", ColorGray), muted,
	)
}

// Info prints informational messages
func Info(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(
		out, "%s %s %s\n",
		prf(), glyph("✓", ColorGreen), message,
	)

}

// Caution displays a highly visible warning block intended strictly for destructive actions.
func Caution(header, format string, args ...any) {
	message := fmt.Sprintf(format, args...)

	// Split the message into lines so we can apply the border to each line
	lines := strings.Split(message, "\n")

	fmt.Fprintln(errOut) // Add a little breathing room above

	// Bold Red Header
	// fmt.Fprintf(errOut, "  %s%s▌ ⚠  CAUTION: DESTRUCTIVE ACTION%s\n", ColorRed, ColorBold, ColorReset)
	fmt.Fprintf(errOut, "  %s%s| /!\\ CAUTION: %s%s\n", ColorRed, ColorBold, header, ColorReset)

	// Red Border with normal text for the body
	for _, line := range lines {
		fmt.Fprintf(errOut, "  %s%s|%s %s\n", ColorRed, ColorBold, ColorReset, line)
	}

	fmt.Fprintln(errOut) // Add breathing room below
}

func prf() string {
	return fmt.Sprintf(
		"%s(%s)%s",
		ColorGray,
		profile,
		ColorReset,
	)
}

func glyph(symbol, color string) string {
	return fmt.Sprintf(
		"%s[%s]%s",
		color,
		symbol,
		ColorReset,
	)
}
