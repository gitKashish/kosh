package cmd

import (
	"crypto/subtle"
	"fmt"

	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/crypto"
	"git.plutolab.org/plutolab/kosh/internal/logger"
	"git.plutolab.org/plutolab/kosh/internal/model"
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
	initialized, err := store.IsVaultInitialized()
	if err != nil {
		logger.Error("%s", constants.ErrFailedToInitializeVault.Error())
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
	cipher, nonce, err := crypto.EncryptSecret(key, priv)
	if err != nil {
		return err
	}

	vault := &model.VaultData{
		Salt:      salt,
		PublicKey: pub,
		Nonce:     nonce,
		Secret:    cipher,
	}

	// save info to the vault
	err = store.InitializeVault(*vault.EncodeToString())
	if err != nil {
		logger.Error("%s", constants.ErrFailedToInitializeVault.Error())
	} else {
		logger.Info(constants.MsgVaultInitializedSuccessfully)
	}
	return err
}

// getPasswordWithConfirmation gets password value from user from terminal. It gets password using silent text input and asks for
// password confirmation by re-entering the password. It throws an error if both passwords do not match.
func getPasswordWithConfirmation() ([]byte, error) {

	password, err := ui.ReadSecretField(constants.MsgEnterMasterPassword)
	if err != nil {
		logger.Error("%s", constants.ErrFailedToReadInput.Error())
		return nil, err
	}

	// Confirm entered password
	confirm, err := ui.ReadSecretField(constants.MsgConfirmMasterPassword)
	if err != nil {
		logger.Error("%s", constants.ErrFailedToReadInput.Error())
		return nil, err
	}

	// compare both entries
	if subtle.ConstantTimeCompare(password, confirm) == 0 {
		return nil, fmt.Errorf("%s", constants.ErrPasswordDoesNotMatch.Error())
	}

	return password, nil
}
