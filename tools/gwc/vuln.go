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
	Stdlib    bool // the vulnerable module is the Go standard library / toolchain
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
	parseIncludeStdlib := parseFlags.Bool("include-stdlib", false, "Also block on reachable standard-library/toolchain advisories (fix by bumping Go)")
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

	return reportVuln(parseReport, *parseStrict, *parseIncludeStdlib)
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
	parseStdlib := map[string]bool{}
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
			if findingIsStdlib(parseMessage.Finding) {
				parseStdlib[parseID] = true
			}
		}
	}

	parseReport := vulnReport{}
	for _, parseID := range parseOrder {
		parseReport.Entries = append(parseReport.Entries, vulnEntry{
			ID:        parseID,
			Summary:   parseSummaries[parseID],
			Reachable: parseReachable[parseID],
			Stdlib:    parseStdlib[parseID],
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

// findingIsStdlib reports whether a finding's vulnerable module is the Go standard library or
// toolchain. govulncheck names that module "stdlib" (and "toolchain" for the compiler), and the
// vulnerable symbol is the first trace frame. Such advisories can only be remediated by bumping
// the Go toolchain, so the gate reports them but — by default — does not block dependency-scope
// CI on them.
func findingIsStdlib(parseFinding *vulnFinding) bool {
	for _, parseFrame := range parseFinding.Trace {
		if parseModule := strings.TrimSpace(parseFrame.Module); parseModule != "" {
			return parseModule == "stdlib" || parseModule == "toolchain"
		}
	}
	return false
}

// reportVuln prints the verdict and returns a non-zero (error) result when the scan should
// fail the build. The gate BLOCKS on reachable dependency advisories (vulnerable code your app
// actually calls), which is what makes it a real merge gate. Reachable standard-library /
// toolchain advisories are reported but do not block by default — they are only fixable by
// bumping the Go toolchain, so failing every PR on them adds no security value; pass
// -include-stdlib to block on those too. -strict additionally blocks on merely-imported
// (unreachable) advisories.
func reportVuln(parseReport vulnReport, parseStrict bool, parseIncludeStdlib bool) error {
	if len(parseReport.Entries) == 0 {
		fmt.Println("GWC vuln: OK — no known vulnerabilities found")
		return nil
	}

	parseBlocking := []vulnEntry{}
	parseStdlibReachable := []vulnEntry{}
	for _, parseEntry := range parseReport.Entries {
		parseLabel := "imported"
		if parseEntry.Reachable {
			parseLabel = "REACHABLE"
		}
		parseScope := "dependency"
		if parseEntry.Stdlib {
			parseScope = "stdlib"
		}
		fmt.Printf("  [%s/%s] %s — %s\n", parseLabel, parseScope, parseEntry.ID, parseEntry.Summary)
		if parseEntry.Reachable {
			if parseEntry.Stdlib && !parseIncludeStdlib {
				parseStdlibReachable = append(parseStdlibReachable, parseEntry)
				continue
			}
			parseBlocking = append(parseBlocking, parseEntry)
		}
	}

	for _, parseEntry := range parseStdlibReachable {
		fmt.Printf("GWC vuln: NOTE — reachable stdlib/toolchain advisory %s; remediate by bumping the Go toolchain (not gating; use -include-stdlib to gate)\n", parseEntry.ID)
	}

	if len(parseBlocking) > 0 {
		fmt.Printf("GWC vuln: FAIL — %d reachable vulnerability(ies) (called by your code)\n", len(parseBlocking))
		return fmt.Errorf("%d reachable vulnerability(ies)", len(parseBlocking))
	}
	if parseStrict {
		fmt.Printf("GWC vuln: FAIL (strict) — %d imported vulnerability(ies)\n", len(parseReport.Entries))
		return fmt.Errorf("%d imported vulnerability(ies) under -strict", len(parseReport.Entries))
	}
	fmt.Printf("GWC vuln: OK — %d vulnerability(ies) present but none block the gate\n", len(parseReport.Entries))
	return nil
}
