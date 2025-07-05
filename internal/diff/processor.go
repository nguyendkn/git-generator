package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/nguyendkn/git-generator/pkg/types"
)

// Processor handles intelligent diff processing and chunking
type Processor struct {
	maxChunkSize int
	maxFiles     int
}

// NewProcessor creates a new diff processor
func NewProcessor(maxChunkSize, maxFiles int) *Processor {
	if maxChunkSize <= 0 {
		maxChunkSize = 4000 // Default max chunk size in characters
	}
	if maxFiles <= 0 {
		maxFiles = 20 // Default max files to process
	}

	return &Processor{
		maxChunkSize: maxChunkSize,
		maxFiles:     maxFiles,
	}
}

// ProcessDiff processes and chunks a diff summary for AI consumption
func (p *Processor) ProcessDiff(summary *types.DiffSummary) (*ProcessedDiff, error) {
	if summary == nil {
		return nil, fmt.Errorf("diff summary is nil")
	}

	// Sort files by importance (prioritize certain file types and smaller changes)
	sortedFiles := p.prioritizeFiles(summary.Files)

	// Limit the number of files to process
	if len(sortedFiles) > p.maxFiles {
		sortedFiles = sortedFiles[:p.maxFiles]
	}

	// Create chunks
	chunks := p.createChunks(sortedFiles)

	// Generate summary
	diffSummary := p.generateSummary(summary, sortedFiles)

	return &ProcessedDiff{
		Summary:      diffSummary,
		Chunks:       chunks,
		TotalFiles:   len(sortedFiles),
		TotalAdded:   summary.TotalAdded,
		TotalDeleted: summary.TotalDeleted,
		Languages:    p.extractLanguages(sortedFiles),
		DiffSummary:  summary, // Store original diff summary for scope detection
	}, nil
}

// ProcessedDiff represents a processed and chunked diff
type ProcessedDiff struct {
	Summary       string               `json:"summary"`
	Chunks        []DiffChunk          `json:"chunks"`
	TotalFiles    int                  `json:"total_files"`
	TotalAdded    int                  `json:"total_added"`
	TotalDeleted  int                  `json:"total_deleted"`
	Languages     map[string]int       `json:"languages"`
	ChangeContext *types.ChangeContext `json:"change_context,omitempty"`
	DiffSummary   *types.DiffSummary   `json:"diff_summary,omitempty"` // Original diff summary for scope detection
}

// SetChangeContext adds change context to the processed diff
func (pd *ProcessedDiff) SetChangeContext(context *types.ChangeContext) {
	pd.ChangeContext = context
}

// DiffChunk represents a chunk of diff data
type DiffChunk struct {
	Files       []types.FileChange `json:"files"`
	Description string             `json:"description"`
	Size        int                `json:"size"` // Size in characters
}

// prioritizeFiles sorts files by importance for commit message generation
func (p *Processor) prioritizeFiles(files []types.FileChange) []types.FileChange {
	sortedFiles := make([]types.FileChange, len(files))
	copy(sortedFiles, files)

	sort.Slice(sortedFiles, func(i, j int) bool {
		fileI, fileJ := sortedFiles[i], sortedFiles[j]

		// Priority 1: Change type (added/deleted are more important than modified)
		priorityI := p.getChangeTypePriority(fileI.ChangeType)
		priorityJ := p.getChangeTypePriority(fileJ.ChangeType)
		if priorityI != priorityJ {
			return priorityI > priorityJ
		}

		// Priority 2: File type importance
		importanceI := p.getFileImportance(fileI.Path)
		importanceJ := p.getFileImportance(fileJ.Path)
		if importanceI != importanceJ {
			return importanceI > importanceJ
		}

		// Priority 3: Smaller changes first (easier to understand)
		totalChangesI := fileI.LinesAdded + fileI.LinesDeleted
		totalChangesJ := fileJ.LinesAdded + fileJ.LinesDeleted
		return totalChangesI < totalChangesJ
	})

	return sortedFiles
}

// getChangeTypePriority returns priority score for change types
func (p *Processor) getChangeTypePriority(changeType types.ChangeType) int {
	switch changeType {
	case types.ChangeTypeAdded:
		return 4
	case types.ChangeTypeDeleted:
		return 3
	case types.ChangeTypeRenamed:
		return 2
	case types.ChangeTypeModified:
		return 1
	default:
		return 0
	}
}

