package cli

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/AbdouB/memory/internal/db"
	"github.com/AbdouB/memory/internal/models"
	"github.com/AbdouB/memory/internal/search"
	"github.com/spf13/cobra"
)

// ActiveSession stores the current active session info
type ActiveSession struct {
	SessionID     string    `json:"session_id"`
	AIID          string    `json:"ai_id"`
	Objective     string    `json:"objective"`
	StartedAt     time.Time `json:"started_at"`
	ProjectID     string    `json:"project_id,omitempty"`
	CurrentGoalID string    `json:"current_goal_id,omitempty"`
}

// getActiveSessionPath returns the path to store active session
func getActiveSessionPath() string {
	// Try project-local first
	if _, err := os.Stat(".memory"); err == nil {
		return ".memory/active-session.json"
	}
	// Fall back to home directory
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".memory", "active-session.json")
}

// saveActiveSession saves the current active session
func saveActiveSession(session *ActiveSession) error {
	path := getActiveSessionPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// loadActiveSession loads the current active session
func loadActiveSession() (*ActiveSession, error) {
	path := getActiveSessionPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var session ActiveSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// clearActiveSession removes the active session file
func clearActiveSession() error {
	path := getActiveSessionPath()
	return os.Remove(path)
}

// requireActiveSession gets the active session or returns an error
func requireActiveSession() (*ActiveSession, error) {
	session, err := loadActiveSession()
	if err != nil {
		return nil, fmt.Errorf("no active session. Run 'memory start \"objective\"' first")
	}
	return session, nil
}

// getOrCreateDefaultProject gets or creates a default project based on current directory
func getOrCreateDefaultProject() (*models.Project, error) {
	// Get current directory name as default project name
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "default"
	}
	projectName := filepath.Base(cwd)

	repo := db.NewProjectRepository(database)

	// Try to find existing project
	project, err := repo.GetByName(projectName)
	if err != nil {
		return nil, err
	}
	if project != nil {
		return project, nil
	}

	// Create new project
	project = models.NewProject(projectName, nil)
	if err := repo.Create(project); err != nil {
		return nil, err
	}

	return project, nil
}

// EpistemicState represents the calculated epistemic vectors
type EpistemicState struct {
	Know        float64 `json:"know"`
	Uncertainty float64 `json:"uncertainty"`
	Clarity     float64 `json:"clarity"`
	Coherence   float64 `json:"coherence"`
	Completion  float64 `json:"completion"`
	Engagement  float64 `json:"engagement"`
	Confidence  float64 `json:"confidence"`

	// Derived states
	PassesEngagementGate bool   `json:"passes_engagement_gate"`
	ReadyToProceed       bool   `json:"ready_to_proceed"`
	NeedsInvestigation   bool   `json:"needs_investigation"`
	RecommendedAction    string `json:"recommended_action"`
	MoonPhase            string `json:"moon_phase"`
}

// calculateEpistemicState derives epistemic vectors from breadcrumb data
func calculateEpistemicState(
	findings []*models.Finding,
	openUnknowns []*models.Unknown,
	resolvedUnknowns []*models.Unknown,
	deadEnds []*models.DeadEnd,
	sessionStart time.Time,
) *EpistemicState {
	state := &EpistemicState{}

	// Know: base 0.5 + findings × 0.1 + resolved × 0.15
	state.Know = 0.5 + float64(len(findings))*0.1 + float64(len(resolvedUnknowns))*0.15
	if state.Know > 1.0 {
		state.Know = 1.0
	}

	// Uncertainty: base 0.5 + open × 0.1 - resolved × 0.1
	state.Uncertainty = 0.5 + float64(len(openUnknowns))*0.1 - float64(len(resolvedUnknowns))*0.1
	if state.Uncertainty < 0 {
		state.Uncertainty = 0
	}
	if state.Uncertainty > 1.0 {
		state.Uncertainty = 1.0
	}

	// Clarity: ratio of fresh findings
	if len(findings) > 0 {
		freshCount := 0
		for _, f := range findings {
			fileChanged := false
			if f.Subject != nil && f.SubjectGitHash != nil {
				fileChanged = checkFileChanged(*f.Subject, *f.SubjectGitHash)
			}
			if f.GetStalenessStatus(fileChanged) == models.StatusFresh {
				freshCount++
			}
		}
		state.Clarity = float64(freshCount) / float64(len(findings))
	} else {
		state.Clarity = 0.5 // neutral when no findings
	}

	// Coherence: 1.0 - (dead_ends / total_breadcrumbs)
	totalBreadcrumbs := len(findings) + len(openUnknowns) + len(resolvedUnknowns) + len(deadEnds)
	if totalBreadcrumbs > 0 {
		state.Coherence = 1.0 - (float64(len(deadEnds)) / float64(totalBreadcrumbs))
	} else {
		state.Coherence = 1.0 // perfect coherence when nothing logged
	}

	// Completion: resolved / total unknowns
	totalUnknowns := len(openUnknowns) + len(resolvedUnknowns)
	if totalUnknowns > 0 {
		state.Completion = float64(len(resolvedUnknowns)) / float64(totalUnknowns)
	} else {
		state.Completion = 0.5 // neutral when no unknowns
	}

	// Engagement: decay based on session activity (2-hour half-life)
	hoursSinceStart := time.Since(sessionStart).Hours()
	lambda := math.Log(2) / 2.0 // 2-hour half-life
	state.Engagement = math.Exp(-lambda * hoursSinceStart)
	if state.Engagement < 0.1 {
		state.Engagement = 0.1 // minimum engagement
	}

	// Overall Confidence Score
	state.Confidence = (state.Know * 0.30) +
		(state.Clarity * 0.20) +
		(state.Coherence * 0.20) +
		(state.Completion * 0.15) +
		(state.Engagement * 0.15) -
		(state.Uncertainty * 0.15)
	if state.Confidence < 0 {
		state.Confidence = 0
	}
	if state.Confidence > 1.0 {
		state.Confidence = 1.0
	}

	// Derived states
	state.PassesEngagementGate = state.Engagement >= 0.60
	state.ReadyToProceed = state.Know >= 0.50 && state.Uncertainty <= 0.50
	state.NeedsInvestigation = state.Know < 0.50 || state.Uncertainty > 0.50

	// Recommended action
	if !state.PassesEngagementGate {
		state.RecommendedAction = "stop"
	} else if state.Coherence < 0.50 {
		state.RecommendedAction = "reset"
	} else if state.Clarity < 0.40 {
		state.RecommendedAction = "verify"
	} else if state.NeedsInvestigation {
		state.RecommendedAction = "investigate"
	} else {
		state.RecommendedAction = "proceed"
	}

	// Moon phase visualization
	state.MoonPhase = getMoonPhase(state.Confidence)

	return state
}

// getMoonPhase returns moon emoji for confidence level
func getMoonPhase(confidence float64) string {
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

// formatVectorBar creates a visual bar for a vector value
func formatVectorBar(value float64) string {
	filled := int(value * 10)
	empty := 10 - filled
	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}

// startCmd starts a new session with bootstrap context
var startCmd = &cobra.Command{
	Use:   "start [objective]",
	Short: "Start a new session",
	Long: `Start a new memory session and receive AI-optimized context from previous sessions.

The objective describes what you're working on. Memory will return:
- Decision guidance: whether to proceed, investigate, or verify
- Stale findings requiring verification (with commands)
- Dead ends to avoid (with reasons why they failed)
- Fresh knowledge you can rely on
- Open questions from previous sessions
- Handoff context from last session

Example:
  memory start "Implement user authentication"
  memory start "Fix bug in payment flow"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		objective := args[0]
		aiID, _ := cmd.Flags().GetString("ai-id")
		if aiID == "" {
			aiID = "claude-code"
		}

		// Get or create project
		project, err := getOrCreateDefaultProject()
		if err != nil {
			return fmt.Errorf("failed to get project: %w", err)
		}

		// Create new session
		session := models.NewSession(aiID)
		session.ProjectID = &project.ID
		session.Subject = &objective

		sessionRepo := db.NewSessionRepository(database)
		if err := sessionRepo.Create(session); err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}

		// Save as active session
		active := &ActiveSession{
			SessionID: session.SessionID,
			AIID:      aiID,
			Objective: objective,
			StartedAt: time.Now(),
			ProjectID: project.ID,
		}
		if err := saveActiveSession(active); err != nil {
			return fmt.Errorf("failed to save active session: %w", err)
		}

		// Build AI-first session context
		ctx := buildSessionContext(session.SessionID, project.ID, objective, aiID, active.StartedAt)

		if outputText {
			// Human-readable output
			fmt.Printf("Session started: %s\n", objective)
			fmt.Printf("ID: %s\n", session.SessionID)
			fmt.Println(strings.Repeat("─", 50))

			// Decision guidance
			if ctx.Decision != nil {
				fmt.Printf("\n%s %s (%.0f%% confidence)\n",
					ctx.Decision.ConfidencePhase,
					strings.ToUpper(ctx.Decision.Action),
					ctx.Decision.Confidence*100)
				fmt.Printf("  %s\n", ctx.Decision.Reason)

				if len(ctx.Decision.Prerequisites) > 0 {
					fmt.Println("\n  Before proceeding:")
					for _, p := range ctx.Decision.Prerequisites {
						fmt.Printf("    → %s\n", p)
					}
				}
			}

			// Verification needed
			if len(ctx.RequiresVerification) > 0 {
				fmt.Printf("\n⚠ VERIFY BEFORE USING (%d):\n", len(ctx.RequiresVerification))
				for _, v := range ctx.RequiresVerification {
					extra := ""
					if v.FileChanged {
						extra = " [file changed]"
					}
					fmt.Printf("  • %s (%dd old%s)\n", v.Finding, v.DaysStale, extra)
					fmt.Printf("    %s\n", v.VerifyCommand)
				}
			}

			// Dead ends
			if len(ctx.DeadEnds) > 0 {
				fmt.Printf("\n✗ DO NOT REPEAT (%d):\n", len(ctx.DeadEnds))
				for _, d := range ctx.DeadEnds {
					fmt.Printf("  • %s\n", d.Approach)
					fmt.Printf("    Why: %s\n", d.WhyFailed)
				}
			}

			// Knowledge
			if len(ctx.Knowledge) > 0 {
				fmt.Printf("\n✓ KNOWN (%d):\n", len(ctx.Knowledge))
				for _, k := range ctx.Knowledge {
					status := "✓"
					if k.Status == "aging" {
						status = "○"
					}
					fmt.Printf("  %s %s\n", status, k.Finding)
				}
			}

			// Open questions
			if len(ctx.OpenQuestions) > 0 {
				fmt.Printf("\n? OPEN QUESTIONS (%d):\n", len(ctx.OpenQuestions))
				for _, q := range ctx.OpenQuestions {
					fmt.Printf("  • %s\n", q)
				}
			}

			// Continuity
			if ctx.Continuity != nil {
				fmt.Println("\n─ Last Session ─")
				if ctx.Continuity.TimeSinceLastSession != "" {
					fmt.Printf("  %s\n", ctx.Continuity.TimeSinceLastSession)
				}
				if ctx.Continuity.Summary != "" {
					fmt.Printf("  %s\n", ctx.Continuity.Summary)
				}
				if ctx.Continuity.Recommendations != "" {
					fmt.Printf("  Recommendations: %s\n", ctx.Continuity.Recommendations)
				}
			}
		} else {
			// JSON output (default for LLMs)
			response := &models.StartResponse{
				Status:  "started",
				Context: ctx,
			}
			outputResult(response)
		}
		return nil
	},
}

// buildSessionContext creates an AI-first session context with all information
// needed for successful task completion
func buildSessionContext(sessionID, projectID, objective, aiID string, sessionStart time.Time) *models.SessionContext {
	ctx := &models.SessionContext{
		SessionID: sessionID,
		ProjectID: projectID,
		Objective: objective,
	}

	bcRepo := db.NewBreadcrumbRepository(database)

	// Get all relevant data
	findings, _ := bcRepo.ListFindingsWithStaleness(projectID, "", 20)
	resolved := false
	openUnknowns, _ := bcRepo.ListUnknowns(projectID, "", &resolved, 10)
	resolvedFlag := true
	resolvedUnknowns, _ := bcRepo.ListUnknowns(projectID, "", &resolvedFlag, 10)
	deadEnds, _ := bcRepo.ListDeadEnds(projectID, "", 10)

	// Calculate epistemic state
	epistemic := calculateEpistemicState(findings, openUnknowns, resolvedUnknowns, deadEnds, sessionStart)

	// Build epistemic snapshot
	ctx.Vectors = &models.EpistemicSnapshot{
		Know:        epistemic.Know,
		Uncertainty: epistemic.Uncertainty,
		Clarity:     epistemic.Clarity,
		Coherence:   epistemic.Coherence,
		Completion:  epistemic.Completion,
		Engagement:  epistemic.Engagement,
		Overall:     epistemic.Confidence,
	}

	// Build decision guidance - the most important part for AI
	ctx.Decision = buildDecisionGuidance(epistemic, findings, openUnknowns, deadEnds)

	// Categorize findings by staleness
	for _, f := range findings {
		fileChanged := false
		scope := ""
		if f.Subject != nil {
			scope = *f.Subject
			if f.SubjectGitHash != nil {
				fileChanged = checkFileChanged(*f.Subject, *f.SubjectGitHash)
			}
		}

		status := f.GetStalenessStatus(fileChanged)
		confidence := f.CalculateConfidence()
		daysStale := int(f.DaysSinceVerified())

		switch status {
		case models.StatusStale:
			// Stale findings need verification
			verifyCmd := fmt.Sprintf("memory verify \"%s\"", truncateText(f.Finding, 30))
			if len(f.ID) >= 8 {
				verifyCmd = fmt.Sprintf("memory verify --id %s", f.ID[:8])
			}

			ctx.RequiresVerification = append(ctx.RequiresVerification, models.VerificationNeeded{
				Finding:       f.Finding,
				ID:            f.ID,
				DaysStale:     daysStale,
				Confidence:    confidence,
				FileChanged:   fileChanged,
				Scope:         scope,
				VerifyCommand: verifyCmd,
			})

		case models.StatusFresh, models.StatusAging:
			// Fresh and aging findings go to knowledge
			statusStr := "fresh"
			if status == models.StatusAging {
				statusStr = "aging"
			}
			ctx.Knowledge = append(ctx.Knowledge, models.KnowledgeItem{
				Finding:    f.Finding,
				Confidence: confidence,
				Status:     statusStr,
				Scope:      scope,
			})
		}
	}

	// Add dead ends as warnings
	for _, d := range deadEnds {
		scope := ""
		if d.Subject != nil {
			scope = *d.Subject
		}
		ctx.DeadEnds = append(ctx.DeadEnds, models.DeadEndWarning{
			Approach:  d.Approach,
			WhyFailed: d.WhyFailed,
			Scope:     scope,
		})
	}

	// Add open questions
	for _, u := range openUnknowns {
		ctx.OpenQuestions = append(ctx.OpenQuestions, u.Unknown)
	}

	// Build continuity context from last handoff (project-scoped)
	handoffRepo := db.NewHandoffRepository(database)
	handoffs, _ := handoffRepo.List(projectID, aiID, 1)
	if len(handoffs) > 0 {
		h := handoffs[0]
		continuity := &models.ContinuityContext{}
		hasContent := false

		if h.TaskSummary != nil && *h.TaskSummary != "" {
			continuity.Summary = *h.TaskSummary
			hasContent = true
		}
		if h.NextSessionContext != nil && *h.NextSessionContext != "" {
			continuity.Recommendations = *h.NextSessionContext
			hasContent = true
		}
		if h.KeyFindings != nil && *h.KeyFindings != "" {
			// Parse key findings (stored as JSON array string)
			var highlights []string
			if err := json.Unmarshal([]byte(*h.KeyFindings), &highlights); err == nil && len(highlights) > 0 {
				// Only include top 3 highlights
				if len(highlights) > 3 {
					highlights = highlights[:3]
				}
				continuity.Highlights = highlights
				hasContent = true
			}
		}

		// Calculate time since last session
		if h.CreatedAt > 0 {
			lastTime := time.Unix(int64(h.CreatedAt), 0)
			duration := time.Since(lastTime)
			if duration.Hours() < 1 {
				continuity.TimeSinceLastSession = fmt.Sprintf("%d minutes ago", int(duration.Minutes()))
			} else if duration.Hours() < 24 {
				continuity.TimeSinceLastSession = fmt.Sprintf("%.1f hours ago", duration.Hours())
			} else {
				continuity.TimeSinceLastSession = fmt.Sprintf("%.1f days ago", duration.Hours()/24)
			}
			hasContent = true
		}

		if hasContent {
			ctx.Continuity = continuity
		}
	}

	return ctx
}

// buildDecisionGuidance creates the decision support section
func buildDecisionGuidance(
	epistemic *EpistemicState,
	findings []*models.Finding,
	openUnknowns []*models.Unknown,
	deadEnds []*models.DeadEnd,
) *models.DecisionGuidance {
	guidance := &models.DecisionGuidance{
		ReadyToProceed:  epistemic.ReadyToProceed,
		Action:          epistemic.RecommendedAction,
		ConfidencePhase: epistemic.MoonPhase,
		Confidence:      epistemic.Confidence,
	}

	// Count stale findings
	staleCount := 0
	for _, f := range findings {
		fileChanged := false
		if f.Subject != nil && f.SubjectGitHash != nil {
			fileChanged = checkFileChanged(*f.Subject, *f.SubjectGitHash)
		}
		if f.GetStalenessStatus(fileChanged) == models.StatusStale {
			staleCount++
		}
	}

	// Build reason and prerequisites based on state
	var prerequisites []string

	if epistemic.ReadyToProceed {
		guidance.Reason = "Knowledge is fresh and uncertainty is manageable. Safe to proceed with the task."
	} else {
		switch epistemic.RecommendedAction {
		case "investigate":
			guidance.Reason = "Uncertainty is high or knowledge is low. Gather more information before acting."
			if len(openUnknowns) > 0 {
				prerequisites = append(prerequisites, fmt.Sprintf("Resolve %d open question(s)", len(openUnknowns)))
			}
			if epistemic.Know < 0.50 {
				prerequisites = append(prerequisites, "Log discoveries with `memory learned`")
			}

		case "verify":
			guidance.Reason = fmt.Sprintf("%d finding(s) may be outdated. Verify before relying on them.", staleCount)
			prerequisites = append(prerequisites, "Verify stale findings with `memory verify`")

		case "reset":
			guidance.Reason = "Too many failed approaches have reduced coherence. Consider a fresh approach."
			if len(deadEnds) > 0 {
				prerequisites = append(prerequisites, fmt.Sprintf("Review %d dead end(s) to avoid repeating mistakes", len(deadEnds)))
			}

		case "stop":
			guidance.Reason = "Session engagement is too low. Consider taking a break or starting fresh."

		default:
			guidance.Reason = "Proceed with caution."
		}
	}

	guidance.Prerequisites = prerequisites
	return guidance
}

// truncateText truncates text to maxLen and adds ellipsis
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}

// buildBootstrapContext is deprecated, use buildSessionContext instead
// Kept for backward compatibility
func buildBootstrapContext(projectID, aiID string, sessionStart time.Time) map[string]interface{} {
	context := map[string]interface{}{}

	bcRepo := db.NewBreadcrumbRepository(database)

	// Get recent findings with staleness data
	findings, _ := bcRepo.ListFindingsWithStaleness(projectID, "", 20)

	// Get open unknowns
	resolved := false
	unknowns, _ := bcRepo.ListUnknowns(projectID, "", &resolved, 10)

	// Get resolved unknowns for epistemic calculation
	resolvedFlag := true
	resolvedUnknowns, _ := bcRepo.ListUnknowns(projectID, "", &resolvedFlag, 10)

	// Get dead ends to avoid
	deadEnds, _ := bcRepo.ListDeadEnds(projectID, "", 5)

	// Calculate epistemic state from historical project data
	epistemic := calculateEpistemicState(findings, unknowns, resolvedUnknowns, deadEnds, sessionStart)
	context["epistemic_state"] = epistemic

	// Process findings
	if len(findings) > 0 {
		// Separate stale vs fresh findings
		var staleFindings []map[string]interface{}
		var freshFindings []string

		for _, f := range findings {
			fileChanged := false
			if f.Subject != nil && f.SubjectGitHash != nil {
				fileChanged = checkFileChanged(*f.Subject, *f.SubjectGitHash)
			}
			status := f.GetStalenessStatus(fileChanged)

			if status == models.StatusStale {
				staleFindings = append(staleFindings, map[string]interface{}{
					"text":         f.Finding,
					"confidence":   f.CalculateConfidence(),
					"days_old":     int(f.DaysSinceVerified()),
					"file_changed": fileChanged,
				})
			} else {
				freshFindings = append(freshFindings, f.Finding)
			}
		}

		if len(staleFindings) > 0 {
			context["stale_findings"] = staleFindings
			context["stale_warning"] = fmt.Sprintf("%d finding(s) need verification before use", len(staleFindings))
		}
		if len(freshFindings) > 0 {
			context["findings"] = freshFindings
		}
	}

	// Add open unknowns
	if len(unknowns) > 0 {
		unknownStrs := make([]string, 0, len(unknowns))
		for _, u := range unknowns {
			unknownStrs = append(unknownStrs, u.Unknown)
		}
		context["open_unknowns"] = unknownStrs
	}

	// Add dead ends
	if len(deadEnds) > 0 {
		deadEndStrs := make([]string, 0, len(deadEnds))
		for _, d := range deadEnds {
			deadEndStrs = append(deadEndStrs, fmt.Sprintf("%s (%s)", d.Approach, d.WhyFailed))
		}
		context["dead_ends_to_avoid"] = deadEndStrs
	}

	// Get last session handoff (project-scoped)
	handoffRepo := db.NewHandoffRepository(database)
	handoffs, _ := handoffRepo.List(projectID, aiID, 1)
	if len(handoffs) > 0 {
		h := handoffs[0]
		lastSession := map[string]interface{}{}
		if h.TaskSummary != nil {
			lastSession["summary"] = *h.TaskSummary
		}
		if h.NextSessionContext != nil {
			lastSession["recommendations"] = *h.NextSessionContext
		}
		if len(lastSession) > 0 {
			context["last_session"] = lastSession
		}
	}

	return context
}

// doneCmd ends the current session
var doneCmd = &cobra.Command{
	Use:   "done [summary]",
	Short: "End the current session",
	Long: `End the current session with a summary of what was accomplished.

This will:
- Calculate epistemic state and show delta from baseline
- Create a handoff for future sessions
- Store remaining unknowns for next time

Example:
  memory done "Implemented JWT authentication with refresh tokens"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		summary := args[0]

		active, err := requireActiveSession()
		if err != nil {
			return err
		}

		// Calculate session stats
		bcRepo := db.NewBreadcrumbRepository(database)
		findings, _ := bcRepo.ListFindingsWithStaleness(active.ProjectID, active.SessionID, 100)
		resolved := true
		resolvedUnknowns, _ := bcRepo.ListUnknowns(active.ProjectID, active.SessionID, &resolved, 100)
		unresolved := false
		openUnknowns, _ := bcRepo.ListUnknowns(active.ProjectID, active.SessionID, &unresolved, 100)
		deadEnds, _ := bcRepo.ListDeadEnds(active.ProjectID, active.SessionID, 100)

		// Calculate full epistemic state
		epistemic := calculateEpistemicState(findings, openUnknowns, resolvedUnknowns, deadEnds, active.StartedAt)

		// Create handoff (project-scoped)
		handoffRepo := db.NewHandoffRepository(database)
		handoffInput := &models.HandoffCreateInput{
			SessionID:   active.SessionID,
			ProjectID:   active.ProjectID,
			TaskSummary: summary,
		}

		// Collect key findings
		keyFindings := make([]string, 0)
		for _, f := range findings {
			keyFindings = append(keyFindings, f.Finding)
		}
		handoffInput.KeyFindings = keyFindings

		// Collect remaining unknowns
		remainingUnknowns := make([]string, 0)
		for _, u := range openUnknowns {
			remainingUnknowns = append(remainingUnknowns, u.Unknown)
		}
		handoffInput.RemainingUnknowns = remainingUnknowns

		handoffRepo.Create(handoffInput, active.AIID)

		// End session
		sessionRepo := db.NewSessionRepository(database)
		sessionRepo.End(active.SessionID)

		// Clear active session
		clearActiveSession()

		duration := time.Since(active.StartedAt)

		if !outputText {
			result := map[string]interface{}{
				"status":          "completed",
				"objective":       active.Objective,
				"summary":         summary,
				"duration":        duration.String(),
				"epistemic_state": epistemic,
				"stats": map[string]interface{}{
					"findings":          len(findings),
					"unknowns_resolved": len(resolvedUnknowns),
					"unknowns_open":     len(openUnknowns),
					"dead_ends":         len(deadEnds),
				},
				"delta": map[string]interface{}{
					"know":        epistemic.Know - 0.5,
					"uncertainty": epistemic.Uncertainty - 0.5,
					"clarity":     epistemic.Clarity - 0.5,
				},
			}
			outputResult(result)
		} else {
			fmt.Printf("Session completed: %s\n", active.Objective)
			fmt.Println(strings.Repeat("─", 50))
			fmt.Printf("Duration: %s\n\n", duration.Round(time.Minute))

			fmt.Println("Epistemic Delta:")
			fmt.Printf("  Know:        %+.2f (0.50 → %.2f)\n", epistemic.Know-0.5, epistemic.Know)
			fmt.Printf("  Uncertainty: %+.2f (0.50 → %.2f)\n", epistemic.Uncertainty-0.5, epistemic.Uncertainty)
			fmt.Printf("  Clarity:     %+.2f (0.50 → %.2f)\n", epistemic.Clarity-0.5, epistemic.Clarity)

			// Final state
			confidenceLabel := "Critical"
			if epistemic.Confidence >= 0.75 {
				confidenceLabel = "Good"
			} else if epistemic.Confidence >= 0.50 {
				confidenceLabel = "Moderate"
			} else if epistemic.Confidence >= 0.25 {
				confidenceLabel = "Low"
			}
			fmt.Printf("\nFinal: %s %s (%.0f%% confidence)\n", epistemic.MoonPhase, confidenceLabel, epistemic.Confidence*100)

			// Stats
			fmt.Printf("\nStats: %d findings, %d resolved, %d open, %d dead ends\n",
				len(findings), len(resolvedUnknowns), len(openUnknowns), len(deadEnds))
		}
		return nil
	},
}

// learnedCmd logs a finding/discovery
var learnedCmd = &cobra.Command{
	Use:   "learned [insight]",
	Short: "Log something you learned",
	Long: `Log a finding, discovery, or insight gained during work.

Use --scope to associate the finding with a specific file for staleness tracking.

Example:
  memory learned "Auth uses JWT with 15min expiry"
  memory learned "Database connection pool is set to 10" --scope config/db.go
  memory learned "Rate limiting is handled by nginx"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		findingText := args[0]
		scope, _ := cmd.Flags().GetString("scope")

		active, err := requireActiveSession()
		if err != nil {
			return err
		}

		finding := models.NewFinding(active.ProjectID, active.SessionID, findingText, 0.5)

		// Set scope and capture git hash for staleness tracking
		if scope != "" {
			finding.Subject = &scope
			hash := getFileGitHash(scope)
			if hash != "" {
				finding.SubjectGitHash = &hash
			}
		}

		// Set initial verification timestamp to creation time
		finding.LastVerifiedTimestamp = &finding.CreatedTimestamp

		repo := db.NewBreadcrumbRepository(database)
		if err := repo.CreateFinding(finding); err != nil {
			return fmt.Errorf("failed to log finding: %w", err)
		}

		if !outputText {
			result := map[string]interface{}{
				"status":  "logged",
				"type":    "finding",
				"finding": findingText,
			}
			if scope != "" {
				result["scope"] = scope
				if finding.SubjectGitHash != nil {
					result["git_hash"] = *finding.SubjectGitHash
				}
			}
			outputResult(result)
		} else {
			fmt.Printf("✓ Learned: %s\n", findingText)
			if scope != "" {
				fmt.Printf("  (scoped to: %s)\n", scope)
			}
		}
		return nil
	},
}

// uncertainCmd logs an unknown/knowledge gap
var uncertainCmd = &cobra.Command{
	Use:   "uncertain [question]",
	Short: "Log something you're uncertain about",
	Long: `Log a question, knowledge gap, or area of uncertainty.

Example:
  memory uncertain "How does token refresh work?"
  memory uncertain "What's the rate limiting strategy?"
  memory uncertain "Where is the config stored?"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		unknownText := args[0]
		scope, _ := cmd.Flags().GetString("scope")

		active, err := requireActiveSession()
		if err != nil {
			return err
		}

		unknown := models.NewUnknown(active.ProjectID, active.SessionID, unknownText, 0.5)
		if scope != "" {
			unknown.Subject = &scope
		}

		repo := db.NewBreadcrumbRepository(database)
		if err := repo.CreateUnknown(unknown); err != nil {
			return fmt.Errorf("failed to log unknown: %w", err)
		}

		if !outputText {
			outputResult(map[string]interface{}{
				"status":  "logged",
				"type":    "unknown",
				"unknown": unknownText,
			})
		} else {
			fmt.Printf("? Uncertain: %s\n", unknownText)
		}
		return nil
	},
}

