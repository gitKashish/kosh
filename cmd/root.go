package cmd

import (
	"os"
	"strings"

	"git.plutolab.org/plutolab/kosh/internal/logger"
	"git.plutolab.org/plutolab/kosh/internal/storage"
	"github.com/spf13/cobra"
)

const DEFAULT_COMMAND = "search"

var rootCmd = &cobra.Command{
	Use:   "kosh",
	Short: "A secure CLI password manager",
	Long:  "Kosh is a secure, local vault for storing and generating credentials.",

	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if err := storage.Initialize(); err != nil {
			logger.Error("error connecting to databse")
			logger.Debug("%s", err.Error())
			os.Exit(1)
		}
	},

	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		storage.Close()
	},
}

func Execute() {

	// Intercept os.Args to support shorthand `kosh <credential>`
	if len(os.Args) > 1 {
		firstArg := os.Args[1]

		// If first arg is not a flag (like --help)
		// and not a built-in command (like add, init, list)
		if !strings.HasPrefix(firstArg, "-") && !isKnownCommand(firstArg) {
			// Inject default command implicitly.
			// Example: ["kosh", "launch_codes"] becomes ["kosh", "search", "launch_codes"]
			os.Args = append(os.Args[:1], append([]string{DEFAULT_COMMAND}, os.Args[1:]...)...)
		}
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var builtinCommands = map[string]bool{
	"help":             true,
	"completion":       true,
	"__complete":       true,
	"__completeNoDesc": true,
}

func isKnownCommand(arg string) bool {
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
