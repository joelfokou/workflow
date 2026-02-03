package run

import (
	"database/sql"
	"encoding/json"
	"time"
)

type WorkflowStatus string
type TaskStatus string

const (
	StatusPending WorkflowStatus = "pending"
	StatusRunning WorkflowStatus = "running"
	StatusSuccess WorkflowStatus = "success"
	StatusFailed  WorkflowStatus = "failed"
)

const (
	TaskPending TaskStatus = "pending"
	TaskRunning TaskStatus = "running"
	TaskSuccess TaskStatus = "success"
	TaskFailed  TaskStatus = "failed"
)

const dbschema = `
CREATE TABLE IF NOT EXISTS workflow_runs (
    id TEXT PRIMARY KEY,
    workflow TEXT NOT NULL,
    workflow_hash TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at TIMESTAMP NOT NULL,
    ended_at TIMESTAMP,
    exit_code INTEGER,
    meta TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS task_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at TIMESTAMP NOT NULL,
    ended_at TIMESTAMP,
    attempts INTEGER NOT NULL DEFAULT 0,
    exit_code INTEGER,
    log_path TEXT,
    last_error TEXT,
    FOREIGN KEY (run_id) REFERENCES workflow_runs(id)
);

CREATE INDEX IF NOT EXISTS idx_task_runs_run_id ON task_runs(run_id);
`

const (
	QueryCreateWorkflowRun = `
        INSERT INTO workflow_runs (id, workflow, workflow_hash, status, started_at, created_at)
        VALUES (?, ?, ?, ?, ?, ?)
    `

	QueryUpdateWorkflowRun = `
        UPDATE workflow_runs
        SET status = ?, ended_at = ?, exit_code = ?, meta = ?
        WHERE id = ?
    `

	QueryLoadWorkflowRun = `
        SELECT id, workflow, workflow_hash, status, started_at, ended_at, exit_code, meta, created_at
        FROM workflow_runs
        WHERE id = ?
    `

	QueryListRuns = `
		SELECT id, workflow, workflow_hash, status, started_at, ended_at, exit_code, meta, created_at
		FROM workflow_runs
		WHERE (? = '' OR workflow = ?)
			AND (? = '' OR status = ?)
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	QueryCreateTaskRun = `
        INSERT INTO task_runs (run_id, name, status, started_at, ended_at, attempts, exit_code, log_path, last_error)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `

	QueryUpdateTaskRun = `
        UPDATE task_runs
        SET status = ?, ended_at = ?, attempts = ?, exit_code = ?, log_path = ?, last_error = ?
        WHERE id = ?
    `

	QueryLoadTaskRuns = `
        SELECT id, run_id, name, status, started_at, ended_at, attempts, exit_code, log_path, last_error
        FROM task_runs
        WHERE run_id = ?
    `

	QueryGetTaskRun = `
		SELECT id, run_id, name, status, started_at, ended_at, attempts, exit_code, log_path, last_error
		FROM task_runs
		WHERE run_id = ? AND name = ?
	`
)

// TaskPlan represents the plan for a single task in a workflow.
type TaskPlan struct {
	Order     int      `json:"order"`
	Name      string   `json:"name"`
	Cmd       string   `json:"cmd"`
	DependsOn []string `json:"depends_on"`
	Retries   int      `json:"retries"`
}

// WorkflowPlan represents the plan for a workflow.
type WorkflowPlan struct {
	Workflow string     `json:"workflow"`
	Tasks    []TaskPlan `json:"tasks"`
}

// WorkflowRun represents a single execution of a workflow.
type WorkflowRun struct {
	ID           string         `db:"id"`
	Workflow     string         `db:"workflow"`
	WorkflowHash string         `db:"workflow_hash"`
	Status       WorkflowStatus `db:"status"`
	StartedAt    time.Time      `db:"started_at"`
	EndedAt      sql.NullTime   `db:"ended_at"`
	ExitCode     sql.NullInt64  `db:"exit_code"`
	Meta         sql.NullString `db:"meta"` // JSON string
	CreatedAt    time.Time      `db:"created_at"`
}

// TaskRun represents the execution details of a single task within a workflow.
type TaskRun struct {
	ID        int64         `db:"id"`
	RunID     string        `db:"run_id"` // Foreign key to WorkflowRun
	Name      string        `db:"name"`
	Status    TaskStatus    `db:"status"`
	StartedAt time.Time     `db:"started_at"`
	EndedAt   sql.NullTime  `db:"ended_at"`
	Attempts  int           `db:"attempts"`
	ExitCode  sql.NullInt64 `db:"exit_code"`
	LogPath   string        `db:"log_path"`
	LastError string        `db:"last_error"`
}

// MarshalMeta converts Meta map to JSON string for storage
func (w *WorkflowRun) MarshalMeta(meta map[string]interface{}) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	w.Meta = sql.NullString{String: string(data), Valid: true}
	return nil
}

// UnmarshalMeta converts JSON string back to Meta map
func (w *WorkflowRun) UnmarshalMeta() (map[string]interface{}, error) {
	var meta map[string]interface{}
	if !w.Meta.Valid {
		return meta, nil
	}
	err := json.Unmarshal([]byte(w.Meta.String), &meta)
	return meta, err
}

// MarshalRun converts a WorkflowRun to JSON bytes.
func MarshalRun(w *WorkflowRun) ([]byte, error) {
	type runOutput struct {
		ID        string      `json:"id"`
		Workflow  string      `json:"workflow"`
		Status    string      `json:"status"`
		StartedAt time.Time   `json:"started_at"`
		EndedAt   *time.Time  `json:"ended_at,omitempty"`
		ExitCode  *int64      `json:"exit_code,omitempty"`
		Meta      interface{} `json:"meta,omitempty"`
		CreatedAt time.Time   `json:"created_at"`
	}

	var endedAt *time.Time
	if w.EndedAt.Valid {
		endedAt = &w.EndedAt.Time
	}

	var exitCode *int64
	if w.ExitCode.Valid {
		exitCode = &w.ExitCode.Int64
	}

	var meta interface{}
	if w.Meta.Valid {
		_ = json.Unmarshal([]byte(w.Meta.String), &meta)
	}

	return json.Marshal(runOutput{
		ID:        w.ID,
		Workflow:  w.Workflow,
		Status:    string(w.Status),
		StartedAt: w.StartedAt,
		EndedAt:   endedAt,
		ExitCode:  exitCode,
		Meta:      meta,
		CreatedAt: w.CreatedAt,
	})
}
