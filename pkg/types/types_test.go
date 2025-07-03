package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommitMessage_String(t *testing.T) {
	tests := []struct {
		name     string
		msg      CommitMessage
		expected string
	}{
		{
			name: "simple conventional commit",
			msg: CommitMessage{
				Type:        CommitTypeFeat,
				Description: "add new feature",
			},
			expected: "feat: add new feature",
		},
		{
			name: "commit with scope",
			msg: CommitMessage{
				Type:        CommitTypeFix,
				Scope:       "auth",
				Description: "fix login issue",
			},
			expected: "fix(auth): fix login issue",
		},
		{
			name: "breaking change",
			msg: CommitMessage{
				Type:        CommitTypeFeat,
				Scope:       "api",
				Description: "change endpoint structure",
				Breaking:    true,
			},
			expected: "feat(api)!: change endpoint structure",
		},
		{
			name: "commit with body",
			msg: CommitMessage{
				Type:        CommitTypeFeat,
				Description: "add user authentication",
				Body:        "Implement JWT-based authentication system\nwith refresh token support",
			},
			expected: "feat: add user authentication\n\nImplement JWT-based authentication system\nwith refresh token support",
		},
		{
			name: "commit with body and footer",
			msg: CommitMessage{
				Type:        CommitTypeFix,
				Description: "resolve memory leak",
				Body:        "Fix memory leak in connection pool",
				Footer:      "Closes #123",
			},
			expected: "fix: resolve memory leak\n\nFix memory leak in connection pool\n\nCloses #123",
		},
		{
			name: "complete commit message",
			msg: CommitMessage{
				Type:        CommitTypeFeat,
				Scope:       "core",
				Description: "implement new caching layer",
				Body:        "Add Redis-based caching for improved performance",
				Footer:      "BREAKING CHANGE: Cache configuration required",
				Breaking:    true,
			},
			expected: "feat(core)!: implement new caching layer\n\nAdd Redis-based caching for improved performance\n\nBREAKING CHANGE: Cache configuration required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.msg.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCommitType_Constants(t *testing.T) {
	// Test that all commit type constants are properly defined
	assert.Equal(t, CommitType("feat"), CommitTypeFeat)
	assert.Equal(t, CommitType("fix"), CommitTypeFix)
	assert.Equal(t, CommitType("docs"), CommitTypeDocs)
	assert.Equal(t, CommitType("style"), CommitTypeStyle)
	assert.Equal(t, CommitType("refactor"), CommitTypeRefactor)
	assert.Equal(t, CommitType("perf"), CommitTypePerf)
	assert.Equal(t, CommitType("test"), CommitTypeTest)
	assert.Equal(t, CommitType("build"), CommitTypeBuild)
	assert.Equal(t, CommitType("ci"), CommitTypeCI)
	assert.Equal(t, CommitType("chore"), CommitTypeChore)
	assert.Equal(t, CommitType("revert"), CommitTypeRevert)
}

func TestChangeType_Constants(t *testing.T) {
	// Test that all change type constants are properly defined
	assert.Equal(t, ChangeType("added"), ChangeTypeAdded)
	assert.Equal(t, ChangeType("modified"), ChangeTypeModified)
	assert.Equal(t, ChangeType("deleted"), ChangeTypeDeleted)
	assert.Equal(t, ChangeType("renamed"), ChangeTypeRenamed)
	assert.Equal(t, ChangeType("copied"), ChangeTypeCopied)
}

func TestFileChange_Structure(t *testing.T) {
	fileChange := FileChange{
		Path:         "src/main.go",
		OldPath:      "src/old_main.go",
		ChangeType:   ChangeTypeRenamed,
		LinesAdded:   10,
		LinesDeleted: 5,
		Content:      "diff content here",
		Language:     "Go",
	}

	assert.Equal(t, "src/main.go", fileChange.Path)
	assert.Equal(t, "src/old_main.go", fileChange.OldPath)
	assert.Equal(t, ChangeTypeRenamed, fileChange.ChangeType)
	assert.Equal(t, 10, fileChange.LinesAdded)
	assert.Equal(t, 5, fileChange.LinesDeleted)
	assert.Equal(t, "diff content here", fileChange.Content)
	assert.Equal(t, "Go", fileChange.Language)
}

func TestDiffSummary_Structure(t *testing.T) {
	files := []FileChange{
		{
			Path:         "file1.go",
			ChangeType:   ChangeTypeAdded,
			LinesAdded:   20,
			LinesDeleted: 0,
		},
		{
			Path:         "file2.go",
			ChangeType:   ChangeTypeModified,
			LinesAdded:   5,
			LinesDeleted: 3,
		},
	}

	summary := DiffSummary{
		Files:        files,
		TotalAdded:   25,
		TotalDeleted: 3,
		TotalFiles:   2,
	}

	assert.Len(t, summary.Files, 2)
	assert.Equal(t, 25, summary.TotalAdded)
	assert.Equal(t, 3, summary.TotalDeleted)
	assert.Equal(t, 2, summary.TotalFiles)
}

func TestConfig_DefaultValues(t *testing.T) {
	config := Config{
		Gemini: GeminiConfig{
			Model:       "gemini-1.5-flash",
			Temperature: 0.3,
			MaxTokens:   1000,
		},
		Git: GitConfig{
			MaxDiffSize:   10000,
			IncludeStaged: true,
		},
		Output: OutputConfig{
			Style:    "conventional",
			MaxLines: 100,
			DryRun:   false,
		},
	}

	assert.Equal(t, "gemini-1.5-flash", config.Gemini.Model)
	assert.Equal(t, float32(0.3), config.Gemini.Temperature)
	assert.Equal(t, 1000, config.Gemini.MaxTokens)
	assert.Equal(t, 10000, config.Git.MaxDiffSize)
	assert.True(t, config.Git.IncludeStaged)
	assert.Equal(t, "conventional", config.Output.Style)
	assert.Equal(t, 100, config.Output.MaxLines)
	assert.False(t, config.Output.DryRun)
}
