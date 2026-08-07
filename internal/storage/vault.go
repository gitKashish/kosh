package storage

import (
	"database/sql"
	"fmt"
	"log/slog"

	"git.plutolab.org/plutolab/kosh/internal/model"
)

// IsVaultInitialized checks if vault exists and has a valid record in it
func (v *VaultStore) IsVaultInitialized() (bool, error) {
	if v.db == nil {
		return false, fmt.Errorf("database connection not initialized")
	}

	// check if vault table exists
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name='vault'`
	var tableName string
	err := v.db.QueryRow(query).Scan(&tableName)
	if err == sql.ErrNoRows {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("check vault table: %w", err)
	}

	// check if vault has a valid entry
	var count int
	query = `SELECT COUNT(*) FROM vault`
	err = v.db.QueryRow(query).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("count vault records: %w", err)
	}
	slog.Debug("found vault", "count", count)
	return count > 0, nil
}

// InitializeVault adds a valid record to the vault config
func (v *VaultStore) InitializeVault(vault model.Vault) error {
	// Start transaction
	transaction, err := v.db.Begin()
	if err != nil {
		return fmt.Errorf("begin vault transaction: %w", err)
	}
	defer transaction.Rollback()

	// insert vault secret
	_, err = transaction.Exec(`
		INSERT INTO vault (public_key, nonce, secret, salt)
		VALUES (?, ?, ?, ?)`,
		vault.PublicKey, vault.Nonce, vault.Secret, vault.Salt,
	)
	if err != nil {
		return fmt.Errorf("insert vault secret: %w", err)
	}

	// Commit transaction
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit vault transaction: %w", err)
	}

	return nil
}

func (v *VaultStore) GetVaultInfo() (*model.Vault, error) {
	initialized, err := v.IsVaultInitialized()
	if err != nil {
		return nil, err
	}

	if !initialized {
		return nil, fmt.Errorf("vault is not initialized")
	}

	// get vault info from database
	var vault model.Vault

	err = v.db.QueryRow(`
		SELECT public_key, secret, nonce, salt FROM vault
	`).Scan(&vault.PublicKey, &vault.Secret, &vault.Nonce, &vault.Salt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("vault is not initialized: %w", err)
	}

	if err != nil {
		return nil, fmt.Errorf("read vault info: %w", err)
	}

	return &vault, nil
}

var migrations = []string{
	// Migration - 00
	`CREATE TABLE IF NOT EXISTS vault (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		public_key TEXT NOT NULL,
		nonce TEXT NOT NULL,
		secret TEXT NOT NULL,
		salt TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TRIGGER IF NOT EXISTS update_vault_timestamp
		AFTER UPDATE ON vault
		FOR EACH ROW
		BEGIN
			UPDATE vault SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END;

	CREATE TABLE IF NOT EXISTS credentials (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		label TEXT NOT NULL,
		user TEXT NOT NULL,
		access_count NUMBER NOT NULL DEFAULT 0,
		secret TEXT NOT NULL,
		ephemeral TEXT NOT NULL,
		nonce TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		accessed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(label, user)
	);

	CREATE TRIGGER IF NOT EXISTS update_credential_timestamp
		AFTER UPDATE ON credentials
		FOR EACH ROW
		BEGIN
			UPDATE credentials SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END;
	`,
}

func (v *VaultStore) RunMigrations() error {
	// Ensure migration table exists
	_, err := v.db.Exec("CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY)")
	if err != nil {
		return err
	}

	// Get current version
	var currentVersion int
	err = v.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&currentVersion)
	if err != nil {
		return err
	}

	// Apply any remaining migrations
	for i := currentVersion; i < len(migrations); i++ {
		tx, err := v.db.Begin()
		if err != nil {
			return err
		}

		// Execute migration script
		if _, err := tx.Exec(migrations[i]); err != nil {
			tx.Rollback()
			return err
		}

		// Record new version
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, i+1); err != nil {
			tx.Rollback()
			return err
		}

		tx.Commit()
		slog.Debug("applied vault migration", "version", i+1)
	}

	return nil
}
