package context

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nguyendkn/git-generator/internal/git"
	"github.com/nguyendkn/git-generator/pkg/types"
)

// Analyzer analyzes changes and provides context for enhanced commit message generation
type Analyzer struct {
	gitService *git.Service
}

// NewAnalyzer creates a new context analyzer
func NewAnalyzer(gitService *git.Service) *Analyzer {
	return &Analyzer{
		gitService: gitService,
	}
}

// AnalyzeChangeContext analyzes the current changes and provides comprehensive context
func (a *Analyzer) AnalyzeChangeContext(diffSummary *types.DiffSummary) (*types.ChangeContext, error) {
	context := &types.ChangeContext{
		ChangePatterns: make(map[string]any),
	}

	// Get recent commits for context
	recentCommits, err := a.gitService.GetRecentCommits(10)
	if err != nil {
		// Don't fail if we can't get history, just continue without it
		recentCommits = []*types.CommitInfo{}
	}
	context.RecentCommits = recentCommits

	// Get related commits for changed files
	if len(diffSummary.Files) > 0 {
		filePaths := make([]string, len(diffSummary.Files))
		for i, file := range diffSummary.Files {
			filePaths[i] = file.Path
		}

		relatedCommits, err := a.gitService.GetFileHistory(filePaths, 5)
		if err == nil {
			context.RelatedCommits = relatedCommits
		}
	}

	// Analyze configuration changes
	configChanges := a.analyzeConfigChanges(diffSummary)
	if len(configChanges) > 0 {
		context.ConfigChanges = configChanges
	}

	// Analyze function changes
	functionChanges := a.analyzeFunctionChanges(diffSummary)
	if len(functionChanges) > 0 {
		context.FunctionChanges = functionChanges
	}

	// Detect performance-related changes
	performanceHints := a.detectPerformanceChanges(diffSummary)
	if len(performanceHints) > 0 {
		context.PerformanceHints = performanceHints
	}

	// Analyze change patterns
	a.analyzeChangePatterns(diffSummary, context)

	return context, nil
}

// analyzeConfigChanges detects configuration parameter changes
func (a *Analyzer) analyzeConfigChanges(diffSummary *types.DiffSummary) []*types.ConfigChange {
	var configChanges []*types.ConfigChange

	configFilePatterns := []string{
		`\.yaml$`, `\.yml$`, `\.json$`, `\.toml$`, `\.ini$`, `\.conf$`, `\.config$`,
		`config\.go$`, `settings\.go$`, `constants\.go$`,
	}

	for _, file := range diffSummary.Files {
		isConfigFile := false
		for _, pattern := range configFilePatterns {
			if matched, _ := regexp.MatchString(pattern, file.Path); matched {
				isConfigFile = true
				break
			}
		}

		if isConfigFile {
			changes := a.extractConfigChanges(file)
			configChanges = append(configChanges, changes...)
		}
	}

	return configChanges
}

// extractConfigChanges extracts specific configuration changes from file content
func (a *Analyzer) extractConfigChanges(file types.FileChange) []*types.ConfigChange {
	var changes []*types.ConfigChange

	// Parse diff content for configuration changes
	lines := strings.Split(file.Content, "\n")
	var removedConfigs = make(map[string]*types.ConfigChange)
	var addedConfigs = make(map[string]*types.ConfigChange)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for configuration patterns
		if strings.HasPrefix(line, "-") && (strings.Contains(line, ":") || strings.Contains(line, "=")) {
			// Removed configuration
			change := a.parseConfigLine(file.Path, line[1:], "removed")
			if change != nil {
				removedConfigs[change.Parameter] = change
			}
		} else if strings.HasPrefix(line, "+") && (strings.Contains(line, ":") || strings.Contains(line, "=")) {
			// Added configuration
			change := a.parseConfigLine(file.Path, line[1:], "added")
			if change != nil {
				addedConfigs[change.Parameter] = change
			}
		}
	}

	// Match removed and added configs to detect value changes
	for param, removedConfig := range removedConfigs {
		if addedConfig, exists := addedConfigs[param]; exists {
			// This is a value change, not addition/removal
			change := &types.ConfigChange{
				File:      file.Path,
				Parameter: param,
				OldValue:  removedConfig.NewValue, // In removed config, NewValue is actually the old value
				NewValue:  addedConfig.NewValue,
				Context:   a.analyzeConfigChangeContext(param, removedConfig.NewValue, addedConfig.NewValue),
			}
			changes = append(changes, change)
			// Remove from maps so they don't get added as separate changes
			delete(removedConfigs, param)
			delete(addedConfigs, param)
		}
	}

	// Add remaining removed configs
	for _, change := range removedConfigs {
		change.Context = fmt.Sprintf("Configuration parameter removed: %s", change.Parameter)
		changes = append(changes, change)
	}

	// Add remaining added configs
	for _, change := range addedConfigs {
		change.Context = fmt.Sprintf("New configuration parameter added: %s", change.Parameter)
		changes = append(changes, change)
	}

	return changes
}

