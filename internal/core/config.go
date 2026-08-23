package core

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config represents the persisted preferences for Cogni installations
type Config struct {
	SelectedHarnesses []string `json:"selected_harnesses"`
}

// LoadConfig reads the configuration file from ~/.cogni/config.json
func LoadConfig(homeDir string) (*Config, error) {
	cfgPath := filepath.Join(homeDir, ".cogni", "config.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{SelectedHarnesses: []string{}}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.SelectedHarnesses == nil {
		cfg.SelectedHarnesses = []string{}
	}

	return &cfg, nil
}

// SaveConfig persists the configuration to ~/.cogni/config.json
func SaveConfig(homeDir string, cfg *Config) error {
	dir := filepath.Join(homeDir, ".cogni")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	cfgPath := filepath.Join(dir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cfgPath, append(data, '\n'), 0644)
}
