package crypto

import (
	"crypto/rand"
	"fmt"

	"github.com/gitKashish/kosh/src/internals/logger"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
)

const (
	keyTime    = 1
	keyMemory  = 64 * 1024
	keyThreads = 4
	keyLength  = 32
)

func GenerateSymmetricKey(secret, salt []byte) []byte {
	return argon2.IDKey(secret, salt, keyTime, keyMemory, keyThreads, keyLength)
}

func DecryptPrivateKey(unlockKey []byte, secret []byte, nonce []byte) ([]byte, error) {
	aead, _ := chacha20poly1305.NewX(unlockKey)
	return aead.Open(nil, nonce, secret, nil)
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

func EncryptSecret(key, secret []byte) (cipher, nonce []byte) {
	// create AEAD with the shared secret
	aead, _ := chacha20poly1305.NewX(key)

	// generate nonce for encrypting the secret
	nonce = make([]byte, aead.NonceSize())
	_, _ = rand.Read(nonce)

	// encrypt the secret
	cipher = aead.Seal(nil, nonce, secret, nil)

	return cipher, nonce
}

func DecryptSecret(key, cipher, nonce []byte) ([]byte, error) {
	// create AEAD with the shared secret
	aead, _ := chacha20poly1305.NewX(key)

	// verify nonce length
	if len(nonce) != aead.NonceSize() {
		return nil, fmt.Errorf("incorrect nonce len")
	}

	// decrypt the secret
	secret, err := aead.Open(nil, nonce, cipher, nil)
	if err != nil {
		logger.Error("unable to decrypt secret")
		return nil, err
	}

	return secret, nil
}