// parseConfigLine parses a configuration line and extracts parameter and value
func (a *Analyzer) parseConfigLine(filePath, line, changeType string) *types.ConfigChange {
	line = strings.TrimSpace(line)

	// YAML/JSON style (key: value)
	if strings.Contains(line, ":") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Remove quotes if present
			value = strings.Trim(value, `"'`)

			return &types.ConfigChange{
				File:      filePath,
				Parameter: key,
				NewValue:  value,
				Context:   fmt.Sprintf("Configuration %s", changeType),
			}
		}
	}

	// Properties style (key=value)
	if strings.Contains(line, "=") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			return &types.ConfigChange{
				File:      filePath,
				Parameter: key,
				NewValue:  value,
				Context:   fmt.Sprintf("Configuration %s", changeType),
			}
		}
	}

	return nil
}

// analyzeConfigChangeContext provides detailed context about configuration changes
func (a *Analyzer) analyzeConfigChangeContext(parameter string, oldValue, newValue any) string {
	paramLower := strings.ToLower(parameter)

	// Convert values to strings for analysis
	oldStr := fmt.Sprintf("%v", oldValue)
	newStr := fmt.Sprintf("%v", newValue)

	// Analyze specific types of configuration changes
	if strings.Contains(paramLower, "timeout") || strings.Contains(paramLower, "delay") {
		return a.analyzeTimeoutChange(parameter, oldStr, newStr)
	}

	if strings.Contains(paramLower, "port") || strings.Contains(paramLower, "host") || strings.Contains(paramLower, "url") {
		return a.analyzeNetworkChange(parameter, oldStr, newStr)
	}

	if strings.Contains(paramLower, "size") || strings.Contains(paramLower, "limit") || strings.Contains(paramLower, "max") || strings.Contains(paramLower, "min") {
		return a.analyzeLimitChange(parameter, oldStr, newStr)
	}

	if strings.Contains(paramLower, "enable") || strings.Contains(paramLower, "disable") || oldStr == "true" || oldStr == "false" || newStr == "true" || newStr == "false" {
		return a.analyzeBooleanChange(parameter, oldStr, newStr)
	}

	if strings.Contains(paramLower, "version") || strings.Contains(paramLower, "model") {
		return a.analyzeVersionChange(parameter, oldStr, newStr)
	}

	if strings.Contains(paramLower, "level") || strings.Contains(paramLower, "mode") {
		return a.analyzeModeChange(parameter, oldStr, newStr)
	}

	// Generic change analysis
	return fmt.Sprintf("Configuration value changed from '%s' to '%s'", oldStr, newStr)
}

// analyzeTimeoutChange analyzes timeout and delay configuration changes
func (a *Analyzer) analyzeTimeoutChange(parameter, oldValue, newValue string) string {
	return fmt.Sprintf("Timeout configuration '%s' adjusted from %s to %s - likely for performance optimization or reliability improvement", parameter, oldValue, newValue)
}

// analyzeNetworkChange analyzes network-related configuration changes
func (a *Analyzer) analyzeNetworkChange(parameter, oldValue, newValue string) string {
	return fmt.Sprintf("Network configuration '%s' updated from %s to %s - may indicate environment change or service migration", parameter, oldValue, newValue)
}

// analyzeLimitChange analyzes size, limit, and boundary configuration changes
func (a *Analyzer) analyzeLimitChange(parameter, oldValue, newValue string) string {
	return fmt.Sprintf("Limit configuration '%s' changed from %s to %s - likely for capacity planning or performance tuning", parameter, oldValue, newValue)
}

// analyzeBooleanChange analyzes boolean flag configuration changes
func (a *Analyzer) analyzeBooleanChange(parameter, oldValue, newValue string) string {
	if newValue == "true" && oldValue == "false" {
		return fmt.Sprintf("Feature '%s' enabled - functionality activation", parameter)
	} else if newValue == "false" && oldValue == "true" {
		return fmt.Sprintf("Feature '%s' disabled - functionality deactivation", parameter)
	}
	return fmt.Sprintf("Boolean configuration '%s' toggled from %s to %s", parameter, oldValue, newValue)
}

// analyzeVersionChange analyzes version and model configuration changes
func (a *Analyzer) analyzeVersionChange(parameter, oldValue, newValue string) string {
	return fmt.Sprintf("Version configuration '%s' updated from %s to %s - likely upgrade or compatibility change", parameter, oldValue, newValue)
}

