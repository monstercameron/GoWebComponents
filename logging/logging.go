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
func New(parseScope string) Logger {
	return Logger{scope: strings.TrimSpace(parseScope)}
}

// Scope returns the logger scope.
func (parseLogger Logger) Scope() string {
	return parseLogger.scope
}

// Log writes a structured entry using the provided level.
func (parseLogger Logger) Log(parseLevel, parseMessage string, parseFields Fields) {
	Log(parseLevel, parseLogger.scope, parseMessage, parseFields)
}

// Debug writes a debug-level entry.
func (parseLogger Logger) Debug(parseMessage string, parseFields Fields) {
	parseLogger.Log("debug", parseMessage, parseFields)
}

// Info writes an info-level entry.
func (parseLogger Logger) Info(parseMessage string, parseFields Fields) {
	parseLogger.Log("info", parseMessage, parseFields)
}

// Warn writes a warning entry.
func (parseLogger Logger) Warn(parseMessage string, parseFields Fields) {
	parseLogger.Log("warn", parseMessage, parseFields)
}

// Error writes an error entry.
func (parseLogger Logger) Error(parseMessage string, parseFields Fields) {
	parseLogger.Log("error", parseMessage, parseFields)
}

// Log writes a structured entry to the configured browser console or fallback output.
func Log(parseLevel, parseScope, parseMessage string, parseFields Fields) {
	writeStructured(parseLevel, strings.TrimSpace(parseScope), parseMessage, cloneFields(parseFields))
}

func cloneFields(parseFields Fields) map[string]interface{} {
	if len(parseFields) == 0 {
		return nil
	}
	parseCloned := make(map[string]interface{}, len(parseFields))
	for parseKey, parseValue := range parseFields {
		parseCloned[parseKey] = parseValue
	}
	return parseCloned
}
