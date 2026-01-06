package db

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/AbdouB/memory/internal/models"
)

// GoalRepository handles goal database operations
type GoalRepository struct {
	db *DB
}

// NewGoalRepository creates a new goal repository
func NewGoalRepository(db *DB) *GoalRepository {
	return &GoalRepository{db: db}
}

// Create creates a new goal
func (r *GoalRepository) Create(goal *models.Goal) error {
	// Serialize scope and full goal data
	scopeJSON, err := json.Marshal(goal.Scope)
	if err != nil {
		return err
	}

	goalData, err := json.Marshal(goal)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO goals (
			id, session_id, objective, scope, estimated_complexity,
			created_timestamp, is_completed, goal_data, status, beads_issue_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = r.db.Exec(query,
		goal.ID,
		goal.SessionID,
		goal.Objective,
		string(scopeJSON),
		goal.EstimatedComplexity,
		goal.CreatedTimestamp,
		goal.IsCompleted,
		string(goalData),
		goal.Status,
		goal.BeadsIssueID,
	)
	return err
}

// Get retrieves a goal by ID
func (r *GoalRepository) Get(goalID string) (*models.Goal, error) {
	var goal models.Goal
	var goalData string

	query := `SELECT id, session_id, objective, scope, estimated_complexity, 
	          created_timestamp, completed_timestamp, is_completed, goal_data, 
	          status, beads_issue_id FROM goals WHERE id = ?`

	row := r.db.QueryRow(query, goalID)
	err := row.Scan(
		&goal.ID,
		&goal.SessionID,
		&goal.Objective,
		&goal.ScopeJSON,
		&goal.EstimatedComplexity,
		&goal.CreatedTimestamp,
		&goal.CompletedTimestamp,
		&goal.IsCompleted,
		&goalData,
		&goal.Status,
		&goal.BeadsIssueID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Deserialize scope
	if err := json.Unmarshal([]byte(goal.ScopeJSON), &goal.Scope); err != nil {
		return nil, err
	}

	// Deserialize full goal data for other fields
	if err := json.Unmarshal([]byte(goalData), &goal); err != nil {
		return nil, err
	}

	return &goal, nil
}

// List lists goals with optional filtering
func (r *GoalRepository) List(sessionID string, completed *bool, limit int) ([]*models.Goal, error) {
	var goals []*models.Goal
	var query string
	var args []interface{}

	if sessionID != "" && completed != nil {
		query = `SELECT goal_data FROM goals WHERE session_id = ? AND is_completed = ? ORDER BY created_timestamp DESC LIMIT ?`
		args = []interface{}{sessionID, *completed, limit}
	} else if sessionID != "" {
		query = `SELECT goal_data FROM goals WHERE session_id = ? ORDER BY created_timestamp DESC LIMIT ?`
		args = []interface{}{sessionID, limit}
	} else if completed != nil {
		query = `SELECT goal_data FROM goals WHERE is_completed = ? ORDER BY created_timestamp DESC LIMIT ?`
		args = []interface{}{*completed, limit}
	} else {
		query = `SELECT goal_data FROM goals ORDER BY created_timestamp DESC LIMIT ?`
		args = []interface{}{limit}
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var goalData string
		if err := rows.Scan(&goalData); err != nil {
			return nil, err
		}

		var goal models.Goal
		if err := json.Unmarshal([]byte(goalData), &goal); err != nil {
			return nil, err
		}
		goals = append(goals, &goal)
	}

	return goals, rows.Err()
}

// Complete marks a goal as completed
func (r *GoalRepository) Complete(goalID string, reason string) error {
	now := float64(time.Now().UnixMilli()) / 1000.0
	query := `
		UPDATE goals SET 
			is_completed = 1,
			completed_timestamp = ?,
			status = 'complete'
		WHERE id = ?
	`
	_, err := r.db.Exec(query, now, goalID)
	return err
}

// UpdateStatus updates a goal's status
func (r *GoalRepository) UpdateStatus(goalID string, status models.GoalStatus) error {
	query := `UPDATE goals SET status = ? WHERE id = ?`
	_, err := r.db.Exec(query, status, goalID)
	return err
}

// SubtaskRepository handles subtask database operations
type SubtaskRepository struct {
	db *DB
}

// NewSubtaskRepository creates a new subtask repository
func NewSubtaskRepository(db *DB) *SubtaskRepository {
	return &SubtaskRepository{db: db}
}

// Create creates a new subtask
func (r *SubtaskRepository) Create(subtask *models.SubTask) error {
	subtaskData, err := json.Marshal(subtask)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO subtasks (
			id, goal_id, description, status, epistemic_importance,
			estimated_tokens, notes, created_timestamp, subtask_data
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = r.db.Exec(query,
		subtask.ID,
		subtask.GoalID,
		subtask.Description,
		subtask.Status,
		subtask.EpistemicImportance,
		subtask.EstimatedTokens,
		subtask.Notes,
		subtask.CreatedTimestamp,
		string(subtaskData),
	)
	return err
}

// Get retrieves a subtask by ID
func (r *SubtaskRepository) Get(subtaskID string) (*models.SubTask, error) {
	var subtaskData string
	query := `SELECT subtask_data FROM subtasks WHERE id = ?`
	err := r.db.QueryRow(query, subtaskID).Scan(&subtaskData)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var subtask models.SubTask
	if err := json.Unmarshal([]byte(subtaskData), &subtask); err != nil {
		return nil, err
	}
	return &subtask, nil
}

// ListByGoal lists subtasks for a goal
func (r *SubtaskRepository) ListByGoal(goalID string) ([]*models.SubTask, error) {
	var subtasks []*models.SubTask
	query := `SELECT subtask_data FROM subtasks WHERE goal_id = ? ORDER BY created_timestamp ASC`

	rows, err := r.db.Query(query, goalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var subtaskData string
		if err := rows.Scan(&subtaskData); err != nil {
			return nil, err
		}

		var subtask models.SubTask
		if err := json.Unmarshal([]byte(subtaskData), &subtask); err != nil {
			return nil, err
		}
		subtasks = append(subtasks, &subtask)
	}

	return subtasks, rows.Err()
}

// Complete marks a subtask as completed
func (r *SubtaskRepository) Complete(subtaskID string, evidence string) error {
	now := float64(time.Now().UnixMilli()) / 1000.0

	// First get the current subtask data
	subtask, err := r.Get(subtaskID)
	if err != nil {
		return err
	}
	if subtask == nil {
		return sql.ErrNoRows
	}

	subtask.Status = models.TaskStatusCompleted
	subtask.CompletedTimestamp = &now
	subtask.CompletionEvidence = &evidence

	subtaskData, err := json.Marshal(subtask)
	if err != nil {
		return err
	}

	query := `
		UPDATE subtasks SET 
			status = ?,
			completed_timestamp = ?,
			completion_evidence = ?,
			subtask_data = ?
		WHERE id = ?
	`
	_, err = r.db.Exec(query,
		subtask.Status,
		now,
		evidence,
		string(subtaskData),
		subtaskID,
	)
	return err
}

// UpdateStatus updates a subtask's status
func (r *SubtaskRepository) UpdateStatus(subtaskID string, status models.TaskStatus) error {
	query := `UPDATE subtasks SET status = ? WHERE id = ?`
	_, err := r.db.Exec(query, status, subtaskID)
	return err
}
