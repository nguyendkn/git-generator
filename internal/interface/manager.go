package interfaces

import (
	"context"
	"fmt"

	"github.com/nguyendkn/git-generator/internal/ai"
	"github.com/nguyendkn/git-generator/internal/config"
	"github.com/nguyendkn/git-generator/internal/diff"
	"github.com/nguyendkn/git-generator/internal/generator"
	"github.com/nguyendkn/git-generator/internal/git"
	"github.com/nguyendkn/git-generator/internal/ui"
	"github.com/nguyendkn/git-generator/pkg/types"
)

// InterfaceMode represents the interface mode
type InterfaceMode string

const (
	ModeCLI         InterfaceMode = "cli"
	ModeInteractive InterfaceMode = "interactive"
	ModeAuto        InterfaceMode = "auto" // Auto-detect based on context
)

// GenerateRequest represents a unified request for commit message generation
type GenerateRequest struct {
	Style        string
	DryRun       bool
	Staged       bool
	Multiple     bool
	CustomPrompt string
	Language     string
	MaxSubject   int
	Validation   bool
	IncludeScope bool
	Mode         InterfaceMode
}

// Manager handles dual interface support
type Manager struct {
	config     *types.Config
	configMgr  *config.Manager
	gitService *git.Service
	diffProc   *diff.Processor
	aiClient   *ai.GeminiClient
	genService *generator.Service
	version    string
}

// NewManager creates a new interface manager
func NewManager(cfg *types.Config, cfgMgr *config.Manager, version string) (*Manager, error) {
	gitService := git.NewService(".")
	diffProcessor := diff.NewProcessor(cfg.Git.MaxDiffSize, 20)

	aiClient, err := ai.NewGeminiClient(cfg.Gemini)
	if err != nil {
		return nil, fmt.Errorf("failed to create AI client: %w", err)
	}

	genService := generator.NewService(gitService, diffProcessor, aiClient, *cfg)

	return &Manager{
		config:     cfg,
		configMgr:  cfgMgr,
		gitService: gitService,
		diffProc:   diffProcessor,
		aiClient:   aiClient,
		genService: genService,
		version:    version,
	}, nil
}

// Close closes the manager and its resources
func (m *Manager) Close() error {
	if m.aiClient != nil {
		return m.aiClient.Close()
	}
	return nil
}

// Generate generates commit message using the specified interface mode
func (m *Manager) Generate(ctx context.Context, req GenerateRequest) (*generator.GenerateResult, error) {
	// Validate changes first
	if err := m.genService.ValidateChanges(req.Staged); err != nil {
		return nil, err
	}

	// Determine actual mode if auto
	actualMode := req.Mode
	if actualMode == ModeAuto {
		actualMode = m.detectMode(req)
	}

	// Handle based on mode
	switch actualMode {
	case ModeCLI:
		return m.generateCLI(ctx, req)
	case ModeInteractive:
		return m.generateInteractive(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported interface mode: %s", actualMode)
	}
}

// generateCLI handles CLI-based generation
func (m *Manager) generateCLI(ctx context.Context, req GenerateRequest) (*generator.GenerateResult, error) {
	if req.Multiple {
		// Generate multiple options
		messages, err := m.genService.GenerateMultipleOptions(ctx, generator.GenerateOptions{
			Style:         req.Style,
			IncludeStaged: req.Staged,
			DryRun:        true, // Always dry run for multiple options
		}, 3)
		if err != nil {
			return nil, err
		}

		ui.ShowInfoMessage("Các tùy chọn commit message được tạo:")
		for i, msg := range messages {
			fmt.Printf("\n%s%d.%s %s\n", ui.ColorCyan, i+1, ui.ColorReset, msg.String())
		}
		return &generator.GenerateResult{
			CommitMessage: messages[0], // Return first option
			Preview:       "Multiple options displayed",
		}, nil
	}

	// Generate single commit message
	return m.genService.Generate(ctx, generator.GenerateOptions{
		Style:         req.Style,
		IncludeStaged: req.Staged,
		DryRun:        req.DryRun,
	})
}

// generateInteractive handles interactive-based generation
func (m *Manager) generateInteractive(ctx context.Context, req GenerateRequest) (*generator.GenerateResult, error) {
	// Show banner and welcome
	ui.ShowBanner(m.version)
	ui.ShowWelcomeMessage()

	// Get interactive options (merge with request)
	options, err := ui.RunInteractiveMode()
	if err != nil {
		return nil, fmt.Errorf("lỗi trong chế độ tương tác: %w", err)
	}

	// Merge interactive options with request
	mergedReq := m.mergeOptions(req, options)

	// Show configuration summary
	ui.ShowConfigurationSummary(options)

	ui.ShowInfoMessage("Đang tạo commit message...")

	// Generate commit message
	return m.genService.Generate(ctx, generator.GenerateOptions{
		Style:         mergedReq.Style,
		IncludeStaged: mergedReq.Staged,
		DryRun:        mergedReq.DryRun,
	})
}

// detectMode automatically detects the appropriate interface mode
func (m *Manager) detectMode(req GenerateRequest) InterfaceMode {
	// If specific CLI flags are provided, use CLI mode
	if req.Multiple || req.Style != "" || req.CustomPrompt != "" {
		return ModeCLI
	}

	// Default to interactive for better UX
	return ModeInteractive
}

// mergeOptions merges CLI request with interactive options
func (m *Manager) mergeOptions(req GenerateRequest, options *ui.InteractiveOptions) GenerateRequest {
	merged := req

	// Interactive options take precedence for user experience
	if options.Style != "" {
		merged.Style = string(options.Style)
	}
	if options.DryRun {
		merged.DryRun = options.DryRun
	}
	if options.Language != "" {
		merged.Language = string(options.Language)
	}
	if options.MaxLength > 0 {
		merged.MaxSubject = options.MaxLength
	}
	merged.Validation = options.Validate
	merged.IncludeScope = options.IncludeScope

	return merged
}

// GetSupportedModes returns list of supported interface modes
func (m *Manager) GetSupportedModes() []InterfaceMode {
	return []InterfaceMode{ModeCLI, ModeInteractive, ModeAuto}
}

// SwitchMode allows switching between interface modes
func (m *Manager) SwitchMode(newMode InterfaceMode) error {
	// Validate mode
	supported := m.GetSupportedModes()
	for _, mode := range supported {
		if mode == newMode {
			return nil // Mode is supported
		}
	}
	return fmt.Errorf("unsupported interface mode: %s", newMode)
}

// SavePreferences saves user preferences for future sessions
func (m *Manager) SavePreferences(req GenerateRequest) error {
	// Update config with user preferences
	if req.Style != "" {
		m.config.Output.Style = req.Style
	}
	if req.Language != "" {
		m.config.Output.Language = req.Language
	}
	if req.MaxSubject > 0 {
		m.config.Output.MaxSubjectLength = req.MaxSubject
	}

	// Save to config file
	return m.configMgr.Save()
}

// LoadPreferences loads user preferences from config
func (m *Manager) LoadPreferences() GenerateRequest {
	return GenerateRequest{
		Style:        m.config.Output.Style,
		Language:     m.config.Output.Language,
		MaxSubject:   m.config.Output.MaxSubjectLength,
		Validation:   true, // Default to enabled
		IncludeScope: true, // Default to enabled
		Staged:       true, // Default to staged changes
		Mode:         ModeAuto,
	}
}
