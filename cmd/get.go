package cmd

import (
	"database/sql"
	"time"

	"git.plutolab.org/plutolab/kosh/internal/app"
	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/ui"
	"github.com/spf13/cobra"
)

func NewCmdGet(ctx *app.Context) *cobra.Command {
	getCmd := &cobra.Command{
		Use:   "get <label> <user>",
		Short: "Copy a credential matched by exact label and user",
		Long: `Retrieve a credential by its exact label and user.

Both arguments are matched literally - no fuzzy matching and no partial
matches - so this is the command to use in scripts or when several credentials
share a similar name. Use "kosh search" when you only remember part of the
label.

After the master password is verified the secret is decrypted and copied to the
clipboard; it is never printed to the terminal. A successful lookup also bumps
the credential's access count, which ranks it higher in future searches.`,

		Example: `  Copy the secret for a specific account:
    kosh get github alice

  Labels or users containing spaces must be quoted:
    kosh get "aws prod" "ops-team"`,

		Args: cobra.ExactArgs(2),

		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(cmd, ctx, args[0], args[1])
		},
	}
	return getCmd
}

func runGet(_ *cobra.Command, ctx *app.Context, desiredGroup string, desiredUser string) error {
	// fetch credential info
	credential, err := ctx.Store.GetCredentialByLabelAndUser(desiredGroup, desiredUser)
	if credential == nil && err == sql.ErrNoRows {
		// credential does not exist
		ui.Error("%s", constants.ErrCredentialMatchNotFound.Error())
		return nil
	}

	if err != nil {
		return err
	}

	// get password from user
	password, err := ui.ReadSecretField(constants.MsgEnterMasterPassword)
	if err != nil {
		ui.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	secret, err := ctx.Vault.DecryptCredential(credential, password)
	if err != nil {
		return err
	}

	ui.CopyToClipboard(secret)
	ui.Info(constants.MsgCredentialCopiedToClipboard)

	// on successful access update the access info for the credential,
	// increment access count by 2 on get because it has been fetched
	// with intention meaning that user might be wanting this more
	ctx.Store.UpdateCredentialAccessCount(credential.Id, 2, time.Now())
	return nil
}
