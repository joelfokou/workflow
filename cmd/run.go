package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"

	"github.com/joelfokou/workflow/internal/config"
	"github.com/joelfokou/workflow/internal/dag"
	"github.com/joelfokou/workflow/internal/executor"
	"github.com/joelfokou/workflow/internal/logger"
	"github.com/joelfokou/workflow/internal/run"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	runDryRun bool
	runJSON   bool
)

// runCmd executes a specified workflow by loading its definition, setting up a context with cancellation support, handling interrupts (Ctrl+C), and then running the workflow using an executor.
var runCmd = &cobra.Command{
	Use:   "run <workflow>",
	Short: "Run a workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workflowName := args[0]

		// Load workflow DAG
		d, err := dag.Load(workflowName)
		if err != nil {
			logger.L().Error("failed to load workflow", zap.String("workflow", workflowName), zap.Error(err))
			return err
		}

		if runDryRun {
			plan, err := planRun(d)
			if err != nil {
				logger.L().Error("failed to generate execution plan", zap.String("workflow", workflowName), zap.Error(err))
				return fmt.Errorf("failed to generate execution plan: %w", err)
			}
			if runJSON {
				return printPlanJSON(plan)
			}

			printPlan(plan)
			fmt.Println("\nNo tasks were executed.")
			return nil
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

		// Initialise run store
		dbPath := config.C.Paths.Database
		store, err := run.NewStore(dbPath)
		if err != nil {
			logger.L().Error("failed to initialise run store", zap.Error(err))
			return fmt.Errorf("failed to initialise run store: %w", err)
		}
		defer store.Close()

		// Create executor and run workflow
		executor := executor.NewExecutor(store)
		if err := executor.Run(ctx, d); err != nil {
			logger.L().Error("workflow execution failed", zap.String("workflow", workflowName), zap.Error(err))
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().BoolVar(&runDryRun, "dry-run", false, "Print execution plan without running tasks")
	runCmd.Flags().BoolVar(&runJSON, "json", false, "Output in JSON format")

}

func planRun(d *dag.DAG) (*run.WorkflowPlan, error) {
	order, err := d.TopologicalSort()
	if err != nil {
		return nil, err
	}

	plan := &run.WorkflowPlan{
		Workflow: d.Name,
		Tasks:    []run.TaskPlan{},
	}

	for i, t := range order {
		plan.Tasks = append(plan.Tasks, run.TaskPlan{
			Order:     i + 1,
			Name:      t.Name,
			Cmd:       t.Cmd,
			DependsOn: t.DependsOn,
			Retries:   t.Retries,
		})
	}
	return plan, nil
}

func printPlan(plan *run.WorkflowPlan) {
	fmt.Print("========== DRY RUN MODE ==========\n\n")
	fmt.Printf("Execution Plan for Workflow: %s\n", plan.Workflow)
	fmt.Println("--------------------------------------------------")
	for _, task := range plan.Tasks {
		fmt.Printf("Task %d: %s\n", task.Order, task.Name)
		fmt.Printf("  Command: %s\n", task.Cmd)
		if len(task.DependsOn) > 0 {
			fmt.Printf("  Depends On: %v\n", task.DependsOn)
		}
		fmt.Printf("  Retries: %d\n", task.Retries)
		fmt.Println("--------------------------------------------------")
	}
}

func printPlanJSON(plan *run.WorkflowPlan) error {
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plan to JSON: %w", err)
	}
	fmt.Print("========== DRY RUN MODE ==========\n\n")
	fmt.Printf("Execution Plan for Workflow: %s\n", plan.Workflow)
	fmt.Println("--------------------------------------------------")
	fmt.Println(string(data))
	return nil
}
