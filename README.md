# Kosh — Secure Password Manager

**Kosh** is a command-line password manager that securely stores credentials in an encrypted local vault using **SQLite**.
It employs modern cryptographic standards such as **Curve25519**, **ChaCha20-Poly1305**, and **Argon2** to ensure your data remains private and protected.

---

## Features

* Secure end-to-end encryption for all credentials
* Local, lightweight SQLite-based storage
* Master password protection for the vault
* Simple, cross-platform CLI interface
* Fast and dependency-free operation

---

## Installation

### From Source

```bash
git clone https://github.com/gitKashish/kosh.git
cd kosh
./build.sh
```

### Using Go Install (Recommended)

```bash
go install github.com/gitKashish/kosh@latest
```

---

## Command Reference

| Command                   | Description                                   |
| ------------------------- | --------------------------------------------- |
| `kosh help`               | Display help information                      |
| `kosh init`               | Initialize a new vault with a master password |
| `kosh add`                | Add or update a credential                    |
| `kosh get <label> <user>` | Retrieve a stored credential                  |
| `kosh delete <id>`        | Delete a stored credential                    |
---

## Usage

### Get Help

Get command reference and usage details:

```bash
kosh help
```

### Initialize Vault

Set up a new vault and master password:

```bash
kosh init
```

### Add Credential

Add or update a stored credential through an interactive prompt:

```bash
kosh add
```

### Retrieve Credential

Fetch a credential by label (group) and username:

```bash
kosh get github pluto   # Copy credential with label=github & user=pluto
```

### List Credential

Get a list of stored credentials filter by `label` and/or `user`:

```bash
kosh list                   # List all credentials
kosh list pluto             # Search users containing 'pluto'
kosh list --user pluto      # Same as above
kosh list --label github    # Search labels containing 'github'
```

### Delte Credential

Delete a stored credential by its ID:

```bash
kosh delete 101   # Delete credential with id=101
```
---

## How It Works

### Credential Encryption

1. Retrieve the vault’s public key.
2. Generate an ephemeral key pair for session key exchange.
3. Derive a symmetric key using Curve25519 with the vault’s public key and ephemeral private key.
4. Hash the symmetric key with SHA-256 to obtain a 32-byte key for ChaCha20.
5. Create a ChaCha20-Poly1305 AEAD instance.
6. Generate a random nonce.
7. Encrypt the plaintext secret using AEAD and the nonce.
8. Store the nonce, ephemeral public key, and cipher text (all base64-encoded) in the database.

### Credential Decryption

1. Decrypt the vault’s private key using the master password with Argon2.
2. Derive the symmetric key using the vault’s private key and the stored ephemeral public key.
3. Hash the symmetric key with SHA-256 to recreate the ChaCha key.
4. Decrypt the cipher text using the AEAD and nonce.

---

## Roadmap

* Implement credential deletion
* Implement vault export as JSON
* Add configuration management and enhanced CLI options
