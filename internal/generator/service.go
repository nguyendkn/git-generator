package generator

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/nguyendkn/git-generator/internal/ai"
	contextanalyzer "github.com/nguyendkn/git-generator/internal/context"
	"github.com/nguyendkn/git-generator/internal/diff"
	"github.com/nguyendkn/git-generator/internal/formatter"
	"github.com/nguyendkn/git-generator/internal/git"
	"github.com/nguyendkn/git-generator/internal/validation"
	"github.com/nguyendkn/git-generator/pkg/types"
)

// Service orchestrates the commit message generation process
type Service struct {
	gitService      *git.Service
	diffProcessor   *diff.Processor
	aiClient        *ai.GeminiClient
	contextAnalyzer *contextanalyzer.Analyzer
	formatter       *formatter.MessageFormatter
	validator       *validation.Validator
	config          types.Config
}

// NewService creates a new generator service
func NewService(gitService *git.Service, diffProcessor *diff.Processor, aiClient *ai.GeminiClient, config types.Config) *Service {
	contextAnalyzer := contextanalyzer.NewAnalyzer(gitService)

	// Create formatter with config
	formatterConfig := formatter.FormatterConfig{
		MaxSubjectLength:  50,
		MaxBodyLineLength: 72,
		AutoWrapBody:      true,
		BreakOnSentence:   true,
		EnforceBlankLine:  true,
	}
	messageFormatter := formatter.NewMessageFormatterWithConfig(formatterConfig)

	// Create validator with config
	validatorConfig := validation.ValidationConfig{
		MaxSubjectLength:      50,
		MaxBodyLineLength:     72,
		EnforceImperative:     true,
		EnforceCapitalization: true,
		AllowedTypes:          []string{"feat", "fix", "docs", "style", "refactor", "perf", "test", "build", "ci", "chore", "revert", "security", "deps"},
		RequireBody:           false,
		Language:              "vi", // Default to Vietnamese
	}
	messageValidator := validation.NewValidator(validatorConfig)

	return &Service{
		gitService:      gitService,
		diffProcessor:   diffProcessor,
		aiClient:        aiClient,
		contextAnalyzer: contextAnalyzer,
		formatter:       messageFormatter,
		validator:       messageValidator,
		config:          config,
	}
}

// GenerateOptions contains options for commit message generation
type GenerateOptions struct {
	Style         string // conventional, simple, detailed
	IncludeStaged bool   // Include staged changes
	DryRun        bool   // Preview only, don't commit
	Interactive   bool   // Allow user to edit the message
}

// GenerateResult contains the result of commit message generation
type GenerateResult struct {
	CommitMessage *types.CommitMessage `json:"commit_message"`
	ProcessedDiff *diff.ProcessedDiff  `json:"processed_diff"`
	Preview       string               `json:"preview"`
	Applied       bool                 `json:"applied"`
}

// Generate generates a commit message based on current changes
func (s *Service) Generate(ctx context.Context, options GenerateOptions) (*GenerateResult, error) {
	// Validate that we're in a Git repository
	if !s.gitService.IsGitRepository() {
		return nil, fmt.Errorf("not in a Git repository")
	}

	// Check for changes
	hasStaged, err := s.gitService.HasStagedChanges()
	if err != nil {
		return nil, fmt.Errorf("failed to check for staged changes: %w", err)
	}

	if !hasStaged && options.IncludeStaged {
		return nil, fmt.Errorf("no staged changes found. Use 'git add' to stage changes first")
	}

	// Get diff summary
	diffSummary, err := s.gitService.GetDiffSummary(options.IncludeStaged)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff summary: %w", err)
	}

	if len(diffSummary.Files) == 0 {
		return nil, fmt.Errorf("no changes detected")
	}

	// Process the diff
	processedDiff, err := s.diffProcessor.ProcessDiff(diffSummary)
	if err != nil {
		return nil, fmt.Errorf("failed to process diff: %w", err)
	}

	// Analyze change context for enhanced commit message generation
	changeContext, err := s.contextAnalyzer.AnalyzeChangeContext(diffSummary)
	if err != nil {
		// Don't fail if context analysis fails, just log and continue without context
		fmt.Printf("Warning: Failed to analyze change context: %v\n", err)
		changeContext = nil
	}

	// Add context to processed diff
	if changeContext != nil {
		processedDiff.SetChangeContext(changeContext)
	}

	// Generate commit message using AI
	commitMessage, err := s.aiClient.GenerateCommitMessage(ctx, processedDiff, options.Style)
	if err != nil {
		return nil, fmt.Errorf("failed to generate commit message: %w", err)
	}

	// Format the commit message
	formattedMessage := s.formatter.FormatCommitMessage(commitMessage)
	commitMessage.FormattedMessage = formattedMessage

	// Create a temporary message for validation with formatted content
	formattedCommitMessage := &types.CommitMessage{
		Type:     commitMessage.Type,
		Scope:    commitMessage.Scope,
		Subject:  s.extractSubjectFromFormatted(formattedMessage),
		Body:     s.extractBodyFromFormatted(formattedMessage),
		Footer:   commitMessage.Footer,
		Breaking: commitMessage.Breaking,
	}

	// Validate the formatted commit message
	validationResult := s.validator.ValidateCommitMessage(formattedCommitMessage)
	commitMessage.ValidationResult = s.convertValidationResult(validationResult)

	// Create preview with validation info
	preview := s.createPreviewWithValidation(commitMessage, processedDiff, validationResult)

	result := &GenerateResult{
		CommitMessage: commitMessage,
		ProcessedDiff: processedDiff,
		Preview:       preview,
		Applied:       false,
	}

	// Apply the commit if not in dry-run mode
	if !options.DryRun {
		if err := s.applyCommit(commitMessage); err != nil {
			return result, fmt.Errorf("failed to apply commit: %w", err)
		}
		result.Applied = true
	}

	return result, nil
}

