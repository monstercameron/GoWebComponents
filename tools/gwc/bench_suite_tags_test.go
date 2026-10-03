package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeBenchSuiteTags(parseT *testing.T) {
	parseGot, parseErr := normalizeBenchSuiteTags(" production, b,production a ")
	if parseErr != nil || parseGot != "a,b,production" {
		parseT.Fatalf("got %q, %v", parseGot, parseErr)
	}
	if parseEmpty, parseErr2 := normalizeBenchSuiteTags(""); parseErr2 != nil || parseEmpty != "" {
		parseT.Fatalf("empty tags: %q, %v", parseEmpty, parseErr2)
	}
	if _, parseErr3 := normalizeBenchSuiteTags("good,bad;tag"); parseErr3 == nil {
		parseT.Fatal("a tag with punctuation must be rejected")
	}
}

func TestFilterBenchSuitePackages(parseT *testing.T) {
	parseNative := []string{"./css", "./html", "./internal/platform/mockdom", "./internal/runtime", "./ui"}
	parseWasm := []string{"./internal/platform/jsdom", "./ui"}
	parseLanes := []string{"native", "wasm"}

	parseCases := []struct {
		parsePatterns []string
		parseWantNat  []string
		parseWantWasm []string
	}{
		{nil, parseNative, parseWasm},
		{[]string{"./ui", "./css"}, []string{"./css", "./ui"}, []string{"./ui"}},
		{[]string{"./internal/platform/*"}, []string{"./internal/platform/mockdom"}, []string{"./internal/platform/jsdom"}},
		{[]string{"./internal/..."}, []string{"./internal/platform/mockdom", "./internal/runtime"}, []string{"./internal/platform/jsdom"}},
	}
	for _, parseCase := range parseCases {
		parseGotNat, parseGotWasm, parseErr := filterBenchSuitePackages(parseNative, parseWasm, parseLanes, parseCase.parsePatterns)
		if parseErr != nil {
			parseT.Fatalf("%v: %v", parseCase.parsePatterns, parseErr)
		}
		if !reflect.DeepEqual(parseGotNat, parseCase.parseWantNat) || !reflect.DeepEqual(parseGotWasm, parseCase.parseWantWasm) {
			parseT.Fatalf("%v: native %v wasm %v", parseCase.parsePatterns, parseGotNat, parseGotWasm)
		}
	}

	_, _, parseErr2 := filterBenchSuitePackages(parseNative, parseWasm, parseLanes, []string{"./ui", "./nope", "./also/missing"})
	if parseErr2 == nil || !strings.Contains(parseErr2.Error(), "./nope, ./also/missing") || !strings.Contains(parseErr2.Error(), "valid packages: ./css, ./html") {
		parseT.Fatalf("unknown entries must be listed with the valid packages: %v", parseErr2)
	}
	// A package that only exists in a lane that was not selected is unknown for this run.
	_, _, parseErr3 := filterBenchSuitePackages(parseNative, parseWasm, []string{"native"}, []string{"./internal/platform/jsdom"})
	if parseErr3 == nil {
		parseT.Fatal("a wasm-only package must be unknown when only the native lane runs")
	}
}

func TestBenchSuiteConfigHashCoversTagsAndPackages(parseT *testing.T) {
	parseBase := benchSuiteConfig{suiteName: "s", lanes: []string{"native"}, count: 1}
	parseTagged := parseBase
	parseTagged.tags = "production"
	parseScoped := parseBase
	parseScoped.packages = []string{"./ui"}
	parseHash := hashBenchSuiteConfig(parseBase)
	if hashBenchSuiteConfig(parseTagged) == parseHash || hashBenchSuiteConfig(parseScoped) == parseHash {
		parseT.Fatal("tags and packages must change the configuration hash used by -resume")
	}
	if hashBenchSuiteConfig(parseBase) != parseHash {
		parseT.Fatal("hash must be stable")
	}
	parseResolved, parseErr := resolveBenchSuiteConfig(benchSuiteConfig{count: 1, tags: "production, production", packages: []string{"ui", "./ui/", ".\\css"}})
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if parseResolved.tags != "production" || !reflect.DeepEqual(parseResolved.packages, []string{"./ui", "./css"}) {
		parseT.Fatalf("resolved tags %q packages %v", parseResolved.tags, parseResolved.packages)
	}
	if _, parseErr2 := resolveBenchSuiteConfig(benchSuiteConfig{count: 1, tags: "bad tag!x;"}); parseErr2 == nil {
		parseT.Fatal("invalid tag must be rejected")
	}
}

