package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"github.com/nguyendkn/git-generator/internal/diff"
	"github.com/nguyendkn/git-generator/pkg/types"
)

// GeminiClient handles interactions with Google Gemini API
type GeminiClient struct {
	client      *genai.Client
	model       *genai.GenerativeModel
	config      types.GeminiConfig
	rateLimiter *RateLimiter
}

// RateLimiter implements simple rate limiting
type RateLimiter struct {
	lastRequest time.Time
	minInterval time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	interval := time.Minute / time.Duration(requestsPerMinute)
	return &RateLimiter{
		minInterval: interval,
	}
}

// Wait waits if necessary to respect rate limits
func (rl *RateLimiter) Wait() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRequest)
	if elapsed < rl.minInterval {
		time.Sleep(rl.minInterval - elapsed)
	}
	rl.lastRequest = time.Now()
}

// NewGeminiClient creates a new Gemini API client
func NewGeminiClient(config types.GeminiConfig) (*GeminiClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf(":Google Gemini API key is required")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(config.APIKey))
	if err != nil {
		return nil, fmt.Errorf(":Failed to create Gemini client: %w", err)
	}

	// Set default model if not specified
	if config.Model == "" {
		config.Model = "gemini-1.5-flash"
	}

	// Set default temperature if not specified
	if config.Temperature == 0 {
		config.Temperature = 0.3
	}

	// Set default max tokens if not specified
	if config.MaxTokens == 0 {
		config.MaxTokens = 1000
	}

	model := client.GenerativeModel(config.Model)
	model.SetTemperature(config.Temperature)
	model.SetMaxOutputTokens(int32(config.MaxTokens))

	// Configure safety settings to be more permissive for code content
	model.SafetySettings = []*genai.SafetySetting{
		{
			Category:  genai.HarmCategoryHarassment,
			Threshold: genai.HarmBlockMediumAndAbove,
		},
		{
			Category:  genai.HarmCategoryHateSpeech,
			Threshold: genai.HarmBlockMediumAndAbove,
		},
		{
			Category:  genai.HarmCategoryDangerousContent,
			Threshold: genai.HarmBlockMediumAndAbove,
		},
		{
			Category:  genai.HarmCategorySexuallyExplicit,
			Threshold: genai.HarmBlockMediumAndAbove,
		},
	}

	return &GeminiClient{
		client:      client,
		model:       model,
		config:      config,
		rateLimiter: NewRateLimiter(10), // 10 requests per minute
	}, nil
}

// Close closes the Gemini client
func (gc *GeminiClient) Close() error {
	return gc.client.Close()
}

// GenerateCommitMessage generates a commit message from processed diff data
func (gc *GeminiClient) GenerateCommitMessage(ctx context.Context, processedDiff *diff.ProcessedDiff, style string) (*types.CommitMessage, error) {
	if processedDiff == nil {
		return nil, fmt.Errorf("processed diff is nil")
	}

	// Apply rate limiting
	gc.rateLimiter.Wait()

	prompt := gc.buildPrompt(processedDiff, style)

	resp, err := gc.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no response candidates received")
	}

	candidate := resp.Candidates[0]
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response content")
	}

	responseText := ""
	for _, part := range candidate.Content.Parts {
		if textPart, ok := part.(genai.Text); ok {
			responseText += string(textPart)
		}
	}

	return gc.parseCommitMessage(responseText, style)
}

// AnalyzeChangesForVersioning analyzes changes to determine semantic version bump type
func (gc *GeminiClient) AnalyzeChangesForVersioning(ctx context.Context, processedDiff *diff.ProcessedDiff, recentCommits []*types.CommitInfo) (*types.VersionAnalysis, error) {
	if processedDiff == nil {
		return nil, fmt.Errorf("processed diff is nil")
	}

	// Apply rate limiting
	gc.rateLimiter.Wait()

	prompt := gc.buildVersionAnalysisPrompt(processedDiff, recentCommits)

	resp, err := gc.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate version analysis: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no response candidates received")
	}

	candidate := resp.Candidates[0]
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response content")
	}

	responseText := ""
	for _, part := range candidate.Content.Parts {
		if textPart, ok := part.(genai.Text); ok {
			responseText += string(textPart)
		}
	}

	return gc.parseVersionAnalysis(responseText)
}

