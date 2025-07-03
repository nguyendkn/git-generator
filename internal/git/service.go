package git

import (
	"bufio"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nguyendkn/git-generator/pkg/types"
)

// Service handles Git operations
type Service struct {
	repoPath string
}

// NewService creates a new Git service
func NewService(repoPath string) *Service {
	if repoPath == "" {
		repoPath = "."
	}
	return &Service{
		repoPath: repoPath,
	}
}

// IsGitRepository checks if the current directory is a Git repository
func (s *Service) IsGitRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = s.repoPath
	return cmd.Run() == nil
}

// HasStagedChanges checks if there are staged changes
func (s *Service) HasStagedChanges() (bool, error) {
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = s.repoPath
	err := cmd.Run()
	if err != nil {
		// Exit code 1 means there are differences
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return true, nil
		}
		return false, fmt.Errorf("failed to check staged changes: %w", err)
	}
	return false, nil
}

// GetStagedDiff returns the diff of staged changes
func (s *Service) GetStagedDiff() (string, error) {
	cmd := exec.Command("git", "diff", "--cached")
	cmd.Dir = s.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get staged diff: %w", err)
	}
	return string(output), nil
}

// GetWorkingDiff returns the diff of working directory changes
func (s *Service) GetWorkingDiff() (string, error) {
	cmd := exec.Command("git", "diff")
	cmd.Dir = s.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get working diff: %w", err)
	}
	return string(output), nil
}

// GetDiffSummary parses git diff output and returns a structured summary
func (s *Service) GetDiffSummary(includeStaged bool) (*types.DiffSummary, error) {
	var diffOutput string
	var err error

	if includeStaged {
		diffOutput, err = s.GetStagedDiff()
	} else {
		diffOutput, err = s.GetWorkingDiff()
	}

	if err != nil {
		return nil, err
	}

	if diffOutput == "" {
		return &types.DiffSummary{
			Files:     []types.FileChange{},
			Timestamp: time.Now(),
		}, nil
	}

	return s.parseDiff(diffOutput)
}

// parseDiff parses git diff output into structured data
func (s *Service) parseDiff(diffOutput string) (*types.DiffSummary, error) {
	var files []types.FileChange
	var totalAdded, totalDeleted int

	// Split diff into individual file sections
	fileSections := s.splitDiffByFile(diffOutput)

	for _, section := range fileSections {
		fileChange, err := s.parseFileSection(section)
		if err != nil {
			continue // Skip files that can't be parsed
		}

		files = append(files, *fileChange)
		totalAdded += fileChange.LinesAdded
		totalDeleted += fileChange.LinesDeleted
	}

	return &types.DiffSummary{
		Files:        files,
		TotalAdded:   totalAdded,
		TotalDeleted: totalDeleted,
		TotalFiles:   len(files),
		Timestamp:    time.Now(),
	}, nil
}

// splitDiffByFile splits the diff output into sections for each file
func (s *Service) splitDiffByFile(diffOutput string) []string {
	var sections []string
	var currentSection strings.Builder

	scanner := bufio.NewScanner(strings.NewReader(diffOutput))
	for scanner.Scan() {
		line := scanner.Text()

		// New file section starts with "diff --git"
		if strings.HasPrefix(line, "diff --git") {
			if currentSection.Len() > 0 {
				sections = append(sections, currentSection.String())
				currentSection.Reset()
			}
		}

		currentSection.WriteString(line + "\n")
	}

	if currentSection.Len() > 0 {
		sections = append(sections, currentSection.String())
	}

	return sections
}

// parseFileSection parses a single file's diff section
func (s *Service) parseFileSection(section string) (*types.FileChange, error) {
	lines := strings.Split(section, "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("invalid file section")
	}

	fileChange := &types.FileChange{
		Content: section,
	}

	// Parse the diff header
	for i, line := range lines {
		if strings.HasPrefix(line, "diff --git") {
			// Extract file paths
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				fileChange.Path = strings.TrimPrefix(parts[3], "b/")
				if parts[2] != parts[3] {
					fileChange.OldPath = strings.TrimPrefix(parts[2], "a/")
					fileChange.ChangeType = types.ChangeTypeRenamed
				}
			}
		} else if strings.HasPrefix(line, "new file mode") {
			fileChange.ChangeType = types.ChangeTypeAdded
		} else if strings.HasPrefix(line, "deleted file mode") {
			fileChange.ChangeType = types.ChangeTypeDeleted
		} else if strings.HasPrefix(line, "@@") {
			// Parse hunk header to get line counts
			fileChange.LinesAdded, fileChange.LinesDeleted = s.parseHunkCounts(lines[i:])
			break
		}
	}

	// Set default change type if not determined
	if fileChange.ChangeType == "" {
		fileChange.ChangeType = types.ChangeTypeModified
	}

	// Detect language
	fileChange.Language = s.detectLanguage(fileChange.Path)

	return fileChange, nil
}

