# Memory

Epistemic self-awareness framework for AI agents - Go implementation.

## Scientific Foundation

Memory is built on established research in cognitive science, epistemology, and AI safety:

### Metacognition & Self-Monitoring
- **Metacognitive Monitoring**: Based on Nelson & Narens (1990) metacognition framework - the ability to monitor and control one's own cognitive processes
- **Calibration**: Inspired by research on epistemic calibration (Lichtenstein et al., 1982) - matching confidence to actual accuracy
- **Dunning-Kruger Mitigation**: Explicit uncertainty tracking helps AI avoid overconfidence in unfamiliar domains

### Epistemic Vectors
The 13-dimensional vector space draws from:
- **Bayesian Epistemology**: Tracking degrees of belief and updating based on evidence
- **Signal Detection Theory**: Distinguishing signal from noise in information processing
- **Coherentism**: Measuring logical consistency across beliefs (BonJour, 1985)

### CASCADE Workflow
The Preflight-Check-Postflight workflow is grounded in:
- **Reflective Practice** (Schön, 1983): Reflection-in-action and reflection-on-action
- **Deliberate Practice** (Ericsson, 1993): Structured cycles of performance and feedback
- **Learning Loops**: Single and double-loop learning (Argyris & Schön, 1978)

### Knowledge Logging (Breadcrumbs)
- **Episodic Memory**: Logging findings creates retrievable episodic traces
- **Error-Driven Learning**: Dead-end and mistake tracking enables learning from failures
- **Spaced Repetition Principles**: Important discoveries are reinforced through structured recall

### Multi-Agent Coordination
- **Distributed Cognition** (Hutchins, 1995): Knowledge distributed across agents and artifacts
- **Transactive Memory** (Wegner, 1987): Shared understanding of "who knows what"

## Installation

### From Source

```bash
# Clone and build
git clone https://github.com/AbdouB/memory
cd memory
go build -o memory ./cmd/memory

# Install to PATH
go install ./cmd/memory
```

### Using Go Install

```bash
go install github.com/AbdouB/memory/cmd/memory@latest
```

### Pre-built Binaries

Download from the [releases page](https://github.com/AbdouB/memory/releases).

## Quick Start

```bash
# Initialize in your project
memory project-init

# Create a session
echo '{"ai_id": "claude-code"}' | memory session-create - --output

# Submit preflight assessment
cat <<EOF | memory preflight-submit - --output
{"session_id": "<ID>", "vectors": {"know": 0.5, "context": 0.6}, "reasoning": "Initial assessment"}
EOF

# Log findings during work
memory finding-log --session-id <ID> --finding "Discovered X" --impact 0.7

# Submit postflight assessment
cat <<EOF | memory postflight-submit - --output
{"session_id": "<ID>", "vectors": {"know": 0.9, "context": 0.9}, "reasoning": "What I learned"}
EOF
```

## Commands

### Session Management
- `session-create` - Create a new session
- `sessions-list` - List sessions
- `sessions-show` - Show session details
- `sessions-resume` - Resume previous session

### CASCADE Workflow
- `preflight-submit` - Submit preflight assessment (before work)
- `check-submit` - Submit check at decision points
- `postflight-submit` - Submit postflight assessment (after work)

### Goals
- `goals-create` - Create a goal
- `goals-list` - List goals
- `goals-complete` - Complete a goal
- `goals-add-subtask` - Add subtask
- `goals-complete-subtask` - Complete subtask
- `goals-progress` - Show progress

### Breadcrumbs (Knowledge Logging)
- `finding-log` - Log a discovery
- `unknown-log` - Log a knowledge gap
- `unknown-resolve` - Resolve an unknown
- `deadend-log` - Log a failed approach
- `mistake-log` - Log a mistake
- `mistake-query` - Query mistakes

### Project
- `project-init` - Initialize Memory
- `project-create` - Create project
- `project-list` - List projects
- `project-bootstrap` - Load context for AI bootstrap

### Checkpoints
- `checkpoint-create` - Create git checkpoint
- `checkpoint-list` - List checkpoints

### Handoffs
- `handoff-create` - Create handoff report
- `handoff-query` - Query handoffs

### Monitoring
- `monitor` - Show dashboard
- `check-drift` - Detect epistemic drift
- `assess-state` - Assess epistemic state

## Epistemic Vectors

Memory tracks 13 dimensions of AI knowledge state:

| Vector | Description | Healthy Range |
|--------|-------------|---------------|
| `engagement` | Task engagement level | >= 0.60 |
| `know` | Domain knowledge | >= 0.50 |
| `do` | Execution capability | >= 0.50 |
| `context` | Situational awareness | >= 0.50 |
| `clarity` | Understanding clarity | >= 0.50 |
| `coherence` | Logical consistency | >= 0.50 |
| `signal` | Signal-to-noise | >= 0.50 |
| `density` | Information density | <= 0.90 |
| `state` | Current state mapping | >= 0.50 |
| `change` | Safe change capability | >= 0.50 |
| `completion` | Task completion | 0.0-1.0 |
| `impact` | Expected impact | >= 0.50 |
| `uncertainty` | Explicit doubt | <= 0.50 |

## Configuration for AI Agents

Add to your AI's system prompt (e.g., `~/.claude/CLAUDE.md`):

```markdown
# Memory Epistemic Self-Assessment

You have Memory installed. Use it to track your epistemic state.

## Minimum Workflow

1. START: Create session before any work
   memory session-create --ai-id claude-code --output json

2. PREFLIGHT: Assess what you know
   cat <<EOF | memory preflight-submit -
   {"session_id": "<ID>", "vectors": {"know": 0.5, "context": 0.6}, "reasoning": "Initial assessment"}
   EOF

3. POSTFLIGHT: Measure what you learned
   cat <<EOF | memory postflight-submit -
   {"session_id": "<ID>", "vectors": {"know": 0.9, "context": 0.9}, "reasoning": "What I learned"}
   EOF

## Log As You Work
memory finding-log --session-id <ID> --finding "Discovered X" --impact 0.7
memory unknown-log --session-id <ID> --unknown "Need to investigate Y"
memory deadend-log --session-id <ID> --approach "Tried X" --why-failed "Because Y"
```

## Data Storage

- Project-local: `.memory/sessions.db`
- Global fallback: `~/.memory/sessions.db`
- Git checkpoints: `.git/refs/notes/memory/*`

## References

- Argyris, C., & Schön, D. A. (1978). Organizational learning: A theory of action perspective
- BonJour, L. (1985). The Structure of Empirical Knowledge
- Ericsson, K. A., Krampe, R. T., & Tesch-Römer, C. (1993). The role of deliberate practice
- Hutchins, E. (1995). Cognition in the Wild
- Lichtenstein, S., Fischhoff, B., & Phillips, L. D. (1982). Calibration of probabilities
- Nelson, T. O., & Narens, L. (1990). Metamemory: A theoretical framework
- Schön, D. A. (1983). The Reflective Practitioner
- Wegner, D. M. (1987). Transactive memory: A contemporary analysis of the group mind

## License

MIT