// buildPrompt creates a prompt for the AI model
func (gc *GeminiClient) buildPrompt(processedDiff *diff.ProcessedDiff, style string) string {
	var prompt strings.Builder

	prompt.WriteString("You are an expert software developer tasked with generating a high-quality Git commit message that explains both WHAT changed and WHY the changes were made.\n\n")

	// Add enhanced instructions for contextual analysis
	prompt.WriteString("## Core Principles:\n")
	prompt.WriteString("1. Analyze the INTENT and PURPOSE behind each change, not just what was modified\n")
	prompt.WriteString("2. Explain the business or technical reasoning for the changes\n")
	prompt.WriteString("3. Consider the context of recent changes and evolution patterns\n")
	prompt.WriteString("4. Identify the impact and benefits of the modifications\n")
	prompt.WriteString("5. Provide specific examples when configuration or function changes are involved\n\n")

	// Add style-specific instructions
	switch style {
	case "conventional":
		prompt.WriteString("Generate a commit message following the Conventional Commits specification:\n")
		prompt.WriteString("Format: <type>[optional scope]: <description>\n")
		prompt.WriteString("\n<why the change was made and its purpose>\n")
		prompt.WriteString("<context from previous related changes if relevant>\n")
		prompt.WriteString("[optional footer(s)]\n\n")
		prompt.WriteString("IMPORTANT: Choose ONLY ONE type that best represents the primary change:\n")
		prompt.WriteString("Types: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert\n")
		prompt.WriteString("- Use 'feat' for new features or functionality\n")
		prompt.WriteString("- Use 'fix' for bug fixes\n")
		prompt.WriteString("- Use 'docs' for documentation changes\n")
		prompt.WriteString("- Use 'refactor' for code refactoring without changing functionality\n")
		prompt.WriteString("- Use 'test' for test-related changes\n")
		prompt.WriteString("- Use 'chore' for maintenance tasks, build changes, or tooling\n")
		prompt.WriteString("- Use 'style' for formatting, missing semicolons, etc.\n")
		prompt.WriteString("- Use 'perf' for performance improvements\n")
		prompt.WriteString("- Use 'build' for build system or external dependencies\n")
		prompt.WriteString("- Use 'ci' for CI configuration files and scripts\n\n")
		prompt.WriteString("DO NOT mix multiple types in one commit message. Choose the most appropriate single type.\n\n")
	case "simple":
		prompt.WriteString("Generate a simple, clear commit message that describes what was changed and why.\n")
		prompt.WriteString("Keep it concise but include the reasoning behind the change.\n\n")
	default:
		prompt.WriteString("Generate a detailed commit message that clearly explains the changes and their purpose.\n")
		prompt.WriteString("Include a subject line and body that covers both what changed and why.\n\n")
	}

	// Add diff summary
	prompt.WriteString(fmt.Sprintf("## Change Summary\n%s\n\n", processedDiff.Summary))

	// Add language information
	if len(processedDiff.Languages) > 0 {
		prompt.WriteString("## Languages involved:\n")
		for lang, count := range processedDiff.Languages {
			prompt.WriteString(fmt.Sprintf("- %s (%d files)\n", lang, count))
		}
		prompt.WriteString("\n")
	}

	// Add context information if available
	if processedDiff.ChangeContext != nil {
		context := processedDiff.ChangeContext

		// Add recent commit history for context
		if len(context.RecentCommits) > 0 {
			prompt.WriteString("## Recent Commit History (for context):\n")
			for i, commit := range context.RecentCommits {
				if i >= 5 { // Limit to 5 recent commits
					break
				}
				prompt.WriteString(fmt.Sprintf("- %s: %s\n", commit.Hash[:8], commit.Subject))
			}
			prompt.WriteString("\n")
		}

		// Add configuration changes analysis
		if len(context.ConfigChanges) > 0 {
			prompt.WriteString("## Configuration Changes Detected:\n")
			for _, change := range context.ConfigChanges {
				prompt.WriteString(fmt.Sprintf("- %s in %s: %v\n", change.Parameter, change.File, change.NewValue))
				if change.Context != "" {
					prompt.WriteString(fmt.Sprintf("  Context: %s\n", change.Context))
				}
			}
			prompt.WriteString("IMPORTANT: Explain WHY these configuration values were changed and their impact.\n\n")
		}

		// Add function changes analysis
		if len(context.FunctionChanges) > 0 {
			prompt.WriteString("## Function Changes Detected:\n")
			for _, change := range context.FunctionChanges {
				prompt.WriteString(fmt.Sprintf("- Function '%s' in %s: %s\n", change.FunctionName, change.File, change.ChangeType))
				if change.Impact != "" {
					prompt.WriteString(fmt.Sprintf("  Impact: %s\n", change.Impact))
				}
			}
			prompt.WriteString("IMPORTANT: Explain the purpose and impact of these function changes.\n\n")
		}

		// Add performance hints
		if len(context.PerformanceHints) > 0 {
			prompt.WriteString("## Performance-Related Changes:\n")
			for _, hint := range context.PerformanceHints {
				prompt.WriteString(fmt.Sprintf("- %s\n", hint))
			}
			prompt.WriteString("IMPORTANT: Explain the performance benefits or optimizations introduced.\n\n")
		}

		// Add change patterns
		if len(context.ChangePatterns) > 0 {
			prompt.WriteString("## Change Patterns Detected:\n")
			if refactoring, ok := context.ChangePatterns["likely_refactoring"].(bool); ok && refactoring {
				prompt.WriteString("- This appears to be a refactoring effort\n")
			}
			if newFeature, ok := context.ChangePatterns["likely_new_feature"].(bool); ok && newFeature {
				prompt.WriteString("- This appears to be a new feature implementation\n")
			}
			if docUpdate, ok := context.ChangePatterns["documentation_update"].(bool); ok && docUpdate {
				prompt.WriteString("- Documentation updates detected\n")
			}
			prompt.WriteString("\n")
		}
	}

	// Add chunk information (limited to avoid token limits)
	if len(processedDiff.Chunks) > 0 {
		prompt.WriteString("## File Changes:\n")
		for i, chunk := range processedDiff.Chunks {
			if i >= 3 { // Limit to first 3 chunks to avoid token limits
				prompt.WriteString(fmt.Sprintf("... and %d more chunks\n", len(processedDiff.Chunks)-i))
				break
			}
			prompt.WriteString(fmt.Sprintf("Chunk %d: %s\n", i+1, chunk.Description))
		}
		prompt.WriteString("\n")
	}

	prompt.WriteString("## Enhanced Instructions:\n")
	prompt.WriteString("1. Analyze the changes and determine the primary PURPOSE and INTENT\n")
	prompt.WriteString("2. Choose the most appropriate commit type based on the actual impact\n")
	prompt.WriteString("3. Write a clear, concise description that explains WHAT changed\n")
	prompt.WriteString("4. In the body, explain WHY the changes were made and their purpose\n")
	prompt.WriteString("5. Include context from recent changes if relevant to understanding the evolution\n")
	prompt.WriteString("6. For configuration changes: explain the reasoning behind new values vs old values\n")
	prompt.WriteString("7. For function changes: explain the purpose and impact of modifications\n")
	prompt.WriteString("8. For performance changes: explain the expected benefits or optimizations\n")
	prompt.WriteString("9. Keep the subject line under 50 characters\n")
	prompt.WriteString("10. Use imperative mood (e.g., 'Add feature' not 'Added feature')\n\n")

	prompt.WriteString("## Expected Format:\n")
	prompt.WriteString("<type>: <what changed>\n\n")
	prompt.WriteString("<why the change was made and its purpose>\n")
	prompt.WriteString("<context from previous related changes if relevant>\n\n")

	prompt.WriteString("Generate only the commit message following this format, no additional text or explanations.")

	return prompt.String()
}