// analyzeModeChange analyzes mode and level configuration changes
func (a *Analyzer) analyzeModeChange(parameter, oldValue, newValue string) string {
	return fmt.Sprintf("Mode configuration '%s' changed from %s to %s - operational behavior modification", parameter, oldValue, newValue)
}

// analyzeFunctionChanges detects function signature and behavior changes
func (a *Analyzer) analyzeFunctionChanges(diffSummary *types.DiffSummary) []*types.FunctionChange {
	var functionChanges []*types.FunctionChange

	// Function patterns for different languages
	functionPatterns := map[string][]string{
		"go":         {`func\s+(\w+)\s*\(`, `func\s+\(\w+\s+\*?\w+\)\s+(\w+)\s*\(`},
		"javascript": {`function\s+(\w+)\s*\(`, `(\w+)\s*:\s*function\s*\(`, `(\w+)\s*=>\s*`},
		"python":     {`def\s+(\w+)\s*\(`},
		"java":       {`(public|private|protected)?\s*(static)?\s*\w+\s+(\w+)\s*\(`},
		"typescript": {`function\s+(\w+)\s*\(`, `(\w+)\s*:\s*\(.*\)\s*=>`},
	}

	for _, file := range diffSummary.Files {
		if file.Language == "" {
			continue
		}

		patterns, exists := functionPatterns[strings.ToLower(file.Language)]
		if !exists {
			continue
		}

		changes := a.extractFunctionChanges(file, patterns)
		functionChanges = append(functionChanges, changes...)
	}

	return functionChanges
}

// extractFunctionChanges extracts function changes from file content
func (a *Analyzer) extractFunctionChanges(file types.FileChange, patterns []string) []*types.FunctionChange {
	var changes []*types.FunctionChange

	lines := strings.Split(file.Content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
			changeType := "modified"
			if strings.HasPrefix(line, "+") {
				changeType = "added"
			} else {
				changeType = "removed"
			}

			cleanLine := strings.TrimSpace(line[1:])

			for _, pattern := range patterns {
				re := regexp.MustCompile(pattern)
				matches := re.FindStringSubmatch(cleanLine)
				if len(matches) > 1 {
					functionName := matches[1]
					if functionName != "" {
						changes = append(changes, &types.FunctionChange{
							File:         file.Path,
							FunctionName: functionName,
							ChangeType:   changeType,
							Impact:       a.determineFunctionImpact(cleanLine),
						})
					}
				}
			}
		}
	}

	return changes
}

// determineFunctionImpact analyzes the impact of function changes
func (a *Analyzer) determineFunctionImpact(line string) string {
	line = strings.ToLower(line)

	// Analyze visibility and scope
	if strings.Contains(line, "public") {
		return "Public API change - may affect external consumers and require version bump"
	}
	if strings.Contains(line, "private") || strings.Contains(line, "internal") {
		return "Internal implementation change - refactoring or optimization"
	}
	if strings.Contains(line, "protected") {
		return "Protected method change - may affect inheritance hierarchy"
	}

	// Analyze function types
	if strings.Contains(line, "test") || strings.Contains(line, "_test") {
		return "Test function change - improving test coverage or fixing test issues"
	}
	if strings.Contains(line, "main") {
		return "Entry point change - application startup or configuration modification"
	}
	if strings.Contains(line, "init") {
		return "Initialization function change - setup or configuration modification"
	}
	if strings.Contains(line, "handler") || strings.Contains(line, "controller") {
		return "Request handler change - API endpoint or business logic modification"
	}
	if strings.Contains(line, "service") || strings.Contains(line, "manager") {
		return "Service layer change - business logic or data processing modification"
	}
	if strings.Contains(line, "util") || strings.Contains(line, "helper") {
		return "Utility function change - shared functionality improvement"
	}
	if strings.Contains(line, "validate") || strings.Contains(line, "check") {
		return "Validation logic change - input validation or business rule modification"
	}
	if strings.Contains(line, "parse") || strings.Contains(line, "format") {
		return "Data processing change - parsing or formatting logic modification"
	}
	if strings.Contains(line, "async") || strings.Contains(line, "await") || strings.Contains(line, "goroutine") {
		return "Asynchronous function change - concurrency or performance optimization"
	}

	// Analyze parameter patterns
	if strings.Contains(line, "context") {
		return "Context-aware function change - timeout or cancellation handling"
	}
	if strings.Contains(line, "error") {
		return "Error handling function change - improved error management"
	}

	return "Function signature or implementation change - logic modification or enhancement"
}

