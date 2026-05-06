# MCO Agentic Docs Evaluation Framework

Tests whether AI agents can successfully use `./ai-docs/` and `AGENTS.md` to complete Machine Config Operator (MCO) development tasks.

## Quick Start

```bash
cd test/eval
EVAL_RUNS=1 go test -v
```

## Configuration

Environment variables:

```bash
EVAL_RUNS=3              # Runs per scenario (default: 3)
EVAL_THRESHOLD=80        # Pass threshold % (default: 80)
EVAL_VERBOSE=1           # Detailed output (default: 0)
EVAL_AGENT_MODEL=sonnet  # Agent model (default: uses CLI default)
EVAL_JUDGE_MODEL=haiku   # Judge model (default: uses CLI default)
```

## Test Scenarios

### Navigation Tests

Test if agents can discover MCO-specific documentation:
- `navigation/finding-kubelet-docs` - Discover KubeletConfig documentation
- `navigation/finding-architecture-docs` - Find MCO architecture documentation

### Component Usage Tests

Test if agents can apply MCO domain knowledge:
- `component-usage/custom-machineconfig` - Create custom MachineConfig with kernel parameters

## How It Works

1. **Agent** receives a prompt and attempts the task using docs in `./ai-docs/`
2. **Judge** (LLM) evaluates if agent found expected patterns/docs
3. **Score** calculated: % of expected behaviors agent demonstrated
4. **Pass** if score ≥ 80% (configurable via `EVAL_THRESHOLD`)

## Adding New Scenarios

1. Create directory: `test/eval/testdata/<category>/<scenario-name>/`
2. Write `prompt.txt` (the task given to agent)
3. Write `expected.txt` (behaviors agent should demonstrate)
4. Run: `EVAL_RUNS=1 go test -v -ginkgo.focus="<scenario-name>"`

## Expected Behaviors Format

List specific, verifiable behaviors (one per line):

```
Includes Documentation Used section at end of response
Lists AGENTS.md or CLAUDE.md in Documentation Used
References at least one file from ./ai-docs/domain/
Mentions MachineConfig API or resource
Describes machineConfigSelector for targeting
```