// GenerateMultipleOptions generates multiple commit message options
func (s *Service) GenerateMultipleOptions(ctx context.Context, options GenerateOptions, count int) ([]*types.CommitMessage, error) {
	if count <= 0 {
		count = 3
	}
	if count > 5 {
		count = 5 // Limit to avoid excessive API calls
	}

	// Get diff summary once
	diffSummary, err := s.gitService.GetDiffSummary(options.IncludeStaged)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff summary: %w", err)
	}

	if len(diffSummary.Files) == 0 {
		return nil, fmt.Errorf("no changes detected")
	}

	// Process the diff once
	processedDiff, err := s.diffProcessor.ProcessDiff(diffSummary)
	if err != nil {
		return nil, fmt.Errorf("failed to process diff: %w", err)
	}

	// Generate multiple options
	var messages []*types.CommitMessage
	styles := []string{"conventional", "simple", "detailed"}

	for i := 0; i < count; i++ {
		style := styles[i%len(styles)]
		if options.Style != "" {
			style = options.Style
		}

		commitMessage, err := s.aiClient.GenerateCommitMessage(ctx, processedDiff, style)
		if err != nil {
			log.Printf("Failed to generate option %d: %v", i+1, err)
			continue
		}

		messages = append(messages, commitMessage)
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("failed to generate any commit message options")
	}

	return messages, nil
}

// ValidateChanges validates that there are changes to commit
func (s *Service) ValidateChanges(includeStaged bool) error {
	if !s.gitService.IsGitRepository() {
		return fmt.Errorf("not in a Git repository")
	}

	hasStaged, err := s.gitService.HasStagedChanges()
	if err != nil {
		return fmt.Errorf("failed to check for staged changes: %w", err)
	}

	if includeStaged && !hasStaged {
		return fmt.Errorf("no staged changes found")
	}

	diffSummary, err := s.gitService.GetDiffSummary(includeStaged)
	if err != nil {
		return fmt.Errorf("failed to get diff summary: %w", err)
	}

	if len(diffSummary.Files) == 0 {
		return fmt.Errorf("no changes detected")
	}

	return nil
}

// GetChangeSummary returns a summary of current changes
func (s *Service) GetChangeSummary(includeStaged bool) (*diff.ProcessedDiff, error) {
	diffSummary, err := s.gitService.GetDiffSummary(includeStaged)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff summary: %w", err)
	}

	return s.diffProcessor.ProcessDiff(diffSummary)
}

