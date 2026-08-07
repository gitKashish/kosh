# Kosh — Secure, Local-First Password Manager

Kosh is a fast, offline-first CLI password manager written in Go. It stores credentials in encrypted SQLite vaults using **Curve25519**, **XChaCha20-Poly1305**, and **Argon2id** — no cloud, no network, nothing leaves your machine.

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

Debug mode prints verbose log lines including file/line caller info. Enabled by setting `KOSH_DEBUG` to a truthy value
like `1`, `t`, `true` etc.


---

## Quick start

```sh
# 1. Initialize the vault of the active profile (first time only)
kosh init

# 2. Add a credential
kosh add

# 3. Search (default command — runs when no subcommand is given)
kosh github
# equivalent to:
kosh search github
```

Kosh is multi-profile. Every command reads from and writes to the **active profile**, and each profile is its own
encrypted vault with its own master password:

```
~/.kosh/
├── config.json            # { "active_profile": "work" }
└── profiles/
    ├── default.db         # vault + credentials, master password A
    ├── work.db            # vault + credentials, master password B
    └── personal.db        # vault + credentials, master password C
```

`config.json` is written `0600`, `~/.kosh` and `~/.kosh/profiles` are `0700`. The default profile is `default`.

A legacy single-vault install (`~/.kosh/kosh.db`) is migrated automatically on first run: `profiles/` is created and
`kosh.db` is renamed to `profiles/default.db`. Nothing is re-encrypted and the master password is unchanged. The
migration bails out early if `profiles/default.db` already exists, so it can never clobber an existing profile.

---

## Commands

| Command | Description |
|---|---|
| `kosh` | Short-hand for `kosh search` (no args) |
| `kosh init` | Initialize the active profile's vault with a master password |
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
| `kosh use [profile]` | Switch the active profile (no args → interactive picker) |
| `kosh profile list [filter]` | Show profiles and which one is active |
| `kosh profile create <profile>` | Create a profile, switch to it, initialize its vault |
| `kosh profile delete <profile>` | Permanently delete a profile and its credentials |
| `kosh copy <id> <profile>` | Copy a credential into another profile's vault |

`kosh profile list` and `kosh use` read only filenames — they open no vault and need no master password.
`kosh profile create` initializes the vault as part of creation, so `kosh init` is not needed afterwards.

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

### Profile names

`kosh profile create` sanitizes the name before it becomes a filename (`core.SanitizeProfileName`):

- accents are folded to ASCII (`café` → `cafe`)
- anything outside `A-Z a-z 0-9 _ - <space>` is removed
- whitespace runs become `_`; repeated `_` or `-` collapse to one; leading/trailing `_` and `-` are trimmed
- the result is truncated to 252 characters (255-char filename limit minus `.db`)
- an empty result, or a Windows reserved device name (`con`, `nul`, `lpt1`, …), is rejected — on every platform, so a
  `profiles/` directory stays portable

Profiles are looked up **case-insensitively** everywhere (`core.ProfileService.ResolveProfile`), which is what stops
`work` and `Work` from coexisting. The lookup returns the spelling stored on disk, and callers must use that name for
anything that then touches the file — opening a vault under the name the user typed would silently create an empty
one beside the real one on a case-sensitive filesystem.

