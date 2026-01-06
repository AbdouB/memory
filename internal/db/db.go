// Package db provides database access for Memory
package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the database connection
type DB struct {
	*sqlx.DB
	path string
}

// DefaultDBPath returns the default database path
func DefaultDBPath() string {
	// Try project-local first
	localPath := ".memory/sessions.db"
	if _, err := os.Stat(".memory"); err == nil {
		return localPath
	}

	// Fall back to home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return localPath
	}
	return filepath.Join(home, ".memory", "sessions.db")
}

// Open opens or creates the database
func Open(path string) (*DB, error) {
	if path == "" {
		path = DefaultDBPath()
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database
	db, err := sqlx.Open("sqlite3", path+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	d := &DB{DB: db, path: path}

	// Run migrations
	if err := d.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return d, nil
}

// Path returns the database file path
func (d *DB) Path() string {
	return d.path
}

// migrate runs database migrations
func (d *DB) migrate() error {
	migrations := []string{
		migrationSessions,
		migrationCascades,
		migrationReflexes,
		migrationGoals,
		migrationSubtasks,
		migrationProjects,
		migrationFindings,
		migrationUnknowns,
		migrationDeadEnds,
		migrationMistakes,
		migrationHandoffs,
		migrationBranches,
		migrationIndexes,
	}

	for _, m := range migrations {
		if _, err := d.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	// Run ALTER TABLE migrations (ignore errors - columns may already exist)
	alterMigrations := []string{
		migrationFindingStaleness,
		migrationFindingStaleness2,
		migrationHandoffProjectID,
	}
	for _, m := range alterMigrations {
		d.Exec(m) // Ignore errors - column may already exist
	}

	return nil
}

const migrationSessions = `
CREATE TABLE IF NOT EXISTS sessions (
    session_id TEXT PRIMARY KEY,
    ai_id TEXT NOT NULL,
    user_id TEXT,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    components_loaded INTEGER NOT NULL DEFAULT 0,
    total_turns INTEGER DEFAULT 0,
    total_cascades INTEGER DEFAULT 0,
    avg_confidence REAL,
    drift_detected BOOLEAN DEFAULT 0,
    session_notes TEXT,
    bootstrap_level INTEGER DEFAULT 1,
    project_id TEXT,
    subject TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`

const migrationCascades = `
CREATE TABLE IF NOT EXISTS cascades (
    cascade_id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    task TEXT NOT NULL,
    context_json TEXT,
    goal_id TEXT,
    goal_json TEXT,
    preflight_completed BOOLEAN DEFAULT 0,
    think_completed BOOLEAN DEFAULT 0,
    plan_completed BOOLEAN DEFAULT 0,
    investigate_completed BOOLEAN DEFAULT 0,
    check_completed BOOLEAN DEFAULT 0,
    act_completed BOOLEAN DEFAULT 0,
    postflight_completed BOOLEAN DEFAULT 0,
    final_action TEXT,
    final_confidence REAL,
    investigation_rounds INTEGER DEFAULT 0,
    duration_ms INTEGER,
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    engagement_gate_passed BOOLEAN,
    bayesian_active BOOLEAN DEFAULT 0,
    drift_monitored BOOLEAN DEFAULT 0,
    FOREIGN KEY (session_id) REFERENCES sessions(session_id)
);
`

const migrationReflexes = `
CREATE TABLE IF NOT EXISTS reflexes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    cascade_id TEXT,
    phase TEXT NOT NULL,
    round INTEGER DEFAULT 1,
    timestamp REAL NOT NULL,
    engagement REAL,
    know REAL,
    do_vec REAL,
    context REAL,
    clarity REAL,
    coherence REAL,
    signal REAL,
    density REAL,
    state REAL,
    change REAL,
    completion REAL,
    impact REAL,
    uncertainty REAL,
    reflex_data TEXT,
    reasoning TEXT,
    evidence TEXT,
    FOREIGN KEY (session_id) REFERENCES sessions(session_id)
);
`

const migrationGoals = `
CREATE TABLE IF NOT EXISTS goals (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    objective TEXT NOT NULL,
    scope TEXT NOT NULL,
    estimated_complexity REAL,
    created_timestamp REAL NOT NULL,
    completed_timestamp REAL,
    is_completed BOOLEAN DEFAULT 0,
    goal_data TEXT NOT NULL,
    status TEXT DEFAULT 'in_progress',
    beads_issue_id TEXT,
    FOREIGN KEY (session_id) REFERENCES sessions(session_id)
);
`

const migrationSubtasks = `
CREATE TABLE IF NOT EXISTS subtasks (
    id TEXT PRIMARY KEY,
    goal_id TEXT NOT NULL,
    description TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    epistemic_importance TEXT NOT NULL DEFAULT 'medium',
    estimated_tokens INTEGER,
    actual_tokens INTEGER,
    completion_evidence TEXT,
    notes TEXT,
    created_timestamp REAL NOT NULL,
    completed_timestamp REAL,
    subtask_data TEXT NOT NULL,
    FOREIGN KEY (goal_id) REFERENCES goals(id)
);
`

const migrationProjects = `
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    repos TEXT,
    created_timestamp REAL NOT NULL,
    last_activity_timestamp REAL,
    status TEXT DEFAULT 'active',
    metadata TEXT,
    total_sessions INTEGER DEFAULT 0,
    total_goals INTEGER DEFAULT 0,
    total_epistemic_deltas TEXT,
    project_data TEXT NOT NULL
);
`

const migrationFindings = `
CREATE TABLE IF NOT EXISTS project_findings (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    goal_id TEXT,
    subtask_id TEXT,
    finding TEXT NOT NULL,
    created_timestamp REAL NOT NULL,
    finding_data TEXT NOT NULL,
    subject TEXT,
    impact REAL DEFAULT 0.5,
    FOREIGN KEY (project_id) REFERENCES projects(id)
);
`

const migrationUnknowns = `
CREATE TABLE IF NOT EXISTS project_unknowns (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    goal_id TEXT,
    subtask_id TEXT,
    unknown TEXT NOT NULL,
    is_resolved BOOLEAN DEFAULT FALSE,
    resolved_by TEXT,
    created_timestamp REAL NOT NULL,
    resolved_timestamp REAL,
    unknown_data TEXT NOT NULL,
    subject TEXT,
    impact REAL DEFAULT 0.5,
    FOREIGN KEY (project_id) REFERENCES projects(id)
);
`

const migrationDeadEnds = `
CREATE TABLE IF NOT EXISTS project_dead_ends (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    goal_id TEXT,
    subtask_id TEXT,
    approach TEXT NOT NULL,
    why_failed TEXT NOT NULL,
    created_timestamp REAL NOT NULL,
    dead_end_data TEXT NOT NULL,
    subject TEXT,
    impact REAL DEFAULT 0.5,
    FOREIGN KEY (project_id) REFERENCES projects(id)
);
`

const migrationMistakes = `
CREATE TABLE IF NOT EXISTS mistakes_made (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    goal_id TEXT,
    project_id TEXT,
    mistake TEXT NOT NULL,
    why_wrong TEXT NOT NULL,
    cost_estimate TEXT,
    root_cause_vector TEXT,
    prevention TEXT,
    created_timestamp REAL NOT NULL,
    mistake_data TEXT NOT NULL,
    FOREIGN KEY (session_id) REFERENCES sessions(session_id)
);
`

const migrationHandoffs = `
CREATE TABLE IF NOT EXISTS handoff_reports (
    session_id TEXT PRIMARY KEY,
    ai_id TEXT NOT NULL,
    timestamp TEXT NOT NULL,
    task_summary TEXT,
    duration_seconds REAL,
    epistemic_deltas TEXT,
    key_findings TEXT,
    knowledge_gaps_filled TEXT,
    remaining_unknowns TEXT,
    noetic_tools TEXT,
    next_session_context TEXT,
    recommended_next_steps TEXT,
    artifacts_created TEXT,
    calibration_status TEXT,
    overall_confidence_delta REAL,
    compressed_json TEXT,
    markdown_report TEXT,
    created_at REAL NOT NULL,
    FOREIGN KEY (session_id) REFERENCES sessions(session_id)
);
`

const migrationBranches = `
CREATE TABLE IF NOT EXISTS investigation_branches (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    branch_name TEXT NOT NULL,
    investigation_path TEXT NOT NULL,
    git_branch_name TEXT NOT NULL,
    preflight_vectors TEXT NOT NULL,
    postflight_vectors TEXT,
    tokens_spent INTEGER DEFAULT 0,
    time_spent_minutes INTEGER DEFAULT 0,
    merge_score REAL,
    epistemic_quality REAL,
    is_winner BOOLEAN DEFAULT FALSE,
    created_timestamp REAL NOT NULL,
    checkpoint_timestamp REAL,
    merged_timestamp REAL,
    status TEXT DEFAULT 'active',
    branch_metadata TEXT,
    FOREIGN KEY (session_id) REFERENCES sessions(session_id)
);

CREATE TABLE IF NOT EXISTS merge_decisions (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    investigation_round INTEGER NOT NULL,
    winning_branch_id TEXT NOT NULL,
    winning_branch_name TEXT,
    winning_score REAL NOT NULL,
    other_branches TEXT,
    decision_rationale TEXT NOT NULL,
    auto_merged BOOLEAN DEFAULT TRUE,
    created_timestamp REAL NOT NULL,
    decision_metadata TEXT,
    FOREIGN KEY (session_id) REFERENCES sessions(session_id)
);
`

const migrationIndexes = `
CREATE INDEX IF NOT EXISTS idx_sessions_ai_id ON sessions(ai_id);
CREATE INDEX IF NOT EXISTS idx_sessions_project_id ON sessions(project_id);
CREATE INDEX IF NOT EXISTS idx_cascades_session_id ON cascades(session_id);
CREATE INDEX IF NOT EXISTS idx_reflexes_session_id ON reflexes(session_id);
CREATE INDEX IF NOT EXISTS idx_reflexes_phase ON reflexes(phase);
CREATE INDEX IF NOT EXISTS idx_goals_session_id ON goals(session_id);
CREATE INDEX IF NOT EXISTS idx_subtasks_goal_id ON subtasks(goal_id);
CREATE INDEX IF NOT EXISTS idx_findings_project_id ON project_findings(project_id);
CREATE INDEX IF NOT EXISTS idx_findings_session_id ON project_findings(session_id);
CREATE INDEX IF NOT EXISTS idx_unknowns_project_id ON project_unknowns(project_id);
CREATE INDEX IF NOT EXISTS idx_unknowns_resolved ON project_unknowns(is_resolved);
CREATE INDEX IF NOT EXISTS idx_dead_ends_project_id ON project_dead_ends(project_id);
CREATE INDEX IF NOT EXISTS idx_mistakes_session_id ON mistakes_made(session_id);
CREATE INDEX IF NOT EXISTS idx_branches_session_id ON investigation_branches(session_id);
`

// migrationFindingStaleness adds staleness tracking columns to findings
// Uses ALTER TABLE which will fail silently if columns already exist
const migrationFindingStaleness = `
ALTER TABLE project_findings ADD COLUMN last_verified_timestamp REAL;
`

const migrationFindingStaleness2 = `
ALTER TABLE project_findings ADD COLUMN subject_git_hash TEXT;
`

// migrationHandoffProjectID adds project_id to handoff_reports for project isolation
const migrationHandoffProjectID = `
ALTER TABLE handoff_reports ADD COLUMN project_id TEXT;
`
