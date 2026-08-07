package cmd

import (
	"crypto/subtle"
	"fmt"
	"log/slog"
	"strconv"

	"git.plutolab.org/plutolab/kosh/internal/app"
	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/model"
	"git.plutolab.org/plutolab/kosh/internal/ui"
	"github.com/spf13/cobra"
)

func NewCmdUpdate(ctx *app.Context) *cobra.Command {
	updateCmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update the label, user or secret of a credential",
		Long: `Update an existing credential, identified by its numeric ID.

Run "kosh list" to find the ID. After the master password is verified you choose
which single field to change - label, user or secret - and are asked to confirm
before anything is written.

Renaming a label or user cannot produce a duplicate: if another credential
already uses the resulting label and user pair, the update is aborted. Changing
the secret re-encrypts it with a fresh ephemeral key and nonce, and the previous
value is gone for good.`,

		Example: `  Update the credential with ID 3:
    kosh update 3

  Look up the ID first:
    kosh list -l github
    kosh update 7`,

		Args: cobra.ExactArgs(1),

		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				ui.Error("%s", constants.ErrIdMustBeInteger.Error())
				return err
			}
			return runUpdate(cmd, ctx, id)
		},
	}
	return updateCmd
}

func runUpdate(_ *cobra.Command, ctx *app.Context, id int) error {
	password, err := ui.ReadSecretField(constants.MsgEnterMasterPassword)
	if err != nil {
		ui.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	if err := ctx.Vault.VerifyMasterPassword(password); err != nil {
		ui.Error("%s", constants.ErrIncorrectMasterPassword.Error())
		return err
	}

	// check credential existence
	credential, err := ctx.Store.GetCredentialById(id)
	if err == constants.ErrCredentialNotFound {
		// credential does not exist
		ui.Error("%s", constants.ErrCredentialNotFound.Error())
		return nil
	}

	if err != nil {
		ui.Error("%s", constants.ErrFailedToFetchCredential.Error())
		return err
	}

	updateOptions := []string{"label", "user", "secret", "abort"}
	option := ui.GetOptionFieldWithRetry(
		constants.MsgSelectCredentialField,
		updateOptions,
		3,
	)

	switch option {
	case 0:
		err = updateLabel(ctx, credential)
	case 1:
		err = updateUser(ctx, credential)
	case 2:
		err = updateSecret(ctx, credential)
	case 3:
		ui.Info(constants.MsgOperationAborted)
		return nil
	default:
		ui.Error("%s", constants.ErrInvalidArguments.Error())
		return nil
	}

	if err != nil {
		ui.Error("failed to update %s", updateOptions[option])
	}

	return err
}

func updateLabel(ctx *app.Context, credential *model.Credential) error {
	newLabel, err := ui.ReadStringField(constants.MsgEnterCredentialLabel)
	if err != nil {
		ui.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	existingCredential, err := ctx.Store.GetCredentialByLabelAndUser(newLabel, credential.User)
	if err != nil && err != constants.ErrCredentialNotFound {
		ui.Error("%s", constants.ErrFailedToFetchCredential.Error())
		return err
	}

	if existingCredential != nil {
		ui.Error("%s", constants.ErrCredentialAlreadyExists.Error())
		ui.Info(constants.MsgOperationAborted)
		return nil
	}

	confirmationText := fmt.Sprintf(
		"update label from %s to %s",
		credential.Label,
		newLabel,
	)
	confirm, err := ui.ConfirmWithText(
		constants.MsgConfirmCredentialChange,
		confirmationText,
	)
	if err != nil {
		ui.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	if !confirm {
		ui.Info(constants.MsgOperationAborted)
		return nil
	}

	err = ctx.Store.UpdateCredential(&model.Credential{
		Label: newLabel,
		Id:    credential.Id,
	})

	if err == nil {
		ui.Info(constants.MsgCredentialUpdated)
	}

	return err
}

func updateUser(ctx *app.Context, credential *model.Credential) error {
	newUser, err := ui.ReadStringField(constants.MsgEnterCredentialUser)
	if err != nil {
		ui.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	existingCredential, err := ctx.Store.GetCredentialByLabelAndUser(credential.Label, newUser)
	if err != nil && err != constants.ErrCredentialNotFound {
		ui.Error("%s", constants.ErrFailedToFetchCredential.Error())
		return err
	}

	if existingCredential != nil {
		ui.Error("%s", constants.ErrCredentialAlreadyExists.Error())
		ui.Info(constants.MsgOperationAborted)
		return nil
	}

	confirmationText := fmt.Sprintf(
		"update user from %s to %s",
		credential.User,
		newUser,
	)
	confirm, err := ui.ConfirmWithText(
		constants.MsgConfirmCredentialChange,
		confirmationText,
	)
	if err != nil {
		ui.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	if !confirm {
		ui.Info(constants.MsgOperationAborted)
		return nil
	}

	err = ctx.Store.UpdateCredential(&model.Credential{
		User: newUser,
		Id:   credential.Id,
	})

	if err == nil {
		ui.Info(constants.MsgCredentialUpdated)
	}

	return err
}

func updateSecret(ctx *app.Context, credential *model.Credential) error {
	newSecret, err := ui.ReadSecretField(constants.MsgEnterCredentialSecret)
	if err != nil {
		ui.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	confirmSecret, err := ui.ReadSecretField(constants.MsgConfirmCredentialSecret)
	if err != nil {
		ui.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	if subtle.ConstantTimeCompare(newSecret, confirmSecret) == 0 {
		ui.Error("%s", constants.ErrSecretDoesNotMatch.Error())
		return nil
	}

	confirm, err := ui.ConfirmWithText(
		constants.MsgOperationIsPermanent,
		fmt.Sprintf("update %s credential secret", credential.Label),
	)
	if err != nil {
		ui.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	if !confirm {
		ui.Info(constants.MsgOperationAborted)
		return nil
	}

	err = ctx.Vault.UpdateCredentialSecret(credential.Id, newSecret)
	if err != nil {
		ui.Error("%s", constants.ErrFailedToSaveCredential.Error())
		// error is deliberately not propagated, so this is the only record of it
		slog.Debug("failed to update credential secret", "error", err)
	} else {
		ui.Info(constants.MsgCredentialUpdated)
	}

	return nil
}
