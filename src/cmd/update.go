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
	Commands["Cmd_Update_Dev"] = CommandInfo{
		Exec:        UpdateCmd,
		Description: "update existing credential",
		Usage:       "kosh update <id>",
	}
}

func UpdateCmd(args ...string) error {
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
		logger.Error("update id must be a number")
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

	return nil
}
