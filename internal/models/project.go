package models

import (
	"time"

	"github.com/google/uuid"
)

// ProjectStatus represents the current state of a project
type ProjectStatus string

const (
	ProjectStatusActive   ProjectStatus = "active"
	ProjectStatusInactive ProjectStatus = "inactive"
	ProjectStatusComplete ProjectStatus = "complete"
)

// Project represents an Empirica project for cross-session tracking
type Project struct {
	ID                    string        `json:"id" db:"id"`
	Name                  string        `json:"name" db:"name"`
	Description           *string       `json:"description,omitempty" db:"description"`
	Repos                 []string      `json:"repos"` // Git repositories
	ReposJSON             string        `json:"-" db:"repos"`
	CreatedTimestamp      float64       `json:"created_timestamp" db:"created_timestamp"`
	LastActivityTimestamp *float64      `json:"last_activity_timestamp,omitempty" db:"last_activity_timestamp"`
	Status                ProjectStatus `json:"status" db:"status"`
	Metadata              *string       `json:"metadata,omitempty" db:"metadata"`
	TotalSessions         int           `json:"total_sessions" db:"total_sessions"`
	TotalGoals            int           `json:"total_goals" db:"total_goals"`
	TotalEpistemicDeltas  *string       `json:"total_epistemic_deltas,omitempty" db:"total_epistemic_deltas"`
	ProjectData           string        `json:"-" db:"project_data"`
}

// NewProject creates a new project
func NewProject(name string, description *string) *Project {
	return &Project{
		ID:               uuid.New().String(),
		Name:             name,
		Description:      description,
		Repos:            []string{},
		CreatedTimestamp: float64(time.Now().UnixMilli()) / 1000.0,
		Status:           ProjectStatusActive,
		TotalSessions:    0,
		TotalGoals:       0,
	}
}

// ProjectCreateInput represents input for creating a project
type ProjectCreateInput struct {
	Name        string   `json:"name"`
	Description *string  `json:"description,omitempty"`
	Repos       []string `json:"repos,omitempty"`
}

// ProjectHandoff represents a project-level handoff for continuity
type ProjectHandoff struct {
	ID                   string  `json:"id" db:"id"`
	ProjectID            string  `json:"project_id" db:"project_id"`
	CreatedTimestamp     float64 `json:"created_timestamp" db:"created_timestamp"`
	ProjectSummary       string  `json:"project_summary" db:"project_summary"`
	SessionsIncluded     string  `json:"sessions_included" db:"sessions_included"` // JSON array
	TotalLearningDeltas  *string `json:"total_learning_deltas,omitempty" db:"total_learning_deltas"`
	KeyDecisions         *string `json:"key_decisions,omitempty" db:"key_decisions"`
	PatternsDiscovered   *string `json:"patterns_discovered,omitempty" db:"patterns_discovered"`
	MistakesSummary      *string `json:"mistakes_summary,omitempty" db:"mistakes_summary"`
	RemainingWork        *string `json:"remaining_work,omitempty" db:"remaining_work"`
	ReposTouched         *string `json:"repos_touched,omitempty" db:"repos_touched"`
	NextSessionBootstrap *string `json:"next_session_bootstrap,omitempty" db:"next_session_bootstrap"`
	HandoffData          string  `json:"-" db:"handoff_data"`
}

// ReferenceDoc represents a reference document for a project
type ReferenceDoc struct {
	ID               string  `json:"id" db:"id"`
	ProjectID        string  `json:"project_id" db:"project_id"`
	DocPath          string  `json:"doc_path" db:"doc_path"`
	DocType          *string `json:"doc_type,omitempty" db:"doc_type"`
	Description      *string `json:"description,omitempty" db:"description"`
	CreatedTimestamp float64 `json:"created_timestamp" db:"created_timestamp"`
	DocData          string  `json:"-" db:"doc_data"`
}

// NewReferenceDoc creates a new reference document
func NewReferenceDoc(projectID, docPath string, docType, description *string) *ReferenceDoc {
	return &ReferenceDoc{
		ID:               uuid.New().String(),
		ProjectID:        projectID,
		DocPath:          docPath,
		DocType:          docType,
		Description:      description,
		CreatedTimestamp: float64(time.Now().UnixMilli()) / 1000.0,
	}
}

