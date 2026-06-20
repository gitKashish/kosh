# Architecture

This document covers the internals of Kosh: cryptographic design, database schema, module responsibilities, and data flow.

---

## Module map

| Package | Responsibility |
|---|---|
| `cmd/` | CLI surface — one file per subcommand, wired to cobra |
| `internal/core` | Business logic: vault operations and credential lifecycle |
| `internal/crypto` | Thin wrappers around Go crypto primitives |
| `internal/storage` | SQLite persistence: Store interface + VaultStore implementation |
| `internal/model` | Plain data structs and encode/decode helpers |
| `internal/search` | Scoring and ranking logic |
| `internal/ui` | Terminal I/O: interactive search, input fields, clipboard |
| `internal/logger` | Colored output; debug mode controlled at build time |
| `internal/encoding` | Base64 helpers used at the model boundary |
| `internal/constants` | Sentinel errors, user-facing strings, tuning constants |

---

## Cryptographic design

### Primitives

| Purpose | Algorithm |
|---|---|
| Key derivation | Argon2id |
| Symmetric encryption | XChaCha20-Poly1305 |
| Key exchange | Curve25519 (X25519) |
| Shared-secret hashing | SHA-256 |

### Vault initialization (`kosh init`)

```
master_password + random_salt (16 bytes)
        │
        ▼
   Argon2id(t=1, m=64MB, p=4) ─────────► unlock_key (32 bytes)
                                                │
Generate Curve25519 keypair                     │
  private_key (32 bytes)                        │
  public_key  (32 bytes)                        │
        │                                       │
        └──── XChaCha20-Poly1305(unlock_key) ──►  cipher + nonce
```

Stored in the `vault` table:
- `public_key` — Curve25519 public key (base64)
- `secret` — encrypted Curve25519 private key (base64)
- `nonce` — nonce for the above encryption (base64)
- `salt` — Argon2id salt (base64)

The master password is **never stored**. It is re-derived on every operation that needs the vault private key.

### Adding a credential (`kosh add`, `kosh generate`)

```
Generate ephemeral Curve25519 keypair
  ephemeral_private  (32 bytes)
  ephemeral_public   (32 bytes)

X25519(ephemeral_private, vault_public_key)
        │
        ▼
   shared_secret (32 bytes)
        │
     SHA-256
        │
        ▼
   encryption_key (32 bytes)
        │
XChaCha20-Poly1305(encryption_key, plaintext_secret)
        │
        ▼
   cipher + nonce
```

Stored per credential in the `credentials` table:
- `ephemeral` — ephemeral public key (base64)
- `secret` — ciphertext (base64)
- `nonce` — nonce (base64)
- `label`, `user` — plaintext metadata

Each credential has its own ephemeral keypair and nonce. There is no key reuse between credentials.

### Decrypting a credential (`kosh search`, `kosh get`)

```
master_password + vault.salt
        │
   Argon2id ────────────────────► unlock_key
                                        │
                             XChaCha20-Poly1305.Open(vault.secret, vault.nonce)
                                        │
                                        ▼
                                vault_private_key (32 bytes)

X25519(vault_private_key, credential.ephemeral)
        │
     SHA-256
        │
        ▼
   decryption_key
        │
XChaCha20-Poly1305.Open(credential.secret, credential.nonce)
        │
        ▼
   plaintext_secret
```

The plaintext secret is held in memory only for the duration of the operation (copy to clipboard) and never written to disk.

---

## Database schema

The vault file is at `~/.kosh/kosh.db`. All crypto values are base64-encoded strings.

### `vault` table

