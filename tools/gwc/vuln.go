package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// runVulnCommand routes the `gwc vuln` known-vulnerability scan.
var runVulnCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runVuln(parseArgs)
}

// vulnMessage is one govulncheck -json stream message; exactly one field is set.
type vulnMessage struct {
	OSV     *vulnOSV     `json:"osv"`
	Finding *vulnFinding `json:"finding"`
}

// vulnOSV is a vulnerability database entry.
type vulnOSV struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
}

// vulnFinding is one detected use of a vulnerability, with the call/import trace.
type vulnFinding struct {
	OSV   string      `json:"osv"`
	Trace []vulnFrame `json:"trace"`
}

// vulnFrame is one frame of a finding's trace.
type vulnFrame struct {
	Module   string `json:"module"`
	Package  string `json:"package"`
	Function string `json:"function"`
}

// vulnEntry is the deduplicated, classified result for one vulnerability.
type vulnEntry struct {
	ID        string
	Summary   string
	Reachable bool // a finding's trace reaches a called function, not just an import
}

// vulnReport is the parsed outcome of a govulncheck scan.
type vulnReport struct {
	Entries []vulnEntry
}

// Reachable returns the vulnerabilities whose vulnerable code is actually called.
func (parseR vulnReport) Reachable() []vulnEntry {
	var parseOut []vulnEntry
	for _, parseEntry := range parseR.Entries {
		if parseEntry.Reachable {
			parseOut = append(parseOut, parseEntry)
		}
	}
	return parseOut
}

// runVuln parses `gwc vuln [-pattern ./...]` and runs govulncheck, reporting reachable and
// imported-only vulnerabilities.
func (parseL launcher) runVuln(parseArgs []string) error {
	parseFlags := flag.NewFlagSet("vuln", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parsePattern := parseFlags.String("pattern", "./...", "Package pattern to scan")
	parseStrict := parseFlags.Bool("strict", false, "Exit non-zero when ANY vulnerability is imported, not only when one is reachable")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseStdout, parseErr := runGovulncheck(*parsePattern)
	if parseErr != nil {
		return parseErr
	}

	parseReport, parseErr := parseVulnFindings(strings.NewReader(parseStdout))
	if parseErr != nil {
		return fmt.Errorf("parse govulncheck output: %w", parseErr)
	}

	return reportVuln(parseReport, *parseStrict)
}

// runGovulncheck executes govulncheck -json over the pattern, returning its stdout. A
// missing binary returns an actionable install hint.
func runGovulncheck(parsePattern string) (string, error) {
	parseCmd := exec.Command("govulncheck", "-json", parsePattern)
	parseOut, parseErr := parseCmd.Output()
	if parseErr != nil {
		var parseExitErr *exec.ExitError
		if errors.As(parseErr, &parseExitErr) {
			// govulncheck exits non-zero when vulnerabilities are found; the JSON is still
			// on stdout, so treat that as success for parsing.
			return string(parseOut), nil
		}
		if errors.Is(parseErr, exec.ErrNotFound) {
			return "", errors.New("govulncheck not found; install it with: go install golang.org/x/vuln/cmd/govulncheck@latest")
		}
		return "", fmt.Errorf("run govulncheck: %w", parseErr)
	}
	return string(parseOut), nil
}

// parseVulnFindings decodes the govulncheck -json stream into a deduplicated, classified
// report: it pairs findings with their OSV summaries and marks a vulnerability reachable
// when any of its findings traces to a called function.
func parseVulnFindings(parseReader io.Reader) (vulnReport, error) {
	parseDecoder := json.NewDecoder(parseReader)
	parseSummaries := map[string]string{}
	parseReachable := map[string]bool{}
	parseSeen := map[string]bool{}
	var parseOrder []string

	for {
		var parseMessage vulnMessage
		if parseErr := parseDecoder.Decode(&parseMessage); parseErr != nil {
			if errors.Is(parseErr, io.EOF) {
				break
			}
			return vulnReport{}, parseErr
		}
		if parseMessage.OSV != nil && parseMessage.OSV.ID != "" {
			parseSummaries[parseMessage.OSV.ID] = parseMessage.OSV.Summary
		}
		if parseMessage.Finding != nil && parseMessage.Finding.OSV != "" {
			parseID := parseMessage.Finding.OSV
			if !parseSeen[parseID] {
				parseSeen[parseID] = true
				parseOrder = append(parseOrder, parseID)
			}
			if findingReachesFunction(parseMessage.Finding) {
				parseReachable[parseID] = true
			}
		}
	}

	parseReport := vulnReport{}
	for _, parseID := range parseOrder {
		parseReport.Entries = append(parseReport.Entries, vulnEntry{
			ID:        parseID,
			Summary:   parseSummaries[parseID],
			Reachable: parseReachable[parseID],
		})
	}
	sort.Slice(parseReport.Entries, func(parseA, parseB int) bool {
		return parseReport.Entries[parseA].ID < parseReport.Entries[parseB].ID
	})
	return parseReport, nil
}

// findingReachesFunction reports whether a finding's trace reaches a specific called
// function (the most specific frame names one), as opposed to a mere import.
func findingReachesFunction(parseFinding *vulnFinding) bool {
	for _, parseFrame := range parseFinding.Trace {
		if strings.TrimSpace(parseFrame.Function) != "" {
			return true
		}
	}
	return false
}

// reportVuln prints the verdict and returns a non-zero (error) result when the scan should
// fail the build.
func reportVuln(parseReport vulnReport, parseStrict bool) error {
	parseReachable := parseReport.Reachable()
	if len(parseReport.Entries) == 0 {
		fmt.Println("GWC vuln: OK — no known vulnerabilities found")
		return nil
	}

	for _, parseEntry := range parseReport.Entries {
		parseLabel := "imported"
		if parseEntry.Reachable {
			parseLabel = "REACHABLE"
		}
		fmt.Printf("  [%s] %s — %s\n", parseLabel, parseEntry.ID, parseEntry.Summary)
	}

	if len(parseReachable) > 0 {
		fmt.Printf("GWC vuln: FAIL — %d reachable vulnerability(ies) (called by your code)\n", len(parseReachable))
		return fmt.Errorf("%d reachable vulnerability(ies)", len(parseReachable))
	}
	if parseStrict {
		fmt.Printf("GWC vuln: FAIL (strict) — %d imported vulnerability(ies)\n", len(parseReport.Entries))
		return fmt.Errorf("%d imported vulnerability(ies) under -strict", len(parseReport.Entries))
	}
	fmt.Printf("GWC vuln: OK — %d vulnerability(ies) present but none reachable (use -strict to fail on these)\n", len(parseReport.Entries))
	return nil
}
