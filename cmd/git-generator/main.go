package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/nguyendkn/git-generator/internal/ai"
	"github.com/nguyendkn/git-generator/internal/config"
	"github.com/nguyendkn/git-generator/internal/diff"
	"github.com/nguyendkn/git-generator/internal/generator"
	"github.com/nguyendkn/git-generator/internal/git"
	interfaces "github.com/nguyendkn/git-generator/internal/interface"
	"github.com/nguyendkn/git-generator/internal/ui"
	versioning "github.com/nguyendkn/git-generator/internal/version"
	"github.com/nguyendkn/git-generator/pkg/types"
	"github.com/spf13/cobra"
)

var (
	version      = "1.0.0"
	cfgManager   *config.Manager
	appConfig    *types.Config
	interfaceMgr *interfaces.Manager
)

func main() {
	// Show banner when running without subcommands or with generate command
	if len(os.Args) == 1 || (len(os.Args) > 1 && (os.Args[1] == "generate" || os.Args[1] == "gen" || os.Args[1] == "g")) {
		ui.ShowBanner()
		ui.ShowWelcomeMessage()
	}

	// Cleanup interface manager on exit
	defer func() {
		if interfaceMgr != nil {
			interfaceMgr.Close()
		}
	}()

	if err := rootCmd.Execute(); err != nil {
		ui.ShowErrorMessage(fmt.Sprintf("Lỗi: %v", err))
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "git-generator",
	Short: "AI-powered Git commit message generator",
	Long: `Git Generator is a CLI tool that uses AI to generate high-quality Git commit messages
based on your staged changes. It analyzes your diff and creates conventional commit messages
that follow best practices.`,
	Version: version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for certain commands
		if cmd.Name() == "init" || cmd.Name() == "version" || cmd.Name() == "help" {
			return nil
		}

		// Initialize config manager
		cfgManager = config.NewManager()

		var err error
		appConfig, err = cfgManager.LoadOrCreate()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Initialize interface manager for generate and interactive commands
		if cmd.Name() == "generate" || cmd.Name() == "interactive" {
			interfaceMgr, err = interfaces.NewManager(appConfig, cfgManager)
			if err != nil {
				return fmt.Errorf("failed to initialize interface manager: %w", err)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(interactiveCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(modeCmd)
	rootCmd.AddCommand(tagCmd)
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a commit message for staged changes",
	Long: `Generate an AI-powered commit message based on your staged changes.
The tool analyzes your git diff and creates a conventional commit message.
By default, it will automatically stage all changes (git add .) before generating the commit message.`,
	Aliases: []string{"gen", "g"},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		style, _ := cmd.Flags().GetString("style")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		staged, _ := cmd.Flags().GetBool("staged")
		multiple, _ := cmd.Flags().GetBool("multiple")
		noAdd, _ := cmd.Flags().GetBool("no-add")

		// Auto-stage changes unless --no-add is specified
		if !noAdd && !dryRun {
			gitService := git.NewService(".")
			if !gitService.IsGitRepository() {
				ui.ShowErrorMessage("Không phải trong Git repository")
				return fmt.Errorf("not in a Git repository")
			}

			// Check if there are unstaged changes
			hasUnstaged, err := gitService.HasUnstagedChanges()
			if err != nil {
				ui.ShowWarningMessage(fmt.Sprintf("Không thể kiểm tra unstaged changes: %v", err))
			} else if hasUnstaged {
				ui.ShowInfoMessage("🔄 Đang stage tất cả thay đổi (git add .)...")
				if err := gitService.AddAll(); err != nil {
					ui.ShowErrorMessage(fmt.Sprintf("Lỗi khi stage changes: %v", err))
					return fmt.Errorf("failed to stage changes: %w", err)
				}
				ui.ShowSuccessMessage("✅ Đã stage tất cả thay đổi")
			}
		}

		// Create generate request
		req := interfaces.GenerateRequest{
			Style:    style,
			DryRun:   dryRun,
			Staged:   staged,
			Multiple: multiple,
			Mode:     interfaces.ModeCLI,
		}

		// Use interface manager
		ctx := context.Background()
		result, err := interfaceMgr.Generate(ctx, req)
		if err != nil {
			return err
		}

		// Display result
		if dryRun {
			ui.ShowInfoMessage("Xem trước (chế độ dry-run):")
			fmt.Println(result.Preview)
		} else {
			ui.ShowSuccessMessage("Commit message đã được tạo và áp dụng:")
			// Use formatted message if available, otherwise fall back to String()
			messageText := result.CommitMessage.FormattedMessage
			if messageText == "" {
				messageText = result.CommitMessage.String()
			}
			fmt.Println(messageText)
		}

		return nil
	},
}

var interactiveCmd = &cobra.Command{
	Use:   "interactive",
	Short: "Chế độ tương tác để cấu hình và tạo commit message",
	Long: `Chạy git-generator trong chế độ tương tác với giao diện terminal thân thiện.
Bạn có thể chọn các tùy chọn thông qua menu thay vì dòng lệnh.
Tự động stage tất cả thay đổi trước khi tạo commit message.`,
	Aliases: []string{"i", "int"},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Auto-stage changes in interactive mode
		gitService := git.NewService(".")
		if !gitService.IsGitRepository() {
			ui.ShowErrorMessage("Không phải trong Git repository")
			return fmt.Errorf("not in a Git repository")
		}

		// Check if there are unstaged changes
		hasUnstaged, err := gitService.HasUnstagedChanges()
		if err != nil {
			ui.ShowWarningMessage(fmt.Sprintf("Không thể kiểm tra unstaged changes: %v", err))
		} else if hasUnstaged {
			ui.ShowInfoMessage("🔄 Đang stage tất cả thay đổi (git add .)...")
			if err := gitService.AddAll(); err != nil {
				ui.ShowErrorMessage(fmt.Sprintf("Lỗi khi stage changes: %v", err))
				return fmt.Errorf("failed to stage changes: %w", err)
			}
			ui.ShowSuccessMessage("✅ Đã stage tất cả thay đổi")
		}

		// Create generate request for interactive mode
		req := interfaces.GenerateRequest{
			Staged: true, // Default to staged changes in interactive mode
			Mode:   interfaces.ModeInteractive,
		}

		// Use interface manager
		ctx := context.Background()
		result, err := interfaceMgr.Generate(ctx, req)
		if err != nil {
			ui.ShowErrorMessage(fmt.Sprintf("Lỗi trong chế độ tương tác: %v", err))
			return err
		}

		// Display result
		ui.PrintHeader("Kết quả")
		if req.DryRun {
			ui.ShowInfoMessage("Xem trước (chế độ dry-run):")
			fmt.Println(result.Preview)
		} else {
			ui.ShowSuccessMessage("Commit message đã được tạo và áp dụng:")
			fmt.Println(result.CommitMessage.String())
		}

		return nil
	},
}

func init() {
	generateCmd.Flags().StringP("style", "s", "conventional", "Commit message style (conventional, simple, detailed)")
	generateCmd.Flags().BoolP("dry-run", "d", false, "Preview the commit message without applying it")
	generateCmd.Flags().BoolP("staged", "S", true, "Use staged changes (default: true)")
	generateCmd.Flags().BoolP("multiple", "m", false, "Generate multiple commit message options")
	generateCmd.Flags().Bool("no-add", false, "Skip automatic staging of changes (git add .)")
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Khởi tạo cấu hình git-generator",
	Long:  `Tạo file cấu hình mặc định với tất cả các tùy chọn có sẵn.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.ShowInfoMessage("Đang khởi tạo cấu hình git-generator...")
		manager := config.NewManager()
		err := manager.CreateDefaultConfig()
		if err != nil {
			ui.ShowErrorMessage(fmt.Sprintf("Lỗi tạo cấu hình: %v", err))
			return err
		}
		ui.ShowSuccessMessage("Cấu hình đã được tạo thành công!")
		return nil
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `View and manage git-generator configuration settings.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Hiển thị cấu hình hiện tại",
	RunE: func(cmd *cobra.Command, args []string) error {
		if appConfig == nil {
			ui.ShowErrorMessage("Cấu hình chưa được tải")
			return fmt.Errorf("configuration not loaded")
		}

		ui.PrintHeader("Cấu hình hiện tại")
		fmt.Printf("🤖 Gemini Model: %s%s%s\n", ui.ColorCyan, appConfig.Gemini.Model, ui.ColorReset)
		fmt.Printf("🌡️  Temperature: %s%.2f%s\n", ui.ColorYellow, appConfig.Gemini.Temperature, ui.ColorReset)
		fmt.Printf("📝 Max Tokens: %s%d%s\n", ui.ColorBlue, appConfig.Gemini.MaxTokens, ui.ColorReset)
		fmt.Printf("🎨 Output Style: %s%s%s\n", ui.ColorGreen, appConfig.Output.Style, ui.ColorReset)
		fmt.Printf("📊 Max Diff Size: %s%d%s\n", ui.ColorPurple, appConfig.Git.MaxDiffSize, ui.ColorReset)

		if appConfig.Gemini.APIKey != "" {
			fmt.Printf("🔑 API Key: %s%s...%s%s\n", ui.ColorGreen,
				appConfig.Gemini.APIKey[:8],
				appConfig.Gemini.APIKey[len(appConfig.Gemini.APIKey)-4:], ui.ColorReset)
		} else {
			fmt.Printf("🔑 API Key: %sChưa được thiết lập%s\n", ui.ColorRed, ui.ColorReset)
		}

		configPath, _ := cfgManager.GetConfigPath()
		fmt.Printf("📁 Config File: %s%s%s\n", ui.ColorBlue, configPath, ui.ColorReset)

		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set-api-key [key]",
	Short: "Thiết lập Gemini API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfgManager == nil {
			cfgManager = config.NewManager()
		}

		ui.ShowInfoMessage("Đang cập nhật API key...")
		if err := cfgManager.UpdateAPIKey(args[0]); err != nil {
			ui.ShowErrorMessage(fmt.Sprintf("Lỗi cập nhật API key: %v", err))
			return fmt.Errorf("failed to update API key: %w", err)
		}

		ui.ShowSuccessMessage("API key đã được cập nhật thành công!")
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Hiển thị trạng thái repository và tóm tắt thay đổi",
	RunE: func(cmd *cobra.Command, args []string) error {
		gitService := git.NewService(".")

		if !gitService.IsGitRepository() {
			ui.ShowErrorMessage("Không phải trong Git repository")
			return fmt.Errorf("not in a Git repository")
		}

		ui.PrintHeader("Trạng thái Repository")

		// Check for staged changes
		hasStaged, err := gitService.HasStagedChanges()
		if err != nil {
			ui.ShowErrorMessage(fmt.Sprintf("Lỗi kiểm tra staged changes: %v", err))
			return fmt.Errorf("failed to check staged changes: %w", err)
		}

		if hasStaged {
			ui.ShowSuccessMessage("Có staged changes")
		} else {
			ui.ShowWarningMessage("Không có staged changes")
		}

		if hasStaged {
			diffProcessor := diff.NewProcessor(appConfig.Git.MaxDiffSize, 20)
			genService := generator.NewService(gitService, diffProcessor, nil, *appConfig)

			summary, err := genService.GetChangeSummary(true)
			if err != nil {
				ui.ShowErrorMessage(fmt.Sprintf("Lỗi lấy tóm tắt thay đổi: %v", err))
				return fmt.Errorf("failed to get change summary: %w", err)
			}

			ui.PrintSubHeader("Tóm tắt thay đổi")
			fmt.Printf("%s\n", summary.Summary)

			if len(summary.Languages) > 0 {
				ui.PrintSubHeader("Ngôn ngữ lập trình")
				for lang, count := range summary.Languages {
					fmt.Printf("  %s• %s%s: %s%d files%s\n", ui.ColorBlue, ui.ColorCyan, lang, ui.ColorYellow, count, ui.ColorReset)
				}
			}
		}

		// Get file stats
		stats, err := gitService.GetFileStats()
		if err == nil && len(stats) > 0 {
			ui.PrintSubHeader("Ngôn ngữ trong Repository")
			for lang, count := range stats {
				if lang != "Unknown" {
					fmt.Printf("  %s• %s%s: %s%d files%s\n", ui.ColorGreen, ui.ColorCyan, lang, ui.ColorYellow, count, ui.ColorReset)
				}
			}
		}

		return nil
	},
}

var modeCmd = &cobra.Command{
	Use:   "mode",
	Short: "Quản lý chế độ giao diện",
	Long:  `Chuyển đổi và quản lý chế độ giao diện giữa CLI và Interactive mode.`,
}

var modeListCmd = &cobra.Command{
	Use:   "list",
	Short: "Hiển thị các chế độ giao diện có sẵn",
	RunE: func(cmd *cobra.Command, args []string) error {
		modes := []interfaces.InterfaceMode{interfaces.ModeCLI, interfaces.ModeInteractive, interfaces.ModeAuto}
		ui.PrintHeader("Các chế độ giao diện có sẵn")
		for _, mode := range modes {
			switch mode {
			case interfaces.ModeCLI:
				fmt.Printf("  %s• %sCLI%s - Command Line Interface (dòng lệnh)\n", ui.ColorBlue, ui.ColorCyan, ui.ColorReset)
			case interfaces.ModeInteractive:
				fmt.Printf("  %s• %sInteractive%s - Interactive Terminal UI (tương tác)\n", ui.ColorGreen, ui.ColorCyan, ui.ColorReset)
			case interfaces.ModeAuto:
				fmt.Printf("  %s• %sAuto%s - Tự động phát hiện chế độ phù hợp\n", ui.ColorYellow, ui.ColorCyan, ui.ColorReset)
			}
		}
		return nil
	},
}

func init() {
	modeCmd.AddCommand(modeListCmd)
}

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Tạo semantic version tag tự động",
	Long: `Phân tích thay đổi và tạo semantic version tag tự động dựa trên:
- Conventional commits
- Phân tích diff với AI
- Lịch sử commit gần đây
- Quy tắc semantic versioning (semver)`,
	Aliases: []string{"version", "v"},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		forceBump, _ := cmd.Flags().GetString("type")
		preRelease, _ := cmd.Flags().GetString("pre-release")
		message, _ := cmd.Flags().GetString("message")
		push, _ := cmd.Flags().GetBool("push")
		annotated, _ := cmd.Flags().GetBool("annotated")

		// Initialize services
		gitService := git.NewService(".")
		diffProcessor := diff.NewProcessor(appConfig.Git.MaxDiffSize, 20)

		// Initialize AI client if API key is available
		var aiClient *ai.GeminiClient
		if appConfig.Gemini.APIKey != "" {
			var err error
			aiClient, err = ai.NewGeminiClient(appConfig.Gemini)
			if err != nil {
				ui.ShowErrorMessage(fmt.Sprintf("Lỗi khởi tạo AI client: %v", err))
				return fmt.Errorf("failed to initialize AI client: %w", err)
			}
		} else {
			ui.ShowErrorMessage("Cần cấu hình Google Gemini API key để sử dụng tính năng này")
			ui.ShowInfoMessage("Chạy 'git-generator init' để cấu hình API key")
			return fmt.Errorf("gemini API key is required for version analysis")
		}

		// Initialize version service
		versionService := versioning.NewService(gitService, diffProcessor, aiClient, *appConfig)

		// Show banner
		ui.ShowBanner()
		ui.PrintHeader("🏷️  Semantic Version Tagging")

		// Validate repository state
		if err := versionService.ValidateRepositoryState(); err != nil {
			ui.ShowErrorMessage(fmt.Sprintf("Lỗi trạng thái repository: %v", err))
			return err
		}

		ui.ShowSuccessMessage("✅ Repository sạch, sẵn sàng tạo tag")

		// Get current version
		currentVersion, err := versionService.GetLatestVersion()
		if err != nil {
			ui.ShowErrorMessage(fmt.Sprintf("Lỗi lấy version hiện tại: %v", err))
			return err
		}

		ui.ShowInfoMessage(fmt.Sprintf("📊 Version hiện tại: %s", currentVersion.TagName()))

		// Analyze changes for versioning
		ui.ShowInfoMessage("🤖 Đang phân tích thay đổi để xác định version bump...")

		ctx := context.Background()
		analysis, err := versionService.AnalyzeChangesForVersioning(ctx, true)
		if err != nil {
			ui.ShowErrorMessage(fmt.Sprintf("Lỗi phân tích thay đổi: %v", err))
			return err
		}

		// Display analysis results
		ui.PrintSubHeader("Kết quả phân tích AI")
		fmt.Printf("  %s• Đề xuất: %s%s%s version bump\n",
			ui.ColorBlue, ui.ColorCyan, strings.ToUpper(string(analysis.RecommendedBump)), ui.ColorReset)
		fmt.Printf("  %s• Độ tin cậy: %s%.1f%%%s\n",
			ui.ColorBlue, ui.ColorYellow, analysis.Confidence*100, ui.ColorReset)

		if analysis.Reasoning != "" {
			fmt.Printf("  %s• Lý do: %s%s%s\n",
				ui.ColorBlue, ui.ColorWhite, analysis.Reasoning, ui.ColorReset)
		}

		// Show change details
		if len(analysis.BreakingChanges) > 0 {
			fmt.Printf("  %s• Breaking changes: %s%d%s\n",
				ui.ColorRed, ui.ColorYellow, len(analysis.BreakingChanges), ui.ColorReset)
		}
		if len(analysis.NewFeatures) > 0 {
			fmt.Printf("  %s• Features mới: %s%d%s\n",
				ui.ColorGreen, ui.ColorYellow, len(analysis.NewFeatures), ui.ColorReset)
		}
		if len(analysis.BugFixes) > 0 {
			fmt.Printf("  %s• Bug fixes: %s%d%s\n",
				ui.ColorBlue, ui.ColorYellow, len(analysis.BugFixes), ui.ColorReset)
		}

		// Create tagging options
		options := types.TaggingOptions{
			DryRun:    dryRun,
			Message:   message,
			Push:      push,
			Annotated: annotated,
		}

		// Handle force bump type
		if forceBump != "" {
			switch strings.ToLower(forceBump) {
			case "major":
				options.ForceBump = types.VersionBumpMajor
			case "minor":
				options.ForceBump = types.VersionBumpMinor
			case "patch":
				options.ForceBump = types.VersionBumpPatch
			default:
				ui.ShowErrorMessage(fmt.Sprintf("Loại bump không hợp lệ: %s (chỉ chấp nhận: major, minor, patch)", forceBump))
				return fmt.Errorf("invalid bump type: %s", forceBump)
			}
		}

		// Handle pre-release
		if preRelease != "" {
			switch strings.ToLower(preRelease) {
			case "alpha":
				options.PreRelease = types.PreReleaseAlpha
			case "beta":
				options.PreRelease = types.PreReleaseBeta
			case "rc":
				options.PreRelease = types.PreReleaseRC
			default:
				ui.ShowErrorMessage(fmt.Sprintf("Loại pre-release không hợp lệ: %s (chỉ chấp nhận: alpha, beta, rc)", preRelease))
				return fmt.Errorf("invalid pre-release type: %s", preRelease)
			}
		}

		// Calculate next version
		nextVersion := versionService.CalculateNextVersion(currentVersion, analysis, options)

		ui.PrintSubHeader("Version mới")
		fmt.Printf("  %s%s → %s%s\n",
			ui.ColorYellow, currentVersion.TagName(), nextVersion.TagName(), ui.ColorReset)

		if dryRun {
			ui.ShowInfoMessage("🔍 Chế độ dry-run: Không tạo tag thực tế")
			return nil
		}

		// Create the tag
		ui.ShowInfoMessage(fmt.Sprintf("🏷️  Đang tạo tag %s...", nextVersion.TagName()))

		if err := versionService.CreateTag(ctx, nextVersion, options); err != nil {
			ui.ShowErrorMessage(fmt.Sprintf("Lỗi tạo tag: %v", err))
			return err
		}

		ui.ShowSuccessMessage(fmt.Sprintf("✅ Đã tạo tag %s thành công!", nextVersion.TagName()))

		if options.Push {
			ui.ShowInfoMessage("📤 Đang push tag lên remote...")
			// TODO: Implement push functionality
		}

		return nil
	},
}

func init() {
	tagCmd.Flags().Bool("dry-run", false, "Xem trước version mà không tạo tag thực tế")
	tagCmd.Flags().String("type", "", "Ép kiểu version bump (major|minor|patch)")
	tagCmd.Flags().String("pre-release", "", "Tạo pre-release version (alpha|beta|rc)")
	tagCmd.Flags().String("message", "", "Custom tag annotation message")
	tagCmd.Flags().Bool("push", false, "Push tag lên remote sau khi tạo")
	tagCmd.Flags().Bool("annotated", true, "Tạo annotated tag (mặc định: true)")
}
