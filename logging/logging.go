package logging

import (
	"context"
	"maps"
	"strings"
)

// Fields is the structured payload attached to a log entry.
type Fields = map[string]any

// Logger writes structured entries under a stable scope.
type Logger struct {
	parseContext context.Context
	scope        string
	baseArgs     []any
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
		baseArgs:     parseScopedLogger.baseArgs,
	}
}

// With returns a copy of the logger that attaches the given fields to every subsequent entry, so a
// component name, request id, or handler need not be repeated on each call. Accepts the same forms
// as the Log methods (key/value pairs, a Fields map, or slog.Attr). Per-call fields override With
// fields on a key collision. Mirrors slog.Logger.With.
func (parseScopedLogger Logger) With(parseLogArgs ...any) Logger {
	parseMerged := make([]any, 0, len(parseScopedLogger.baseArgs)+len(parseLogArgs))
	parseMerged = append(parseMerged, parseScopedLogger.baseArgs...)
	parseMerged = append(parseMerged, parseLogArgs...)
	return Logger{
		parseContext: parseScopedLogger.parseContext,
		scope:        parseScopedLogger.scope,
		baseArgs:     parseMerged,
	}
}

// Scope returns the logger scope.
func (parseScopedLogger Logger) Scope() string {
	return parseScopedLogger.scope
}

// Log writes a structured entry using the provided level. Any fields attached via With are merged
// first (so per-call fields override them on a key collision).
func (parseScopedLogger Logger) Log(parseLogLevel, parseLogMessage string, parseLogArgs ...any) {
	if len(parseScopedLogger.baseArgs) > 0 {
		parseLogArgs = append(append([]any{}, parseScopedLogger.baseArgs...), parseLogArgs...)
	}
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
	LogContext(context.Background(), parseLogLevel, parseLogScope, parseLogMessage, parseLogArgs...)
}

// LogContext writes a structured entry while resolving context-backed correlation and trace metadata.
func LogContext(parseLogContext context.Context, parseLogLevel, parseLogScope, parseLogMessage string, parseLogArgs ...any) {
	parseLogFields := buildLogFields(parseLogArgs)
	writeStructuredContext(parseLogContext, parseLogLevel, strings.TrimSpace(parseLogScope), parseLogMessage, parseLogFields)
}

// writeStructured writes one structured entry without context-backed metadata resolution.
func writeStructured(parseLogLevel, parseLogScope, parseLogMessage string, parseLogFields map[string]any) {
	writeStructuredContext(context.Background(), parseLogLevel, parseLogScope, parseLogMessage, parseLogFields)
}

func cloneFields(parseLogFields Fields) map[string]any {
	if len(parseLogFields) == 0 {
		return nil
	}
	parseLogFieldsCloned := make(map[string]any, len(parseLogFields))
	maps.Copy(parseLogFieldsCloned, parseLogFields)
	return parseLogFieldsCloned
}
