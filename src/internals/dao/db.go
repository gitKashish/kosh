package dao

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// Initialize establishes connection with database
func Initialize() error {
	userDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("[Error] failed to get user home directory")
		return err
	}

	koshDir := filepath.Join(userDir, ".kosh")

	// Create directory if it is not present
	if err := os.MkdirAll(koshDir, 0700); err != nil {
		fmt.Println("[Error] failed to create .kosh directry")
		return err
	}

	dbFilePath := filepath.Join(koshDir, "kosh.db")

	db, err = sql.Open("sqlite3", dbFilePath)
	if err != nil {
		fmt.Println("[Error] failed to open connection to DB at ", dbFilePath)
	}

	if err := db.Ping(); err == nil {
		fmt.Println("[Debug] connection established")
		return nil
	} else {
		fmt.Println("[Debug] failed to establish connection")
		return err
	}
}

// Close closes existing connection to the database
func Close() error {
	if db != nil {
		if err := db.Close(); err != nil {
			fmt.Println("[Error] failed to close DB connection")
		}
		return nil
	}
	return nil
}
