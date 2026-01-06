package cmd

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"time"

	"git.plutolab.org/plutolab/kosh/src/internals/constants"
	"git.plutolab.org/plutolab/kosh/src/internals/crypto"
	"git.plutolab.org/plutolab/kosh/src/internals/dao"
	"git.plutolab.org/plutolab/kosh/src/internals/interaction"
	"git.plutolab.org/plutolab/kosh/src/internals/logger"
	"git.plutolab.org/plutolab/kosh/src/internals/model"
	"golang.org/x/crypto/curve25519"
)

func init() {
	Commands["get"] = CommandInfo{
		Exec:        GetCmd,
		Description: "retrieve a stored credential",
		Usage:       "kosh get <label> <user>",
	}
}

func GetCmd(args ...string) error {
	if len(args) < 2 {
		logger.Error(constants.ErrInvalidArguments)
		HelpCmd()
		return fmt.Errorf("missing arguments, got %d, want %d", len(args), 2)
	}

	desiredGroup := args[0]
	desiredUser := args[1]

	// fetch vault info
	vault, err := dao.GetVaultInfo()
	if err != nil {
		return err
	}
	vaultData := vault.GetRawData()

	// fetch credential info
	credential, err := dao.GetCredentialByLabelAndUser(desiredGroup, desiredUser)
	if credential == nil && err == sql.ErrNoRows {
		// credential does not exist
		logger.Error(constants.ErrCredentialMatchNotFound)
		return nil
	}

	if err != nil {
		return err
	}

	// get password from user
	password, err := interaction.ReadSecretField(constants.MsgEnterMasterPassword)
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
	dao.UpdateCredentialAccessCount(credential.Id, 2, time.Now())

	interaction.CopyToClipboard(secret)
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
