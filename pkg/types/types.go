package types

import (
	"fmt"
	"time"
)

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

// VersionBumpType represents the type of semantic version bump
type VersionBumpType string

const (
	VersionBumpPatch VersionBumpType = "patch" // Bug fixes, documentation updates
	VersionBumpMinor VersionBumpType = "minor" // New features, backwards-compatible changes
	VersionBumpMajor VersionBumpType = "major" // Breaking changes, API changes
)

// PreReleaseType represents pre-release version types
type PreReleaseType string

const (
	PreReleaseAlpha PreReleaseType = "alpha"
	PreReleaseBeta  PreReleaseType = "beta"
	PreReleaseRC    PreReleaseType = "rc"
)

// SemanticVersion represents a semantic version
type SemanticVersion struct {
	Major      int            `json:"major"`
	Minor      int            `json:"minor"`
	Patch      int            `json:"patch"`
	PreRelease PreReleaseType `json:"pre_release,omitempty"`
	PreNumber  int            `json:"pre_number,omitempty"`
	Raw        string         `json:"raw"` // Original version string
}

// String returns the formatted semantic version
func (sv *SemanticVersion) String() string {
	version := fmt.Sprintf("%d.%d.%d", sv.Major, sv.Minor, sv.Patch)
	if sv.PreRelease != "" {
		version += fmt.Sprintf("-%s", sv.PreRelease)
		if sv.PreNumber > 0 {
			version += fmt.Sprintf(".%d", sv.PreNumber)
		}
	}
	return version
}

// TagName returns the version with 'v' prefix for Git tags
func (sv *SemanticVersion) TagName() string {
	return "v" + sv.String()
}

// VersionAnalysis represents AI analysis of changes for version determination
type VersionAnalysis struct {
	RecommendedBump VersionBumpType `json:"recommended_bump"`
	Confidence      float64         `json:"confidence"` // 0.0 to 1.0
	Reasoning       string          `json:"reasoning"`  // AI explanation
	BreakingChanges []string        `json:"breaking_changes,omitempty"`
	NewFeatures     []string        `json:"new_features,omitempty"`
	BugFixes        []string        `json:"bug_fixes,omitempty"`
	Documentation   []string        `json:"documentation,omitempty"`
	Dependencies    []string        `json:"dependencies,omitempty"`
	Metadata        map[string]any  `json:"metadata,omitempty"`
}

// GitTag represents a Git tag with version information
type GitTag struct {
	Name        string           `json:"name"`         // Full tag name (e.g., "v1.2.3")
	Version     *SemanticVersion `json:"version"`      // Parsed semantic version
	Hash        string           `json:"hash"`         // Commit hash
	Date        time.Time        `json:"date"`         // Tag creation date
	Message     string           `json:"message"`      // Tag annotation message
	IsAnnotated bool             `json:"is_annotated"` // Whether it's an annotated tag
}

// TaggingOptions represents options for version tagging
type TaggingOptions struct {
	DryRun     bool            `json:"dry_run"`               // Preview only, don't create tag
	ForceBump  VersionBumpType `json:"force_bump,omitempty"`  // Force specific version type
	PreRelease PreReleaseType  `json:"pre_release,omitempty"` // Create pre-release version
	Message    string          `json:"message,omitempty"`     // Custom tag annotation message
	Push       bool            `json:"push"`                  // Push tag to remote after creation
	Annotated  bool            `json:"annotated"`             // Create annotated tag
}
