package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestScanPackageSurfaceIncludesPublicContracts(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "widget/widget.go", `package widget

const Version = "v1"
const internalVersion = "v0"

var Registry = map[string]string{}
var internalRegistry = map[string]string{}

type Embedded struct{}

type Config struct {
	Name string
	private string
	Embedded
}

type Runner interface {
	Run() error
	Read([]byte) (int, error)
	private()
}

type Service[T any] struct{}
type serviceInternal struct{}

func NewService[T any]() Service[T] { return Service[T]{} }
func internalNewService() {}

func (Service[T]) Start() {}
func (serviceInternal) Close() {}
func (serviceInternal) private() {}
`)

	symbols, err := scanPackage(root, "widget", targetSpec{Name: "native", GOOS: "linux", GOARCH: "amd64"})
	if err != nil {
		t.Fatalf("scan package: %v", err)
	}

	want := []string{
		"const Version",
		"field Config.Embedded",
		"field Config.Name",
		"func NewService",
		"interface Runner.Read",
		"interface Runner.Run",
		"method (Service[T]) Start",
		"method (serviceInternal) Close",
		"type Config",
		"type Embedded",
		"type Runner",
		"type Service",
		"var Registry",
	}
	for _, symbol := range want {
		assertContains(t, symbols, symbol)
	}

	notWant := []string{
		"const internalVersion",
		"field Config.private",
		"func internalNewService",
		"method (serviceInternal) private",
		"type serviceInternal",
		"var internalRegistry",
	}
	for _, symbol := range notWant {
		assertNotContains(t, symbols, symbol)
	}
}

// TestScanPackageCapturesSignaturesForBreakingChanges pins that the guard is
// signature-aware: a changed function/method/field/interface-method signature
// produces a different symbol set, so an incompatible reshape that keeps the name
// is caught. Name-only identity (the prior behavior) missed every such break.
func TestScanPackageCapturesSignaturesForBreakingChanges(t *testing.T) {
	scan := func(body string) []string {
		root := t.TempDir()
		writeFixture(t, root, "api/api.go", "package api\n\n"+body)
		symbols, err := scanPackage(root, "api", targetSpec{Name: "native", GOOS: "linux", GOARCH: "amd64"})
		if err != nil {
			t.Fatalf("scan package: %v", err)
		}
		return symbols
	}

	before := scan(`type Cfg struct{ Timeout int }
type Reader interface{ Read(p []byte) (int, error) }
func Do(a int) error { return nil }
func (Cfg) Run(x int) {}
`)
	after := scan(`type Cfg struct{ Timeout string }
type Reader interface{ Read(p []byte) (int64, error) }
func Do(a int, b string) error { return nil }
func (Cfg) Run(x int64) {}
`)

	// Each name-only symbol survives (removal-detection unchanged)...
	for _, name := range []string{"func Do", "method (Cfg) Run", "field Cfg.Timeout", "interface Reader.Read"} {
		assertContains(t, before, name)
		assertContains(t, after, name)
	}
	// ...but the signature entries differ, so the breaking change is visible: every
	// "before" signature entry must be ABSENT from "after".
	beforeSigs := signatureEntries(before)
	if len(beforeSigs) == 0 {
		t.Fatal("expected signature entries to be recorded")
	}
	afterSet := map[string]struct{}{}
	for _, s := range after {
		afterSet[s] = struct{}{}
	}
	for _, sig := range beforeSigs {
		if _, ok := afterSet[sig]; ok {
			t.Fatalf("signature entry %q survived a breaking change; the guard is not signature-aware", sig)
		}
	}
}

func signatureEntries(symbols []string) []string {
	var out []string
	for _, s := range symbols {
		if strings.HasPrefix(s, "funcsig ") || strings.HasPrefix(s, "methodsig ") ||
			strings.HasPrefix(s, "fieldtype ") || strings.HasPrefix(s, "interfacesig ") {
			out = append(out, s)
		}
	}
	return out
}

func TestScanPackageHonorsBuildTargets(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "target/common.go", `package target

func Common() {}
`)
	writeFixture(t, root, "target/native.go", `//go:build !js || !wasm

package target

func NativeOnly() {}
`)
	writeFixture(t, root, "target/wasm.go", `//go:build js && wasm

package target

func BrowserOnly() {}
`)

	nativeSymbols, err := scanPackage(root, "target", targetSpec{Name: "native", GOOS: "linux", GOARCH: "amd64"})
	if err != nil {
		t.Fatalf("scan native package: %v", err)
	}
	assertContains(t, nativeSymbols, "func Common")
	assertContains(t, nativeSymbols, "func NativeOnly")
	assertNotContains(t, nativeSymbols, "func BrowserOnly")

	wasmSymbols, err := scanPackage(root, "target", targetSpec{Name: "wasm", GOOS: "js", GOARCH: "wasm"})
	if err != nil {
		t.Fatalf("scan wasm package: %v", err)
	}
	assertContains(t, wasmSymbols, "func Common")
	assertContains(t, wasmSymbols, "func BrowserOnly")
	assertNotContains(t, wasmSymbols, "func NativeOnly")
}

