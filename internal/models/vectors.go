// Package models contains core data structures for Empirica
package models

import (
	"encoding/json"
	"math"
)

// EpistemicVectors represents the 13-dimensional epistemic vector space
// Used for tracking AI agent's knowledge state across CASCADE workflow
type EpistemicVectors struct {
	// Gate (required threshold: >= 0.60)
	Engagement float64 `json:"engagement"`

	// Foundation (Tier 0) - Weight: 35%
	Know    float64 `json:"know"`    // Domain knowledge level
	Do      float64 `json:"do"`      // Execution capability
	Context float64 `json:"context"` // Situational awareness

	// Comprehension (Tier 1) - Weight: 25%
	Clarity   float64 `json:"clarity"`   // Clear understanding
	Coherence float64 `json:"coherence"` // Logical consistency
	Signal    float64 `json:"signal"`    // Signal-to-noise ratio
	Density   float64 `json:"density"`   // Information density (high = overload)

	// Execution (Tier 2) - Weight: 25%
	State      float64 `json:"state"`      // Current state mapping
	Change     float64 `json:"change"`     // Safe change capability
	Completion float64 `json:"completion"` // Task completion
	Impact     float64 `json:"impact"`     // Expected impact

	// Meta
	Uncertainty float64 `json:"uncertainty"` // Explicit doubt level (lower is better)
}

// Canonical weights for tier calculations
var CanonicalWeights = map[string]float64{
	"foundation":    0.35,
	"comprehension": 0.25,
	"execution":     0.25,
	"engagement":    0.15,
}

// Thresholds for confidence levels
const (
	EngagementThreshold = 0.60
	ConfidenceHigh      = 0.85
	ConfidenceModerate  = 0.70
	ConfidenceLow       = 0.50
	UncertaintyLow      = 0.30 // Below this is good
	UncertaintyModerate = 0.50
	UncertaintyHigh     = 0.70 // Above this is concerning
)

// CriticalThresholds define system action boundaries
var CriticalThresholds = map[string]float64{
	"coherence_min": 0.50,
	"density_max":   0.90,
	"change_min":    0.50,
}

// NewDefaultVectors creates a new EpistemicVectors with moderate defaults
func NewDefaultVectors() *EpistemicVectors {
	return &EpistemicVectors{
		Engagement:  0.5,
		Know:        0.5,
		Do:          0.5,
		Context:     0.5,
		Clarity:     0.5,
		Coherence:   0.5,
		Signal:      0.5,
		Density:     0.5,
		State:       0.5,
		Change:      0.5,
		Completion:  0.0,
		Impact:      0.5,
		Uncertainty: 0.5,
	}
}

// FoundationScore calculates the weighted foundation tier score
func (v *EpistemicVectors) FoundationScore() float64 {
	return (v.Know + v.Do + v.Context) / 3.0
}

// ComprehensionScore calculates the weighted comprehension tier score
func (v *EpistemicVectors) ComprehensionScore() float64 {
	return (v.Clarity + v.Coherence + v.Signal + v.Density) / 4.0
}

// ExecutionScore calculates the weighted execution tier score
func (v *EpistemicVectors) ExecutionScore() float64 {
	return (v.State + v.Change + v.Completion + v.Impact) / 4.0
}

// OverallConfidence calculates weighted overall confidence score
func (v *EpistemicVectors) OverallConfidence() float64 {
	foundation := v.FoundationScore() * CanonicalWeights["foundation"]
	comprehension := v.ComprehensionScore() * CanonicalWeights["comprehension"]
	execution := v.ExecutionScore() * CanonicalWeights["execution"]
	engagement := v.Engagement * CanonicalWeights["engagement"]

	// Uncertainty acts as a penalty
	base := foundation + comprehension + execution + engagement
	penalty := v.Uncertainty * 0.15

	return math.Max(0, math.Min(1, base-penalty))
}

// PassesEngagementGate checks if engagement meets threshold
func (v *EpistemicVectors) PassesEngagementGate() bool {
	return v.Engagement >= EngagementThreshold
}

// IsReadyToProceed checks if the agent is ready to act
func (v *EpistemicVectors) IsReadyToProceed() bool {
	return v.PassesEngagementGate() &&
		v.Know >= ConfidenceLow &&
		v.Uncertainty <= UncertaintyModerate
}

// NeedsInvestigation checks if more research is needed
func (v *EpistemicVectors) NeedsInvestigation() bool {
	return v.Know < ConfidenceLow || v.Uncertainty > UncertaintyModerate
}

