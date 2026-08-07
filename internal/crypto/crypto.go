package crypto

import (
	"crypto/rand"
	"fmt"
	"log/slog"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
)

const (
	// TODO: store these parameters in the vault
	keyTime    = 1
	keyMemory  = 64 * 1024
	keyThreads = 4
	keyLength  = 32
)

func DeriveKey(secret, salt []byte) []byte {
	return argon2.IDKey(secret, salt, keyTime, keyMemory, keyThreads, keyLength)
}

func GenerateSalt() []byte {
	salt := make([]byte, 16)
	_, _ = rand.Read(salt)
	return salt
}

func GenerateAsymmetricKeyPair() (privateKey, publicKey []byte) {
	// generate a random private key
	privateKey = make([]byte, 32)
	_, _ = rand.Read(privateKey)

	// generate corresponding public key
	publicKey, _ = curve25519.X25519(privateKey, curve25519.Basepoint)

	return privateKey, publicKey
}

func EncryptSecret(key, secret []byte) (cipher, nonce []byte, err error) {
	// create AEAD with the shared secret
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, nil, err
	}

	// generate nonce for encrypting the secret
	nonce = make([]byte, aead.NonceSize())
	_, _ = rand.Read(nonce)

	// encrypt the secret
	cipher = aead.Seal(nil, nonce, secret, nil)

	return cipher, nonce, nil
}

func DecryptSecret(key, cipher, nonce []byte) ([]byte, error) {
	// create AEAD with the shared secret
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		slog.Debug("failed to create AED from key", "error", err)
		return nil, err
	}

	// verify nonce length
	if len(nonce) != aead.NonceSize() {
		return nil, fmt.Errorf("incorrect nonce len")
	}

	// decrypt the secret
	secret, err := aead.Open(nil, nonce, cipher, nil)
	if err != nil {
		return nil, err
	}

	return secret, nil
}
