package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/nguyendkn/git-generator/pkg/types"
)

// InteractiveOptions represents the options collected from interactive mode
type InteractiveOptions struct {
	Language     string
	Style        types.CommitMessageStyle
	MaxLength    int
	DryRun       bool
	Validate     bool
	IncludeScope bool
	CustomPrompt string
}

// RunInteractiveMode runs the interactive configuration mode
func RunInteractiveMode() (*InteractiveOptions, error) {
	options := &InteractiveOptions{}

	PrintHeader("Cấu hình Git Generator")

	// Language selection
	language, err := selectLanguage()
	if err != nil {
		return nil, fmt.Errorf("lỗi chọn ngôn ngữ: %w", err)
	}
	options.Language = language

	// Style selection
	style, err := selectCommitStyle()
	if err != nil {
		return nil, fmt.Errorf("lỗi chọn style: %w", err)
	}
	options.Style = style

	// Max length configuration
	maxLength, err := selectMaxLength()
	if err != nil {
		return nil, fmt.Errorf("lỗi cấu hình độ dài: %w", err)
	}
	options.MaxLength = maxLength

	// Additional options
	additionalOptions, err := selectAdditionalOptions()
	if err != nil {
		return nil, fmt.Errorf("lỗi chọn tùy chọn bổ sung: %w", err)
	}
	options.DryRun = additionalOptions["dry_run"]
	options.Validate = additionalOptions["validate"]
	options.IncludeScope = additionalOptions["include_scope"]

	// Custom prompt (optional)
	customPrompt, err := getCustomPrompt()
	if err != nil {
		return nil, fmt.Errorf("lỗi nhập prompt tùy chỉnh: %w", err)
	}
	options.CustomPrompt = customPrompt

	return options, nil
}

// selectLanguage allows user to select language
func selectLanguage() (string, error) {
	prompt := promptui.Select{
		Label: "🌐 Chọn ngôn ngữ cho commit message",
		Items: []string{
			"🇺🇸 English (Tiếng Anh)",
			"🇻🇳 Tiếng Việt (Vietnamese)",
		},
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}?",
			Active:   "▶ {{ . | cyan }}",
			Inactive: "  {{ . | white }}",
			Selected: "✅ {{ . | green }}",
		},
	}

	index, _, err := prompt.Run()
	if err != nil {
		return "", err
	}

	switch index {
	case 0:
		return "en", nil
	case 1:
		return "vi", nil
	default:
		return "en", nil
	}
}

// selectCommitStyle allows user to select commit message style
func selectCommitStyle() (types.CommitMessageStyle, error) {
	prompt := promptui.Select{
		Label: "📝 Chọn style commit message",
		Items: []string{
			"🔧 Conventional Commits (feat:, fix:, docs:, ...)",
			"📋 Traditional (Mô tả truyền thống)",
			"📖 Detailed (Chi tiết với context)",
			"⚡ Minimal (Ngắn gọn)",
		},
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}?",
			Active:   "▶ {{ . | cyan }}",
			Inactive: "  {{ . | white }}",
			Selected: "✅ {{ . | green }}",
		},
	}

	index, _, err := prompt.Run()
	if err != nil {
		return types.StyleConventional, err
	}

	switch index {
	case 0:
		return types.StyleConventional, nil
	case 1:
		return types.StyleTraditional, nil
	case 2:
		return types.StyleDetailed, nil
	case 3:
		return types.StyleMinimal, nil
	default:
		return types.StyleConventional, nil
	}
}

// selectMaxLength allows user to configure maximum subject length
func selectMaxLength() (int, error) {
	prompt := promptui.Select{
		Label: "📏 Chọn độ dài tối đa cho subject line",
		Items: []string{
			"50 ký tự (Khuyến nghị Git)",
			"72 ký tự (Truyền thống)",
			"100 ký tự (Linh hoạt)",
			"Tùy chỉnh",
		},
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}?",
			Active:   "▶ {{ . | cyan }}",
			Inactive: "  {{ . | white }}",
			Selected: "✅ {{ . | green }}",
		},
	}

	index, _, err := prompt.Run()
	if err != nil {
		return 50, err
	}

	switch index {
	case 0:
		return 50, nil
	case 1:
		return 72, nil
	case 2:
		return 100, nil
	case 3:
		return getCustomMaxLength()
	default:
		return 50, nil
	}
}

