package cmd

import (
	"crypto/sha256"
	"database/sql"
	"fmt"

	"github.com/gitKashish/kosh/src/internals/crypto"
	"github.com/gitKashish/kosh/src/internals/dao"
	"github.com/gitKashish/kosh/src/internals/interaction"
	"github.com/gitKashish/kosh/src/internals/logger"
	"github.com/gitKashish/kosh/src/internals/model"
	"golang.org/x/crypto/curve25519"
)

func init() {
	Commands["get"] = CommandInfo{
		Exec:        GetCmd,
		Description: "Retrieve a stored credential",
		Usage:       "kosh get <label> <user>",
	}
}

func GetCmd(args ...string) error {
	if len(args) < 2 {
		logger.Error("missing arguments")
		HelpCmd()
		return fmt.Errorf("missing arguments, got %d, want %d", len(args), 2)
	}

	desiredGroup := args[0]
	desiredUser := args[1]

	// fetch vault info
	vault, err := dao.GetVaultInfo()
	if err != nil {
		return err
	}
	vaultData := vault.GetRawData()

	// fetch credential info
	credential, err := dao.GetCredentialByLabelAndUser(desiredGroup, desiredUser)
	if credential == nil && err == sql.ErrNoRows {
		// credential does not exist
		logger.Error("no matching credential found")
		return nil
	}

	if err != nil {
		return err
	}

	// get password from user
	password, err := interaction.ReadSecretField("master password > ")
	if err != nil {
		logger.Error("unable to read password")
		return err
	}

	secret, err := extractSecret(credential.GetRawData(), vaultData, []byte(password))
	if err != nil {
		return err
	}
	interaction.CopyToClipboard(secret)
	logger.Info("copied secret to clipboard")
	return nil
}

func extractSecret(credential *model.CredentialData, vault *model.VaultData, masterPassword []byte) ([]byte, error) {
	// decrypt private key using master password
	unlockKey := crypto.GenerateSymmetricKey(masterPassword, vault.Salt)
	privateKey, err := crypto.DecryptSecret(unlockKey, vault.Secret, vault.Nonce)
	if err != nil {
		logger.Error("master password is incorrect")
		return nil, err
	}

	// decrypt credential using private key
	sharedSecret, err := curve25519.X25519(privateKey, credential.Ephemeral)
	if err != nil {
		logger.Error("failed to decrypt credential")
		return nil, err
	}

	key := sha256.Sum256(sharedSecret)

	return crypto.DecryptSecret(key[:], credential.Secret, credential.Nonce)
}
