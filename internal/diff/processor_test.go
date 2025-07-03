package diff

import (
	"testing"
	"time"

	"github.com/nguyendkn/git-generator/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProcessor(t *testing.T) {
	tests := []struct {
		name          string
		maxChunkSize  int
		maxFiles      int
		expectedSize  int
		expectedFiles int
	}{
		{
			name:          "default values",
			maxChunkSize:  0,
			maxFiles:      0,
			expectedSize:  4000,
			expectedFiles: 20,
		},
		{
			name:          "custom values",
			maxChunkSize:  8000,
			maxFiles:      10,
			expectedSize:  8000,
			expectedFiles: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewProcessor(tt.maxChunkSize, tt.maxFiles)
			assert.Equal(t, tt.expectedSize, processor.maxChunkSize)
			assert.Equal(t, tt.expectedFiles, processor.maxFiles)
		})
	}
}

func TestProcessor_ProcessDiff(t *testing.T) {
	processor := NewProcessor(1000, 5)

	files := []types.FileChange{
		{
			Path:         "src/main.go",
			ChangeType:   types.ChangeTypeAdded,
			LinesAdded:   50,
			LinesDeleted: 0,
			Content:      "package main\n\nfunc main() {\n\tprintln(\"Hello\")\n}",
			Language:     "Go",
		},
		{
			Path:         "README.md",
			ChangeType:   types.ChangeTypeModified,
			LinesAdded:   5,
			LinesDeleted: 2,
			Content:      "# Project\n\nUpdated documentation",
			Language:     "Markdown",
		},
		{
			Path:         "config.json",
			ChangeType:   types.ChangeTypeAdded,
			LinesAdded:   10,
			LinesDeleted: 0,
			Content:      "{\"version\": \"1.0.0\"}",
			Language:     "JSON",
		},
	}

	summary := &types.DiffSummary{
		Files:        files,
		TotalAdded:   65,
		TotalDeleted: 2,
		TotalFiles:   3,
		Timestamp:    time.Now(),
	}

	result, err := processor.ProcessDiff(summary)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 3, result.TotalFiles)
	assert.Equal(t, 65, result.TotalAdded)
	assert.Equal(t, 2, result.TotalDeleted)
	assert.Contains(t, result.Summary, "files")
	assert.Contains(t, result.Summary, "65 additions")
	assert.Contains(t, result.Summary, "2 deletions")

	// Check languages
	assert.Contains(t, result.Languages, "Go")
	assert.Contains(t, result.Languages, "Markdown")
	assert.Contains(t, result.Languages, "JSON")
	assert.Equal(t, 1, result.Languages["Go"])
	assert.Equal(t, 1, result.Languages["Markdown"])
	assert.Equal(t, 1, result.Languages["JSON"])

	// Check chunks
	assert.NotEmpty(t, result.Chunks)
}

func TestProcessor_ProcessDiff_NilSummary(t *testing.T) {
	processor := NewProcessor(1000, 5)

	result, err := processor.ProcessDiff(nil)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "diff summary is nil")
}

func TestProcessor_PrioritizeFiles(t *testing.T) {
	processor := NewProcessor(1000, 5)

	files := []types.FileChange{
		{
			Path:         "src/utils.go",
			ChangeType:   types.ChangeTypeModified,
			LinesAdded:   10,
			LinesDeleted: 5,
		},
		{
			Path:         "package.json",
			ChangeType:   types.ChangeTypeModified,
			LinesAdded:   2,
			LinesDeleted: 1,
		},
		{
			Path:         "new_feature.go",
			ChangeType:   types.ChangeTypeAdded,
			LinesAdded:   100,
			LinesDeleted: 0,
		},
		{
			Path:         "old_file.go",
			ChangeType:   types.ChangeTypeDeleted,
			LinesAdded:   0,
			LinesDeleted: 50,
		},
	}

	sorted := processor.prioritizeFiles(files)

	// Added files should come first (highest priority)
	assert.Equal(t, types.ChangeTypeAdded, sorted[0].ChangeType)
	assert.Equal(t, "new_feature.go", sorted[0].Path)

	// Deleted files should come second
	assert.Equal(t, types.ChangeTypeDeleted, sorted[1].ChangeType)
	assert.Equal(t, "old_file.go", sorted[1].Path)

	// Among modified files, package.json should come before utils.go (higher importance)
	modifiedFiles := []types.FileChange{}
	for _, file := range sorted {
		if file.ChangeType == types.ChangeTypeModified {
			modifiedFiles = append(modifiedFiles, file)
		}
	}
	assert.Equal(t, "package.json", modifiedFiles[0].Path)
	assert.Equal(t, "src/utils.go", modifiedFiles[1].Path)
}

