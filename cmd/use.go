package cmd

import (
	"log/slog"
	"strings"

	"git.plutolab.org/plutolab/kosh/internal/app"
	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/model"
	"git.plutolab.org/plutolab/kosh/internal/ui"
	"github.com/spf13/cobra"
)

func NewCmdUse(ctx *app.Context) *cobra.Command {
	useCmd := &cobra.Command{
		Use:   "use [profile]",
		Short: "Switch the active profile",
		Long: `Switch the profile that every other command operates on.

Each profile is a separate vault file under ~/.kosh/profiles, with its own
credentials and its own master password - useful for keeping work and personal
secrets apart.

Given a profile name, kosh switches to it, failing if it does not exist. With no
arguments the available profiles are listed in an interactive search: type to
filter and press enter to select.

To switch to a profile that does not exist yet, create it first with
"kosh profile create", which also sets up its vault.

The choice is written to the kosh config and persists across runs.`,

		Example: `  Pick a profile interactively:
    kosh use

  Switch to an existing profile:
    kosh use work

  Create a profile, then switch back:
    kosh profile create work
    kosh use default`,

		Args: cobra.RangeArgs(0, 1),

		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return searchProfiles(ctx)
			} else {
				return useProfile(ctx, model.Profile(args[0]))
			}
		},
	}
	return useCmd
}

func useProfile(ctx *app.Context, profile model.Profile) error {
	// Look the profile up ignoring case, then switch to the spelling it is
	// stored under - switching to the one the user typed would point the config
	// at a path that does not exist on a case-sensitive filesystem.
	stored, exists, err := ctx.Profile.ResolveProfile(string(profile))
	if err != nil {
		return err
	}
	if !exists {
		return constants.ErrProfileDoesNotExist
	}

	if err := ctx.Profile.SwitchProfile(stored); err != nil {
		return err
	}

	ui.Info(constants.MsgProfileSwitched)
	return nil
}

func searchProfiles(ctx *app.Context) error {
	var filter string
	profiles, err := ctx.Profile.LoadProfiles(filter, false)
	if err != nil {
		return err
	}
	slog.Debug("loaded profiles", "count", len(profiles))
	searchResult := make([]model.Profile, 0, len(profiles))

	profile, err := ui.InteractiveSearch(
		constants.MsgSearchProfile,
		func(query string) []model.Profile {
			clear(searchResult)
			searchResult = searchResult[:0]

			for _, p := range profiles {
				if strings.Contains(string(p), query) || strings.Contains(query, string(p)) {
					searchResult = append(searchResult, p)
				}
			}
			return searchResult
		},
	)
	if err != nil {
		return err
	}

	if err := ctx.Profile.SwitchProfile(profile); err != nil {
		return err
	}

	ui.Info(constants.MsgProfileSwitched)
	return nil
}
