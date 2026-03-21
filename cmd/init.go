package cmd

import (
	"fmt"

	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/crypto"
	"git.plutolab.org/plutolab/kosh/internal/logger"
	"git.plutolab.org/plutolab/kosh/internal/model"
	"git.plutolab.org/plutolab/kosh/internal/storage"
	"git.plutolab.org/plutolab/kosh/internal/ui"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize vault with password manager",

	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit()
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

// InitCmd sets up the vault, generates crypto information based on user's provided
// master password
func runInit() error {
	// Check if vault is already initialized
	initialized, err := storage.IsVaultInitialized()
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
	err = storage.InitializeVault(*vault.EncodeToString())
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

	password, err := ui.ReadSecretField(constants.MsgEnterMasterPassword)
	if err != nil {
		logger.Error(constants.ErrFailedToReadInput)
		return "", err
	}

	// Confirm entered password
	confirm, err := ui.ReadSecretField(constants.MsgConfirmMasterPassword)
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
