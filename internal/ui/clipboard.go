package ui

import (
	"context"
	"log/slog"
	"os"

	"plutolab.org/klip"
)

func CopyToClipboard(content []byte, autoClear bool) error {
	clipboard := klip.NewClipboard()
	opts := klip.WriteOptions{
		Secret: true,
	}
	if err := clipboard.Write(context.Background(), content, opts); err != nil {
		slog.Debug("failed to write to clipboard", "error", err.Error())
		return err
	}
	if autoClear {
		return klip.LaunchDetached(os.Args[0], []string{"__clipd"}, content)
	}
	return nil
}