// triedCmd logs a failed approach
var triedCmd = &cobra.Command{
	Use:   "tried [approach] [why-failed]",
	Short: "Log a failed approach",
	Long: `Log an approach that was tried but didn't work, to avoid repeating it.

Example:
  memory tried "passport-local" "Too complex for our needs"
  memory tried "localStorage for tokens" "XSS vulnerability"
  memory tried "sync file writes" "Blocking the event loop"`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		approach := args[0]
		whyFailed := args[1]

		active, err := requireActiveSession()
		if err != nil {
			return err
		}

		deadEnd := models.NewDeadEnd(active.ProjectID, active.SessionID, approach, whyFailed, 0.5)

		repo := db.NewBreadcrumbRepository(database)
		if err := repo.CreateDeadEnd(deadEnd); err != nil {
			return fmt.Errorf("failed to log dead end: %w", err)
		}

		if !outputText {
			outputResult(map[string]interface{}{
				"status":     "logged",
				"type":       "dead_end",
				"approach":   approach,
				"why_failed": whyFailed,
			})
		} else {
			fmt.Printf("✗ Tried: %s → %s\n", approach, whyFailed)
		}
		return nil
	},
}

// statusCmd shows current session status
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current session status",
	Long:  `Show the current session status with AI-optimized context including decision guidance, knowledge state, and progress.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		active, err := loadActiveSession()
		if err != nil {
			if !outputText {
				response := &models.StatusResponse{
					Status:  "no_session",
					Message: "No active session. Run 'memory start \"objective\"' to begin.",
				}
				outputResult(response)
			} else {
				fmt.Println("No active session. Run 'memory start \"objective\"' to begin.")
			}
			return nil
		}

		duration := time.Since(active.StartedAt)

		// Build the same context structure as start for consistency
		ctx := buildSessionContext(active.SessionID, active.ProjectID, active.Objective, active.AIID, active.StartedAt)

		// Calculate counts from context
		counts := &models.BreadcrumbCounts{
			DeadEnds:     len(ctx.DeadEnds),
			UnknownsOpen: len(ctx.OpenQuestions),
		}

		// Count findings by status
		for _, k := range ctx.Knowledge {
			counts.Findings++
			if k.Status == "fresh" {
				counts.FindingsFresh++
			} else if k.Status == "aging" {
				counts.FindingsAging++
			}
		}
		counts.FindingsStale = len(ctx.RequiresVerification)
		counts.Findings += counts.FindingsStale

		if !outputText {
			response := &models.StatusResponse{
				Status:   "active",
				Duration: duration.Round(time.Second).String(),
				Counts:   counts,
				Context:  ctx,
			}
			outputResult(response)
		} else {
			fmt.Printf("Session: %s (%s)\n", active.Objective, duration.Round(time.Minute))
			fmt.Println(strings.Repeat("─", 50))

			// Decision guidance
			if ctx.Decision != nil {
				fmt.Printf("\n%s %s (%.0f%% confidence)\n",
					ctx.Decision.ConfidencePhase,
					strings.ToUpper(ctx.Decision.Action),
					ctx.Decision.Confidence*100)
				fmt.Printf("  %s\n", ctx.Decision.Reason)

				if len(ctx.Decision.Prerequisites) > 0 {
					fmt.Println("\n  Before proceeding:")
					for _, p := range ctx.Decision.Prerequisites {
						fmt.Printf("    → %s\n", p)
					}
				}
			}

			// Vectors
			if ctx.Vectors != nil {
				fmt.Println("\nVectors:")
				fmt.Printf("  Know:        %s %.0f%%\n", formatVectorBar(ctx.Vectors.Know), ctx.Vectors.Know*100)
				fmt.Printf("  Uncertainty: %s %.0f%%\n", formatVectorBar(ctx.Vectors.Uncertainty), ctx.Vectors.Uncertainty*100)
				fmt.Printf("  Clarity:     %s %.0f%%\n", formatVectorBar(ctx.Vectors.Clarity), ctx.Vectors.Clarity*100)
				fmt.Printf("  Coherence:   %s %.0f%%\n", formatVectorBar(ctx.Vectors.Coherence), ctx.Vectors.Coherence*100)
				fmt.Printf("  Completion:  %s %.0f%%\n", formatVectorBar(ctx.Vectors.Completion), ctx.Vectors.Completion*100)
				fmt.Printf("  Engagement:  %s %.0f%%\n", formatVectorBar(ctx.Vectors.Engagement), ctx.Vectors.Engagement*100)
			}

			// Verification needed
			if len(ctx.RequiresVerification) > 0 {
				fmt.Printf("\n⚠ VERIFY BEFORE USING (%d):\n", len(ctx.RequiresVerification))
				for _, v := range ctx.RequiresVerification {
					extra := ""
					if v.FileChanged {
						extra = " [file changed]"
					}
					fmt.Printf("  • %s (%dd old%s)\n", v.Finding, v.DaysStale, extra)
					fmt.Printf("    %s\n", v.VerifyCommand)
				}
			}

			// Dead ends
			if len(ctx.DeadEnds) > 0 {
				fmt.Printf("\n✗ DO NOT REPEAT (%d):\n", len(ctx.DeadEnds))
				for _, d := range ctx.DeadEnds {
					fmt.Printf("  • %s\n", d.Approach)
					fmt.Printf("    Why: %s\n", d.WhyFailed)
				}
			}

			// Knowledge
			if len(ctx.Knowledge) > 0 {
				fmt.Printf("\n✓ KNOWN (%d):\n", len(ctx.Knowledge))
				for _, k := range ctx.Knowledge {
					status := "✓"
					if k.Status == "aging" {
						status = "○"
					}
					fmt.Printf("  %s %s\n", status, k.Finding)
				}
			}

			// Open questions
			if len(ctx.OpenQuestions) > 0 {
				fmt.Printf("\n? OPEN QUESTIONS (%d):\n", len(ctx.OpenQuestions))
				for _, q := range ctx.OpenQuestions {
					fmt.Printf("  • %s\n", q)
				}
			}

			// Summary counts
			fmt.Printf("\nSession: %d findings, %d open questions, %d dead ends\n",
				counts.Findings, counts.UnknownsOpen, counts.DeadEnds)
		}
		return nil
	},
}

// verifyCmd verifies/refreshes a stale finding
var verifyCmd = &cobra.Command{
	Use:   "verify [search-text]",
	Short: "Verify a stale finding",
	Long: `Verify a finding to refresh its confidence timestamp.

