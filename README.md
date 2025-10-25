# Pluto-Vault

## Requirements
#### CRUD
    - Create a credential entry. DONE
    - Fetch existing credential entry.
    - Update old credential entry. DONE
    - Delete existing credential entry.
    - Dump all data as a JSON file.

#### Credential Encryption
    1. Get vault's public key.
    2. Create an ephemeral key pair for generating a symmetric key. (curve25519)
    3. Generate a symmetric key using ephemeral private key and vault's public key. (curve25519)
    4. Symmetric key is hashed to get 32 byte key for ChaCha. (sha256)
    5. generate ChaCha AEAD. (chacha20poly1305)
    6. Randomly generate a Nonce for uniqueness.
    7. AEAD is used to encrypt the plain text secrete with the Nonce.
    8. Store Nonce, Ephemeral Public Key and the Cipher as base64 encoded strings.

#### Credential Decryption
    1. Decrypt the vault's private key using the user entered master password and the salt. (argon2)
    2. Generate the symmetric key for the credential with the vault's private key and credential's ephemeral public key. (curve25519)
    3. Hash the symmetric key to get a 32 byte key for ChaCha. (sha256)
    4. generate ChaCha AEAD using the symmetric key. (chacha20poly1305)
    5. Decrypt the cipher with the AEAD alongside the stored Nonce.

### Command Structure
- `init` : Setup kosh vault. Ask user for master password and initialize cypto data.
- `add` : Start interactive session to add a new credential. Update credential if it already exists.
- `get` : Retrieve a stored secret by mentioning the group and the username for that credential.
