# Kosh — Secure, Local-First Password Manager

**Developer & Contributor README**

Kosh is a fast, secure, offline-first command-line password manager.
It uses an encrypted SQLite vault and modern cryptographic primitives such as
**Curve25519**, **ChaCha20-Poly1305**, and **Argon2id**.

This README is intended **for developers and contributors**.
For end-user documentation, installation guides, and usage tutorials, visit:

👉 **[Kosh Docs](https://kosh.plutolab.org)**

---

## 📚 Documentation

All user-facing docs (installation, usage, guides, architecture explanations) now live at:

➡️ **[Getting Started](https://kosh.plutolab.org/guides/getting-started)**

Developer-focused docs such as architecture, cryptography, and system internals are also gradually being consolidated there.

---

# 🧩 Project Overview

Kosh emphasizes:

* **Local-first security**—all encryption happens on device, nothing leaves the machine
* **Zero external dependencies**—only standard Go + modern crypto libs
* **Deterministic + minimal code paths**
* **Security-focused design**—memory is overwritten where possible, SQLite secure-delete, master password never stored

Kosh is written entirely in **Go**, with a small and clean internal module structure.

---

# 🏗 Architecture (High-Level)

### 🔐 Cryptography

* Master password → Argon2id → symmetric vault key
* Vault unlock secret encrypted using **ChaCha20-Poly1305**
* Each credential encrypted with an ephemeral Curve25519 key pair + shared secret
* Nonces generated per-entry, no reuse
* Secrets decrypted only when necessary, wiped immediately after usage

### 🗄 SQLite Vault

* Single encrypted SQLite file
* WAL + secure-delete enabled
* Tables:

  * `credentials` — encrypted payloads + cryptographic and usage metadata
  * `vault` — encrypted master secret, salt, Curve25519 public key

### 🔎 Adaptive Search

* Fuzzy search across label + user
* Weighted scoring:

  * label matching
  * user matching
  * recency (time decay)
  * frequency (logarithmic)
* Tie breakers: usage > label lexicographic
* Constant-time Levenshtein for normalization

---

# 🚀 Development

## 1. Clone the project

```bash
git clone https://github.com/gitKashish/kosh.git
cd kosh
```

## 2. Build

```bash
go build
```

or use the included build script:

```bash
./build.sh
```

The `kosh` binary will be generated in the project root.

## 3. Run tests (coming soon)

When tests are added:

```bash
go test ./...
```

---

# 🤝 Contributing

Contributions are welcome! Areas that need help include:

* Improving test coverage
* Performance tuning search / database IO
* Better error messages & user experience
* Security audits & design review
* Documentation contributions (architecture, diagrams, deeper cryptography explanations)

Before submitting a PR:

1. Ensure the code passes `go vet` and builds cleanly
2. Keep PRs small and focused
3. Follow the existing project structure and naming patterns
4. Do **not** introduce unnecessary dependencies

---

# 🔐 Security Model (Developer Notes)

* Master password cannot currently be changed after vault initialization
* Losing the master password **permanently locks the vault**
* No backdoor, recovery mechanism, or plaintext fallback
* Secrets and sensitive buffers should be overridden when possible
* SQLite **secure-delete** ensures deleted rows cannot be recovered

For detailed design docs, cryptography explanations, and diagrams:

👉 **[Encryption Architecture](https://kosh.plutolab.org/technical/encryption)**

---
