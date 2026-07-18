# Kosh — Secure, Local-First Password Manager

Kosh is a fast, offline-first CLI password manager written in Go. It stores credentials in an encrypted SQLite vault using **Curve25519**, **XChaCha20-Poly1305**, and **Argon2id** — no cloud, no network, nothing leaves your machine.

This README is for **developers and contributors**. For end-user docs visit [kosh.plutolab.org](https://kosh.plutolab.org).

---

## Prerequisites

- Go 1.26 or later
- No CGO required (uses a pure-Go SQLite driver)

---

## Build

```sh
go build
```

This produces a `kosh` binary in the project root.

### Debug build

Debug builds print verbose log lines including file/line caller info. Enabled via an `ldflags` injection:

```sh
go build -ldflags="-X git.plutolab.org/plutolab/kosh/internal/logger.BuildMode=debug"
```

The flag sets `logger.BuildMode` from its default `"production"` to `"debug"`, enabling all `logger.Debug(...)` calls throughout the codebase.

---

## Quick start

```sh
# 1. Initialize the vault (first time only)
kosh init

# 2. Add a credential
kosh add

# 3. Search (default command — runs when no subcommand is given)
kosh github
# equivalent to:
kosh search github
```

The vault lives at `~/.kosh/kosh.db`.

---

## Commands

| Command | Description |
|---|---|
| `kosh` | Short-hand for `kosh search` (no args) |
| `kosh init` | Initialize the vault with a master password |
| `kosh add` | Interactively add a new credential |
| `kosh search [label] [user]` | Fuzzy-search credentials (default command) |
| `kosh search` (no args) | Interactive live-filter search (arrow keys + enter) |
| `kosh get <label> <user>` | Retrieve credential by exact label + user |
| `kosh list` | List all credentials |
| `kosh list -l <label> -u <user>` | List with filters |
| `kosh update <id>` | Update label, user, or secret for a credential |
| `kosh delete <id>` | Delete a credential by ID |
| `kosh generate <label> <user>` | Generate and store a strong password |
| `kosh generate -n` | Generate a password without saving it |

### Shorthand

Any argument that isn't a known subcommand is treated as a search query:

```sh
kosh aws        # → kosh search aws
kosh gh alice   # → kosh search gh alice
```

### Password generation flags

```sh
kosh generate -l 32 --require "upper=2,lower=10,digit=5,symbol=3" <label> <user>
kosh generate --symbol=false <label> <user>
kosh generate -n   # generate only, copy to clipboard, don't save
```

---

## Project structure

```
kosh/
├── main.go                     # Entry point
├── cmd/                        # CLI commands (cobra)
│   ├── root.go                 # Root command, arg interception, Execute()
│   ├── init.go                 # kosh init
│   ├── add.go                  # kosh add
│   ├── get.go                  # kosh get
│   ├── search.go               # kosh search (default)
│   ├── list.go                 # kosh list
│   ├── update.go               # kosh update
│   ├── delete.go               # kosh delete
│   └── generate.go             # kosh generate
├── internal/
│   ├── core/
│   │   └── vault_service.go    # Business logic: add/decrypt/update credentials
│   ├── crypto/
│   │   └── crypto.go           # Argon2id, XChaCha20-Poly1305, Curve25519 wrappers
│   ├── storage/
│   │   ├── store.go            # Store interface + SQLite init/pragmas
│   │   ├── vault.go            # Vault table CRUD
│   │   └── credential.go       # Credentials table CRUD
│   ├── model/
│   │   ├── credential.go       # Credential / CredentialData / CredentialSummary
│   │   └── vault.go            # Vault / VaultData models
│   ├── search/
│   │   └── search.go           # Weighted fuzzy search + Levenshtein scoring
│   ├── ui/
│   │   ├── search.go           # Interactive TUI search (raw terminal mode)
│   │   ├── field.go            # Input helpers (secret field, string field, confirm)
│   │   └── clipboard.go        # Clipboard copy
│   ├── logger/
│   │   └── logger.go           # Colored terminal logger; BuildMode controls debug output
│   ├── encoding/
│   │   └── text.go             # Base64 encode/decode helpers
│   └── constants/
│       ├── credential.go       # AccessCountResetThreshold
│       ├── errors.go           # Sentinel errors
│       ├── messages.go         # User-facing message strings
│       └── prompts.go          # Prompt strings
└── .goreleaser.yaml            # Release automation (Linux / macOS / Windows)
```

---

## Architecture

See [docs/architecture.md](docs/architecture.md) for the full write-up. Summary:

**Vault key derivation**
Master password + random 16-byte salt → Argon2id (t=1, m=64MB, p=4) → 32-byte unlock key.

**Vault storage**
A Curve25519 keypair is generated at `kosh init`. The private key is encrypted with the unlock key via XChaCha20-Poly1305. The public key and ciphertext are stored in the `vault` table.

**Credential encryption**
Each credential uses a fresh ephemeral Curve25519 keypair. The shared secret (`X25519(ephemeral_priv, vault_pub)`) is hashed with SHA-256 to produce the encryption key. The secret is encrypted with XChaCha20-Poly1305; the ephemeral public key, ciphertext, and nonce are all stored in the `credentials` table.

**Decryption**
Derive unlock key → decrypt vault private key → `X25519(vault_priv, ephemeral_pub)` → SHA-256 → decrypt credential.

**Search**
Weighted scoring across label (60%), user (20%), recency (12%), and frequency (5%). String similarity uses Levenshtein distance with prefix/substring/subsequence boosts. Results above a threshold of 0.2 are returned sorted by score.

---

## Running tests

```sh
go test ./...
```

Tests currently cover the password generator and search functionality. More coverage is a welcome contribution.

---

## Contributing

1. `go vet ./...` must pass before submitting
2. Keep PRs small and focused on one thing
3. Do not add unnecessary dependencies
4. Follow existing naming and package conventions

Areas that need help: test coverage, security audits, documentation.

---

## Security model

- The master password is never stored; it is derived each time
- Losing the master password **permanently locks the vault** — no recovery mechanism exists
- Each credential uses a unique ephemeral keypair and nonce — no key or nonce reuse
- SQLite is opened with `secure_delete=ON`; deleted rows are overwritten
- The vault file permissions are `0700` on the `.kosh` directory

For the full cryptographic design see [docs/architecture.md](docs/architecture.md).
