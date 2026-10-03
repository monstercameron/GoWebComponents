package app

import (
	"log/slog"
	"testing"
	"time"
)

// BenchmarkParseDeriveErrorBoundaryFromRecord measures record boundary derivation cost.
func BenchmarkParseDeriveErrorBoundaryFromRecord(parseB *testing.B) {
	parseRecord := slog.NewRecord(time.Now(), slog.LevelError, "send failed", 0)
	parseRecord.AddAttrs(slog.String("log.scope", "chat-wizard"))

	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = parseDeriveErrorBoundaryFromRecord(parseRecord)
	}
}

// BenchmarkParseRecordHasAttrKey measures attr-key presence checks used during error enrichment dedupe.
func BenchmarkParseRecordHasAttrKey(parseB *testing.B) {
	parseRecord := slog.NewRecord(time.Now(), slog.LevelError, "rpc.Send: provider stream error", 0)
	parseRecord.AddAttrs(
		slog.String("log.scope", "chat-wizard"),
		slog.String("error", "provider timeout"),
	)

	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = parseRecordHasAttrKey(parseRecord, "error.boundary")
	}
}