func TestProcessor_GetFileImportance(t *testing.T) {
	processor := NewProcessor(1000, 5)

	tests := []struct {
		path     string
		expected int
	}{
		{"package.json", 9},
		{"go.mod", 9},
		{"requirements.txt", 9},
		{"config.yaml", 8},
		{".env", 8},
		{"README.md", 7},
		{"docs/guide.md", 7},
		{"src/main.go", 6},
		{"app.js", 6},
		{"test/main_test.go", 5}, // Test files get priority 5
		{"spec/app.spec.js", 5},  // Test files get priority 5
		{"styles.css", 4},
		{"index.html", 4},
		{"other.txt", 3},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			importance := processor.getFileImportance(tt.path)
			assert.Equal(t, tt.expected, importance)
		})
	}
}

func TestProcessor_GetChangeTypePriority(t *testing.T) {
	processor := NewProcessor(1000, 5)

	tests := []struct {
		changeType types.ChangeType
		expected   int
	}{
		{types.ChangeTypeAdded, 4},
		{types.ChangeTypeDeleted, 3},
		{types.ChangeTypeRenamed, 2},
		{types.ChangeTypeModified, 1},
		{types.ChangeTypeCopied, 0},
	}

	for _, tt := range tests {
		t.Run(string(tt.changeType), func(t *testing.T) {
			priority := processor.getChangeTypePriority(tt.changeType)
			assert.Equal(t, tt.expected, priority)
		})
	}
}

func TestProcessor_CreateChunks(t *testing.T) {
	processor := NewProcessor(100, 5) // Small chunk size for testing

	files := []types.FileChange{
		{
			Path:    "file1.go",
			Content: "short content",
		},
		{
			Path:    "file2.go",
			Content: "this is a much longer content that should definitely exceed the chunk size limit and force creation of a new chunk",
		},
		{
			Path:    "file3.go",
			Content: "another short content",
		},
	}

	chunks := processor.createChunks(files)

	// Should create multiple chunks due to size limits
	assert.Greater(t, len(chunks), 1)

	// Each chunk should have a description
	for _, chunk := range chunks {
		assert.NotEmpty(t, chunk.Description)
		assert.Greater(t, chunk.Size, 0)
		assert.NotEmpty(t, chunk.Files)
	}
}

func TestProcessor_GenerateChunkDescription(t *testing.T) {
	processor := NewProcessor(1000, 5)

	tests := []struct {
		name     string
		files    []types.FileChange
		expected string
	}{
		{
			name:     "empty files",
			files:    []types.FileChange{},
			expected: "Empty chunk",
		},
		{
			name: "single file",
			files: []types.FileChange{
				{
					Path:         "main.go",
					ChangeType:   types.ChangeTypeAdded,
					LinesAdded:   10,
					LinesDeleted: 0,
				},
			},
			expected: "added: main.go (10+, 0-)",
		},
		{
			name: "multiple files",
			files: []types.FileChange{
				{
					Path:         "file1.go",
					ChangeType:   types.ChangeTypeAdded,
					LinesAdded:   10,
					LinesDeleted: 0,
				},
				{
					Path:         "file2.go",
					ChangeType:   types.ChangeTypeModified,
					LinesAdded:   5,
					LinesDeleted: 3,
				},
			},
			expected: "2 files:", // Just check the prefix since order is not deterministic
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description := processor.generateChunkDescription(tt.files)
			if tt.name == "multiple files" {
				// For multiple files, just check that it contains the expected prefix
				assert.Contains(t, description, tt.expected)
				assert.Contains(t, description, "(15+, 3-)")
				assert.Contains(t, description, "1 added")
				assert.Contains(t, description, "1 modified")
			} else {
				assert.Equal(t, tt.expected, description)
			}
		})
	}
}

func TestProcessor_ExtractLanguages(t *testing.T) {
	processor := NewProcessor(1000, 5)

	files := []types.FileChange{
		{Language: "Go"},
		{Language: "Go"},
		{Language: "JavaScript"},
		{Language: "Unknown"},
		{Language: "Python"},
	}

	languages := processor.extractLanguages(files)

	assert.Equal(t, 2, languages["Go"])
	assert.Equal(t, 1, languages["JavaScript"])
	assert.Equal(t, 1, languages["Python"])
	assert.NotContains(t, languages, "Unknown")
}
