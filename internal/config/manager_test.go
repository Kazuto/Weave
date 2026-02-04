package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewFileConfigManager(t *testing.T) {
	manager := NewFileConfigManager()

	if manager == nil {
		t.Fatal("NewFileConfigManager() returned nil")
	}

	configPath := manager.GetConfigPath()
	if configPath == "" {
		t.Error("NewFileConfigManager() returned empty config path")
	}

	if !strings.Contains(configPath, filepath.Join("weave", "config.yaml")) {
		t.Errorf("Config path should contain 'weave/config.yaml', got: %s", configPath)
	}
}

func TestNewConfigManager(t *testing.T) {
	manager := NewConfigManager()
	if manager == nil {
		t.Error("NewConfigManager() returned nil")
	}

	if _, ok := manager.(*FileConfigManager); !ok {
		t.Error("NewConfigManager() should return a *FileConfigManager")
	}
}

func TestFileConfigManager_GetConfigPath(t *testing.T) {
	expectedPath := "/test/path/config.yaml"
	manager := &FileConfigManager{configPath: expectedPath}

	if got := manager.GetConfigPath(); got != expectedPath {
		t.Errorf("GetConfigPath() = %v, want %v", got, expectedPath)
	}
}

func TestFileConfigManager_EnsureExists(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) string
		expectError bool
		errorCheck  func(error) bool
	}{
		{
			name: "creates config in new directory",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), ".config", "weave", "config.yaml")
			},
			expectError: false,
		},
		{
			name: "creates config when parent directory exists",
			setup: func(t *testing.T) string {
				tempDir := t.TempDir()
				configDir := filepath.Join(tempDir, ".config", "weave")
				if err := os.MkdirAll(configDir, 0750); err != nil {
					t.Fatalf("failed to create test directory: %v", err)
				}
				return filepath.Join(configDir, "config.yaml")
			},
			expectError: false,
		},
		{
			name: "no-op when config already exists",
			setup: func(t *testing.T) string {
				tempDir := t.TempDir()
				configPath := filepath.Join(tempDir, ".config", "weave", "config.yaml")
				manager := &FileConfigManager{configPath: configPath}
				if err := manager.EnsureExists(); err != nil {
					t.Fatalf("failed to create initial config: %v", err)
				}
				return configPath
			},
			expectError: false,
		},
		{
			name: "fails with read-only directory",
			setup: func(t *testing.T) string {
				tempDir := t.TempDir()
				readOnlyDir := filepath.Join(tempDir, "readonly")
				if err := os.MkdirAll(readOnlyDir, 0444); err != nil {
					t.Fatalf("failed to create read-only directory: %v", err)
				}
				return filepath.Join(readOnlyDir, ".config", "weave", "config.yaml")
			},
			expectError: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "failed to create configuration directory")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := tt.setup(t)
			manager := &FileConfigManager{configPath: configPath}

			err := manager.EnsureExists()

			if tt.expectError {
				if err == nil {
					t.Error("EnsureExists() expected error but got none")
				} else if tt.errorCheck != nil && !tt.errorCheck(err) {
					t.Errorf("EnsureExists() error validation failed: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("EnsureExists() unexpected error: %v", err)
			}

			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				t.Errorf("Configuration file was not created at %s", configPath)
			}
		})
	}
}

func TestFileConfigManager_Load(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) *FileConfigManager
		expectError bool
		errorCheck  func(error) bool
	}{
		{
			name: "loads existing valid config",
			setup: func(t *testing.T) *FileConfigManager {
				tempDir := t.TempDir()
				configPath := filepath.Join(tempDir, "config.yaml")
				manager := &FileConfigManager{configPath: configPath}
				if err := manager.EnsureExists(); err != nil {
					t.Fatalf("failed to create config: %v", err)
				}
				return manager
			},
			expectError: false,
		},
		{
			name: "creates default when config missing",
			setup: func(t *testing.T) *FileConfigManager {
				tempDir := t.TempDir()
				configPath := filepath.Join(tempDir, "nonexistent", "config.yaml")
				return &FileConfigManager{configPath: configPath}
			},
			expectError: false,
		},
		{
			name: "fails with invalid YAML",
			setup: func(t *testing.T) *FileConfigManager {
				tempDir := t.TempDir()
				configPath := filepath.Join(tempDir, "config.yaml")

				invalidYAML := "invalid: yaml: content: ["
				if err := os.WriteFile(configPath, []byte(invalidYAML), 0600); err != nil {
					t.Fatalf("failed to write test config: %v", err)
				}
				return &FileConfigManager{configPath: configPath}
			},
			expectError: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "failed to parse YAML configuration")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := tt.setup(t)

			config, err := manager.Load()

			if tt.expectError {
				if err == nil {
					t.Error("Load() expected error but got none")
				} else if tt.errorCheck != nil && !tt.errorCheck(err) {
					t.Errorf("Load() error validation failed: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}

			if config == nil {
				t.Error("Load() returned nil config")
			}
		})
	}
}

func TestFileConfigManager_Validate(t *testing.T) {
	manager := &FileConfigManager{}

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name:    "valid config",
			config:  GetDefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid max_length",
			config: &Config{
				Branch: BranchConfig{
					MaxLength:   5,
					DefaultType: "feature",
					Types:       map[string]string{"feature": "feature"},
					Sanitization: SanitizationConfig{
						Separator: "-",
					},
				},
				Commit: GetDefaultConfig().Commit,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetConfigPath_XDGConfigHome(t *testing.T) {
	tempDir := t.TempDir()

	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	os.Setenv("XDG_CONFIG_HOME", tempDir)

	path := getConfigPath()

	expectedPath := filepath.Join(tempDir, "weave", "config.yaml")
	if path != expectedPath {
		t.Errorf("getConfigPath() = %v, want %v", path, expectedPath)
	}
}

func TestGetConfigPath_FallbackToHome(t *testing.T) {
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	os.Unsetenv("XDG_CONFIG_HOME")

	path := getConfigPath()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	expectedPath := filepath.Join(homeDir, ".config", "weave", "config.yaml")
	if path != expectedPath {
		t.Errorf("getConfigPath() = %v, want %v", path, expectedPath)
	}
}