// parseCommitMessage parses the AI response into a structured commit message
func (gc *GeminiClient) parseCommitMessage(response, style string) (*types.CommitMessage, error) {
	response = strings.TrimSpace(response)
	if response == "" {
		return nil, fmt.Errorf("empty response from AI")
	}

	lines := strings.Split(response, "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("invalid response format")
	}

	commitMsg := &types.CommitMessage{}

	// Parse the first line (subject)
	subject := strings.TrimSpace(lines[0])

	if style == "conventional" {
		// Parse conventional commit format
		if err := gc.parseConventionalCommit(subject, commitMsg); err != nil {
			// If parsing fails, try to extract type manually or fallback gracefully
			if strings.Contains(subject, ":") {
				// Try to extract type from malformed conventional commit
				parts := strings.SplitN(subject, ":", 2)
				if len(parts) == 2 {
					typeStr := strings.TrimSpace(parts[0])
					// Remove scope if present
					if idx := strings.Index(typeStr, "("); idx != -1 {
						typeStr = typeStr[:idx]
					}
					// Remove breaking change indicator
					typeStr = strings.TrimSuffix(typeStr, "!")

					commitMsg.Type = types.CommitType(typeStr)
					commitMsg.Description = strings.TrimSpace(parts[1])
				} else {
					commitMsg.Type = types.CommitTypeChore
					commitMsg.Description = subject
				}
			} else {
				// No colon found, treat as simple description
				commitMsg.Type = types.CommitTypeChore
				commitMsg.Description = subject
			}
		}
	} else {
		// Simple format
		commitMsg.Type = types.CommitTypeChore
		commitMsg.Description = subject
	}

	// Parse body and footer if present
	if len(lines) > 1 {
		var bodyLines []string
		var footerLines []string
		inFooter := false

		for i := 1; i < len(lines); i++ {
			line := strings.TrimSpace(lines[i])
			if line == "" {
				continue
			}

			// Check if this looks like a footer (contains a colon)
			if strings.Contains(line, ":") && (strings.HasPrefix(line, "BREAKING CHANGE") ||
				strings.HasPrefix(line, "Closes") || strings.HasPrefix(line, "Fixes")) {
				inFooter = true
			}

			if inFooter {
				footerLines = append(footerLines, line)
			} else {
				bodyLines = append(bodyLines, line)
			}
		}

		if len(bodyLines) > 0 {
			commitMsg.Body = strings.Join(bodyLines, "\n")
		}

		if len(footerLines) > 0 {
			commitMsg.Footer = strings.Join(footerLines, "\n")
			// Check for breaking changes
			if strings.Contains(commitMsg.Footer, "BREAKING CHANGE") {
				commitMsg.Breaking = true
			}
		}
	}

	return commitMsg, nil
}

