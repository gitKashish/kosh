package cmd

import (
	"crypto/sha256"
	"database/sql"
	"fmt"

	"golang.org/x/crypto/curve25519"

	"git.plutolab.org/plutolab/kosh/src/internals/constants"
	"git.plutolab.org/plutolab/kosh/src/internals/crypto"
	"git.plutolab.org/plutolab/kosh/src/internals/dao"
	"git.plutolab.org/plutolab/kosh/src/internals/interaction"
	"git.plutolab.org/plutolab/kosh/src/internals/logger"
	"git.plutolab.org/plutolab/kosh/src/internals/model"
)

func init() {
	Commands["add"] = CommandInfo{
		Exec:        AddCmd,
		Description: "add a new credential to vault",
		Usage:       "kosh add",
	}
}

func AddCmd(args ...string) error {
	// load vault info
	vault, err := dao.GetVaultInfo()
	if err != nil {
		logger.Error(constants.ErrFailedToFetchVaultInfo)
		return nil
	}
	vaultData := vault.GetRawData()

	// get master password
	password, err := interaction.ReadSecretField(constants.MsgEnterMasterPassword)
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
	label, err := interaction.ReadStringField(constants.MsgEnterCredentialLabel)
	if err != nil {
		logger.Error(constants.ErrFailedToReadInput)
		return err
	}
	// check if provided label is same as a registered command
	if _, found := Commands[label]; found {
		logger.Error(constants.ErrLabelCannotBeCommand)
		logger.Info(constants.MsgListCommandsWithHelp)
		return nil
	}

	user, err := interaction.ReadStringField(constants.MsgEnterCredentialUsername)
	if err != nil {
		logger.Error(constants.ErrFailedToReadInput)
		return err
	}

	// check if a credential already exists for the label and user
	check, err := dao.GetCredentialByLabelAndUser(label, user)
	if check != nil {
		logger.Warn(constants.MsgOperationIsPermanent)
		confirm, err := interaction.ConfirmWithText(
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

	secret, err := interaction.ReadSecretField(constants.MsgEnterCredentialSecret)
	if err != nil {
		logger.Error(constants.ErrFailedToReadInput)
		return err
	}
	confirm, err := interaction.ReadSecretField(constants.MsgConfirmCredentialSecret)
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
	err = dao.AddCredential(credential.EncodeToString())
	if err != nil {
		logger.Error(constants.ErrFailedToSaveCredential)
	} else {
		logger.Info(constants.MsgSavedCredential)
	}
	return err
}
