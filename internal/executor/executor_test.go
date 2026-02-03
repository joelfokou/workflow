package executor

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joelfokou/workflow/internal/config"
	"github.com/joelfokou/workflow/internal/dag"
	"github.com/joelfokou/workflow/internal/logger"
	"github.com/joelfokou/workflow/internal/run"
)

func init() {
	logger.Init(logger.Config{
		Level:  "info",
		Format: "console",
	})
}

// TestNewExecutor tests the creation of a new Executor instance.
func TestNewExecutor(t *testing.T) {
	tmpDir := t.TempDir()
	config.C.Paths.Logs = tmpDir

	store, err := run.NewStore(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	executor := NewExecutor(store)

	if executor.RunStore != store {
		t.Errorf("expected RunStore to be set")
	}
	if executor.DefaultTaskTimeout != 0 {
		t.Errorf("expected DefaultTaskTimeout to be 0, got %v", executor.DefaultTaskTimeout)
	}
}

// TestExecutorRunSuccess tests the successful execution of a simple workflow.
func TestExecutorRunSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	config.C.Paths.Logs = tmpDir

	store, err := run.NewStore(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	executor := NewExecutor(store)

	d := &dag.DAG{
		Name: "test-workflow",
		Tasks: map[string]*dag.Task{
			"task1": {Name: "task1", Cmd: "echo hello", Retries: 0},
		},
	}

	err = executor.Run(context.Background(), d)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Verify workflow run was saved
	_, err = store.LoadTaskRuns(d.Name)
	if err != nil && err != sql.ErrNoRows {
		t.Errorf("failed to verify workflow run: %v", err)
	}
}

// TestExecutorRunTaskFailure tests the execution of a workflow with a failing task.
func TestExecutorRunTaskFailure(t *testing.T) {
	tmpDir := t.TempDir()
	config.C.Paths.Logs = tmpDir

	store, err := run.NewStore(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	executor := NewExecutor(store)

	d := &dag.DAG{
		Name: "test-workflow",
		Tasks: map[string]*dag.Task{
			"task1": {Name: "task1", Cmd: "exit 1", Retries: 0},
		},
	}

	err = executor.Run(context.Background(), d)
	if err == nil {
		t.Errorf("expected error for failed task")
	}
}

// TestExecutorRunContextCancellation tests the cancellation of a workflow execution.
func TestExecutorRunContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	config.C.Paths.Logs = tmpDir

	store, err := run.NewStore(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	executor := NewExecutor(store)

	d := &dag.DAG{
		Name: "test-workflow",
		Tasks: map[string]*dag.Task{
			"task1": {Name: "task1", Cmd: "sleep 10", Retries: 0},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)

	err = executor.Run(ctx, d)
	if err == nil {
		t.Errorf("expected error for cancelled context")
	}
}

// TestExecutorLogFileCreation tests that log files are created for executed tasks.
func TestExecutorLogFileCreation(t *testing.T) {
	tmpDir := t.TempDir()
	config.C.Paths.Logs = tmpDir

	store, err := run.NewStore(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	executor := NewExecutor(store)

	d := &dag.DAG{
		Name: "test-workflow",
		Tasks: map[string]*dag.Task{
			"task1": {Name: "task1", Cmd: "echo test output", Retries: 0},
		},
	}

	err = executor.Run(context.Background(), d)
	if err != nil {
		t.Errorf("execution failed: %v", err)
	}

	// Verify log file was created
	logDir := filepath.Join(tmpDir, "*")
	matches, err := filepath.Glob(logDir)
	if err != nil || len(matches) == 0 {
		t.Errorf("expected log directory to be created")
	}

	// Verify log file exists
	if len(matches) > 0 {
		taskLogGlob := filepath.Join(matches[0], "task1_*.log")
		taskLogs, err := filepath.Glob(taskLogGlob)
		if err != nil || len(taskLogs) == 0 {
			t.Errorf("expected task log file to be created")
		}

		// Verify log file contains output
		if len(taskLogs) > 0 {
			content, err := os.ReadFile(taskLogs[0])
			if err != nil {
				t.Errorf("failed to read log file: %v", err)
			}
			if len(content) == 0 {
				t.Errorf("expected log file to contain output")
			}
		}
	}
}

// TestExecutorTaskRetry tests that tasks are retried on failure.
func TestExecutorTaskRetry(t *testing.T) {
	tmpDir := t.TempDir()
	config.C.Paths.Logs = tmpDir

	store, err := run.NewStore(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	executor := NewExecutor(store)

	d := &dag.DAG{
		Name: "test-workflow",
		Tasks: map[string]*dag.Task{
			"task1": {Name: "task1", Cmd: "exit 1", Retries: 2},
		},
	}

	err = executor.Run(context.Background(), d)
	if err == nil {
		t.Errorf("expected error after retries exhausted")
	}
}
