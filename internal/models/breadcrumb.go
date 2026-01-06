package models

import (
	"math"
	"time"

	"github.com/google/uuid"
)

// StalenessStatus represents how fresh a finding is
type StalenessStatus string

const (
	StatusFresh StalenessStatus = "fresh" // >=70% confidence
	StatusAging StalenessStatus = "aging" // 40-70% confidence
	StatusStale StalenessStatus = "stale" // <40% confidence
)

// DecayHalfLifeDays is the number of days for confidence to halve
const DecayHalfLifeDays = 14.0

// FileChangeConfidenceMultiplier is applied when referenced file changes
const FileChangeConfidenceMultiplier = 0.5

// BreadcrumbScope determines where breadcrumbs are stored
type BreadcrumbScope string

const (
	ScopeSession BreadcrumbScope = "session" // Ephemeral, session-specific
	ScopeProject BreadcrumbScope = "project" // Persistent, cross-session
	ScopeBoth    BreadcrumbScope = "both"    // Dual-log for important discoveries
)

// Finding represents a discovered fact or insight
type Finding struct {
	ID                    string   `json:"id" db:"id"`
	ProjectID             string   `json:"project_id" db:"project_id"`
	SessionID             string   `json:"session_id" db:"session_id"`
	GoalID                *string  `json:"goal_id,omitempty" db:"goal_id"`
	SubtaskID             *string  `json:"subtask_id,omitempty" db:"subtask_id"`
	Finding               string   `json:"finding" db:"finding"`
	CreatedTimestamp      float64  `json:"created_timestamp" db:"created_timestamp"`
	Subject               *string  `json:"subject,omitempty" db:"subject"`
	Impact                float64  `json:"impact" db:"impact"` // 0.0-1.0
	FindingData           string   `json:"-" db:"finding_data"`
	LastVerifiedTimestamp *float64 `json:"last_verified_timestamp,omitempty" db:"last_verified_timestamp"`
	SubjectGitHash        *string  `json:"subject_git_hash,omitempty" db:"subject_git_hash"`
}

// CalculateConfidence returns the time-decayed confidence (0.0-1.0)
// Uses exponential decay with 14-day half-life
func (f *Finding) CalculateConfidence() float64 {
	// Use last verified timestamp if available, otherwise use created timestamp
	baseTime := f.CreatedTimestamp
	if f.LastVerifiedTimestamp != nil {
		baseTime = *f.LastVerifiedTimestamp
	}

	// Calculate days since base time
	now := float64(time.Now().UnixMilli()) / 1000.0
	daysSince := (now - baseTime) / (24 * 60 * 60)

	// Exponential decay: confidence = e^(-lambda * t)
	// where lambda = ln(2) / half_life
	lambda := math.Log(2) / DecayHalfLifeDays
	confidence := math.Exp(-lambda * daysSince)

	return confidence
}

// GetStalenessStatus returns the staleness status based on confidence and file changes
func (f *Finding) GetStalenessStatus(fileChanged bool) StalenessStatus {
	confidence := f.CalculateConfidence()

	// Apply file change penalty
	if fileChanged {
		confidence *= FileChangeConfidenceMultiplier
	}

	if confidence >= 0.70 {
		return StatusFresh
	} else if confidence >= 0.40 {
		return StatusAging
	}
	return StatusStale
}

// DaysSinceVerified returns the number of days since last verification (or creation)
func (f *Finding) DaysSinceVerified() float64 {
	baseTime := f.CreatedTimestamp
	if f.LastVerifiedTimestamp != nil {
		baseTime = *f.LastVerifiedTimestamp
	}
	now := float64(time.Now().UnixMilli()) / 1000.0
	return (now - baseTime) / (24 * 60 * 60)
}

// NewFinding creates a new finding
func NewFinding(projectID, sessionID, finding string, impact float64) *Finding {
	return &Finding{
		ID:               uuid.New().String(),
		ProjectID:        projectID,
		SessionID:        sessionID,
		Finding:          finding,
		CreatedTimestamp: float64(time.Now().UnixMilli()) / 1000.0,
		Impact:           impact,
	}
}

// FindingLogInput represents input for logging a finding
type FindingLogInput struct {
	ProjectID string          `json:"project_id,omitempty"`
	SessionID string          `json:"session_id"`
	Finding   string          `json:"finding"`
	GoalID    *string         `json:"goal_id,omitempty"`
	SubtaskID *string         `json:"subtask_id,omitempty"`
	Subject   *string         `json:"subject,omitempty"`
	Impact    float64         `json:"impact"`
	Scope     BreadcrumbScope `json:"scope,omitempty"`
}

// Unknown represents a knowledge gap or unanswered question
type Unknown struct {
	ID                string   `json:"id" db:"id"`
	ProjectID         string   `json:"project_id" db:"project_id"`
	SessionID         string   `json:"session_id" db:"session_id"`
	GoalID            *string  `json:"goal_id,omitempty" db:"goal_id"`
	SubtaskID         *string  `json:"subtask_id,omitempty" db:"subtask_id"`
	Unknown           string   `json:"unknown" db:"unknown"`
	IsResolved        bool     `json:"is_resolved" db:"is_resolved"`
	ResolvedBy        *string  `json:"resolved_by,omitempty" db:"resolved_by"`
	CreatedTimestamp  float64  `json:"created_timestamp" db:"created_timestamp"`
	ResolvedTimestamp *float64 `json:"resolved_timestamp,omitempty" db:"resolved_timestamp"`
	Subject           *string  `json:"subject,omitempty" db:"subject"`
	Impact            float64  `json:"impact" db:"impact"`
	UnknownData       string   `json:"-" db:"unknown_data"`
}

