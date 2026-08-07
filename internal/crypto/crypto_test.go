package crypto

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
)

// fixedKey returns a deterministic key of the right size. Tests that only
// exercise the AEAD use this instead of DeriveKey, which costs ~27ms a call.
func fixedKey(b byte) []byte {
	return bytes.Repeat([]byte{b}, chacha20poly1305.KeySize)
}

// Test Symmetric key generation
func TestSymmetricKey(t *testing.T) {
	secret := "oohh! my secret."
	salt := GenerateSalt()

	t.Run("symmetric key should be able to both encrypt and decrypt secret", func(t *testing.T) {
		key := DeriveKey([]byte(secret), salt)
		plaintext := []byte("launch codes: 0000")

		cipher, nonce, err := EncryptSecret(key, plaintext)
		if err != nil {
			t.Fatalf("unexpected error encrypting: %v", err)
		}

		got, err := DecryptSecret(key, cipher, nonce)
		if err != nil {
			t.Fatalf("unexpected error decrypting: %v", err)
		}
		if !bytes.Equal(got, plaintext) {
			t.Errorf("got %q, want %q", got, plaintext)
		}
	})

	t.Run("derived key has the expected length", func(t *testing.T) {
		if got := len(DeriveKey([]byte(secret), salt)); got != keyLength {
			t.Errorf("key length = %d, want %d", got, keyLength)
		}
	})

	t.Run("same secret and salt always derive the same key", func(t *testing.T) {
		// the property that lets a vault be opened a second time
		if !bytes.Equal(DeriveKey([]byte(secret), salt), DeriveKey([]byte(secret), salt)) {
			t.Error("same secret and salt produced different keys")
		}
	})

	t.Run("a different salt derives a different key", func(t *testing.T) {
		if bytes.Equal(DeriveKey([]byte(secret), salt), DeriveKey([]byte(secret), GenerateSalt())) {
			t.Error("different salts produced the same key")
		}
	})

	t.Run("a different secret derives a different key", func(t *testing.T) {
		// one character shorter
		if bytes.Equal(DeriveKey([]byte(secret), salt), DeriveKey([]byte("oohh! my secret"), salt)) {
			t.Error("different secrets produced the same key")
		}
	})

	t.Run("key derived from the wrong secret cannot decrypt", func(t *testing.T) {
		plaintext := []byte("launch codes: 0000")

		cipher, nonce, err := EncryptSecret(DeriveKey([]byte(secret), salt), plaintext)
		if err != nil {
			t.Fatalf("unexpected error encrypting: %v", err)
		}

		wrongKey := DeriveKey([]byte("not my secret."), salt)
		if _, err := DecryptSecret(wrongKey, cipher, nonce); err == nil {
			t.Error("expected an error, got nil")
		}
	})
}

func TestGenerateSalt(t *testing.T) {
	t.Run("has the expected length", func(t *testing.T) {
		if got := len(GenerateSalt()); got != 16 {
			t.Errorf("salt length = %d, want 16", got)
		}
	})

	t.Run("is not repeated across calls", func(t *testing.T) {
		seen := make(map[string]bool)
		for range 100 {
			s := string(GenerateSalt())
			if seen[s] {
				t.Fatal("GenerateSalt returned a duplicate salt")
			}
			seen[s] = true
		}
	})

	t.Run("is not all zero", func(t *testing.T) {
		if bytes.Equal(GenerateSalt(), make([]byte, 16)) {
			t.Error("salt is all zero, randomness is not being read")
		}
	})
}