Note that only `create` sanitizes. `use`, `profile delete` and `copy` match against the names actually on disk.

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
│   ├── generate.go             # kosh generate
│   ├── use.go                  # kosh use
│   ├── copy.go                 # kosh copy
│   └── profile/                # kosh profile
│       ├── root.go             # Parent command
│       ├── list.go             # kosh profile list
│       ├── create.go           # kosh profile create (+ rollback of partial creates)
│       └── delete.go           # kosh profile delete
├── internal/
│   ├── app/
│   │   └── context.go          # app.Context: Config, Store, Vault, Profile (dependency injection)
│   ├── config/
│   │   └── config.go           # ~/.kosh/config.json load/save, active profile
│   ├── core/
│   │   ├── vault_service.go    # Business logic: add/decrypt/update credentials
│   │   └── profile_service.go  # Profile listing, resolution, switching, deletion, name sanitization
│   ├── crypto/
│   │   ├── crypto.go           # Argon2id, XChaCha20-Poly1305, Curve25519 wrappers
│   │   └── file.go             # Random-overwrite file scrubbing
│   ├── storage/
│   │   ├── store.go            # Store interface + SQLite init/pragmas
│   │   ├── vault.go            # Vault table CRUD + schema migrations
│   │   └── credential.go       # Credentials table CRUD
│   ├── model/
│   │   ├── credential.go       # Credential / CredentialData / CredentialSummary
│   │   ├── vault.go            # Vault / VaultData models
│   │   └── profile.go          # Profile model
│   ├── search/
│   │   └── search.go           # Weighted fuzzy search + Damerau-Levenshtein scoring
│   ├── ui/
│   │   ├── search.go           # Interactive TUI search (raw terminal mode)
│   │   ├── field.go            # Input helpers (secret field, string field, confirm)
│   │   ├── output.go           # Colored terminal output with glyphs + active-profile prefix
│   │   ├── table.go            # Auto-sized table with active-row pointer
│   │   ├── time.go             # Relative timestamps
│   │   └── clipboard.go        # Clipboard copy
│   ├── logger/
│   │   └── logger.go           # Structured logger setup (KOSH_DEBUG)
│   ├── encoding/
│   │   └── text.go             # Base64 encode/decode + accent folding helpers
│   └── constants/
│       ├── credential.go       # AccessCountResetThreshold
│       ├── errors.go           # Sentinel errors
│       ├── messages.go         # User-facing message strings
│       └── prompts.go          # Prompt strings
└── .goreleaser.yaml            # Release automation (Linux / macOS / Windows)
```

Commands are `NewCmdX(ctx *app.Context)` constructors registered in `cmd/root.go`. The store is opened in
`PersistentPreRun` and closed in `PersistentPostRun`. `core.VaultService` and `core.ProfileService` are interfaces, so
commands can be tested with fakes.

---

## Architecture

See [docs/architecture.md](docs/architecture.md) for the full write-up. Summary:

**Vault key derivation**
Master password + random 16-byte salt → Argon2id (t=1, m=64MB, p=4) → 32-byte unlock key.

**Vault storage**
A Curve25519 keypair is generated at `kosh init`. The private key is encrypted with the unlock key via XChaCha20-Poly1305. The public key and ciphertext are stored in the `vault` table. Each profile has its own keypair, salt and master password — one vault per profile, no shared keys.

**Credential encryption**
Each credential uses a fresh ephemeral Curve25519 keypair. The shared secret (`X25519(ephemeral_priv, vault_pub)`) is hashed with SHA-256 to produce the encryption key. The secret is encrypted with XChaCha20-Poly1305; the ephemeral public key, ciphertext, and nonce are all stored in the `credentials` table.

**Decryption**
Derive unlock key → decrypt vault private key → `X25519(vault_priv, ephemeral_pub)` → SHA-256 → decrypt credential.

**Schema migrations**
An ordered `migrations` list in `internal/storage/vault.go` is applied transactionally on every store init, tracked in a `schema_migrations` table.

**Search**
The match score is `(label × 0.60 + user × 0.20)`, multiplied by `(1 + recency × 0.12 + frequency × 0.05)`. Recency and frequency **modulate** the match rather than adding to it, so usage habit can never promote a credential above a closer match — it only breaks ties. String similarity uses Damerau-Levenshtein distance (adjacent transpositions cost one edit) with asymptotic prefix (0.8), substring (0.5) and ordered-subsequence (0.4) boosts, giving a fixed ranking hierarchy:

```
exact > prefix > substring > subsequence > fuzzy edit-distance
```

The subsequence boost is what makes abbreviations work — `gpat` matches `git_personal_access_token`. Results above a threshold of 0.2 are returned sorted by score.

---

## Running tests

```sh
go test ./...
```

Tests currently cover the password generator, search scoring, crypto helpers, relative time formatting, and the profile
service (name sanitization, case-insensitive resolution, path-traversal rejection). More coverage is a welcome
contribution.

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
- Each profile is cryptographically isolated: its own keypair, salt and master password. Unlocking one grants no access to another
- Each credential uses a unique ephemeral keypair and nonce — no key or nonce reuse
- SQLite is opened with `secure_delete=ON`; deleted rows are overwritten
- `kosh profile delete` overwrites the vault file with random bytes and `fsync`s before unlinking (best-effort on CoW/log-structured filesystems and wear-levelled SSDs)
- `kosh copy` needs only the target profile's public key to write, so it does not prompt for the target's master password. Writing into a profile does not imply the ability to read it
- Master-password and secret confirmations use constant-time comparison (`crypto/subtle`)
- `Credential`, `CredentialData` and `CredentialSummary` implement `slog.LogValue`, so debug logs can never print ciphertext, nonces or ephemeral keys
- The vault file permissions are `0700` on the `.kosh` directory

For the full cryptographic design see [docs/architecture.md](docs/architecture.md).
