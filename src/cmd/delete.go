package cmd

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/gitKashish/kosh/src/internals/crypto"
	"github.com/gitKashish/kosh/src/internals/dao"
	"github.com/gitKashish/kosh/src/internals/interaction"
	"github.com/gitKashish/kosh/src/internals/logger"
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
	logger.Warn("Are you sure you want to delete the following credential? This action is permanent and cannot be reverted.")
	logger.Info("id: %d", credential.Id)
	logger.Info("label: %s", credential.Label)
	logger.Info("user: %s", credential.User)

	confirmation_key := fmt.Sprintf("delete %s %s", credential.Label, credential.User)
	confirmation_text := interaction.ReadStringField(fmt.Sprintf("enter `%s` to confirm or anything else to cancel > ", confirmation_key))
	if confirmation_text != confirmation_key {
		logger.Warn("confirmation key does not match")
		logger.Info("deletion operation cancelled")
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
