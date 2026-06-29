//go:build doclintcmd

package doclint

import (
	"sort"
	"strings"
	"testing"
	"time"
)

// TestDocsGwcCommandsExecute is the opt-in command-execution lane for docs.
// Run it explicitly with:
//
//	go test -tags doclintcmd ./docs/doclint -run TestDocsGwcCommandsExecute -count=1 -v -timeout 10m
func TestDocsGwcCommandsExecute(t *testing.T) {
	root := repoRoot(t)
	summary, err := RunDocGwcCommands(root, GwcCommandRunOptions{
		OutputRoot:  t.TempDir(),
		Timeout:     2 * time.Minute,
		MaxCommands: 1000,
	})
	if err != nil {
		t.Fatalf("run documented gwc commands: %v", err)
	}
	if len(summary.Executed) == 0 {
		t.Fatal("no documented gwc commands were executable; the command planner may be too strict")
	}
	skipReasons := map[string]int{}
	for _, result := range summary.Skipped {
		skipReasons[result.Plan.SkipReason]++
	}

	var failures []string
	for _, result := range summary.Executed {
		if result.Err == nil {
			continue
		}
		output := strings.TrimSpace(result.Output)
		if len(output) > 800 {
			output = output[:800] + "...(truncated)"
		}
		failures = append(failures, result.Plan.Ref.DocPath+":"+itoa(result.Plan.Ref.Line)+" -> "+
			strings.Join(result.Plan.Args, " ")+"\n"+result.Err.Error()+"\n"+output)
	}
	if len(failures) > 0 {
		sort.Strings(failures)
		t.Fatalf("documented gwc command execution failed for %d command(s):\n%s", len(failures), strings.Join(failures, "\n\n"))
	}
	t.Logf("executed %d documented gwc commands; skipped %d with explicit reasons", len(summary.Executed), len(summary.Skipped))
	for reason, count := range skipReasons {
		t.Logf("skipped %d: %s", count, reason)
	}
}
