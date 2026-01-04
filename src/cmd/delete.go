package cmd

import (
	"database/sql"
	"fmt"
	"strconv"

	"git.plutolab.org/plutolab/kosh/src/internals/crypto"
	"git.plutolab.org/plutolab/kosh/src/internals/dao"
	"git.plutolab.org/plutolab/kosh/src/internals/interaction"
	"git.plutolab.org/plutolab/kosh/src/internals/logger"
)

func init() {
	Commands["delete"] = CommandInfo{
		Exec:        DeleteCmd,
		Usage:       "kosh delete <credential_id>",
		Description: "Delete a stored credential.",
	}
}

func DeleteCmd(args ...string) error {
	vault, err := dao.GetVaultInfo()
	if err != nil {
		logger.Error("error fetching vault info")
		return err
	}
	vaultData := vault.GetRawData()

	if len(args) != 1 {
		logger.Error("missing argument")
		HelpCmd()
		return fmt.Errorf("missing argument got %d, want 1", len(args))
	}

	delete_id, err := strconv.Atoi(args[0])
	if err != nil {
		logger.Error("delete id must be a number")
		return err
	}

	password, err := interaction.ReadSecretField("master password > ")
	if err != nil {
		logger.Error("cannot read password")
		return err
	}
	// verify master password and get encryption info
	unlockKey := crypto.GenerateSymmetricKey([]byte(password), vaultData.Salt)

	if _, err := crypto.DecryptSecret(unlockKey, vaultData.Secret, vaultData.Nonce); err != nil {
		logger.Error("master password is incorrect")
		return err
	}

	// check credential existence
	credential, err := dao.GetCredentialById(delete_id)
	if credential == nil && err == sql.ErrNoRows {
		// credential does not exist
		logger.Error("no matching credential found")
		return nil
	}

	if err != nil {
		return err
	}

	// get deletion confirmation
	confirm, err := interaction.ConfirmWithText(
		"delete operation is permanent and cannot be undone. are you sure?",
		fmt.Sprintf("delete %s %s", credential.Label, credential.User),
	)
	if err != nil {
		logger.Error("error confirming with text prompt")
		return err
	}

	if !confirm {
		logger.Info("operation aborted")
		return nil
	}

	err = dao.DeleteCredentialById(delete_id)
	if err != nil {
		logger.Error("unable to delete credential")
	} else {
		logger.Info("credential deleted permanently")
	}
	return err
}
