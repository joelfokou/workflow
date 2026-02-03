package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/joelfokou/workflow/internal/config"
	"github.com/joelfokou/workflow/internal/executor"
	"github.com/joelfokou/workflow/internal/logger"
	"github.com/joelfokou/workflow/internal/run"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var resumeCmd = &cobra.Command{
	Use:   "resume <run_id>",
	Short: "Resume a failed workflow run",
	Long:  "Resume a failed workflow run from the point of failure",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID := args[0]

		// Initialise run store
		dbPath := config.C.Paths.Database
		store, err := run.NewStore(dbPath)
		if err != nil {
			logger.L().Error("failed to initialise run store", zap.Error(err))
			return fmt.Errorf("failed to initialise run store: %w", err)
		}
		defer store.Close()

		// Verify run exists
		workflowRun, err := store.Load(runID)
		if err != nil {
			logger.L().Error("run not found", zap.String("run_id", runID), zap.Error(err))
			return fmt.Errorf("run '%s' not found: %w", runID, err)
		}

		// Check if the run is in a resumable state
		if workflowRun.Status != run.StatusFailed {
			logger.L().Warn("workflow run is not in a resumable state", zap.String("run_id", runID), zap.String("status", string(workflowRun.Status)))
			return fmt.Errorf("workflow run '%s' is not in a resumable state (current status: %s)", runID, workflowRun.Status)
		}

		// Setup context with cancellation
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Handle Ctrl+C
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt)
		go func() {
			<-sigChan
			fmt.Println("\n✖ Received interrupt. Cancelling workflow...")
			cancel()
		}()

		// Create executor and resume workflow
		executor := executor.NewExecutor(store)
		err = executor.Resume(ctx, workflowRun)
		if err != nil {
			logger.L().Error("failed to resume workflow run", zap.String("run_id", runID), zap.Error(err))
			return fmt.Errorf("failed to resume workflow run '%s': %w", runID, err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(resumeCmd)
}
