// Package executor provides functionality to execute workflows defined as directed acyclic graphs (DAGs).
package executor

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/joelfokou/workflow/internal/config"
	"github.com/joelfokou/workflow/internal/dag"
	"github.com/joelfokou/workflow/internal/logger"
	"github.com/joelfokou/workflow/internal/run"
	"go.uber.org/zap"
)

// Executor is responsible for executing workflows defined as DAGs.
type Executor struct {
	RunStore           *run.Store
	DefaultTaskTimeout time.Duration // Optional global timeout per task (0 = none)
}

// NewExecutor is a creates a new Executor with the given RunStore.
func NewExecutor(store *run.Store) *Executor {
	return &Executor{
		RunStore:           store,
		DefaultTaskTimeout: 0,
	}
}

// Run executes the given DAG workflow.
func (e *Executor) Run(ctx context.Context, d *dag.DAG) error {
	logger.L().Info("running workflow", zap.String("workflow", d.Name))
	fmt.Println("Running workflow:", d.Name)

	dagHash, err := d.ComputeHash()
	if err != nil {
		return err
	}

	wr, err := e.RunStore.NewWorkflowRun(d.Name, dagHash)
	if err != nil {
		return err
	}

	order, err := d.TopologicalSort()
	if err != nil {
		now := time.Now()
		wr.Status = run.StatusFailed
		wr.EndedAt = sql.NullTime{Time: now, Valid: true}
		_ = e.RunStore.Update(wr)
		logger.L().Error("topological sort error", zap.String("workflow", d.Name), zap.Error(err))
		return fmt.Errorf("topological sort error: %w", err)
	}

	for _, t := range order {
		logger.L().Info("running task", zap.String("task", t.Name))
		fmt.Println("Running task:", t.Name)

		select {
		case <-ctx.Done():
			now := time.Now()
			wr.Status = run.StatusFailed
			wr.EndedAt = sql.NullTime{Time: now, Valid: true}
			_ = e.RunStore.Update(wr)
			logger.L().Error("workflow cancelled", zap.String("workflow", d.Name), zap.Error(ctx.Err()))
			return fmt.Errorf("workflow cancelled: %w", ctx.Err())
		default:
		}

		tr := &run.TaskRun{
			RunID:     wr.ID,
			Name:      t.Name,
			Status:    run.TaskRunning,
			StartedAt: time.Now(),
			Attempts:  0,
		}
		err = e.RunStore.SaveTaskRun(tr)
		if err != nil {
			logger.L().Error("failed to save task run", zap.String("task", t.Name), zap.Error(err))
			return err
		}

		for attempt := 1; attempt <= t.Retries+1; attempt++ {
			tr.Attempts = attempt

			cmd := exec.CommandContext(ctx, "bash", "-c", t.Cmd)
			setCmdProcessAttrs(cmd)

			// Capture output and execute command
			out, err := cmd.CombinedOutput()

			// Ensure log directory exists
			dir := filepath.Join(config.C.Paths.Logs, wr.ID)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			logPath := filepath.Join(dir, fmt.Sprintf("%s_%d.log", t.Name, attempt))
			tr.LogPath = logPath
			os.WriteFile(logPath, out, 0644)

			// Extract exit code from error
			if exitErr, ok := err.(*exec.ExitError); ok {
				code := int64(exitErr.ExitCode())
				tr.ExitCode = sql.NullInt64{Int64: code, Valid: true}
				tr.LastError = exitErr.Error()
				_ = e.RunStore.UpdateTaskRun(tr)
			} else if err != nil {
				// Command execution error (not an exit code error)
				tr.LastError = err.Error()
				tr.ExitCode = sql.NullInt64{Int64: 1, Valid: true}
				_ = e.RunStore.UpdateTaskRun(tr)
			} else {
				// Success
				tr.ExitCode = sql.NullInt64{Int64: 0, Valid: true}
				_ = e.RunStore.UpdateTaskRun(tr)
			}

			if err == nil {
				now := time.Now()
				tr.Status = run.TaskSuccess
				tr.EndedAt = sql.NullTime{Time: now, Valid: true}
				_ = e.RunStore.UpdateTaskRun(tr)

				logger.L().Info("task completed", zap.String("task", t.Name))
				fmt.Println("Task completed:", t.Name)
				break
			}

			if attempt == t.Retries+1 {
				now := time.Now()
				tr.Status = run.TaskFailed
				tr.EndedAt = sql.NullTime{Time: now, Valid: true}
				_ = e.RunStore.UpdateTaskRun(tr)

				wr.Status = run.StatusFailed
				wr.EndedAt = sql.NullTime{Time: now, Valid: true}
				_ = e.RunStore.Update(wr)

				logger.L().Error("task failed => workflow failed", zap.String("task", t.Name), zap.String("workflow", d.Name), zap.Error(err))
				return fmt.Errorf("task %s failed => workflow %s failed: %w", t.Name, d.Name, err)
			}

			logger.L().Debug("retrying task",
				zap.String("workflow", d.Name),
				zap.String("task", t.Name),
				zap.Int("attempt", attempt),
			)
			fmt.Println("Retrying:", t.Name)
		}
	}

	now := time.Now()
	wr.Status = run.StatusSuccess
	wr.EndedAt = sql.NullTime{Time: now, Valid: true}
	e.RunStore.Update(wr)

	logger.L().Info("workflow completed", zap.String("workflow", d.Name))
	fmt.Println("Workflow completed:", d.Name)
	return nil
}

