package version

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/nguyendkn/git-generator/internal/ai"
	"github.com/nguyendkn/git-generator/internal/diff"
	"github.com/nguyendkn/git-generator/internal/git"
	"github.com/nguyendkn/git-generator/pkg/types"
)

// Service handles semantic version analysis and tagging
type Service struct {
	gitService    *git.Service
	diffProcessor *diff.Processor
	aiClient      *ai.GeminiClient
	config        types.Config
}

// NewService creates a new version service
func NewService(gitService *git.Service, diffProcessor *diff.Processor, aiClient *ai.GeminiClient, config types.Config) *Service {
	return &Service{
		gitService:    gitService,
		diffProcessor: diffProcessor,
		aiClient:      aiClient,
		config:        config,
	}
}

// AnalyzeChangesForVersioning analyzes git changes to determine semantic version bump
func (s *Service) AnalyzeChangesForVersioning(ctx context.Context, includeStaged bool) (*types.VersionAnalysis, error) {
	// Validate that we're in a Git repository
	if !s.gitService.IsGitRepository() {
		return nil, fmt.Errorf("not in a Git repository")
	}

	// Get diff summary - if no uncommitted changes, analyze the latest commit
	diffSummary, err := s.gitService.GetDiffSummary(includeStaged)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff summary: %w", err)
	}

	// If no uncommitted changes, analyze the latest commit
	if len(diffSummary.Files) == 0 {
		// Get the latest commit diff summary
		latestCommitDiff, err := s.gitService.GetCommitDiffSummary("HEAD")
		if err != nil {
			return nil, fmt.Errorf("failed to get latest commit diff: %w", err)
		}
		diffSummary = latestCommitDiff

		if len(diffSummary.Files) == 0 {
			return nil, fmt.Errorf("no changes detected in latest commit")
		}
	}

	// Process the diff
	processedDiff, err := s.diffProcessor.ProcessDiff(diffSummary)
	if err != nil {
		return nil, fmt.Errorf("failed to process diff: %w", err)
	}

	// Get recent commits for context
	recentCommits, err := s.gitService.GetRecentCommits(10)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent commits: %w", err)
	}

	// Use AI to analyze changes for version determination
	analysis, err := s.aiClient.AnalyzeChangesForVersioning(ctx, processedDiff, recentCommits)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze changes: %w", err)
	}

	return analysis, nil
}

// GetLatestVersion retrieves the latest semantic version tag from the repository
func (s *Service) GetLatestVersion() (*types.SemanticVersion, error) {
	tags, err := s.gitService.GetTags()
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}

	var latestVersion *types.SemanticVersion
	for _, tag := range tags {
		if tag.Version != nil {
			if latestVersion == nil || s.isVersionNewer(tag.Version, latestVersion) {
				latestVersion = tag.Version
			}
		}
	}

	// If no version tags found, start with 0.0.0
	if latestVersion == nil {
		latestVersion = &types.SemanticVersion{
			Major: 0,
			Minor: 0,
			Patch: 0,
			Raw:   "0.0.0",
		}
	}

	return latestVersion, nil
}

// CalculateNextVersion calculates the next version based on analysis and options
func (s *Service) CalculateNextVersion(currentVersion *types.SemanticVersion, analysis *types.VersionAnalysis, options types.TaggingOptions) *types.SemanticVersion {
	nextVersion := &types.SemanticVersion{
		Major: currentVersion.Major,
		Minor: currentVersion.Minor,
		Patch: currentVersion.Patch,
	}

	// Determine bump type
	bumpType := analysis.RecommendedBump
	if options.ForceBump != "" {
		bumpType = options.ForceBump
	}

	// Apply version bump
	switch bumpType {
	case types.VersionBumpMajor:
		nextVersion.Major++
		nextVersion.Minor = 0
		nextVersion.Patch = 0
	case types.VersionBumpMinor:
		nextVersion.Minor++
		nextVersion.Patch = 0
	case types.VersionBumpPatch:
		nextVersion.Patch++
	}

	// Handle pre-release
	if options.PreRelease != "" {
		nextVersion.PreRelease = options.PreRelease
		nextVersion.PreNumber = 1 // Start with .1 for pre-releases
	}

	nextVersion.Raw = nextVersion.String()
	return nextVersion
}

// CreateTag creates a new Git tag with the specified version
func (s *Service) CreateTag(ctx context.Context, version *types.SemanticVersion, options types.TaggingOptions) error {
	if options.DryRun {
		return nil // Don't actually create tag in dry-run mode
	}

	tagName := version.TagName()
	message := options.Message
	if message == "" {
		message = fmt.Sprintf("Release %s", version.String())
	}

	return s.gitService.CreateTag(tagName, message, options.Annotated)
}

// ParseVersion parses a version string into SemanticVersion
func (s *Service) ParseVersion(versionStr string) (*types.SemanticVersion, error) {
	// Remove 'v' prefix if present
	versionStr = strings.TrimPrefix(versionStr, "v")

	// Regex for semantic version with optional pre-release
	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-([a-zA-Z]+)(?:\.(\d+))?)?$`)
	matches := re.FindStringSubmatch(versionStr)

	if len(matches) < 4 {
		return nil, fmt.Errorf("invalid semantic version format: %s", versionStr)
	}

	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, fmt.Errorf("invalid major version: %s", matches[1])
	}

	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return nil, fmt.Errorf("invalid minor version: %s", matches[2])
	}

	patch, err := strconv.Atoi(matches[3])
	if err != nil {
		return nil, fmt.Errorf("invalid patch version: %s", matches[3])
	}

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
			preNumber, err := strconv.Atoi(matches[5])
			if err == nil {
				version.PreNumber = preNumber
			}
		}
	}

	return version, nil
}

// isVersionNewer compares two semantic versions
func (s *Service) isVersionNewer(v1, v2 *types.SemanticVersion) bool {
	if v1.Major != v2.Major {
		return v1.Major > v2.Major
	}
	if v1.Minor != v2.Minor {
		return v1.Minor > v2.Minor
	}
	if v1.Patch != v2.Patch {
		return v1.Patch > v2.Patch
	}

	// Handle pre-release comparison
	if v1.PreRelease == "" && v2.PreRelease != "" {
		return true // Release version is newer than pre-release
	}
	if v1.PreRelease != "" && v2.PreRelease == "" {
		return false // Pre-release is older than release
	}
	if v1.PreRelease != "" && v2.PreRelease != "" {
		if v1.PreRelease != v2.PreRelease {
			// Compare pre-release types: alpha < beta < rc
			order := map[types.PreReleaseType]int{
				types.PreReleaseAlpha: 1,
				types.PreReleaseBeta:  2,
				types.PreReleaseRC:    3,
			}
			return order[v1.PreRelease] > order[v2.PreRelease]
		}
		return v1.PreNumber > v2.PreNumber
	}

	return false // Versions are equal
}

// ValidateRepositoryState validates that the repository is ready for tagging
func (s *Service) ValidateRepositoryState() error {
	if !s.gitService.IsGitRepository() {
		return fmt.Errorf("not in a Git repository")
	}

	// Check for uncommitted changes
	hasStaged, err := s.gitService.HasStagedChanges()
	if err != nil {
		return fmt.Errorf("failed to check staged changes: %w", err)
	}

	hasUnstaged, err := s.gitService.HasUnstagedChanges()
	if err != nil {
		return fmt.Errorf("failed to check unstaged changes: %w", err)
	}

	if hasStaged || hasUnstaged {
		return fmt.Errorf("repository has uncommitted changes. Please commit or stash changes before tagging")
	}

	return nil
}
