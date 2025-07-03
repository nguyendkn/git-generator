package types

import "time"

// CommitType represents the type of commit following conventional commit standards
type CommitType string

const (
	CommitTypeFeat     CommitType = "feat"     // A new feature
	CommitTypeFix      CommitType = "fix"      // A bug fix
	CommitTypeDocs     CommitType = "docs"     // Documentation only changes
	CommitTypeStyle    CommitType = "style"    // Changes that do not affect the meaning of the code
	CommitTypeRefactor CommitType = "refactor" // A code change that neither fixes a bug nor adds a feature
	CommitTypePerf     CommitType = "perf"     // A code change that improves performance
	CommitTypeTest     CommitType = "test"     // Adding missing tests or correcting existing tests
	CommitTypeBuild    CommitType = "build"    // Changes that affect the build system or external dependencies
	CommitTypeCI       CommitType = "ci"       // Changes to our CI configuration files and scripts
	CommitTypeChore    CommitType = "chore"    // Other changes that don't modify src or test files
	CommitTypeRevert   CommitType = "revert"   // Reverts a previous commit
)

// ChangeType represents the type of change in a file
type ChangeType string

const (
	ChangeTypeAdded    ChangeType = "added"
	ChangeTypeModified ChangeType = "modified"
	ChangeTypeDeleted  ChangeType = "deleted"
	ChangeTypeRenamed  ChangeType = "renamed"
	ChangeTypeCopied   ChangeType = "copied"
)

// FileChange represents a change to a single file
type FileChange struct {
	Path         string     `json:"path"`
	OldPath      string     `json:"old_path,omitempty"` // For renames/copies
	ChangeType   ChangeType `json:"change_type"`
	LinesAdded   int        `json:"lines_added"`
	LinesDeleted int        `json:"lines_deleted"`
	Content      string     `json:"content,omitempty"`  // Actual diff content
	Language     string     `json:"language,omitempty"` // Programming language detected
}

// DiffSummary represents a summary of all changes
type DiffSummary struct {
	Files         []FileChange      `json:"files"`
	TotalAdded    int               `json:"total_added"`
	TotalDeleted  int               `json:"total_deleted"`
	TotalFiles    int               `json:"total_files"`
	Timestamp     time.Time         `json:"timestamp"`
	Additions     int               `json:"additions"`
	Deletions     int               `json:"deletions"`
	Languages     map[string]int    `json:"languages"`
	FileLanguages map[string]string `json:"file_languages"`
}

// CommitInfo represents information about a git commit
type CommitInfo struct {
	Hash    string    `json:"hash"`
	Subject string    `json:"subject"`
	Author  string    `json:"author"`
	Date    time.Time `json:"date"`
	Slug    string    `json:"slug"`
	Files   []string  `json:"files,omitempty"`
}

// ChangeContext represents context about changes for enhanced commit message generation
type ChangeContext struct {
	RecentCommits    []*CommitInfo     `json:"recent_commits"`
	RelatedCommits   []*CommitInfo     `json:"related_commits"`
	ConfigChanges    []*ConfigChange   `json:"config_changes,omitempty"`
	FunctionChanges  []*FunctionChange `json:"function_changes,omitempty"`
	PerformanceHints []string          `json:"performance_hints,omitempty"`
	ChangePatterns   map[string]any    `json:"change_patterns,omitempty"`
}

// CommitMessageStyle represents different commit message styles
type CommitMessageStyle string

const (
	StyleConventional CommitMessageStyle = "conventional"
	StyleTraditional  CommitMessageStyle = "traditional"
	StyleDetailed     CommitMessageStyle = "detailed"
	StyleMinimal      CommitMessageStyle = "minimal"
)

// ScopeDetectionRule represents a rule for automatic scope detection
type ScopeDetectionRule struct {
	Pattern     string `json:"pattern"`     // Regex pattern to match file paths
	Scope       string `json:"scope"`       // Scope to assign when pattern matches
	Priority    int    `json:"priority"`    // Priority for rule application (higher = more priority)
	Description string `json:"description"` // Human-readable description of the rule
}

// LanguageConfig represents language-specific configuration
type LanguageConfig struct {
	Language         string            `json:"language"`          // "en", "vi"
	CommitTypes      map[string]string `json:"commit_types"`      // localized commit types
	CommonPhrases    map[string]string `json:"common_phrases"`    // localized common phrases
	StylePreferences map[string]string `json:"style_preferences"` // cultural style preferences
}

// ConfigChange represents a configuration parameter change
type ConfigChange struct {
	File      string `json:"file"`
	Parameter string `json:"parameter"`
	OldValue  any    `json:"old_value"`
	NewValue  any    `json:"new_value"`
	Context   string `json:"context,omitempty"`
}

// FunctionChange represents a function signature or behavior change
type FunctionChange struct {
	File         string   `json:"file"`
	FunctionName string   `json:"function_name"`
	ChangeType   string   `json:"change_type"` // "signature", "behavior", "new", "removed"
	Impact       string   `json:"impact,omitempty"`
	Parameters   []string `json:"parameters,omitempty"`
}

// CommitMessage represents a generated commit message
type CommitMessage struct {
	Type             CommitType              `json:"type"`
	Scope            string                  `json:"scope,omitempty"`
	Description      string                  `json:"description"`
	Subject          string                  `json:"subject,omitempty"` // full subject line
	Body             string                  `json:"body,omitempty"`
	Footer           string                  `json:"footer,omitempty"`
	Breaking         bool                    `json:"breaking"`
	Language         string                  `json:"language,omitempty"` // en, vi
	Metadata         map[string]string       `json:"metadata,omitempty"` // additional metadata
	FormattedMessage string                  `json:"formatted_message,omitempty"`
	ValidationResult *CommitValidationResult `json:"validation_result,omitempty"`
}

// CommitValidationResult represents validation results for commit message
type CommitValidationResult struct {
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

// String returns the formatted commit message
func (cm *CommitMessage) String() string {
	var result string

	// Type and scope
	if cm.Scope != "" {
		result = string(cm.Type) + "(" + cm.Scope + ")"
	} else {
		result = string(cm.Type)
	}

	// Breaking change indicator
	if cm.Breaking {
		result += "!"
	}

	// Description
	result += ": " + cm.Description

	// Body
	if cm.Body != "" {
		result += "\n\n" + cm.Body
	}

	// Footer
	if cm.Footer != "" {
		result += "\n\n" + cm.Footer
	}

	return result
}

// Config represents the application configuration
type Config struct {
	Gemini GeminiConfig `mapstructure:"gemini"`
	Git    GitConfig    `mapstructure:"git"`
	Output OutputConfig `mapstructure:"output"`
}

// GeminiConfig represents Gemini API configuration
type GeminiConfig struct {
	APIKey      string  `mapstructure:"api_key"`
	Model       string  `mapstructure:"model"`
	Temperature float32 `mapstructure:"temperature"`
	MaxTokens   int     `mapstructure:"max_tokens"`
}

// GitConfig represents Git-related configuration
type GitConfig struct {
	MaxDiffSize   int      `mapstructure:"max_diff_size"`
	IgnoreFiles   []string `mapstructure:"ignore_files"`
	IncludeStaged bool     `mapstructure:"include_staged"`
}

// OutputConfig represents output formatting configuration
type OutputConfig struct {
	Style            string `mapstructure:"style"` // conventional, simple, detailed
	MaxLines         int    `mapstructure:"max_lines"`
	DryRun           bool   `mapstructure:"dry_run"`
	Language         string `mapstructure:"language"`           // vi, en
	MaxSubjectLength int    `mapstructure:"max_subject_length"` // Max subject line length
}
