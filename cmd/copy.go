package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"git.plutolab.org/plutolab/kosh/internal/app"
	"git.plutolab.org/plutolab/kosh/internal/config"
	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/core"
	"git.plutolab.org/plutolab/kosh/internal/storage"
	"git.plutolab.org/plutolab/kosh/internal/ui"
	"github.com/spf13/cobra"
)

func NewCmdCopy(ctx *app.Context) *cobra.Command {
	moveCmd := &cobra.Command{
		Use:   "copy <id> <profile>",
		Short: "Copy a credential into another profile",
		Long: `Copy a credential from the active profile into another profile's vault.

Run "kosh list" to find the ID and "kosh profile list" to see the available
profiles. The secret is decrypted with the active profile's master password and
re-encrypted for the target profile, which therefore has to be initialized
already. The original credential is left untouched.

A credential is identified by its label and user together, so copying only
collides when the target profile already holds that same pair. When it does, you
must type out the confirmation phrase shown on screen before the existing secret
is overwritten, and the overwritten value is gone for good.`,

		Example: `  Copy the credential with ID 3 into the work profile:
    kosh copy 3 work

  Look up the ID first:
    kosh list -l github
    kosh copy 7 personal`,

		Args: cobra.ExactArgs(2), // For Future: Add interactive credential and profile selection

		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return constants.ErrIdMustBeInteger
			}
			targetProfile := args[1]

			return runCopy(cmd, ctx, id, targetProfile)
		},
	}
	return moveCmd
}

func runCopy(_ *cobra.Command, ctx *app.Context, id int, targetProfile string) error {
	// Verify target profile, and adopt the spelling it is stored under -
	// copyCredential opens the vault by that name further down.
	stored, ok, err := ctx.Profile.ResolveProfile(targetProfile)
	if err != nil {
		slog.Debug("failed to check for profile", "error", err)
		return constants.ErrFailedToFetchProfile
	}
	if !ok {
		return constants.ErrProfileDoesNotExist
	}
	targetProfile = string(stored)

	if strings.EqualFold(targetProfile, ctx.Config.ActiveProfile) {
		return constants.ErrCannotCopyToActiveProfile
	}

	// Verify credential ID
	credential, err := ctx.Store.GetCredentialById(id)
	if err == constants.ErrCredentialNotFound {
		return err
	} else if err != nil {
		slog.Debug("failed to get credential by id", "error", err)
		return constants.ErrFailedToFetchCredential
	}

	// decrypt credential
	password, err := ui.ReadSecretField(constants.MsgEnterMasterPassword)
	if err != nil {
		slog.Debug("failed to read master password", "error", err)
		return constants.ErrFailedToReadInput
	}

	secret, err := ctx.Vault.DecryptCredential(credential, password)
	if err != nil {
		slog.Debug("failed to decrypt credential", "error", err)
		return constants.ErrIncorrectMasterPassword
	}

	if err := copyCredential(credential.Label, credential.User, secret, targetProfile); err != nil {
		if errors.Is(err, constants.ErrOperationAborted) {
			ui.Info(constants.MsgOperationAborted)
			return nil
		}
		return err
	}

	ui.Info(constants.MsgCredentialCopied)
	return nil
}

func copyCredential(label, user string, secret []byte, targetProfile string) error {
	// Initialize target profile store
	store, err := storage.InitializeStore(&config.Config{ActiveProfile: targetProfile})
	if err != nil {
		slog.Debug("failed to setup target profile storage", "error", err)
		return constants.ErrFailedToInitializeStore
	}
	defer store.CloseStore()

	// Verify that vault is intialized
	initialized, err := store.IsVaultInitialized()
	if err != nil {
		slog.Debug("failed to check store initialization", "error", err)
		return constants.ErrFailedToInitializeVault
	}
	if !initialized {
		return constants.ErrTargetVaultNotInitialized
	}

	// Check if credential with same label and user already exists
	// If exists, confirm for overwrite
	credential, err := store.GetCredentialByLabelAndUser(label, user)
	if err != nil && err != constants.ErrCredentialNotFound {
		slog.Debug("failed to get credential by label and user", "error", err)
		return constants.ErrFailedToFetchCredential
	}
	if credential != nil {
		// confirm for over-write
		ui.Caution(constants.MsgCautionOverwrite, "%s", constants.MsgWarnTargetCredentialOverwrite)

		confirm, err := ui.ConfirmWithText(
			constants.MsgConfirmCredentialOverwrite,
			fmt.Sprintf(
				"overwrite %s %s in %s",
				label, user, targetProfile,
			),
		)
		if err != nil {
			slog.Debug("failed to get overwrite confirmation", "error", err)
			return constants.ErrFailedToReadInput
		}
		if !confirm {
			return constants.ErrOperationAborted
		}
	}

	// Upsert credential in target vault
	vault := core.NewVaultService(store)

	if err := vault.AddCredential(label, user, secret); err != nil {
		slog.Debug("failed to add credential to the target store", "error", err)
		return constants.ErrFailedToSaveCredential
	}
	return nil
}
