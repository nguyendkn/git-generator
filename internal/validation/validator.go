package validation

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/nguyendkn/git-generator/pkg/types"
)

// ValidationResult represents the result of commit message validation
type ValidationResult struct {
	IsValid     bool                   `json:"is_valid"`
	Errors      []ValidationError      `json:"errors,omitempty"`
	Warnings    []ValidationWarning    `json:"warnings,omitempty"`
	Suggestions []ValidationSuggestion `json:"suggestions,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Type     string `json:"type"`
	Message  string `json:"message"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Severity string `json:"severity"` // "error", "warning", "info"
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Type       string `json:"type"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// ValidationSuggestion represents a validation suggestion
type ValidationSuggestion struct {
	Type      string `json:"type"`
	Message   string `json:"message"`
	Original  string `json:"original,omitempty"`
	Suggested string `json:"suggested,omitempty"`
}

// Validator provides commit message validation functionality
type Validator struct {
	config ValidationConfig
}

// ValidationConfig holds validation configuration
type ValidationConfig struct {
	MaxSubjectLength      int      `json:"max_subject_length"`
	MaxBodyLineLength     int      `json:"max_body_line_length"`
	EnforceImperative     bool     `json:"enforce_imperative"`
	EnforceCapitalization bool     `json:"enforce_capitalization"`
	AllowedTypes          []string `json:"allowed_types"`
	RequireBody           bool     `json:"require_body"`
	Language              string   `json:"language"` // "en", "vi"
}

// NewValidator creates a new commit message validator
func NewValidator(config ValidationConfig) *Validator {
	// Set defaults if not provided
	if config.MaxSubjectLength == 0 {
		config.MaxSubjectLength = 50
	}
	if config.MaxBodyLineLength == 0 {
		config.MaxBodyLineLength = 72
	}
	if len(config.AllowedTypes) == 0 {
		config.AllowedTypes = []string{"feat", "fix", "docs", "style", "refactor", "perf", "test", "build", "ci", "chore", "revert"}
	}
	if config.Language == "" {
		config.Language = "en"
	}

	return &Validator{config: config}
}

// ValidateCommitMessage validates a commit message against Git best practices
func (v *Validator) ValidateCommitMessage(message *types.CommitMessage) *ValidationResult {
	result := &ValidationResult{
		IsValid:     true,
		Errors:      []ValidationError{},
		Warnings:    []ValidationWarning{},
		Suggestions: []ValidationSuggestion{},
	}

	// Validate subject line - use Subject if available, otherwise use Description
	subject := message.Subject
	if subject == "" {
		subject = message.Description
	}
	v.validateSubjectLine(subject, result)

	// Validate body if present
	if message.Body != "" {
		v.validateBody(message.Body, result)
	}

	// Validate conventional commits format if applicable
	v.validateConventionalCommits(message, result)

	// Check for atomic commit principles
	v.validateAtomicCommit(message, result)

	// Set overall validity
	result.IsValid = len(result.Errors) == 0

	return result
}

// validateSubjectLine validates the commit message subject line
func (v *Validator) validateSubjectLine(subject string, result *ValidationResult) {
	// Check length
	if len(subject) > v.config.MaxSubjectLength {
		result.Errors = append(result.Errors, ValidationError{
			Type:     "subject_length",
			Message:  v.getLocalizedMessage("subject_too_long", len(subject), v.config.MaxSubjectLength),
			Severity: "error",
		})

		// Suggest truncation
		truncated := v.truncateSubject(subject, v.config.MaxSubjectLength)
		result.Suggestions = append(result.Suggestions, ValidationSuggestion{
			Type:      "subject_truncation",
			Message:   v.getLocalizedMessage("suggest_truncation"),
			Original:  subject,
			Suggested: truncated,
		})
	}

	// Check for trailing period
	if strings.HasSuffix(subject, ".") {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Type:       "trailing_period",
			Message:    v.getLocalizedMessage("no_trailing_period"),
			Suggestion: strings.TrimSuffix(subject, "."),
		})
	}

	// Check capitalization
	if v.config.EnforceCapitalization && len(subject) > 0 {
		firstChar := rune(subject[0])
		if !unicode.IsUpper(firstChar) {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Type:       "capitalization",
				Message:    v.getLocalizedMessage("capitalize_first_letter"),
				Suggestion: strings.ToUpper(string(firstChar)) + subject[1:],
			})
		}
	}

	// Check imperative mood
	if v.config.EnforceImperative {
		if !v.isImperativeMood(subject) {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Type:       "imperative_mood",
				Message:    v.getLocalizedMessage("use_imperative_mood"),
				Suggestion: v.suggestImperativeMood(subject),
			})
		}
	}

	// Check for empty subject
	if strings.TrimSpace(subject) == "" {
		result.Errors = append(result.Errors, ValidationError{
			Type:     "empty_subject",
			Message:  v.getLocalizedMessage("empty_subject"),
			Severity: "error",
		})
	}
}

// validateBody validates the commit message body
func (v *Validator) validateBody(body string, result *ValidationResult) {
	lines := strings.Split(body, "\n")

	for i, line := range lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check line length
		if len(line) > v.config.MaxBodyLineLength {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Type:    "body_line_length",
				Message: v.getLocalizedMessage("body_line_too_long", i+1, len(line), v.config.MaxBodyLineLength),
			})
		}
	}

	// Check for proper separation between subject and body
	if !strings.HasPrefix(body, "\n") {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Type:    "body_separation",
			Message: v.getLocalizedMessage("body_needs_blank_line"),
		})
	}
}

// validateConventionalCommits validates conventional commits format
func (v *Validator) validateConventionalCommits(message *types.CommitMessage, result *ValidationResult) {
	// Check if subject follows conventional commits pattern
	conventionalPattern := regexp.MustCompile(`^([a-z]+)(\([^)]+\))?(!)?: .+`)

	if !conventionalPattern.MatchString(message.Subject) {
		// This might not be a conventional commit, which is okay
		return
	}

	// Extract type from subject
	typePattern := regexp.MustCompile(`^([a-z]+)`)
	matches := typePattern.FindStringSubmatch(message.Subject)

	if len(matches) > 1 {
		commitType := matches[1]

		// Validate type
		if !v.isValidCommitType(commitType) {
			result.Errors = append(result.Errors, ValidationError{
				Type:     "invalid_type",
				Message:  v.getLocalizedMessage("invalid_commit_type", commitType),
				Severity: "error",
			})
		}
	}

	// Check for breaking changes
	v.validateBreakingChanges(message, result)
}

// validateAtomicCommit checks if the commit represents a single logical unit
func (v *Validator) validateAtomicCommit(message *types.CommitMessage, result *ValidationResult) {
	subject := strings.ToLower(message.Subject)

	// Look for indicators of multiple changes
	multipleChangeIndicators := []string{
		" and ", " & ", ", ", " + ", " also ", " additionally ", " furthermore ",
		" moreover ", " besides ", " as well as ", " along with ",
	}

	for _, indicator := range multipleChangeIndicators {
		if strings.Contains(subject, indicator) {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Type:    "atomic_commit",
				Message: v.getLocalizedMessage("consider_atomic_commits"),
			})
			break
		}
	}
}

// Helper methods

// isImperativeMood checks if the subject line uses imperative mood
func (v *Validator) isImperativeMood(subject string) bool {
	// Remove conventional commit prefix if present
	cleanSubject := regexp.MustCompile(`^[a-z]+(\([^)]+\))?(!)?: `).ReplaceAllString(subject, "")

	// Common non-imperative patterns
	nonImperativePatterns := []string{
		`^(added|adding)`,
		`^(fixed|fixing)`,
		`^(updated|updating)`,
		`^(removed|removing)`,
		`^(changed|changing)`,
		`^(implemented|implementing)`,
		`^(refactored|refactoring)`,
	}

	cleanSubjectLower := strings.ToLower(cleanSubject)
	for _, pattern := range nonImperativePatterns {
		if matched, _ := regexp.MatchString(pattern, cleanSubjectLower); matched {
			return false
		}
	}

	return true
}

// suggestImperativeMood suggests imperative mood alternatives
func (v *Validator) suggestImperativeMood(subject string) string {
	replacements := map[string]string{
		"added":        "add",
		"adding":       "add",
		"fixed":        "fix",
		"fixing":       "fix",
		"updated":      "update",
		"updating":     "update",
		"removed":      "remove",
		"removing":     "remove",
		"changed":      "change",
		"changing":     "change",
		"implemented":  "implement",
		"implementing": "implement",
		"refactored":   "refactor",
		"refactoring":  "refactor",
	}

	words := strings.Fields(subject)
	if len(words) > 0 {
		firstWord := strings.ToLower(words[0])
		if replacement, exists := replacements[firstWord]; exists {
			words[0] = strings.ToUpper(replacement[:1]) + replacement[1:]
			return strings.Join(words, " ")
		}
	}

	return subject
}

// truncateSubject truncates subject to specified length while preserving word boundaries
func (v *Validator) truncateSubject(subject string, maxLength int) string {
	if len(subject) <= maxLength {
		return subject
	}

	// Try to truncate at word boundary
	words := strings.Fields(subject)
	result := ""

	for _, word := range words {
		if len(result)+len(word)+1 <= maxLength {
			if result != "" {
				result += " "
			}
			result += word
		} else {
			break
		}
	}

	if result == "" && len(words) > 0 {
		// If even the first word is too long, truncate it
		result = subject[:maxLength-3] + "..."
	}

	return result
}

// isValidCommitType checks if the commit type is valid
func (v *Validator) isValidCommitType(commitType string) bool {
	return slices.Contains(v.config.AllowedTypes, commitType)
}

// validateBreakingChanges validates breaking change indicators
func (v *Validator) validateBreakingChanges(message *types.CommitMessage, result *ValidationResult) {
	hasBreakingIndicator := strings.Contains(message.Subject, "!")
	hasBreakingFooter := strings.Contains(message.Body, "BREAKING CHANGE:")

	if hasBreakingIndicator && !hasBreakingFooter {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Type:    "breaking_change_footer",
			Message: v.getLocalizedMessage("breaking_change_needs_footer"),
		})
	}
}

// getLocalizedMessage returns localized validation messages
func (v *Validator) getLocalizedMessage(key string, args ...any) string {
	messages := v.getMessages()

	if msg, exists := messages[key]; exists {
		return fmt.Sprintf(msg, args...)
	}

	// Fallback to English if message not found
	englishMessages := v.getEnglishMessages()
	if msg, exists := englishMessages[key]; exists {
		return fmt.Sprintf(msg, args...)
	}

	return fmt.Sprintf("Validation message not found: %s", key)
}

// getMessages returns messages for the configured language
func (v *Validator) getMessages() map[string]string {
	switch v.config.Language {
	case "vi":
		return v.getVietnameseMessages()
	default:
		return v.getEnglishMessages()
	}
}

// getEnglishMessages returns English validation messages
func (v *Validator) getEnglishMessages() map[string]string {
	return map[string]string{
		"subject_too_long":             "Subject line is %d characters, should be %d or fewer",
		"suggest_truncation":           "Consider shortening the subject line",
		"no_trailing_period":           "Remove trailing period from subject line",
		"capitalize_first_letter":      "Capitalize the first letter of the subject line",
		"use_imperative_mood":          "Use imperative mood (e.g., 'Fix bug' not 'Fixed bug')",
		"empty_subject":                "Subject line cannot be empty",
		"body_line_too_long":           "Line %d is %d characters, should be %d or fewer",
		"body_needs_blank_line":        "Add blank line between subject and body",
		"invalid_commit_type":          "Invalid commit type '%s'. Use: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert",
		"consider_atomic_commits":      "Consider splitting into multiple atomic commits",
		"breaking_change_needs_footer": "Breaking changes should include 'BREAKING CHANGE:' footer",
	}
}

// getVietnameseMessages returns Vietnamese validation messages
func (v *Validator) getVietnameseMessages() map[string]string {
	return map[string]string{
		"subject_too_long":             "Dòng tiêu đề có %d ký tự, nên có %d ký tự hoặc ít hơn",
		"suggest_truncation":           "Nên rút ngắn dòng tiêu đề",
		"no_trailing_period":           "Bỏ dấu chấm cuối dòng tiêu đề",
		"capitalize_first_letter":      "Viết hoa chữ cái đầu tiên của dòng tiêu đề",
		"use_imperative_mood":          "Sử dụng thể mệnh lệnh (ví dụ: 'Sửa lỗi' không phải 'Đã sửa lỗi')",
		"empty_subject":                "Dòng tiêu đề không được để trống",
		"body_line_too_long":           "Dòng %d có %d ký tự, nên có %d ký tự hoặc ít hơn",
		"body_needs_blank_line":        "Thêm dòng trống giữa tiêu đề và nội dung",
		"invalid_commit_type":          "Loại commit '%s' không hợp lệ. Sử dụng: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert",
		"consider_atomic_commits":      "Nên chia thành nhiều commit nguyên tử",
		"breaking_change_needs_footer": "Thay đổi phá vỡ nên bao gồm footer 'BREAKING CHANGE:'",
	}
}
