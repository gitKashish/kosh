package dao

import (
	"database/sql"
	"os"
	"path/filepath"

	"github.com/gitKashish/kosh/src/internals/logger"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// Initialize establishes connection with database
func Initialize() error {
	userDir, err := os.UserHomeDir()
	if err != nil {
		logger.Error("failed to get user home directory")
		return err
	}

	koshDir := filepath.Join(userDir, ".kosh")

	// Create directory if it is not present
	if err := os.MkdirAll(koshDir, 0700); err != nil {
		logger.Error("failed to create .kosh directry")
		return err
	}

	dbFilePath := filepath.Join(koshDir, "kosh.db")

	db, err = sql.Open("sqlite3", dbFilePath)

	if err != nil {
		logger.Error("failed to connect to databse")
	}

	// Set pragmas for this connection
	if err := initDatabase(db); err != nil {
		return err
	}

	return err
}

// Close closes existing connection to the database
func Close() error {
	if db != nil {
		if err := db.Close(); err != nil {
			logger.Error("failed to close database connection")
		}
		return nil
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
