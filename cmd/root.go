package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"git.plutolab.org/plutolab/kosh/cmd/profile"
	"git.plutolab.org/plutolab/kosh/internal/app"
	"git.plutolab.org/plutolab/kosh/internal/config"
	"git.plutolab.org/plutolab/kosh/internal/core"
	"git.plutolab.org/plutolab/kosh/internal/storage"
	"git.plutolab.org/plutolab/kosh/internal/ui"
	"github.com/spf13/cobra"
)

const (
	DEFAULT_COMMAND = "search"
)

var (
	AppVersion      = "dev"
	builtinCommands = map[string]bool{
		"help":             true,
		"completion":       true,
		"__complete":       true,
		"__completeNoDesc": true,
	}
)

func init() {
	if AppVersion == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			if info.Main.Version != "(devel)" && info.Main.Version != "" {
				AppVersion = info.Main.Version
			}
		}
	}
}

func Execute() {
	// migrate legacy ~/.kosh/kosh.db file structure
	// to new ~/.kosh/profiles/<profile-name>.db
	if err := migrateToProfileFS(); err != nil {
		ui.Error("failed to migrate vault")
		slog.Debug("failed vault migration", "error", err)
		os.Exit(1)
	}

	appCtx := &app.Context{}

	rootCmd := &cobra.Command{
		Use:   "kosh",
		Short: "A secure, local-first CLI password manager",
		Long: `Kosh is a secure, offline vault for storing, generating and retrieving credentials.

Credentials are encrypted with Curve25519, XChaCha20-Poly1305 and Argon2id, and
kept in a local SQLite vault under ~/.kosh - nothing ever leaves your machine.

Retrieved secrets are copied to the clipboard, never printed to the terminal.

Any argument that is not a known subcommand is treated as a search query, so
"kosh github" is shorthand for "kosh search github".

Run "kosh init" once to create the vault before using any other command.`,

		Example: `  Initialize the vault (first time only):
    kosh init

  Add a credential:
    kosh add

  Copy a secret to the clipboard via fuzzy search:
    kosh github alice

  List everything stored in the active profile:
    kosh list`,

		Version:       AppVersion,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// load config
			cfg, err := config.Load()
			if err != nil {
				ui.Error("failed to load kosh config")
				slog.Debug("failed to load config", "error", err)
				os.Exit(1)
			}

			// set profile for UI output messages
			appCtx.Config = cfg
			ui.SetProfile(cfg.ActiveProfile)

			// connect to current profile store
			appCtx.Store, err = storage.InitializeStore(cfg)
			if err != nil {
				ui.Error("error connecting to database")
				slog.Debug("can't connect to database", "error", err)
				os.Exit(1)
			}

			// initialize services
			appCtx.Vault = core.NewVaultService(appCtx.Store)
			appCtx.Profile = core.NewProfileService()
		},

		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if appCtx.Store != nil {
				appCtx.Store.CloseStore()
			}
		},
	}

	// Register children commands
	rootCmd.AddCommand(
		NewCmdUse(appCtx),
		NewCmdInit(appCtx),
		NewCmdList(appCtx),
		NewCmdAdd(appCtx),
		NewCmdGenerate(appCtx),
		NewCmdGet(appCtx),
		NewCmdSearch(appCtx),
		NewCmdUpdate(appCtx),
		NewCmdDelete(appCtx),
		NewCmdCopy(appCtx),
		profile.NewCmdProfile(appCtx),
	)

	// Intercept os.Args to support shorthand `kosh <credential>`
	if len(os.Args) == 1 {
		os.Args = append(os.Args, DEFAULT_COMMAND)
	} else {
		firstArg := os.Args[1]
		// If first arg is not a flag (like --help)
		// and not a built-in command (like add, init, list)
		if !strings.HasPrefix(firstArg, "-") && !isKnownCommand(rootCmd, firstArg) {
			// Inject default command implicitly.
			// Example: ["kosh", "launch_codes"] becomes ["kosh", "search", "launch_codes"]
			os.Args = append(os.Args[:1], append([]string{DEFAULT_COMMAND}, os.Args[1:]...)...)
		}
	}

	if err := rootCmd.Execute(); err != nil {
		// single diagnostic record for any error propagated out of a command,
		// so the layers below can return errors without logging them
		slog.Debug("command failed", "error", err)
		ui.Error("%s", err.Error())
		os.Exit(1)
	}
}

func isKnownCommand(rootCmd *cobra.Command, arg string) bool {
	// Cobra builtins not always in .Commands()
	if builtinCommands[arg] {
		return true
	}
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == arg || cmd.HasAlias(arg) {
			return true
		}
	}
	return false
}

func migrateToProfileFS() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	baseDir := filepath.Join(homeDir, ".kosh")

	legacyPath := filepath.Join(baseDir, "kosh.db")

	profilesDir := filepath.Join(baseDir, "profiles")
	newPath := filepath.Join(profilesDir, "default.db")

	// Exit early if profiles exist.
	if _, err := os.Stat(newPath); err == nil {
		return nil
	}

	// Check if the legacy file exists. If not, exit early.
	if _, err := os.Stat(legacyPath); os.IsNotExist(err) {
		return nil
	}

	ui.Warn("legacy vault found")
	ui.Info("migrating to multi-profile system")

	// Create new profiles directory

	if err := os.MkdirAll(profilesDir, 0700); err != nil {
		return err
	}

	// Move old vault to become new "default" vault
	if err := os.Rename(legacyPath, newPath); err != nil {
		return fmt.Errorf("move %s to %s: %w", legacyPath, newPath, err)
	}

	ui.Info("migrated to multi-profile system")
	return nil
}