// detectPerformanceChanges identifies potential performance-related changes
func (a *Analyzer) detectPerformanceChanges(diffSummary *types.DiffSummary) []string {
	var hints []string

	// Categorized performance keywords with specific analysis
	performanceCategories := map[string][]string{
		"caching":     {"cache", "memoize", "redis", "memcached", "lru", "ttl"},
		"database":    {"index", "query", "sql", "database", "db", "orm", "transaction"},
		"concurrency": {"goroutine", "thread", "async", "await", "concurrent", "parallel", "mutex", "lock", "channel"},
		"memory":      {"memory", "heap", "gc", "garbage", "leak", "allocation", "buffer", "pool"},
		"algorithm":   {"algorithm", "complexity", "optimize", "efficient", "sort", "search", "hash"},
		"io":          {"io", "read", "write", "stream", "batch", "bulk", "pipeline"},
		"network":     {"timeout", "retry", "connection", "keepalive", "compression"},
		"monitoring":  {"benchmark", "profile", "metric", "monitor", "trace", "performance"},
	}

	for _, file := range diffSummary.Files {
		content := strings.ToLower(file.Content)

		for category, keywords := range performanceCategories {
			for _, keyword := range keywords {
				if strings.Contains(content, keyword) {
					hint := a.analyzePerformanceChange(file.Path, category, keyword, content)
					hints = append(hints, hint)
					break // Only add one hint per category per file
				}
			}
		}

		// Detect specific performance patterns
		if strings.Contains(content, "+") && strings.Contains(content, "time.") {
			hints = append(hints, fmt.Sprintf("Timing optimization detected in %s - performance measurement or timeout handling", file.Path))
		}

		if strings.Contains(content, "sync.") || strings.Contains(content, "atomic.") {
			hints = append(hints, fmt.Sprintf("Synchronization optimization detected in %s - concurrency safety improvement", file.Path))
		}
	}

	return hints
}

// analyzePerformanceChange provides detailed analysis of performance-related changes
func (a *Analyzer) analyzePerformanceChange(filePath, category, keyword, content string) string {
	switch category {
	case "caching":
		return fmt.Sprintf("Caching optimization in %s (%s) - likely improving response times and reducing computational overhead", filePath, keyword)
	case "database":
		return fmt.Sprintf("Database optimization in %s (%s) - likely improving query performance and data access efficiency", filePath, keyword)
	case "concurrency":
		return fmt.Sprintf("Concurrency optimization in %s (%s) - likely improving throughput and resource utilization", filePath, keyword)
	case "memory":
		return fmt.Sprintf("Memory optimization in %s (%s) - likely reducing memory usage and preventing leaks", filePath, keyword)
	case "algorithm":
		return fmt.Sprintf("Algorithm optimization in %s (%s) - likely improving computational efficiency and reducing complexity", filePath, keyword)
	case "io":
		return fmt.Sprintf("I/O optimization in %s (%s) - likely improving data processing throughput and reducing latency", filePath, keyword)
	case "network":
		return fmt.Sprintf("Network optimization in %s (%s) - likely improving connection reliability and reducing network overhead", filePath, keyword)
	case "monitoring":
		return fmt.Sprintf("Performance monitoring enhancement in %s (%s) - likely improving observability and performance tracking", filePath, keyword)
	default:
		return fmt.Sprintf("Performance-related change detected in %s: %s", filePath, keyword)
	}
}

// analyzeChangePatterns identifies patterns in the changes
func (a *Analyzer) analyzeChangePatterns(diffSummary *types.DiffSummary, context *types.ChangeContext) {
	patterns := context.ChangePatterns

	// Analyze file types
	fileTypes := make(map[string]int)
	for _, file := range diffSummary.Files {
		if file.Language != "" {
			fileTypes[file.Language]++
		}
	}
	patterns["file_types"] = fileTypes

	// Analyze change types
	changeTypes := make(map[string]int)
	for _, file := range diffSummary.Files {
		changeTypes[string(file.ChangeType)]++
	}
	patterns["change_types"] = changeTypes

	// Detect refactoring patterns
	if len(diffSummary.Files) > 3 && changeTypes["modified"] > changeTypes["added"] {
		patterns["likely_refactoring"] = true
	}

	// Detect new feature patterns
	if changeTypes["added"] > changeTypes["modified"] {
		patterns["likely_new_feature"] = true
	}

	// Detect documentation updates
	docFiles := 0
	for _, file := range diffSummary.Files {
		if strings.HasSuffix(strings.ToLower(file.Path), ".md") ||
			strings.HasSuffix(strings.ToLower(file.Path), ".txt") ||
			strings.Contains(strings.ToLower(file.Path), "doc") {
			docFiles++
		}
	}
	if docFiles > 0 {
		patterns["documentation_update"] = true
		patterns["documentation_files"] = docFiles
	}
}
