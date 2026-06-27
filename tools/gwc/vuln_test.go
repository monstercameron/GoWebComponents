package main

import (
	"strings"
	"testing"
)

// govulncheckStream is a representative govulncheck -json stream: two OSV entries, one
// reachable (its finding's trace names a called function) and one imported-only (the
// trace's most specific frame has no function).
const govulncheckStream = `
{"osv":{"id":"GO-2024-0001","summary":"Reachable bug in foo"}}
{"osv":{"id":"GO-2024-0002","summary":"Imported-only bug in bar"}}
{"progress":{"message":"scanning"}}
{"finding":{"osv":"GO-2024-0001","trace":[{"module":"foo","package":"foo","function":"Vulnerable"}]}}
{"finding":{"osv":"GO-2024-0002","trace":[{"module":"bar","package":"bar"}]}}
{"finding":{"osv":"GO-2024-0001","trace":[{"module":"foo","package":"foo","function":"AlsoVulnerable"}]}}
`

// TestParseVulnFindingsClassifiesReachability proves findings are deduplicated by OSV id,
// paired with their summaries, and classified reachable when any trace names a function.
func TestParseVulnFindingsClassifiesReachability(parseT *testing.T) {
	parseReport, parseErr := parseVulnFindings(strings.NewReader(govulncheckStream))
	if parseErr != nil {
		parseT.Fatalf("parseVulnFindings: %v", parseErr)
	}
	if len(parseReport.Entries) != 2 {
		parseT.Fatalf("expected 2 deduplicated entries, got %d: %+v", len(parseReport.Entries), parseReport.Entries)
	}

	parseByID := map[string]vulnEntry{}
	for _, parseEntry := range parseReport.Entries {
		parseByID[parseEntry.ID] = parseEntry
	}
	if !parseByID["GO-2024-0001"].Reachable {
		parseT.Fatal("GO-2024-0001 should be reachable (trace names a function)")
	}
	if parseByID["GO-2024-0002"].Reachable {
		parseT.Fatal("GO-2024-0002 should be imported-only (no function in trace)")
	}
	if parseByID["GO-2024-0001"].Summary != "Reachable bug in foo" {
		parseT.Fatalf("summary not paired: %+v", parseByID["GO-2024-0001"])
	}
	if parseGot := parseReport.Reachable(); len(parseGot) != 1 || parseGot[0].ID != "GO-2024-0001" {
		parseT.Fatalf("expected one reachable entry GO-2024-0001, got %+v", parseGot)
	}
}

// TestParseVulnFindingsCleanStream proves a stream with no findings yields an empty report.
func TestParseVulnFindingsCleanStream(parseT *testing.T) {
	parseReport, parseErr := parseVulnFindings(strings.NewReader(`{"progress":{"message":"no vulnerabilities"}}` + "\n"))
	if parseErr != nil {
		parseT.Fatalf("parseVulnFindings: %v", parseErr)
	}
	if len(parseReport.Entries) != 0 {
		parseT.Fatalf("expected no entries, got %+v", parseReport.Entries)
	}
}

// TestReportVulnVerdicts proves the build-failure policy: reachable always fails; imported
// passes unless -strict.
func TestReportVulnVerdicts(parseT *testing.T) {
	parseClean := vulnReport{}
	if parseErr := reportVuln(parseClean, false); parseErr != nil {
		parseT.Fatalf("clean report should pass, got %v", parseErr)
	}

	parseImported := vulnReport{Entries: []vulnEntry{{ID: "GO-1", Reachable: false}}}
	if parseErr := reportVuln(parseImported, false); parseErr != nil {
		parseT.Fatalf("imported-only should pass without -strict, got %v", parseErr)
	}
	if parseErr := reportVuln(parseImported, true); parseErr == nil {
		parseT.Fatal("imported-only should fail under -strict")
	}

	parseReachable := vulnReport{Entries: []vulnEntry{{ID: "GO-2", Reachable: true}}}
	if parseErr := reportVuln(parseReachable, false); parseErr == nil {
		parseT.Fatal("reachable vulnerability should always fail the build")
	}
}
