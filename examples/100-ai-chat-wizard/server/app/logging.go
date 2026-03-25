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
	serverLogDir      = "log"
	serverLogFilename = "chat-wizard-server.log"
	serverServiceName = "chat-wizard-server"
)

type otelJSONHandler struct {
	next        slog.Handler
	serviceName string
}

func newOTELLogger(writer io.Writer, serviceName string) *slog.Logger {
	base := slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			switch attr.Key {
			case slog.TimeKey:
				attr.Key = "timestamp"
			case slog.LevelKey:
				attr.Key = "severity_text"
				attr.Value = slog.StringValue(strings.ToUpper(attr.Value.String()))
			case slog.MessageKey:
				attr.Key = "body"
			}
			return attr
		},
	})
	return slog.New(&otelJSONHandler{next: base, serviceName: strings.TrimSpace(serviceName)})
}

func newServerLogger() (*slog.Logger, func(), error) {
	if err := os.MkdirAll(serverLogDir, 0o755); err != nil {
		return nil, nil, err
	}
	logPath := filepath.Join(serverLogDir, serverLogFilename)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}

	logger := newOTELLogger(io.MultiWriter(os.Stderr, logFile), serverServiceName)
	closeFn := func() {
		_ = logFile.Close()
	}
	return logger.With(slog.String("log.file.path", logPath)), closeFn, nil
}

func (h *otelJSONHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *otelJSONHandler) Handle(ctx context.Context, record slog.Record) error {
	cloned := record.Clone()
	if strings.TrimSpace(h.serviceName) != "" {
		cloned.AddAttrs(slog.String("service.name", h.serviceName))
	}
	cloned.AddAttrs(slog.Int("severity_number", otelSeverityNumber(record.Level)))

	if record.Level >= slog.LevelError {
		if boundary := errorBoundaryFromMessage(record.Message); boundary != "" {
			cloned.AddAttrs(slog.String("error.boundary", boundary))
		}
		errMessage, errType := errorDetailsFromRecord(record)
		if errMessage == "" {
			errMessage = strings.TrimSpace(record.Message)
		}
		if errMessage != "" {
			cloned.AddAttrs(slog.String("error.message", errMessage))
		}
		if errType != "" {
			cloned.AddAttrs(slog.String("error.type", errType))
		}
	}

	return h.next.Handle(ctx, cloned)
}

func (h *otelJSONHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &otelJSONHandler{
		next:        h.next.WithAttrs(attrs),
		serviceName: h.serviceName,
	}
}

func (h *otelJSONHandler) WithGroup(name string) slog.Handler {
	return &otelJSONHandler{
		next:        h.next.WithGroup(name),
		serviceName: h.serviceName,
	}
}

func otelSeverityNumber(level slog.Level) int {
	switch {
	case level >= slog.LevelError:
		return 17
	case level >= slog.LevelWarn:
		return 13
	case level >= slog.LevelInfo:
		return 9
	default:
		return 5
	}
}

func errorBoundaryFromMessage(message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return ""
	}
	if separatorIndex := strings.Index(trimmed, ":"); separatorIndex > 0 {
		return strings.TrimSpace(trimmed[:separatorIndex])
	}
	return trimmed
}

func errorDetailsFromRecord(record slog.Record) (errorMessage string, errorType string) {
	record.Attrs(func(attr slog.Attr) bool {
		if strings.TrimSpace(attr.Key) != "error" {
			return true
		}
		resolvedAttr := attr.Value.Resolve()
		switch resolvedAttr.Kind() {
		case slog.KindAny:
			anyValue := resolvedAttr.Any()
			if anyValue == nil {
				return false
			}
			if errValue, ok := anyValue.(error); ok {
				errorMessage = strings.TrimSpace(errValue.Error())
				errorType = fmt.Sprintf("%T", errValue)
				return false
			}
			errorMessage = strings.TrimSpace(fmt.Sprint(anyValue))
			errorType = fmt.Sprintf("%T", anyValue)
			return false
		default:
			errorMessage = strings.TrimSpace(resolvedAttr.String())
			if errorMessage == "" {
				errorMessage = strings.TrimSpace(fmt.Sprint(resolvedAttr.Any()))
			}
			return false
		}
	})
	return errorMessage, errorType
}
