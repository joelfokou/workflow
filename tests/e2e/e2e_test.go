// Package e2e contains end-to-end tests for the complete workflow application.
package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

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

// TestE2ECompleteWorkflow tests the entire CLI workflow from start to finish.
func TestE2ECompleteWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	fs := helpers.NewTestFS(t)
	defer fs.Cleanup()

	// Setup project structure
	if err := setupProject(fs); err != nil {
		t.Fatalf("failed to setup project: %v", err)
	}

	// Test init command
	t.Run("init", func(t *testing.T) {
		testInit(t, fs)
	})

	// Test validate command
	t.Run("validate", func(t *testing.T) {
		testValidate(t, fs)
	})

	// Test list command
	t.Run("list", func(t *testing.T) {
		testList(t, fs)
	})

	// Test graph command
	t.Run("graph", func(t *testing.T) {
		testGraph(t, fs)
	})

	// Test run command
	t.Run("run", func(t *testing.T) {
		testRun(t, fs)
	})

	// Test logs command
	t.Run("logs", func(t *testing.T) {
		testLogs(t, fs)
	})

	// Test runs command
	t.Run("runs", func(t *testing.T) {
		testRuns(t, fs)
	})

	// Test resume command
	t.Run("resume", func(t *testing.T) {
		testResume(t, fs)
	})
}

// TestE2EErrorHandling tests error scenarios across the CLI.
func TestE2EErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	fs := helpers.NewTestFS(t)
	defer fs.Cleanup()

	t.Run("invalid_workflow", func(t *testing.T) {
		testInvalidWorkflow(t, fs)
	})

	t.Run("missing_workflow", func(t *testing.T) {
		testMissingWorkflow(t, fs)
	})

	t.Run("cycle_detection", func(t *testing.T) {
		testCycleDetection(t, fs)
	})

	t.Run("missing_dependency", func(t *testing.T) {
		testMissingDependency(t, fs)
	})
}

// TestE2EConfigManagement tests configuration loading and overrides.
func TestE2EConfigManagement(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	fs := helpers.NewTestFS(t)
	defer fs.Cleanup()

	t.Run("env_var_override", func(t *testing.T) {
		testEnvVarOverride(t, fs)
	})

	t.Run("config_file_override", func(t *testing.T) {
		testConfigFileOverride(t, fs)
	})
}

// setupProject creates the necessary project structure for E2E testing.
func setupProject(fs *helpers.TestFS) error {
	// Build the wf binary
	output, err := exec.Command("go", "build", "-o", filepath.Join(fs.Path("."), "wf"), "../..").CombinedOutput()
	if err != nil {
		panic(fmt.Sprintf("failed to build wf binary: %v\noutput: %s", err, string(output)))
	}

	// Create directories
	dirs := []string{"workflows", "logs"}
	for _, dir := range dirs {
		if err := os.MkdirAll(fs.Path(dir), 0755); err != nil {
			return err
		}
	}

	// Create database file
	dbPath := filepath.Join(fs.Path("test.db"))
	if _, err := os.Create(dbPath); err != nil {
		return err
	}

	// Create example workflows
	workflows := map[string]string{
		"simple.toml": helpers.SimpleWorkflow(),
		"multi.toml":  helpers.MultiTaskWorkflow(),
		"resume.toml": helpers.ResumeWorkflow(),
	}

	for name, content := range workflows {
		path := filepath.Join(fs.Path("workflows"), name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}

// testInit tests the init command.
func testInit(t *testing.T, fs *helpers.TestFS) {
	cmd := newCmd(fs, "init")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("init command failed: %v\noutput: %s", err, string(output))
	}

	// Verify directories exist
	for _, dir := range []string{"workflows", "logs"} {
		path := fs.Path(dir)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected directory %s to exist", path)
		}
	}

	// Verify database was created
	dbPath := filepath.Join(fs.Path("test.db"))
	if _, err := os.Stat(dbPath); err != nil {
		t.Errorf("expected database file at %s", dbPath)
	}

	if !strings.Contains(string(output), "initialised") {
		t.Error("expected success message in output, got :", string(output))
	}
}

// testValidate tests the validate command.
func testValidate(t *testing.T, fs *helpers.TestFS) {
	// Test validating all workflows
	cmd := newCmd(fs, "validate")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("validate command failed: %v\noutput: %s", err, string(output))
	}

	// Should succeed for valid workflows
	validCount := strings.Count(string(output), "✓")
	if validCount == 0 {
		t.Error("expected at least one valid workflow")
	}

	// Should show invalid workflows
	if strings.Contains(string(output), "invalid") {
		invalidCount := strings.Count(string(output), "✗")
		if invalidCount == 0 {
			t.Error("expected invalid workflow to be reported")
		}
	}

	// Test validating specific workflow
	cmd = newCmd(fs, "validate", "simple")
	output, err = cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("validate simple command failed: %v", err)
	}

	if !strings.Contains(string(output), "simple") {
		t.Error("expected workflow name in output")
	}
}

