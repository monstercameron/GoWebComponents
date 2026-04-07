package logging

import (
	"context"
	"strings"
)

// Fields is the structured payload attached to a log entry.
type Fields = map[string]interface{}

// Logger writes structured entries under a stable scope.
type Logger struct {
	parseContext context.Context
	scope        string
}

// New returns a scoped logger for application code.
func New(parseLogScope string) Logger {
	return Logger{scope: strings.TrimSpace(parseLogScope)}
}

// NewContext returns a scoped logger that resolves correlation and trace metadata from context.
func NewContext(parseLogContext context.Context, parseLogScope string) Logger {
	return Logger{
		parseContext: parseLogContext,
		scope:        strings.TrimSpace(parseLogScope),
	}
}

// WithContext returns one scoped logger that resolves correlation and trace metadata from context.
func (parseScopedLogger Logger) WithContext(parseLogContext context.Context) Logger {
	return Logger{
		parseContext: parseLogContext,
		scope:        parseScopedLogger.scope,
	}
}

// Scope returns the logger scope.
func (parseScopedLogger Logger) Scope() string {
	return parseScopedLogger.scope
}

// Log writes a structured entry using the provided level.
func (parseScopedLogger Logger) Log(parseLogLevel, parseLogMessage string, parseLogArgs ...any) {
	LogContext(parseScopedLogger.parseContext, parseLogLevel, parseScopedLogger.scope, parseLogMessage, parseLogArgs...)
}

// Debug writes a debug-level entry.
func (parseScopedLogger Logger) Debug(parseLogMessage string, parseLogArgs ...any) {
	parseScopedLogger.Log("debug", parseLogMessage, parseLogArgs...)
}

// Info writes an info-level entry.
func (parseScopedLogger Logger) Info(parseLogMessage string, parseLogArgs ...any) {
	parseScopedLogger.Log("info", parseLogMessage, parseLogArgs...)
}

// Warn writes a warning entry.
func (parseScopedLogger Logger) Warn(parseLogMessage string, parseLogArgs ...any) {
	parseScopedLogger.Log("warn", parseLogMessage, parseLogArgs...)
}

// Error writes an error entry.
func (parseScopedLogger Logger) Error(parseLogMessage string, parseLogArgs ...any) {
	parseScopedLogger.Log("error", parseLogMessage, parseLogArgs...)
}

// Log writes a structured entry to the configured browser console or fallback output.
func Log(parseLogLevel, parseLogScope, parseLogMessage string, parseLogArgs ...any) {
	LogContext(nil, parseLogLevel, parseLogScope, parseLogMessage, parseLogArgs...)
}

// LogContext writes a structured entry while resolving context-backed correlation and trace metadata.
func LogContext(parseLogContext context.Context, parseLogLevel, parseLogScope, parseLogMessage string, parseLogArgs ...any) {
	parseLogFields := buildLogFields(parseLogArgs)
	writeStructuredContext(parseLogContext, parseLogLevel, strings.TrimSpace(parseLogScope), parseLogMessage, parseLogFields)
}

// writeStructured writes one structured entry without context-backed metadata resolution.
func writeStructured(parseLogLevel, parseLogScope, parseLogMessage string, parseLogFields map[string]interface{}) {
	writeStructuredContext(nil, parseLogLevel, parseLogScope, parseLogMessage, parseLogFields)
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
