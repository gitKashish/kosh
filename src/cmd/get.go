package cmd

import (
	"crypto/sha256"
	"fmt"

	"github.com/gitKashish/kosh/src/internals/crypto"
	"github.com/gitKashish/kosh/src/internals/dao"
	"github.com/gitKashish/kosh/src/internals/interaction"
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
		fmt.Println("[Error] missing arguments")
		HelpCmd()
		return fmt.Errorf("missing arguments, got %d, want %d", len(args), 2)
	}

	desiredGroup := args[0]
	desiredUser := args[1]

	// fetch vault info
	vault, err := dao.GetVaultInfo()
	if err != nil {
		fmt.Println("[Error] unable to get vault info")
		return err
	}
	vaultData := vault.GetRawData()

	// fetch credential info
	credential, err := dao.GetCredentialByLabelAndUser(desiredGroup, desiredUser)
	if err != nil {
		return err
	}

	if credential == nil {
		// credential does not exist
		fmt.Println("[Error] credential does not exist")
		return nil
	}

	// get password from user
	password, err := interaction.ReadSecretField("master password > ")
	if err != nil {
		fmt.Println("[Error] unable to read password")
		return err
	}

	secret, err := extractSecret(credential.GetRawData(), vaultData, []byte(password))
	if err != nil {
		return err
	}
	interaction.CopyToClipboard(secret)
	fmt.Println("[Info] copied secret to clipboard")
	return nil
}

func extractSecret(credential *model.CredentialData, vault *model.VaultData, masterPassword []byte) ([]byte, error) {
	// decrypt private key using master password
	unlockKey := crypto.GenerateSymmetricKey(masterPassword, vault.Salt)
	privateKey, err := crypto.DecryptPrivateKey(unlockKey, vault.Secret, vault.Nonce)
	if err != nil {
		fmt.Println("[Error] master password is incorrect")
		return nil, err
	}

	// decrypt credential using private key
	sharedSecret, err := curve25519.X25519(privateKey, credential.Ephemeral)
	if err != nil {
		fmt.Println("[Error] failed to decrypt credential")
		return nil, err
	}

	key := sha256.Sum256(sharedSecret)

	return crypto.DecryptSecret(key[:], credential.Secret, credential.Nonce)
}
