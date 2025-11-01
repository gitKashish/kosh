package cmd

import (
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/gitKashish/kosh/src/internals/crypto"
	"github.com/gitKashish/kosh/src/internals/dao"
	"github.com/gitKashish/kosh/src/internals/interaction"
	"github.com/gitKashish/kosh/src/internals/model"
	"golang.org/x/crypto/curve25519"
)

func init() {
	Commands["get"] = GetCommand
}

func GetCommand(args ...string) {
	if len(args) < 2 {
		Help()
		os.Exit(2)
	}

	desiredGroup := args[0]
	desiredUser := args[1]

	// fetch vault info
	vault, err := dao.GetVaultInfo()
	if err != nil {
		fmt.Println("[Error] unable to get vault info")
		return
	}
	vaultData := vault.GetRawData()

	// fetch credential info
	credential := dao.GetCredentialByLabelAndUser(desiredGroup, desiredUser)
	if credential == nil {
		// credential does not exist
		fmt.Println("[Error] credential does not exist")
		return
	}

	// get password from user
	password, err := interaction.ReadSecretField("master password > ")
	if err != nil {
		fmt.Println("[Error] unable to read password")
		fmt.Printf("[Debug] %s\n", err.Error())
	}

	secret, err := extractSecret(credential.GetRawData(), vaultData, []byte(password))
	if err != nil {
		fmt.Printf("[Debug] %s\n", err.Error())
		return
	}
	interaction.CopyToClipboard(secret)
	fmt.Println("[Info] copied secret to clipboard")

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
