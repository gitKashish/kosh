package core

import (
	"crypto/sha256"

	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/crypto"
	"git.plutolab.org/plutolab/kosh/internal/logger"
	"git.plutolab.org/plutolab/kosh/internal/model"
	"git.plutolab.org/plutolab/kosh/internal/storage"
	"golang.org/x/crypto/curve25519"
)

type VaultService struct {
	store storage.Store
}

// NewVaultService creates a new service instance
func NewVaultService(store storage.Store) *VaultService {
	return &VaultService{store}
}

// verifyMasterPassword checks if the provided master password can unlock the vault.
// It returns an error if the password is incorrect or if the vault cannot be read.
func (s *VaultService) VerifyMasterPassword(password []byte) error {
	vault, err := s.store.GetVaultInfo()
	if err != nil {
		return constants.ErrFailedToFetchVaultInfo
	}
	vaultData := vault.GetRawData()

	unlockKey := crypto.GenerateSymmetricKey(password, vaultData.Salt)
	if _, err := crypto.DecryptSecret(unlockKey, vaultData.Secret, vaultData.Nonce); err != nil {
		return constants.ErrIncorrectMasterPassword
	}

	return nil
}

func (s *VaultService) AddCredential(label, user string, secret []byte) error {
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

func (s *VaultService) DecryptCredential(credential *model.Credential, password []byte) (string, error) {
	vaultInfo, err := s.store.GetVaultInfo()
	if err != nil {
		logger.Debug("decryptCredential:failed to get vault info")
		return "", err
	}
	vaultData := vaultInfo.GetRawData()

	// Derive unlock key
	unlockKey := crypto.GenerateSymmetricKey(password, vaultData.Salt)

	// Decrypt vault private key
	vaultPrivateKey, err := crypto.DecryptSecret(unlockKey, vaultData.Secret, vaultData.Nonce)
	if err != nil {
		logger.Debug("decryptCredential:failed to get private key from vault")
		return "", constants.ErrFailedToDecryptCredential
	}

	// Generate shared secret
	credData := credential.GetRawData()
	decryptionKey, _ := curve25519.X25519(vaultPrivateKey, credData.Ephemeral)

	// Hash to get 32-bit consistent key
	key := sha256.Sum256(decryptionKey)

	plainText, err := crypto.DecryptSecret(key[:], credData.Secret, credData.Nonce)
	if err != nil {
		return "", constants.ErrFailedToDecryptCredential
	}

	return string(plainText), nil
}

// UpdateCredentialSecret encrypts a new secret for an existing credential and saves it.
func (s *VaultService) UpdateCredentialSecret(id int, newSecret []byte) error {
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
