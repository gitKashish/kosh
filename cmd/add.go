package cmd

import (
	"crypto/sha256"
	"database/sql"
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/curve25519"

	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/crypto"
	"git.plutolab.org/plutolab/kosh/internal/logger"
	"git.plutolab.org/plutolab/kosh/internal/model"
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
	// load vault info
	vault, err := store.GetVaultInfo()
	if err != nil {
		logger.Error(constants.ErrFailedToFetchVaultInfo)
		return nil
	}
	vaultData := vault.GetRawData()

	// get master password
	password, err := ui.ReadSecretField(constants.MsgEnterMasterPassword)
	if err != nil {
		logger.Error(constants.ErrFailedToReadInput)
		return nil
	}

	// verify master password and get encryption info
	unlockKey := crypto.GenerateSymmetricKey([]byte(password), vaultData.Salt)

	if _, err := crypto.DecryptSecret(unlockKey, vaultData.Secret, vaultData.Nonce); err != nil {
		logger.Error(constants.ErrIncorrectMasterPassword)
		return err
	}

	// get credential details
	label, err := ui.ReadStringField(constants.MsgEnterCredentialLabel)
	if err != nil {
		logger.Error(constants.ErrFailedToReadInput)
		return err
	}
	// check if provided label is same as a registered command
	if reserved := isKnownCommand(label); reserved {
		logger.Error(constants.ErrLabelCannotBeCommand)
		logger.Info(constants.MsgListCommandsWithHelp)
		return nil
	}

	user, err := ui.ReadStringField(constants.MsgEnterCredentialUsername)
	if err != nil {
		logger.Error(constants.ErrFailedToReadInput)
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
			logger.Error(constants.ErrFailedToReadInput)
		}

		if !confirm {
			return nil
		}
	}
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	secret, err := ui.ReadSecretField(constants.MsgEnterCredentialSecret)
	if err != nil {
		logger.Error(constants.ErrFailedToReadInput)
		return err
	}
	confirm, err := ui.ReadSecretField(constants.MsgConfirmCredentialSecret)
	if err != nil {
		logger.Error(constants.ErrFailedToReadInput)
		return err
	}

	if secret != confirm {
		logger.Error(constants.ErrSecretDoesNotMatch)
		return nil
	}

	ephemeralPrivateKey, ephemeralPublicKey := crypto.GenerateAsymmetricKeyPair()

	// generate symmetric shared secret
	encryptionKey, _ := curve25519.X25519(ephemeralPrivateKey, vaultData.PublicKey)

	// hash to get 32 bit consistent key for encryption
	key := sha256.Sum256(encryptionKey)

	cipher, nonce := crypto.EncryptSecret(key[:], []byte(secret))

	credential := model.CredentialData{
		Label:     label,
		User:      user,
		Nonce:     nonce,
		Secret:    cipher,
		Ephemeral: ephemeralPublicKey,
	}

	// save credential
	err = store.AddCredential(credential.EncodeToString())
	if err != nil {
		logger.Error(constants.ErrFailedToSaveCredential)
	} else {
		logger.Info(constants.MsgSavedCredential)
	}
	return err
}