func TestScanPackageHandlesEmbeddedSelectorAndGenericMembers(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "contract/contract.go", `package contract

import (
	"bytes"
	"io"
)

type Exported struct{}
type Box[T any] struct{}
type privateBox[T any] struct{}

type Config struct {
	*Exported
	bytes.Buffer
	Box[int]
	privateBox[int]
}

type Readerish interface {
	io.Reader
	Close() error
	private()
}
`)

	symbols, err := scanPackage(root, "contract", targetSpec{Name: "native", GOOS: "linux", GOARCH: "amd64"})
	if err != nil {
		t.Fatalf("scan package: %v", err)
	}

	want := []string{
		"field Config.Box",
		"field Config.Buffer",
		"field Config.Exported",
		"interface Readerish.Close",
		"interface Readerish.Reader",
		"type Box",
		"type Config",
		"type Exported",
		"type Readerish",
	}
	for _, symbol := range want {
		assertContains(t, symbols, symbol)
	}

	notWant := []string{
		"field Config.privateBox",
		"interface Readerish.private",
		"type privateBox",
	}
	for _, symbol := range notWant {
		assertNotContains(t, symbols, symbol)
	}
}

func TestScanPackageReturnsSortedSymbolsAndIgnoresTests(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "ordered/z.go", `package ordered

func Zebra() {}
`)
	writeFixture(t, root, "ordered/a.go", `package ordered

const Alpha = "alpha"
`)
	writeFixture(t, root, "ordered/ordered_test.go", `package ordered

func TestOnly() {}
`)

	symbols, err := scanPackage(root, "ordered", targetSpec{Name: "native", GOOS: "linux", GOARCH: "amd64"})
	if err != nil {
		t.Fatalf("scan package: %v", err)
	}

	want := []string{"const Alpha", "func Zebra", "funcsig Zebra func()"}
	if !slices.Equal(symbols, want) {
		t.Fatalf("symbols = %v, want %v", symbols, want)
	}
}

func TestCompareBaselinesAllowsAdditionsAndReportsRemovals(t *testing.T) {
	expected := apiBaseline{
		Targets: map[string]targetBaseline{
			"native": {
				Packages: map[string][]string{
					"ui": {"func Kept", "type Removed"},
				},
			},
		},
	}
	current := apiBaseline{
		Targets: map[string]targetBaseline{
			"native": {
				Packages: map[string][]string{
					"ui": {"func Added", "func Kept"},
				},
			},
		},
	}

	issues := compareBaselines(expected, current)
	if len(issues) != 1 {
		t.Fatalf("expected one removal issue, got %d: %v", len(issues), issues)
	}
	if issues[0].Target != "native" || issues[0].Package != "ui" || issues[0].Symbol != "type Removed" {
		t.Fatalf("unexpected issue: %+v", issues[0])
	}
}

func TestCompareBaselinesReportsMissingTargetsAndPackagesInStableOrder(t *testing.T) {
	expected := apiBaseline{
		Targets: map[string]targetBaseline{
			"wasm": {
				Packages: map[string][]string{
					"z": {"type Removed", "func Missing"},
				},
			},
			"native": {
				Packages: map[string][]string{
					"a": {"type Gone"},
				},
			},
		},
	}
	current := apiBaseline{
		Targets: map[string]targetBaseline{
			"native": {
				Packages: map[string][]string{
					"a": {},
				},
			},
		},
	}

	issues := compareBaselines(expected, current)
	want := []compatIssue{
		{Target: "native", Package: "a", Symbol: "type Gone"},
		{Target: "wasm", Package: "z", Symbol: "func Missing"},
		{Target: "wasm", Package: "z", Symbol: "type Removed"},
	}
	if !slices.Equal(issues, want) {
		t.Fatalf("issues = %+v, want %+v", issues, want)
	}

	report := formatIssues(issues)
	wantReport := strings.Join([]string{
		"- native/a missing type Gone",
		"- wasm/z missing func Missing",
		"- wasm/z missing type Removed",
	}, "\n")
	if report != wantReport {
		t.Fatalf("report = %q, want %q", report, wantReport)
	}
}

func TestParseTargets(t *testing.T) {
	targets, err := parseTargets("wasm,native,compat=windows/amd64:enterprise+debug")
	if err != nil {
		t.Fatalf("parse targets: %v", err)
	}

	byName := map[string]targetSpec{}
	for _, target := range targets {
		byName[target.Name] = target
	}

	if byName["native"].GOOS != "linux" || byName["native"].GOARCH != "amd64" {
		t.Fatalf("native target parsed incorrectly: %+v", byName["native"])
	}
	if byName["wasm"].GOOS != "js" || byName["wasm"].GOARCH != "wasm" {
		t.Fatalf("wasm target parsed incorrectly: %+v", byName["wasm"])
	}
	compat := byName["compat"]
	if compat.GOOS != "windows" || compat.GOARCH != "amd64" {
		t.Fatalf("explicit target parsed incorrectly: %+v", compat)
	}
	if strings.Join(compat.BuildTags, ",") != "debug,enterprise" {
		t.Fatalf("explicit target tags parsed incorrectly: %+v", compat.BuildTags)
	}
}

