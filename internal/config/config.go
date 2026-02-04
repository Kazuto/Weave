package config

type Config struct{}

type ConfigManager interface {
	Load() (*Config, error)
	EnsureExists() error
	GetConfigPath() string
	Validate(*Config) error
}

func NewConfigManager() ConfigManager {
	return NewFileConfigManager()
}
