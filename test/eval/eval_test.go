package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEval(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MCO Agentic Docs Evaluation Suite")
}

type Scenario struct {
	Name         string
	Category     string
	PromptPath   string
	ExpectedPath string
	PatchPath    string
	HasPatch     bool
}

var (
	costTracker   *CostTracker
	evalRuns      int
	evalThreshold float64
	evalVerbose   bool
)

// Discover scenarios at init time (before specs are registered)
var scenarios = discoverScenarios("./testdata")

var _ = BeforeSuite(func() {
	// Load configuration
	evalRuns = getEnvInt("EVAL_RUNS", 3)
	evalThreshold = getEnvFloat("EVAL_THRESHOLD", 80.0)
	evalVerbose = getEnvBool("EVAL_VERBOSE", false)

	// Initialize cost tracker
	costTracker = NewCostTracker()

	// Verify we found scenarios
	Expect(scenarios).NotTo(BeEmpty(), "No test scenarios found in ./testdata")

	fmt.Printf("\n=== MCO Agentic Docs Evaluation ===\n")
	fmt.Printf("Found %d scenarios\n", len(scenarios))
	fmt.Printf("Runs per scenario: %d\n", evalRuns)
	fmt.Printf("Pass threshold: %.1f%%\n\n", evalThreshold)
})

var _ = Describe("MCO Agentic Documentation", func() {
	for _, scenario := range scenarios {
		scenario := scenario // Capture range variable

		It(fmt.Sprintf("should pass: %s/%s", scenario.Category, scenario.Name), func() {
			ctx := context.Background()
			scores := make([]float64, evalRuns)
			var lastVerdicts []Verdict

			for i := 0; i < evalRuns; i++ {
				if !evalVerbose {
					fmt.Printf("    Run %d/%d...", i+1, evalRuns)
				}

				// Load prompt
				prompt, err := os.ReadFile(scenario.PromptPath)
				Expect(err).NotTo(HaveOccurred())

				// Load expected behaviors
				expected, err := loadExpectedBehaviors(scenario.ExpectedPath)
				Expect(err).NotTo(HaveOccurred())

				// Run agent via Claude Code CLI
				if !evalVerbose {
					fmt.Printf(" agent...")
				}
				agentOutput := runAgent(ctx, string(prompt), scenario.HasPatch)

				// Verbose: show agent output length
				if evalVerbose {
					fmt.Printf("    Agent output length: %d chars\n", len(agentOutput))
					if len(agentOutput) > 0 {
						preview := agentOutput
						if len(preview) > 200 {
							preview = preview[:200] + "..."
						}
						fmt.Printf("    Preview: %s\n", preview)
					}
				}

				// Judge agent output
				if !evalVerbose {
					fmt.Printf(" judge...")
				}
				verdicts, judgeCost := judgeOutput(ctx, string(prompt), expected, agentOutput)
				costTracker.AddJudgeCost(judgeCost)

				if !evalVerbose {
					fmt.Printf(" done\n")
				}

				// Calculate score
				scores[i] = calculateScore(verdicts)
				lastVerdicts = verdicts

				if evalVerbose {
					fmt.Printf("\n  Run %d/%d: %.1f%%\n", i+1, evalRuns, scores[i])
					for _, v := range verdicts {
						status := "✅"
						if v.Verdict == "MISSED" {
							status = "❌"
						}
						fmt.Printf("    %s %s\n", status, v.Item)
					}
				}
			}

			// Calculate average score
			avgScore := average(scores)

			// Report
			if !evalVerbose {
				fmt.Printf("\n  %s/%s: %.1f%% ", scenario.Category, scenario.Name, avgScore)
				if avgScore >= evalThreshold {
					fmt.Printf("✅\n")
				} else {
					fmt.Printf("❌\n")
				}
			}

			// Show missed items if failed
			if avgScore < evalThreshold {
				fmt.Printf("\n  Missed behaviors:\n")
				for _, v := range lastVerdicts {
					if v.Verdict == "MISSED" {
						fmt.Printf("    - %s\n", v.Item)
					}
				}
				fmt.Printf("\n")
			}

			// Assert
			Expect(avgScore).To(BeNumerically(">=", evalThreshold),
				fmt.Sprintf("Average score %.1f%% below threshold %.1f%%", avgScore, evalThreshold))
		})
	}
})

