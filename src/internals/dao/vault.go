package dao

import (
	"database/sql"
	"fmt"

	"github.com/gitKashish/kosh/src/internals/model"
)

// IsVaultInitialized checks if vault exists and has a valid record in it
func IsVaultInitialized() (bool, error) {
	if db == nil {
		return false, fmt.Errorf("database connection not initialized")
	}

	// check if vault table exists
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name='vault'`
	var tableName string
	err := db.QueryRow(query).Scan(&tableName)
	if err == sql.ErrNoRows {
		return false, nil
	}

	if err != nil {
		fmt.Println("[Error] unable to fetch table name from database")
		return false, err
	}

	// check if vault has a valid entry
	var count int
	query = `SELECT COUNT(*) FROM vault`
	err = db.QueryRow(query).Scan(&count)
	if err != nil {
		fmt.Println("[Error] failed to count the number of records in vault table")
		return false, err
	}
	fmt.Printf("[Debug] found %d vault\n", count)
	return count > 0, nil
}

// InitializeVault adds a valid record to the vault config
func InitializeVault(vault model.Vault) error {
	initialized, err := IsVaultInitialized()
	if err != nil {
		fmt.Println("[Error] failed to check vault initialization")
		return err
	}

	if initialized {
		fmt.Println("[Info] vault already initialized")
		return fmt.Errorf("vault already initialized")
	}

	// Start transaction
	transaction, err := db.Begin()
	if err != nil {
		fmt.Println("[Error] failed to start transaction")
		return err
	}
	defer transaction.Rollback()

	// create vault table
	_, err = transaction.Exec(`
		CREATE TABLE IF NOT EXISTS vault (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			public_key TEXT NOT NULL,
			nonce TEXT NOT NULL,
			secret TEXT NOT NULL,
			salt TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)

	if err != nil {
		fmt.Println("[Error] failed to create vault table")
		return err
	}

	// create update trigger to keep vault updated_at timestamp up-to-date
	_, err = transaction.Exec(`
		CREATE TRIGGER IF NOT EXISTS update_vault_timestamp
		AFTER UPDATE ON vault
		FOR EACH ROW
		BEGIN
			UPDATE vault SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END
	`)

	if err != nil {
		fmt.Println("[Error] failed to create update trigger on vault")
		return err
	}

	// create credentials table
	_, err = transaction.Exec(`
		CREATE TABLE IF NOT EXISTS credentials (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			label TEXT NOT NULL,
			user TEXT NOT NULL,
			secret TEXT NOT NULL,
			ephemeral TEXT NOT NULL,
			nonce TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(label, user)
		)
	`)

	if err != nil {
		fmt.Println("[Error] failed to create credentials table")
		return err
	}

	// create update trigger to keep credential updated_at timestamp up-to-date
	_, err = transaction.Exec(`
		CREATE TRIGGER IF NOT EXISTS update_credential_timestamp
		AFTER UPDATE ON credentials
		FOR EACH ROW
		BEGIN
			UPDATE credentials SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END
	`)

	if err != nil {
		fmt.Println("[Error] failed to create update trigger on credentials")
		return err
	}

	// insert vault secret
	stmt, err := transaction.Prepare(`
		INSERT INTO vault (public_key, nonce, secret, salt)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		fmt.Println("[Error] failed to prepare vault insert statement")
		return err
	}

	_, err = stmt.Exec(vault.PublicKey, vault.Nonce, vault.Secret, vault.Salt)
	if err != nil {
		fmt.Println("[Error] failed to insert vault secret")
		return err
	}

	// Commit transaction
	if err := transaction.Commit(); err != nil {
		fmt.Println("[Error] failed to commit transaction")
		return err
	}

	return nil
}

func GetVaultInfo() (*model.Vault, error) {
	initialized, err := IsVaultInitialized()
	if err != nil {
		fmt.Println("[Error] error checking vault initialized status")
		return nil, err
	}

	if !initialized {
		fmt.Println("[Error] vault is not initialized")
		return nil, fmt.Errorf("vault is not initialized")
	}

	// get vault info from database
	var vault model.Vault

	err = db.QueryRow(`
		SELECT public_key, secret, nonce, salt FROM vault
	`).Scan(&vault.PublicKey, &vault.Secret, &vault.Nonce, &vault.Salt)

	if err == sql.ErrNoRows {
		fmt.Println("[Error] vault not initialized")
		return nil, err
	}

	if err != nil {
		fmt.Println("[Error] failed to get vault info")
		return nil, err
	}

	return &vault, nil
}
