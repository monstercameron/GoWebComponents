package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// BenchmarkParseReadTailLines measures bounded reverse-tail reads for large diagnostic logs.
func BenchmarkParseReadTailLines(parseB *testing.B) {
	parseTempDir := parseB.TempDir()
	parseLogPath := filepath.Join(parseTempDir, "chat-wizard-server.log")
	parseLogBuilder := strings.Builder{}
	for range 3000 {
		parseLogBuilder.WriteString(`{"timestamp":"2026-03-27T14:10:00Z","message":"line `)
		parseLogBuilder.WriteString(strings.Repeat("x", 12))
		parseLogBuilder.WriteString(`"}` + "\n")
	}
	if parseErr := os.WriteFile(parseLogPath, []byte(parseLogBuilder.String()), 0o644); parseErr != nil {
		parseB.Fatalf("WriteFile: %v", parseErr)
	}

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseLines, parseErr2 := parseReadTailLines(parseLogPath, 500, logTailMaximumReadBytes)
		if parseErr2 != nil {
			parseB.Fatalf("parseReadTailLines: %v", parseErr2)
		}
		if len(parseLines) == 0 {
			parseB.Fatal("expected non-empty lines")
		}
	}
}

// BenchmarkParseFilterLogTailLines measures case-insensitive substring filtering cost.
func BenchmarkParseFilterLogTailLines(parseB *testing.B) {
	parseLines := make([]parseLogTailLine, 0, 1500)
	for parseI := range 1500 {
		parseValue := "info request completed"
		if parseI%4 == 0 {
			parseValue = "error request failed"
		}
		parseLines = append(parseLines, parseLogTailLine{parseLine: parseValue})
	}

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseFiltered := parseFilterLogTailLines(parseLines, "error")
		if len(parseFiltered) == 0 {
			parseB.Fatal("expected filtered lines")
		}
	}
}