// getCustomMaxLength prompts for custom max length
func getCustomMaxLength() (int, error) {
	validate := func(input string) error {
		length, err := strconv.Atoi(input)
		if err != nil {
			return fmt.Errorf("vui lòng nhập số hợp lệ")
		}
		if length < 20 || length > 200 {
			return fmt.Errorf("độ dài phải từ 20-200 ký tự")
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    "Nhập độ dài tối đa (20-200)",
		Validate: validate,
		Default:  "50",
	}

	result, err := prompt.Run()
	if err != nil {
		return 50, err
	}

	length, _ := strconv.Atoi(result)
	return length, nil
}

// selectAdditionalOptions allows user to select additional options
func selectAdditionalOptions() (map[string]bool, error) {
	options := map[string]bool{
		"dry_run":       false,
		"validate":      true,
		"include_scope": true,
	}

	items := []string{
		"🔍 Dry run (Xem trước không commit)",
		"✅ Validation (Kiểm tra chất lượng)",
		"🎯 Include scope (Tự động detect scope)",
	}

	// Note: promptui.MultiSelect doesn't exist, so we'll use multiple Select prompts
	for i, item := range items {
		selectPrompt := promptui.Select{
			Label: fmt.Sprintf("Bật %s?", item),
			Items: []string{"Có", "Không"},
			Templates: &promptui.SelectTemplates{
				Label:    "{{ . }}?",
				Active:   "▶ {{ . | cyan }}",
				Inactive: "  {{ . | white }}",
				Selected: "✅ {{ . | green }}",
			},
		}

		index, _, err := selectPrompt.Run()
		if err != nil {
			return options, err
		}

		enabled := index == 0
		switch i {
		case 0:
			options["dry_run"] = enabled
		case 1:
			options["validate"] = enabled
		case 2:
			options["include_scope"] = enabled
		}
	}

	return options, nil
}

// getCustomPrompt allows user to enter custom prompt
func getCustomPrompt() (string, error) {
	prompt := promptui.Select{
		Label: "🎨 Bạn có muốn thêm prompt tùy chỉnh?",
		Items: []string{"Không", "Có"},
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}?",
			Active:   "▶ {{ . | cyan }}",
			Inactive: "  {{ . | white }}",
			Selected: "✅ {{ . | green }}",
		},
	}

	index, _, err := prompt.Run()
	if err != nil {
		return "", err
	}

	if index == 0 {
		return "", nil
	}

	// Get custom prompt
	textPrompt := promptui.Prompt{
		Label: "Nhập prompt tùy chỉnh",
		Validate: func(input string) error {
			if len(strings.TrimSpace(input)) == 0 {
				return fmt.Errorf("prompt không được để trống")
			}
			return nil
		},
	}

	return textPrompt.Run()
}

// ShowConfigurationSummary displays the selected configuration
func ShowConfigurationSummary(options *InteractiveOptions) {
	PrintHeader("Tóm tắt cấu hình")

	fmt.Printf("🌐 Ngôn ngữ: %s\n", getLanguageDisplay(options.Language))
	fmt.Printf("📝 Style: %s\n", getStyleDisplay(options.Style))
	fmt.Printf("📏 Độ dài tối đa: %d ký tự\n", options.MaxLength)
	fmt.Printf("🔍 Dry run: %s\n", getBoolDisplay(options.DryRun))
	fmt.Printf("✅ Validation: %s\n", getBoolDisplay(options.Validate))
	fmt.Printf("🎯 Include scope: %s\n", getBoolDisplay(options.IncludeScope))

	if options.CustomPrompt != "" {
		fmt.Printf("🎨 Custom prompt: %s\n", options.CustomPrompt)
	}

	PrintSeparator()
}

// Helper functions for display
func getLanguageDisplay(lang string) string {
	switch lang {
	case "vi":
		return "🇻🇳 Tiếng Việt"
	case "en":
		return "🇺🇸 English"
	default:
		return "🇺🇸 English"
	}
}

func getStyleDisplay(style types.CommitMessageStyle) string {
	switch style {
	case types.StyleConventional:
		return "🔧 Conventional Commits"
	case types.StyleTraditional:
		return "📋 Traditional"
	case types.StyleDetailed:
		return "📖 Detailed"
	case types.StyleMinimal:
		return "⚡ Minimal"
	default:
		return "🔧 Conventional Commits"
	}
}

func getBoolDisplay(value bool) string {
	if value {
		return "✅ Bật"
	}
	return "❌ Tắt"
}
