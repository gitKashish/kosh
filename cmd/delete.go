package cmd

import (
	"database/sql"
	"fmt"
	"strconv"

	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/crypto"
	"git.plutolab.org/plutolab/kosh/internal/logger"
	"git.plutolab.org/plutolab/kosh/internal/ui"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an existing credential by ID",

	Args: cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			logger.Error("%s", constants.ErrIdMustBeInteger.Error())
			return err
		}

		return runDelete(id)
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}

func runDelete(id int) error {
	vault, err := store.GetVaultInfo()
	if err != nil {
		logger.Error("%s", constants.ErrFailedToFetchVaultInfo.Error())
		return err
	}
	vaultData := vault.GetRawData()

	password, err := ui.ReadSecretField(constants.MsgEnterMasterPassword)
	if err != nil {
		logger.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}
	// verify master password and get encryption info
	unlockKey := crypto.GenerateSymmetricKey([]byte(password), vaultData.Salt)

	if _, err := crypto.DecryptSecret(unlockKey, vaultData.Secret, vaultData.Nonce); err != nil {
		logger.Error("%s", constants.ErrIncorrectMasterPassword.Error())
		return err
	}

	// check credential existence
	credential, err := store.GetCredentialById(id)
	if credential == nil && err == sql.ErrNoRows {
		// credential does not exist
		logger.Error("%s", constants.ErrCredentialMatchNotFound.Error())
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
		logger.Error("%s", constants.ErrFailedToReadInput.Error())
		return err
	}

	if !confirm {
		logger.Info(constants.MsgOperationAborted)
		return nil
	}

	err = store.DeleteCredentialById(id)
	if err != nil {
		logger.Error("%s", constants.ErrFailedToDeleteCredential.Error())
	} else {
		logger.Info(constants.MsgDeletedCredential)
	}
	return err
}
