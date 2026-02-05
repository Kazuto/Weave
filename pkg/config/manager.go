package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	appName    = "weave"
	configFile = "config.yaml"
)

type FileConfigManager struct {
	configPath string
}

func NewFileConfigManager() *FileConfigManager {
	return &FileConfigManager{
		configPath: getConfigPath(),
	}
}

func getConfigPath() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = "."
		}
		configDir = filepath.Join(homeDir, ".config")
	}

	return filepath.Join(configDir, appName, configFile)
}

func (m *FileConfigManager) GetConfigPath() string {
	return m.configPath
}

func (m *FileConfigManager) EnsureExists() error {
	if _, err := os.Stat(m.configPath); err == nil {
		return nil
	}

	configDir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return fmt.Errorf("failed to create configuration directory: %w", err)
	}

	data, err := yaml.Marshal(GetDefaultConfig())
	if err != nil {
		return fmt.Errorf("failed to generate default configuration: %w", err)
	}

	content := append([]byte("# Weave configuration file\n\n"), data...)

	if err := os.WriteFile(m.configPath, content, 0600); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	return nil
}

func (m *FileConfigManager) Load() (*Config, error) {
	if err := m.EnsureExists(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML configuration: %w", err)
	}

	result := ValidateAndFix(&config)
	if !result.IsValid() {
		return nil, result.Errors[0]
	}

	if result.Fixed && len(result.Warnings) > 0 {
		for _, warning := range result.Warnings {
			fmt.Fprintf(os.Stderr, "config warning: %s\n", warning)
		}
	}

	return &config, nil
}

func (m *FileConfigManager) Validate(config *Config) error {
	return ValidateStrict(config)
}
