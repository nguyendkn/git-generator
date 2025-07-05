package scope

import (
	"regexp"
	"sort"
	"strings"

	"github.com/nguyendkn/git-generator/pkg/types"
)

// Detector provides automatic scope detection for conventional commits
type Detector struct {
	rules []types.ScopeDetectionRule
}

// NewDetector creates a new scope detector with default rules
func NewDetector() *Detector {
	return &Detector{
		rules: getDefaultScopeRules(),
	}
}

// NewDetectorWithRules creates a new scope detector with custom rules
func NewDetectorWithRules(rules []types.ScopeDetectionRule) *Detector {
	// Sort rules by priority (higher priority first)
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})

	return &Detector{rules: rules}
}

// DetectScope analyzes file changes and suggests appropriate scope
func (d *Detector) DetectScope(diffSummary *types.DiffSummary) string {
	if len(diffSummary.Files) == 0 {
		return ""
	}

	// Collect all file paths
	var filePaths []string
	for _, file := range diffSummary.Files {
		filePaths = append(filePaths, file.Path)
	}

	// Find the best matching scope with priority consideration
	scopeCounts := make(map[string]int)
	scopePriorities := make(map[string]int)

	for _, filePath := range filePaths {
		scope, priority := d.detectScopeForFileWithPriority(filePath)
		if scope != "" {
			scopeCounts[scope]++
			// Keep track of the highest priority for each scope
			if existingPriority, exists := scopePriorities[scope]; !exists || priority > existingPriority {
				scopePriorities[scope] = priority
			}
		}
	}

	// Return the most common scope, or empty if no clear winner
	if len(scopeCounts) == 0 {
		return ""
	}

	// Find the scope with the highest count, with priority as tiebreaker
	var bestScope string
	var maxCount int
	var bestPriority int

	for scope, count := range scopeCounts {
		priority := scopePriorities[scope]
		if count > maxCount || (count == maxCount && priority > bestPriority) {
			maxCount = count
			bestScope = scope
			bestPriority = priority
		}
	}

	// Return scope based on different criteria
	totalFiles := len(filePaths)
	coverage := float64(maxCount) / float64(totalFiles)

	// For single file changes, always return the detected scope
	if totalFiles == 1 && bestScope != "" {
		return bestScope
	}

	// For multiple files, require at least 50% coverage OR if it's a high-priority scope
	if coverage >= 0.5 {
		return bestScope
	}

	// For mixed changes, check if there's a dominant high-priority scope
	if maxCount >= 2 && d.isHighPriorityScope(bestScope) {
		return bestScope
	}

	return ""
}

// DetectMultipleScopes returns all detected scopes with their confidence scores
func (d *Detector) DetectMultipleScopes(diffSummary *types.DiffSummary) map[string]float64 {
	if len(diffSummary.Files) == 0 {
		return nil
	}

	scopeCounts := make(map[string]int)
	totalFiles := len(diffSummary.Files)

	for _, file := range diffSummary.Files {
		scope := d.detectScopeForFile(file.Path)
		if scope != "" {
			scopeCounts[scope]++
		}
	}

	// Convert counts to confidence scores
	result := make(map[string]float64)
	for scope, count := range scopeCounts {
		result[scope] = float64(count) / float64(totalFiles)
	}

	return result
}

// detectScopeForFile detects scope for a single file
func (d *Detector) detectScopeForFile(filePath string) string {
	for _, rule := range d.rules {
		re, err := regexp.Compile(rule.Pattern)
		if err != nil {
			continue // Skip invalid regex patterns
		}

		matches := re.FindStringSubmatch(filePath)
		if len(matches) > 0 {
			scope := rule.Scope
			// Replace capture groups in scope (e.g., "$1" with first capture group)
			if len(matches) > 1 && strings.Contains(scope, "$1") {
				scope = strings.ReplaceAll(scope, "$1", matches[1])
			}
			return scope
		}
	}

	return ""
}

// detectScopeForFileWithPriority detects scope for a single file and returns priority
func (d *Detector) detectScopeForFileWithPriority(filePath string) (string, int) {
	for _, rule := range d.rules {
		re, err := regexp.Compile(rule.Pattern)
		if err != nil {
			continue // Skip invalid regex patterns
		}

		matches := re.FindStringSubmatch(filePath)
		if len(matches) > 0 {
			scope := rule.Scope
			// Replace capture groups in scope (e.g., "$1" with first capture group)
			if len(matches) > 1 && strings.Contains(scope, "$1") {
				scope = strings.ReplaceAll(scope, "$1", matches[1])
			}
			return scope, rule.Priority
		}
	}

	return "", 0
}

