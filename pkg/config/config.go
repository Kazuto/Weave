package config

type Config struct {
	Branch BranchConfig `yaml:"branch"`
	Commit CommitConfig `yaml:"commit"`
	PR     PRConfig     `yaml:"pr"`
}

type PRConfig struct {
	DefaultBase   string `yaml:"default_base"`
	DefaultRemote string `yaml:"default_remote"`
	MaxDiff       int    `yaml:"max_diff"`
	Prompt        string `yaml:"prompt"`
}

type CommitConfig struct {
	Ollama OllamaConfig `yaml:"ollama"`
	Types  []string     `yaml:"types"`
	Prompt string       `yaml:"prompt"`
}

type OllamaConfig struct {
	Model       string  `yaml:"model"`
	Host        string  `yaml:"host"`
	Temperature float64 `yaml:"temperature"`
	TopP        float64 `yaml:"top_p"`
	MaxDiff     int     `yaml:"max_diff"`
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
