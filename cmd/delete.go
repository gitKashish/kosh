package cmd

import (
	"database/sql"
	"fmt"
	"strconv"

	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/crypto"
	"git.plutolab.org/plutolab/kosh/internal/logger"
	"git.plutolab.org/plutolab/kosh/internal/storage"
	"git.plutolab.org/plutolab/kosh/internal/ui"
)

func init() {
	Commands["delete"] = CommandInfo{
		Exec:        DeleteCmd,
		Usage:       "kosh delete <credential_id>",
		Description: "delete a stored credential.",
	}
}

func DeleteCmd(args ...string) error {
	vault, err := storage.GetVaultInfo()
	if err != nil {
		logger.Error(constants.ErrFailedToFetchVaultInfo)
		return err
	}
	vaultData := vault.GetRawData()

	if len(args) != 1 {
		logger.Error(constants.ErrInvalidArguments)
		HelpCmd()
		return fmt.Errorf("missing argument got %d, want 1", len(args))
	}

	delete_id, err := strconv.Atoi(args[0])
	if err != nil {
		logger.Error(constants.ErrIdMustBeInteger)
		return err
	}

	password, err := ui.ReadSecretField(constants.MsgEnterMasterPassword)
	if err != nil {
		logger.Error(constants.ErrFailedToReadInput)
		return err
	}
	// verify master password and get encryption info
	unlockKey := crypto.GenerateSymmetricKey([]byte(password), vaultData.Salt)

	if _, err := crypto.DecryptSecret(unlockKey, vaultData.Secret, vaultData.Nonce); err != nil {
		logger.Error(constants.ErrIncorrectMasterPassword)
		return err
	}

	// check credential existence
	credential, err := storage.GetCredentialById(delete_id)
	if credential == nil && err == sql.ErrNoRows {
		// credential does not exist
		logger.Error(constants.ErrCredentialMatchNotFound)
		return nil
	}

	if err != nil {
		return err
	}

	// get deletion confirmation
	logger.Warn(constants.MsgOperationIsPermanent)
	confirm, err := ui.ConfirmWithText(
		fmt.Sprintf("%s %s", constants.MsgDeleteCredential, constants.MsgAreYouSure),
		fmt.Sprintf("delete %s %s", credential.Label, credential.User),
	)
	if err != nil {
		logger.Error(constants.ErrFailedToReadInput)
		return err
	}

	if !confirm {
		logger.Info(constants.MsgOperationAborted)
		return nil
	}

	err = storage.DeleteCredentialById(delete_id)
	if err != nil {
		logger.Error(constants.ErrFailedToDeleteCredential)
	} else {
		logger.Info(constants.MsgDeletedCredential)
	}
	return err
}
