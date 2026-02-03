// Package run implements storage for workflow runs.
package run

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// Store manages the persistence of WorkflowRun instances using SQLite.
type Store struct {
	db *sql.DB
}

// NewStore initialises a new Store with SQLite database at the given path.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		return nil, err
	}

	return store, nil
}

// migrate creates the necessary tables if they don't exist.
func (s *Store) migrate() error {
	_, err := s.db.Exec(dbschema)
	return err
}

// NewWorkflowRun creates and stores a new WorkflowRun with the given workflow name and DAG hash.
func (s *Store) NewWorkflowRun(workflow string, dagHash string) (*WorkflowRun, error) {
	id := uuid.New().String()

	run := &WorkflowRun{
		ID:           id,
		Workflow:     workflow,
		WorkflowHash: dagHash,
		Status:       StatusRunning,
		StartedAt:    time.Now(),
		CreatedAt:    time.Now(),
	}

	_, err := s.db.Exec(QueryCreateWorkflowRun, run.ID, run.Workflow, run.WorkflowHash, run.Status, run.StartedAt, run.CreatedAt)
	if err != nil {
		return nil, err
	}

	return run, nil
}

// Update persists changes to an existing WorkflowRun.
func (s *Store) Update(run *WorkflowRun) error {
	_, err := s.db.Exec(QueryUpdateWorkflowRun, run.Status, run.EndedAt, run.ExitCode, run.Meta, run.ID)
	return err
}

// Load retrieves a WorkflowRun by its ID.
func (s *Store) Load(id string) (*WorkflowRun, error) {
	run := &WorkflowRun{}
	err := s.db.QueryRow(QueryLoadWorkflowRun, id).Scan(&run.ID, &run.Workflow, &run.WorkflowHash, &run.Status, &run.StartedAt, &run.EndedAt, &run.ExitCode, &run.Meta, &run.CreatedAt)
	if err != nil {
		return nil, err
	}

	return run, nil
}

// ListRuns retrieves workflow runs with optional filtering and pagination.
func (s *Store) ListRuns(workflow, status string, limit, offset int) ([]*WorkflowRun, error) {
	rows, err := s.db.Query(QueryListRuns, workflow, workflow, status, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*WorkflowRun
	for rows.Next() {
		run := &WorkflowRun{}
		if err := rows.Scan(&run.ID, &run.Workflow, &run.WorkflowHash, &run.Status, &run.StartedAt, &run.EndedAt, &run.ExitCode, &run.Meta, &run.CreatedAt); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}

	return runs, rows.Err()
}

// SaveTaskRun persists a TaskRun to the database.
func (s *Store) SaveTaskRun(task *TaskRun) error {
	result, err := s.db.Exec(QueryCreateTaskRun, task.RunID, task.Name, task.Status, task.StartedAt, task.EndedAt, task.Attempts, task.ExitCode, task.LogPath, task.LastError)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	task.ID = id
	return nil
}

// UpdateTaskRun updates an existing TaskRun.
func (s *Store) UpdateTaskRun(task *TaskRun) error {
	_, err := s.db.Exec(QueryUpdateTaskRun, task.Status, task.EndedAt, task.Attempts, task.ExitCode, task.LogPath, task.LastError, task.ID)
	return err
}

// LoadTaskRuns retrieves all TaskRuns for a given WorkflowRun.
func (s *Store) LoadTaskRuns(runID string) ([]TaskRun, error) {
	rows, err := s.db.Query(QueryLoadTaskRuns, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []TaskRun
	for rows.Next() {
		var task TaskRun
		if err := rows.Scan(&task.ID, &task.RunID, &task.Name, &task.Status, &task.StartedAt, &task.EndedAt, &task.Attempts, &task.ExitCode, &task.LogPath, &task.LastError); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// GetTaskRun retrieves a specific TaskRun by run ID and task name.
func (s *Store) GetTaskRun(runID, taskName string) (*TaskRun, error) {
	task := &TaskRun{}
	err := s.db.QueryRow(QueryGetTaskRun, runID, taskName).Scan(&task.ID, &task.RunID, &task.Name, &task.Status, &task.StartedAt, &task.EndedAt, &task.Attempts, &task.ExitCode, &task.LogPath, &task.LastError)
	if err != nil {
		return nil, err
	}

	return task, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}