// Test Asymmetric key pairs
func TestAsymmetricKeyPairs(t *testing.T) {
	t.Run(
		"public key encrypted secret should be decrypted using private key",
		func(t *testing.T) {
			pvt, pub := GenerateAsymmetricKeyPair()
			plaintext := []byte("s3cr3t-github-token")

			// sender: agree against the public key with a throwaway pair
			ephemeralPvt, ephemeralPub := GenerateAsymmetricKeyPair()
			sendShared, err := curve25519.X25519(ephemeralPvt, pub)
			if err != nil {
				t.Fatalf("unexpected error agreeing the shared secret: %v", err)
			}
			sendKey := sha256.Sum256(sendShared)

			cipher, nonce, err := EncryptSecret(sendKey[:], plaintext)
			if err != nil {
				t.Fatalf("unexpected error encrypting: %v", err)
			}

			// reader: re-agree using the private key and the ephemeral public key
			readShared, err := curve25519.X25519(pvt, ephemeralPub)
			if err != nil {
				t.Fatalf("unexpected error agreeing the shared secret: %v", err)
			}
			readKey := sha256.Sum256(readShared)

			got, err := DecryptSecret(readKey[:], cipher, nonce)
			if err != nil {
				t.Fatalf("unexpected error decrypting: %v", err)
			}
			if !bytes.Equal(got, plaintext) {
				t.Errorf("got %q, want %q", got, plaintext)
			}
		},
	)

	t.Run("both keys have the expected length", func(t *testing.T) {
		pvt, pub := GenerateAsymmetricKeyPair()
		if len(pvt) != curve25519.ScalarSize {
			t.Errorf("private key length = %d, want %d", len(pvt), curve25519.ScalarSize)
		}
		if len(pub) != curve25519.PointSize {
			t.Errorf("public key length = %d, want %d", len(pub), curve25519.PointSize)
		}
	})

	t.Run("neither key is all zero", func(t *testing.T) {
		pvt, pub := GenerateAsymmetricKeyPair()
		if bytes.Equal(pvt, make([]byte, len(pvt))) {
			t.Error("private key is all zero")
		}
		if bytes.Equal(pub, make([]byte, len(pub))) {
			t.Error("public key is all zero")
		}
	})

	t.Run("successive pairs differ", func(t *testing.T) {
		pvtA, pubA := GenerateAsymmetricKeyPair()
		pvtB, pubB := GenerateAsymmetricKeyPair()
		if bytes.Equal(pvtA, pvtB) {
			t.Error("two calls returned the same private key")
		}
		if bytes.Equal(pubA, pubB) {
			t.Error("two calls returned the same public key")
		}
	})

	t.Run("public key corresponds to the private key", func(t *testing.T) {
		pvt, pub := GenerateAsymmetricKeyPair()

		want, err := curve25519.X25519(pvt, curve25519.Basepoint)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(pub, want) {
			t.Error("public key does not correspond to the returned private key")
		}
	})

	t.Run("both sides agree on the same shared secret", func(t *testing.T) {
		pvtA, pubA := GenerateAsymmetricKeyPair()
		pvtB, pubB := GenerateAsymmetricKeyPair()

		fromA, err := curve25519.X25519(pvtA, pubB)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		fromB, err := curve25519.X25519(pvtB, pubA)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !bytes.Equal(fromA, fromB) {
			t.Error("the two sides derived different shared secrets")
		}
	})

	t.Run("an unrelated private key agrees on a different secret", func(t *testing.T) {
		pvtA, _ := GenerateAsymmetricKeyPair()
		_, pubB := GenerateAsymmetricKeyPair()
		pvtC, _ := GenerateAsymmetricKeyPair()

		shared, err := curve25519.X25519(pvtA, pubB)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		eavesdropper, err := curve25519.X25519(pvtC, pubB)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if bytes.Equal(shared, eavesdropper) {
			t.Error("an unrelated private key derived the same shared secret")
		}
	})
}

// Test secret encryption
func TestEncryptSecret(t *testing.T) {
	key := fixedKey(0xA5)
	plaintext := []byte("hunter2")

	t.Run("does not leak the plaintext", func(t *testing.T) {
		cipher, _, err := EncryptSecret(key, plaintext)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bytes.Contains(cipher, plaintext) {
			t.Error("ciphertext contains the plaintext")
		}
	})

	t.Run("nonce and ciphertext have the expected sizes", func(t *testing.T) {
		cipher, nonce, err := EncryptSecret(key, plaintext)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(nonce) != chacha20poly1305.NonceSizeX {
			t.Errorf("nonce length = %d, want %d", len(nonce), chacha20poly1305.NonceSizeX)
		}
		// the AEAD appends a 16 byte authentication tag
		if want := len(plaintext) + chacha20poly1305.Overhead; len(cipher) != want {
			t.Errorf("ciphertext length = %d, want %d", len(cipher), want)
		}
	})

	t.Run("encrypting the same secret twice gives different output", func(t *testing.T) {
		// reusing a nonce under one key would leak the plaintext
		firstCipher, firstNonce, err := EncryptSecret(key, plaintext)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		secondCipher, secondNonce, err := EncryptSecret(key, plaintext)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if bytes.Equal(firstNonce, secondNonce) {
			t.Error("nonce was reused across two encryptions")
		}
		if bytes.Equal(firstCipher, secondCipher) {
			t.Error("the same ciphertext was produced twice")
		}
	})

	t.Run("rejects a key of the wrong length", func(t *testing.T) {
		for _, size := range []int{0, 16, 31, 33, 64} {
			if _, _, err := EncryptSecret(bytes.Repeat([]byte{1}, size), plaintext); err == nil {
				t.Errorf("key size %d: expected an error, got nil", size)
			}
		}
	})
}

