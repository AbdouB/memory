package models

import (
	"time"

	"github.com/google/uuid"
)

// Session represents an Empirica session for tracking AI agent epistemic state
type Session struct {
	SessionID        string     `json:"session_id" db:"session_id"`
	AIID             string     `json:"ai_id" db:"ai_id"`
	UserID           *string    `json:"user_id,omitempty" db:"user_id"`
	StartTime        time.Time  `json:"start_time" db:"start_time"`
	EndTime          *time.Time `json:"end_time,omitempty" db:"end_time"`
	ComponentsLoaded int        `json:"components_loaded" db:"components_loaded"`
	TotalTurns       int        `json:"total_turns" db:"total_turns"`
	TotalCascades    int        `json:"total_cascades" db:"total_cascades"`
	AvgConfidence    *float64   `json:"avg_confidence,omitempty" db:"avg_confidence"`
	DriftDetected    bool       `json:"drift_detected" db:"drift_detected"`
	SessionNotes     *string    `json:"session_notes,omitempty" db:"session_notes"`
	BootstrapLevel   int        `json:"bootstrap_level" db:"bootstrap_level"` // 1-3
	ProjectID        *string    `json:"project_id,omitempty" db:"project_id"`
	Subject          *string    `json:"subject,omitempty" db:"subject"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}

// NewSession creates a new session with default values
func NewSession(aiID string) *Session {
	now := time.Now()
	return &Session{
		SessionID:        uuid.New().String(),
		AIID:             aiID,
		StartTime:        now,
		ComponentsLoaded: 0,
		TotalTurns:       0,
		TotalCascades:    0,
		DriftDetected:    false,
		BootstrapLevel:   1,
		CreatedAt:        now,
	}
}

// SessionCreateInput represents input for creating a new session
type SessionCreateInput struct {
	AIID        string  `json:"ai_id"`
	UserID      *string `json:"user_id,omitempty"`
	ProjectID   *string `json:"project_id,omitempty"`
	Subject     *string `json:"subject,omitempty"`
	SessionType *string `json:"session_type,omitempty"` // development, research, etc.
}

// SessionOutput represents the output format for session commands
type SessionOutput struct {
	SessionID      string  `json:"session_id"`
	AIID           string  `json:"ai_id"`
	Status         string  `json:"status"`
	BootstrapLevel int     `json:"bootstrap_level"`
	Message        string  `json:"message,omitempty"`
	ProjectID      *string `json:"project_id,omitempty"`
}

// Cascade represents a CASCADE workflow instance within a session
type Cascade struct {
	CascadeID            string     `json:"cascade_id" db:"cascade_id"`
	SessionID            string     `json:"session_id" db:"session_id"`
	Task                 string     `json:"task" db:"task"`
	ContextJSON          *string    `json:"context_json,omitempty" db:"context_json"`
	GoalID               *string    `json:"goal_id,omitempty" db:"goal_id"`
	GoalJSON             *string    `json:"goal_json,omitempty" db:"goal_json"`
	PreflightCompleted   bool       `json:"preflight_completed" db:"preflight_completed"`
	ThinkCompleted       bool       `json:"think_completed" db:"think_completed"`
	PlanCompleted        bool       `json:"plan_completed" db:"plan_completed"`
	InvestigateCompleted bool       `json:"investigate_completed" db:"investigate_completed"`
	CheckCompleted       bool       `json:"check_completed" db:"check_completed"`
	ActCompleted         bool       `json:"act_completed" db:"act_completed"`
	PostflightCompleted  bool       `json:"postflight_completed" db:"postflight_completed"`
	FinalAction          *string    `json:"final_action,omitempty" db:"final_action"`
	FinalConfidence      *float64   `json:"final_confidence,omitempty" db:"final_confidence"`
	InvestigationRounds  int        `json:"investigation_rounds" db:"investigation_rounds"`
	DurationMS           *int       `json:"duration_ms,omitempty" db:"duration_ms"`
	StartedAt            time.Time  `json:"started_at" db:"started_at"`
	CompletedAt          *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	EngagementGatePassed *bool      `json:"engagement_gate_passed,omitempty" db:"engagement_gate_passed"`
	BayesianActive       bool       `json:"bayesian_active" db:"bayesian_active"`
	DriftMonitored       bool       `json:"drift_monitored" db:"drift_monitored"`
}

// NewCascade creates a new CASCADE workflow instance
func NewCascade(sessionID, task string) *Cascade {
	return &Cascade{
		CascadeID:           uuid.New().String(),
		SessionID:           sessionID,
		Task:                task,
		StartedAt:           time.Now(),
		InvestigationRounds: 0,
		BayesianActive:      false,
		DriftMonitored:      false,
	}
}

// Reflex represents an epistemic checkpoint within a CASCADE workflow
type Reflex struct {
	ID          int64    `json:"id" db:"id"`
	SessionID   string   `json:"session_id" db:"session_id"`
	CascadeID   *string  `json:"cascade_id,omitempty" db:"cascade_id"`
	Phase       string   `json:"phase" db:"phase"` // PREFLIGHT, CHECK, POSTFLIGHT
	Round       int      `json:"round" db:"round"`
	Timestamp   float64  `json:"timestamp" db:"timestamp"`
	Engagement  *float64 `json:"engagement,omitempty" db:"engagement"`
	Know        *float64 `json:"know,omitempty" db:"know"`
	Do          *float64 `json:"do,omitempty" db:"do_vec"`
	Context     *float64 `json:"context,omitempty" db:"context"`
	Clarity     *float64 `json:"clarity,omitempty" db:"clarity"`
	Coherence   *float64 `json:"coherence,omitempty" db:"coherence"`
	Signal      *float64 `json:"signal,omitempty" db:"signal"`
	Density     *float64 `json:"density,omitempty" db:"density"`
	State       *float64 `json:"state,omitempty" db:"state"`
	Change      *float64 `json:"change,omitempty" db:"change"`
	Completion  *float64 `json:"completion,omitempty" db:"completion"`
	Impact      *float64 `json:"impact,omitempty" db:"impact"`
	Uncertainty *float64 `json:"uncertainty,omitempty" db:"uncertainty"`
	ReflexData  *string  `json:"reflex_data,omitempty" db:"reflex_data"`
	Reasoning   *string  `json:"reasoning,omitempty" db:"reasoning"`
	Evidence    *string  `json:"evidence,omitempty" db:"evidence"`
}

// NewReflex creates a new epistemic reflex/checkpoint
func NewReflex(sessionID, phase string, vectors *EpistemicVectors, round int) *Reflex {
	r := &Reflex{
		SessionID: sessionID,
		Phase:     phase,
		Round:     round,
		Timestamp: float64(time.Now().UnixMilli()) / 1000.0,
	}

	if vectors != nil {
		r.Engagement = &vectors.Engagement
		r.Know = &vectors.Know
		r.Do = &vectors.Do
		r.Context = &vectors.Context
		r.Clarity = &vectors.Clarity
		r.Coherence = &vectors.Coherence
		r.Signal = &vectors.Signal
		r.Density = &vectors.Density
		r.State = &vectors.State
		r.Change = &vectors.Change
		r.Completion = &vectors.Completion
		r.Impact = &vectors.Impact
		r.Uncertainty = &vectors.Uncertainty
	}

	return r
}

// ToVectors converts a reflex to EpistemicVectors
func (r *Reflex) ToVectors() *EpistemicVectors {
	v := &EpistemicVectors{}
	if r.Engagement != nil {
		v.Engagement = *r.Engagement
	}
	if r.Know != nil {
		v.Know = *r.Know
	}
	if r.Do != nil {
		v.Do = *r.Do
	}
	if r.Context != nil {
		v.Context = *r.Context
	}
	if r.Clarity != nil {
		v.Clarity = *r.Clarity
	}
	if r.Coherence != nil {
		v.Coherence = *r.Coherence
	}
	if r.Signal != nil {
		v.Signal = *r.Signal
	}
	if r.Density != nil {
		v.Density = *r.Density
	}
	if r.State != nil {
		v.State = *r.State
	}
	if r.Change != nil {
		v.Change = *r.Change
	}
	if r.Completion != nil {
		v.Completion = *r.Completion
	}
	if r.Impact != nil {
		v.Impact = *r.Impact
	}
	if r.Uncertainty != nil {
		v.Uncertainty = *r.Uncertainty
	}
	return v
}

// CASCADEPhase represents a phase in the CASCADE workflow
type CASCADEPhase string

const (
	PhasePreflight   CASCADEPhase = "PREFLIGHT"
	PhaseCheck       CASCADEPhase = "CHECK"
	PhasePostflight  CASCADEPhase = "POSTFLIGHT"
	PhaseThink       CASCADEPhase = "THINK"
	PhasePlan        CASCADEPhase = "PLAN"
	PhaseInvestigate CASCADEPhase = "INVESTIGATE"
	PhaseAct         CASCADEPhase = "ACT"
)

// HandoffReport represents a session handoff report for continuity
type HandoffReport struct {
	SessionID              string   `json:"session_id" db:"session_id"`
	AIID                   string   `json:"ai_id" db:"ai_id"`
	ProjectID              *string  `json:"project_id,omitempty" db:"project_id"`
	Timestamp              string   `json:"timestamp" db:"timestamp"`
	TaskSummary            *string  `json:"task_summary,omitempty" db:"task_summary"`
	DurationSeconds        *float64 `json:"duration_seconds,omitempty" db:"duration_seconds"`
	EpistemicDeltas        *string  `json:"epistemic_deltas,omitempty" db:"epistemic_deltas"`
	KeyFindings            *string  `json:"key_findings,omitempty" db:"key_findings"`
	KnowledgeGapsFilled    *string  `json:"knowledge_gaps_filled,omitempty" db:"knowledge_gaps_filled"`
	RemainingUnknowns      *string  `json:"remaining_unknowns,omitempty" db:"remaining_unknowns"`
	NoeticTools            *string  `json:"noetic_tools,omitempty" db:"noetic_tools"`
	NextSessionContext     *string  `json:"next_session_context,omitempty" db:"next_session_context"`
	RecommendedNextSteps   *string  `json:"recommended_next_steps,omitempty" db:"recommended_next_steps"`
	ArtifactsCreated       *string  `json:"artifacts_created,omitempty" db:"artifacts_created"`
	CalibrationStatus      *string  `json:"calibration_status,omitempty" db:"calibration_status"`
	OverallConfidenceDelta *float64 `json:"overall_confidence_delta,omitempty" db:"overall_confidence_delta"`
	CompressedJSON         *string  `json:"compressed_json,omitempty" db:"compressed_json"`
	MarkdownReport         *string  `json:"markdown_report,omitempty" db:"markdown_report"`
	CreatedAt              float64  `json:"created_at" db:"created_at"`
}

// HandoffCreateInput represents input for creating a handoff
type HandoffCreateInput struct {
	SessionID          string   `json:"session_id"`
	ProjectID          string   `json:"project_id,omitempty"`
	TaskSummary        string   `json:"task_summary"`
	KeyFindings        []string `json:"key_findings,omitempty"`
	RemainingUnknowns  []string `json:"remaining_unknowns,omitempty"`
	NextSessionContext string   `json:"next_session_context,omitempty"`
	Artifacts          []string `json:"artifacts,omitempty"`
	PlanningOnly       bool     `json:"planning_only,omitempty"`
}