// parseHunkCounts counts added and deleted lines from hunk data
func (s *Service) parseHunkCounts(hunkLines []string) (added, deleted int) {
	for _, line := range hunkLines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			added++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			deleted++
		}
	}
	return added, deleted
}

// detectLanguage detects programming language from file extension
func (s *Service) detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	languageMap := map[string]string{
		".go":   "Go",
		".js":   "JavaScript",
		".ts":   "TypeScript",
		".py":   "Python",
		".java": "Java",
		".cpp":  "C++",
		".c":    "C",
		".cs":   "C#",
		".php":  "PHP",
		".rb":   "Ruby",
		".rs":   "Rust",
		".sh":   "Shell",
		".sql":  "SQL",
		".html": "HTML",
		".css":  "CSS",
		".scss": "SCSS",
		".sass": "Sass",
		".json": "JSON",
		".xml":  "XML",
		".yaml": "YAML",
		".yml":  "YAML",
		".md":   "Markdown",
		".txt":  "Text",
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}

	return "Unknown"
}

// GetFileStats returns statistics about the repository
func (s *Service) GetFileStats() (map[string]int, error) {
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = s.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get file list: %w", err)
	}

	stats := make(map[string]int)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		filePath := scanner.Text()
		lang := s.detectLanguage(filePath)
		stats[lang]++
	}

	return stats, nil
}

// GetRecentCommits returns recent commit history for context analysis
func (s *Service) GetRecentCommits(count int) ([]*types.CommitInfo, error) {
	if count <= 0 {
		count = 10
	}

	cmd := exec.Command("git", "log", "--oneline", "--no-merges", fmt.Sprintf("-%d", count), "--pretty=format:%H|%s|%an|%ad|%f", "--date=iso")
	cmd.Dir = s.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get recent commits: %w", err)
	}

	var commits []*types.CommitInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) >= 5 {
			dateStr := parts[3]
			parsedDate, _ := time.Parse("2006-01-02 15:04:05 -0700", dateStr)

			commit := &types.CommitInfo{
				Hash:    parts[0],
				Subject: parts[1],
				Author:  parts[2],
				Date:    parsedDate,
				Slug:    parts[4],
			}
			commits = append(commits, commit)
		}
	}

	return commits, nil
}

// GetFileHistory returns commit history for specific files
func (s *Service) GetFileHistory(filePaths []string, count int) ([]*types.CommitInfo, error) {
	if count <= 0 {
		count = 5
	}

	args := []string{"log", "--oneline", "--no-merges", fmt.Sprintf("-%d", count), "--pretty=format:%H|%s|%an|%ad|%f", "--date=iso", "--"}
	args = append(args, filePaths...)

	cmd := exec.Command("git", args...)
	cmd.Dir = s.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get file history: %w", err)
	}

	var commits []*types.CommitInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) >= 5 {
			dateStr := parts[3]
			parsedDate, _ := time.Parse("2006-01-02 15:04:05 -0700", dateStr)

			commit := &types.CommitInfo{
				Hash:    parts[0],
				Subject: parts[1],
				Author:  parts[2],
				Date:    parsedDate,
				Slug:    parts[4],
			}
			commits = append(commits, commit)
		}
	}

	return commits, nil
}

// GetCommitDiff returns the diff for a specific commit
func (s *Service) GetCommitDiff(commitHash string) (string, error) {
	cmd := exec.Command("git", "show", "--no-merges", "--format=", commitHash)
	cmd.Dir = s.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit diff: %w", err)
	}

	return string(output), nil
}

