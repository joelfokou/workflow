package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/joelfokou/workflow/internal/config"
	"github.com/joelfokou/workflow/internal/dag"
	"github.com/joelfokou/workflow/internal/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	validateJSON bool
)

// validateResult holds the result of validating a single workflow.
type validateResult struct {
	Name  string `json:"name"`
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

// validateCmd checks the validity of all workflow definitions in the configured workflows directory, logging errors if any are found and confirming success if all workflows are valid.
var validateCmd = &cobra.Command{
	Use:   "validate [workflow]",
	Short: "Validate workflow definitions",
	Long:  "Validate all workflows or a specific workflow in the configured workflows directory",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			return validateSingleWorkflow(args[0])
		}

		return validateAllWorkflows()
	},
}

// validateSingleWorkflow validates a specific workflow.
func validateSingleWorkflow(workflowName string) error {
	d, err := dag.Load(workflowName)
	if err != nil {
		logger.L().Error("workflow validation failed",
			zap.String("workflow", workflowName),
			zap.Error(err),
		)

		result := validateResult{
			Name:  workflowName,
			Valid: false,
			Error: err.Error(),
		}

		if validateJSON {
			return printValidateJSON([]validateResult{result})
		}

		fmt.Printf("✗ %s: %v\n", workflowName, err)
		return err
	}

	// Additional validation checks
	order, err := d.TopologicalSort()
	if err != nil {
		logger.L().Error("topological sort failed",
			zap.String("workflow", workflowName),
			zap.Error(err),
		)

		result := validateResult{
			Name:  workflowName,
			Valid: false,
			Error: err.Error(),
		}

		if validateJSON {
			return printValidateJSON([]validateResult{result})
		}

		fmt.Printf("✗ %s: %v\n", workflowName, err)
		return err
	}

	result := validateResult{
		Name:  workflowName,
		Valid: true,
	}

	logger.L().Info("workflow validation successful",
		zap.String("workflow", workflowName),
		zap.Int("tasks", len(d.Tasks)),
		zap.Int("execution_order_length", len(order)),
	)

	if validateJSON {
		return printValidateJSON([]validateResult{result})
	}

	fmt.Printf("✓ %s: valid (%d tasks)\n", workflowName, len(d.Tasks))
	return nil
}

// validateAllWorkflows validates all workflows in the directory.
func validateAllWorkflows() error {
	entries, err := os.ReadDir(config.C.Paths.Workflows)
	if err != nil {
		logger.L().Error("failed to read workflows directory",
			zap.String("directory", config.C.Paths.Workflows),
			zap.Error(err),
		)
		return fmt.Errorf("failed to read workflows directory: %w", err)
	}

	var results []validateResult
	var failedCount int

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		workflowName := strings.TrimSuffix(entry.Name(), ".toml")
		d, err := dag.Load(workflowName)

		if err != nil {
			logger.L().Warn("workflow validation failed",
				zap.String("workflow", workflowName),
				zap.Error(err),
			)

			results = append(results, validateResult{
				Name:  workflowName,
				Valid: false,
				Error: err.Error(),
			})
			failedCount++
			continue
		}

		// Check topological sort
		if _, err := d.TopologicalSort(); err != nil {
			logger.L().Warn("topological sort failed",
				zap.String("workflow", workflowName),
				zap.Error(err),
			)

			results = append(results, validateResult{
				Name:  workflowName,
				Valid: false,
				Error: err.Error(),
			})
			failedCount++
			continue
		}

		results = append(results, validateResult{
			Name:  workflowName,
			Valid: true,
		})
	}

	if validateJSON {
		return printValidateJSON(results)
	}

	return printValidateTable(results, failedCount)
}

// printValidateTable displays validation results in table format.
func printValidateTable(results []validateResult, failedCount int) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintf(w, "WORKFLOW\tSTATUS\tERROR\n")
	fmt.Fprintf(w, "--------\t------\t-----\n")

	for _, r := range results {
		status := "✓ valid"
		errMsg := "-"

		if !r.Valid {
			status = "✗ invalid"
			errMsg = truncateError(r.Error, 50)
		}

		fmt.Fprintf(w, "%s\t%s\t%s\n", r.Name, status, errMsg)
	}

	w.Flush()

	fmt.Printf("\n%d/%d workflows valid\n", len(results)-failedCount, len(results))

	if failedCount > 0 {
		logger.L().Error("validation failed", zap.Int("failed_count", failedCount), zap.Int("total_count", len(results)))
		return fmt.Errorf("%d workflow(s) failed validation", failedCount)
	}

	logger.L().Info("all workflows validated successfully", zap.Int("count", len(results)))
	return nil
}

// printValidateJSON outputs validation results in JSON format.
func printValidateJSON(results []validateResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

// truncateError truncates error messages to a maximum length.
func truncateError(msg string, maxLen int) string {
	if len(msg) > maxLen {
		return msg[:maxLen-3] + "..."
	}
	return msg
}

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().BoolVar(&validateJSON, "json", false, "Output in JSON format")
}
