package logging

import (
	"strings"
)

// Fields is the structured payload attached to a log entry.
type Fields = map[string]interface{}

// Logger writes structured entries under a stable scope.
type Logger struct {
	scope string
}

// New returns a scoped logger for application code.
func New(scope string) Logger {
	return Logger{scope: strings.TrimSpace(scope)}
}

// Scope returns the logger scope.
func (logger Logger) Scope() string {
	return logger.scope
}

// Log writes a structured entry using the provided level.
func (logger Logger) Log(level, message string, fields Fields) {
	Log(level, logger.scope, message, fields)
}

// Debug writes a debug-level entry.
func (logger Logger) Debug(message string, fields Fields) {
	logger.Log("debug", message, fields)
}

// Info writes an info-level entry.
func (logger Logger) Info(message string, fields Fields) {
	logger.Log("info", message, fields)
}

// Warn writes a warning entry.
func (logger Logger) Warn(message string, fields Fields) {
	logger.Log("warn", message, fields)
}

// Error writes an error entry.
func (logger Logger) Error(message string, fields Fields) {
	logger.Log("error", message, fields)
}

// Log writes a structured entry to the configured browser console or fallback output.
func Log(level, scope, message string, fields Fields) {
	writeStructured(level, strings.TrimSpace(scope), message, cloneFields(fields))
}

func cloneFields(fields Fields) map[string]interface{} {
	if len(fields) == 0 {
		return nil
	}
	cloned := make(map[string]interface{}, len(fields))
	for key, value := range fields {
		cloned[key] = value
	}
	return cloned
}