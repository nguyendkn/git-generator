package formatter

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/nguyendkn/git-generator/pkg/types"
)

// MessageFormatter handles commit message formatting according to Git best practices
type MessageFormatter struct {
	config FormatterConfig
}

// FormatterConfig contains configuration for message formatting
type FormatterConfig struct {
	MaxSubjectLength  int  `json:"max_subject_length"`   // Default: 50
	MaxBodyLineLength int  `json:"max_body_line_length"` // Default: 72
	AutoWrapBody      bool `json:"auto_wrap_body"`       // Default: true
	BreakOnSentence   bool `json:"break_on_sentence"`    // Default: true
	EnforceBlankLine  bool `json:"enforce_blank_line"`   // Default: true
}

// NewMessageFormatter creates a new message formatter with default config
func NewMessageFormatter() *MessageFormatter {
	return &MessageFormatter{
		config: FormatterConfig{
			MaxSubjectLength:  50,
			MaxBodyLineLength: 72,
			AutoWrapBody:      true,
			BreakOnSentence:   true,
			EnforceBlankLine:  true,
		},
	}
}

// NewMessageFormatterWithConfig creates a formatter with custom config
func NewMessageFormatterWithConfig(config FormatterConfig) *MessageFormatter {
	return &MessageFormatter{config: config}
}

// FormatCommitMessage formats a commit message according to Git best practices
func (f *MessageFormatter) FormatCommitMessage(message *types.CommitMessage) string {
	var result strings.Builder

	// Format subject line
	subject := f.formatSubjectLine(message)
	result.WriteString(subject)

	// Add body if present
	if message.Body != "" {
		if f.config.EnforceBlankLine {
			result.WriteString("\n\n")
		} else {
			result.WriteString("\n")
		}

		formattedBody := f.formatBody(message.Body)
		result.WriteString(formattedBody)
	}

	// Add footer if present
	if message.Footer != "" {
		result.WriteString("\n\n")
		result.WriteString(message.Footer)
	}

	return result.String()
}

// formatSubjectLine formats the subject line according to conventional commits
func (f *MessageFormatter) formatSubjectLine(message *types.CommitMessage) string {
	var subject strings.Builder

	// Add type
	if message.Type != "" {
		subject.WriteString(strings.ToLower(string(message.Type)))
	}

	// Add scope if present
	if message.Scope != "" {
		subject.WriteString("(")
		subject.WriteString(message.Scope)
		subject.WriteString(")")
	}

	// Add breaking change indicator
	if message.Breaking {
		subject.WriteString("!")
	}

	// Add separator
	subject.WriteString(": ")

	// Add description
	description := message.Description
	if message.Subject != "" {
		// Use Subject if available (newer format)
		description = message.Subject
	}

	// Ensure first letter is capitalized and remove trailing period
	description = f.capitalizeFirst(description)
	description = strings.TrimSuffix(description, ".")

	// Truncate if too long
	if len(subject.String()+description) > f.config.MaxSubjectLength {
		maxDescLength := f.config.MaxSubjectLength - len(subject.String())
		if maxDescLength > 0 {
			description = f.truncateAtWord(description, maxDescLength)
		}
	}

	subject.WriteString(description)

	return subject.String()
}

// formatBody formats the commit message body with proper line wrapping
func (f *MessageFormatter) formatBody(body string) string {
	if !f.config.AutoWrapBody {
		return body
	}

	// Split into paragraphs
	paragraphs := strings.Split(body, "\n\n")
	var formattedParagraphs []string

	for _, paragraph := range paragraphs {
		if strings.TrimSpace(paragraph) == "" {
			continue
		}

		formattedParagraph := f.formatParagraph(paragraph)
		formattedParagraphs = append(formattedParagraphs, formattedParagraph)
	}

	return strings.Join(formattedParagraphs, "\n\n")
}

// formatParagraph formats a single paragraph with sentence-aware line breaking
func (f *MessageFormatter) formatParagraph(paragraph string) string {
	// Clean up the paragraph
	paragraph = strings.TrimSpace(paragraph)
	paragraph = regexp.MustCompile(`\s+`).ReplaceAllString(paragraph, " ")

	if f.config.BreakOnSentence {
		return f.formatWithSentenceBreaks(paragraph)
	}

	return f.wrapText(paragraph, f.config.MaxBodyLineLength)
}