// testList tests the list command.
func testList(t *testing.T, fs *helpers.TestFS) {
	cmd := newCmd(fs, "list")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("list command failed: %v", err)
	}

	// Verify output contains workflow names
	if !strings.Contains(string(output), "simple") {
		t.Error("expected 'simple' workflow in list")
	}

	if !strings.Contains(string(output), "multi") {
		t.Error("expected 'multi' workflow in list")
	}

	// Test with JSON output
	cmd = newCmd(fs, "list", "--json")
	output, err = cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("list --json command failed: %v", err)
	}

	if !strings.Contains(string(output), "\"name\"") {
		t.Error("expected JSON output format")
	}
}

// testGraph tests the graph command.
func testGraph(t *testing.T, fs *helpers.TestFS) {
	// Test ASCII format (default)
	cmd := newCmd(fs, "graph", "simple")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("graph command failed: %v", err)
	}

	if len(string(output)) == 0 {
		t.Error("expected graph output")
	}

	// Test DOT format
	cmd = newCmd(fs, "graph", "simple", "--format", "dot")
	output, err = cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("graph --format dot command failed: %v", err)
	}

	if !strings.Contains(string(output), "digraph") {
		t.Error("expected DOT format output")
	}

	// Test JSON format
	cmd = newCmd(fs, "graph", "simple", "--format", "json")
	output, err = cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("graph --format json command failed: %v", err)
	}

	if !strings.Contains(string(output), "\"name\"") {
		t.Error("expected JSON format output")
	}
}

// testRun tests the run command.
func testRun(t *testing.T, fs *helpers.TestFS) {
	cmd := newCmd(fs, "run", "simple")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("run command failed: %v\noutput: %s", err, string(output))
	}

	// Verify database has run recorded
	dbPath := filepath.Join(fs.Path("test.db"))
	store, err := run.NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer store.Close()

	runs, err := store.ListRuns("simple", "", 10, 0)
	if err != nil {
		t.Fatalf("failed to list runs: %v", err)
	}

	if len(runs) == 0 {
		t.Fatal("expected run to be recorded in database")
	}

	if runs[0].Status != run.StatusSuccess {
		t.Errorf("expected status success, got %s", runs[0].Status)
	}

	// Test dry-run mode
	cmd = newCmd(fs, "run", "simple", "--dry-run")
	output, err = cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("run --dry-run command failed: %v", err)
	}

	if !strings.Contains(string(output), "DRY RUN MODE") {
		t.Error("expected dry run output")
	}

	// Test JSON output
	cmd = newCmd(fs, "run", "simple", "--dry-run", "--json")
	output, err = cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("run --dry-run --json command failed: %v", err)
	}

	if !strings.Contains(string(output), "\"workflow\"") {
		t.Error("expected JSON output format")
	}
}

// testLogs tests the logs command.
func testLogs(t *testing.T, fs *helpers.TestFS) {
	// First run a workflow
	cmd := newCmd(fs, "run", "simple")
	if _, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run command failed: %v", err)
	}

	// Get the run ID
	dbPath := filepath.Join(fs.Path("test.db"))
	store, err := run.NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	runs, err := store.ListRuns("simple", "", 10, 0)
	store.Close()

	if err != nil || len(runs) == 0 {
		t.Fatal("no runs found to display logs for")
	}

	runID := runs[0].ID

	// Test logs command with run ID
	cmd = newCmd(fs, "logs", runID)
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("logs command failed: %v", err)
	}

	if len(string(output)) == 0 {
		t.Error("expected log output")
	}
}

// testRuns tests the runs command.
func testRuns(t *testing.T, fs *helpers.TestFS) {
	// First run a workflow
	cmd := newCmd(fs, "run", "simple")
	if _, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run command failed: %v", err)
	}

	// Test runs command
	cmd = newCmd(fs, "runs")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("runs command failed: %v", err)
	}

	if !strings.Contains(string(output), "simple") {
		t.Error("expected workflow name in runs output")
	}

	// Test with JSON output
	cmd = newCmd(fs, "runs", "--json")
	output, err = cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("runs --json command failed: %v", err)
	}

	if !strings.Contains(string(output), "\"workflow\"") {
		t.Error("expected JSON output format")
	}

	// Test with filters
	cmd = newCmd(fs, "runs", "--workflow", "simple")
	output, err = cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("runs --workflow command failed: %v", err)
	}

	if !strings.Contains(string(output), "simple") {
		t.Error("expected filtered results")
	}
}

