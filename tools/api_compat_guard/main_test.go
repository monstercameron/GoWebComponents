package main

import (
	"os"
	"path/filepath"
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
	for _, symbol := range symbols {
		if symbol == want {
			return
		}
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