// formatWithSentenceBreaks formats text with automatic line breaks at sentence endings and bullet points
func (f *MessageFormatter) formatWithSentenceBreaks(text string) string {
	// Split into sentences
	sentences := f.splitIntoSentences(text)
	var lines []string

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		// Add bullet point prefix
		bulletPrefix := "- "
		maxLineLength := f.config.MaxBodyLineLength - len(bulletPrefix)

		// If sentence is too long, wrap it while preserving bullet point
		if len(sentence) > maxLineLength {
			wrappedSentence := f.wrapText(sentence, maxLineLength)
			sentenceLines := strings.Split(wrappedSentence, "\n")

			// Add bullet point to first line
			if len(sentenceLines) > 0 {
				lines = append(lines, bulletPrefix+sentenceLines[0])

				// Add continuation lines with proper indentation
				for i := 1; i < len(sentenceLines); i++ {
					lines = append(lines, "  "+sentenceLines[i]) // 2 spaces for continuation
				}
			}
		} else {
			// Sentence fits on one line
			lines = append(lines, bulletPrefix+sentence)
		}
	}

	return strings.Join(lines, "\n")
}

// splitIntoSentences splits text into sentences
func (f *MessageFormatter) splitIntoSentences(text string) []string {
	// Simple sentence splitting - can be enhanced with more sophisticated logic
	sentenceEnders := regexp.MustCompile(`[.!?]+\s+`)
	parts := sentenceEnders.Split(text, -1)

	var sentences []string
	matches := sentenceEnders.FindAllString(text, -1)

	for i, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}

		sentence := strings.TrimSpace(part)
		if i < len(matches) {
			// Add back the sentence ender
			ender := strings.TrimSpace(matches[i])
			if ender != "" {
				sentence += ender[:1] // Just the punctuation, not the space
			}
		}

		sentences = append(sentences, sentence)
	}

	return sentences
}

// wrapText wraps text to specified line length
func (f *MessageFormatter) wrapText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}

	words := strings.Fields(text)
	var lines []string
	var currentLine strings.Builder

	for _, word := range words {
		// Check if adding this word would exceed line length
		testLine := currentLine.String()
		if testLine != "" {
			testLine += " " + word
		} else {
			testLine = word
		}

		if len(testLine) <= maxLength {
			if currentLine.Len() > 0 {
				currentLine.WriteString(" ")
			}
			currentLine.WriteString(word)
		} else {
			// Start new line
			if currentLine.Len() > 0 {
				lines = append(lines, currentLine.String())
				currentLine.Reset()
			}
			currentLine.WriteString(word)
		}
	}

	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return strings.Join(lines, "\n")
}

// capitalizeFirst capitalizes the first letter of a string
func (f *MessageFormatter) capitalizeFirst(s string) string {
	if s == "" {
		return s
	}

	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// truncateAtWord truncates text at word boundary
func (f *MessageFormatter) truncateAtWord(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}

	// Find last space before maxLength
	truncated := text[:maxLength]
	lastSpace := strings.LastIndex(truncated, " ")

	if lastSpace > 0 {
		return text[:lastSpace]
	}

	// No space found, hard truncate
	return text[:maxLength-3] + "..."
}

// ValidateFormat validates commit message format
func (f *MessageFormatter) ValidateFormat(message *types.CommitMessage) []string {
	var issues []string

	subject := f.formatSubjectLine(message)

	// Check subject length
	if len(subject) > f.config.MaxSubjectLength {
		issues = append(issues, "Subject line quá dài (>50 ký tự)")
	}

	// Check subject ends with period
	if strings.HasSuffix(subject, ".") {
		issues = append(issues, "Subject line không nên kết thúc bằng dấu chấm")
	}

	// Check body line lengths
	if message.Body != "" {
		lines := strings.Split(message.Body, "\n")
		for i, line := range lines {
			if len(line) > f.config.MaxBodyLineLength {
				issues = append(issues, fmt.Sprintf("Dòng %d trong body quá dài (>72 ký tự)", i+1))
			}
		}
	}

	return issues
}
