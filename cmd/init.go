package cmd

import (
	"git.plutolab.org/plutolab/kosh/internal/app"
	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/crypto"
	"git.plutolab.org/plutolab/kosh/internal/model"
	"git.plutolab.org/plutolab/kosh/internal/ui"
	"github.com/spf13/cobra"
)

func NewCmdInit(ctx *app.Context) *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the vault for the active profile",
		Long: `Initialize the vault of the active profile with a master password.

You are prompted for a master password twice. The password itself is never
stored: it is combined with a random salt to derive the key that encrypts the
vault's private key, and re-derived every time you unlock the vault.

Losing the master password permanently locks the vault - there is no recovery
mechanism and no way to reset it.

This command is safe to re-run: if the vault is already initialized it reports
that and exits without touching any existing data. Each profile has its own
vault and master password, but profiles made with "kosh profile create" are
initialized as part of creating them, so this is only needed for the first
profile.`,

		Example: `  Initialize the active profile's vault:
    kosh init`,

		Args: cobra.ExactArgs(0),

		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd, ctx)
		},
	}
	return initCmd
}

// InitCmd sets up the vault, generates crypto information based on user's provided
// master password
func runInit(_ *cobra.Command, ctx *app.Context) error {
	// Check if vault is already initialized
	initialized, err := ctx.Store.IsVaultInitialized()
	if err != nil {
		ui.Error("%s", constants.ErrFailedToInitializeVault.Error())
		return err
	}
	if initialized {
		ui.Info(constants.MsgVaultAlreadyInitialized)
		return nil
	}

	password, err := ui.ReadSecretWithConfirmation(
		constants.MsgEnterMasterPassword,
		constants.MsgConfirmMasterPassword,
	)
	if err != nil {
		return err
	}

	// Generate random salt for generating key
	salt := crypto.GenerateSalt()

	// Derive unlock key
	key := crypto.DeriveKey([]byte(password), salt)

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
	err = ctx.Store.InitializeVault(*vault.EncodeToString())
	if err != nil {
		ui.Error("%s", constants.ErrFailedToInitializeVault.Error())
	} else {
		ui.Info(constants.MsgVaultInitialized)
	}
	return err
}
