// Package config handles the loading and management of application configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Paths struct {
	Workflows string `mapstructure:"workflows"`
	Logs      string `mapstructure:"logs"`
	Database  string `mapstructure:"database"`
	LogsFile  string `mapstructure:"logs_file"`
}

type Config struct {
	LogLevel string `mapstructure:"log_level"`
	Paths    Paths  `mapstructure:"paths"`
}

var C Config

// getDefaultConfigDir returns the default configuration directory for the application.
func getDefaultConfigDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		panic("unable to determine user config dir")
	}
	return filepath.Join(dir, "workflow")
}

// getDefaultDataDir returns the default data directory for the application.
func getDefaultDataDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		panic("unable to determine user cache dir")
	}
	return filepath.Join(dir, "workflow")
}

// DefaultConfig returns the default configuration file content as a string.
func DefaultConfig() string {
	return fmt.Sprintf(`# workflow configuration file
# This file controls global behaviour of the workflow CLI and can be overridden by environment variables or command-line flags.
# Paths should correspond to your OS conventions.
# Modify as needed

paths:
  workflows: %s
  logs: %s
  database: %s
  logs_file: %s

log_level: info
`, filepath.Join(getDefaultDataDir(), "workflows"),
		filepath.Join(getDefaultDataDir(), "logs"),
		filepath.Join(getDefaultDataDir(), "workflow.db"),
		filepath.Join(getDefaultDataDir(), "logs", "workflow.log"))
}

func ConfigFile() string {
	return filepath.Join(getDefaultConfigDir(), "config.yaml")
}

// Load reads configuration from file and environment variables into the Config struct.
func Load(configFilePath ...string) error {
	// Defaults
	dataDir := getDefaultDataDir()
	viper.SetDefault("paths.workflows", filepath.Join(dataDir, "workflows"))
	viper.SetDefault("paths.logs", filepath.Join(dataDir, "logs"))
	viper.SetDefault("paths.database", filepath.Join(dataDir, "workflow.db"))
	viper.SetDefault("paths.logs_file", filepath.Join(dataDir, "logs", "workflow.log"))

	// Environment variables
	viper.SetEnvPrefix("WF")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Config file
	cfgDir := getDefaultConfigDir()
	viper.AddConfigPath(cfgDir)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Use optional config file if provided
	if len(configFilePath) > 0 && configFilePath[0] != "" {
		viper.SetConfigFile(configFilePath[0])
	}

	// Ignore error if config file doesn't exist
	_ = viper.ReadInConfig()

	return viper.Unmarshal(&C)
}
