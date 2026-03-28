package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const (
	serverLogDir         = "bin/runtime/logs"
	serverLogFilename    = "chat-wizard-server.log"
	clientLogFilename    = "chat-wizard-client.log"
	serverServiceName    = "chat-wizard-server"
	clientLogServiceName = "chat-wizard-client"
)

type otelJSONHandler struct {
	next        slog.Handler
	serviceName string
}

type serverLoggers struct {
	parseServerLogger *slog.Logger
	parseClientLogger *slog.Logger
	parseClose        func()
}

func parseNewOTELLogger(parseWriter io.Writer, parseServiceName string) *slog.Logger {
	parseBase := slog.NewJSONHandler(parseWriter, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
		ReplaceAttr: func(_ []string, parseAttr slog.Attr) slog.Attr {
			switch parseAttr.Key {
			case slog.TimeKey:
				parseAttr.Key = "timestamp"
			case slog.LevelKey:
				parseAttr.Key = "severity_text"
				parseAttr.Value = slog.StringValue(strings.ToUpper(parseAttr.Value.String()))
			case slog.MessageKey:
				parseAttr.Key = "message"
			}
			return parseAttr
		},
	})
	return slog.New(&otelJSONHandler{next: parseBase, serviceName: strings.TrimSpace(parseServiceName)})
}

