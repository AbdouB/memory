package db

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/AbdouB/memory/internal/models"
)

// BreadcrumbRepository handles breadcrumb (findings, unknowns, dead ends) database operations
type BreadcrumbRepository struct {
	db *DB
}

// NewBreadcrumbRepository creates a new breadcrumb repository
func NewBreadcrumbRepository(db *DB) *BreadcrumbRepository {
	return &BreadcrumbRepository{db: db}
}

// CreateFinding creates a new finding
func (r *BreadcrumbRepository) CreateFinding(finding *models.Finding) error {
	findingData, err := json.Marshal(finding)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO project_findings (
			id, project_id, session_id, goal_id, subtask_id,
			finding, created_timestamp, finding_data, subject, impact,
			last_verified_timestamp, subject_git_hash
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = r.db.Exec(query,
		finding.ID,
		finding.ProjectID,
		finding.SessionID,
		finding.GoalID,
		finding.SubtaskID,
		finding.Finding,
		finding.CreatedTimestamp,
		string(findingData),
		finding.Subject,
		finding.Impact,
		finding.LastVerifiedTimestamp,
		finding.SubjectGitHash,
	)
	return err
}

// GetFinding retrieves a finding by ID
func (r *BreadcrumbRepository) GetFinding(findingID string) (*models.Finding, error) {
	var findingData string
	query := `SELECT finding_data FROM project_findings WHERE id = ?`
	err := r.db.QueryRow(query, findingID).Scan(&findingData)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var finding models.Finding
	if err := json.Unmarshal([]byte(findingData), &finding); err != nil {
		return nil, err
	}
	return &finding, nil
}

// ListFindingsWithStaleness lists findings with their staleness metadata loaded from db columns
func (r *BreadcrumbRepository) ListFindingsWithStaleness(projectID, sessionID string, limit int) ([]*models.Finding, error) {
	var findings []*models.Finding
	var query string
	var args []interface{}

	// Select individual columns including staleness fields
	selectCols := `id, project_id, session_id, goal_id, subtask_id, finding,
		created_timestamp, subject, impact, last_verified_timestamp, subject_git_hash`

	if projectID != "" && sessionID != "" {
		query = `SELECT ` + selectCols + ` FROM project_findings WHERE project_id = ? AND session_id = ? ORDER BY created_timestamp DESC LIMIT ?`
		args = []interface{}{projectID, sessionID, limit}
	} else if projectID != "" {
		query = `SELECT ` + selectCols + ` FROM project_findings WHERE project_id = ? ORDER BY created_timestamp DESC LIMIT ?`
		args = []interface{}{projectID, limit}
	} else if sessionID != "" {
		query = `SELECT ` + selectCols + ` FROM project_findings WHERE session_id = ? ORDER BY created_timestamp DESC LIMIT ?`
		args = []interface{}{sessionID, limit}
	} else {
		query = `SELECT ` + selectCols + ` FROM project_findings ORDER BY created_timestamp DESC LIMIT ?`
		args = []interface{}{limit}
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var f models.Finding
		if err := rows.Scan(
			&f.ID,
			&f.ProjectID,
			&f.SessionID,
			&f.GoalID,
			&f.SubtaskID,
			&f.Finding,
			&f.CreatedTimestamp,
			&f.Subject,
			&f.Impact,
			&f.LastVerifiedTimestamp,
			&f.SubjectGitHash,
		); err != nil {
			return nil, err
		}
		findings = append(findings, &f)
	}

	return findings, rows.Err()
}

// VerifyFinding refreshes the verification timestamp and optionally updates the text and git hash
func (r *BreadcrumbRepository) VerifyFinding(findingID string, newGitHash, updatedText *string) error {
	now := float64(time.Now().UnixMilli()) / 1000.0

	// Build update query based on what needs updating
	query := `UPDATE project_findings SET last_verified_timestamp = ?`
	args := []interface{}{now}

	if newGitHash != nil {
		query += `, subject_git_hash = ?`
		args = append(args, *newGitHash)
	}
	if updatedText != nil {
		query += `, finding = ?`
		args = append(args, *updatedText)
	}

	query += ` WHERE id = ?`
	args = append(args, findingID)

	result, err := r.db.Exec(query, args...)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// FindFindingByText searches for findings containing the given text
func (r *BreadcrumbRepository) FindFindingByText(projectID, searchText string) ([]*models.Finding, error) {
	var findings []*models.Finding

	selectCols := `id, project_id, session_id, goal_id, subtask_id, finding,
		created_timestamp, subject, impact, last_verified_timestamp, subject_git_hash`

	query := `SELECT ` + selectCols + ` FROM project_findings WHERE finding LIKE ?`
	args := []interface{}{"%" + searchText + "%"}

	if projectID != "" {
		query += ` AND project_id = ?`
		args = append(args, projectID)
	}

	query += ` ORDER BY created_timestamp DESC LIMIT 10`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var f models.Finding
		if err := rows.Scan(
			&f.ID,
			&f.ProjectID,
			&f.SessionID,
			&f.GoalID,
			&f.SubtaskID,
			&f.Finding,
			&f.CreatedTimestamp,
			&f.Subject,
			&f.Impact,
			&f.LastVerifiedTimestamp,
			&f.SubjectGitHash,
		); err != nil {
			return nil, err
		}
		findings = append(findings, &f)
	}

	return findings, rows.Err()
}

// ListFindings lists findings with filtering
func (r *BreadcrumbRepository) ListFindings(projectID, sessionID string, limit int) ([]*models.Finding, error) {
	var findings []*models.Finding
	var query string
	var args []interface{}

	if projectID != "" && sessionID != "" {
		query = `SELECT finding_data FROM project_findings WHERE project_id = ? AND session_id = ? ORDER BY created_timestamp DESC LIMIT ?`
		args = []interface{}{projectID, sessionID, limit}
	} else if projectID != "" {
		query = `SELECT finding_data FROM project_findings WHERE project_id = ? ORDER BY created_timestamp DESC LIMIT ?`
		args = []interface{}{projectID, limit}
	} else if sessionID != "" {
		query = `SELECT finding_data FROM project_findings WHERE session_id = ? ORDER BY created_timestamp DESC LIMIT ?`
		args = []interface{}{sessionID, limit}
	} else {
		query = `SELECT finding_data FROM project_findings ORDER BY created_timestamp DESC LIMIT ?`
		args = []interface{}{limit}
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var findingData string
		if err := rows.Scan(&findingData); err != nil {
			return nil, err
		}

		var finding models.Finding
		if err := json.Unmarshal([]byte(findingData), &finding); err != nil {
			return nil, err
		}
		findings = append(findings, &finding)
	}

	return findings, rows.Err()
}

// CreateUnknown creates a new unknown
func (r *BreadcrumbRepository) CreateUnknown(unknown *models.Unknown) error {
	unknownData, err := json.Marshal(unknown)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO project_unknowns (
			id, project_id, session_id, goal_id, subtask_id,
			unknown, is_resolved, created_timestamp, unknown_data, subject, impact
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = r.db.Exec(query,
		unknown.ID,
		unknown.ProjectID,
		unknown.SessionID,
		unknown.GoalID,
		unknown.SubtaskID,
		unknown.Unknown,
		unknown.IsResolved,
		unknown.CreatedTimestamp,
		string(unknownData),
		unknown.Subject,
		unknown.Impact,
	)
	return err
}

// GetUnknown retrieves an unknown by ID
func (r *BreadcrumbRepository) GetUnknown(unknownID string) (*models.Unknown, error) {
	var unknownData string
	query := `SELECT unknown_data FROM project_unknowns WHERE id = ?`
	err := r.db.QueryRow(query, unknownID).Scan(&unknownData)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var unknown models.Unknown
	if err := json.Unmarshal([]byte(unknownData), &unknown); err != nil {
		return nil, err
	}
	return &unknown, nil
}

// ListUnknowns lists unknowns with filtering
func (r *BreadcrumbRepository) ListUnknowns(projectID, sessionID string, resolved *bool, limit int) ([]*models.Unknown, error) {
	var unknowns []*models.Unknown
	var query string
	var args []interface{}

	baseQuery := `SELECT unknown_data FROM project_unknowns WHERE 1=1`

	if projectID != "" {
		baseQuery += ` AND project_id = ?`
		args = append(args, projectID)
	}
	if sessionID != "" {
		baseQuery += ` AND session_id = ?`
		args = append(args, sessionID)
	}
	if resolved != nil {
		baseQuery += ` AND is_resolved = ?`
		args = append(args, *resolved)
	}

	query = baseQuery + ` ORDER BY created_timestamp DESC LIMIT ?`
	args = append(args, limit)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var unknownData string
		if err := rows.Scan(&unknownData); err != nil {
			return nil, err
		}

		var unknown models.Unknown
		if err := json.Unmarshal([]byte(unknownData), &unknown); err != nil {
			return nil, err
		}
		unknowns = append(unknowns, &unknown)
	}

	return unknowns, rows.Err()
}

// ResolveUnknown marks an unknown as resolved
func (r *BreadcrumbRepository) ResolveUnknown(unknownID, resolvedBy string) error {
	now := float64(time.Now().UnixMilli()) / 1000.0

	// Get current unknown
	unknown, err := r.GetUnknown(unknownID)
	if err != nil {
		return err
	}
	if unknown == nil {
		return sql.ErrNoRows
	}

	unknown.IsResolved = true
	unknown.ResolvedBy = &resolvedBy
	unknown.ResolvedTimestamp = &now

	unknownData, err := json.Marshal(unknown)
	if err != nil {
		return err
	}

	query := `
		UPDATE project_unknowns SET 
			is_resolved = 1,
			resolved_by = ?,
			resolved_timestamp = ?,
			unknown_data = ?
		WHERE id = ?
	`
	_, err = r.db.Exec(query, resolvedBy, now, string(unknownData), unknownID)
	return err
}

// CreateDeadEnd creates a new dead end
func (r *BreadcrumbRepository) CreateDeadEnd(deadEnd *models.DeadEnd) error {
	deadEndData, err := json.Marshal(deadEnd)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO project_dead_ends (
			id, project_id, session_id, goal_id, subtask_id,
			approach, why_failed, created_timestamp, dead_end_data, subject, impact
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = r.db.Exec(query,
		deadEnd.ID,
		deadEnd.ProjectID,
		deadEnd.SessionID,
		deadEnd.GoalID,
		deadEnd.SubtaskID,
		deadEnd.Approach,
		deadEnd.WhyFailed,
		deadEnd.CreatedTimestamp,
		string(deadEndData),
		deadEnd.Subject,
		deadEnd.Impact,
	)
	return err
}

// ListDeadEnds lists dead ends with filtering
func (r *BreadcrumbRepository) ListDeadEnds(projectID, sessionID string, limit int) ([]*models.DeadEnd, error) {
	var deadEnds []*models.DeadEnd
	var query string
	var args []interface{}

	if projectID != "" && sessionID != "" {
		query = `SELECT dead_end_data FROM project_dead_ends WHERE project_id = ? AND session_id = ? ORDER BY created_timestamp DESC LIMIT ?`
		args = []interface{}{projectID, sessionID, limit}
	} else if projectID != "" {
		query = `SELECT dead_end_data FROM project_dead_ends WHERE project_id = ? ORDER BY created_timestamp DESC LIMIT ?`
		args = []interface{}{projectID, limit}
	} else if sessionID != "" {
		query = `SELECT dead_end_data FROM project_dead_ends WHERE session_id = ? ORDER BY created_timestamp DESC LIMIT ?`
		args = []interface{}{sessionID, limit}
	} else {
		query = `SELECT dead_end_data FROM project_dead_ends ORDER BY created_timestamp DESC LIMIT ?`
		args = []interface{}{limit}
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var deadEndData string
		if err := rows.Scan(&deadEndData); err != nil {
			return nil, err
		}

		var deadEnd models.DeadEnd
		if err := json.Unmarshal([]byte(deadEndData), &deadEnd); err != nil {
			return nil, err
		}
		deadEnds = append(deadEnds, &deadEnd)
	}

	return deadEnds, rows.Err()
}

// MistakeRepository handles mistake database operations
type MistakeRepository struct {
	db *DB
}

// NewMistakeRepository creates a new mistake repository
func NewMistakeRepository(db *DB) *MistakeRepository {
	return &MistakeRepository{db: db}
}

// Create creates a new mistake
func (r *MistakeRepository) Create(mistake *models.Mistake) error {
	mistakeData, err := json.Marshal(mistake)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO mistakes_made (
			id, session_id, goal_id, project_id, mistake, why_wrong,
			cost_estimate, root_cause_vector, prevention, created_timestamp, mistake_data
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = r.db.Exec(query,
		mistake.ID,
		mistake.SessionID,
		mistake.GoalID,
		mistake.ProjectID,
		mistake.Mistake,
		mistake.WhyWrong,
		mistake.CostEstimate,
		mistake.RootCauseVector,
		mistake.Prevention,
		mistake.CreatedTimestamp,
		string(mistakeData),
	)
	return err
}

// List lists mistakes with filtering
func (r *MistakeRepository) List(sessionID string, goalID *string, limit int) ([]*models.Mistake, error) {
	var mistakes []*models.Mistake
	var query string
	var args []interface{}

	if sessionID != "" && goalID != nil {
		query = `SELECT mistake_data FROM mistakes_made WHERE session_id = ? AND goal_id = ? ORDER BY created_timestamp DESC LIMIT ?`
		args = []interface{}{sessionID, *goalID, limit}
	} else if sessionID != "" {
		query = `SELECT mistake_data FROM mistakes_made WHERE session_id = ? ORDER BY created_timestamp DESC LIMIT ?`
		args = []interface{}{sessionID, limit}
	} else {
		query = `SELECT mistake_data FROM mistakes_made ORDER BY created_timestamp DESC LIMIT ?`
		args = []interface{}{limit}
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var mistakeData string
		if err := rows.Scan(&mistakeData); err != nil {
			return nil, err
		}

		var mistake models.Mistake
		if err := json.Unmarshal([]byte(mistakeData), &mistake); err != nil {
			return nil, err
		}
		mistakes = append(mistakes, &mistake)
	}

	return mistakes, rows.Err()
}
