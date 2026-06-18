package logger

import (
	"fmt"
	"io"
	"os"
	"runtime"
)

const (
	BUILD_MODE_PRODUCTION = "production"
	BUILD_MODE_DEBUG      = "debug"
)

var BuildMode = BUILD_MODE_PRODUCTION

const (
	ColorReset  = "\033[0m"
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
	out io.Writer = os.Stdout
	errOut io.Writer = os.Stderr
)

// Pause silences all logger output and returns a function that restores the
// previous writers. Use it around interactive/raw-mode sessions that own the
// terminal cursor, so stray log lines don't corrupt the display:
//
//	defer logger.Pause()()
func Pause() func() {
	prevOut, prevErr := out, errOut
	out, errOut = io.Discard, io.Discard
	return func() {
		out, errOut = prevOut, prevErr
	}
}

// Error prints error messages
func Error(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(errOut, "%s[✗]%s %s\n", ColorRed, ColorReset, message)
}

// Info prints informational messages
func Info(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(out, "%s[✓]%s %s\n", ColorGreen, ColorReset, message)
}

// Warn prints warning messages
func Warn(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(out, "%s[!]%s %s\n", ColorYellow, ColorReset, message)
}

// Debug prints debug messages
func Debug(format string, args ...any) {
	if BuildMode == BUILD_MODE_PRODUCTION {
		return
	}
	_, file, line, ok := runtime.Caller(1)
	caller := ""
	if ok {
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				file = file[i+1:]
				break
			}
		}
		caller = fmt.Sprintf("%s:%d", file, line)
	}
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(out, "%s[→]%s %s %s[%s]%s\n",
		ColorBlue, ColorReset,
		message,
		ColorGray, caller, ColorReset)
}

// Prompt prints a prompt for user input
func Prompt(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(out, "%s[?]%s %s", ColorCyan, ColorReset, message)
}

// Muted prints muted text messages
func Muted(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(out, "%s[•] %s%s\n", ColorGray, message, ColorReset)
}