// GetCommitDiffSummary returns the diff summary for a specific commit
func (s *Service) GetCommitDiffSummary(commitHash string) (*types.DiffSummary, error) {
	// Get the diff for the commit
	cmd := exec.Command("git", "show", "--numstat", "--format=", commitHash)
	cmd.Dir = s.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commit diff summary: %w", err)
	}

	// Parse the numstat output
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []types.FileChange
	totalAdded, totalDeleted := 0, 0

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		added, _ := strconv.Atoi(parts[0])
		deleted, _ := strconv.Atoi(parts[1])
		path := parts[2]

		files = append(files, types.FileChange{
			Path:         path,
			ChangeType:   types.ChangeTypeModified,
			LinesAdded:   added,
			LinesDeleted: deleted,
		})

		totalAdded += added
		totalDeleted += deleted
	}

	return &types.DiffSummary{
		Files:        files,
		TotalAdded:   totalAdded,
		TotalDeleted: totalDeleted,
		TotalFiles:   len(files),
		Timestamp:    time.Now(),
		Additions:    totalAdded,
		Deletions:    totalDeleted,
	}, nil
}

// GetTags returns all Git tags with version information
func (s *Service) GetTags() ([]*types.GitTag, error) {
	cmd := exec.Command("git", "tag", "-l", "--sort=-version:refname", "--format=%(refname:short)|%(objectname)|%(creatordate:iso)|%(contents:subject)")
	cmd.Dir = s.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var tags []*types.GitTag

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}

		tag := &types.GitTag{
			Name:    parts[0],
			Hash:    parts[1],
			Message: parts[3],
		}

		// Parse date
		if date, err := time.Parse("2006-01-02 15:04:05 -0700", parts[2]); err == nil {
			tag.Date = date
		}

		// Check if it's an annotated tag
		tag.IsAnnotated = s.isAnnotatedTag(tag.Name)

		// Try to parse as semantic version
		if version, err := s.parseSemanticVersion(tag.Name); err == nil {
			tag.Version = version
		}

		tags = append(tags, tag)
	}

	return tags, nil
}

// CreateTag creates a new Git tag
func (s *Service) CreateTag(tagName, message string, annotated bool) error {
	var cmd *exec.Cmd
	if annotated {
		cmd = exec.Command("git", "tag", "-a", tagName, "-m", message)
	} else {
		cmd = exec.Command("git", "tag", tagName)
	}
	cmd.Dir = s.repoPath

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tag %s: %w", tagName, err)
	}

	return nil
}

// HasUnstagedChanges checks if there are unstaged changes
func (s *Service) HasUnstagedChanges() (bool, error) {
	cmd := exec.Command("git", "diff", "--quiet")
	cmd.Dir = s.repoPath
	err := cmd.Run()
	if err != nil {
		// git diff --quiet returns non-zero exit code if there are differences
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode() == 1, nil
		}
		return false, fmt.Errorf("failed to check unstaged changes: %w", err)
	}
	return false, nil
}

// isAnnotatedTag checks if a tag is annotated
func (s *Service) isAnnotatedTag(tagName string) bool {
	cmd := exec.Command("git", "cat-file", "-t", tagName)
	cmd.Dir = s.repoPath
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "tag"
}

// parseSemanticVersion attempts to parse a tag name as semantic version
func (s *Service) parseSemanticVersion(tagName string) (*types.SemanticVersion, error) {
	// Remove 'v' prefix if present
	versionStr := strings.TrimPrefix(tagName, "v")

	// Basic regex for semantic version
	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-([a-zA-Z]+)(?:\.(\d+))?)?$`)
	matches := re.FindStringSubmatch(versionStr)

	if len(matches) < 4 {
		return nil, fmt.Errorf("not a semantic version")
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	version := &types.SemanticVersion{
		Major: major,
		Minor: minor,
		Patch: patch,
		Raw:   versionStr,
	}

	// Handle pre-release
	if len(matches) > 4 && matches[4] != "" {
		version.PreRelease = types.PreReleaseType(matches[4])
		if len(matches) > 5 && matches[5] != "" {
			if preNumber, err := strconv.Atoi(matches[5]); err == nil {
				version.PreNumber = preNumber
			}
		}
	}

	return version, nil
}