// SuggestScopeFromContent analyzes file content to suggest more specific scopes
func (d *Detector) SuggestScopeFromContent(diffSummary *types.DiffSummary) string {
	// Analyze content patterns for more specific scope detection
	contentPatterns := map[string][]string{
		"auth":     {"login", "password", "token", "authentication", "authorization", "jwt", "oauth"},
		"api":      {"endpoint", "route", "handler", "controller", "middleware"},
		"db":       {"database", "query", "migration", "schema", "table", "model"},
		"ui":       {"component", "template", "style", "css", "html", "jsx", "vue"},
		"test":     {"test", "spec", "mock", "fixture", "assert", "expect"},
		"config":   {"config", "setting", "environment", "env", "constant"},
		"security": {"security", "encrypt", "decrypt", "hash", "validate", "sanitize"},
		"perf":     {"performance", "optimize", "cache", "memory", "speed", "benchmark"},
		"docs":     {"documentation", "readme", "comment", "doc", "guide", "manual"},
		"ci":       {"pipeline", "build", "deploy", "workflow", "action", "jenkins"},
	}

	scopeScores := make(map[string]int)

	for _, file := range diffSummary.Files {
		content := strings.ToLower(file.Content)

		for scope, keywords := range contentPatterns {
			for _, keyword := range keywords {
				if strings.Contains(content, keyword) {
					scopeScores[scope]++
				}
			}
		}
	}

	// Return the scope with the highest score
	var bestScope string
	var maxScore int

	for scope, score := range scopeScores {
		if score > maxScore {
			maxScore = score
			bestScope = scope
		}
	}

	return bestScope
}

