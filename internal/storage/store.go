package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	"git.plutolab.org/plutolab/kosh/internal/logger"
	"git.plutolab.org/plutolab/kosh/internal/model"
	_ "github.com/mattn/go-sqlite3"
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
func InitializeStore() (Store, error) {
	userDir, err := os.UserHomeDir()
	if err != nil {
		logger.Error("failed to get user home directory")
		return nil, err
	}

	koshDir := filepath.Join(userDir, ".kosh")

	// Create directory if it is not present
	if err := os.MkdirAll(koshDir, 0700); err != nil {
		logger.Error("failed to create .kosh directry")
		return nil, err
	}

	dbFilePath := filepath.Join(koshDir, "kosh.db")

	db, err := sql.Open("sqlite3", dbFilePath)

	if err != nil {
		logger.Error("failed to connect to databse")
	}

	// Set pragmas for this connection
	if err := initDatabase(db); err != nil {
		return nil, err
	}

	return &VaultStore{db}, nil
}

// CloseStore closes existing connection to the database
func (v *VaultStore) CloseStore() error {
	if v != nil {
		if err := v.db.Close(); err != nil {
			logger.Error("failed to close database connection")
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
		logger.Debug("failed to run pragmas")
		return err
	}
	return nil
}
