package profile

import (
	"log/slog"

	"git.plutolab.org/plutolab/kosh/internal/app"
	"git.plutolab.org/plutolab/kosh/internal/model"
	"git.plutolab.org/plutolab/kosh/internal/ui"
	"github.com/spf13/cobra"
)

func NewCmdList(ctx *app.Context) *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list [filter]",
		Short: "Show the profiles and which one is active",
		Long: `List the profiles stored under ~/.kosh/profiles.

The table shows each profile's name and whether it is the active one - the
profile that every other command reads from and writes to. Switch between them
with "kosh use".

Only profile names are read, so no vault is opened and no master password is
needed. Pass a filter to limit the table to profiles whose name contains that
substring.`,

		Example: `  List every profile:
    kosh profile list

  Only profiles whose name contains "work":
    kosh profile list work`,

		Args: cobra.RangeArgs(0, 1),

		RunE: func(cmd *cobra.Command, args []string) error {
			var filter string
			if len(args) == 1 {
				filter = args[0]
			}
			return runList(cmd, ctx, filter)
		},
	}

	return listCmd
}

func runList(_ *cobra.Command, ctx *app.Context, filter string) error {
	profiles, err := ctx.Profile.LoadProfiles(filter, false)
	if err != nil {
		slog.Debug("failed to load profile list", "error", err)
		return err
	}

	displayProfiles(profiles, ctx.Config.ActiveProfile)
	return nil
}

func displayProfiles(profiles []model.Profile, activeProfileName string) {
	t := ui.NewTable("Profile", "Status")
	for i, p := range profiles {
		status := "inactive"
		if string(p) == activeProfileName {
			status = "active"
			t.SetActive(i)
		}
		t.AddRow(p.Display(), status)
	}
	t.Display()
}
