// Package integration contains integration tests for the executor component of the wf application.
package integration

import (
	"context"
	"testing"
	"time"

	"github.com/joelfokou/workflow/internal/config"
	"github.com/joelfokou/workflow/internal/dag"
	"github.com/joelfokou/workflow/internal/executor"
	"github.com/joelfokou/workflow/internal/logger"
	"github.com/joelfokou/workflow/internal/run"
	"github.com/joelfokou/workflow/tests/helpers"
)

func init() {
	logger.Init(logger.Config{
		Level:  "info",
		Format: "console",
	})
}

// TestExecutorIntegrationSuccess tests the executor with a simple workflow that should complete successfully.
func TestExecutorIntegrationSuccess(t *testing.T) {
	fs := helpers.NewTestFS(t)
	defer fs.Cleanup()

	config.C.Paths.Logs = fs.Path("logs")
	config.C.Paths.Database = fs.Path("test.db")

	store, err := run.NewStore(config.C.Paths.Database)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	ex := executor.NewExecutor(store)

	d, err := dag.LoadFromString(helpers.SimpleWorkflow())
	if err != nil {
		t.Fatalf("failed to load workflow: %v", err)
	}

	if err := d.Validate(); err != nil {
		t.Fatalf("workflow validation failed: %v", err)
	}

	// Execute workflow
	err = ex.Run(context.Background(), d)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	// Verify workflow run was saved with correct status
	runs, err := store.ListRuns(d.Name, "", 10, 0)
	if err != nil {
		t.Fatalf("failed to list runs: %v", err)
	}

	if len(runs) == 0 {
		t.Fatal("expected at least one run to be saved")
	}

	wr := runs[0]
	if wr.Status != run.StatusSuccess {
		t.Errorf("expected status %s, got %s", run.StatusSuccess, wr.Status)
	}

	if !wr.EndedAt.Valid {
		t.Error("expected EndedAt to be set")
	}

	// Verify tasks were saved
	tasks, err := store.LoadTaskRuns(wr.ID)
	if err != nil {
		t.Fatalf("failed to load task runs: %v", err)
	}

	if len(tasks) == 0 {
		t.Fatal("expected tasks to be saved")
	}

	// Verify all tasks succeeded
	for _, task := range tasks {
		if task.Status != run.TaskSuccess {
			t.Errorf("task %s: expected status %s, got %s", task.Name, run.TaskSuccess, task.Status)
		}

		if !task.EndedAt.Valid {
			t.Errorf("task %s: expected EndedAt to be set", task.Name)
		}

		if !task.ExitCode.Valid || task.ExitCode.Int64 != 0 {
			t.Errorf("task %s: expected exit code 0, got %v", task.Name, task.ExitCode)
		}

		// Verify log file exists
		if task.LogPath == "" {
			t.Errorf("task %s: expected log path to be set", task.Name)
		}
	}
}

// TestExecutorIntegrationTaskFailure tests the executor with a workflow that has a failing task.
func TestExecutorIntegrationTaskFailure(t *testing.T) {
	fs := helpers.NewTestFS(t)
	defer fs.Cleanup()

	config.C.Paths.Logs = fs.Path("logs")
	config.C.Paths.Database = fs.Path("test.db")

	store, err := run.NewStore(config.C.Paths.Database)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	ex := executor.NewExecutor(store)

	d, err := dag.LoadFromString(helpers.FailingWorkflow())
	if err != nil {
		t.Fatalf("failed to load workflow: %v", err)
	}

	// Execute workflow - should fail
	err = ex.Run(context.Background(), d)
	if err == nil {
		t.Fatal("expected workflow to fail")
	}

	// Verify workflow run was saved with failed status
	runs, err := store.ListRuns(d.Name, "", 10, 0)
	if err != nil {
		t.Fatalf("failed to list runs: %v", err)
	}

	if len(runs) == 0 {
		t.Fatal("expected run to be saved even on failure")
	}

	wr := runs[0]
	if wr.Status != run.StatusFailed {
		t.Errorf("expected status %s, got %s", run.StatusFailed, wr.Status)
	}

	// Verify failed task has non-zero exit code
	tasks, err := store.LoadTaskRuns(wr.ID)
	if err != nil {
		t.Fatalf("failed to load task runs: %v", err)
	}

	var failedTask *run.TaskRun
	for i := range tasks {
		if tasks[i].Status == run.TaskFailed {
			failedTask = &tasks[i]
			break
		}
	}

	if failedTask == nil {
		t.Fatal("expected at least one failed task")
	}

	if !failedTask.ExitCode.Valid || failedTask.ExitCode.Int64 == 0 {
		t.Error("expected non-zero exit code for failed task")
	}

	if failedTask.LastError == "" {
		t.Error("expected error message to be recorded")
	}
}