// testResume tests the resume command.
func testResume(t *testing.T, fs *helpers.TestFS) {
	// Run a workflow that will fail
	cmd := newCmd(fs, "run", "resume")
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatal("expected run to fail for 'resume' workflow")
	}

	// Update the workflow to fix the failure
	resumeWorkflowFixed := helpers.ResumeWorkflowFixed()
	path := filepath.Join(fs.Path("workflows"), "resume.toml")
	if err := os.WriteFile(path, []byte(resumeWorkflowFixed), 0644); err != nil {
		t.Fatalf("failed to update resume workflow: %v", err)
	}

	// Retrieve the run ID of the failed run
	dbPath := filepath.Join(fs.Path("test.db"))
	store, err := run.NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer store.Close()

	runs, err := store.ListRuns("resume", "", 10, 0)
	if err != nil || len(runs) == 0 {
		t.Fatal("no runs found to resume")
	}

	failedRunID := runs[0].ID

	// Resume the workflow
	cmd = newCmd(fs, "resume", failedRunID)
	output, err = cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("resume command failed: %v\noutput: %s", err, string(output))
	}

	if !strings.Contains(string(output), "Resuming workflow run") {
		t.Error("expected resume output")
	}

	runs, err = store.ListRuns("resume", "", 10, 0)
	if err != nil {
		t.Fatalf("failed to list runs: %v", err)
	}

	if len(runs) == 0 {
		t.Fatal("expected run to be recorded in database")
	}

	if runs[0].Status != run.StatusSuccess {
		t.Errorf("expected status success after resume, got %s", runs[0].Status)
	}
}

// testInvalidWorkflow tests behavior with invalid workflow.
func testInvalidWorkflow(t *testing.T, fs *helpers.TestFS) {
	setupProject(fs)

	cmd := newCmd(fs, "validate", "invalid")
	_, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatal("expected validate to fail for invalid workflow")
	}
}

// testMissingWorkflow tests behavior with missing workflow.
func testMissingWorkflow(t *testing.T, fs *helpers.TestFS) {
	setupProject(fs)

	cmd := newCmd(fs, "run", "nonexistent")
	_, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatal("expected run to fail for nonexistent workflow")
	}
}

// testCycleDetection tests cycle detection in workflows.
func testCycleDetection(t *testing.T, fs *helpers.TestFS) {
	setupProject(fs)

	cycleWorkflow := helpers.CycleWorkflow()

	path := filepath.Join(fs.Path("workflows"), "cycle.toml")
	if err := os.WriteFile(path, []byte(cycleWorkflow), 0644); err != nil {
		t.Fatalf("failed to create cycle workflow: %v", err)
	}

	cmd := newCmd(fs, "validate", "cycle")
	_, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatal("expected validation to fail for cyclic workflow")
	}
}

// testMissingDependency tests detection of missing dependencies.
func testMissingDependency(t *testing.T, fs *helpers.TestFS) {
	setupProject(fs)

	cmd := newCmd(fs, "validate", "invalid")
	_, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatal("expected validation to fail for missing dependency")
	}
}

// testEnvVarOverride tests environment variable configuration override.
func testEnvVarOverride(t *testing.T, fs *helpers.TestFS) {
	setupProject(fs)

	cmd := newCmd(fs, "init")
	cmd.Env = append(cmd.Env,
		fmt.Sprintf("WF_PATHS_WORKFLOWS=%s", fs.Path("workflows")),
		fmt.Sprintf("WF_PATHS_LOGS=%s", fs.Path("logs")),
		fmt.Sprintf("WF_PATHS_DATABASE=%s", fs.Path("test.db")),
	)

	_, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("init with env vars failed: %v", err)
	}
}

// testConfigFileOverride tests config file configuration override.
func testConfigFileOverride(t *testing.T, fs *helpers.TestFS) {
	setupProject(fs)

	configContent := fmt.Sprintf(`
paths:
  workflows: %s
  logs: %s
  database: %s
`, fs.Path("workflows"), fs.Path("logs"), fs.Path("test.db"))

	configPath := filepath.Join(fs.Path("."), "workflow.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	cmd := newCmd(fs, "--config", configPath, "init")
	_, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("init with config file failed: %v", err)
	}
}

// newCmd creates a new command with proper environment.
func newCmd(fs *helpers.TestFS, args ...string) *exec.Cmd {
	binary := "./wf"

	cmd := exec.Command(binary, args...)
	cmd.Dir = fs.Path(".")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("WF_PATHS_WORKFLOWS=%s", fs.Path("workflows")),
		fmt.Sprintf("WF_PATHS_LOGS=%s", fs.Path("logs")),
		fmt.Sprintf("WF_PATHS_DATABASE=%s", fs.Path("test.db")),
	)
	return cmd
}
