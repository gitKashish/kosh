package cmd

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/term"
)

type Credential struct {
	Group        string `json:"group"`
	User         string `json:"user"`
	Secret       string `json:"secret"`
	EphemeralPub string `json:"ephemeral_pub"`
	Nonce        string `json:"nonce"`
}

func init() {
	Commands["add"] = AddCmd
}

func AddCmd(args ...string) {
	// Load vault info
	home, _ := os.UserHomeDir()
	vaultPath := filepath.Join(home, ".kosh", "vault.json")
	data, err := os.ReadFile(vaultPath)
	if err != nil {
		fmt.Println("[Error] vault not found, run `kosh init` first")
		return
	}

	var vault Vault
	json.Unmarshal(data, &vault)

	fmt.Print("Enter master password: ")
	password, _ := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()

	// Verify master password and get encryption info
	salt, _ := base64.StdEncoding.DecodeString(vault.Salt)
	unlockKey := deriveKey(password, salt)

	if _, err := decryptPrivateKey(unlockKey, vault.PrivateKey); err != nil {
		fmt.Println("[Error] master password is incorrect")
		os.Exit(126)
	}

	// Get credential details
	var title, username, secret string
	fmt.Print("Title: ")
	fmt.Scanln(&title)
	fmt.Print("Username: ")
	fmt.Scanln(&username)
	fmt.Print("Secret: ")
	secret1, _ := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Print("Confirm Secret: ")
	secret2, _ := term.ReadPassword(int(os.Stdin.Fd()))

	if string(secret1) != string(secret2) {
		fmt.Println("[Error] entered secrets do not match")
		return
	}

	secret = string(secret1)

	// Encrypt credential with public key
	// Derive shared secrete from vault public key + ephemeral private key
	var ephPriv [32]byte
	rand.Read(ephPriv[:])
	ephPub, _ := curve25519.X25519(ephPriv[:], curve25519.Basepoint)
	sharedSecret, _ := curve25519.X25519(ephPriv[:], []byte(vault.PublicKey))
	key := sha256.Sum256(sharedSecret)

	aead, err := chacha20poly1305.NewX(key[:])
	if err != nil {
		fmt.Printf("[Error] %s", err)
		panic(err)
	}
	nonce := make([]byte, aead.NonceSize())
	rand.Read(nonce)
	cipher := aead.Seal(nil, nonce, []byte(secret), nil)

	credential := Credential{
		Group:        title,
		User:         username,
		Nonce:        base64.StdEncoding.EncodeToString(nonce),
		Secret:       base64.StdEncoding.EncodeToString(cipher),
		EphemeralPub: base64.RawStdEncoding.EncodeToString(ephPub),
	}

	// Save credential
	if err := saveCredential(&credential); err != nil {
		fmt.Println("[Error] unable to save credential")
		fmt.Printf("[DEBUG] error: %s", err)
	}
}

func deriveKey(password, salt []byte) []byte {
	return argon2.IDKey(password, salt, 1, 64*1024, 4, 32)
}

func decryptPrivateKey(unlockKey []byte, privKeyData struct {
	Nonce  string `json:"nonce"`
	Cipher string `json:"cipher"`
}) ([]byte, error) {
	nonce, _ := base64.StdEncoding.DecodeString(privKeyData.Nonce)
	cipher, _ := base64.StdEncoding.DecodeString(privKeyData.Cipher)

	aead, _ := chacha20poly1305.NewX(unlockKey)
	return aead.Open(nil, nonce, cipher, nil)
}

func saveCredential(credential *Credential) error {
	// Ensure that `creds` directory exists
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".kosh", "creds")
	os.MkdirAll(dir, 0700)

	// Get existing group file (if it exists)
	groupFile := filepath.Join(dir, fmt.Sprintf("%s.json", credential.Group))

	var credentials []Credential
	if _, err := os.Stat(groupFile); err == nil {
		existing, _ := os.ReadFile(groupFile)
		json.Unmarshal(existing, &credentials)

		// Check if creds for the username already exist
		for idx, cred := range credentials {
			if cred.User == credential.User {
				var overrideInput string
				fmt.Printf("[Info] credentials for %s alread exist\n", cred.User)
				fmt.Print("Overwrite the credentials? (y/N): ")
				fmt.Scanln(&overrideInput)
				if strings.ToLower(overrideInput) == "y" {
					credentials = slices.Delete(credentials, idx, idx+1)
				} else {
					return nil
				}
				break
			}
		}
	}

	// Append new credential
	credentials = append(credentials, *credential)

	// Save back to file
	credData, _ := json.MarshalIndent(credentials, "", "  ")
	os.WriteFile(groupFile, credData, 0600)
	return nil
}