// EpistemicSource represents a source of epistemic claims
type EpistemicSource struct {
	ID              string  `json:"id" db:"id"`
	ProjectID       string  `json:"project_id" db:"project_id"`
	SessionID       *string `json:"session_id,omitempty" db:"session_id"`
	SourceType      string  `json:"source_type" db:"source_type"` // doc, url, code, etc.
	SourceURL       *string `json:"source_url,omitempty" db:"source_url"`
	Title           string  `json:"title" db:"title"`
	Description     *string `json:"description,omitempty" db:"description"`
	Confidence      float64 `json:"confidence" db:"confidence"`
	EpistemicLayer  *string `json:"epistemic_layer,omitempty" db:"epistemic_layer"`
	SupportsVectors *string `json:"supports_vectors,omitempty" db:"supports_vectors"` // JSON
	RelatedFindings *string `json:"related_findings,omitempty" db:"related_findings"` // JSON array
	DiscoveredByAI  *string `json:"discovered_by_ai,omitempty" db:"discovered_by_ai"`
	DiscoveredAt    string  `json:"discovered_at" db:"discovered_at"`
	SourceMetadata  *string `json:"source_metadata,omitempty" db:"source_metadata"` // JSON
}

// InvestigationBranch represents a parallel investigation branch
type InvestigationBranch struct {
	ID                  string   `json:"id" db:"id"`
	SessionID           string   `json:"session_id" db:"session_id"`
	BranchName          string   `json:"branch_name" db:"branch_name"`
	InvestigationPath   string   `json:"investigation_path" db:"investigation_path"`
	GitBranchName       string   `json:"git_branch_name" db:"git_branch_name"`
	PreflightVectors    string   `json:"preflight_vectors" db:"preflight_vectors"` // JSON
	PostflightVectors   *string  `json:"postflight_vectors,omitempty" db:"postflight_vectors"`
	TokensSpent         int      `json:"tokens_spent" db:"tokens_spent"`
	TimeSpentMinutes    int      `json:"time_spent_minutes" db:"time_spent_minutes"`
	MergeScore          *float64 `json:"merge_score,omitempty" db:"merge_score"`
	EpistemicQuality    *float64 `json:"epistemic_quality,omitempty" db:"epistemic_quality"`
	IsWinner            bool     `json:"is_winner" db:"is_winner"`
	CreatedTimestamp    float64  `json:"created_timestamp" db:"created_timestamp"`
	CheckpointTimestamp *float64 `json:"checkpoint_timestamp,omitempty" db:"checkpoint_timestamp"`
	MergedTimestamp     *float64 `json:"merged_timestamp,omitempty" db:"merged_timestamp"`
	Status              string   `json:"status" db:"status"` // active, merged, abandoned
	BranchMetadata      *string  `json:"branch_metadata,omitempty" db:"branch_metadata"`
}

// NewInvestigationBranch creates a new investigation branch
func NewInvestigationBranch(sessionID, branchName, path, gitBranch string) *InvestigationBranch {
	return &InvestigationBranch{
		ID:                sessionID + "-" + branchName,
		SessionID:         sessionID,
		BranchName:        branchName,
		InvestigationPath: path,
		GitBranchName:     gitBranch,
		TokensSpent:       0,
		TimeSpentMinutes:  0,
		IsWinner:          false,
		CreatedTimestamp:  float64(time.Now().UnixMilli()) / 1000.0,
		Status:            "active",
	}
}

// MergeDecision represents a decision to merge investigation branches
type MergeDecision struct {
	ID                 string  `json:"id" db:"id"`
	SessionID          string  `json:"session_id" db:"session_id"`
	InvestigationRound int     `json:"investigation_round" db:"investigation_round"`
	WinningBranchID    string  `json:"winning_branch_id" db:"winning_branch_id"`
	WinningBranchName  *string `json:"winning_branch_name,omitempty" db:"winning_branch_name"`
	WinningScore       float64 `json:"winning_score" db:"winning_score"`
	OtherBranches      *string `json:"other_branches,omitempty" db:"other_branches"` // JSON
	DecisionRationale  string  `json:"decision_rationale" db:"decision_rationale"`
	AutoMerged         bool    `json:"auto_merged" db:"auto_merged"`
	CreatedTimestamp   float64 `json:"created_timestamp" db:"created_timestamp"`
	DecisionMetadata   *string `json:"decision_metadata,omitempty" db:"decision_metadata"`
}
