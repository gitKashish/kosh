package profile

import (
	"fmt"
	"log/slog"

	"git.plutolab.org/plutolab/kosh/internal/app"
	"git.plutolab.org/plutolab/kosh/internal/config"
	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/core"
	"git.plutolab.org/plutolab/kosh/internal/crypto"
	"git.plutolab.org/plutolab/kosh/internal/model"
	"git.plutolab.org/plutolab/kosh/internal/storage"
	"git.plutolab.org/plutolab/kosh/internal/ui"
	"github.com/spf13/cobra"
)

func NewCmdCreate(ctx *app.Context) *cobra.Command {
	createCmd := &cobra.Command{
		Use:   "create <profile>",
		Short: "Create a profile and initialize its vault",
		Long: `Create a new profile, switch to it and set up its vault.

The name must not already be taken. Once created the profile becomes the active
one, and you are prompted twice for the master password that will protect it -
every profile has its own, independent of the others.

The password is never stored: it is combined with a random salt to derive the
key that encrypts the new vault's private key, and re-derived on every unlock.
Losing it permanently locks that profile's vault, with no recovery mechanism.

The vault is ready to use straight away, so there is no need to run "kosh init"
afterwards.`,

		Example: `  Create a profile named work:
    kosh profile create work

  Add a credential to the new profile:
    kosh profile create work
    kosh add`,

		Args: cobra.ExactArgs(1),

		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd, ctx, args[0])
		},
	}
	return createCmd
}

func runCreate(_ *cobra.Command, ctx *app.Context, name string) error {
	clean, err := core.SanitizeProfileName(name)
	if err != nil {
		slog.Debug("un-sanitable profile name", "name", name, "error", err)
		return err
	}
	name = clean

	// Verify that the profile name is not already taken. This ignores case, so
	// "work" is taken by an existing "Work" even where the filesystem would
	// happily keep both - the two would collide on the next machine.
	_, exists, err := ctx.Profile.ResolveProfile(name)
	if err != nil {
		slog.Debug(
			"failed to load profiles matching new name",
			"name", name,
			"error", err,
		)
		return err
	}
	if exists {
		return constants.ErrProfileAlreadyExists
	}

	// The prompts below concern the profile being created, not the active one.
	defer ui.WithProfile(name)()

	// Build the vault before touching any persistent state.
	if err := initVault(name); err != nil {
		slog.Debug("failed to initialize vault", "error", err)
		rollbackCreate(ctx, name)
		return constants.ErrFailedToInitializeVault
	}

	// The vault exists now, so switching to it cannot strand the config on a
	// profile that was never finished.
	if err := ctx.Profile.SwitchProfile(model.Profile(name)); err != nil {
		slog.Debug("failed to switch to the newly created profile", "error", err)
		// The vault is complete and staying: only the switch failed, and
		// "kosh use" finishes the job.
		return fmt.Errorf("%w: profile %q was created but could not be activated", err, name)
	}

	// Update the current App Context Config.
	ctx.Config.ActiveProfile = name

	ui.Info(constants.MsgProfileCreated)
	return nil
}

// rollbackCreate undoes a partially completed profile creation by removing the
// vault file initVault may have created before failing.
func rollbackCreate(ctx *app.Context, name string) {
	// initVault creates the vault file before writing to it, so a failure can
	// leave an empty one behind.
	if stored, exists, err := ctx.Profile.ResolveProfile(name); err != nil {
		slog.Debug("failed to check partially created profile", "name", name, "error", err)
	} else if exists {
		if err := ctx.Profile.DeleteProfile(stored); err != nil {
			slog.Debug("failed to remove partially created profile", "name", stored, "error", err)
		}
	}
}

// initVault builds the vault for profile. It takes the profile name rather than
// the app context because it runs before the profile becomes the active one.
func initVault(profile string) error {
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

	// Initialize new vault service
	store, err := storage.InitializeStore(&config.Config{ActiveProfile: profile})
	if err != nil {
		slog.Debug("failed to create new profile store", "error", err)
		return constants.ErrFailedToInitializeStore
	}
	defer store.CloseStore()
	return store.InitializeVault(*vault.EncodeToString())
}
