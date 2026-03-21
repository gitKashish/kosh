package cmd

import (
	"crypto/sha256"
	"database/sql"
	"time"

	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/crypto"
	"git.plutolab.org/plutolab/kosh/internal/logger"
	"git.plutolab.org/plutolab/kosh/internal/model"
	"git.plutolab.org/plutolab/kosh/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/curve25519"
)

var getCmd = &cobra.Command{
	Use:   "get <label> <user>",
	Short: "Retrieve credential by exact label and user",
	Args:  cobra.ExactArgs(2),

	RunE: func(cmd *cobra.Command, args []string) error {
		return runGet(args[0], args[1])
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}

func runGet(desiredGroup string, desiredUser string) error {

	// fetch vault info
	vault, err := store.GetVaultInfo()
	if err != nil {
		return err
	}
	vaultData := vault.GetRawData()

	// fetch credential info
	credential, err := store.GetCredentialByLabelAndUser(desiredGroup, desiredUser)
	if credential == nil && err == sql.ErrNoRows {
		// credential does not exist
		logger.Error(constants.ErrCredentialMatchNotFound)
		return nil
	}

	if err != nil {
		return err
	}

	// get password from user
	password, err := ui.ReadSecretField(constants.MsgEnterMasterPassword)
	if err != nil {
		logger.Error(constants.ErrFailedToReadInput)
		return err
	}

	secret, err := extractSecret(credential.GetRawData(), vaultData, []byte(password))
	if err != nil {
		return err
	}
	// on successful access update the access info for the credential,
	// increment access count by 2 on get because it has been fetched
	// with intention meaning that user might be wanting this more
	store.UpdateCredentialAccessCount(credential.Id, 2, time.Now())

	ui.CopyToClipboard(secret)
	logger.Info(constants.MsgCopiedCredential)
	return nil
}

func extractSecret(credential *model.CredentialData, vault *model.VaultData, masterPassword []byte) ([]byte, error) {
	// decrypt private key using master password
	unlockKey := crypto.GenerateSymmetricKey(masterPassword, vault.Salt)
	privateKey, err := crypto.DecryptSecret(unlockKey, vault.Secret, vault.Nonce)
	if err != nil {
		logger.Error(constants.ErrIncorrectMasterPassword)
		return nil, err
	}

	// decrypt credential using private key
	sharedSecret, err := curve25519.X25519(privateKey, credential.Ephemeral)
	if err != nil {
		logger.Error(constants.ErrFailedToDecryptCredential)
		return nil, err
	}

	key := sha256.Sum256(sharedSecret)

	return crypto.DecryptSecret(key[:], credential.Secret, credential.Nonce)
}