// createPreview creates a formatted preview of the commit message and changes
func (s *Service) createPreview(commitMessage *types.CommitMessage, processedDiff *diff.ProcessedDiff) string {
	// Use formatted message if available, otherwise fall back to String()
	messageText := commitMessage.FormattedMessage
	if messageText == "" {
		messageText = commitMessage.String()
	}
	preview := fmt.Sprintf("Commit Message:\n%s\n\n", messageText)

	preview += fmt.Sprintf("Changes Summary:\n%s\n\n", processedDiff.Summary)

	if len(processedDiff.Languages) > 0 {
		preview += "Languages:\n"
		for lang, count := range processedDiff.Languages {
			preview += fmt.Sprintf("  - %s: %d files\n", lang, count)
		}
		preview += "\n"
	}

	if len(processedDiff.Chunks) > 0 {
		preview += "File Changes:\n"
		for i, chunk := range processedDiff.Chunks {
			if i >= 5 { // Limit preview to first 5 chunks
				preview += fmt.Sprintf("  ... and %d more chunks\n", len(processedDiff.Chunks)-i)
				break
			}
			preview += fmt.Sprintf("  %d. %s\n", i+1, chunk.Description)
		}
	}

	return preview
}

// createPreviewWithValidation creates a formatted preview with validation information
func (s *Service) createPreviewWithValidation(commitMessage *types.CommitMessage, processedDiff *diff.ProcessedDiff, validationResult *validation.ValidationResult) string {
	preview := s.createPreview(commitMessage, processedDiff)

	// Add validation information
	if validationResult != nil {
		preview += "\n=== Validation Results ===\n"

		if validationResult.IsValid {
			preview += "✅ Commit message follows Git best practices\n"
		} else {
			preview += "❌ Commit message has validation issues\n"
		}

		// Show errors
		if len(validationResult.Errors) > 0 {
			preview += "\n🚨 Errors:\n"
			for _, err := range validationResult.Errors {
				preview += fmt.Sprintf("  - %s\n", err.Message)
			}
		}

		// Show warnings
		if len(validationResult.Warnings) > 0 {
			preview += "\n⚠️  Warnings:\n"
			for _, warning := range validationResult.Warnings {
				preview += fmt.Sprintf("  - %s", warning.Message)
				if warning.Suggestion != "" {
					preview += fmt.Sprintf(" (Suggestion: %s)", warning.Suggestion)
				}
				preview += "\n"
			}
		}

		// Show suggestions
		if len(validationResult.Suggestions) > 0 {
			preview += "\n💡 Suggestions:\n"
			for _, suggestion := range validationResult.Suggestions {
				preview += fmt.Sprintf("  - %s", suggestion.Message)
				if suggestion.Suggested != "" {
					preview += fmt.Sprintf("\n    Suggested: %s", suggestion.Suggested)
				}
				preview += "\n"
			}
		}
	}

	return preview
}

// extractBodyFromFormatted extracts the body part from a formatted commit message
func (s *Service) extractBodyFromFormatted(formattedMessage string) string {
	lines := strings.Split(formattedMessage, "\n")

	if len(lines) <= 1 {
		return "" // No body, only subject
	}

	// Find the first empty line (separator between subject and body)
	bodyStartIndex := -1
	for i := 1; i < len(lines); i++ { // Start from line 1 (skip subject)
		if strings.TrimSpace(lines[i]) == "" {
			bodyStartIndex = i + 1
			break
		}
	}

	if bodyStartIndex == -1 {
		// No empty line found, check if line 1 is body (no proper separation)
		if len(lines) > 1 && strings.TrimSpace(lines[1]) != "" {
			bodyStartIndex = 1
		} else {
			return "" // No body found
		}
	}

	if bodyStartIndex >= len(lines) {
		return "" // No body content
	}

	// Find the end of body (before footer if any)
	bodyEndIndex := len(lines)
	for i := bodyStartIndex; i < len(lines); i++ {
		// Look for footer patterns (like "Fixes #123", "Co-authored-by:", etc.)
		line := strings.TrimSpace(lines[i])
		if line == "" && i+1 < len(lines) {
			nextLine := strings.TrimSpace(lines[i+1])
			if strings.Contains(nextLine, ":") || strings.HasPrefix(nextLine, "Fixes") ||
				strings.HasPrefix(nextLine, "Closes") || strings.HasPrefix(nextLine, "Co-authored-by") {
				bodyEndIndex = i
				break
			}
		}
	}

	// Extract body lines
	bodyLines := lines[bodyStartIndex:bodyEndIndex]

	// Remove trailing empty lines
	for len(bodyLines) > 0 && strings.TrimSpace(bodyLines[len(bodyLines)-1]) == "" {
		bodyLines = bodyLines[:len(bodyLines)-1]
	}

	return strings.Join(bodyLines, "\n")
}

// extractSubjectFromFormatted extracts the subject line from a formatted commit message
func (s *Service) extractSubjectFromFormatted(formattedMessage string) string {
	lines := strings.Split(formattedMessage, "\n")

	if len(lines) == 0 {
		return ""
	}

	// Subject is always the first line
	return strings.TrimSpace(lines[0])
}

