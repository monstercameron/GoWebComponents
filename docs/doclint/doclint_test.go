package doclint

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// repoRoot resolves the module root from the test working directory.
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root, ok := FindRepoRoot(wd)
	if !ok {
		t.Fatalf("module root not found from %s", wd)
	}
	return root
}

// TestDocsHaveNoBrokenGoSamples is the doc-sample drift guard (F1 "docs can't silently lie"):
// every complete-file ```go sample in the docs must parse, so a renamed API or a fat-fingered
// edit that leaves a copy-paste-runnable sample broken fails CI instead of shipping. Fragments
// (snippets without a package clause) are intentionally not checked, keeping zero false positives.
// Runs under `go test ./...`, which gates PRs.
func TestDocsHaveNoBrokenGoSamples(t *testing.T) {
	root := repoRoot(t)
	errs, err := ValidateGoBlocks(root)
	if err != nil {
		t.Fatalf("validate doc go samples: %v", err)
	}
	if len(errs) > 0 {
		var b strings.Builder
		for _, e := range errs {
			fmt.Fprintf(&b, "  %s:%d — %s\n", e.DocPath, e.Line, e.Err)
		}
		t.Fatalf("docs contain %d complete-file ```go sample(s) that no longer parse:\n%s", len(errs), b.String())
	}
}

// TestGoSampleGuardCatchesBreakage proves the sample guard actually fails on a broken
// complete-file block (and ignores a fragment), so the zero-error baseline is meaningful.
func TestGoSampleGuardCatchesBreakage(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "broken.md"),
		[]byte("```go\npackage main\nfunc main( {\n```\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	errs, err := ValidateGoBlocks(dir)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(errs) != 1 {
		t.Fatalf("expected the planted broken sample to be caught, got %d", len(errs))
	}
}

// TestDocsHaveNoBrokenRepoPaths is the drift guard: every repo-relative input
// path referenced in a fenced doc command must resolve on disk. This is the
// check that would have failed the build the moment examples/01-counter was
// relocated without updating the README/getting-started/AGENTS commands, and it
// stays at zero tolerance so any future rename that strands a doc command fails
// CI instead of shipping a broken front door.
func TestDocsHaveNoBrokenRepoPaths(t *testing.T) {
	root := repoRoot(t)
	refs, err := ScanRepo(root)
	if err != nil {
		t.Fatalf("scan repo docs: %v", err)
	}
	if len(refs) == 0 {
		t.Fatal("guard found zero repo-relative doc paths to check; the scanner is likely misconfigured")
	}
	broken := Resolve(root, refs)
	if len(broken) > 0 {
		lines := make([]string, 0, len(broken))
		for _, b := range broken {
			lines = append(lines, b.DocPath+":"+itoa(b.Line)+" -> "+b.Raw+" (resolved "+b.Rel+")")
		}
		sort.Strings(lines)
		t.Fatalf("docs reference %d repo path(s) that no longer exist:\n%s\n\n"+
			"Fix each path to its current location (the example/dir was likely renamed or moved).",
			len(broken), strings.Join(lines, "\n"))
	}
}

// TestGuardCatchesPlantedBreakage proves the guard itself works: a fixture doc
// with a known-dead repo path must be reported, and a fixture with a live path
// must not. Without this, a green guard could mean "nothing broken" or "scanner
// silently broke" and we could not tell them apart.
func TestGuardCatchesPlantedBreakage(t *testing.T) {
	root := t.TempDir()
	// Minimal fake repo: a go.mod and a real top-level dir the doc can point at.
	mustWrite(t, filepath.Join(root, "go.mod"), "module example.com/fixture\n")
	mustMkdir(t, filepath.Join(root, "examples", "public", "counter"))
	mustWrite(t, filepath.Join(root, "examples", "public", "counter", "main.go"), "package main\n")

	dead := "# Doc\n\n```powershell\n" +
		"go run ./tools/gwc dev -app .\\examples\\public\\counter\\main.go\n" + // live
		"go run ./tools/gwc dev -app .\\examples\\01-counter\\main.go\n" + // dead
		"go run ./tools/gwc build -app .\\main.go -out .\\bin\\x.wasm\n" + // placeholder + output, both skipped
		"```\n"
	mustWrite(t, filepath.Join(root, "README.md"), dead)

	refs, err := ScanRepo(root)
	if err != nil {
		t.Fatalf("scan fixture: %v", err)
	}
	broken := Resolve(root, refs)
	if len(broken) != 1 {
		t.Fatalf("expected exactly 1 broken ref, got %d: %+v", len(broken), broken)
	}
	if !strings.Contains(filepath.ToSlash(broken[0].Rel), "examples/01-counter") {
		t.Fatalf("guard flagged the wrong path: %q", broken[0].Rel)
	}
	// The live path and the user-project placeholder/output must NOT be flagged.
	for _, b := range broken {
		if strings.Contains(b.Rel, "public/counter") {
			t.Fatalf("guard wrongly flagged the live counter path: %q", b.Rel)
		}
	}
}

// TestSnapshotDocsAreExempt confirms the archival example-site docs subtree is
// not scanned, so its preserved historical numbered paths do not fail the lane.
func TestSnapshotDocsAreExempt(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "go.mod"), "module example.com/fixture\n")
	mustMkdir(t, filepath.Join(root, "examples"))
	snapDir := filepath.Join(root, filepath.FromSlash(snapshotDocDir))
	mustMkdir(t, snapDir)
	mustWrite(t, filepath.Join(snapDir, "history.md"),
		"```powershell\ngo run ./tools/gwc dev -app .\\examples\\01-counter\\main.go\n```\n")

	refs, err := ScanRepo(root)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(Resolve(root, refs)) != 0 {
		t.Fatalf("snapshot docs should be exempt but a path was flagged: %+v", Resolve(root, refs))
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

// itoa avoids importing strconv for one tiny conversion in the failure path.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
