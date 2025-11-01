package cmd

import (
	"crypto/sha256"
	"fmt"

	"golang.org/x/crypto/curve25519"

	"github.com/gitKashish/kosh/src/internals/crypto"
	"github.com/gitKashish/kosh/src/internals/dao"
	"github.com/gitKashish/kosh/src/internals/interaction"
	"github.com/gitKashish/kosh/src/internals/model"
)

func init() {
	Commands["add"] = CommandInfo{
		Exec:        AddCmd,
		Description: "Add a new credential",
		Usage:       "kosh add",
	}
}

func AddCmd(args ...string) error {
	// load vault info
	vault, err := dao.GetVaultInfo()
	if err != nil {
		fmt.Println("[Error] error fetching vault info")
		return nil
	}
	vaultData := vault.GetRawData()

	// get master password
	password, err := interaction.ReadSecretField("master password > ")
	if err != nil {
		fmt.Println("[Error] cannot read password")
		return nil
	}

	// verify master password and get encryption info
	unlockKey := crypto.GenerateSymmetricKey([]byte(password), vaultData.Salt)

	if _, err := crypto.DecryptPrivateKey(unlockKey, vaultData.Secret, vaultData.Nonce); err != nil {
		fmt.Println("[Error] master password is incorrect")
		return err
	}

	// get credential details
	label := interaction.ReadStringField("enter label > ")
	username := interaction.ReadStringField("enter username > ")
	secret, err := interaction.ReadSecretField("enter secret > ")
	if err != nil {
		fmt.Println("[Error] cannot read secret")
		return err
	}
	confirm, err := interaction.ReadSecretField("confirm secret > ")
	if err != nil {
		fmt.Println("[Error] cannot read confirmation")
		return err
	}

	if secret != confirm {
		fmt.Println("[Error] entered secrets do not match")
		return nil
	}

	ephemeralPrivateKey, ephemeralPublicKey := crypto.GenerateAsymmetricKeyPair()

	// generate symmetric shared secret
	encryptionKey, _ := curve25519.X25519(ephemeralPrivateKey, vaultData.PublicKey)

	// hash to get 32 bit consistent key for encryption
	key := sha256.Sum256(encryptionKey)

	cipher, nonce := crypto.EncryptSecret(key[:], []byte(secret))

	credential := model.CredentialData{
		Label:     label,
		User:      username,
		Nonce:     nonce,
		Secret:    cipher,
		Ephemeral: ephemeralPublicKey,
	}

	// save credential
	err = dao.AddCredential(credential.EncodeToString())
	if err != nil {
		fmt.Println("[Error] unable to save credential")
	}
	return err
}
