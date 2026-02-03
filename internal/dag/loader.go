package dag

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joelfokou/workflow/internal/config"
	"github.com/joelfokou/workflow/internal/logger"
	"github.com/pelletier/go-toml/v2"
	"go.uber.org/zap"
)

// rawWorkflow is an internal representation of the workflow structure in TOML format.
type rawWorkflow struct {
	Name  string `toml:"name"`
	Tasks map[string]struct {
		Cmd       string   `toml:"cmd"`
		Retries   int      `toml:"retries"`
		DependsOn []string `toml:"depends_on"`
	} `toml:"tasks"`
}

// Load reads a workflow from a TOML file located in the configured workflows directory.
func Load(path string) (*DAG, error) {
	path = strings.TrimSuffix(path, ".toml")

	filePath := filepath.Join(config.C.Paths.Workflows, path+".toml")
	data, err := os.ReadFile(filePath)
	if err != nil {
		logger.L().Error("failed to read workflow file", zap.String("path", filePath), zap.Error(err))
		return nil, fmt.Errorf("failed to read workflow file %s: %w", filePath, err)
	}

	dag, err := parseWorkflow(data)
	if err != nil {
		logger.L().Error("failed to parse workflow", zap.String("path", filePath), zap.Error(err))
		return nil, err
	}

	if err := dag.Validate(); err != nil {
		logger.L().Error("workflow validation failed", zap.String("workflow", dag.Name), zap.Error(err))
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	logger.L().Info("workflow loaded successfully", zap.String("workflow", dag.Name), zap.Int("tasks", len(dag.Tasks)))
	return dag, nil
}

// LoadFromString reads a workflow from a TOML-formatted string.
func LoadFromString(data string) (*DAG, error) {
	dag, err := parseWorkflow([]byte(data))
	if err != nil {
		logger.L().Error("failed to parse workflow from string", zap.Error(err))
		return nil, err
	}

	if err := dag.Validate(); err != nil {
		logger.L().Error("workflow validation failed", zap.String("workflow", dag.Name), zap.Error(err))
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	logger.L().Info("workflow loaded from string", zap.String("workflow", dag.Name), zap.Int("tasks", len(dag.Tasks)))
	return dag, nil
}

// parseWorkflow converts raw TOML bytes into a DAG structure.
func parseWorkflow(data []byte) (*DAG, error) {
	var wf rawWorkflow
	if err := toml.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal TOML: %w", err)
	}

	if wf.Name == "" {
		return nil, fmt.Errorf("workflow name is required")
	}

	dag := &DAG{
		Name:  wf.Name,
		Tasks: make(map[string]*Task, len(wf.Tasks)),
	}

	for name, t := range wf.Tasks {
		dag.Tasks[name] = &Task{
			Name:      name,
			Cmd:       t.Cmd,
			Retries:   t.Retries,
			DependsOn: t.DependsOn,
		}
	}

	return dag, nil
}

// ValidateAll checks all workflow files in the specified directory for validity.
func ValidateAll(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		logger.L().Error("failed to read workflows directory", zap.String("dir", dir), zap.Error(err))
		return fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	var validationErrors []error
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		workflowName := entry.Name()[:len(entry.Name())-5] // remove .toml
		_, err := Load(workflowName)
		if err != nil {
			logger.L().Error("invalid workflow", zap.String("workflow", workflowName), zap.Error(err))
			validationErrors = append(validationErrors, err)
		}
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf("validation failed: %d workflow(s) invalid", len(validationErrors))
	}

	logger.L().Info("all workflows validated successfully", zap.Int("count", len(entries)))
	return nil
}
