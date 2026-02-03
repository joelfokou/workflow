package dag

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joelfokou/workflow/internal/config"
	"github.com/joelfokou/workflow/internal/logger"
)

func init() {
	logger.Init(logger.Config{
		Level:  "info",
		Format: "console",
	})
}

// TestDAGTopoSort tests the topological sorting of tasks in a DAG.
func TestDAGTopoSort(t *testing.T) {
	d := &DAG{
		Name: "test",
		Tasks: map[string]*Task{
			"a": {Name: "a", Cmd: "echo a"},
			"b": {Name: "b", Cmd: "echo b", DependsOn: []string{"a"}},
			"c": {Name: "c", Cmd: "echo c", DependsOn: []string{"b"}},
		},
	}

	order, err := d.TopologicalSort()
	if err != nil {
		t.Fatalf("TopologicalSort failed: %v", err)
	}

	want := []string{"a", "b", "c"}
	if len(order) != len(want) {
		t.Fatalf("expected %d tasks, got %d", len(want), len(order))
	}

	for i, task := range order {
		if task.Name != want[i] {
			t.Errorf("order mismatch at %d: want %s got %s", i, want[i], task.Name)
		}
	}
}

// TestDAGTopoSortMultipleDependencies tests topological sort with multiple dependencies.
func TestDAGTopoSortMultipleDependencies(t *testing.T) {
	d := &DAG{
		Name: "test",
		Tasks: map[string]*Task{
			"a": {Name: "a", Cmd: "echo a"},
			"b": {Name: "b", Cmd: "echo b"},
			"c": {Name: "c", Cmd: "echo c", DependsOn: []string{"a", "b"}},
		},
	}

	order, err := d.TopologicalSort()
	if err != nil {
		t.Fatalf("TopologicalSort failed: %v", err)
	}

	// c should come after both a and b
	cIndex := -1
	aIndex := -1
	bIndex := -1

	for i, task := range order {
		if task.Name == "c" {
			cIndex = i
		}
		if task.Name == "a" {
			aIndex = i
		}
		if task.Name == "b" {
			bIndex = i
		}
	}

	if cIndex < 0 || aIndex < 0 || bIndex < 0 {
		t.Fatal("not all tasks found in sorted order")
	}
	if cIndex <= aIndex || cIndex <= bIndex {
		t.Error("task c should come after both a and b")
	}
}

// TestDAGCycleDetection tests that a cycle in the DAG is detected.
func TestDAGCycleDetection(t *testing.T) {
	d := &DAG{
		Name: "test",
		Tasks: map[string]*Task{
			"a": {Name: "a", Cmd: "echo a", DependsOn: []string{"b"}},
			"b": {Name: "b", Cmd: "echo b", DependsOn: []string{"a"}},
		},
	}

	err := d.Validate()
	if err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}
	if err.Error() != "cycle detected in DAG" {
		t.Errorf("expected 'cycle detected in DAG', got: %v", err)
	}
}

// TestDAGValidateMissingDependency tests validation catches missing dependencies.
func TestDAGValidateMissingDependency(t *testing.T) {
	d := &DAG{
		Name: "test",
		Tasks: map[string]*Task{
			"a": {Name: "a", Cmd: "echo a", DependsOn: []string{"nonexistent"}},
		},
	}

	err := d.Validate()
	if err == nil {
		t.Fatal("expected missing dependency error, got nil")
	}
}

// TestDAGValidateEmptyName tests validation catches empty workflow name.
func TestDAGValidateEmptyName(t *testing.T) {
	d := &DAG{
		Name: "",
		Tasks: map[string]*Task{
			"a": {Name: "a", Cmd: "echo a"},
		},
	}

	err := d.Validate()
	if err == nil {
		t.Fatal("expected empty name error, got nil")
	}
}

// TestDAGValidateNoTasks tests validation catches empty task list.
func TestDAGValidateNoTasks(t *testing.T) {
	d := &DAG{
		Name:  "test",
		Tasks: map[string]*Task{},
	}

	err := d.Validate()
	if err == nil {
		t.Fatal("expected no tasks error, got nil")
	}
}

// TestDAGValidateMissingCommand tests validation catches missing task command.
func TestDAGValidateMissingCommand(t *testing.T) {
	d := &DAG{
		Name: "test",
		Tasks: map[string]*Task{
			"a": {Name: "a", Cmd: ""},
		},
	}

	err := d.Validate()
	if err == nil {
		t.Fatal("expected missing command error, got nil")
	}
}

