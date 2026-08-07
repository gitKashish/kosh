package storage

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"git.plutolab.org/plutolab/kosh/internal/config"
	"git.plutolab.org/plutolab/kosh/internal/model"
	_ "modernc.org/sqlite"
)

type Store interface {
	// Vault functions
	GetVaultInfo() (*model.Vault, error)
	InitializeVault(vault model.Vault) error
	IsVaultInitialized() (bool, error)

	// Credential functions
	AddCredential(credential *model.Credential) error
	DeleteCredentialById(id int) error
	GetAllCredentials() ([]model.Credential, error)
	GetCredentialById(id int) (*model.Credential, error)
	GetCredentialByLabelAndUser(label, user string) (*model.Credential, error)
	SearchCredentialByLabelOrUser(label, user string) ([]model.CredentialSummary, error)
	UpdateCredential(credential *model.Credential) error
	UpdateCredentialAccessCount(id, delta int, accessTime time.Time) error

	// Data Store functions
	CloseStore() error
}

type VaultStore struct {
	db *sql.DB
}

// InitializeStore establishes connection with database
func InitializeStore(cfg *config.Config) (Store, error) {
	userDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	profilesPath := filepath.Join(userDir, ".kosh", "profiles")
	if err := os.MkdirAll(profilesPath, 0700); err != nil {
		return nil, fmt.Errorf("create profile directory %s: %w", profilesPath, err)
	}

	dbFilePath := filepath.Join(
		profilesPath,
		fmt.Sprintf("%s.db", cfg.ActiveProfile),
	)

	db, err := sql.Open("sqlite", dbFilePath)

	if err != nil {
		return nil, fmt.Errorf("open vault %s: %w", dbFilePath, err)
	}

	// Set pragmas for this connection
	if err := initDatabase(db); err != nil {
		return nil, err
	}

	vault := &VaultStore{db}
	if err := vault.RunMigrations(); err != nil {
		return nil, err
	}

	slog.Debug("store intialized", "store", vault)

	return vault, nil
}

// CloseStore closes existing connection to the database
func (v *VaultStore) CloseStore() error {
	if v != nil {
		if err := v.db.Close(); err != nil {
			// callers discard this error, so this is the only record of it
			slog.Debug("failed to close database connection", "error", err)
			return err
		}
	}
	return nil
}

func initDatabase(db *sql.DB) error {
	pragma := `PRAGMA journal_mode=WAL;  
				PRAGMA synchronous=NORMAL; 
				PRAGMA foreign_keys=ON;    
				PRAGMA temp_store=MEMORY;  
				PRAGMA secure_delete=ON;   
				PRAGMA trusted_schema=OFF;`

	if _, err := db.Exec(pragma); err != nil {
		return err
	}
	return nil
}
