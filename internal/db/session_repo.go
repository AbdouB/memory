package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/AbdouB/memory/internal/models"
)

// SessionRepository handles session database operations
type SessionRepository struct {
	db *DB
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create creates a new session
func (r *SessionRepository) Create(session *models.Session) error {
	query := `
		INSERT INTO sessions (
			session_id, ai_id, user_id, start_time, components_loaded,
			total_turns, total_cascades, drift_detected, bootstrap_level,
			project_id, subject, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(query,
		session.SessionID,
		session.AIID,
		session.UserID,
		session.StartTime,
		session.ComponentsLoaded,
		session.TotalTurns,
		session.TotalCascades,
		session.DriftDetected,
		session.BootstrapLevel,
		session.ProjectID,
		session.Subject,
		session.CreatedAt,
	)
	return err
}

// Get retrieves a session by ID
func (r *SessionRepository) Get(sessionID string) (*models.Session, error) {
	var session models.Session
	query := `SELECT * FROM sessions WHERE session_id = ?`
	err := r.db.Get(&session, query, sessionID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// List lists sessions with optional filtering
func (r *SessionRepository) List(aiID string, limit int) ([]*models.Session, error) {
	var sessions []*models.Session
	var query string
	var args []interface{}

	if aiID != "" {
		query = `SELECT * FROM sessions WHERE ai_id = ? ORDER BY created_at DESC LIMIT ?`
		args = []interface{}{aiID, limit}
	} else {
		query = `SELECT * FROM sessions ORDER BY created_at DESC LIMIT ?`
		args = []interface{}{limit}
	}

	err := r.db.Select(&sessions, query, args...)
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

// GetLatest gets the most recent session for an AI
func (r *SessionRepository) GetLatest(aiID string) (*models.Session, error) {
	var session models.Session
	query := `SELECT * FROM sessions WHERE ai_id = ? ORDER BY created_at DESC LIMIT 1`
	err := r.db.Get(&session, query, aiID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// Update updates a session
func (r *SessionRepository) Update(session *models.Session) error {
	query := `
		UPDATE sessions SET
			end_time = ?,
			total_turns = ?,
			total_cascades = ?,
			avg_confidence = ?,
			drift_detected = ?,
			session_notes = ?,
			bootstrap_level = ?
		WHERE session_id = ?
	`
	_, err := r.db.Exec(query,
		session.EndTime,
		session.TotalTurns,
		session.TotalCascades,
		session.AvgConfidence,
		session.DriftDetected,
		session.SessionNotes,
		session.BootstrapLevel,
		session.SessionID,
	)
	return err
}

// End marks a session as ended
func (r *SessionRepository) End(sessionID string) error {
	now := time.Now()
	query := `UPDATE sessions SET end_time = ? WHERE session_id = ?`
	_, err := r.db.Exec(query, now, sessionID)
	return err
}

// ReflexRepository handles reflex (epistemic checkpoint) database operations
type ReflexRepository struct {
	db *DB
}

// NewReflexRepository creates a new reflex repository
func NewReflexRepository(db *DB) *ReflexRepository {
	return &ReflexRepository{db: db}
}

// Create creates a new reflex
func (r *ReflexRepository) Create(reflex *models.Reflex) error {
	query := `
		INSERT INTO reflexes (
			session_id, cascade_id, phase, round, timestamp,
			engagement, know, do_vec, context, clarity, coherence,
			signal, density, state, change, completion, impact, uncertainty,
			reflex_data, reasoning, evidence
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.Exec(query,
		reflex.SessionID,
		reflex.CascadeID,
		reflex.Phase,
		reflex.Round,
		reflex.Timestamp,
		reflex.Engagement,
		reflex.Know,
		reflex.Do,
		reflex.Context,
		reflex.Clarity,
		reflex.Coherence,
		reflex.Signal,
		reflex.Density,
		reflex.State,
		reflex.Change,
		reflex.Completion,
		reflex.Impact,
		reflex.Uncertainty,
		reflex.ReflexData,
		reflex.Reasoning,
		reflex.Evidence,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	reflex.ID = id
	return nil
}

// GetLatestByPhase gets the most recent reflex for a session and phase
func (r *ReflexRepository) GetLatestByPhase(sessionID, phase string) (*models.Reflex, error) {
	var reflex models.Reflex
	query := `
		SELECT * FROM reflexes 
		WHERE session_id = ? AND phase = ? 
		ORDER BY timestamp DESC LIMIT 1
	`
	err := r.db.Get(&reflex, query, sessionID, phase)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &reflex, nil
}

// ListBySession lists all reflexes for a session
func (r *ReflexRepository) ListBySession(sessionID string, limit int) ([]*models.Reflex, error) {
	var reflexes []*models.Reflex
	query := `SELECT * FROM reflexes WHERE session_id = ? ORDER BY timestamp DESC LIMIT ?`
	err := r.db.Select(&reflexes, query, sessionID, limit)
	if err != nil {
		return nil, err
	}
	return reflexes, nil
}

// GetDelta calculates the epistemic delta between two reflexes
func (r *ReflexRepository) GetDelta(sessionID string) (*models.EpistemicVectors, error) {
	preflight, err := r.GetLatestByPhase(sessionID, "PREFLIGHT")
	if err != nil || preflight == nil {
		return nil, err
	}

	postflight, err := r.GetLatestByPhase(sessionID, "POSTFLIGHT")
	if err != nil || postflight == nil {
		return nil, err
	}

	preVectors := preflight.ToVectors()
	postVectors := postflight.ToVectors()

	return postVectors.Delta(preVectors), nil
}

// CascadeRepository handles cascade database operations
type CascadeRepository struct {
	db *DB
}

// NewCascadeRepository creates a new cascade repository
func NewCascadeRepository(db *DB) *CascadeRepository {
	return &CascadeRepository{db: db}
}

// Create creates a new cascade
func (r *CascadeRepository) Create(cascade *models.Cascade) error {
	query := `
		INSERT INTO cascades (
			cascade_id, session_id, task, context_json, goal_id, goal_json,
			preflight_completed, think_completed, plan_completed,
			investigate_completed, check_completed, act_completed, postflight_completed,
			investigation_rounds, started_at, bayesian_active, drift_monitored
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(query,
		cascade.CascadeID,
		cascade.SessionID,
		cascade.Task,
		cascade.ContextJSON,
		cascade.GoalID,
		cascade.GoalJSON,
		cascade.PreflightCompleted,
		cascade.ThinkCompleted,
		cascade.PlanCompleted,
		cascade.InvestigateCompleted,
		cascade.CheckCompleted,
		cascade.ActCompleted,
		cascade.PostflightCompleted,
		cascade.InvestigationRounds,
		cascade.StartedAt,
		cascade.BayesianActive,
		cascade.DriftMonitored,
	)
	return err
}

// Get retrieves a cascade by ID
func (r *CascadeRepository) Get(cascadeID string) (*models.Cascade, error) {
	var cascade models.Cascade
	query := `SELECT * FROM cascades WHERE cascade_id = ?`
	err := r.db.Get(&cascade, query, cascadeID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cascade, nil
}

// UpdatePhase updates a cascade phase completion status
func (r *CascadeRepository) UpdatePhase(cascadeID, phase string, completed bool) error {
	var column string
	switch phase {
	case "PREFLIGHT":
		column = "preflight_completed"
	case "THINK":
		column = "think_completed"
	case "PLAN":
		column = "plan_completed"
	case "INVESTIGATE":
		column = "investigate_completed"
	case "CHECK":
		column = "check_completed"
	case "ACT":
		column = "act_completed"
	case "POSTFLIGHT":
		column = "postflight_completed"
	default:
		return fmt.Errorf("unknown phase: %s", phase)
	}

	query := fmt.Sprintf("UPDATE cascades SET %s = ? WHERE cascade_id = ?", column)
	_, err := r.db.Exec(query, completed, cascadeID)
	return err
}

// Complete marks a cascade as completed
func (r *CascadeRepository) Complete(cascadeID string, action string, confidence float64) error {
	now := time.Now()
	query := `
		UPDATE cascades SET 
			completed_at = ?,
			final_action = ?,
			final_confidence = ?
		WHERE cascade_id = ?
	`
	_, err := r.db.Exec(query, now, action, confidence, cascadeID)
	return err
}

// HandoffRepository handles handoff report database operations
type HandoffRepository struct {
	db *DB
}

// NewHandoffRepository creates a new handoff repository
func NewHandoffRepository(db *DB) *HandoffRepository {
	return &HandoffRepository{db: db}
}

// Create creates a new handoff report
func (r *HandoffRepository) Create(input *models.HandoffCreateInput, aiID string) (*models.HandoffReport, error) {
	now := time.Now()

	keyFindingsJSON, _ := json.Marshal(input.KeyFindings)
	unknownsJSON, _ := json.Marshal(input.RemainingUnknowns)
	artifactsJSON, _ := json.Marshal(input.Artifacts)

	var projectID *string
	if input.ProjectID != "" {
		projectID = &input.ProjectID
	}

	report := &models.HandoffReport{
		SessionID:          input.SessionID,
		AIID:               aiID,
		ProjectID:          projectID,
		Timestamp:          now.Format(time.RFC3339),
		TaskSummary:        &input.TaskSummary,
		KeyFindings:        strPtr(string(keyFindingsJSON)),
		RemainingUnknowns:  strPtr(string(unknownsJSON)),
		NextSessionContext: strPtr(input.NextSessionContext),
		ArtifactsCreated:   strPtr(string(artifactsJSON)),
		CreatedAt:          float64(now.UnixMilli()) / 1000.0,
	}

	query := `
		INSERT INTO handoff_reports (
			session_id, ai_id, project_id, timestamp, task_summary,
			key_findings, remaining_unknowns, next_session_context,
			artifacts_created, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(query,
		report.SessionID,
		report.AIID,
		report.ProjectID,
		report.Timestamp,
		report.TaskSummary,
		report.KeyFindings,
		report.RemainingUnknowns,
		report.NextSessionContext,
		report.ArtifactsCreated,
		report.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return report, nil
}

// Get retrieves a handoff report by session ID
func (r *HandoffRepository) Get(sessionID string) (*models.HandoffReport, error) {
	var report models.HandoffReport
	query := `SELECT * FROM handoff_reports WHERE session_id = ?`
	err := r.db.Get(&report, query, sessionID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &report, nil
}

// List lists handoff reports filtered by project and/or AI ID
func (r *HandoffRepository) List(projectID, aiID string, limit int) ([]*models.HandoffReport, error) {
	var reports []*models.HandoffReport
	var query string
	var args []interface{}

	if projectID != "" && aiID != "" {
		query = `SELECT * FROM handoff_reports WHERE project_id = ? AND ai_id = ? ORDER BY created_at DESC LIMIT ?`
		args = []interface{}{projectID, aiID, limit}
	} else if projectID != "" {
		query = `SELECT * FROM handoff_reports WHERE project_id = ? ORDER BY created_at DESC LIMIT ?`
		args = []interface{}{projectID, limit}
	} else if aiID != "" {
		query = `SELECT * FROM handoff_reports WHERE ai_id = ? ORDER BY created_at DESC LIMIT ?`
		args = []interface{}{aiID, limit}
	} else {
		query = `SELECT * FROM handoff_reports ORDER BY created_at DESC LIMIT ?`
		args = []interface{}{limit}
	}

	err := r.db.Select(&reports, query, args...)
	if err != nil {
		return nil, err
	}
	return reports, nil
}

func strPtr(s string) *string {
	return &s
}
