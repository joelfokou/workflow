package dag

import (
	"fmt"
	"regexp"

	"github.com/joelfokou/workflow/internal/logger"
	"go.uber.org/zap"
)

// taskNamePattern defines valid characters for task names
var taskNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// Validate checks the DAG for common issues:
// - Valid workflow name
// - Tasks exist
// - No cycles
// - No duplicate task names
// - Valid characters in task names
// - Tasks have commands
// - All dependencies reference existing tasks
func (d *DAG) Validate() error {
	// Check workflow name
	if d.Name == "" {
		return fmt.Errorf("workflow name is required")
	}

	// Check tasks exist
	if len(d.Tasks) == 0 {
		return fmt.Errorf("no tasks defined")
	}

	// Check for duplicate task names and invalid characters
	seen := make(map[string]struct{}, len(d.Tasks))
	for name, t := range d.Tasks {
		// Check for duplicate task names
		if _, ok := seen[name]; ok {
			return fmt.Errorf("duplicate task name: %s", name)
		}
		seen[name] = struct{}{}

		// Validate task name format
		if !taskNamePattern.MatchString(name) {
			return fmt.Errorf("invalid task name %q (allowed: letters, digits, _, -)", name)
		}

		// Check task has a command
		if t.Cmd == "" {
			logger.L().Error("task missing command", zap.String("task", name))
			return fmt.Errorf("task %s has no command defined", name)
		}

		// Check dependencies exist
		for _, dep := range t.DependsOn {
			if _, ok := d.Tasks[dep]; !ok {
				logger.L().Error("missing dependency", zap.String("task", name), zap.String("dependency", dep))
				return fmt.Errorf("task %s depends on missing task %s", name, dep)
			}
		}
	}

	// Check for cycles
	if _, err := d.TopologicalSort(); err != nil {
		return err
	}

	return nil
}
