package cmd

import (
	"io"
	"log/slog"
	"os"
	"time"

	"git.plutolab.org/plutolab/kosh/internal/app"
	"github.com/spf13/cobra"
	"plutolab.org/klip"
)

func NewClipDCmd(ctx *app.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "__clipd",
		Short: "Short-lived daemon to manage secrets on clipboard.",
		Long: `A short lived daemon that is triggered when secret is copied to clipboard.
Primary task is to clear secrets after configured seconds (default 30s).
		`,
		Args:               cobra.NoArgs,
		Hidden:             true,
		SilenceUsage:       true,
		DisableFlagParsing: true,
		DisableSuggestions: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunClipDCmd(ctx)
		},
	}
}

func RunClipDCmd(ctx *app.Context) error {
	payload, err := io.ReadAll(os.Stdin)
	if err != nil {
		slog.Debug("failed to read payload from stdin", "error", err)
		return err
	}
	clipboard := klip.NewClipboard()
	return clipboard.ClearAfter(
		klip.DirectCompare(payload),
		time.Duration(ctx.Config.SecretClearTimeout)*time.Second,
	)
}