// parseConventionalCommit parses a conventional commit subject line
func (gc *GeminiClient) parseConventionalCommit(subject string, commitMsg *types.CommitMessage) error {
	// Pattern: type(scope): description or type!: description
	parts := strings.SplitN(subject, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid conventional commit format")
	}

	typeAndScope := strings.TrimSpace(parts[0])
	description := strings.TrimSpace(parts[1])

	// Check for breaking change indicator
	if strings.HasSuffix(typeAndScope, "!") {
		commitMsg.Breaking = true
		typeAndScope = strings.TrimSuffix(typeAndScope, "!")
	}

	// Parse type and scope
	if strings.Contains(typeAndScope, "(") && strings.Contains(typeAndScope, ")") {
		// Has scope
		typeParts := strings.SplitN(typeAndScope, "(", 2)
		commitType := strings.TrimSpace(typeParts[0])
		scope := strings.TrimSpace(strings.TrimSuffix(typeParts[1], ")"))

		commitMsg.Type = types.CommitType(commitType)
		commitMsg.Scope = scope
	} else {
		// No scope
		commitMsg.Type = types.CommitType(typeAndScope)
	}

	// Validate that description doesn't contain another type
	description = gc.cleanDescription(description)
	commitMsg.Description = description
	return nil
}

// cleanDescription removes any commit type prefixes from the description
func (gc *GeminiClient) cleanDescription(description string) string {
	// List of commit types that might appear in description
	commitTypes := []string{"feat:", "fix:", "docs:", "style:", "refactor:", "perf:", "test:", "build:", "ci:", "chore:", "revert:"}

	// Remove any commit type prefix from description
	for _, commitType := range commitTypes {
		if strings.HasPrefix(strings.ToLower(description), commitType) {
			description = strings.TrimSpace(description[len(commitType):])
			break
		}
	}

	return description
}

