package run

import (
	"path/filepath"
	"testing"
)

// TestNewWorkflowRun tests the NewWorkflowRun method of the Store.
func TestNewWorkflowRun(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "/test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Test
	workflowName := "test-workflow"
	dagHash := "dummy-dag-hash"
	run, err := store.NewWorkflowRun(workflowName, dagHash)
	if err != nil {
		t.Fatalf("NewWorkflowRun failed: %v", err)
	}

	// Assertions
	if run.ID == "" {
		t.Error("WorkflowRun ID is empty")
	}
	if run.Workflow != workflowName {
		t.Errorf("expected workflow %s, got %s", workflowName, run.Workflow)
	}
	if run.Status != StatusRunning {
		t.Errorf("expected status Running, got %s", run.Status)
	}
	if run.WorkflowHash != dagHash {
		t.Errorf("expected workflow hash %s, got %s", dagHash, run.WorkflowHash)
	}
	if run.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
	if run.StartedAt.IsZero() {
		t.Error("StartedAt is zero")
	}

	// Verify database persistence
	loadedRun, err := store.Load(run.ID)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loadedRun.ID != run.ID {
		t.Errorf("expected run ID %s, got %s", run.ID, loadedRun.ID)
	}
	if loadedRun.Workflow != workflowName {
		t.Errorf("expected workflow %s, got %s", workflowName, loadedRun.Workflow)
	}
}

// TestTaskRuns tests SaveTaskRun, UpdateTaskRun, and LoadTaskRuns methods.
func TestTaskRuns(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "/test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Create a workflow run first
	run, err := store.NewWorkflowRun("test-workflow", "dag-hash")
	if err != nil {
		t.Fatalf("NewWorkflowRun failed: %v", err)
	}

	// Test SaveTaskRun
	task := &TaskRun{
		RunID:  run.ID,
		Name:   "test-task",
		Status: TaskRunning,
	}

	err = store.SaveTaskRun(task)
	if err != nil {
		t.Fatalf("SaveTaskRun failed: %v", err)
	}

	if task.ID == 0 {
		t.Error("TaskRun ID is zero after save")
	}

	// Test LoadTaskRuns
	tasks, err := store.LoadTaskRuns(run.ID)
	if err != nil {
		t.Fatalf("LoadTaskRuns failed: %v", err)
	}

	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Name != "test-task" {
		t.Errorf("expected task name test-task, got %s", tasks[0].Name)
	}

	// Test UpdateTaskRun
	tasks[0].Status = TaskSuccess
	err = store.UpdateTaskRun(&tasks[0])
	if err != nil {
		t.Fatalf("UpdateTaskRun failed: %v", err)
	}

	// Verify update
	updatedTasks, err := store.LoadTaskRuns(run.ID)
	if err != nil {
		t.Fatalf("LoadTaskRuns failed after update: %v", err)
	}
	if updatedTasks[0].Status != TaskSuccess {
		t.Errorf("expected status %s, got %s", TaskSuccess, updatedTasks[0].Status)
	}
}
