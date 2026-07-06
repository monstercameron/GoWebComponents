package doclint

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestExtractKnownGwcFlagsFindsVarFlags pins that both definition styles are
// captured, including flag.Var flags (e.g. -lane, -ext) whose name is the
// second argument.
func TestExtractKnownGwcFlagsFindsVarFlags(t *testing.T) {
	root := repoRoot(t)
	known, err := ExtractKnownGwcFlags(filepath.Join(root, gwcDirRel))
	if err != nil {
		t.Fatalf("extract known flags: %v", err)
	}
	for _, want := range []string{"app", "root", "profile", "lane", "ext", "exclude-dir", "compression"} {
		if !known[want] {
			t.Fatalf("expected known gwc flag %q to be extracted", want)
		}
	}
}

// TestExtractKnownGwcFlagsFindsTypedVarAndUint64 pins the previously-missing
// definition forms: the *Var family (name is the 2nd arg — .StringVar, .BoolVar,
// .DurationVar, …) and .Uint64. Without them, a flag defined this way would be
// absent from the known set and every doc referencing it flagged as unknown (a
// false-fail). gwc uses none today, so this fixture-based test guards the gap.
func TestExtractKnownGwcFlagsFindsTypedVarAndUint64(t *testing.T) {
	dir := t.TempDir()
	src := `package main

import "flag"

func setup(parseFs *flag.FlagSet) {
	var cfg struct{ Name string; On bool; N uint64; Every int }
	parseFs.StringVar(&cfg.Name, "profile-name", "", "usage")
	parseFs.BoolVar(&cfg.On, "verbose-mode", false, "usage")
	parseFs.Uint64("max-bytes", 0, "usage")
	parseFs.DurationVar(nil, "poll-interval", 0, "usage")
	_ = cfg
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	known, err := ExtractKnownGwcFlags(dir)
	if err != nil {
		t.Fatalf("extract known flags: %v", err)
	}
	for _, want := range []string{"profile-name", "verbose-mode", "max-bytes", "poll-interval"} {
		if !known[want] {
			t.Fatalf("expected typed-var / Uint64 flag %q to be extracted, got %v", want, known)
		}
	}
}

// TestDocsGwcFlagsExist is the flag-existence guard: every -flag used on a gwc
// command line in the docs must correspond to a real launcher flag. A renamed
// or removed flag still cited in docs fails here.
func TestDocsGwcFlagsExist(t *testing.T) {
	root := repoRoot(t)
	known, err := ExtractKnownGwcFlags(filepath.Join(root, gwcDirRel))
	if err != nil {
		t.Fatalf("extract known flags: %v", err)
	}
	unknown, err := ScanDocGwcFlags(root, known)
	if err != nil {
		t.Fatalf("scan doc flags: %v", err)
	}
	if len(unknown) > 0 {
		lines := make([]string, 0, len(unknown))
		for _, u := range unknown {
			lines = append(lines, u.DocPath+":"+itoa(u.Line)+" -> -"+u.Flag)
		}
		sort.Strings(lines)
		t.Fatalf("docs reference %d gwc flag(s) that no longer exist:\n%s", len(unknown), strings.Join(lines, "\n"))
	}
}

// TestScanDocGwcFlagsCatchesPlantedUnknown proves the guard works: a fixture doc
// citing a removed flag is reported, while a real flag and a non-gwc tool's flag
// on another line are not.
func TestScanDocGwcFlagsCatchesPlantedUnknown(t *testing.T) {
	root := t.TempDir()
	doc := "# Doc\n\n```powershell\n" +
		"go run ./tools/gwc dev -app .\\main.go -totallyfakeflag x\n" + // gwc line: app ok, totallyfakeflag bad
		"go test ./... -count=1 -v\n" + // not a gwc line: -count/-v must be ignored
		"```\n"
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(doc), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	known := map[string]bool{"app": true, "h": true}
	unknown, err := ScanDocGwcFlags(root, known)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(unknown) != 1 {
		t.Fatalf("expected exactly 1 unknown flag, got %d: %+v", len(unknown), unknown)
	}
	if unknown[0].Flag != "totallyfakeflag" {
		t.Fatalf("guard flagged the wrong token: %q", unknown[0].Flag)
	}
}
