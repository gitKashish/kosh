package profile

import (
	"git.plutolab.org/plutolab/kosh/internal/app"
	"github.com/spf13/cobra"
)

func NewCmdProfile(ctx *app.Context) *cobra.Command {
	profileCmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage profiles and their vaults",
		Long: `Manage the profiles kept under ~/.kosh/profiles.

A profile is a separate vault file with its own credentials and its own master
password, which keeps unrelated secrets - work and personal, say - fully
isolated from each other. Every other kosh command operates on the active
profile only.

The subcommands here list, create and delete profiles. To switch the active
profile use "kosh use", and to copy a credential from one profile to another use
"kosh copy".`,

		Example: `  List the profiles:
    kosh profile list

  Create a profile and set up its vault:
    kosh profile create work

  Delete a profile and every credential in it:
    kosh profile delete work`,
	}

	// Register children commands
	profileCmd.AddCommand(
		NewCmdList(ctx),
		NewCmdCreate(ctx),
		NewCmdDelete(ctx),
	)
	return profileCmd
}
