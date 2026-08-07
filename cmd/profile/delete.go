package profile

import (
	"fmt"
	"log/slog"
	"strings"

	"git.plutolab.org/plutolab/kosh/internal/app"
	"git.plutolab.org/plutolab/kosh/internal/config"
	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/core"
	"git.plutolab.org/plutolab/kosh/internal/model"
	"git.plutolab.org/plutolab/kosh/internal/storage"
	"git.plutolab.org/plutolab/kosh/internal/ui"
	"github.com/spf13/cobra"
)

func NewCmdDelete(ctx *app.Context) *cobra.Command {
	deleteCmd := &cobra.Command{
		Use:   "delete <profile>",
		Short: "Permanently delete a profile and its credentials",
		Long: `Delete a profile along with the vault file holding its credentials.

The active profile cannot be deleted; switch elsewhere with "kosh use" first.
You are asked for the target profile's own master password to prove ownership,
and then have to type out the confirmation phrase shown on screen.

Deletion is permanent. The vault file is overwritten on disk before it is
removed, so every credential in that profile is gone for good - there is no undo
and nothing to restore from.`,

		Example: `  Delete a profile:
    kosh profile delete work

  Switch away first if it is the active profile:
    kosh use default
    kosh profile delete work`,

		Args: cobra.ExactArgs(1),

		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(cmd, ctx, args[0])
		},
	}
	return deleteCmd
}

func runDelete(_ *cobra.Command, ctx *app.Context, profileName string) error {
	// Check if profile exists, and adopt the spelling it is stored under. Every
	// step below opens or removes the file, so they must all agree on the name.
	stored, exists, err := ctx.Profile.ResolveProfile(profileName)
	if err != nil {
		slog.Debug("failed to check profile existence", "error", err)
		return err
	}
	if !exists {
		return constants.ErrProfileDoesNotExist
	}
	profileName = string(stored)

	// Check if profile to be deleted is the current profile
	// User cannot delete current profile
	if strings.EqualFold(profileName, ctx.Config.ActiveProfile) {
		return constants.ErrCannotDeleteActiveProfile
	}

	if err := verifyProfileOwnership(profileName); err != nil {
		return err
	}

	// Give Caution and Get text confirmation
	ui.Caution(constants.MsgCautionDestructive, "%s", constants.MsgWarnProfileDelete)

	confirm, err := ui.ConfirmWithText(
		constants.MsgConfirmProfileDelete,
		fmt.Sprintf("permanently delete %s with credentials", profileName),
	)
	if err != nil {
		slog.Debug("failed to get confirmation text", "error", err)
		return err
	}

	// Abort cancellation
	if !confirm {
		ui.Info(constants.MsgOperationAborted)
		return nil
	}

	if err := ctx.Profile.DeleteProfile(model.Profile(profileName)); err != nil {
		return err
	}

	ui.Info(constants.MsgProfileDeleted)
	return nil
}

func verifyProfileOwnership(profile string) error {
	// Everything printed here concerns the profile being deleted, not the
	// active one, so label it accordingly and restore the prefix on the way out.
	defer ui.WithProfile(profile)()

	store, err := storage.InitializeStore(&config.Config{ActiveProfile: profile})
	if err != nil {
		slog.Debug("failed to initialize profile storage to be deleted", "error", err)
		return err
	}
	defer store.CloseStore()

	initialized, err := store.IsVaultInitialized()
	if err != nil {
		slog.Debug("unable to check vault initialization status", "error", err)
		return err
	}

	// Delete un-initialized profiles without password
	if !initialized {
		ui.Info("%s", constants.ErrVaultNotInitialized.Error())
		slog.Debug("profile not initialized, no need to verify ownership")
		return nil
	}

	password, err := ui.ReadSecretField(constants.MsgEnterMasterPassword)
	if err != nil {
		slog.Debug("failed to read master password", "error", err)
		return err
	}

	service := core.NewVaultService(store)
	if err := service.VerifyMasterPassword(password); err != nil {
		slog.Debug("master password could not be verified", "error", err)
		return constants.ErrIncorrectMasterPassword
	}
	return nil
}
