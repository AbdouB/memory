package models

// SessionContext is the AI-first response when starting a new session.
// Designed to provide all information an AI agent needs for a successful session.
type SessionContext struct {
	// === IDENTITY ===
	SessionID string `json:"session_id"`
	ProjectID string `json:"project_id"`
	Objective string `json:"objective"`

	// === DECISION SUPPORT ===
	// These fields tell the AI what to do RIGHT NOW
	Decision *DecisionGuidance `json:"decision"`

	// === CRITICAL: VERIFY BEFORE USING ===
	// Stale knowledge that MUST be verified before relying on it
	// Empty means nothing needs verification
	RequiresVerification []VerificationNeeded `json:"requires_verification,omitempty"`

	// === WARNINGS: DO NOT REPEAT ===
	// Failed approaches from previous sessions - avoid these mistakes
	// Each entry includes WHY it failed so the AI can understand the reasoning
	DeadEnds []DeadEndWarning `json:"dead_ends,omitempty"`

	// === CURRENT KNOWLEDGE ===
	// Fresh, reliable findings that can be used with confidence
	Knowledge []KnowledgeItem `json:"knowledge,omitempty"`

	// === OPEN QUESTIONS ===
	// Unresolved uncertainties from previous sessions
	// Consider investigating these if relevant to current objective
	OpenQuestions []string `json:"open_questions,omitempty"`

	// === LAST SESSION HANDOFF ===
	// Context from the previous session for continuity
	Continuity *ContinuityContext `json:"continuity,omitempty"`

	// === EPISTEMIC STATE ===
	// Numerical vectors for agents that want to reason about confidence
	Vectors *EpistemicSnapshot `json:"vectors,omitempty"`
}

// DecisionGuidance provides immediate actionable guidance for the AI
type DecisionGuidance struct {
	// Can the AI proceed with confidence, or should it investigate first?
	ReadyToProceed bool `json:"ready_to_proceed"`

	// Primary recommended action: "proceed", "investigate", "verify", "reset"
	Action string `json:"action"`

	// Human-readable explanation of the recommendation
	Reason string `json:"reason"`

	// Specific things to do before proceeding (if not ready)
	Prerequisites []string `json:"prerequisites,omitempty"`

	// Confidence level as moon phase for quick visual parsing
	// 🌑 = critical, 🌒 = low, 🌓 = moderate, 🌔 = good, 🌕 = excellent
	ConfidencePhase string `json:"confidence_phase"`

	// Numeric confidence 0.0-1.0 for programmatic use
	Confidence float64 `json:"confidence"`
}

// VerificationNeeded represents a piece of knowledge that should be verified
type VerificationNeeded struct {
	// The finding text that may be outdated
	Finding string `json:"finding"`

	// Finding ID for use with `memory verify --id`
	ID string `json:"id"`

	// Days since last verification
	DaysStale int `json:"days_stale"`

	// Current confidence level (0.0-1.0)
	Confidence float64 `json:"confidence"`

	// If scoped to a file, whether that file has changed
	FileChanged bool `json:"file_changed,omitempty"`

	// The file this finding is scoped to (if any)
	Scope string `json:"scope,omitempty"`

	// Suggested verification command
	VerifyCommand string `json:"verify_command"`
}

// DeadEndWarning represents a failed approach that should NOT be repeated
type DeadEndWarning struct {
	// What was tried
	Approach string `json:"approach"`

	// Why it failed - this is crucial context for the AI
	WhyFailed string `json:"why_failed"`

	// Related subject/file if applicable
	Scope string `json:"scope,omitempty"`
}

// KnowledgeItem represents a verified, fresh finding
type KnowledgeItem struct {
	// The finding/insight
	Finding string `json:"finding"`

	// Confidence level 0.0-1.0 (fresh findings are >= 0.7)
	Confidence float64 `json:"confidence"`

	// Staleness indicator: "fresh", "aging"
	Status string `json:"status"`

	// File scope if applicable
	Scope string `json:"scope,omitempty"`
}

// ContinuityContext provides handoff from previous session
type ContinuityContext struct {
	// What was accomplished in the last session
	Summary string `json:"summary,omitempty"`

	// Recommendations for this session
	Recommendations string `json:"recommendations,omitempty"`

	// Key findings from last session (already included in Knowledge, but highlighted)
	Highlights []string `json:"highlights,omitempty"`

	// Time since last session ended
	TimeSinceLastSession string `json:"time_since_last_session,omitempty"`
}

// EpistemicSnapshot provides numeric vectors for programmatic reasoning
type EpistemicSnapshot struct {
	// Core vectors (0.0-1.0)
	Know        float64 `json:"know"`        // Domain knowledge level
	Uncertainty float64 `json:"uncertainty"` // Knowledge gaps (lower is better)
	Clarity     float64 `json:"clarity"`     // Information freshness
	Coherence   float64 `json:"coherence"`   // Logical consistency (dead ends reduce this)
	Completion  float64 `json:"completion"`  // Resolved vs open unknowns
	Engagement  float64 `json:"engagement"`  // Session activity/freshness

	// Aggregate confidence score
	Overall float64 `json:"overall"`
}

// StartResponse is the complete response from `memory start`
type StartResponse struct {
	// Status is always "started" on success
	Status string `json:"status"`

	// The full session context
	Context *SessionContext `json:"context"`
}

// StatusResponse is the response from `memory status`
type StatusResponse struct {
	// Status indicates session state: "active", "no_session"
	Status string `json:"status"`

	// Session duration
	Duration string `json:"duration,omitempty"`

	// Session breadcrumb counts
	Counts *BreadcrumbCounts `json:"counts,omitempty"`

	// The full session context (same structure as start)
	Context *SessionContext `json:"context,omitempty"`

	// Message when no session is active
	Message string `json:"message,omitempty"`
}

// BreadcrumbCounts provides counts of different breadcrumb types
type BreadcrumbCounts struct {
	Findings         int `json:"findings"`
	FindingsFresh    int `json:"findings_fresh"`
	FindingsAging    int `json:"findings_aging"`
	FindingsStale    int `json:"findings_stale"`
	UnknownsResolved int `json:"unknowns_resolved"`
	UnknownsOpen     int `json:"unknowns_open"`
	DeadEnds         int `json:"dead_ends"`
}