// TestDAGValidateInvalidTaskName tests validation catches invalid task names.
func TestDAGValidateInvalidTaskName(t *testing.T) {
	invalidNames := []string{"task 1", "task!", "@bad", "task.name"}

	for _, name := range invalidNames {
		d := &DAG{
			Name: "test",
			Tasks: map[string]*Task{
				name: {Name: name, Cmd: "echo test"},
			},
		}

		err := d.Validate()
		if err == nil {
			t.Errorf("expected invalid name error for %q, got nil", name)
		}
	}
}

// TestDAGValidateValidTaskNames tests validation accepts valid task names.
func TestDAGValidateValidTaskNames(t *testing.T) {
	validNames := []string{"task_1", "A", "deploy-prod", "task123"}

	for _, name := range validNames {
		d := &DAG{
			Name: "test",
			Tasks: map[string]*Task{
				name: {Name: name, Cmd: "echo test"},
			},
		}

		err := d.Validate()
		if err != nil {
			t.Errorf("expected no error for valid name %q, got: %v", name, err)
		}
	}
}

// TestDAGLoadMissingWorkflowFile tests loading a non-existent workflow file.
func TestDAGLoadMissingWorkflowFile(t *testing.T) {
	workflowDir := t.TempDir()
	config.C.Paths.Workflows = workflowDir

	_, err := Load("missing")
	if err == nil {
		t.Fatal("expected error for missing workflow file, got nil")
	}
}

// TestDAGLoadExistingWorkflowFile tests loading an existing workflow file.
func TestDAGLoadExistingWorkflowFile(t *testing.T) {
	workflowDir := t.TempDir()
	config.C.Paths.Workflows = workflowDir

	workflowContent := `
name = "test-workflow"

[tasks.task1]
cmd = "echo Task 1"
retries = 1

[tasks.task2]
cmd = "echo Task 2"
depends_on = ["task1"]
`

	workflowPath := filepath.Join(workflowDir, "test-workflow.toml")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("failed to write workflow file: %v", err)
	}

	dag, err := Load("test-workflow")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if dag.Name != "test-workflow" {
		t.Errorf("expected workflow name 'test-workflow', got: %s", dag.Name)
	}

	if len(dag.Tasks) != 2 {
		t.Errorf("expected 2 tasks, got: %d", len(dag.Tasks))
	}

	if dag.Tasks["task1"].Cmd != "echo Task 1" {
		t.Errorf("expected task1 cmd 'echo Task 1', got: %s", dag.Tasks["task1"].Cmd)
	}

	if dag.Tasks["task2"].DependsOn[0] != "task1" {
		t.Error("expected task2 to depend on task1")
	}
}

// TestDAGLoadFromString tests loading workflow from TOML string.
func TestDAGLoadFromString(t *testing.T) {
	workflowContent := `
name = "string-workflow"

[tasks.task1]
cmd = "echo hello"
`

	dag, err := LoadFromString(workflowContent)
	if err != nil {
		t.Fatalf("LoadFromString failed: %v", err)
	}

	if dag.Name != "string-workflow" {
		t.Errorf("expected name 'string-workflow', got: %s", dag.Name)
	}
}

// TestDAGComputeHash tests hash computation is deterministic.
func TestDAGComputeHash(t *testing.T) {
	d := &DAG{
		Name: "test",
		Tasks: map[string]*Task{
			"a": {Name: "a", Cmd: "echo a"},
			"b": {Name: "b", Cmd: "echo b", DependsOn: []string{"a"}},
		},
	}

	hash1, err := d.ComputeHash()
	if err != nil {
		t.Fatalf("ComputeHash failed: %v", err)
	}

	hash2, err := d.ComputeHash()
	if err != nil {
		t.Fatalf("ComputeHash failed: %v", err)
	}

	if hash1 != hash2 {
		t.Error("expected consistent hash computation")
	}
}

// TestDAGRoots tests Roots() returns tasks with no dependencies.
func TestDAGRoots(t *testing.T) {
	d := &DAG{
		Name: "test",
		Tasks: map[string]*Task{
			"a": {Name: "a", Cmd: "echo a"},
			"b": {Name: "b", Cmd: "echo b", DependsOn: []string{"a"}},
			"c": {Name: "c", Cmd: "echo c"},
		},
	}

	roots := d.Roots()
	if len(roots) != 2 {
		t.Fatalf("expected 2 root tasks, got %d", len(roots))
	}
}