Use this when you've confirmed a finding is still accurate.

Examples:
  memory verify "JWT"                    # Find and verify findings containing "JWT"
  memory verify --id abc123              # Verify by ID
  memory verify "old text" --update "new text"  # Update the finding text`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		findingID, _ := cmd.Flags().GetString("id")
		updateText, _ := cmd.Flags().GetString("update")

		// Get active session for project context
		active, err := loadActiveSession()
		projectID := ""
		if err == nil && active != nil {
			projectID = active.ProjectID
		}

		repo := db.NewBreadcrumbRepository(database)

		// Find the finding either by ID or text search
		var targetFinding *models.Finding

		if findingID != "" {
			// Look up by ID
			targetFinding, err = repo.GetFinding(findingID)
			if err != nil {
				return fmt.Errorf("failed to get finding: %w", err)
			}
			if targetFinding == nil {
				return fmt.Errorf("finding not found: %s", findingID)
			}
		} else if len(args) > 0 {
			// Search by text
			searchText := args[0]
			findings, err := repo.FindFindingByText(projectID, searchText)
			if err != nil {
				return fmt.Errorf("failed to search findings: %w", err)
			}
			if len(findings) == 0 {
				return fmt.Errorf("no findings found matching: %s", searchText)
			}
			if len(findings) > 1 {
				// Show matches and ask user to be more specific
				if !outputText {
					result := map[string]interface{}{
						"status":  "multiple_matches",
						"message": "Multiple findings match. Use --id to specify.",
						"matches": make([]map[string]interface{}, 0),
					}
					for _, f := range findings {
						fileChanged := false
						if f.Subject != nil && f.SubjectGitHash != nil {
							fileChanged = checkFileChanged(*f.Subject, *f.SubjectGitHash)
						}
						result["matches"] = append(result["matches"].([]map[string]interface{}), map[string]interface{}{
							"id":           f.ID,
							"finding":      f.Finding,
							"status":       string(f.GetStalenessStatus(fileChanged)),
							"days_old":     int(f.DaysSinceVerified()),
							"file_changed": fileChanged,
						})
					}
					outputResult(result)
				} else {
					fmt.Println("Multiple matches found. Use --id to specify:")
					for _, f := range findings {
						fileChanged := false
						if f.Subject != nil && f.SubjectGitHash != nil {
							fileChanged = checkFileChanged(*f.Subject, *f.SubjectGitHash)
						}
						status := f.GetStalenessStatus(fileChanged)
						statusIcon := "✓"
						if status == models.StatusAging {
							statusIcon = "○"
						} else if status == models.StatusStale {
							statusIcon = "⚠"
						}
						fmt.Printf("  %s %s (id: %s)\n", statusIcon, f.Finding, f.ID[:8])
					}
				}
				return nil
			}
			targetFinding = findings[0]
		} else {
			return fmt.Errorf("provide search text or --id flag")
		}

		// Calculate new git hash if finding has a subject file
		var newGitHash *string
		if targetFinding.Subject != nil {
			hash := getFileGitHash(*targetFinding.Subject)
			if hash != "" {
				newGitHash = &hash
			}
		}

		// Update text if provided
		var newText *string
		if updateText != "" {
			newText = &updateText
		}

		// Verify the finding
		if err := repo.VerifyFinding(targetFinding.ID, newGitHash, newText); err != nil {
			return fmt.Errorf("failed to verify finding: %w", err)
		}

		displayText := targetFinding.Finding
		if newText != nil {
			displayText = *newText
		}

		if !outputText {
			outputResult(map[string]interface{}{
				"status":   "verified",
				"id":       targetFinding.ID,
				"finding":  displayText,
				"updated":  newText != nil,
				"git_hash": newGitHash,
			})
		} else {
			fmt.Printf("✓ Verified: %s\n", displayText)
			if newText != nil {
				fmt.Printf("  (updated from: %s)\n", targetFinding.Finding)
			}
		}

		return nil
	},
}

