package cmd

import (
	"crypto/subtle"
	"database/sql"
	"fmt"

	"github.com/spf13/cobra"

	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/logger"
	"git.plutolab.org/plutolab/kosh/internal/ui"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Interactively add a new credential to the vault",

	RunE: func(cmd *cobra.Command, args []string) error {
		return runAdd()
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}

func runAdd() error {
	// get master password
	password, err := ui.ReadSecretField(constants.MsgEnterMasterPassword)
	if err != nil {
		logger.Error("%s", constants.ErrFailedToReadInput.Error())
		return nil
	}

	// verify master password
	err = vault.VerifyMasterPassword(password)
	if err != nil {
		logger.Debug("wrong master password provided")
		return err
	}

	// get credential label
	label, err := ui.ReadStringField(constants.MsgEnterCredentialLabel)
	if err != nil {
		logger.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	// check if provided label is same as a registered command
	if reserved := isKnownCommand(label); reserved {
		logger.Error("%s", constants.ErrLabelCannotBeCommand.Error())
		logger.Info(constants.MsgListCommandsWithHelp)
		return nil
	}

	// get credential user
	user, err := ui.ReadStringField(constants.MsgEnterCredentialUsername)
	if err != nil {
		logger.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	// check if a credential already exists for the label and user
	check, err := store.GetCredentialByLabelAndUser(label, user)
	if check != nil {
		logger.Warn(constants.MsgOperationIsPermanent)
		confirm, err := ui.ConfirmWithText(
			fmt.Sprintf("%s %s", constants.MsgOverwriteCredential, constants.MsgAreYouSure),
			fmt.Sprintf("overwrite %s %s", label, user),
		)

		if err != nil {
			logger.Error("%s", constants.ErrFailedToReadInput.Error())
		}

		if !confirm {
			return nil
		}
	}
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	// get new secret and confirm it
	secret, err := ui.ReadSecretField(constants.MsgEnterCredentialSecret)
	if err != nil {
		logger.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}
	confirm, err := ui.ReadSecretField(constants.MsgConfirmCredentialSecret)
	if err != nil {
		logger.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}
	if subtle.ConstantTimeCompare(secret, confirm) == 0 {
		logger.Error("%s", constants.ErrSecretDoesNotMatch.Error())
		return nil
	}

	// save credential to vault
	if err := vault.AddCredential(label, user, secret); err != nil {
		logger.Error("%s", err.Error())
		return err
	}
	logger.Info(constants.MsgSavedCredential)
	return nil
}
