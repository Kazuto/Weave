package config

type Config struct {
	Branch BranchConfig `yaml:"branch"`
}

type BranchConfig struct {
	MaxLength    int               `yaml:"max_length"`
	DefaultType  string            `yaml:"default_type"`
	Types        map[string]string `yaml:"types"`
	Sanitization SanitizationConfig `yaml:"sanitization"`
}

type SanitizationConfig struct {
	Separator     string `yaml:"separator"`
	Lowercase     bool   `yaml:"lowercase"`
	RemoveUmlauts bool   `yaml:"remove_umlauts"`
}

type ConfigManager interface {
	Load() (*Config, error)
	EnsureExists() error
	GetConfigPath() string
	Validate(*Config) error
}

func NewConfigManager() ConfigManager {
	return NewFileConfigManager()
}
