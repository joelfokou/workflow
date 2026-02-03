// Package cmd implements the command-line interface for the application.
package cmd

import (
	"fmt"
	"os"

	"github.com/joelfokou/workflow/internal/config"
	"github.com/joelfokou/workflow/internal/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	verbose    bool
	logLevel   string
	configFile string
)

var rootCmd = &cobra.Command{
	Use:   "wf",
	Short: "wf - lightweight local workflow runner",
	Long: `wf is a minimal, deterministic workflow orchestrator.

A simple yet powerful tool for defining and executing task workflows locally.
Workflows are defined in TOML format and executed in topological order.`,
	Version: "0.1.0", // Set this from build flags
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		logger.Sync()
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// initConfig initialises configuration, logger, and validates setup.
func initConfig() {
	// Load configuration with optional override
	if err := config.Load(configFile); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	if config.C.LogLevel != "" {
		logLevel = config.C.LogLevel
	}

	// Override log level if verbose flag set
	if verbose {
		logLevel = "debug"
	}

	// Initialise logger with configured level
	loggerConfig := logger.Config{
		Level:      logLevel,
		Format:     "console",
		OutputFile: config.C.Paths.LogsFile,
	}

	if err := logger.Init(loggerConfig); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialise logger: %v\n", err)
		os.Exit(1)
	}

	logger.L().Debug("configuration loaded", zap.String("config_path", configFile))
	logger.L().Debug("logger initialised", zap.String("level", logLevel))
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level: debug, info, warn, error")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Path to config file (overrides defaults)")

	// Help is called by default for -h/--help
	rootCmd.CompletionOptions.DisableDefaultCmd = false
}
