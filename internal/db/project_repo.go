package db

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/AbdouB/memory/internal/models"
)

// ProjectRepository handles project database operations
type ProjectRepository struct {
	db *DB
}

// NewProjectRepository creates a new project repository
func NewProjectRepository(db *DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// Create creates a new project
func (r *ProjectRepository) Create(project *models.Project) error {
	reposJSON, err := json.Marshal(project.Repos)
	if err != nil {
		return err
	}

	projectData, err := json.Marshal(project)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO projects (
			id, name, description, repos, created_timestamp,
			status, total_sessions, total_goals, project_data
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = r.db.Exec(query,
		project.ID,
		project.Name,
		project.Description,
		string(reposJSON),
		project.CreatedTimestamp,
		project.Status,
		project.TotalSessions,
		project.TotalGoals,
		string(projectData),
	)
	return err
}

// Get retrieves a project by ID
func (r *ProjectRepository) Get(projectID string) (*models.Project, error) {
	var projectData string
	query := `SELECT project_data FROM projects WHERE id = ?`
	err := r.db.QueryRow(query, projectID).Scan(&projectData)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var project models.Project
	if err := json.Unmarshal([]byte(projectData), &project); err != nil {
		return nil, err
	}
	return &project, nil
}

// GetByName retrieves a project by name
func (r *ProjectRepository) GetByName(name string) (*models.Project, error) {
	var projectData string
	query := `SELECT project_data FROM projects WHERE name = ?`
	err := r.db.QueryRow(query, name).Scan(&projectData)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var project models.Project
	if err := json.Unmarshal([]byte(projectData), &project); err != nil {
		return nil, err
	}
	return &project, nil
}

// List lists all projects
func (r *ProjectRepository) List(status *models.ProjectStatus, limit int) ([]*models.Project, error) {
	var projects []*models.Project
	var query string
	var args []interface{}

	if status != nil {
		query = `SELECT project_data FROM projects WHERE status = ? ORDER BY last_activity_timestamp DESC NULLS LAST, created_timestamp DESC LIMIT ?`
		args = []interface{}{*status, limit}
	} else {
		query = `SELECT project_data FROM projects ORDER BY last_activity_timestamp DESC NULLS LAST, created_timestamp DESC LIMIT ?`
		args = []interface{}{limit}
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var projectData string
		if err := rows.Scan(&projectData); err != nil {
			return nil, err
		}

		var project models.Project
		if err := json.Unmarshal([]byte(projectData), &project); err != nil {
			return nil, err
		}
		projects = append(projects, &project)
	}

	return projects, rows.Err()
}

// Update updates a project
func (r *ProjectRepository) Update(project *models.Project) error {
	now := float64(time.Now().UnixMilli()) / 1000.0
	project.LastActivityTimestamp = &now

	reposJSON, err := json.Marshal(project.Repos)
	if err != nil {
		return err
	}

	projectData, err := json.Marshal(project)
	if err != nil {
		return err
	}

	query := `
		UPDATE projects SET
			name = ?,
			description = ?,
			repos = ?,
			last_activity_timestamp = ?,
			status = ?,
			metadata = ?,
			total_sessions = ?,
			total_goals = ?,
			project_data = ?
		WHERE id = ?
	`
	_, err = r.db.Exec(query,
		project.Name,
		project.Description,
		string(reposJSON),
		project.LastActivityTimestamp,
		project.Status,
		project.Metadata,
		project.TotalSessions,
		project.TotalGoals,
		string(projectData),
		project.ID,
	)
	return err
}

// UpdateStatus updates a project's status
func (r *ProjectRepository) UpdateStatus(projectID string, status models.ProjectStatus) error {
	query := `UPDATE projects SET status = ? WHERE id = ?`
	_, err := r.db.Exec(query, status, projectID)
	return err
}

// IncrementSessions increments the session count for a project
func (r *ProjectRepository) IncrementSessions(projectID string) error {
	now := float64(time.Now().UnixMilli()) / 1000.0
	query := `UPDATE projects SET total_sessions = total_sessions + 1, last_activity_timestamp = ? WHERE id = ?`
	_, err := r.db.Exec(query, now, projectID)
	return err
}

// IncrementGoals increments the goal count for a project
func (r *ProjectRepository) IncrementGoals(projectID string) error {
	now := float64(time.Now().UnixMilli()) / 1000.0
	query := `UPDATE projects SET total_goals = total_goals + 1, last_activity_timestamp = ? WHERE id = ?`
	_, err := r.db.Exec(query, now, projectID)
	return err
}

// ReferenceDocRepository handles reference document database operations
type ReferenceDocRepository struct {
	db *DB
}

// NewReferenceDocRepository creates a new reference doc repository
func NewReferenceDocRepository(db *DB) *ReferenceDocRepository {
	return &ReferenceDocRepository{db: db}
}

// BranchRepository handles investigation branch database operations
type BranchRepository struct {
	db *DB
}

// NewBranchRepository creates a new branch repository
func NewBranchRepository(db *DB) *BranchRepository {
	return &BranchRepository{db: db}
}

// Create creates a new investigation branch
func (r *BranchRepository) Create(branch *models.InvestigationBranch) error {
	query := `
		INSERT INTO investigation_branches (
			id, session_id, branch_name, investigation_path, git_branch_name,
			preflight_vectors, tokens_spent, time_spent_minutes, is_winner,
			created_timestamp, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(query,
		branch.ID,
		branch.SessionID,
		branch.BranchName,
		branch.InvestigationPath,
		branch.GitBranchName,
		branch.PreflightVectors,
		branch.TokensSpent,
		branch.TimeSpentMinutes,
		branch.IsWinner,
		branch.CreatedTimestamp,
		branch.Status,
	)
	return err
}

// Get retrieves a branch by ID
func (r *BranchRepository) Get(branchID string) (*models.InvestigationBranch, error) {
	var branch models.InvestigationBranch
	query := `SELECT * FROM investigation_branches WHERE id = ?`
	err := r.db.Get(&branch, query, branchID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &branch, nil
}

// ListBySession lists branches for a session
func (r *BranchRepository) ListBySession(sessionID string) ([]*models.InvestigationBranch, error) {
	var branches []*models.InvestigationBranch
	query := `SELECT * FROM investigation_branches WHERE session_id = ? ORDER BY created_timestamp DESC`
	err := r.db.Select(&branches, query, sessionID)
	if err != nil {
		return nil, err
	}
	return branches, nil
}

// Checkpoint updates a branch with postflight data
func (r *BranchRepository) Checkpoint(branchID string, postflightVectors string, tokensSpent, timeSpent int) error {
	now := float64(time.Now().UnixMilli()) / 1000.0
	query := `
		UPDATE investigation_branches SET
			postflight_vectors = ?,
			tokens_spent = ?,
			time_spent_minutes = ?,
			checkpoint_timestamp = ?
		WHERE id = ?
	`
	_, err := r.db.Exec(query, postflightVectors, tokensSpent, timeSpent, now, branchID)
	return err
}

// MarkWinner marks a branch as the winner
func (r *BranchRepository) MarkWinner(branchID string, score float64) error {
	now := float64(time.Now().UnixMilli()) / 1000.0
	query := `
		UPDATE investigation_branches SET
			is_winner = 1,
			merge_score = ?,
			merged_timestamp = ?,
			status = 'merged'
		WHERE id = ?
	`
	_, err := r.db.Exec(query, score, now, branchID)
	return err
}
