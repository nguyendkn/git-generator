package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nguyendkn/git-generator/pkg/types"
	"github.com/spf13/viper"
)

const (
	ConfigFileName = "git-generator"
	ConfigFileType = "yaml"
)

// Manager handles configuration loading and management
type Manager struct {
	config *types.Config
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	return &Manager{}
}

// Load loads configuration from various sources
func (m *Manager) Load() (*types.Config, error) {
	// Set up Viper
	viper.SetConfigName(ConfigFileName)
	viper.SetConfigType(ConfigFileType)

	// Add config paths
	viper.AddConfigPath(".")                    // Current directory
	viper.AddConfigPath("$HOME/.git-generator") // Home directory
	viper.AddConfigPath("/etc/git-generator/")  // System directory

	// Set environment variable prefix
	viper.SetEnvPrefix("GIT_GENERATOR")
	viper.AutomaticEnv()

	// Set defaults
	m.setDefaults()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// If there's a config file but it's malformed, remove it silently
			configPath, pathErr := m.GetConfigPath()
			if pathErr == nil {
				if _, statErr := os.Stat(configPath); statErr == nil {
					// Config file exists but is malformed, remove it
					os.Remove(configPath)
				}
			}
		}
		// Config file not found or malformed is OK, we'll use defaults and env vars
	}

	// Unmarshal into struct
	config := &types.Config{}
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := m.validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	m.config = config
	return config, nil
}

// setDefaults sets default configuration values
func (m *Manager) setDefaults() {
	// Gemini defaults
	viper.SetDefault("gemini.model", "gemini-1.5-flash")
	viper.SetDefault("gemini.temperature", 0.3)
	viper.SetDefault("gemini.max_tokens", 1000)

	// Git defaults
	viper.SetDefault("git.max_diff_size", 10000)
	viper.SetDefault("git.include_staged", true)
	viper.SetDefault("git.ignore_files", []string{
		"*.log",
		"*.tmp",
		"node_modules/*",
		".git/*",
		"vendor/*",
		"target/*",
		"build/*",
		"dist/*",
	})

	// Output defaults
	viper.SetDefault("output.style", "conventional")
	viper.SetDefault("output.max_lines", 100)
	viper.SetDefault("output.dry_run", false)
}

// validateConfig validates the loaded configuration
func (m *Manager) validateConfig(config *types.Config) error {
	// Validate Gemini config
	if config.Gemini.APIKey == "" {
		// Try to get from environment
		if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey != "" {
			config.Gemini.APIKey = apiKey
		} else {
			return fmt.Errorf("gemini API key is required (set GEMINI_API_KEY environment variable or add to config file)")
		}
	}

	if config.Gemini.Temperature < 0 || config.Gemini.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}

	if config.Gemini.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be positive")
	}

	// Validate Git config
	if config.Git.MaxDiffSize <= 0 {
		return fmt.Errorf("max_diff_size must be positive")
	}

	// Validate Output config
	validStyles := map[string]bool{
		"conventional": true,
		"simple":       true,
		"detailed":     true,
	}
	if !validStyles[config.Output.Style] {
		return fmt.Errorf("invalid output style: %s (must be one of: conventional, simple, detailed)", config.Output.Style)
	}

	if config.Output.MaxLines <= 0 {
		return fmt.Errorf("max_lines must be positive")
	}

	return nil
}

// Save saves the current configuration to file
func (m *Manager) Save() error {
	if m.config == nil {
		return fmt.Errorf("no configuration loaded")
	}

	// Get config directory
	configDir, err := m.getConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Set config file path
	configFile := filepath.Join(configDir, ConfigFileName+"."+ConfigFileType)
	viper.SetConfigFile(configFile)

	// Write config
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// getConfigDir returns the configuration directory path
func (m *Manager) getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".git-generator"), nil
}

// CreateDefaultConfig creates a default configuration file
func (m *Manager) CreateDefaultConfig() error {
	configDir, err := m.getConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configFile := filepath.Join(configDir, ConfigFileName+"."+ConfigFileType)

	// Check if config file already exists
	if _, err := os.Stat(configFile); err == nil {
		return fmt.Errorf("config file already exists: %s", configFile)
	}

	// Create default config content as a simple string
	defaultConfigContent := []byte(`gemini:
  api_key: ""
  model: "gemini-1.5-flash"
  temperature: 0.5
  max_tokens: 1000

git:
  max_diff_size: 10000
  include_staged: true
  ignore_files:
    - "*.log"
    - "*.tmp"
    - "node_modules/*"
    - ".git/*"
    - "vendor/*"
    - "target/*"
    - "build/*"
    - "dist/*"

output:
  style: "conventional"
  max_lines: 100
  dry_run: false
`)

	// Write the config file with explicit UTF-8 encoding
	if err := os.WriteFile(configFile, defaultConfigContent, 0644); err != nil {
		return fmt.Errorf("failed to write default config: %w", err)
	}

	fmt.Printf("Default configuration created at: %s\n", configFile)
	fmt.Println("Please edit the file to add your Gemini API key.")
	return nil
}

// GetConfigPath returns the path to the configuration file
func (m *Manager) GetConfigPath() (string, error) {
	configDir, err := m.getConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, ConfigFileName+"."+ConfigFileType), nil
}

// UpdateAPIKey updates the Gemini API key in the configuration
func (m *Manager) UpdateAPIKey(apiKey string) error {
	if m.config == nil {
		return fmt.Errorf("no configuration loaded")
	}

	m.config.Gemini.APIKey = apiKey
	viper.Set("gemini.api_key", apiKey)
	return m.Save()
}

// GetConfig returns the current configuration
func (m *Manager) GetConfig() *types.Config {
	return m.config
}

// SetConfig sets the configuration
func (m *Manager) SetConfig(config *types.Config) {
	m.config = config
}

// LoadOrCreate loads existing config or creates a default one
func (m *Manager) LoadOrCreate() (*types.Config, error) {
	config, err := m.Load()
	if err != nil {
		// If it's a missing API key error, try to create default config
		if err.Error() == "Gemini API key is required (set GEMINI_API_KEY environment variable or add to config file)" {
			if createErr := m.CreateDefaultConfig(); createErr != nil {
				return nil, fmt.Errorf("failed to create default config: %w", createErr)
			}
			return nil, fmt.Errorf("configuration file created. Please add your Gemini API key and try again")
		}
		return nil, err
	}
	return config, nil
}