func (e *Executor) Resume(ctx context.Context, wr *run.WorkflowRun) error {
	fmt.Printf("Resuming workflow run: %s\n", wr.ID)
	logger.L().Info("resuming workflow", zap.String("workflow", wr.Workflow), zap.String("run_id", wr.ID))

	d, err := dag.Load(wr.Workflow)
	if err != nil {
		logger.L().Error("failed to load workflow", zap.String("workflow", wr.Workflow), zap.Error(err))
		return fmt.Errorf("failed to load workflow '%s': %w", wr.Workflow, err)
	}

	order, err := d.TopologicalSort()
	if err != nil {
		now := time.Now()
		wr.Status = run.StatusFailed
		wr.EndedAt = sql.NullTime{Time: now, Valid: true}
		_ = e.RunStore.Update(wr)
		logger.L().Error("topological sort error", zap.String("workflow", d.Name), zap.Error(err))
		return fmt.Errorf("topological sort error: %w", err)
	}

	for _, t := range order {
		// Check if task was already completed
		tr, err := e.RunStore.GetTaskRun(wr.ID, t.Name)
		if err != nil && err != sql.ErrNoRows {
			logger.L().Error("failed to load task run", zap.String("task", t.Name), zap.Error(err))
			return err
		}
		if tr != nil && tr.Status == run.TaskSuccess {
			logger.L().Info("skipping completed task", zap.String("task", t.Name))
			fmt.Println("Skipping completed task:", t.Name)
			continue
		}
		logger.L().Info("running task", zap.String("task", t.Name))
		fmt.Println("Running task:", t.Name)

		select {
		case <-ctx.Done():
			now := time.Now()
			wr.Status = run.StatusFailed
			wr.EndedAt = sql.NullTime{Time: now, Valid: true}
			_ = e.RunStore.Update(wr)
			logger.L().Error("workflow cancelled", zap.String("workflow", d.Name), zap.Error(ctx.Err()))
			return fmt.Errorf("workflow cancelled: %w", ctx.Err())
		default:
		}

		if tr == nil {
			tr = &run.TaskRun{
				RunID:     wr.ID,
				Name:      t.Name,
				Status:    run.TaskRunning,
				StartedAt: time.Now(),
				Attempts:  0,
			}
			err = e.RunStore.SaveTaskRun(tr)
			if err != nil {
				logger.L().Error("failed to save task run", zap.String("task", t.Name), zap.Error(err))
				return err
			}
		}

		for attempt := 1; attempt <= t.Retries+1; attempt++ {
			tr.Attempts = attempt

			cmd := exec.CommandContext(ctx, "bash", "-c", t.Cmd)
			setCmdProcessAttrs(cmd)

			// Capture output and execute command
			out, err := cmd.CombinedOutput()

			// Ensure log directory exists
			dir := filepath.Join(config.C.Paths.Logs, wr.ID)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			logPath := filepath.Join(dir, fmt.Sprintf("%s_%d.log", t.Name, attempt))
			tr.LogPath = logPath
			os.WriteFile(logPath, out, 0644)

			// Extract exit code from error
			if exitErr, ok := err.(*exec.ExitError); ok {
				code := int64(exitErr.ExitCode())
				tr.ExitCode = sql.NullInt64{Int64: code, Valid: true}
				tr.LastError = exitErr.Error()
				_ = e.RunStore.UpdateTaskRun(tr)
			} else if err != nil {
				// Command execution error (not an exit code error)
				tr.LastError = err.Error()
				tr.ExitCode = sql.NullInt64{Int64: 1, Valid: true}
				_ = e.RunStore.UpdateTaskRun(tr)
			} else {
				// Success
				tr.ExitCode = sql.NullInt64{Int64: 0, Valid: true}
				_ = e.RunStore.UpdateTaskRun(tr)
			}

			if err == nil {
				now := time.Now()
				tr.Status = run.TaskSuccess
				tr.EndedAt = sql.NullTime{Time: now, Valid: true}
				_ = e.RunStore.UpdateTaskRun(tr)

				logger.L().Info("task completed", zap.String("task", t.Name))
				fmt.Println("Task completed:", t.Name)
				break
			}

			if attempt == t.Retries+1 {
				now := time.Now()
				tr.Status = run.TaskFailed
				tr.EndedAt = sql.NullTime{Time: now, Valid: true}
				_ = e.RunStore.UpdateTaskRun(tr)

				wr.Status = run.StatusFailed
				wr.EndedAt = sql.NullTime{Time: now, Valid: true}
				_ = e.RunStore.Update(wr)

				logger.L().Error("task failed => workflow failed", zap.String("task", t.Name), zap.String("workflow", d.Name), zap.Error(err))
				return fmt.Errorf("task %s failed => workflow %s failed: %w", t.Name, d.Name, err)
			}

			logger.L().Debug("retrying task",
				zap.String("workflow", d.Name),
				zap.String("task", t.Name),
				zap.Int("attempt", attempt),
			)
			fmt.Println("Retrying:", t.Name)
		}
	}

	now := time.Now()
	wr.Status = run.StatusSuccess
	wr.EndedAt = sql.NullTime{Time: now, Valid: true}
	e.RunStore.Update(wr)

	logger.L().Info("workflow resumed and completed", zap.String("workflow", d.Name))
	fmt.Println("Workflow resumed and completed:", d.Name)

	return nil
}
