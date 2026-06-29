package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBuildSnapshotDiffReportByStableRefs(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseBeforePath := filepath.Join(parseRoot, "before.json")
	parseAfterPath := filepath.Join(parseRoot, "after.json")
	parseBefore := `{"root":{"name":"App","agentRef":"app","children":[{"name":"Button","agentRef":"app/button","text":"Save"},{"name":"Old","agentRef":"app/old"}]}}`
	parseAfter := `{"root":{"name":"App","agentRef":"app","children":[{"name":"Button","agentRef":"app/button","text":"Saved"},{"name":"New","agentRef":"app/new"}]}}`
	if parseErr := os.WriteFile(parseBeforePath, []byte(parseBefore), 0o644); parseErr != nil {
		parseT.Fatalf("write before: %v", parseErr)
	}
	if parseErr := os.WriteFile(parseAfterPath, []byte(parseAfter), 0o644); parseErr != nil {
		parseT.Fatalf("write after: %v", parseErr)
	}
	parseReport, parseErr := buildSnapshotDiffReport(snapshotDiffConfig{beforePath: parseBeforePath, afterPath: parseAfterPath})
	if parseErr != nil {
		parseT.Fatalf("snapshot diff: %v", parseErr)
	}
	if !reflect.DeepEqual(parseReport.Added, []string{"app/new"}) {
		parseT.Fatalf("added = %#v", parseReport.Added)
	}
	if !reflect.DeepEqual(parseReport.Removed, []string{"app/old"}) {
		parseT.Fatalf("removed = %#v", parseReport.Removed)
	}
	if !reflect.DeepEqual(parseReport.Changed, []string{"app", "app/button"}) {
		parseT.Fatalf("changed = %#v", parseReport.Changed)
	}
}