func TestParseInputsRejectInvalidValues(t *testing.T) {
	if got := parseCommaList(" ui,fetch,,router "); !slices.Equal(got, []string{"fetch", "router", "ui"}) {
		t.Fatalf("parseCommaList sorted/trimming mismatch: %v", got)
	}

	invalidTargets := []string{
		"broken",
		"=linux/amd64",
		"compat=",
		"compat=linux",
		"compat=/amd64",
		"compat=linux/",
	}
	for _, value := range invalidTargets {
		if _, err := parseTargets(value); err == nil {
			t.Fatalf("parseTargets(%q) succeeded, want error", value)
		}
	}
}

func TestRunUpdateWritesBaselineAndCheckReportsRemovedSymbol(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "pkg/pkg.go", `package pkg

func Foo() {}
`)

	baselinePath := filepath.Join(root, "baseline.json")
	if err := run(root, baselinePath, "pkg", "test=linux/amd64", true); err != nil {
		t.Fatalf("run update: %v", err)
	}

	baseline, err := readBaseline(baselinePath)
	if err != nil {
		t.Fatalf("read updated baseline: %v", err)
	}
	if baseline.GeneratedAtUTC == "" {
		t.Fatal("updated baseline should include a generation timestamp")
	}
	if !slices.Equal(baseline.Scope, []string{"pkg"}) {
		t.Fatalf("baseline scope = %v, want [pkg]", baseline.Scope)
	}
	if got := baseline.Targets["test"].Packages["pkg"]; !slices.Equal(got, []string{"func Foo", "funcsig Foo func()"}) {
		t.Fatalf("baseline symbols = %v, want [func Foo funcsig Foo func()]", got)
	}

	if err := run(root, baselinePath, "pkg", "test=linux/amd64", false); err != nil {
		t.Fatalf("run check with unchanged package: %v", err)
	}

	writeFixture(t, root, "pkg/pkg.go", `package pkg

func Bar() {}
`)

	err = run(root, baselinePath, "pkg", "test=linux/amd64", false)
	if err == nil {
		t.Fatal("run check succeeded after removing a baseline symbol")
	}
	message := err.Error()
	if !strings.Contains(message, "API compatibility guard failed:") {
		t.Fatalf("error should include failure header, got: %v", err)
	}
	if !strings.Contains(message, "- test/pkg missing func Foo") {
		t.Fatalf("error should include missing symbol report, got: %v", err)
	}
	if !strings.Contains(message, "go run ./tools/api_compat_guard -update") {
		t.Fatalf("error should include update guidance, got: %v", err)
	}
}

func TestRunRejectsEmptyPackagesAndTargets(t *testing.T) {
	root := t.TempDir()
	for _, tc := range []struct {
		name       string
		packages   string
		targets    string
		wantErrSub string
	}{
		{name: "packages", packages: " , ", targets: "native", wantErrSub: "at least one package"},
		{name: "targets", packages: "pkg", targets: " , ", wantErrSub: "at least one target"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := run(root, filepath.Join(root, "baseline.json"), tc.packages, tc.targets, false)
			if err == nil {
				t.Fatal("run succeeded, want error")
			}
			if !strings.Contains(err.Error(), tc.wantErrSub) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantErrSub)
			}
		})
	}
}

func TestCommittedBaselineMatchesCurrentSurface(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}

	expected, err := readBaseline(filepath.Join(root, defaultBaselinePath))
	if err != nil {
		t.Fatalf("read committed baseline: %v", err)
	}

	targets := make([]targetSpec, 0, len(expected.Targets))
	for name, target := range expected.Targets {
		targets = append(targets, targetSpec{
			Name:      name,
			GOOS:      target.GOOS,
			GOARCH:    target.GOARCH,
			BuildTags: target.BuildTags,
		})
	}

	current, err := collectBaseline(root, expected.Scope, targets)
	if err != nil {
		t.Fatalf("collect current baseline: %v", err)
	}

	if issues := compareBaselines(expected, current); len(issues) > 0 {
		t.Fatalf("committed API baseline is stale:\n%s", formatIssues(issues))
	}
}

func writeFixture(t *testing.T, root string, relativePath string, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

func assertContains(t *testing.T, symbols []string, want string) {
	t.Helper()
	if slices.Contains(symbols, want) {
		return
	}
	t.Fatalf("expected symbols to contain %q; got %v", want, symbols)
}

func assertNotContains(t *testing.T, symbols []string, want string) {
	t.Helper()
	for _, symbol := range symbols {
		if symbol == want {
			t.Fatalf("expected symbols not to contain %q; got %v", want, symbols)
		}
	}
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
