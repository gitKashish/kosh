package cmd

import (
	"database/sql"
	"fmt"
	"strconv"

	"git.plutolab.org/plutolab/kosh/internal/app"
	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/ui"
	"github.com/spf13/cobra"
)

func NewCmdDelete(ctx *app.Context) *cobra.Command {
	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Permanently delete a credential by ID",
		Long: `Delete a credential from the active profile's vault, identified by its numeric ID.

Run "kosh list" to find the ID. The master password is verified first, then you
must type out the confirmation phrase shown on screen before the credential is
removed.

Deletion is permanent. The vault is opened with secure_delete enabled, so the
row is overwritten on disk; there is no undo and no trash to restore from.`,

		Example: `  Delete the credential with ID 3:
    kosh delete 3

  Look up the ID first:
    kosh list -l github
    kosh delete 7`,

		Args: cobra.ExactArgs(1),

		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return constants.ErrIdMustBeInteger
			}

			return runDelete(cmd, ctx, id)
		},
	}
	return deleteCmd
}

func runDelete(_ *cobra.Command, ctx *app.Context, id int) error {
	password, err := ui.ReadSecretField(constants.MsgEnterMasterPassword)
	if err != nil {
		ui.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}
	// verify master password and get encryption info
	if err := ctx.Vault.VerifyMasterPassword(password); err != nil {
		ui.Error("%s", err)
		return err
	}

	// check credential existence
	credential, err := ctx.Store.GetCredentialById(id)
	if credential == nil && err == sql.ErrNoRows {
		// credential does not exist
		ui.Error("%s", constants.ErrCredentialMatchNotFound.Error())
		return nil
	}
	if err != nil {
		return err
	}

	ui.Caution(constants.MsgCautionDestructive, "%s", constants.MsgWarnCredentialDelete)

	// get deletion confirmation
	confirm, err := ui.ConfirmWithText(
		constants.MsgConfirmCredentialDelete,
		fmt.Sprintf("delete %s %s", credential.Label, credential.User),
	)
	if err != nil {
		ui.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	if !confirm {
		ui.Info(constants.MsgOperationAborted)
		return nil
	}

	err = ctx.Store.DeleteCredentialById(id)
	if err != nil {
		ui.Error("%s", constants.ErrFailedToDeleteCredential.Error())
	} else {
		ui.Info(constants.MsgCredentialDeleted)
	}
	return err
}