// queryCmd allows querying learnings without starting a session
var queryCmd = &cobra.Command{
	Use:   "query [search]",
	Short: "Query learnings without starting a session",
	Long: `Query the knowledge base to see what has been learned across all sessions.

This command does NOT require an active session. Use it to:
- View all findings (learnings)
- View all unknowns (open questions)
- View all dead ends (failed approaches)
- Search for specific topics with fuzzy matching

Examples:
  memory query                    # Show all learnings
  memory query "auth"             # Search for findings containing "auth"
  memory query "authn jwt" -f     # Fuzzy search across all types
  memory query --unknowns         # Show open questions
  memory query --dead-ends        # Show failed approaches
  memory query --all              # Show everything`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		showUnknowns, _ := cmd.Flags().GetBool("unknowns")
		showDeadEnds, _ := cmd.Flags().GetBool("dead-ends")
		showAll, _ := cmd.Flags().GetBool("all")
		fuzzySearch, _ := cmd.Flags().GetBool("fuzzy")
		limit, _ := cmd.Flags().GetInt("limit")
		threshold, _ := cmd.Flags().GetFloat64("threshold")

		searchText := ""
		if len(args) > 0 {
			searchText = args[0]
		}

		// Get project (but don't require active session)
		project, err := getOrCreateDefaultProject()
		if err != nil {
			return fmt.Errorf("failed to get project: %w", err)
		}

		bcRepo := db.NewBreadcrumbRepository(database)

		// Determine what to show
		showFindings := !showUnknowns && !showDeadEnds || showAll
		showUnknownsFlag := showUnknowns || showAll
		showDeadEndsFlag := showDeadEnds || showAll

		// If fuzzy search is enabled, search across all types and return unified results
		if fuzzySearch && searchText != "" {
			return runFuzzyQuery(bcRepo, project.ID, searchText, showFindings, showUnknownsFlag, showDeadEndsFlag, limit, threshold)
		}

		// For JSON output, build structured response
		if !outputText {
			result := map[string]interface{}{
				"project_id": project.ID,
			}

			if showFindings {
				var findings []*models.Finding
				if searchText != "" {
					findings, _ = bcRepo.FindFindingByText(project.ID, searchText)
				} else {
					findings, _ = bcRepo.ListFindingsWithStaleness(project.ID, "", limit)
				}

				findingsList := make([]map[string]interface{}, 0)
				for _, f := range findings {
					fileChanged := false
					if f.Subject != nil && f.SubjectGitHash != nil {
						fileChanged = checkFileChanged(*f.Subject, *f.SubjectGitHash)
					}
					item := map[string]interface{}{
						"id":         f.ID,
						"finding":    f.Finding,
						"status":     string(f.GetStalenessStatus(fileChanged)),
						"confidence": f.CalculateConfidence(),
						"days_old":   int(f.DaysSinceVerified()),
					}
					if f.Subject != nil {
						item["scope"] = *f.Subject
						item["file_changed"] = fileChanged
					}
					findingsList = append(findingsList, item)
				}
				result["findings"] = findingsList
				result["findings_count"] = len(findingsList)
			}

			if showUnknownsFlag {
				resolved := false
				unknowns, _ := bcRepo.ListUnknowns(project.ID, "", &resolved, limit)
				unknownsList := make([]map[string]interface{}, 0)
				for _, u := range unknowns {
					item := map[string]interface{}{
						"id":      u.ID,
						"unknown": u.Unknown,
					}
					if u.Subject != nil {
						item["scope"] = *u.Subject
					}
					unknownsList = append(unknownsList, item)
				}
				result["unknowns"] = unknownsList
				result["unknowns_count"] = len(unknownsList)
			}

			if showDeadEndsFlag {
				deadEnds, _ := bcRepo.ListDeadEnds(project.ID, "", limit)
				deadEndsList := make([]map[string]interface{}, 0)
				for _, d := range deadEnds {
					item := map[string]interface{}{
						"id":         d.ID,
						"approach":   d.Approach,
						"why_failed": d.WhyFailed,
					}
					if d.Subject != nil {
						item["scope"] = *d.Subject
					}
					deadEndsList = append(deadEndsList, item)
				}
				result["dead_ends"] = deadEndsList
				result["dead_ends_count"] = len(deadEndsList)
			}

			outputResult(result)
			return nil
		}

		// Human-readable output
		fmt.Printf("Knowledge Base: %s\n", project.Name)
		fmt.Println(strings.Repeat("─", 50))

		if showFindings {
			var findings []*models.Finding
			if searchText != "" {
				findings, _ = bcRepo.FindFindingByText(project.ID, searchText)
				fmt.Printf("\n✓ FINDINGS matching \"%s\" (%d):\n", searchText, len(findings))
			} else {
				findings, _ = bcRepo.ListFindingsWithStaleness(project.ID, "", limit)
				fmt.Printf("\n✓ FINDINGS (%d):\n", len(findings))
			}

			if len(findings) == 0 {
				fmt.Println("  (none)")
			} else {
				for _, f := range findings {
					fileChanged := false
					if f.Subject != nil && f.SubjectGitHash != nil {
						fileChanged = checkFileChanged(*f.Subject, *f.SubjectGitHash)
					}
					status := f.GetStalenessStatus(fileChanged)
					days := int(f.DaysSinceVerified())

					statusIcon := "✓"
					extra := ""
					if status == models.StatusAging {
						statusIcon = "○"
						extra = fmt.Sprintf(" [%dd]", days)
					} else if status == models.StatusStale {
						statusIcon = "⚠"
						extra = fmt.Sprintf(" [stale: %dd]", days)
						if fileChanged {
							extra += " [file changed]"
						}
					}

					fmt.Printf("  %s %s%s\n", statusIcon, f.Finding, extra)
					if f.Subject != nil {
						fmt.Printf("    scope: %s\n", *f.Subject)
					}
				}
			}
		}

		if showUnknownsFlag {
			resolved := false
			unknowns, _ := bcRepo.ListUnknowns(project.ID, "", &resolved, limit)
			fmt.Printf("\n? OPEN QUESTIONS (%d):\n", len(unknowns))

			if len(unknowns) == 0 {
				fmt.Println("  (none)")
			} else {
				for _, u := range unknowns {
					fmt.Printf("  • %s\n", u.Unknown)
					if u.Subject != nil {
						fmt.Printf("    scope: %s\n", *u.Subject)
					}
				}
			}
		}

		if showDeadEndsFlag {
			deadEnds, _ := bcRepo.ListDeadEnds(project.ID, "", limit)
			fmt.Printf("\n✗ DEAD ENDS (%d):\n", len(deadEnds))

			if len(deadEnds) == 0 {
				fmt.Println("  (none)")
			} else {
				for _, d := range deadEnds {
					fmt.Printf("  • %s\n", d.Approach)
					fmt.Printf("    Why: %s\n", d.WhyFailed)
					if d.Subject != nil {
						fmt.Printf("    scope: %s\n", *d.Subject)
					}
				}
			}
		}

		return nil
	},
}

