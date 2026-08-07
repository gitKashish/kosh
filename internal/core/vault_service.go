package core

import (
	"crypto/sha256"
	"log/slog"

	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/crypto"
	"git.plutolab.org/plutolab/kosh/internal/model"
	"git.plutolab.org/plutolab/kosh/internal/storage"
	"golang.org/x/crypto/curve25519"
)

type KoshVault struct {
	store storage.Store
}

type VaultService interface {
	VerifyMasterPassword([]byte) error
	AddCredential(string, string, []byte) error
	DecryptCredential(*model.Credential, []byte) ([]byte, error)
	UpdateCredentialSecret(int, []byte) error
	ListCredentials(string, string) ([]model.CredentialSummary, error)
}

// NewVaultService creates a new service instance
func NewVaultService(store storage.Store) *KoshVault {
	return &KoshVault{store}
}

// verifyMasterPassword checks if the provided master password can unlock the vault.
// It returns an error if the password is incorrect or if the vault cannot be read.
func (s *KoshVault) VerifyMasterPassword(password []byte) error {
	vault, err := s.store.GetVaultInfo()
	if err != nil {
		return constants.ErrFailedToFetchVaultInfo
	}
	vaultData := vault.GetRawData()

	unlockKey := crypto.DeriveKey(password, vaultData.Salt)
	if _, err := crypto.DecryptSecret(unlockKey, vaultData.Secret, vaultData.Nonce); err != nil {
		return constants.ErrIncorrectMasterPassword
	}

	return nil
}

func (s *KoshVault) AddCredential(label, user string, secret []byte) error {
	vaultInfo, err := s.store.GetVaultInfo()
	if err != nil {
		return err
	}

	vaultData := vaultInfo.GetRawData()
	ephemeralPrivateKey, ephemeralPublicKey := crypto.GenerateAsymmetricKeyPair()

	// generate symmetric shared secret
	encryptionKey, _ := curve25519.X25519(ephemeralPrivateKey, vaultData.PublicKey)

	// hash to get 32 bit consistent key for encryption
	key := sha256.Sum256(encryptionKey)

	cipher, nonce, err := crypto.EncryptSecret(key[:], secret)
	if err != nil {
		return err
	}

	credential := model.CredentialData{
		Label:     label,
		User:      user,
		Nonce:     nonce,
		Secret:    cipher,
		Ephemeral: ephemeralPublicKey,
	}

	// save credential
	err = s.store.AddCredential(credential.EncodeToString())
	if err != nil {
		return constants.ErrFailedToSaveCredential
	}

	return nil
}

func (s *KoshVault) DecryptCredential(credential *model.Credential, password []byte) ([]byte, error) {
	vaultInfo, err := s.store.GetVaultInfo()
	if err != nil {
		return nil, err
	}
	vaultData := vaultInfo.GetRawData()

	// Derive unlock key
	unlockKey := crypto.DeriveKey(password, vaultData.Salt)

	// Decrypt vault private key
	vaultPrivateKey, err := crypto.DecryptSecret(unlockKey, vaultData.Secret, vaultData.Nonce)
	if err != nil {
		// the cause is replaced below, so this is the only record of it
		slog.Debug("failed to unwrap vault private key", "error", err)
		return nil, constants.ErrFailedToDecryptCredential
	}

	// Generate shared secret
	credData := credential.GetRawData()
	decryptionKey, _ := curve25519.X25519(vaultPrivateKey, credData.Ephemeral)

	// Hash to get 32-bit consistent key
	key := sha256.Sum256(decryptionKey)

	plainText, err := crypto.DecryptSecret(key[:], credData.Secret, credData.Nonce)
	if err != nil {
		return nil, constants.ErrFailedToDecryptCredential
	}

	return plainText, nil
}

// UpdateCredentialSecret encrypts a new secret for an existing credential and saves it.
func (s *KoshVault) UpdateCredentialSecret(id int, newSecret []byte) error {
	vaultInfo, err := s.store.GetVaultInfo()
	if err != nil {
		return constants.ErrFailedToFetchVaultInfo
	}
	vaultData := vaultInfo.GetRawData()

	ephemeralPrivateKey, ephemeralPublicKey := crypto.GenerateAsymmetricKeyPair()

	// generate symmetric shared secret
	encryptionKey, _ := curve25519.X25519(ephemeralPrivateKey, vaultData.PublicKey)

	// hash to get 32 bit consistent key for encryption
	key := sha256.Sum256(encryptionKey)

	cipher, nonce, err := crypto.EncryptSecret(key[:], newSecret)
	if err != nil {
		return err
	}

	// Create a credential with ONLY the fields that need updating
	updatedCredential := model.CredentialData{
		Id:        id,
		Nonce:     nonce,
		Secret:    cipher,
		Ephemeral: ephemeralPublicKey,
	}

	return s.store.UpdateCredential(updatedCredential.EncodeToString())
}

func (s *KoshVault) ListCredentials(label, user string) ([]model.CredentialSummary, error) {
	if _, err := s.store.GetVaultInfo(); err != nil {
		return nil, err
	}

	credentials, err := s.store.SearchCredentialByLabelOrUser(label, user)
	if err != nil {
		return nil, err
	}

	return credentials, nil
}
