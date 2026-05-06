package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type Verdict struct {
	Item      string `json:"item"`
	Verdict   string `json:"verdict"` // "COVERED" or "MISSED"
	Reasoning string `json:"reasoning"`
}

type JudgeResponse struct {
	Verdicts []Verdict `json:"verdicts"`
}

type CostTracker struct {
	AgentCosts []float64
	JudgeCosts []float64
}

func NewCostTracker() *CostTracker {
	return &CostTracker{
		AgentCosts: []float64{},
		JudgeCosts: []float64{},
	}
}

func (ct *CostTracker) AddAgentCost(cost float64) {
	ct.AgentCosts = append(ct.AgentCosts, cost)
}

func (ct *CostTracker) AddJudgeCost(cost float64) {
	ct.JudgeCosts = append(ct.JudgeCosts, cost)
}

func (ct *CostTracker) Report() {
	totalAgent := sum(ct.AgentCosts)
	totalJudge := sum(ct.JudgeCosts)
	total := totalAgent + totalJudge

	fmt.Printf("\n=== Cost Report ===\n")
	fmt.Printf("Agent calls: $%.4f (%d calls)\n", totalAgent, len(ct.AgentCosts))
	fmt.Printf("Judge calls: $%.4f (%d calls)\n", totalJudge, len(ct.JudgeCosts))
	fmt.Printf("Total:       $%.4f\n\n", total)
}

// Judge agent output against expected behaviors
func judgeOutput(ctx context.Context, originalPrompt string, expectedBehaviors []string, agentOutput string) ([]Verdict, float64) {
	// Build judge prompt
	judgePrompt := buildJudgePrompt(originalPrompt, expectedBehaviors, agentOutput)

	// Call judge LLM via Claude CLI
	judgeResponseJSON, cost := callJudgeLLM(ctx, judgePrompt)

	// Parse response
	var response JudgeResponse
	if err := json.Unmarshal([]byte(judgeResponseJSON), &response); err != nil {
		// Fallback: mark all as MISSED if parsing fails
		fmt.Fprintf(os.Stderr, "Judge response parse error: %v\nResponse: %s\n", err, judgeResponseJSON)
		verdicts := make([]Verdict, len(expectedBehaviors))
		for i, behavior := range expectedBehaviors {
			verdicts[i] = Verdict{
				Item:      behavior,
				Verdict:   "MISSED",
				Reasoning: "Judge response parsing failed",
			}
		}
		return verdicts, cost
	}

	// Validate we got verdicts for all expected behaviors
	if len(response.Verdicts) != len(expectedBehaviors) {
		fmt.Fprintf(os.Stderr, "Warning: Judge returned %d verdicts but expected %d\n",
			len(response.Verdicts), len(expectedBehaviors))
	}

	return response.Verdicts, cost
}

func buildJudgePrompt(originalPrompt string, expectedBehaviors []string, agentOutput string) string {
	behaviorsText := ""
	for i, behavior := range expectedBehaviors {
		behaviorsText += fmt.Sprintf("%d. %s\n", i+1, behavior)
	}

	// Extract Documentation Used section if present
	docUsed := extractDocumentationUsed(agentOutput)
	docContext := ""
	if len(docUsed) > 0 {
		docContext = fmt.Sprintf("\n\nDOCUMENTATION LISTED BY AGENT:\n%s\n", strings.Join(docUsed, "\n"))
	}

	return fmt.Sprintf(`You are a judge evaluating an AI agent's response.

TASK: %s

EXPECTED BEHAVIORS:
%s
%s
AGENT OUTPUT:
%s

Evaluate each expected behavior. Respond with ONLY a JSON object (no markdown, no explanation):

{
  "verdicts": [
    {"item": "behavior text", "verdict": "COVERED", "reasoning": "why"},
    {"item": "behavior text", "verdict": "MISSED", "reasoning": "why"}
  ]
}

Rules:
- COVERED = agent demonstrated this behavior (explicit evidence)
- MISSED = agent did not demonstrate this behavior
- For "Lists X in Documentation Used": check if X appears in the DOCUMENTATION LISTED BY AGENT section
- For "Includes Documentation Used section": check if agent has "## Documentation Used" heading
- Return ONLY the JSON object, nothing else`, originalPrompt, behaviorsText, docContext, agentOutput)
}

// Extract files from Documentation Used section
func extractDocumentationUsed(output string) []string {
	var files []string

	// Look for "## Documentation Used" section
	lines := strings.Split(output, "\n")
	inDocSection := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Start of documentation section
		if strings.Contains(strings.ToLower(line), "## documentation used") ||
		   strings.Contains(strings.ToLower(line), "## files read") {
			inDocSection = true
			continue
		}

		// End of section (next heading)
		if inDocSection && strings.HasPrefix(line, "##") {
			break
		}

		// Extract file paths (lines starting with - or *)
		if inDocSection && (strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*")) {
			// Remove bullet and extract path
			line = strings.TrimPrefix(line, "-")
			line = strings.TrimPrefix(line, "*")
			line = strings.TrimSpace(line)

			// Extract just the file path (before any parenthetical reason)
			if idx := strings.Index(line, "("); idx > 0 {
				line = strings.TrimSpace(line[:idx])
			}

			if line != "" {
				files = append(files, "- "+line)
			}
		}
	}

	return files
}

func callJudgeLLM(ctx context.Context, prompt string) (string, float64) {
	// Set 10 minute timeout
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// Get judge model from environment (optional - uses CLI default if not set)
	judgeModel := os.Getenv("EVAL_JUDGE_MODEL")

	// Build claude command for judge
	args := []string{
		"--print",
		"--output-format", "json",
		"-p", prompt,
	}

	// Add model if specified
	if judgeModel != "" {
		args = append([]string{"--model", judgeModel}, args...)
	}

	// Execute claude CLI
	cmd := exec.CommandContext(ctx, "claude", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Judge CLI failed: %v\nOutput: %s\n", err, string(output))
		return "{}", 0
	}

	// Parse JSON output
	var result struct {
		Result       string  `json:"result"`
		TotalCostUSD float64 `json:"totalCostUSD"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse judge JSON: %v\nOutput: %s\n", err, string(output))
		return "{}", 0
	}

	// Strip markdown code blocks if present
	cleanedResult := stripMarkdownCodeBlock(result.Result)

	// Also try to extract JSON from mixed content
	cleanedResult = extractJSON(cleanedResult)

	return cleanedResult, result.TotalCostUSD
}

// Strip markdown code blocks from judge output
func stripMarkdownCodeBlock(s string) string {
	// Remove ```json ... ``` or ``` ... ``` blocks
	re := regexp.MustCompile("(?s)```(?:json)?\\s*(.+?)\\s*```")
	matches := re.FindStringSubmatch(s)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return strings.TrimSpace(s)
}

// Extract JSON object from mixed content
func extractJSON(s string) string {
	// Look for {..."verdicts"...}
	start := strings.Index(s, "{")
	if start == -1 {
		return s
	}

	// Find matching closing brace
	depth := 0
	for i := start; i < len(s); i++ {
		if s[i] == '{' {
			depth++
		} else if s[i] == '}' {
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}

	return s
}

func sum(values []float64) float64 {
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total
}