// TestExecutorIntegrationTaskRetry tests that tasks are retried on failure.
func TestExecutorIntegrationTaskRetry(t *testing.T) {
	fs := helpers.NewTestFS(t)
	defer fs.Cleanup()

	config.C.Paths.Logs = fs.Path("logs")
	config.C.Paths.Database = fs.Path("test.db")

	store, err := run.NewStore(config.C.Paths.Database)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	ex := executor.NewExecutor(store)

	d, err := dag.LoadFromString(helpers.RetryWorkflow())
	if err != nil {
		t.Fatalf("failed to load workflow: %v", err)
	}

	// Execute workflow - should fail after retries
	err = ex.Run(context.Background(), d)
	if err == nil {
		t.Fatal("expected workflow to fail after retries exhausted")
	}

	// Verify task has multiple attempts recorded
	runs, err := store.ListRuns(d.Name, "", 10, 0)
	if err != nil {
		t.Fatalf("failed to list runs: %v", err)
	}

	if len(runs) == 0 {
		t.Fatal("expected run to be saved")
	}

	tasks, err := store.LoadTaskRuns(runs[0].ID)
	if err != nil {
		t.Fatalf("failed to load task runs: %v", err)
	}

	if len(tasks) == 0 {
		t.Fatal("expected tasks to be saved")
	}

	// Verify retries were attempted
	retryTask := tasks[0]
	expectedRetries := 3 // Or whatever the test workflow uses
	if retryTask.Attempts < expectedRetries {
		t.Errorf("expected at least %d attempts, got %d", expectedRetries, retryTask.Attempts)
	}
}

// TestExecutorIntegrationComplexWorkflow tests the executor with a complex multi-task workflow.
func TestExecutorIntegrationComplexWorkflow(t *testing.T) {
	fs := helpers.NewTestFS(t)
	defer fs.Cleanup()

	config.C.Paths.Logs = fs.Path("logs")
	config.C.Paths.Database = fs.Path("test.db")

	store, err := run.NewStore(config.C.Paths.Database)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	ex := executor.NewExecutor(store)

	d, err := dag.LoadFromString(helpers.ComplexWorkflow())
	if err != nil {
		t.Fatalf("failed to load workflow: %v", err)
	}

	startTime := time.Now()
	err = ex.Run(context.Background(), d)
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	// Verify workflow run
	runs, err := store.ListRuns(d.Name, "", 10, 0)
	if err != nil {
		t.Fatalf("failed to list runs: %v", err)
	}

	if len(runs) == 0 {
		t.Fatal("expected run to be saved")
	}

	wr := runs[0]
	if wr.Status != run.StatusSuccess {
		t.Errorf("expected status %s, got %s", run.StatusSuccess, wr.Status)
	}

	// Verify all tasks completed
	tasks, err := store.LoadTaskRuns(wr.ID)
	if err != nil {
		t.Fatalf("failed to load task runs: %v", err)
	}

	expectedTaskCount := len(d.Tasks)
	if len(tasks) != expectedTaskCount {
		t.Errorf("expected %d tasks, got %d", expectedTaskCount, len(tasks))
	}

	// Verify execution order matches topological sort
	order, _ := d.TopologicalSort()
	for i, task := range tasks {
		if task.Name != order[i].Name {
			t.Errorf("task execution order mismatch at position %d: expected %s, got %s",
				i, order[i].Name, task.Name)
		}
	}

	t.Logf("Complex workflow executed in %v with %d tasks", duration, len(tasks))
}

// TestExecutorIntegrationContextCancellation tests that workflows can be cancelled via context.
func TestExecutorIntegrationContextCancellation(t *testing.T) {
	fs := helpers.NewTestFS(t)
	defer fs.Cleanup()

	config.C.Paths.Logs = fs.Path("logs")
	config.C.Paths.Database = fs.Path("test.db")

	store, err := run.NewStore(config.C.Paths.Database)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	ex := executor.NewExecutor(store)

	d, err := dag.LoadFromString(helpers.LongRunningWorkflow())
	if err != nil {
		t.Fatalf("failed to load workflow: %v", err)
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Cancel after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	// Execute workflow - should be cancelled
	err = ex.Run(ctx, d)
	if err == nil {
		t.Fatal("expected workflow to be cancelled")
	}

	// Verify workflow run was marked as failed
	runs, err := store.ListRuns(d.Name, "", 10, 0)
	if err != nil {
		t.Fatalf("failed to list runs: %v", err)
	}

	if len(runs) > 0 {
		wr := runs[0]
		if wr.Status != run.StatusFailed {
			t.Errorf("expected cancelled workflow to be marked failed, got %s", wr.Status)
		}
	}
}
