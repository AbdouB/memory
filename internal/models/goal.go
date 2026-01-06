package models

import (
	"time"

	"github.com/google/uuid"
)

// ScopeVector represents the goal scope dimensions
type ScopeVector struct {
	Breadth      float64 `json:"breadth"`      // 0.0-1.0: scope width
	Duration     float64 `json:"duration"`     // 0.0-1.0: expected lifetime
	Coordination float64 `json:"coordination"` // 0.0-1.0: multi-agent need
}

// SuccessCriterion defines a measurable success condition
type SuccessCriterion struct {
	ID               string   `json:"id"`
	Description      string   `json:"description"`
	ValidationMethod string   `json:"validation_method"` // completion, quality_gate, metric_threshold
	Threshold        *float64 `json:"threshold,omitempty"`
	IsRequired       bool     `json:"is_required"`
	IsMet            bool     `json:"is_met"`
}

// Dependency represents a goal dependency
type Dependency struct {
	ID             string `json:"id"`
	GoalID         string `json:"goal_id"`
	DependencyType string `json:"dependency_type"` // prerequisite, concurrent, informational
	Description    string `json:"description"`
}

// GoalStatus represents the current state of a goal
type GoalStatus string

const (
	GoalStatusInProgress GoalStatus = "in_progress"
	GoalStatusComplete   GoalStatus = "complete"
	GoalStatusBlocked    GoalStatus = "blocked"
	GoalStatusCancelled  GoalStatus = "cancelled"
)

// Goal represents an epistemic goal for AI agents
type Goal struct {
	ID                  string             `json:"id" db:"id"`
	SessionID           string             `json:"session_id" db:"session_id"`
	Objective           string             `json:"objective" db:"objective"`
	Scope               ScopeVector        `json:"scope"`
	ScopeJSON           string             `json:"-" db:"scope"` // For DB storage
	SuccessCriteria     []SuccessCriterion `json:"success_criteria"`
	Dependencies        []Dependency       `json:"dependencies"`
	Constraints         map[string]any     `json:"constraints"`
	Metadata            map[string]any     `json:"metadata"`
	EstimatedComplexity *float64           `json:"estimated_complexity,omitempty" db:"estimated_complexity"`
	CreatedTimestamp    float64            `json:"created_timestamp" db:"created_timestamp"`
	CompletedTimestamp  *float64           `json:"completed_timestamp,omitempty" db:"completed_timestamp"`
	IsCompleted         bool               `json:"is_completed" db:"is_completed"`
	Status              GoalStatus         `json:"status" db:"status"`
	BeadsIssueID        *string            `json:"beads_issue_id,omitempty" db:"beads_issue_id"`
	GoalData            string             `json:"-" db:"goal_data"` // Full JSON
}

// NewGoal creates a new goal
func NewGoal(sessionID, objective string, scope ScopeVector) *Goal {
	return &Goal{
		ID:               uuid.New().String(),
		SessionID:        sessionID,
		Objective:        objective,
		Scope:            scope,
		SuccessCriteria:  []SuccessCriterion{},
		Dependencies:     []Dependency{},
		Constraints:      make(map[string]any),
		Metadata:         make(map[string]any),
		CreatedTimestamp: float64(time.Now().UnixMilli()) / 1000.0,
		IsCompleted:      false,
		Status:           GoalStatusInProgress,
	}
}

// GoalCreateInput represents input for creating a goal
type GoalCreateInput struct {
	SessionID           string      `json:"session_id"`
	Objective           string      `json:"objective"`
	Scope               ScopeVector `json:"scope"`
	SuccessCriteria     []string    `json:"success_criteria,omitempty"`
	EstimatedComplexity *float64    `json:"estimated_complexity,omitempty"`
	UseBeads            bool        `json:"use_beads,omitempty"`
}

// EpistemicImportance represents the importance level of a subtask
type EpistemicImportance string

const (
	ImportanceCritical EpistemicImportance = "critical"
	ImportanceHigh     EpistemicImportance = "high"
	ImportanceMedium   EpistemicImportance = "medium"
	ImportanceLow      EpistemicImportance = "low"
)

// TaskStatus represents the status of a subtask
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusBlocked    TaskStatus = "blocked"
	TaskStatusSkipped    TaskStatus = "skipped"
)

// SubTask represents a subtask within a goal
type SubTask struct {
	ID                  string              `json:"id" db:"id"`
	GoalID              string              `json:"goal_id" db:"goal_id"`
	Description         string              `json:"description" db:"description"`
	Status              TaskStatus          `json:"status" db:"status"`
	EpistemicImportance EpistemicImportance `json:"epistemic_importance" db:"epistemic_importance"`
	Dependencies        []string            `json:"dependencies"` // Subtask IDs
	EstimatedTokens     *int                `json:"estimated_tokens,omitempty" db:"estimated_tokens"`
	ActualTokens        *int                `json:"actual_tokens,omitempty" db:"actual_tokens"`
	CompletionEvidence  *string             `json:"completion_evidence,omitempty" db:"completion_evidence"`
	Notes               string              `json:"notes" db:"notes"`
	CreatedTimestamp    float64             `json:"created_timestamp" db:"created_timestamp"`
	CompletedTimestamp  *float64            `json:"completed_timestamp,omitempty" db:"completed_timestamp"`
	Findings            []string            `json:"findings"`  // Finding IDs
	Unknowns            []string            `json:"unknowns"`  // Unknown IDs
	DeadEnds            []string            `json:"dead_ends"` // DeadEnd IDs
	SubtaskData         string              `json:"-" db:"subtask_data"`
}

// NewSubTask creates a new subtask
func NewSubTask(goalID, description string, importance EpistemicImportance) *SubTask {
	return &SubTask{
		ID:                  uuid.New().String(),
		GoalID:              goalID,
		Description:         description,
		Status:              TaskStatusPending,
		EpistemicImportance: importance,
		Dependencies:        []string{},
		CreatedTimestamp:    float64(time.Now().UnixMilli()) / 1000.0,
		Findings:            []string{},
		Unknowns:            []string{},
		DeadEnds:            []string{},
	}
}

// SubTaskCreateInput represents input for creating a subtask
type SubTaskCreateInput struct {
	GoalID       string              `json:"goal_id"`
	Description  string              `json:"description"`
	Importance   EpistemicImportance `json:"importance,omitempty"`
	Dependencies []string            `json:"dependencies,omitempty"`
	UseBeads     bool                `json:"use_beads,omitempty"`
}
