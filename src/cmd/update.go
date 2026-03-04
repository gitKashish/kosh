package cmd

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strconv"

	"git.plutolab.org/plutolab/kosh/src/internals/constants"
	"git.plutolab.org/plutolab/kosh/src/internals/crypto"
	"git.plutolab.org/plutolab/kosh/src/internals/dao"
	"git.plutolab.org/plutolab/kosh/src/internals/interaction"
	"git.plutolab.org/plutolab/kosh/src/internals/logger"
	"git.plutolab.org/plutolab/kosh/src/internals/model"
	"golang.org/x/crypto/curve25519"
)

func init() {
	Commands["update"] = CommandInfo{
		Exec:        UpdateCmd,
		Description: "update existing credential",
		Usage:       "kosh update <id>",
	}
}

func UpdateCmd(args ...string) error {
	vault, err := dao.GetVaultInfo()
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

	update_id, err := strconv.Atoi(args[0])
	if err != nil {
		logger.Error(constants.ErrIdMustBeInteger)
		return err
	}

	password, err := interaction.ReadSecretField(constants.MsgEnterMasterPassword)
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
	credential, err := dao.GetCredentialById(update_id)
	if err == sql.ErrNoRows {
		// credential does not exist
		logger.Error(constants.ErrCredentialNotFound)
		return nil
	}

	if err != nil {
		logger.Error(constants.ErrFailedToFetchCredential)
		return err
	}

	// TODO: show what credential is the user updating

	updateOptions := []string{"label", "user", "secret", "abort"}
	option := interaction.GetOptionFieldWithRetry(
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
		err = updateSecret(credential, vaultData)
	case 3:
		logger.Info(constants.MsgOperationAborted)
		return nil
	default:
		logger.Error(constants.ErrInvalidArguments)
		return nil
	}

	if err != nil {
		logger.Error("failed to update %s", updateOptions[option])
	}

	return err
}

func updateLabel(credential *model.Credential) error {
	newLabel, err := interaction.ReadStringField(constants.MsgEnterCredentialLabel)
	if err != nil {
		logger.Error(constants.ErrFailedToReadInput)
		return err
	}

	existingCredential, err := dao.GetCredentialByLabelAndUser(newLabel, credential.User)
	if err != nil && err != sql.ErrNoRows {
		logger.Error(constants.ErrFailedToFetchCredential)
		return err
	}

	if existingCredential != nil {
		logger.Error(constants.ErrCredentialAlreadyExists)
		logger.Info(constants.MsgOperationAborted)
		return nil
	}

	confirmationText := fmt.Sprintf(
		"update label from %s to %s",
		credential.Label,
		newLabel,
	)
	confirm, err := interaction.ConfirmWithText(
		constants.MsgOperationIsPermanent,
		confirmationText,
	)
	if err != nil {
		logger.Error(constants.ErrFailedToReadInput)
		return err
	}

	if !confirm {
		logger.Info(constants.MsgOperationAborted)
		return nil
	}

	err = dao.UpdateCredential(&model.Credential{
		Label: newLabel,
		Id:    credential.Id,
	})

	if err == nil {
		logger.Info("%s", constants.MsgUpdatedCredential)
	}

	return err
}

func updateUser(credential *model.Credential) error {
	newUser, err := interaction.ReadStringField(constants.MsgEnterCredentialUsername)
	if err != nil {
		logger.Error(constants.ErrFailedToReadInput)
		return err
	}

	existingCredential, err := dao.GetCredentialByLabelAndUser(credential.Label, newUser)
	if err != nil && err != sql.ErrNoRows {
		logger.Error(constants.ErrFailedToFetchCredential)
		return err
	}

	if existingCredential != nil {
		logger.Error(constants.ErrCredentialAlreadyExists)
		logger.Info(constants.MsgOperationAborted)
		return nil
	}

	confirmationText := fmt.Sprintf(
		"update user from %s to %s",
		credential.User,
		newUser,
	)
	confirm, err := interaction.ConfirmWithText(
		constants.MsgOperationIsPermanent,
		confirmationText,
	)
	if err != nil {
		logger.Error(constants.ErrFailedToReadInput)
		return err
	}

	if !confirm {
		logger.Info(constants.MsgOperationAborted)
		return nil
	}

	err = dao.UpdateCredential(&model.Credential{
		User: newUser,
		Id:   credential.Id,
	})

	if err == nil {
		logger.Info("%s", constants.MsgUpdatedCredential)
	}

	return err
}

func updateSecret(credential *model.Credential, vaultData *model.VaultData) error {
	newSecret, err := interaction.ReadSecretField(constants.MsgEnterCredentialSecret)
	if err != nil {
		logger.Error(constants.ErrFailedToReadInput)
		return err
	}

	confirmSecret, err := interaction.ReadSecretField(constants.MsgConfirmCredentialSecret)
	if err != nil {
		logger.Error(constants.ErrFailedToReadInput)
		return err
	}

	if newSecret != confirmSecret {
		logger.Error(constants.ErrSecretDoesNotMatch)
		return nil
	}

	logger.Warn(constants.MsgOverwriteCredential)
	confirm, err := interaction.ConfirmWithText(
		constants.MsgOperationIsPermanent,
		fmt.Sprintf("update %s credential secret", credential.Label),
	)
	if err != nil {
		logger.Error(constants.ErrFailedToReadInput)
		return err
	}

	if !confirm {
		logger.Info(constants.MsgOperationAborted)
		return nil
	}

	ephemeralPrivateKey, ephemeralPublicKey := crypto.GenerateAsymmetricKeyPair()

	// generate symmetric shared secret
	encryptionKey, _ := curve25519.X25519(ephemeralPrivateKey, vaultData.PublicKey)

	// hash to get 32 bit consistent key for encryption
	key := sha256.Sum256(encryptionKey)

	cipher, nonce := crypto.EncryptSecret(key[:], []byte(newSecret))

	updatedCredential := model.CredentialData{
		Id:        credential.Id,
		Label:     credential.Label,
		User:      credential.User,
		Nonce:     nonce,
		Secret:    cipher,
		Ephemeral: ephemeralPublicKey,
	}

	err = dao.UpdateCredential(updatedCredential.EncodeToString())

	if err != nil {
		logger.Error(constants.ErrFailedToSaveCredential)
		logger.Debug("%v", err)
	} else {
		logger.Info("%s", constants.MsgUpdatedCredential)
	}

	return nil
}