// Test secret decryption
func TestDecryptSecret(t *testing.T) {
	key := fixedKey(0xA5)
	plaintext := []byte("hunter2")

	// seal returns a fresh ciphertext and nonce under the shared test key.
	seal := func(t *testing.T, secret []byte) (cipher, nonce []byte) {
		t.Helper()
		cipher, nonce, err := EncryptSecret(key, secret)
		if err != nil {
			t.Fatalf("unexpected error encrypting: %v", err)
		}
		return cipher, nonce
	}

	t.Run("round-trips an empty secret", func(t *testing.T) {
		cipher, nonce := seal(t, []byte{})

		got, err := DecryptSecret(key, cipher, nonce)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("round-trips a large secret", func(t *testing.T) {
		large := bytes.Repeat([]byte("long passphrase "), 4096)
		cipher, nonce := seal(t, large)

		got, err := DecryptSecret(key, cipher, nonce)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(got, large) {
			t.Error("large secret did not survive the round trip")
		}
	})

	t.Run("fails with the wrong key", func(t *testing.T) {
		cipher, nonce := seal(t, plaintext)

		if _, err := DecryptSecret(fixedKey(0x5A), cipher, nonce); err == nil {
			t.Error("expected an error, got nil")
		}
	})

	t.Run("fails when the ciphertext is tampered with", func(t *testing.T) {
		// detecting this is the reason to use an AEAD rather than a raw stream
		cipher, nonce := seal(t, plaintext)
		cipher[0] ^= 0x01

		if _, err := DecryptSecret(key, cipher, nonce); err == nil {
			t.Error("expected an error, got nil")
		}
	})

	t.Run("fails when the authentication tag is tampered with", func(t *testing.T) {
		cipher, nonce := seal(t, plaintext)
		cipher[len(cipher)-1] ^= 0x01

		if _, err := DecryptSecret(key, cipher, nonce); err == nil {
			t.Error("expected an error, got nil")
		}
	})

	t.Run("fails when the ciphertext is truncated", func(t *testing.T) {
		cipher, nonce := seal(t, plaintext)

		if _, err := DecryptSecret(key, cipher[:len(cipher)-1], nonce); err == nil {
			t.Error("expected an error, got nil")
		}
	})

	t.Run("fails with a nonce from another encryption", func(t *testing.T) {
		cipher, _ := seal(t, plaintext)
		_, otherNonce := seal(t, plaintext)

		if _, err := DecryptSecret(key, cipher, otherNonce); err == nil {
			t.Error("expected an error, got nil")
		}
	})

	t.Run("rejects a nonce of the wrong length", func(t *testing.T) {
		cipher, nonce := seal(t, plaintext)

		for _, size := range []int{0, 12, 23} {
			if _, err := DecryptSecret(key, cipher, nonce[:size]); err == nil {
				t.Errorf("nonce size %d: expected an error, got nil", size)
			}
		}
	})

	t.Run("rejects a key of the wrong length without panicking", func(t *testing.T) {
		// a wrong-sized key used to leave a nil AEAD and panic on the next line
		cipher, nonce := seal(t, plaintext)

		for _, size := range []int{0, 16, 31, 33} {
			if _, err := DecryptSecret(bytes.Repeat([]byte{1}, size), cipher, nonce); err == nil {
				t.Errorf("key size %d: expected an error, got nil", size)
			}
		}
	})
}

// Test Credential Encryption and Decryption Cycle

// testVault mirrors the record `kosh init` writes: a Curve25519 key pair whose
// private half is sealed with a key derived from the master password.
type testVault struct {
	salt      []byte
	publicKey []byte
	secret    []byte // the sealed private key
	nonce     []byte
}

// newTestVault mirrors the vault creation in cmd.runInit.
func newTestVault(t *testing.T, masterPassword []byte) testVault {
	t.Helper()

	salt := GenerateSalt()
	unlockKey := DeriveKey(masterPassword, salt)
	pvt, pub := GenerateAsymmetricKeyPair()

	secret, nonce, err := EncryptSecret(unlockKey, pvt)
	if err != nil {
		t.Fatalf("unexpected error sealing the vault key: %v", err)
	}

	return testVault{salt: salt, publicKey: pub, secret: secret, nonce: nonce}
}

// sealedCredential holds what a row in the credentials table stores.
type sealedCredential struct {
	secret    []byte
	nonce     []byte
	ephemeral []byte
}

// sealCredential mirrors core.KoshVault.AddCredential: a throwaway key pair is
// agreed against the vault public key, and only its public half is kept.
func sealCredential(t *testing.T, v testVault, plaintext []byte) sealedCredential {
	t.Helper()

	ephemeralPvt, ephemeralPub := GenerateAsymmetricKeyPair()

	shared, err := curve25519.X25519(ephemeralPvt, v.publicKey)
	if err != nil {
		t.Fatalf("unexpected error agreeing the shared secret: %v", err)
	}
	key := sha256.Sum256(shared)

	secret, nonce, err := EncryptSecret(key[:], plaintext)
	if err != nil {
		t.Fatalf("unexpected error encrypting the credential: %v", err)
	}

	return sealedCredential{secret: secret, nonce: nonce, ephemeral: ephemeralPub}
}

// openCredential mirrors core.KoshVault.DecryptCredential: unwrap the vault
// private key with the master password, re-agree the shared secret against the
// stored ephemeral public key, then open the credential.
func openCredential(v testVault, masterPassword []byte, c sealedCredential) ([]byte, error) {
	unlockKey := DeriveKey(masterPassword, v.salt)

	vaultPvt, err := DecryptSecret(unlockKey, v.secret, v.nonce)
	if err != nil {
		return nil, err
	}

	shared, err := curve25519.X25519(vaultPvt, c.ephemeral)
	if err != nil {
		return nil, err
	}
	key := sha256.Sum256(shared)

	return DecryptSecret(key[:], c.secret, c.nonce)
}

func TestCredentialCycle(t *testing.T) {
	masterPassword := []byte("correct horse battery staple")

	t.Run("a sealed credential opens with the master password", func(t *testing.T) {
		vault := newTestVault(t, masterPassword)
		plaintext := []byte("s3cr3t-github-token")

		got, err := openCredential(vault, masterPassword, sealCredential(t, vault, plaintext))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(got, plaintext) {
			t.Errorf("got %q, want %q", got, plaintext)
		}
	})

	t.Run("the wrong master password fails to open it", func(t *testing.T) {
		vault := newTestVault(t, masterPassword)
		credential := sealCredential(t, vault, []byte("s3cr3t-github-token"))

		// one character shorter
		if _, err := openCredential(vault, []byte("correct horse battery stapl"), credential); err == nil {
			t.Error("expected an error, got nil")
		}
	})

	t.Run("every credential gets its own ephemeral key and nonce", func(t *testing.T) {
		vault := newTestVault(t, masterPassword)
		plaintext := []byte("same secret twice")

		first := sealCredential(t, vault, plaintext)
		second := sealCredential(t, vault, plaintext)

		if bytes.Equal(first.ephemeral, second.ephemeral) {
			t.Error("ephemeral public key was reused across credentials")
		}
		if bytes.Equal(first.nonce, second.nonce) {
			t.Error("nonce was reused across credentials")
		}
		if bytes.Equal(first.secret, second.secret) {
			t.Error("the same secret produced identical ciphertext twice")
		}
	})

	t.Run("secrets of any shape survive the cycle", func(t *testing.T) {
		vault := newTestVault(t, masterPassword)
		secrets := [][]byte{
			[]byte("first"),
			[]byte(""),
			[]byte("a passphrase with spaces and symbols !@#$%^&*()"),
			[]byte("emoji and accents: café 🔐"),
			bytes.Repeat([]byte("x"), 4096),
		}

		for _, want := range secrets {
			got, err := openCredential(vault, masterPassword, sealCredential(t, vault, want))
			if err != nil {
				t.Fatalf("unexpected error for %d-byte secret: %v", len(want), err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("%d-byte secret did not survive the round trip", len(want))
			}
		}
	})

	t.Run("a credential cannot be opened by a different vault", func(t *testing.T) {
		// what `kosh copy` works around: profiles are separate vaults, so
		// ciphertext is not portable between them even under one password
		source := newTestVault(t, masterPassword)
		target := newTestVault(t, masterPassword)

		credential := sealCredential(t, source, []byte("s3cr3t-github-token"))

		if _, err := openCredential(target, masterPassword, credential); err == nil {
			t.Error("a credential sealed for one vault opened in another")
		}
	})
}
