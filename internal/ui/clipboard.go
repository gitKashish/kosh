package ui

import (
	"log/slog"

	"golang.design/x/clipboard"
)

func CopyToClipboard(content []byte) {
	err := clipboard.Init()
	if err != nil {
		slog.Debug("error initializing clipboard", "error", err)
		return
	}

	clipboard.Write(clipboard.FmtText, content)
}