// Delta calculates the difference between two vector states
func (v *EpistemicVectors) Delta(other *EpistemicVectors) *EpistemicVectors {
	if other == nil {
		return v
	}
	return &EpistemicVectors{
		Engagement:  v.Engagement - other.Engagement,
		Know:        v.Know - other.Know,
		Do:          v.Do - other.Do,
		Context:     v.Context - other.Context,
		Clarity:     v.Clarity - other.Clarity,
		Coherence:   v.Coherence - other.Coherence,
		Signal:      v.Signal - other.Signal,
		Density:     v.Density - other.Density,
		State:       v.State - other.State,
		Change:      v.Change - other.Change,
		Completion:  v.Completion - other.Completion,
		Impact:      v.Impact - other.Impact,
		Uncertainty: v.Uncertainty - other.Uncertainty,
	}
}

// ToMap converts vectors to a map for JSON serialization
func (v *EpistemicVectors) ToMap() map[string]float64 {
	return map[string]float64{
		"engagement":  v.Engagement,
		"know":        v.Know,
		"do":          v.Do,
		"context":     v.Context,
		"clarity":     v.Clarity,
		"coherence":   v.Coherence,
		"signal":      v.Signal,
		"density":     v.Density,
		"state":       v.State,
		"change":      v.Change,
		"completion":  v.Completion,
		"impact":      v.Impact,
		"uncertainty": v.Uncertainty,
	}
}

// FromMap populates vectors from a map
func (v *EpistemicVectors) FromMap(m map[string]float64) {
	if val, ok := m["engagement"]; ok {
		v.Engagement = val
	}
	if val, ok := m["know"]; ok {
		v.Know = val
	}
	if val, ok := m["do"]; ok {
		v.Do = val
	}
	if val, ok := m["context"]; ok {
		v.Context = val
	}
	if val, ok := m["clarity"]; ok {
		v.Clarity = val
	}
	if val, ok := m["coherence"]; ok {
		v.Coherence = val
	}
	if val, ok := m["signal"]; ok {
		v.Signal = val
	}
	if val, ok := m["density"]; ok {
		v.Density = val
	}
	if val, ok := m["state"]; ok {
		v.State = val
	}
	if val, ok := m["change"]; ok {
		v.Change = val
	}
	if val, ok := m["completion"]; ok {
		v.Completion = val
	}
	if val, ok := m["impact"]; ok {
		v.Impact = val
	}
	if val, ok := m["uncertainty"]; ok {
		v.Uncertainty = val
	}
}

// ToJSON serializes vectors to JSON
func (v *EpistemicVectors) ToJSON() (string, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// FromJSON deserializes vectors from JSON
func FromJSON(data string) (*EpistemicVectors, error) {
	v := &EpistemicVectors{}
	err := json.Unmarshal([]byte(data), v)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// MoonPhase returns a moon phase indicator for epistemic health
// Used for quick visual feedback in CLI
func (v *EpistemicVectors) MoonPhase() string {
	confidence := v.OverallConfidence()
	switch {
	case confidence < 0.25:
		return "🌑" // New moon - critical
	case confidence < 0.50:
		return "🌒" // Waxing crescent - low
	case confidence < 0.75:
		return "🌓" // First quarter - moderate
	case confidence < 0.90:
		return "🌔" // Waxing gibbous - good
	default:
		return "🌕" // Full moon - excellent
	}
}

// Action represents the recommended action based on epistemic state
type Action string

const (
	ActionProceed     Action = "proceed"
	ActionInvestigate Action = "investigate"
	ActionClarify     Action = "clarify"
	ActionReset       Action = "reset"
	ActionStop        Action = "stop"
)

// RecommendedAction determines what action to take based on vectors
func (v *EpistemicVectors) RecommendedAction() Action {
	if !v.PassesEngagementGate() {
		return ActionStop
	}

	if v.Coherence < CriticalThresholds["coherence_min"] {
		return ActionReset
	}

	if v.Density > CriticalThresholds["density_max"] {
		return ActionClarify
	}

	if v.NeedsInvestigation() {
		return ActionInvestigate
	}

	return ActionProceed
}

// VectorAssessment represents a single vector assessment with rationale
type VectorAssessment struct {
	Score                 float64 `json:"score"`
	Rationale             string  `json:"rationale"`
	Evidence              string  `json:"evidence,omitempty"`
	WarrantsInvestigation bool    `json:"warrants_investigation,omitempty"`
	InvestigationPriority int     `json:"investigation_priority,omitempty"`
}