// getFileImportance returns importance score for different file types
func (p *Processor) getFileImportance(filePath string) int {
	path := strings.ToLower(filePath)

	// Build and dependency files (check first for highest priority)
	if strings.Contains(path, "package.json") || strings.Contains(path, "go.mod") ||
		strings.Contains(path, "requirements.txt") || strings.Contains(path, "cargo.toml") ||
		strings.Contains(path, "pom.xml") || strings.Contains(path, "build.gradle") {
		return 9
	}

	// Configuration files
	if strings.Contains(path, "config") || strings.Contains(path, ".env") ||
		strings.HasSuffix(path, ".json") || strings.HasSuffix(path, ".yaml") ||
		strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".toml") {
		return 8
	}

	// Documentation
	if strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".rst") ||
		strings.Contains(path, "readme") || strings.Contains(path, "doc") {
		return 7
	}

	// Test files (check before source code to give them lower priority)
	if strings.Contains(path, "test") || strings.Contains(path, "spec") ||
		strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, ".test.js") {
		return 5
	}

	// Source code files
	if strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".js") ||
		strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".py") ||
		strings.HasSuffix(path, ".java") || strings.HasSuffix(path, ".cpp") ||
		strings.HasSuffix(path, ".c") || strings.HasSuffix(path, ".rs") {
		return 6
	}

	// Web assets
	if strings.HasSuffix(path, ".css") || strings.HasSuffix(path, ".scss") ||
		strings.HasSuffix(path, ".html") || strings.HasSuffix(path, ".vue") ||
		strings.HasSuffix(path, ".jsx") || strings.HasSuffix(path, ".tsx") {
		return 4
	}

	// Other files
	return 3
}

// createChunks creates manageable chunks from the file list
func (p *Processor) createChunks(files []types.FileChange) []DiffChunk {
	var chunks []DiffChunk
	var currentChunk DiffChunk
	var currentSize int

	for _, file := range files {
		fileSize := len(file.Content)

		// If adding this file would exceed chunk size, start a new chunk
		if currentSize+fileSize > p.maxChunkSize && len(currentChunk.Files) > 0 {
			currentChunk.Description = p.generateChunkDescription(currentChunk.Files)
			currentChunk.Size = currentSize
			chunks = append(chunks, currentChunk)

			// Start new chunk
			currentChunk = DiffChunk{Files: []types.FileChange{}}
			currentSize = 0
		}

		currentChunk.Files = append(currentChunk.Files, file)
		currentSize += fileSize
	}

	// Add the last chunk if it has files
	if len(currentChunk.Files) > 0 {
		currentChunk.Description = p.generateChunkDescription(currentChunk.Files)
		currentChunk.Size = currentSize
		chunks = append(chunks, currentChunk)
	}

	return chunks
}

// generateChunkDescription creates a description for a chunk
func (p *Processor) generateChunkDescription(files []types.FileChange) string {
	if len(files) == 0 {
		return "Empty chunk"
	}

	if len(files) == 1 {
		file := files[0]
		return fmt.Sprintf("%s: %s (%d+, %d-)",
			string(file.ChangeType), file.Path, file.LinesAdded, file.LinesDeleted)
	}

	// Multiple files
	changeTypes := make(map[types.ChangeType]int)
	languages := make(map[string]int)
	totalAdded, totalDeleted := 0, 0

	for _, file := range files {
		changeTypes[file.ChangeType]++
		languages[file.Language]++
		totalAdded += file.LinesAdded
		totalDeleted += file.LinesDeleted
	}

	var parts []string

	// Add change type summary
	for changeType, count := range changeTypes {
		if count > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", count, string(changeType)))
		}
	}

	description := fmt.Sprintf("%d files: %s (%d+, %d-)",
		len(files), strings.Join(parts, ", "), totalAdded, totalDeleted)

	return description
}

// generateSummary creates a high-level summary of the diff
func (p *Processor) generateSummary(summary *types.DiffSummary, files []types.FileChange) string {
	if len(files) == 0 {
		return "No changes detected"
	}

	changeTypes := make(map[types.ChangeType]int)
	languages := make(map[string]int)

	for _, file := range files {
		changeTypes[file.ChangeType]++
		if file.Language != "Unknown" {
			languages[file.Language]++
		}
	}

	var parts []string

	// Change type summary
	for changeType, count := range changeTypes {
		if count > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", count, string(changeType)))
		}
	}

	// Language summary
	var langParts []string
	for lang, count := range languages {
		if count > 0 {
			langParts = append(langParts, fmt.Sprintf("%s (%d)", lang, count))
		}
	}

	result := fmt.Sprintf("Changes: %s files with %d additions and %d deletions",
		strings.Join(parts, ", "), summary.TotalAdded, summary.TotalDeleted)

	if len(langParts) > 0 {
		result += fmt.Sprintf(". Languages: %s", strings.Join(langParts, ", "))
	}

	return result
}

// extractLanguages extracts language statistics from files
func (p *Processor) extractLanguages(files []types.FileChange) map[string]int {
	languages := make(map[string]int)

	for _, file := range files {
		if file.Language != "Unknown" {
			languages[file.Language]++
		}
	}

	return languages
}
