package cmd

import (
	"database/sql"
	"fmt"
	"strconv"

	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/logger"
	"git.plutolab.org/plutolab/kosh/internal/model"
	"git.plutolab.org/plutolab/kosh/internal/ui"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an existing credential by ID",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			logger.Error("%s", constants.ErrIdMustBeInteger.Error())
			return err
		}
		return runUpdate(id)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(id int) error {
	password, err := ui.ReadSecretField(constants.MsgEnterMasterPassword)
	if err != nil {
		logger.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	if err := vault.VerifyMasterPassword(password); err != nil {
		logger.Error("%s", constants.ErrIncorrectMasterPassword.Error())
		return err
	}

	// check credential existence
	credential, err := store.GetCredentialById(id)
	if err == sql.ErrNoRows {
		// credential does not exist
		logger.Error("%s", constants.ErrCredentialNotFound.Error())
		return nil
	}

	if err != nil {
		logger.Error("%s", constants.ErrFailedToFetchCredential.Error())
		return err
	}

	updateOptions := []string{"label", "user", "secret", "abort"}
	option := ui.GetOptionFieldWithRetry(
		constants.MsgSelectCredentialFieldToUpdate,
		updateOptions,
		3,
	)

	switch option {
	case 0:
		err = updateLabel(credential)
	case 1:
		err = updateUser(credential)
	case 2:
		err = updateSecret(credential)
	case 3:
		logger.Info(constants.MsgOperationAborted)
		return nil
	default:
		logger.Error("%s", constants.ErrInvalidArguments.Error())
		return nil
	}

	if err != nil {
		logger.Error("failed to update %s", updateOptions[option])
	}

	return err
}

func updateLabel(credential *model.Credential) error {
	newLabel, err := ui.ReadStringField(constants.MsgEnterCredentialLabel)
	if err != nil {
		logger.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	existingCredential, err := store.GetCredentialByLabelAndUser(newLabel, credential.User)
	if err != nil && err != sql.ErrNoRows {
		logger.Error("%s", constants.ErrFailedToFetchCredential.Error())
		return err
	}

	if existingCredential != nil {
		logger.Error("%s", constants.ErrCredentialAlreadyExists.Error())
		logger.Info(constants.MsgOperationAborted)
		return nil
	}

	confirmationText := fmt.Sprintf(
		"update label from %s to %s",
		credential.Label,
		newLabel,
	)
	confirm, err := ui.ConfirmWithText(
		constants.MsgOperationIsPermanent,
		confirmationText,
	)
	if err != nil {
		logger.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	if !confirm {
		logger.Info(constants.MsgOperationAborted)
		return nil
	}

	err = store.UpdateCredential(&model.Credential{
		Label: newLabel,
		Id:    credential.Id,
	})

	if err == nil {
		logger.Info("%s", constants.MsgUpdatedCredential)
	}

	return err
}

func updateUser(credential *model.Credential) error {
	newUser, err := ui.ReadStringField(constants.MsgEnterCredentialUsername)
	if err != nil {
		logger.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	existingCredential, err := store.GetCredentialByLabelAndUser(credential.Label, newUser)
	if err != nil && err != sql.ErrNoRows {
		logger.Error("%s", constants.ErrFailedToFetchCredential.Error())
		return err
	}

	if existingCredential != nil {
		logger.Error("%s", constants.ErrCredentialAlreadyExists.Error())
		logger.Info(constants.MsgOperationAborted)
		return nil
	}

	confirmationText := fmt.Sprintf(
		"update user from %s to %s",
		credential.User,
		newUser,
	)
	confirm, err := ui.ConfirmWithText(
		constants.MsgOperationIsPermanent,
		confirmationText,
	)
	if err != nil {
		logger.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	if !confirm {
		logger.Info(constants.MsgOperationAborted)
		return nil
	}

	err = store.UpdateCredential(&model.Credential{
		User: newUser,
		Id:   credential.Id,
	})

	if err == nil {
		logger.Info("%s", constants.MsgUpdatedCredential)
	}

	return err
}

func updateSecret(credential *model.Credential) error {
	newSecret, err := ui.ReadSecretField(constants.MsgEnterCredentialSecret)
	if err != nil {
		logger.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	confirmSecret, err := ui.ReadSecretField(constants.MsgConfirmCredentialSecret)
	if err != nil {
		logger.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	if newSecret != confirmSecret {
		logger.Error("%s", constants.ErrSecretDoesNotMatch.Error())
		return nil
	}

	logger.Warn(constants.MsgOverwriteCredential)
	confirm, err := ui.ConfirmWithText(
		constants.MsgOperationIsPermanent,
		fmt.Sprintf("update %s credential secret", credential.Label),
	)
	if err != nil {
		logger.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	if !confirm {
		logger.Info(constants.MsgOperationAborted)
		return nil
	}

	err = vault.UpdateCredentialSecret(credential.Id, newSecret)
	if err != nil {
		logger.Error("%s", constants.ErrFailedToSaveCredential.Error())
		logger.Debug("%v", err)
	} else {
		logger.Info("%s", constants.MsgUpdatedCredential)
	}

	return nil
}
