package ui

import (
	"fmt"
	"strings"

	"github.com/common-nighthawk/go-figure"
)

// Colors for terminal output
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
)

// ShowBanner displays the application banner with figlet
func ShowBanner(version string) {
	// Create figlet banner
	banner := figure.NewFigure("Git Generator", "slant", true)

	// Print banner in cyan color
	fmt.Printf("%s%s%s\n", ColorCyan, banner.String(), ColorReset)

	// Print author information
	fmt.Printf("%s%s%s\n", ColorBold, strings.Repeat("=", 60), ColorReset)
	fmt.Printf("%s%sAuthor:%s Dao Khoi Nguyen - dknguyen2304@gmail.com%s\n",
		ColorBold, ColorGreen, ColorWhite, ColorReset)
	fmt.Printf("%s%sVersion:%s %s - AI-Powered Git Commit Message Generator%s\n",
		ColorBold, ColorBlue, ColorWhite, version, ColorReset)
	fmt.Printf("%s%s%s\n\n", ColorBold, strings.Repeat("=", 60), ColorReset)
}

// ShowWelcomeMessage displays a welcome message in Vietnamese
func ShowWelcomeMessage() {
	fmt.Printf("%s🚀 Chào mừng bạn đến với Git Generator!%s\n", ColorGreen, ColorReset)
	fmt.Printf("%s💡 Công cụ tạo commit message thông minh với AI%s\n\n", ColorYellow, ColorReset)
}

// ShowSuccessMessage displays a success message
func ShowSuccessMessage(message string) {
	fmt.Printf("%s✅ %s%s\n", ColorGreen, message, ColorReset)
}

// ShowErrorMessage displays an error message
func ShowErrorMessage(message string) {
	fmt.Printf("%s❌ %s%s\n", ColorRed, message, ColorReset)
}

// ShowWarningMessage displays a warning message
func ShowWarningMessage(message string) {
	fmt.Printf("%s⚠️  %s%s\n", ColorYellow, message, ColorReset)
}

// ShowInfoMessage displays an info message
func ShowInfoMessage(message string) {
	fmt.Printf("%s💡 %s%s\n", ColorBlue, message, ColorReset)
}

// PrintSeparator prints a visual separator
func PrintSeparator() {
	fmt.Printf("%s%s%s\n", ColorPurple, strings.Repeat("-", 50), ColorReset)
}

// PrintHeader prints a section header
func PrintHeader(title string) {
	fmt.Printf("\n%s%s=== %s ===%s\n", ColorBold, ColorCyan, title, ColorReset)
}

// PrintSubHeader prints a subsection header
func PrintSubHeader(title string) {
	fmt.Printf("\n%s%s--- %s ---%s\n", ColorBold, ColorBlue, title, ColorReset)
}
