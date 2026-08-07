package crypto

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"os"
)

func OverwriteFile(filepath string) error {
	// open file to overwrite
	file, err := os.OpenFile(filepath, os.O_WRONLY, 0600)
	if err != nil {
		slog.Debug("failed to open file to overwrite", "error", err)
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		if os.IsExist(err) {
			slog.Debug("file to be overwritten does not exist", "error", err)
		} else {
			slog.Debug("failed to get file stat", "error", err)
		}
		return err
	}

	size := stat.Size()
	if size > 0 {
		// Generate random byte
		randomBytes := make([]byte, size)
		if _, err := rand.Read(randomBytes); err != nil {
			return fmt.Errorf("failed to generate random bytes")
		}

		// Overwrite file contents
		if _, err := file.WriteAt(randomBytes, 0); err != nil {
			return fmt.Errorf("failed to write bytes onto file")
		}
		// Sync changes to disk
		if err := file.Sync(); err != nil {
			return fmt.Errorf("failed to sync changes to disk")
		}
	}
	return nil
}
