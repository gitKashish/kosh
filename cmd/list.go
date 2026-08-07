package cmd

import (
	"fmt"
	"strings"
	"time"

	"git.plutolab.org/plutolab/kosh/internal/app"
	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/model"
	"git.plutolab.org/plutolab/kosh/internal/ui"
	"github.com/spf13/cobra"
)

func NewCmdList(ctx *app.Context) *cobra.Command {
	var listLabel, listUser string

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Show a table of saved credentials",
		Long: `List the credentials stored in the active profile's vault.

The table shows each credential's ID, label, user, creation, update and last
access times, and how often it has been accessed. The ID is what "kosh update",
"kosh delete" and "kosh copy" take as their argument.

Only metadata is listed. Secrets are never decrypted or displayed here, so the
command needs no master password.

The --label and --user flags narrow the output to entries containing the given
substring; combining them requires both to match.`,

		Example: `  List every credential:
    kosh list

  Only credentials whose label contains "git":
    kosh list --label git

  Combine both filters:
    kosh list -l aws -u alice`,

		Args: cobra.ExactArgs(0),

		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, ctx, listLabel, listUser)
		},
	}

	listCmd.Flags().StringVarP(&listLabel, "label", "l", "", "filter creds that contain label string")
	listCmd.Flags().StringVarP(&listUser, "user", "u", "", "filter creds that contain user string")

	return listCmd
}

func runList(_ *cobra.Command, ctx *app.Context, label string, user string) error {

	credentials, err := ctx.Vault.ListCredentials(label, user)
	if err != nil {
		ui.Error("%s", constants.ErrCredentialMatchNotFound.Error())
		return err
	}

	displayFilters(label, user)
	displayCredentials(credentials)

	return nil
}

func displayCredentials(credentials []model.CredentialSummary) {
	t := ui.NewTable("ID", "Label", "User", "Access Count", "Last Used", "Last Updated", "Created At")
	for _, c := range credentials {
		t.AddRow(
			fmt.Sprintf("%02d", c.Id),
			c.Label,
			c.User,
			fmt.Sprintf("%03d times", c.AccessCount),
			ui.RelativeTime(c.AccessedAt, time.Now()),
			ui.RelativeTime(c.UpdatedAt, time.Now()),
			c.CreatedAt.Format(time.RFC1123),
		)
	}
	t.Display()
}

func displayFilters(label, user string) {
	// Show active filters
	filters := []string{}
	if label != "" || user != "" {
		if label != "" {
			filters = append(filters, fmt.Sprintf("label contains '%s'", label))
		}
		if user != "" {
			filters = append(filters, fmt.Sprintf("user contains '%s'", user))
		}
	} else {
		filters = []string{"none"}
	}
	ui.Muted("filters: %s\n", strings.Join(filters, " and "))
}