// buildVersionAnalysisPrompt creates a prompt for version analysis
func (gc *GeminiClient) buildVersionAnalysisPrompt(processedDiff *diff.ProcessedDiff, recentCommits []*types.CommitInfo) string {
	var prompt strings.Builder

	prompt.WriteString("You are an expert software developer tasked with analyzing code changes to determine the appropriate semantic version bump (MAJOR, MINOR, or PATCH) according to semantic versioning principles.\n\n")

	prompt.WriteString("SEMANTIC VERSIONING RULES:\n")
	prompt.WriteString("- MAJOR: Breaking changes, API changes, incompatible changes\n")
	prompt.WriteString("- MINOR: New features, backwards-compatible functionality additions\n")
	prompt.WriteString("- PATCH: Bug fixes, documentation updates, minor improvements\n\n")

	prompt.WriteString("ANALYSIS CRITERIA:\n")
	prompt.WriteString("1. Breaking Changes: API modifications, removed functions, changed signatures\n")
	prompt.WriteString("2. New Features: Added functions, new capabilities, feature additions\n")
	prompt.WriteString("3. Bug Fixes: Error corrections, performance improvements, minor fixes\n")
	prompt.WriteString("4. Documentation: README updates, comments, documentation changes\n")
	prompt.WriteString("5. Dependencies: Package updates, dependency changes\n\n")

	// Add recent commits context
	if len(recentCommits) > 0 {
		prompt.WriteString("RECENT COMMIT HISTORY (for context):\n")
		for i, commit := range recentCommits {
			if i >= 5 { // Limit to 5 recent commits
				break
			}
			prompt.WriteString(fmt.Sprintf("- %s: %s\n", commit.Hash[:8], commit.Subject))
		}
		prompt.WriteString("\n")
	}

	// Add current changes
	prompt.WriteString("CURRENT CHANGES TO ANALYZE:\n")
	prompt.WriteString(fmt.Sprintf("Files changed: %d\n", processedDiff.TotalFiles))
	prompt.WriteString(fmt.Sprintf("Lines added: %d\n", processedDiff.TotalAdded))
	prompt.WriteString(fmt.Sprintf("Lines deleted: %d\n", processedDiff.TotalDeleted))

	if len(processedDiff.Languages) > 0 {
		prompt.WriteString("Languages: ")
		var langs []string
		for lang := range processedDiff.Languages {
			langs = append(langs, lang)
		}
		prompt.WriteString(strings.Join(langs, ", "))
		prompt.WriteString("\n")
	}

	prompt.WriteString("\nFILE CHANGES:\n")
	for _, chunk := range processedDiff.Chunks {
		for _, file := range chunk.Files {
			prompt.WriteString(fmt.Sprintf("- %s (%s): +%d -%d lines\n",
				file.Path, file.ChangeType, file.LinesAdded, file.LinesDeleted))

			// Include a sample of the diff content for analysis
			if len(file.Content) > 0 {
				lines := strings.Split(file.Content, "\n")
				maxLines := 20 // Limit diff content to avoid token limits
				if len(lines) > maxLines {
					lines = lines[:maxLines]
				}
				prompt.WriteString("  Sample changes:\n")
				for _, line := range lines {
					if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
						prompt.WriteString(fmt.Sprintf("    %s\n", line))
					}
				}
			}
		}
	}

	prompt.WriteString("\nRESPONSE FORMAT:\n")
	prompt.WriteString("Analyze the changes and respond with a JSON object containing:\n")
	prompt.WriteString("{\n")
	prompt.WriteString(`  "recommended_bump": "major|minor|patch",` + "\n")
	prompt.WriteString(`  "confidence": 0.95,` + "\n")
	prompt.WriteString(`  "reasoning": "Detailed explanation of why this version bump is recommended",` + "\n")
	prompt.WriteString(`  "breaking_changes": ["list of breaking changes if any"],` + "\n")
	prompt.WriteString(`  "new_features": ["list of new features if any"],` + "\n")
	prompt.WriteString(`  "bug_fixes": ["list of bug fixes if any"],` + "\n")
	prompt.WriteString(`  "documentation": ["list of documentation changes if any"],` + "\n")
	prompt.WriteString(`  "dependencies": ["list of dependency changes if any"]` + "\n")
	prompt.WriteString("}\n\n")

	prompt.WriteString("Focus on the actual impact of the changes on users and API compatibility. Be conservative with MAJOR bumps - only recommend them for true breaking changes.")

	return prompt.String()
}