var _ = AfterSuite(func() {
	costTracker.Report()
})

// Scenario discovery
func discoverScenarios(root string) []Scenario {
	var scenarios []Scenario

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Name() == "prompt.txt" {
			scenarioDir := filepath.Dir(path)
			relPath, _ := filepath.Rel(root, scenarioDir)
			parts := strings.Split(relPath, string(filepath.Separator))

			category := "unknown"
			name := filepath.Base(scenarioDir)
			if len(parts) >= 2 {
				category = parts[0]
				name = parts[1]
			} else if len(parts) == 1 {
				name = parts[0]
			}

			scenario := Scenario{
				Name:         name,
				Category:     category,
				PromptPath:   path,
				ExpectedPath: filepath.Join(scenarioDir, "expected.txt"),
				PatchPath:    filepath.Join(scenarioDir, "patch.diff"),
			}

			// Check if patch exists
			if _, err := os.Stat(scenario.PatchPath); err == nil {
				scenario.HasPatch = true
			}

			scenarios = append(scenarios, scenario)
		}

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error discovering scenarios: %v\n", err)
	}

	return scenarios
}

// Load expected behaviors from file
func loadExpectedBehaviors(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	var behaviors []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		behaviors = append(behaviors, line)
	}

	return behaviors, nil
}

// Run agent via Claude Code CLI
func runAgent(ctx context.Context, prompt string, hasPatch bool) string {
	// Set 7 minute timeout (leaves room for multiple runs within test timeout)
	ctx, cancel := context.WithTimeout(ctx, 7*time.Minute)
	defer cancel()

	// Get model from environment (optional - uses CLI default if not set)
	model := os.Getenv("EVAL_AGENT_MODEL")

	// Add self-documentation requirement to prompt
	enhancedPrompt := `You are working in the Machine Config Operator (MCO) repository.

CRITICAL REQUIREMENT: You MUST end your response with a "## Documentation Used" section listing all files you read. This is mandatory and will be verified.

` + prompt + `

===================================
MANDATORY: End your response with:

## Documentation Used

- /path/to/file1.md (reason)
- /path/to/file2.md (reason)

DO NOT SKIP THIS SECTION. It will be checked.
===================================`

	// Build claude command
	args := []string{
		"--print",
		"--output-format", "json",
		"-p", enhancedPrompt,
	}

	// Add model if specified
	if model != "" {
		args = append([]string{"--model", model}, args...)
	}

	// Restrict tools based on scenario type
	if hasPatch {
		// Code review scenarios: allow Bash for git operations
		args = append(args, "--allowed-tools", "Read,Grep,Glob,Bash")
	} else {
		// Navigation/authoring scenarios: read-only
		args = append(args, "--allowed-tools", "Read,Grep,Glob")
	}

	// Execute claude CLI
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = filepath.Join("..", "..") // Run from repo root

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Fprintf(os.Stderr, "Agent CLI timed out after 7 minutes\n")
		} else {
			fmt.Fprintf(os.Stderr, "Agent CLI failed: %v\nOutput: %s\n", err, string(output))
		}
		return ""
	}

	// Parse JSON output
	var result claudeOutput
	err = json.Unmarshal(output, &result)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse agent JSON: %v\nOutput: %s\n", err, string(output))
		return ""
	}

	// Track cost
	costTracker.AddAgentCost(result.TotalCostUSD)

	return result.Result
}

type claudeOutput struct {
	Result       string  `json:"result"`
	TotalCostUSD float64 `json:"totalCostUSD"`
}

// Calculate score from verdicts
func calculateScore(verdicts []Verdict) float64 {
	if len(verdicts) == 0 {
		return 0
	}

	covered := 0
	for _, v := range verdicts {
		if v.Verdict == "COVERED" {
			covered++
		}
	}

	return float64(covered) / float64(len(verdicts)) * 100
}

// Calculate average
func average(scores []float64) float64 {
	if len(scores) == 0 {
		return 0
	}

	sum := 0.0
	for _, s := range scores {
		sum += s
	}
	return sum / float64(len(scores))
}

// Helper functions
func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if val := os.Getenv(key); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultValue
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
