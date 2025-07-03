package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Level represents log levels
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

// String returns the string representation of the log level
func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging functionality
type Logger struct {
	level      Level
	fileLogger *log.Logger
	file       *os.File
}

// NewLogger creates a new logger instance
func NewLogger(level Level, logToFile bool) (*Logger, error) {
	logger := &Logger{
		level: level,
	}

	if logToFile {
		if err := logger.setupFileLogging(); err != nil {
			return nil, fmt.Errorf("failed to setup file logging: %w", err)
		}
	}

	return logger, nil
}

// setupFileLogging sets up logging to a file
func (l *Logger) setupFileLogging() error {
	// Create logs directory in user's home
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	logDir := filepath.Join(homeDir, ".git-generator", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02")
	logFile := filepath.Join(logDir, fmt.Sprintf("git-generator-%s.log", timestamp))

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	l.file = file
	l.fileLogger = log.New(file, "", log.LstdFlags)

	return nil
}

// Close closes the logger and any open files
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// shouldLog checks if a message should be logged based on level
func (l *Logger) shouldLog(level Level) bool {
	return level >= l.level
}

// log writes a log message
func (l *Logger) log(level Level, format string, args ...interface{}) {
	if !l.shouldLog(level) {
		return
	}

	message := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("[%s] %s: %s", timestamp, level.String(), message)

	// Always log to stderr for errors and warnings
	if level >= WARN {
		fmt.Fprintln(os.Stderr, logLine)
	} else if level == INFO {
		fmt.Println(logLine)
	}

	// Log to file if file logging is enabled
	if l.fileLogger != nil {
		l.fileLogger.Printf("%s: %s", level.String(), message)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// Global logger instance
var globalLogger *Logger

// InitGlobalLogger initializes the global logger
func InitGlobalLogger(level Level, logToFile bool) error {
	var err error
	globalLogger, err = NewLogger(level, logToFile)
	return err
}

// CloseGlobalLogger closes the global logger
func CloseGlobalLogger() error {
	if globalLogger != nil {
		return globalLogger.Close()
	}
	return nil
}

// Debug logs a debug message using the global logger
func Debug(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Debug(format, args...)
	}
}

// Info logs an info message using the global logger
func Info(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Info(format, args...)
	}
}

// Warn logs a warning message using the global logger
func Warn(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Warn(format, args...)
	}
}

// Error logs an error message using the global logger
func Error(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Error(format, args...)
	}
}

// ErrorWithDetails logs an error with additional context
func ErrorWithDetails(err error, context string, details map[string]interface{}) {
	if globalLogger == nil {
		return
	}

	message := fmt.Sprintf("%s: %v", context, err)
	if len(details) > 0 {
		message += " | Details: "
		for key, value := range details {
			message += fmt.Sprintf("%s=%v ", key, value)
		}
	}

	globalLogger.Error("%s", message)
}

// UserError represents an error that should be displayed to the user
type UserError struct {
	Message string
	Cause   error
}

// Error implements the error interface
func (e *UserError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *UserError) Unwrap() error {
	return e.Cause
}

// NewUserError creates a new user-friendly error
func NewUserError(message string, cause error) *UserError {
	return &UserError{
		Message: message,
		Cause:   cause,
	}
}

// HandleError handles an error appropriately based on its type
func HandleError(err error) {
	if err == nil {
		return
	}

	if userErr, ok := err.(*UserError); ok {
		// User-friendly error
		fmt.Fprintf(os.Stderr, "Error: %s\n", userErr.Message)
		if userErr.Cause != nil {
			Error("User error caused by: %v", userErr.Cause)
		}
	} else {
		// Technical error
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		Error("Technical error: %v", err)
	}
}
