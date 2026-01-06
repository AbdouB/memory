# Memory

Epistemic self-awareness framework for AI agents. Track what you know, don't know, and have tried across sessions.

## Why Memory?

AI agents often:
- Forget what they learned in previous sessions
- Repeat failed approaches
- Act on stale information
- Don't know what they don't know

Memory solves this by providing a simple CLI that tracks findings, uncertainties, and dead ends with automatic staleness detection.

## Installation

### From Source

```bash
git clone https://github.com/AbdouB/memory
cd memory
go build -o memory ./cmd/memory

# Or install to PATH
go install ./cmd/memory
```

### Using Go Install

```bash
go install github.com/AbdouB/memory/cmd/memory@latest
```

## Quick Start

```bash
# Start a session
memory start "Implement user authentication"

# Log what you discover
memory learned "Auth uses JWT with 15min expiry"
memory learned "Tokens stored in httpOnly cookies" --scope src/auth.js

# Log uncertainties
memory uncertain "How does token refresh work?"

# Log failed approaches
memory tried "localStorage for tokens" "XSS vulnerability"

# Check your epistemic state
memory status

# End session with summary
memory done "Implemented JWT auth with secure cookie storage"
```

## Commands

### Core Workflow

| Command | Description |
|---------|-------------|
| `start [objective]` | Start a new session with context from previous sessions |
| `learned [insight]` | Log a finding or discovery |
| `uncertain [question]` | Log a knowledge gap or question |
| `tried [approach] [why-failed]` | Log a failed approach to avoid repeating |
| `status` | Show current session status and epistemic state |
| `done [summary]` | End session and create handoff for next session |
| `verify [text]` | Verify/refresh a stale finding |
| `query [search]` | Query knowledge base (no session required) |

### Command Details

**start** - Begins a session and returns context:
```bash
memory start "Fix payment bug"
# Returns: decision guidance, stale findings, dead ends, fresh knowledge, open questions
```

**learned** - Log discoveries with optional file scope:
```bash
memory learned "API rate limit is 100 req/min"
memory learned "Config in /etc/app.conf" --scope config/settings.go
```

**verify** - Refresh stale findings:
```bash
memory verify "JWT"                      # Search and verify
memory verify --id abc123                # Verify by ID
memory verify "old" --update "new text"  # Update finding text
```

**query** - Search knowledge without a session:
```bash
memory query                     # Show all findings
memory query "auth"              # Search findings
memory query "jwt tokens" -f     # Fuzzy search all types
memory query --unknowns          # Show open questions
memory query --dead-ends         # Show failed approaches
memory query --all               # Show everything
```

## Epistemic Vectors

Memory automatically calculates your epistemic state:

| Vector | Description | Target |
|--------|-------------|--------|
| `know` | Domain knowledge level | Higher is better |
| `uncertainty` | Knowledge gaps | Lower is better |
| `clarity` | Information freshness | Higher is better |
| `coherence` | Logical consistency | Higher is better |
| `completion` | Resolved vs open unknowns | Higher is better |
| `engagement` | Session activity | Decays over time |

### Confidence Phases

```
🌑 < 25%  - Critical: Stop and investigate
🌒 25-50% - Low: Needs more information
🌓 50-75% - Moderate: Proceed with caution
🌔 75-90% - Good: Safe to proceed
🌕 >= 90% - Excellent: High confidence
```

### Decision Guidance

When you start a session, Memory tells you what to do:

- **proceed** - Knowledge is fresh, safe to continue
- **investigate** - Uncertainty is high, gather more info
- **verify** - Stale findings detected, verify before using
- **reset** - Too many dead ends, consider fresh approach

## Staleness Detection

Findings decay over time (14-day half-life):

- **Fresh** (>=70% confidence) - Safe to use
- **Aging** (40-70%) - Verify if critical
- **Stale** (<40%) - Listed in `requires_verification`

File-scoped findings also become stale when the file changes (detected via git hash).

## Output Formats

Default output is JSON (optimized for LLM consumption):
```bash
memory start "task"   # Returns JSON
```

Use `--text` for human-readable output:
```bash
memory status --text
```

## Configuration for AI Agents

Add to your AI's system prompt (e.g., `~/.claude/CLAUDE.md`):

```markdown
# Memory - Epistemic Self-Awareness

Use Memory to track your knowledge state across sessions.

## Required Workflow

| When... | Then... |
|---------|---------|
| Starting a new task | `memory start "task description"` |
| You discover something | `memory learned "what you found"` |
| You don't know something | `memory uncertain "your question"` |
| An approach fails | `memory tried "approach" "why it failed"` |
| You want to check progress | `memory status` |
| You complete the task | `memory done "summary of work"` |
| Start shows stale findings | `memory verify "finding text"` |

## Reading the Context

When you run `memory start`, check the `decision` field:
- `ready_to_proceed: true` → Safe to continue
- `action: "verify"` → Verify stale findings first
- `action: "investigate"` → Gather more information
- `dead_ends` → DO NOT repeat these approaches
- `knowledge` → Fresh findings you can rely on
```

## Data Storage

- **Database**: `~/.memory/sessions.db` (SQLite)
- **Active session**: `~/.memory/active-session.json`
- **Project-local**: `.memory/` directory if present

## Example Session

```bash
$ memory start "Add rate limiting to API"
{
  "status": "started",
  "context": {
    "decision": {
      "ready_to_proceed": false,
      "action": "verify",
      "reason": "2 finding(s) may be outdated. Verify before relying on them."
    },
    "requires_verification": [
      {"finding": "Rate limit is 100/min", "days_stale": 25}
    ],
    "dead_ends": [
      {"approach": "Redis rate limiter", "why_failed": "Too complex for MVP"}
    ],
    "knowledge": [
      {"finding": "Using Express.js middleware", "status": "fresh"}
    ]
  }
}

$ memory verify "Rate limit"
✓ Verified: Rate limit is 100/min

$ memory learned "Changed rate limit to 200/min" --scope config/limits.js

$ memory done "Increased rate limit to 200/min, added per-user tracking"
```

## License

MIT