// parseVersionAnalysis parses the AI response for version analysis
func (gc *GeminiClient) parseVersionAnalysis(responseText string) (*types.VersionAnalysis, error) {
	// Clean up the response text
	responseText = strings.TrimSpace(responseText)

	// Remove markdown code blocks if present
	if strings.HasPrefix(responseText, "```json") {
		responseText = strings.TrimPrefix(responseText, "```json")
		responseText = strings.TrimSuffix(responseText, "```")
	} else if strings.HasPrefix(responseText, "```") {
		responseText = strings.TrimPrefix(responseText, "```")
		responseText = strings.TrimSuffix(responseText, "```")
	}

	responseText = strings.TrimSpace(responseText)

	// Try to extract JSON from the response
	jsonStart := strings.Index(responseText, "{")
	jsonEnd := strings.LastIndex(responseText, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonStart >= jsonEnd {
		return nil, fmt.Errorf("no valid JSON found in response")
	}

	jsonStr := responseText[jsonStart : jsonEnd+1]

	// Parse the JSON response
	var rawAnalysis struct {
		RecommendedBump string   `json:"recommended_bump"`
		Confidence      float64  `json:"confidence"`
		Reasoning       string   `json:"reasoning"`
		BreakingChanges []string `json:"breaking_changes"`
		NewFeatures     []string `json:"new_features"`
		BugFixes        []string `json:"bug_fixes"`
		Documentation   []string `json:"documentation"`
		Dependencies    []string `json:"dependencies"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &rawAnalysis); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Convert to our types
	var bumpType types.VersionBumpType
	switch strings.ToLower(rawAnalysis.RecommendedBump) {
	case "major":
		bumpType = types.VersionBumpMajor
	case "minor":
		bumpType = types.VersionBumpMinor
	case "patch":
		bumpType = types.VersionBumpPatch
	default:
		return nil, fmt.Errorf("invalid bump type: %s", rawAnalysis.RecommendedBump)
	}

	// Ensure confidence is within valid range
	confidence := rawAnalysis.Confidence
	if confidence < 0.0 {
		confidence = 0.0
	} else if confidence > 1.0 {
		confidence = 1.0
	}

	analysis := &types.VersionAnalysis{
		RecommendedBump: bumpType,
		Confidence:      confidence,
		Reasoning:       rawAnalysis.Reasoning,
		BreakingChanges: rawAnalysis.BreakingChanges,
		NewFeatures:     rawAnalysis.NewFeatures,
		BugFixes:        rawAnalysis.BugFixes,
		Documentation:   rawAnalysis.Documentation,
		Dependencies:    rawAnalysis.Dependencies,
	}

	return analysis, nil
}