// runFuzzyQuery performs fuzzy search across all breadcrumb types
func runFuzzyQuery(bcRepo *db.BreadcrumbRepository, projectID, query string, showFindings, showUnknowns, showDeadEnds bool, limit int, threshold float64) error {
	// Collect all items into search items
	var items []search.SearchItem

	// Load findings
	if showFindings {
		findings, _ := bcRepo.ListFindingsWithStaleness(projectID, "", 500)
		for _, f := range findings {
			scope := ""
			if f.Subject != nil {
				scope = *f.Subject
			}
			items = append(items, search.SearchItem{
				ID:    f.ID,
				Type:  "finding",
				Text:  f.Finding,
				Scope: scope,
			})
		}
	}

	// Load unknowns
	if showUnknowns {
		resolved := false
		unknowns, _ := bcRepo.ListUnknowns(projectID, "", &resolved, 500)
		for _, u := range unknowns {
			scope := ""
			if u.Subject != nil {
				scope = *u.Subject
			}
			items = append(items, search.SearchItem{
				ID:    u.ID,
				Type:  "unknown",
				Text:  u.Unknown,
				Scope: scope,
			})
		}
	}

	// Load dead ends
	if showDeadEnds {
		deadEnds, _ := bcRepo.ListDeadEnds(projectID, "", 500)
		for _, d := range deadEnds {
			scope := ""
			if d.Subject != nil {
				scope = *d.Subject
			}
			items = append(items, search.SearchItem{
				ID:            d.ID,
				Type:          "dead_end",
				Text:          d.Approach,
				SecondaryText: d.WhyFailed,
				Scope:         scope,
			})
		}
	}

	// Run fuzzy search
	results := search.FuzzySearch(query, items, threshold)

	// Apply limit
	if len(results) > limit {
		results = results[:limit]
	}

	// Output results
	if !outputText {
		resultsList := make([]map[string]interface{}, 0)
		for _, r := range results {
			item := map[string]interface{}{
				"id":    r.ID,
				"type":  r.Type,
				"text":  r.Text,
				"score": r.Score,
			}
			if r.SecondaryText != "" {
				item["secondary_text"] = r.SecondaryText
			}
			if r.Scope != "" {
				item["scope"] = r.Scope
			}
			resultsList = append(resultsList, item)
		}
		outputResult(map[string]interface{}{
			"query":   query,
			"results": resultsList,
			"count":   len(resultsList),
		})
		return nil
	}

	// Human-readable output
	fmt.Printf("Fuzzy Search: \"%s\"\n", query)
	fmt.Println(strings.Repeat("─", 50))

	if len(results) == 0 {
		fmt.Println("No matches found.")
		return nil
	}

	fmt.Printf("\nFound %d match(es):\n\n", len(results))
	for _, r := range results {
		// Type indicator
		typeIcon := "✓"
		typeLabel := "FINDING"
		switch r.Type {
		case "unknown":
			typeIcon = "?"
			typeLabel = "QUESTION"
		case "dead_end":
			typeIcon = "✗"
			typeLabel = "DEAD END"
		}

		// Score indicator (stars)
		stars := int(r.Score * 5)
		if stars < 1 {
			stars = 1
		}
		scoreBar := strings.Repeat("★", stars) + strings.Repeat("☆", 5-stars)

		fmt.Printf("  %s [%s] %s\n", typeIcon, typeLabel, scoreBar)
		fmt.Printf("    %s\n", r.Text)
		if r.SecondaryText != "" {
			fmt.Printf("    Why: %s\n", r.SecondaryText)
		}
		if r.Scope != "" {
			fmt.Printf("    scope: %s\n", r.Scope)
		}
		fmt.Println()
	}

	return nil
}

