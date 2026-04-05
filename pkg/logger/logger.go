package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// Level represents the log level
type Level string

const (
	// DebugLevel logs are typically voluminous, and are usually disabled in production
	DebugLevel Level = "debug"
	// InfoLevel is the default logging priority
	InfoLevel Level = "info"
	// WarnLevel logs are more important than Info, but don't need individual human review
	WarnLevel Level = "warn"
	// ErrorLevel logs are high-priority. If an application is running smoothly, it shouldn't generate any error-level logs
	ErrorLevel Level = "error"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Time     string                 `json:"time"`
	Level    Level                  `json:"level"`
	Message  string                 `json:"message"`
	Method   string                 `json:"method,omitempty"`
	Path     string                 `json:"path,omitempty"`
	Status   int                    `json:"status,omitempty"`
	Duration float64                `json:"duration_ms,omitempty"`
	Fields   map[string]interface{} `json:"fields,omitempty"`
}

var (
	logger    *log.Logger
	verbose   bool
	mu        sync.Mutex
	output    io.Writer
	jsonOutput bool
)

func init() {
	output = os.Stdout
	logger = log.New(output, "", 0)

	// Check for verbose flag from environment
	if os.Getenv("VERBOSE") != "" || os.Getenv("DEBUG") != "" {
		verbose = true
	}

	// Check for JSON output flag from environment
	if os.Getenv("LOG_JSON") != "" || os.Getenv("JSON_LOGS") != "" {
		jsonOutput = true
	}
}

// SetOutput sets the output destination for logs
func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	output = w
	logger = log.New(output, "", 0)
}

// SetVerbose sets the verbose flag
func SetVerbose(v bool) {
	mu.Lock()
	defer mu.Unlock()
	verbose = v
}

// IsVerbose returns the current verbose setting
func IsVerbose() bool {
	mu.Lock()
	defer mu.Unlock()
	return verbose
}

// SetJSONOutput enables or disables JSON structured logging
func SetJSONOutput(enabled bool) {
	mu.Lock()
	defer mu.Unlock()
	jsonOutput = enabled
}

// logEntry writes a log entry
func logEntry(level Level, message string, fields map[string]interface{}) {
	mu.Lock()
	defer mu.Unlock()

	entry := LogEntry{
		Time:    time.Now().UTC().Format(time.RFC3339),
		Level:   level,
		Message: message,
	}

	if fields != nil {
		entry.Fields = fields
	}

	if jsonOutput {
		jsonData, err := json.Marshal(entry)
		if err != nil {
			logger.Printf(`{"time":"%s","level":"%s","message":"failed to marshal log entry: %v"}`,
				entry.Time, entry.Level, err)
			return
		}
		logger.Println(string(jsonData))
	} else {
		// Plain text format for backwards compatibility
		prefix := fmt.Sprintf("[%s] %s: ", entry.Time, entry.Level)
		logger.Print(prefix + message)
	}
}

// Info logs an info message
func Info(format string, v ...interface{}) {
	logEntry(InfoLevel, fmt.Sprintf(format, v...), nil)
}

// InfoWithFields logs an info message with additional fields
func InfoWithFields(fields map[string]interface{}, format string, v ...interface{}) {
	logEntry(InfoLevel, fmt.Sprintf(format, v...), fields)
}

// Error logs an error message
func Error(format string, v ...interface{}) {
	logEntry(ErrorLevel, fmt.Sprintf(format, v...), nil)
}

// ErrorWithFields logs an error message with additional fields
func ErrorWithFields(fields map[string]interface{}, format string, v ...interface{}) {
	logEntry(ErrorLevel, fmt.Sprintf(format, v...), fields)
}

// Warn logs a warning message
func Warn(format string, v ...interface{}) {
	logEntry(WarnLevel, fmt.Sprintf(format, v...), nil)
}

// WarnWithFields logs a warning message with additional fields
func WarnWithFields(fields map[string]interface{}, format string, v ...interface{}) {
	logEntry(WarnLevel, fmt.Sprintf(format, v...), fields)
}

// Debug logs a debug message
func Debug(format string, v ...interface{}) {
	if !verbose {
		return
	}
	logEntry(DebugLevel, fmt.Sprintf(format, v...), nil)
}

// DebugWithFields logs a debug message with additional fields
func DebugWithFields(fields map[string]interface{}, format string, v ...interface{}) {
	if !verbose {
		return
	}
	logEntry(DebugLevel, fmt.Sprintf(format, v...), fields)
}

// Fatal logs a fatal message and exits
func Fatal(format string, v ...interface{}) {
	logEntry(ErrorLevel, fmt.Sprintf(format, v...), nil)
	os.Exit(1)
}

// LogHTTPRequest logs an HTTP request with method, path, status, and duration
func LogHTTPRequest(method, path string, status int, duration float64) {
	// Determine log level based on status code
	var level Level
	switch {
	case status >= 500:
		level = ErrorLevel
	case status >= 400:
		level = WarnLevel
	default:
		level = InfoLevel
	}

	mu.Lock()
	defer mu.Unlock()

	entry := LogEntry{
		Time:     time.Now().UTC().Format(time.RFC3339),
		Level:    level,
		Message:  "HTTP request",
		Method:   method,
		Path:     path,
		Status:   status,
		Duration: duration,
	}

	if jsonOutput {
		jsonData, err := json.Marshal(entry)
		if err != nil {
			logger.Printf(`{"time":"%s","level":"%s","message":"failed to marshal HTTP log entry: %v"}`,
				entry.Time, entry.Level, err)
			return
		}
		logger.Println(string(jsonData))
	} else {
		logger.Printf("[%s] %s: %s %s - Status: %d - Duration: %.2fms",
			entry.Time, entry.Level, method, path, status, duration)
	}
}
