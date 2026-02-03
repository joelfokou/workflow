package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/joelfokou/workflow/internal/dag"
	"github.com/joelfokou/workflow/internal/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	graphFormat string
	graphDetail bool
)

// graphCmd displays the directed acyclic graph (DAG) structure of a workflow.
// Supports multiple output formats: ascii, dot (Graphviz), and json.
var graphCmd = &cobra.Command{
	Use:   "graph <workflow>",
	Short: "Display workflow DAG structure",
	Long:  "Visualise the workflow as a directed acyclic graph in various formats",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workflowName := args[0]

		d, err := dag.Load(workflowName)
		if err != nil {
			logger.L().Error("failed to load workflow", zap.String("workflow", workflowName), zap.Error(err))
			return fmt.Errorf("failed to load workflow %s: %w", workflowName, err)
		}

		logger.L().Info("rendering workflow graph",
			zap.String("workflow", workflowName),
			zap.String("format", graphFormat),
			zap.Int("tasks", len(d.Tasks)),
		)

		switch graphFormat {
		case "ascii":
			return renderASCII(d, graphDetail)
		case "dot":
			return renderDot(d)
		case "json":
			return renderJSON(d)
		default:
			return fmt.Errorf("unsupported format: %s (supported: ascii, dot, json)", graphFormat)
		}
	},
}

// renderASCII displays the workflow as an ASCII tree.
func renderASCII(d *dag.DAG, detailed bool) error {
	fmt.Println(d.RenderASCII())

	if detailed {
		fmt.Println("\n--- Task Details ---")
		order, _ := d.TopologicalSort()

		for i, task := range order {
			fmt.Printf("\n[%d] %s\n", i+1, task.Name)
			fmt.Printf("    Command:  %s\n", task.Cmd)
			fmt.Printf("    Retries:  %d\n", task.Retries)

			if len(task.DependsOn) > 0 {
				fmt.Printf("    Depends:  %v\n", task.DependsOn)
			} else {
				fmt.Printf("    Depends:  none (root task)\n")
			}
		}
	}

	return nil
}

// renderDot generates Graphviz DOT format for the workflow.
func renderDot(d *dag.DAG) error {
	fmt.Println("digraph " + sanitiseName(d.Name) + " {")
	fmt.Println("  rankdir=LR;")
	fmt.Println("  node [shape=box, style=rounded];")

	// Add nodes
	for _, task := range d.Tasks {
		fmt.Printf("  \"%s\" [label=\"%s\"];\n", task.Name, task.Name)
	}

	fmt.Println()

	// Add edges
	for _, task := range d.Tasks {
		for _, dep := range task.DependsOn {
			fmt.Printf("  \"%s\" -> \"%s\";\n", dep, task.Name)
		}
	}

	fmt.Println("}")

	fmt.Fprintf(os.Stderr, "\nℹ Tip: Visualise with: dot -Tpng workflow.dot -o workflow.png\n")
	fmt.Fprintf(os.Stderr, "\n  Or copy and paste the output into an online Graphviz editor\n")

	return nil
}

// renderJSON outputs the workflow structure as JSON.
func renderJSON(d *dag.DAG) error {
	type taskJSON struct {
		Name      string   `json:"name"`
		Cmd       string   `json:"cmd"`
		Retries   int      `json:"retries"`
		DependsOn []string `json:"depends_on,omitempty"`
	}

	type dagJSON struct {
		Name  string     `json:"name"`
		Tasks []taskJSON `json:"tasks"`
	}

	order, _ := d.TopologicalSort()
	var tasks []taskJSON

	for _, task := range order {
		tasks = append(tasks, taskJSON{
			Name:      task.Name,
			Cmd:       task.Cmd,
			Retries:   task.Retries,
			DependsOn: task.DependsOn,
		})
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(dagJSON{
		Name:  d.Name,
		Tasks: tasks,
	})
}

// sanitiseName removes special characters from workflow name for Graphviz.
func sanitiseName(name string) string {
	var result []rune
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			result = append(result, r)
		} else {
			result = append(result, '_')
		}
	}
	return string(result)
}

func init() {
	rootCmd.AddCommand(graphCmd)

	graphCmd.Flags().StringVarP(&graphFormat, "format", "f", "ascii", "Output format: ascii, dot (Graphviz), or json")
	graphCmd.Flags().BoolVarP(&graphDetail, "detail", "d", false, "Show detailed task information (ASCII only)")
}
