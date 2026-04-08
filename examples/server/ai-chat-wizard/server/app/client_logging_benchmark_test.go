package app

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// BenchmarkParseAppendStructuredClientLogFields measures structured field flattening cost.
func BenchmarkParseAppendStructuredClientLogFields(parseB *testing.B) {
	parseFields, parseErr := structpb.NewStruct(map[string]any{
		"route":   "/app",
		"attempt": float64(3),
		"status":  "reconnecting",
		"nested": map[string]any{
			"reason": "idle timeout",
		},
	})
	if parseErr != nil {
		parseB.Fatalf("structpb.NewStruct: %v", parseErr)
	}
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = parseAppendStructuredClientLogFields(nil, parseFields)
	}
}

// BenchmarkParseExtractTraceContextFromContext measures trace metadata extraction overhead.
func BenchmarkParseExtractTraceContextFromContext(parseB *testing.B) {
	parseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		traceParentMetadataKey, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		traceStateMetadataKey, "vendor=relay",
	))
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_, _, _ = parseExtractTraceContextFromContext(parseCtx)
	}
}

// BenchmarkParseResolveClientLogLevel measures log-level mapping cost for relay hot paths.
func BenchmarkParseResolveClientLogLevel(parseB *testing.B) {
	parseLevels := []string{"debug", "info", "warn", "error", "warning", "unknown"}
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = parseResolveClientLogLevel(parseLevels[parseI%len(parseLevels)])
	}
}

// BenchmarkReportClientLog measures end-to-end server-side client log handling without external I/O.
func BenchmarkReportClientLog(parseB *testing.B) {
	parseLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	parseServer := &chatServer{
		logger:       parseLogger,
		clientLogger: parseLogger,
	}
	parseFields, parseErr := structpb.NewStruct(map[string]any{
		"attempt": float64(1),
		"route":   "/app",
	})
	if parseErr != nil {
		parseB.Fatalf("structpb.NewStruct: %v", parseErr)
	}
	parseReq := &chatpb.ReportClientLogRequest{
		ClientId:   uuid.NewString(),
		Level:      "info",
		Scope:      "chat-wizard",
		Message:    "bridge heartbeat",
		Fields:     parseFields,
		ObservedAt: timestamppb.New(time.Now().UTC()),
	}
	parseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		clientMetadataKey, parseReq.GetClientId(),
		traceParentMetadataKey, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
	))

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		if _, parseErr2 := parseServer.ReportClientLog(parseCtx, parseReq); parseErr2 != nil {
			parseB.Fatalf("ReportClientLog: %v", parseErr2)
		}
	}
}