// getDefaultScopeRules returns the default set of scope detection rules
func getDefaultScopeRules() []types.ScopeDetectionRule {
	return []types.ScopeDetectionRule{
		// Go-specific patterns (high priority)
		{
			Pattern:     `^cmd/([^/]+)/`,
			Scope:       "cmd",
			Priority:    95,
			Description: "Go command applications",
		},
		{
			Pattern:     `^internal/([^/]+)/`,
			Scope:       "$1",
			Priority:    90,
			Description: "Go internal packages",
		},
		{
			Pattern:     `^pkg/([^/]+)/`,
			Scope:       "$1",
			Priority:    90,
			Description: "Go public packages",
		},
		// Frontend/UI
		{
			Pattern:     `\.(js|jsx|ts|tsx|vue|svelte|html|css|scss|sass|less)$`,
			Scope:       "ui",
			Priority:    80,
			Description: "Frontend/UI files",
		},
		{
			Pattern:     `^(src/|app/)?components?/`,
			Scope:       "ui",
			Priority:    85,
			Description: "UI components",
		},
		{
			Pattern:     `^(src/|app/)?pages?/`,
			Scope:       "ui",
			Priority:    85,
			Description: "UI pages",
		},

		// API/Backend
		{
			Pattern:     `^(src/|app/)?(api|routes?|controllers?|handlers?)/`,
			Scope:       "api",
			Priority:    90,
			Description: "API endpoints and controllers",
		},
		{
			Pattern:     `^(src/|app/)?middleware/`,
			Scope:       "api",
			Priority:    85,
			Description: "API middleware",
		},

		// Database
		{
			Pattern:     `^(src/|app/)?(models?|entities?|schemas?)/`,
			Scope:       "db",
			Priority:    90,
			Description: "Database models and schemas",
		},
		{
			Pattern:     `^(src/|app/)?(migrations?|seeds?)/`,
			Scope:       "db",
			Priority:    95,
			Description: "Database migrations and seeds",
		},
		{
			Pattern:     `\.(sql|migration)$`,
			Scope:       "db",
			Priority:    90,
			Description: "SQL and migration files",
		},

		// Authentication/Authorization
		{
			Pattern:     `^(src/|app/)?(auth|security)/`,
			Scope:       "auth",
			Priority:    90,
			Description: "Authentication and security",
		},

		// Configuration
		{
			Pattern:     `^(config|configs?)/`,
			Scope:       "config",
			Priority:    95,
			Description: "Configuration files",
		},
		{
			Pattern:     `\.(env|config|conf|ini|yaml|yml|toml|json)$`,
			Scope:       "config",
			Priority:    85,
			Description: "Configuration file formats",
		},
		{
			Pattern:     `^\.env`,
			Scope:       "config",
			Priority:    90,
			Description: "Environment files",
		},

		// Testing
		{
			Pattern:     `^(test|tests?|spec|specs?)/`,
			Scope:       "test",
			Priority:    95,
			Description: "Test directories",
		},
		{
			Pattern:     `\.(test|spec)\.(js|jsx|ts|tsx|go|py|rb|java|php)$`,
			Scope:       "test",
			Priority:    90,
			Description: "Test files",
		},
		{
			Pattern:     `_test\.(go|rs)$`,
			Scope:       "test",
			Priority:    90,
			Description: "Go/Rust test files",
		},

		// Documentation
		{
			Pattern:     `^(docs?|documentation)/`,
			Scope:       "docs",
			Priority:    95,
			Description: "Documentation directories",
		},
		{
			Pattern:     `\.(md|rst|txt|adoc)$`,
			Scope:       "docs",
			Priority:    80,
			Description: "Documentation files",
		},
		{
			Pattern:     `^README`,
			Scope:       "docs",
			Priority:    85,
			Description: "README files",
		},

		// CI/CD
		{
			Pattern:     `^\.github/workflows/`,
			Scope:       "ci",
			Priority:    95,
			Description: "GitHub Actions workflows",
		},
		{
			Pattern:     `^\.gitlab-ci\.yml$`,
			Scope:       "ci",
			Priority:    95,
			Description: "GitLab CI configuration",
		},
		{
			Pattern:     `^(Dockerfile|docker-compose\.yml|\.dockerignore)$`,
			Scope:       "ci",
			Priority:    90,
			Description: "Docker files",
		},
		{
			Pattern:     `^(Makefile|Jenkinsfile)$`,
			Scope:       "ci",
			Priority:    90,
			Description: "Build files",
		},

		// Package management
		{
			Pattern:     `^(package\.json|yarn\.lock|package-lock\.json|go\.mod|go\.sum|Cargo\.toml|Cargo\.lock|requirements\.txt|Pipfile|composer\.json)$`,
			Scope:       "deps",
			Priority:    85,
			Description: "Dependency files",
		},

		// Utilities
		{
			Pattern:     `^(src/|app/)?(utils?|helpers?|lib|libs?)/`,
			Scope:       "utils",
			Priority:    80,
			Description: "Utility and helper functions",
		},

		// Services
		{
			Pattern:     `^(src/|app/)?services?/`,
			Scope:       "service",
			Priority:    85,
			Description: "Service layer",
		},

		// Core/Internal
		{
			Pattern:     `^(src/|app/)?(core|internal)/`,
			Scope:       "core",
			Priority:    85,
			Description: "Core/internal modules",
		},
	}
}

// AddCustomRule adds a custom scope detection rule
func (d *Detector) AddCustomRule(rule types.ScopeDetectionRule) {
	d.rules = append(d.rules, rule)

	// Re-sort rules by priority
	sort.Slice(d.rules, func(i, j int) bool {
		return d.rules[i].Priority > d.rules[j].Priority
	})
}

// GetMatchingRules returns all rules that match the given file path
func (d *Detector) GetMatchingRules(filePath string) []types.ScopeDetectionRule {
	var matchingRules []types.ScopeDetectionRule

	for _, rule := range d.rules {
		matched, err := regexp.MatchString(rule.Pattern, filePath)
		if err != nil {
			continue
		}

		if matched {
			matchingRules = append(matchingRules, rule)
		}
	}

	return matchingRules
}

// isHighPriorityScope checks if a scope represents a high-priority module
func (d *Detector) isHighPriorityScope(scope string) bool {
	highPriorityScopes := []string{
		"auth", "api", "db", "security", "core", "config", "ci",
		// Add module-specific scopes that are typically important
		"user", "payment", "order", "notification", "admin",
	}

	for _, highPriority := range highPriorityScopes {
		if scope == highPriority {
			return true
		}
	}

	return false
}