// getFileGitHash returns the git blob hash for a file
// Returns empty string if not in a git repo or file doesn't exist
func getFileGitHash(filePath string) string {
	// Try to get git hash for the file
	cmd := exec.Command("git", "hash-object", filePath)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// checkFileChanged compares a stored git hash with the current file's hash
func checkFileChanged(filePath string, storedHash string) bool {
	if storedHash == "" || filePath == "" {
		return false // Can't determine change without both values
	}
	currentHash := getFileGitHash(filePath)
	if currentHash == "" {
		return false // File not in git, can't determine
	}
	return currentHash != storedHash
}

func init() {
	// start command flags
	startCmd.Flags().String("ai-id", "claude-code", "AI identifier")

	// Scope flags for logging commands
	learnedCmd.Flags().String("scope", "", "File/directory scope for the finding")
	uncertainCmd.Flags().String("scope", "", "File/directory scope for the unknown")

	// verify command flags
	verifyCmd.Flags().String("id", "", "Finding ID to verify")
	verifyCmd.Flags().String("update", "", "New text to update the finding with")

	// query command flags
	queryCmd.Flags().BoolP("unknowns", "u", false, "Show open questions/unknowns")
	queryCmd.Flags().BoolP("dead-ends", "d", false, "Show failed approaches/dead ends")
	queryCmd.Flags().BoolP("all", "a", false, "Show all (findings, unknowns, dead ends)")
	queryCmd.Flags().BoolP("fuzzy", "f", false, "Enable fuzzy search across all types")
	queryCmd.Flags().Float64P("threshold", "t", 0.3, "Minimum score threshold for fuzzy matches (0.0-1.0)")
	queryCmd.Flags().IntP("limit", "n", 50, "Maximum number of results")

	// Register core commands
	rootCmd.AddCommand(
		startCmd,
		doneCmd,
		learnedCmd,
		uncertainCmd,
		triedCmd,
		statusCmd,
		verifyCmd,
		queryCmd,
	)
}
