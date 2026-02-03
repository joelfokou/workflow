package cmd

import (
	"fmt"
	"os"

	"github.com/joelfokou/workflow/internal/config"
	"github.com/joelfokou/workflow/internal/logger"
	"github.com/joelfokou/workflow/internal/run"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// logsCmd shows logs for a specific run or task within a run. It queries the database for task information and reads the corresponding log files.
var logsCmd = &cobra.Command{
	Use:   "logs <run_id> [task]",
	Short: "Show logs for a run or specific task",
	Long:  "Display logs for a workflow run or a specific task within that run",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID := args[0]
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

		// Load all tasks for this run
		tasks, err := store.LoadTaskRuns(runID)
		if err != nil {
			logger.L().Error("failed to load tasks for run", zap.String("run_id", runID), zap.Error(err))
			return fmt.Errorf("failed to load tasks for run '%s': %w", runID, err)
		}

		if len(tasks) == 0 {
			fmt.Printf("No tasks found for run '%s'\n", runID)
			return nil
		}

		if len(args) == 2 {
			return showTaskLogs(workflowRun, tasks, args[1])
		}

		return showRunLogs(workflowRun, tasks)
	},
}

// showRunLogs displays logs for all tasks in a run.
func showRunLogs(workflowRun *run.WorkflowRun, tasks []run.TaskRun) error {
	fmt.Printf("=== Logs for Run '%s' (%s) ===\n\n", workflowRun.ID, workflowRun.Workflow)

	for _, task := range tasks {
		fmt.Printf("[%s] Status: %s | Attempts: %d | Exit Code: ", task.Name, task.Status, task.Attempts)

		if task.ExitCode.Valid {
			fmt.Printf("%d\n", task.ExitCode.Int64)
		} else {
			fmt.Printf("N/A\n")
		}

		if task.LogPath != "" {
			content, err := os.ReadFile(task.LogPath)
			if err != nil {
				logger.L().Warn("failed to read task log file",
					zap.String("run_id", workflowRun.ID),
					zap.String("task", task.Name),
					zap.String("file", task.LogPath),
					zap.Error(err),
				)
				fmt.Printf("  (Could not read log file: %v)\n", err)
			} else {
				fmt.Printf("  %s\n\n", content)
			}
		} else {
			fmt.Printf("  (No logs recorded)\n\n")
		}

		if task.LastError != "" {
			fmt.Printf("  Last Error: %s\n\n", task.LastError)
		}
	}

	logger.L().Info("displayed logs for run", zap.String("run_id", workflowRun.ID))

	return nil
}

// showTaskLogs displays logs for a specific task.
func showTaskLogs(workflowRun *run.WorkflowRun, tasks []run.TaskRun, taskName string) error {
	var targetTask *run.TaskRun

	for i := range tasks {
		if tasks[i].Name == taskName {
			targetTask = &tasks[i]
			break
		}
	}

	if targetTask == nil {
		logger.L().Error("task not found in run", zap.String("run_id", workflowRun.ID), zap.String("task", taskName))
		return fmt.Errorf("task '%s' not found in run '%s'", taskName, workflowRun.ID)
	}

	fmt.Printf("=== Logs for Task '%s' in Run '%s' ===\n\n", taskName, workflowRun.ID)
	fmt.Printf("Status: %s\n", targetTask.Status)
	fmt.Printf("Attempts: %d\n", targetTask.Attempts)
	fmt.Printf("Started: %s\n", targetTask.StartedAt.Format("2006-01-02 15:04:05"))

	if targetTask.EndedAt.Valid {
		fmt.Printf("Ended: %s\n", targetTask.EndedAt.Time.Format("2006-01-02 15:04:05"))
		duration := targetTask.EndedAt.Time.Sub(targetTask.StartedAt)
		fmt.Printf("Duration: %.2fs\n", duration.Seconds())
	}

	if targetTask.ExitCode.Valid {
		fmt.Printf("Exit Code: %d\n", targetTask.ExitCode.Int64)
	}

	fmt.Println("\n--- Output ---")

	if targetTask.LogPath != "" {
		content, err := os.ReadFile(targetTask.LogPath)
		if err != nil {
			logger.L().Error("failed to read task log file",
				zap.String("run_id", workflowRun.ID),
				zap.String("task", taskName),
				zap.String("file", targetTask.LogPath),
				zap.Error(err),
			)
			return fmt.Errorf("could not read log file for task '%s': %w", taskName, err)
		}
		fmt.Println(string(content))
	} else {
		fmt.Println("(No logs recorded)")
	}

	if targetTask.LastError != "" {
		fmt.Println("\n--- Error ---")
		fmt.Println(targetTask.LastError)
	}

	logger.L().Info("displayed logs for task",
		zap.String("run_id", workflowRun.ID),
		zap.String("task", taskName),
	)

	return nil
}

func init() {
	rootCmd.AddCommand(logsCmd)
}