// parseNewServerLoggers builds dedicated server and client loggers with
// separate file sinks under one lifecycle close function.
func parseNewServerLoggers() (serverLoggers, error) {
	if parseErr := os.MkdirAll(serverLogDir, 0o755); parseErr != nil {
		return serverLoggers{}, parseErr
	}

	parseServerLogPath := filepath.Join(serverLogDir, serverLogFilename)
	parseServerLogFile, parseErr := os.OpenFile(parseServerLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if parseErr != nil {
		return serverLoggers{}, parseErr
	}
	parseClientLogPath := filepath.Join(serverLogDir, clientLogFilename)
	parseClientLogFile, parseErr2 := os.OpenFile(parseClientLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if parseErr2 != nil {
		_ = parseServerLogFile.Close()
		return serverLoggers{}, parseErr2
	}

	parseServerLogger := parseNewOTELLogger(io.MultiWriter(os.Stderr, parseServerLogFile), serverServiceName).
		With(
			slog.String("log.file.path", parseServerLogPath),
			slog.String("log.source", "server"),
			slog.String("event.dataset", "chat-wizard.server"),
		)
	parseClientLogger := parseNewOTELLogger(parseClientLogFile, clientLogServiceName).
		With(
			slog.String("log.file.path", parseClientLogPath),
			slog.String("log.source", "client"),
			slog.String("event.dataset", "chat-wizard.client"),
		)

	return serverLoggers{
		parseServerLogger: parseServerLogger,
		parseClientLogger: parseClientLogger,
		parseClose: func() {
			_ = parseClientLogFile.Close()
			_ = parseServerLogFile.Close()
		},
	}, nil
}

func parseNewServerLogger() (*slog.Logger, func(), error) {
	parseLoggers, parseErr := parseNewServerLoggers()
	if parseErr != nil {
		return nil, nil, parseErr
	}
	return parseLoggers.parseServerLogger, parseLoggers.parseClose, nil
}

func (parseH *otelJSONHandler) Enabled(parseCtx context.Context, parseLevel slog.Level) bool {
	return parseH.next.Enabled(parseCtx, parseLevel)
}

func (parseH *otelJSONHandler) Handle(parseCtx context.Context, parseRecord slog.Record) error {
	parseCloned := parseRecord.Clone()
	if strings.TrimSpace(parseH.serviceName) != "" {
		parseCloned.AddAttrs(slog.String("service.name", parseH.serviceName))
	}
	parseCloned.AddAttrs(slog.Int("severity_number", parseOtelSeverityNumber(parseRecord.Level)))

	if parseRecord.Level >= slog.LevelError {
		if !parseRecordHasAttrKey(parseRecord, "error.boundary") {
			if parseBoundary := parseDeriveErrorBoundaryFromRecord(parseRecord); parseBoundary != "" {
				parseCloned.AddAttrs(slog.String("error.boundary", parseBoundary))
			}
		}
		parseErrorMessage, parseErrorType := parseErrorDetailsFromRecord(parseRecord)
		if !parseRecordHasAttrKey(parseRecord, "error.message") {
			if parseErrorMessage == "" {
				parseErrorMessage = strings.TrimSpace(parseRecord.Message)
			}
			if parseErrorMessage != "" {
				parseCloned.AddAttrs(slog.String("error.message", parseErrorMessage))
			}
		}
		if !parseRecordHasAttrKey(parseRecord, "error.type") && parseErrorType != "" {
			parseCloned.AddAttrs(slog.String("error.type", parseErrorType))
		}
	}

	return parseH.next.Handle(parseCtx, parseCloned)
}

// parseDeriveErrorBoundaryFromRecord resolves one stable error boundary for a record.
func parseDeriveErrorBoundaryFromRecord(parseRecord slog.Record) string {
	if parseBoundary := parseErrorBoundaryFromMessage(parseRecord.Message); parseBoundary != "" {
		return parseBoundary
	}
	if parseScope := parseRecordStringAttrValue(parseRecord, "log.scope"); parseScope != "" {
		return parseScope
	}
	if parseRPCMethod := parseRecordStringAttrValue(parseRecord, "rpc.method"); parseRPCMethod != "" {
		return parseRPCMethod
	}
	return ""
}

// parseRecordHasAttrKey reports whether the record already contains the given attribute key.
func parseRecordHasAttrKey(parseRecord slog.Record, parseKey string) bool {
	parseKey = strings.TrimSpace(parseKey)
	if parseKey == "" {
		return false
	}
	hasParseAttrKey := false
	parseRecord.Attrs(func(parseAttr slog.Attr) bool {
		if strings.TrimSpace(parseAttr.Key) == parseKey {
			hasParseAttrKey = true
			return false
		}
		return true
	})
	return hasParseAttrKey
}

// parseRecordStringAttrValue returns one trimmed string representation for the requested attribute key.
func parseRecordStringAttrValue(parseRecord slog.Record, parseKey string) string {
	parseKey = strings.TrimSpace(parseKey)
	if parseKey == "" {
		return ""
	}
	parseStringValue := ""
	parseRecord.Attrs(func(parseAttr slog.Attr) bool {
		if strings.TrimSpace(parseAttr.Key) != parseKey {
			return true
		}
		parseResolvedValue := parseAttr.Value.Resolve()
		parseStringValue = strings.TrimSpace(parseResolvedValue.String())
		if parseStringValue == "" {
			parseStringValue = strings.TrimSpace(fmt.Sprint(parseResolvedValue.Any()))
		}
		return false
	})
	return parseStringValue
}

func (parseH *otelJSONHandler) WithAttrs(parseAttrs []slog.Attr) slog.Handler {
	return &otelJSONHandler{
		next:        parseH.next.WithAttrs(parseAttrs),
		serviceName: parseH.serviceName,
	}
}

func (parseH *otelJSONHandler) WithGroup(parseName string) slog.Handler {
	return &otelJSONHandler{
		next:        parseH.next.WithGroup(parseName),
		serviceName: parseH.serviceName,
	}
}

func parseOtelSeverityNumber(parseLevel slog.Level) int {
	switch {
	case parseLevel >= slog.LevelError:
		return 17
	case parseLevel >= slog.LevelWarn:
		return 13
	case parseLevel >= slog.LevelInfo:
		return 9
	default:
		return 5
	}
}

func parseErrorBoundaryFromMessage(parseMessage string) string {
	parseTrimmed := strings.TrimSpace(parseMessage)
	if parseTrimmed == "" {
		return ""
	}
	if parseSeparatorIndex := strings.Index(parseTrimmed, ":"); parseSeparatorIndex > 0 {
		return strings.TrimSpace(parseTrimmed[:parseSeparatorIndex])
	}
	if strings.Contains(parseTrimmed, " ") {
		return ""
	}
	return parseTrimmed
}

func parseErrorDetailsFromRecord(parseRecord slog.Record) (parseErrorMessage string, parseErrorType string) {
	parseRecord.Attrs(func(parseAttr slog.Attr) bool {
		if strings.TrimSpace(parseAttr.Key) != "error" {
			return true
		}
		parseResolvedAttr := parseAttr.Value.Resolve()
		switch parseResolvedAttr.Kind() {
		case slog.KindAny:
			parseAnyValue := parseResolvedAttr.Any()
			if parseAnyValue == nil {
				return false
			}
			if parseErrValue, parseOk := parseAnyValue.(error); parseOk {
				parseErrorMessage = strings.TrimSpace(parseErrValue.Error())
				parseErrorType = fmt.Sprintf("%T", parseErrValue)
				return false
			}
			parseErrorMessage = strings.TrimSpace(fmt.Sprint(parseAnyValue))
			parseErrorType = fmt.Sprintf("%T", parseAnyValue)
			return false
		default:
			parseErrorMessage = strings.TrimSpace(parseResolvedAttr.String())
			if parseErrorMessage == "" {
				parseErrorMessage = strings.TrimSpace(fmt.Sprint(parseResolvedAttr.Any()))
			}
			return false
		}
	})
	return parseErrorMessage, parseErrorType
}