// NewUnknown creates a new unknown
func NewUnknown(projectID, sessionID, unknown string, impact float64) *Unknown {
	return &Unknown{
		ID:               uuid.New().String(),
		ProjectID:        projectID,
		SessionID:        sessionID,
		Unknown:          unknown,
		IsResolved:       false,
		CreatedTimestamp: float64(time.Now().UnixMilli()) / 1000.0,
		Impact:           impact,
	}
}

// UnknownLogInput represents input for logging an unknown
type UnknownLogInput struct {
	ProjectID string          `json:"project_id,omitempty"`
	SessionID string          `json:"session_id"`
	Unknown   string          `json:"unknown"`
	GoalID    *string         `json:"goal_id,omitempty"`
	SubtaskID *string         `json:"subtask_id,omitempty"`
	Subject   *string         `json:"subject,omitempty"`
	Impact    float64         `json:"impact"`
	Scope     BreadcrumbScope `json:"scope,omitempty"`
}

// DeadEnd represents a failed approach that shouldn't be repeated
type DeadEnd struct {
	ID               string  `json:"id" db:"id"`
	ProjectID        string  `json:"project_id" db:"project_id"`
	SessionID        string  `json:"session_id" db:"session_id"`
	GoalID           *string `json:"goal_id,omitempty" db:"goal_id"`
	SubtaskID        *string `json:"subtask_id,omitempty" db:"subtask_id"`
	Approach         string  `json:"approach" db:"approach"`
	WhyFailed        string  `json:"why_failed" db:"why_failed"`
	CreatedTimestamp float64 `json:"created_timestamp" db:"created_timestamp"`
	Subject          *string `json:"subject,omitempty" db:"subject"`
	Impact           float64 `json:"impact" db:"impact"`
	DeadEndData      string  `json:"-" db:"dead_end_data"`
}

// NewDeadEnd creates a new dead end record
func NewDeadEnd(projectID, sessionID, approach, whyFailed string, impact float64) *DeadEnd {
	return &DeadEnd{
		ID:               uuid.New().String(),
		ProjectID:        projectID,
		SessionID:        sessionID,
		Approach:         approach,
		WhyFailed:        whyFailed,
		CreatedTimestamp: float64(time.Now().UnixMilli()) / 1000.0,
		Impact:           impact,
	}
}

// DeadEndLogInput represents input for logging a dead end
type DeadEndLogInput struct {
	ProjectID string          `json:"project_id,omitempty"`
	SessionID string          `json:"session_id"`
	Approach  string          `json:"approach"`
	WhyFailed string          `json:"why_failed"`
	GoalID    *string         `json:"goal_id,omitempty"`
	SubtaskID *string         `json:"subtask_id,omitempty"`
	Subject   *string         `json:"subject,omitempty"`
	Impact    float64         `json:"impact"`
	Scope     BreadcrumbScope `json:"scope,omitempty"`
}

// RootCauseVector represents which epistemic vector caused a mistake
type RootCauseVector string

const (
	RootCauseKnow        RootCauseVector = "KNOW"
	RootCauseContext     RootCauseVector = "CONTEXT"
	RootCauseClarity     RootCauseVector = "CLARITY"
	RootCauseCoherence   RootCauseVector = "COHERENCE"
	RootCauseUncertainty RootCauseVector = "UNCERTAINTY"
)

// Mistake represents an error made by the AI agent
type Mistake struct {
	ID               string           `json:"id" db:"id"`
	SessionID        string           `json:"session_id" db:"session_id"`
	GoalID           *string          `json:"goal_id,omitempty" db:"goal_id"`
	ProjectID        *string          `json:"project_id,omitempty" db:"project_id"`
	Mistake          string           `json:"mistake" db:"mistake"`
	WhyWrong         string           `json:"why_wrong" db:"why_wrong"`
	CostEstimate     *string          `json:"cost_estimate,omitempty" db:"cost_estimate"`
	RootCauseVector  *RootCauseVector `json:"root_cause_vector,omitempty" db:"root_cause_vector"`
	Prevention       *string          `json:"prevention,omitempty" db:"prevention"`
	CreatedTimestamp float64          `json:"created_timestamp" db:"created_timestamp"`
	MistakeData      string           `json:"-" db:"mistake_data"`
}

// NewMistake creates a new mistake record
func NewMistake(sessionID, mistake, whyWrong string) *Mistake {
	return &Mistake{
		ID:               uuid.New().String(),
		SessionID:        sessionID,
		Mistake:          mistake,
		WhyWrong:         whyWrong,
		CreatedTimestamp: float64(time.Now().UnixMilli()) / 1000.0,
	}
}

// MistakeLogInput represents input for logging a mistake
type MistakeLogInput struct {
	SessionID       string           `json:"session_id"`
	Mistake         string           `json:"mistake"`
	WhyWrong        string           `json:"why_wrong"`
	GoalID          *string          `json:"goal_id,omitempty"`
	ProjectID       *string          `json:"project_id,omitempty"`
	CostEstimate    *string          `json:"cost_estimate,omitempty"`
	RootCauseVector *RootCauseVector `json:"root_cause_vector,omitempty"`
	Prevention      *string          `json:"prevention,omitempty"`
	Scope           BreadcrumbScope  `json:"scope,omitempty"`
}
