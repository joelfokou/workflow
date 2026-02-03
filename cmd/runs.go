package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/joelfokou/workflow/internal/config"
	"github.com/joelfokou/workflow/internal/logger"
	"github.com/joelfokou/workflow/internal/run"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	runsWorkflow string
	runsStatus   string
	runsLimit    int
	runsOffset   int
	runsJSON     bool
)

// runsCmd lists workflow runs with filtering and pagination.
var runsCmd = &cobra.Command{
	Use:   "runs",
	Short: "List workflow runs",
	Long:  "List all workflow runs with optional filtering by workflow name and status",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath := config.C.Paths.Database
		store, err := run.NewStore(dbPath)
		if err != nil {
			logger.L().Error("failed to initialise run store", zap.Error(err))
			return fmt.Errorf("failed to initialise run store: %w", err)
		}
		defer store.Close()

		runs, err := store.ListRuns(runsWorkflow, runsStatus, runsLimit, runsOffset)
		if err != nil {
			logger.L().Error("failed to list runs", zap.Error(err))
			return fmt.Errorf("failed to list runs: %w", err)
		}

		if len(runs) == 0 {
			fmt.Println("No runs found")
			return nil
		}

		if runsJSON {
			return printRunsJSON(runs)
		}

		return printRunsTable(runs)
	},
}

// printRunsTable displays runs in a formatted table.
func printRunsTable(runs []*run.WorkflowRun) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintf(w, "RUN ID\tWORKFLOW\tSTATUS\tSTARTED AT\tDURATION\n")
	fmt.Fprintf(w, "------\t--------\t------\t----------\t--------\n")

	for _, r := range runs {
		duration := "-"
		if r.EndedAt.Valid {
			d := r.EndedAt.Time.Sub(r.StartedAt)
			duration = fmt.Sprintf("%.2fs", d.Seconds())
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			r.ID,
			r.Workflow,
			coloriseStatus(r.Status),
			r.StartedAt.Format("2006-01-02 15:04:05"),
			duration,
		)
	}

	logger.L().Info("displayed runs", zap.Int("count", len(runs)))

	w.Flush()
	return nil
}

// printRunsJSON outputs runs in JSON format.
func printRunsJSON(runs []*run.WorkflowRun) error {
	for _, r := range runs {
		data, err := run.MarshalRun(r)
		if err != nil {
			return err
		}

		var out bytes.Buffer
		json.Indent(&out, data, "", "  ")
		fmt.Println(out.String())
	}

	logger.L().Info("displayed runs in JSON", zap.Int("count", len(runs)))

	return nil
}

// coloriseStatus adds color to status strings for better readability.
func coloriseStatus(status run.WorkflowStatus) string {
	switch status {
	case run.StatusSuccess:
		return "✓ " + string(status)
	case run.StatusFailed:
		return "✗ " + string(status)
	case run.StatusRunning:
		return "⟳ " + string(status)
	default:
		return string(status)
	}
}

func init() {
	rootCmd.AddCommand(runsCmd)

	runsCmd.Flags().StringVarP(&runsWorkflow, "workflow", "w", "", "Filter by workflow name")
	runsCmd.Flags().StringVarP(&runsStatus, "status", "s", "", "Filter by status (pending|running|success|failed)")
	runsCmd.Flags().IntVarP(&runsLimit, "limit", "l", 10, "Limit number of results")
	runsCmd.Flags().IntVarP(&runsOffset, "offset", "o", 0, "Offset for pagination")
	runsCmd.Flags().BoolVar(&runsJSON, "json", false, "Output in JSON format")
}