// convertValidationResult converts validation.ValidationResult to types.CommitValidationResult
func (s *Service) convertValidationResult(vr *validation.ValidationResult) *types.CommitValidationResult {
	if vr == nil {
		return nil
	}

	result := &types.CommitValidationResult{
		IsValid:     vr.IsValid,
		Errors:      make([]types.ValidationError, len(vr.Errors)),
		Warnings:    make([]types.ValidationWarning, len(vr.Warnings)),
		Suggestions: make([]types.ValidationSuggestion, len(vr.Suggestions)),
	}

	// Convert errors
	for i, err := range vr.Errors {
		result.Errors[i] = types.ValidationError{
			Type:     err.Type,
			Message:  err.Message,
			Line:     err.Line,
			Column:   err.Column,
			Severity: err.Severity,
		}
	}

	// Convert warnings
	for i, warning := range vr.Warnings {
		result.Warnings[i] = types.ValidationWarning{
			Type:       warning.Type,
			Message:    warning.Message,
			Suggestion: warning.Suggestion,
		}
	}

	// Convert suggestions
	for i, suggestion := range vr.Suggestions {
		result.Suggestions[i] = types.ValidationSuggestion{
			Type:      suggestion.Type,
			Message:   suggestion.Message,
			Original:  suggestion.Original,
			Suggested: suggestion.Suggested,
		}
	}

	return result
}

// applyCommit applies the commit message to the repository
func (s *Service) applyCommit(commitMessage *types.CommitMessage) error {
	// Use formatted message if available, otherwise fall back to String()
	messageText := commitMessage.FormattedMessage
	if messageText == "" {
		messageText = commitMessage.String()
	}

	cmd := exec.Command("git", "commit", "-m", messageText)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	return nil
}

// GenerateInteractive generates a commit message with interactive confirmation
func (s *Service) GenerateInteractive(ctx context.Context, options GenerateOptions) (*GenerateResult, error) {
	// Generate the commit message first
	options.DryRun = true // Always start with dry run for interactive mode
	result, err := s.Generate(ctx, options)
	if err != nil {
		return nil, err
	}

	// Show preview
	fmt.Println("Generated commit message:")
	fmt.Println(result.CommitMessage.String())
	fmt.Println("\nChange summary:")
	fmt.Println(result.ProcessedDiff.Summary)

	// Ask for confirmation
	if s.confirmCommit() {
		// Apply the commit
		if err := s.applyCommit(result.CommitMessage); err != nil {
			return result, fmt.Errorf("failed to apply commit: %w", err)
		}
		result.Applied = true
		fmt.Println("Commit applied successfully!")
	} else {
		fmt.Println("Commit cancelled.")
	}

	return result, nil
}

// confirmCommit asks the user for confirmation
func (s *Service) confirmCommit() bool {
	fmt.Print("Apply this commit? [y/N]: ")

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		return response == "y" || response == "yes"
	}

	return false
}

// EditCommitMessage allows the user to edit the commit message
func (s *Service) EditCommitMessage(commitMessage *types.CommitMessage) (*types.CommitMessage, error) {
	// Create a temporary file with the commit message
	tmpFile, err := os.CreateTemp("", "commit-msg-*.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write the current message to the file
	if _, err := tmpFile.WriteString(commitMessage.String()); err != nil {
		return nil, fmt.Errorf("failed to write to temp file: %w", err)
	}
	tmpFile.Close()

	// Open the file in the user's editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano" // Default editor
	}

	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run editor: %w", err)
	}

	// Read the edited content
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read edited file: %w", err)
	}

	// Parse the edited content back into a commit message
	editedText := strings.TrimSpace(string(content))
	if editedText == "" {
		return nil, fmt.Errorf("commit message cannot be empty")
	}

	// For simplicity, create a new commit message with the edited text
	// In a more sophisticated implementation, we could parse the conventional format
	editedMessage := &types.CommitMessage{
		Type:        commitMessage.Type,
		Scope:       commitMessage.Scope,
		Description: editedText,
		Breaking:    commitMessage.Breaking,
	}

	return editedMessage, nil
}

// GetFileStats returns statistics about the repository
func (s *Service) GetFileStats() (map[string]int, error) {
	return s.gitService.GetFileStats()
}

// Close closes any resources used by the service
func (s *Service) Close() error {
	if s.aiClient != nil {
		return s.aiClient.Close()
	}
	return nil
}
