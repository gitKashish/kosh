package cmd

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"git.plutolab.org/plutolab/kosh/internal/app"
	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/ui"
)

func NewCmdAdd(ctx *app.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "Interactively add a new credential to the vault",
		Long: `Add a credential to the vault of the active profile.

All values are collected through interactive prompts, so nothing sensitive ends
up in your shell history. You are asked for the master password first, then for
the credential's label, user and secret (entered twice to confirm).

A credential is identified by its label and user together, which means the same
label can hold several accounts. If that pair already exists you are asked to
confirm before the stored secret is overwritten.

The label cannot be the name of a kosh subcommand, since bare arguments are
treated as search queries; run "kosh help" to see the reserved names.

To have kosh create the secret for you instead of typing one, use
"kosh generate".`,

		Example: `  Add a credential interactively:
    kosh add`,

		Args: cobra.ExactArgs(0),

		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(cmd, ctx)
		},
	}
}

func runAdd(cmd *cobra.Command, ctx *app.Context) error {
	// get master password
	password, err := ui.ReadSecretField(constants.MsgEnterMasterPassword)
	if err != nil {
		slog.Debug("error reading text field", "error", err)
		fmt.Printf("%s", constants.ErrFailedToReadInput.Error())
		return nil
	}

	// verify master password
	err = ctx.Vault.VerifyMasterPassword([]byte(password))
	if err != nil {
		return err
	}

	// get credential label
	label, err := ui.ReadStringField(constants.MsgEnterCredentialLabel)
	if err != nil {
		ui.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	// check if provided label is same as a registered command
	if reserved := isKnownCommand(cmd.Root(), label); reserved {
		ui.Error("%s", constants.ErrLabelCannotBeCommand.Error())
		ui.Info(constants.MsgHintListCommands)
		return nil
	}

	// get credential user
	user, err := ui.ReadStringField(constants.MsgEnterCredentialUser)
	if err != nil {
		ui.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	// check if a credential already exists for the label and user
	check, err := ctx.Store.GetCredentialByLabelAndUser(label, user)
	if check != nil {
		ui.Caution(constants.MsgCautionOverwrite, "%s", constants.MsgWarnCredentialOverwrite)

		confirm, err := ui.ConfirmWithText(
			constants.MsgConfirmCredentialOverwrite,
			fmt.Sprintf("overwrite %s %s", label, user),
		)

		if err != nil {
			ui.Error("%s", constants.ErrFailedToReadInput.Error())
		}

		if !confirm {
			ui.Info(constants.MsgOperationAborted)
			return nil
		}
	}
	if err != nil && err != constants.ErrCredentialNotFound {
		return err
	}

	// get new secret and confirm it
	secret, err := ui.ReadSecretWithConfirmation(
		constants.MsgEnterCredentialSecret,
		constants.MsgConfirmCredentialSecret,
	)
	if err != nil {
		return err
	}

	// save credential to vault
	if err := ctx.Vault.AddCredential(label, user, secret); err != nil {
		ui.Error("%s", err.Error())
		return err
	}
	ui.Info(constants.MsgCredentialSaved)
	return nil
}
