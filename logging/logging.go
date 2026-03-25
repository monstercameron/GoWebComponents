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
func New(parseLogScope string) Logger {
	return Logger{scope: strings.TrimSpace(parseLogScope)}
}

// Scope returns the logger scope.
func (parseScopedLogger Logger) Scope() string {
	return parseScopedLogger.scope
}

// Log writes a structured entry using the provided level.
func (parseScopedLogger Logger) Log(parseLogLevel, parseLogMessage string, parseLogFields Fields) {
	Log(parseLogLevel, parseScopedLogger.scope, parseLogMessage, parseLogFields)
}

// Debug writes a debug-level entry.
func (parseScopedLogger Logger) Debug(parseLogMessage string, parseLogFields Fields) {
	parseScopedLogger.Log("debug", parseLogMessage, parseLogFields)
}

// Info writes an info-level entry.
func (parseScopedLogger Logger) Info(parseLogMessage string, parseLogFields Fields) {
	parseScopedLogger.Log("info", parseLogMessage, parseLogFields)
}

// Warn writes a warning entry.
func (parseScopedLogger Logger) Warn(parseLogMessage string, parseLogFields Fields) {
	parseScopedLogger.Log("warn", parseLogMessage, parseLogFields)
}

// Error writes an error entry.
func (parseScopedLogger Logger) Error(parseLogMessage string, parseLogFields Fields) {
	parseScopedLogger.Log("error", parseLogMessage, parseLogFields)
}

// Log writes a structured entry to the configured browser console or fallback output.
func Log(parseLogLevel, parseLogScope, parseLogMessage string, parseLogFields Fields) {
	writeStructured(parseLogLevel, strings.TrimSpace(parseLogScope), parseLogMessage, cloneFields(parseLogFields))
}

func cloneFields(parseLogFields Fields) map[string]interface{} {
	if len(parseLogFields) == 0 {
		return nil
	}
	parseLogFieldsCloned := make(map[string]interface{}, len(parseLogFields))
	for parseFieldKey, parseFieldValue := range parseLogFields {
		parseLogFieldsCloned[parseFieldKey] = parseFieldValue
	}
	return parseLogFieldsCloned
}
