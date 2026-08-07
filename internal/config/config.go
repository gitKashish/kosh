package config

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
)

type Config struct {
	ActiveProfile string `json:"active_profile"`
}

var defaultCfg = Config{
	ActiveProfile: "default",
}

func getConfigPath() (string, error) {
	basePath, err := getBaseDirPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(basePath, "config.json"), nil
}

func getBaseDirPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".kosh"), nil
}

// Save overrides the existing config file.
// If the file structure does not exist, it
// creates it.
func Save(cfg *Config) error {
	cfgPath, err := getConfigPath()
	if err != nil {
		return err
	}

	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	dirPath := filepath.Dir(cfgPath)
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return err
	}

	if err := os.WriteFile(cfgPath, b, 0600); err != nil {
		return err
	}
	slog.Debug("config saved", "path", cfgPath)
	return nil
}

// Load fetches the current user config.
// If a config does not exist, it creates
// a new file with the default config.
func Load() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	// Check if config exists. If not, create a default config file.
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := defaultCfg
		if err := Save(&cfg); err != nil {
			return nil, err
		}

		return &cfg, nil
	}

	// Load existing config
	file, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(file, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
