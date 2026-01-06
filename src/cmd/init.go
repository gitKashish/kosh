package cmd

import (
	"fmt"

	"git.plutolab.org/plutolab/kosh/src/internals/constants"
	"git.plutolab.org/plutolab/kosh/src/internals/crypto"
	"git.plutolab.org/plutolab/kosh/src/internals/dao"
	"git.plutolab.org/plutolab/kosh/src/internals/interaction"
	"git.plutolab.org/plutolab/kosh/src/internals/logger"
	"git.plutolab.org/plutolab/kosh/src/internals/model"
)

func init() {
	Commands["init"] = CommandInfo{
		Exec:        InitCmd,
		Description: "initialize vault with master password.",
		Usage:       "kosh init",
	}
}

// InitCmd sets up the vault, generates crypto information based on user's provided
// master password
func InitCmd(args ...string) error {
	// Check if vault is already initialized
	initialized, err := dao.IsVaultInitialized()
	if err != nil {
		logger.Error(constants.ErrFailedToInitializeVault)
		return err
	}
	if initialized {
		logger.Info(constants.MsgVaultAlreadyInitialized)
		return nil
	}

	password, err := getPasswordWithConfirmation()
	if err != nil {
		return err
	}

	// Generate random salt for generating key
	salt := crypto.GenerateSalt()

	// Derive unlock key
	key := crypto.GenerateSymmetricKey([]byte(password), salt)

	// Generate ECC key pair
	priv, pub := crypto.GenerateAsymmetricKeyPair()

	// Encrypt password
	cipher, nonce := crypto.EncryptSecret(key, priv)

	vault := &model.VaultData{
		Salt:      salt,
		PublicKey: pub,
		Nonce:     nonce,
		Secret:    cipher,
	}

	// save info to the vault
	err = dao.InitializeVault(*vault.EncodeToString())
	if err != nil {
		logger.Error(constants.ErrFailedToInitializeVault)
	} else {
		logger.Info(constants.MsgVaultInitializedSuccessfully)
	}
	return err
}

// getPasswordWithConfirmation gets password value from user from terminal. It gets password using silent text input and asks for
// password confirmation by re-entering the password. It throws an error if both passwords do not match.
func getPasswordWithConfirmation() (string, error) {

	password, err := interaction.ReadSecretField(constants.MsgEnterMasterPassword)
	if err != nil {
		logger.Error(constants.ErrFailedToReadInput)
		return "", err
	}

	// Confirm entered password
	confirm, err := interaction.ReadSecretField(constants.MsgConfirmMasterPassword)
	if err != nil {
		logger.Error(constants.ErrFailedToReadInput)
		return "", err
	}

	// compare both entries
	if password != confirm {
		return "", fmt.Errorf(constants.ErrPasswordDoesNotMatch)
	}

	return password, nil
}
