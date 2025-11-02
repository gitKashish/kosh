package cmd

import (
	"fmt"

	"github.com/gitKashish/kosh/src/internals/crypto"
	"github.com/gitKashish/kosh/src/internals/dao"
	"github.com/gitKashish/kosh/src/internals/interaction"
	"github.com/gitKashish/kosh/src/internals/logger"
	"github.com/gitKashish/kosh/src/internals/model"
)

func init() {
	Commands["init"] = CommandInfo{
		Exec:        InitCmd,
		Description: "Initialize vault with master password.",
		Usage:       "kosh init",
	}
}

// InitCmd sets up the vault, generates crypto information based on user's provided
// master password
func InitCmd(args ...string) error {
	// Check if vault is already initialized
	initialized, err := dao.IsVaultInitialized()
	if err != nil {
		logger.Error("failed to check vault initialization")
		return err
	}
	if initialized {
		logger.Info("vault already initialized")
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
		logger.Error("error initializing vault")
	} else {
		logger.Info("vault initialized successfully")
	}
	return err
}

// getPasswordWithConfirmation gets password value from user from terminal. It gets password using silent text input and asks for
// password confirmation by re-entering the password. It throws an error if both passwords do not match.
func getPasswordWithConfirmation() (string, error) {

	password, err := interaction.ReadSecretField("enter master password > ")
	if err != nil {
		logger.Error("unable to read password")
		return "", err
	}

	// Confirm entered password
	confirm, err := interaction.ReadSecretField("confirm master password > ")
	if err != nil {
		logger.Error("unable to read confirmation")
		return "", err
	}

	// compare both entries
	if password != confirm {
		return "", fmt.Errorf("passwords do not match")
	}

	return password, nil
}