```sql
CREATE TABLE vault (
    id         INTEGER PRIMARY KEY CHECK (id = 1),  -- singleton row
    public_key TEXT NOT NULL,
    nonce      TEXT NOT NULL,
    secret     TEXT NOT NULL,
    salt       TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

Constrained to exactly one row (`id = 1`).

### `credentials` table

```sql
CREATE TABLE credentials (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    label        TEXT NOT NULL,
    user         TEXT NOT NULL,
    access_count NUMBER NOT NULL DEFAULT 0,
    secret       TEXT NOT NULL,
    ephemeral    TEXT NOT NULL,
    nonce        TEXT NOT NULL,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    accessed_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(label, user)
);
```

### SQLite pragmas

Kosh sets the following pragmas on every connection:

| Pragma | Value | Reason |
|---|---|---|
| `journal_mode` | `WAL` | Better concurrent reads |
| `synchronous` | `NORMAL` | Safe and faster than FULL |
| `foreign_keys` | `ON` | Enforce referential integrity |
| `temp_store` | `MEMORY` | Keep temp data off disk |
| `secure_delete` | `ON` | Overwrite deleted pages |
| `trusted_schema` | `OFF` | Defence against malicious schema |

---

## Search algorithm

Implemented in `internal/search/search.go`.

### Score formula

```
score = label_score  × 0.60
      + user_score   × 0.20
      + recency_score × 0.12
      + freq_score   × 0.05
```

Credentials below a score threshold of `0.20` are excluded.

### String scoring (`stringScore`)

1. Exact match → `1.0` (max)
2. Otherwise: normalized Levenshtein similarity
3. Prefix match → `+1.0` boost
4. Substring match → `+0.5` boost

### Recency score

Half-life decay of ~12 hours from last access:

```
recency = 1 / (1 + hours_since_last_access / 12)
```

### Frequency score

Logarithmic, normalized:

```
frequency = log(access_count + 1) / 5
```

### Access count baseline reset

When any credential's `access_count` exceeds `AccessCountResetThreshold` (10,000), all counts are reduced by that threshold (`MAX(count - threshold, 0)`). This prevents a single frequently-accessed credential from permanently dominating search rankings.

### Tie-breaking

Results with equal scores are sorted by:
1. Higher `access_count` first
2. Label lexicographic order

---

## Logger and debug mode

`internal/logger` provides six log levels with ANSI color output:

| Function | Symbol | Color | Stream |
|---|---|---|---|
| `Error` | `[✗]` | Red | stderr |
| `Info` | `[✓]` | Green | stdout |
| `Warn` | `[!]` | Yellow | stdout |
| `Debug` | `[→]` | Blue | stdout |
| `Prompt` | `[?]` | Cyan | stdout |
| `Muted` | `[•]` | Gray | stdout |

`Debug` calls are no-ops in production builds. Enable them by setting `BuildMode=debug` at link time:

```sh
go build -ldflags="-X git.plutolab.org/plutolab/kosh/internal/logger.BuildMode=debug"
```

Debug output includes the file and line number of the caller.

`logger.Pause()` silences all output temporarily. It is used by the interactive search TUI to prevent log lines from corrupting the raw-mode terminal display.

---

## Interactive search TUI

`ui.InteractiveSearch` (`internal/ui/search.go`) puts the terminal into raw mode and implements a live-filter picker:

- Each keystroke calls the provided `searchFn` and re-renders results
- Arrow keys move the selection; Enter confirms; Esc/Ctrl-C cancels
- Uses a single `os.Stdout.Write` per frame to avoid flicker
- Restores the terminal on exit via `defer term.Restore(...)`

The function is generic (`InteractiveSearch[T Searchable]`) and can be reused for any type that implements `Display() string`.

---

## Release

Releases are built with [goreleaser](https://goreleaser.com) using `.goreleaser.yaml`. Targets: Linux, macOS, Windows (amd64/arm64). The release build sets:

```
-X git.plutolab.org/plutolab/kosh/cmd.AppVersion={{.Version}}
-X git.plutolab.org/plutolab/kosh/internal/logger.BuildMode=production
```

`CGO_ENABLED=0` is set so the binary is fully static (the SQLite driver is pure Go via `modernc.org/sqlite`).
