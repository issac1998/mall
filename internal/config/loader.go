package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var (
	// GlobalConfig holds the global configuration instance
	GlobalConfig *Config
)

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	config := &Config{}

	// Set up viper
	v := viper.New()
	
	// Set config file path
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Default config paths
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath("../configs")
		v.AddConfigPath("../../configs")
		v.AddConfigPath("/etc/seckill")
		v.AddConfigPath("$HOME/.seckill")
	}

	// Environment variables
	v.SetEnvPrefix("SECKILL")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, use defaults and environment variables
			fmt.Printf("Config file not found, using defaults and environment variables\n")
		} else {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		fmt.Printf("Using config file: %s\n", v.ConfigFileUsed())
	}

	// Load environment-specific config
	env := os.Getenv("SECKILL_ENV")
	if env == "" {
		env = "dev"
	}
	
	envConfigFile := fmt.Sprintf("config.%s.yaml", env)
	envConfigPath := filepath.Join(filepath.Dir(v.ConfigFileUsed()), envConfigFile)
	
	if _, err := os.Stat(envConfigPath); err == nil {
		envViper := viper.New()
		envViper.SetConfigFile(envConfigPath)
		if err := envViper.ReadInConfig(); err == nil {
			// Merge environment-specific config
			if err := envViper.Unmarshal(config); err != nil {
				return nil, fmt.Errorf("failed to unmarshal env config: %w", err)
			}
			fmt.Printf("Loaded environment config: %s\n", envConfigPath)
		}
	}

	// Unmarshal config
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set defaults
	config.SetDefaults()

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Set global config
	GlobalConfig = config

	return config, nil
}

// MustLoadConfig loads configuration and panics on error
func MustLoadConfig(configPath string) *Config {
	config, err := LoadConfig(configPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}
	return config
}

// GetConfig returns the global configuration instance
func GetConfig() *Config {
	if GlobalConfig == nil {
		panic("Config not loaded. Call LoadConfig first.")
	}
	return GlobalConfig
}

// ReloadConfig reloads the configuration
func ReloadConfig() error {
	if GlobalConfig == nil {
		return fmt.Errorf("config not initialized")
	}
	
	configPath := viper.ConfigFileUsed()
	newConfig, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}
	
	GlobalConfig = newConfig
	return nil
}

// WatchConfig watches for configuration file changes
func WatchConfig(callback func()) {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Printf("Config file changed: %s\n", e.Name)
		if err := ReloadConfig(); err != nil {
			fmt.Printf("Failed to reload config: %v\n", err)
			return
		}
		if callback != nil {
			callback()
		}
	})
}

// GetEnv returns environment variable value with fallback
func GetEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// GetEnvBool returns environment variable as boolean with fallback
func GetEnvBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true"
	}
	return fallback
}

// GetEnvInt returns environment variable as integer with fallback
func GetEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return fallback
}

// IsDevelopment returns true if running in development mode
func IsDevelopment() bool {
	env := GetEnv("SECKILL_ENV", "dev")
	return env == "dev" || env == "development"
}

// IsProduction returns true if running in production mode
func IsProduction() bool {
	env := GetEnv("SECKILL_ENV", "dev")
	return env == "prod" || env == "production"
}

// IsTest returns true if running in test mode
func IsTest() bool {
	env := GetEnv("SECKILL_ENV", "dev")
	return env == "test"
}