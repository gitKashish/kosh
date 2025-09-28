package cmd

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/term"
)

type Vault struct {
	Salt       string `json:"salt"`
	PublicKey  string `json:"public_key"`
	PrivateKey struct {
		Nonce  string `json:"nonce"`
		Cipher string `json:"cipher"`
	} `json:"private_key"`
}

func init() {
	Commands["init"] = InitCmd
}

// InitCmd sets up the vault, generates crypto information based on user's provided
// master password
func InitCmd(args ...string) {
	// TODO: Check if kosh has already been initialized. If not, carry on.

	password, err := getPassword()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}

	// Generate random salt for generating key
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		panic(err)
	}

	// Derive unlock key using Argon2id
	key := argon2.IDKey(password, salt, 1, 64*1024, 4, 32)

	// Generate ECC key pair
	var priv [32]byte
	if _, err := rand.Read(priv[:]); err != nil {
		panic(err)
	}

	pub, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		panic(err)
	}

	// Encrypt private key with AES-GCM
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		panic(err)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce[:]); err != nil {
		panic(err)
	}

	cipher := aead.Seal(nil, nonce, priv[:], nil)

	// Build vault structure
	vault := Vault{
		Salt:      base64.StdEncoding.EncodeToString(salt),
		PublicKey: base64.StdEncoding.EncodeToString(pub),
	}

	vault.PrivateKey.Nonce = base64.StdEncoding.EncodeToString(nonce)
	vault.PrivateKey.Cipher = base64.StdEncoding.EncodeToString(cipher)

	// Save to config
	saveVault(&vault)
}

// getPassword gets password value from user from terminal. It gets password using silent text input and asks for
// password confirmation by re-entering the password. It throws an error if both passwords do not match.
func getPassword() ([]byte, error) {
	// Ask user for setting up a master password.
	fmt.Print("Enter a master password: ")
	password, _ := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()

	// Confirm entered password
	fmt.Printf("Re-enter the master password: ")
	confirm, _ := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()

	if string(password) != string(confirm) {
		err := fmt.Errorf("[Error] passwords do not match")
		return nil, err
	}

	return password, nil
}

// saveVault takes a vault object and writes it at `~/.kosh/vault.json`. It creates a new file, if one does not already
// exist and overwrites exsiting `vault.json` file.
func saveVault(vault *Vault) error {
	// Ensure that `.kosh` directory exists
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".kosh")
	os.MkdirAll(dir, 0700)

	// Save vault info to `vault.json`
	data, _ := json.MarshalIndent(vault, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "vault.json"), data, 0600); err != nil {
		panic(err)
	}

	fmt.Printf("[Info] vault initialized at %s\n", filepath.Join(dir, "vault.json"))
	return nil
}
