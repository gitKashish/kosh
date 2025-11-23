# Kosh — Secure Password Manager

**Kosh** is a fast, secure, local-first command-line password manager.
It stores credentials in an encrypted SQLite vault using modern cryptographic
primitives such as **Curve25519**, **ChaCha20-Poly1305**, and **Argon2id**.

Kosh is lightweight, dependency-free, cross-platform, and fully offline.

---

# 📑 Index

1. [Features](#features)
2. [Installation](#installation)

   * [From Source](#from-source)
   * [Using Go Install](#using-go-install-recommended)
3. [Command Reference](#command-reference)
4. [Usage](#usage)

   * [Show Help](#show-help)
   * [Initialize the Vault](#initialize-the-vault)
   * [Add Credential](#add-credential)
   * [Retrieve Credential](#retrieve-credential)
   * [List Credentials](#list-credentials)
   * [Delete Credential](#delete-credential)
   * [Adaptive Search](#adaptive-search)

     * [Single-Argument Search](#single-argument-search)
     * [Two-Argument Search](#two-argument-search)
     * [Scoring Algorithm](#adaptive-search-scoring-algorithm)
5. [SQLite Storage](#sqlite-storage)

---

## Features

* 🔐 End-to-end encryption for all credentials
* 🗄️ Local, portable SQLite vault
* 🔑 Master-password–derived encryption key (Argon2id)
* ⚡ **Adaptive fuzzy search** with scoring, recency, and frequency weighting
* 🧩 Minimum external dependencies
* 🎯 Simple, fast CLI interface
* 📦 Cross-platform (Linux, macOS, Windows)

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

| Command                   | Description                                         |
| ------------------------- | --------------------------------------------------- |
| `kosh help`               | Show help and usage                                 |
| `kosh init`               | Initialize a new encrypted vault                    |
| `kosh add`                | Add or update a credential                          |
| `kosh list`               | List credentials (with optional filters)            |
| `kosh get <label> <user>` | Retrieve and decrypt a credential                   |
| `kosh search <query>`     | Adaptive search for the closest matching credential |
| `kosh <query>`            | Shorthand for `kosh search <query>`                 |
| `kosh delete <id>`        | Delete a credential from the vault                  |

---

## Usage

### Show Help

```bash
kosh help
```

### Initialize the Vault

```bash
kosh init
```

Prompts for a master password, derives a secure key, and sets up the vault.

### Add Credential

```bash
kosh add
```

Interactive prompt for label, username, password, and optional notes.

### Retrieve Credential

```bash
kosh get github pluto
```

Decrypts and prints the stored password securely.

### List Credentials

```bash
kosh list                               # List all entries
kosh list pluto                         # Search users containing 'pluto'
kosh list --label github                # Match labels containing 'github'
kosh list --user pluto                  # Match a user
kosh list --label github --user pluto   # Combine filters
```

### Delete Credential

```bash
kosh delete 101
```

Removes an entry safely from the vault.

---

## Adaptive Search

### Single-Argument Search

Searches **across both label and user**:

```bash
kosh search git
# or simply
kosh git
```

Matches examples like:

* label: `"github"`
* user: `"gitlab-user"`
* label: `"work-git"`

### Two-Argument Search

Treats the first argument as **label query** and the second as **user query**:

```bash
kosh search mail personal
kosh search github pluto
```

Equivalent to fuzzy-matching both fields.

### Adaptive Search Scoring Algorithm

Results are ranked by a weighted combination of:

* Levenshtein distance
* Prefix/substring boosts
* Recency (recently used entries score higher)
* Frequency (frequently used entries score higher)

The top-scoring entry is returned first.

---

## SQLite Storage

Kosh uses a compact, optimized SQLite layout:

* **Credentials table** (encrypted payloads + metadata)
* **Usage metadata** for adaptive search

  * last access time
  * access count
* **WAL mode** enabled automatically
* **Safe delete mode** to avoid accidental cross-page leakage

---