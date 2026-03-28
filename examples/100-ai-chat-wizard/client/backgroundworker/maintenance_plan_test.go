package main

import (
	"testing"
	"time"
)

// TestParseBuildBackgroundWorkerMaintenanceTaskBatch verifies one maintenance loop tick emits all required maintenance task classes.
func TestParseBuildBackgroundWorkerMaintenanceTaskBatch(parseT *testing.T) {
	parseNow := time.Date(2026, time.March, 28, 15, 4, 5, 0, time.UTC)
	parseBatch := parseBuildBackgroundWorkerMaintenanceTaskBatch(backgroundWorkerMaintenanceCommand{
		IntervalMs:  45000,
		ScopeKey:    "scope-a",
		Locale:      "en",
		IsOnline:    true,
		IsGRPCReady: true,
	}, parseNow)
	if len(parseBatch.Tasks) != 5 {
		parseT.Fatalf("expected 5 maintenance tasks, got %d", len(parseBatch.Tasks))
	}
	parseExpected := map[string]bool{
		backgroundWorkerMaintenanceTaskLocaleRefreshCheck:  false,
		backgroundWorkerMaintenanceTaskUndeliveredLogDrain: false,
		backgroundWorkerMaintenanceTaskUnsentMessageRetry:  false,
		backgroundWorkerMaintenanceTaskLowPriorityPersist:  false,
		backgroundWorkerMaintenanceTaskStaleRecordSweep:    false,
	}
	for _, parseTask := range parseBatch.Tasks {
		if _, isParseKnown := parseExpected[parseTask.TaskType]; !isParseKnown {
			parseT.Fatalf("unexpected task type %q", parseTask.TaskType)
		}
		parseExpected[parseTask.TaskType] = true
		if parseTask.ScopeKey != "scope-a" {
			parseT.Fatalf("expected scope-a, got %+v", parseTask)
		}
		if parseTask.CreatedAt == "" {
			parseT.Fatalf("expected created timestamp, got %+v", parseTask)
		}
	}
	for parseTaskType, isParseSeen := range parseExpected {
		if !isParseSeen {
			parseT.Fatalf("missing required maintenance task type %q", parseTaskType)
		}
	}
	if parseBatch.EmittedAt == "" {
		parseT.Fatalf("expected emitted timestamp, got %+v", parseBatch)
	}
}

// TestParseNormalizeBackgroundWorkerMaintenanceCommand verifies interval and trimmed field defaults.
func TestParseNormalizeBackgroundWorkerMaintenanceCommand(parseT *testing.T) {
	parseCommand := parseNormalizeBackgroundWorkerMaintenanceCommand(backgroundWorkerMaintenanceCommand{
		IntervalMs: 0,
		ScopeKey:   " scope-a ",
		Locale:     " en ",
	})
	if parseCommand.IntervalMs != 60000 {
		parseT.Fatalf("expected default interval 60000, got %d", parseCommand.IntervalMs)
	}
	if parseCommand.ScopeKey != "scope-a" || parseCommand.Locale != "en" {
		parseT.Fatalf("expected trimmed command fields, got %+v", parseCommand)
	}
}