func TestRunBenchSuiteWithBuildTagsRestoresGoflags(parseT *testing.T) {
	parseT.Setenv("GOFLAGS", "-mod=mod -tags=old")
	parseSeen := ""
	_, _ = runBenchSuiteWithBuildTags("production", func() (int, error) {
		parseSeen = os.Getenv("GOFLAGS")
		return 0, nil
	})
	if parseSeen != "-mod=mod -tags=production" {
		parseT.Fatalf("GOFLAGS during the call: %q", parseSeen)
	}
	if parseAfter := os.Getenv("GOFLAGS"); parseAfter != "-mod=mod -tags=old" {
		parseT.Fatalf("GOFLAGS not restored: %q", parseAfter)
	}
	_, _ = runBenchSuiteWithBuildTags("", func() (int, error) {
		parseSeen = os.Getenv("GOFLAGS")
		return 0, nil
	})
	if parseSeen != "-mod=mod -tags=old" {
		parseT.Fatalf("empty tags must not touch GOFLAGS: %q", parseSeen)
	}
}

func TestBenchSuiteRunsOnlySelectedPackagesWithTags(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	for _, parseName := range []string{"a", "b", "c"} {
		_ = os.MkdirAll(filepath.Join(parseRoot, parseName), 0o755)
		_ = os.WriteFile(filepath.Join(parseRoot, parseName, "x_test.go"), []byte("package "+parseName+"\nfunc BenchmarkX() {}\n"), 0o644)
	}
	parseOriginal := benchSuiteRunPackage
	parseT.Cleanup(func() { benchSuiteRunPackage = parseOriginal })
	parseRan := []string{}
	parseTags := []string{}
	benchSuiteRunPackage = func(parseL launcher, parseConfig benchmarkConfig, parseLane string, parsePackagePath string) benchmarkPackageReport {
		parseRan = append(parseRan, parsePackagePath)
		parseTags = append(parseTags, parseConfig.tags)
		return benchmarkPackageReport{Lane: parseLane, Package: parsePackagePath, OK: true, Command: "go test -tags " + parseConfig.tags + " .",
			Benchmarks: parseBenchmarkOutput("BenchmarkX-8 10 100 ns/op\n")}
	}
	parseBuild := func(parsePackages []string) (benchSuiteReport, error) {
		parseConfig, parseErr := resolveBenchSuiteConfig(benchSuiteConfig{rootPath: parseRoot, outPath: filepath.Join(parseRoot, "bin", "s.json"), count: 1, lanes: []string{"native"},
			seed: 1, tags: "production", packages: parsePackages, isQuiet: true})
		if parseErr != nil {
			parseT.Fatal(parseErr)
		}
		return launcher{}.executeBenchSuite(parseConfig)
	}
	parseReport, parseErr := parseBuild([]string{"./a", "./c"})
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if !reflect.DeepEqual(parseRan, []string{"./a", "./c"}) || !reflect.DeepEqual(parseTags, []string{"production", "production"}) {
		parseT.Fatalf("ran %v with tags %v", parseRan, parseTags)
	}
	if !reflect.DeepEqual(parseReport.Run.Order, []string{"native/./a", "native/./c"}) {
		parseT.Fatalf("order %v", parseReport.Run.Order)
	}
	if parseReport.Machine == nil || parseReport.Machine.BuildTags != "production" {
		parseT.Fatalf("machine.buildTags not recorded: %+v", parseReport.Machine)
	}
	if _, parseErr2 := parseBuild([]string{"./a", "./missing"}); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "./missing") {
		parseT.Fatalf("unknown package must fail the run, got %v", parseErr2)
	}
}

func TestRunBenchmarkPackagePassesTagsToGoTest(parseT *testing.T) {
	if testing.Short() {
		parseT.Skip("runs go test -bench")
	}
	parseRoot := parseT.TempDir()
	_ = os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte("module tagsmoke\n\ngo 1.24\n"), 0o644)
	_ = os.WriteFile(filepath.Join(parseRoot, "dev.go"), []byte("//go:build !production\n\npackage tagsmoke\n\nconst isProduction = false\n"), 0o644)
	_ = os.WriteFile(filepath.Join(parseRoot, "prod.go"), []byte("//go:build production\n\npackage tagsmoke\n\nconst isProduction = true\n"), 0o644)
	_ = os.WriteFile(filepath.Join(parseRoot, "s_test.go"), []byte("package tagsmoke\n\nimport \"testing\"\n\nfunc BenchmarkTagged(b *testing.B) {\n\tif !isProduction {\n\t\tb.Skip(\"dev build\")\n\t}\n\tfor b.Loop() {\n\t}\n}\n"), 0o644)
	parseRun := func(parseTagList string) benchmarkPackageReport {
		return launcher{}.runBenchmarkPackage(benchmarkConfig{rootPath: parseRoot, bench: ".", count: 1, benchtime: "1x", tags: parseTagList}, "native", ".")
	}
	parseDev := parseRun("")
	if !parseDev.OK || len(parseDev.Benchmarks) != 0 || strings.Contains(parseDev.Command, "-tags") {
		parseT.Fatalf("untagged run must skip and add no -tags flag: %+v", parseDev)
	}
	parseProd := parseRun("production")
	if !parseProd.OK || len(parseProd.Benchmarks) != 1 || !strings.Contains(parseProd.Command, "go test -tags production -run") {
		parseT.Fatalf("tagged run must reach the child: %+v", parseProd)
	}
}
