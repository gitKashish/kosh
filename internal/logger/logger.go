package logger

import (
	"log/slog"
	"math"
	"os"
	"strconv"
)

// levelOff sits above every other log level,
// so nothing is ever emitted.
const levelOff = slog.Level(math.MaxInt32)

var levelVar = new(slog.LevelVar)

// Setup installs the process-wide diagnostic logger,
// Output is off unless $KOSH_DEBUG is set to a truthy
// value (1, true, t, ...)
func Setup() {
	levelVar.Set(levelFromEnv())

	loggerOptions := &slog.HandlerOptions{
		Level:     levelVar,
		AddSource: true,
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, loggerOptions))
	slog.SetDefault(logger)
}

func Pause() func() {
	prev := levelVar.Level()
	levelVar.Set(levelOff)
	return func() { levelVar.Set(prev) }
}

func levelFromEnv() slog.Level {
	v, ok := os.LookupEnv("KOSH_DEBUG")
	if !ok {
		return levelOff
	}

	on, err := strconv.ParseBool(v)
	if err != nil || !on {
		return levelOff
	}
	return slog.LevelDebug
}
