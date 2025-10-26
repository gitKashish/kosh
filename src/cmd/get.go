package cmd

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.design/x/clipboard"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/term"
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

	home, _ := os.UserHomeDir()

	// Check if current vault exists, if not prompt to create one.
	vaultPath := filepath.Join(home, ".kosh", "vault.json")
	data, err := os.ReadFile(vaultPath)
	if err != nil {
		fmt.Println("[Error] vault not found, run `kosh init` first")
		return
	}

	// Unmarshal vault data
	var vault Vault
	json.Unmarshal(data, &vault)

	// Check if requested group and user exist or not
	credPath := filepath.Join(home, ".kosh", "creds")
	groups, err := os.ReadDir(credPath)
	if err != nil {
		fmt.Println("[Error] unable to retrieve creds at ", credPath)
		return
	}

	var found bool
	for _, group := range groups {
		if group.IsDir() {
			continue
		}

		groupName := strings.TrimSuffix(group.Name(), filepath.Ext(group.Name()))
		if groupName == desiredGroup {
			data, err := os.ReadFile(filepath.Join(credPath, group.Name()))
			if err != nil {
				fmt.Println("[Error] unable to read credential group")
				return
			}

			// unmarshal JSON
			var credentials []Credential
			if err := json.Unmarshal(data, &credentials); err != nil {
				fmt.Println("[Error] failed to unmarshall credentials")
				return
			}

			for _, credential := range credentials {
				if credential.User == desiredUser {
					// ask master password
					fmt.Print("Enter master password: ")
					password, _ := term.ReadPassword(int(os.Stdin.Fd()))
					fmt.Println()

					// get credential from master password
					passkey, err := retrieveCredential(credential, vault, password)
					if err != nil {
						fmt.Println("[Error] ", err.Error())
						os.Exit(1)
					} else {
						copyToClipboard(passkey)
						found = true
						fmt.Println("[Info] credential copied to clipboard.")
						break
					}
				}
			}
			if !found {
				break
			}
		}
	}

	if !found {
		fmt.Println("[Info] no matching credential found")
	}
	os.Exit(0)
}

func retrieveCredential(credential Credential, vault Vault, masterPassword []byte) (string, error) {
	// decrypt private key using master password
	salt, _ := base64.StdEncoding.DecodeString(vault.Salt)
	unlockKey := deriveKey(masterPassword, salt)
	privateKey, err := decryptPrivateKey(unlockKey, vault.PrivateKey)
	if err != nil {
		fmt.Println("[Error] master password is incorrect")
		os.Exit(126)
	}

	// decrypt credential using private key
	ephPub, _ := base64.StdEncoding.DecodeString(credential.EphemeralPub)
	sharedSecret, err := curve25519.X25519(privateKey, ephPub)
	if err != nil {
		fmt.Println("[Error] failed to decrypt credential")
		return "", err
	}

	key := sha256.Sum256(sharedSecret)

	aead, err := chacha20poly1305.NewX(key[:])
	if err != nil {
		fmt.Println("[Error] failed to create AEAD")
		return "", err
	}

	nonce, _ := base64.StdEncoding.DecodeString(credential.Nonce)
	if len(nonce) != aead.NonceSize() {
		fmt.Println("[Error] invalid credential nonce")
		return "", err
	}

	cipher, _ := base64.StdEncoding.DecodeString(credential.Secret)
	passkey, err := aead.Open(nil, nonce, cipher, nil)
	if err != nil {
		fmt.Println("[Error] failed to open aead")
		return "", err
	}

	return string(passkey), nil
}

func copyToClipboard(content string) {
	err := clipboard.Init()
	if err != nil {
		panic(err)
	}

	clipboard.Write(clipboard.FmtText, []byte(content))
}
