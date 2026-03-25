package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveReleaseConfigPrefersScaffoldMetadata(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseBudgetsPath := filepath.Join(parseTempApp, "config", "release-budgets.json")
	if parseErr := os.WriteFile(filepath.Join(parseTempApp, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); parseErr != nil {
		parseT.Fatalf("write main.go: %v", parseErr)
	}
	parseMetadata := `{
  "projectName": "metadata-release-app",
  "modulePath": "example.com/metadata-release-app",
  "tooling": {
    "appPath": "main.go",
	"releaseOutDir": "bin/release",
	    "releaseBinaryName": "site.wasm",
	    "releaseCompression": "none",
	    "releaseBudgetsPath": "config/release-budgets.json"
  }
}
`
	if parseErr2 := os.WriteFile(filepath.Join(parseTempApp, "gwc-start.json"), []byte(parseMetadata), 0644); parseErr2 != nil {
		parseT.Fatalf("write gwc-start.json: %v", parseErr2)
	}

	parseOriginalGetwd := buildGetwd
	parseT.Cleanup(func() { buildGetwd = parseOriginalGetwd })
	buildGetwd = func() (string, error) { return parseTempApp, nil }

	parseConfig, parseErr3 := resolveReleaseConfig(releaseConfig{})
	if parseErr3 != nil {
		parseT.Fatalf("resolve release config: %v", parseErr3)
	}
	if parseConfig.appPath != filepath.Join(parseTempApp, "main.go") {
		parseT.Fatalf("expected metadata app path, got %#v", parseConfig)
	}
	if parseConfig.outDir != filepath.Join(parseTempApp, "bin", "release") {
		parseT.Fatalf("expected metadata release out dir, got %#v", parseConfig)
	}
	if parseConfig.binaryName != "site.wasm" {
		parseT.Fatalf("expected metadata release binary name, got %#v", parseConfig)
	}
	if parseConfig.budgetsPath != parseBudgetsPath {
		parseT.Fatalf("expected metadata release budgets path, got %#v", parseConfig)
	}
	if !parseConfig.skipCompression {
		parseT.Fatalf("expected metadata release compression policy to disable compression, got %#v", parseConfig)
	}
	if parseConfig.compression != "none" {
		parseT.Fatalf("expected metadata release compression policy none, got %#v", parseConfig)
	}
	if parseConfig.profile != "release" {
		parseT.Fatalf("expected release profile default, got %#v", parseConfig)
	}
}

func TestResolveReleaseConfigUsesArtifactRootOverride(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseMainPath := filepath.Join(parseRoot, "main.go")
	if parseErr := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr != nil {
		parseT.Fatalf("write main.go: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseRoot, "gwc-runner.json"), []byte(`{"paths":{"artifactRoot":"enterprise-artifacts"}}`), 0644); parseErr2 != nil {
		parseT.Fatalf("write gwc-runner.json: %v", parseErr2)
	}

	parseOriginalGetwd := buildGetwd
	parseT.Cleanup(func() { buildGetwd = parseOriginalGetwd })
	buildGetwd = func() (string, error) { return parseRoot, nil }

	parseConfig, parseErr3 := resolveReleaseConfig(releaseConfig{})
	if parseErr3 != nil {
		parseT.Fatalf("resolve release config: %v", parseErr3)
	}
	parseWant := filepath.Join(parseRoot, "enterprise-artifacts", filepath.Base(parseRoot), "wasm-release")
	if parseConfig.outDir != parseWant {
		parseT.Fatalf("expected artifact-root release out dir %q, got %#v", parseWant, parseConfig)
	}
	if parseConfig.resolution["output"] != "gwc-runner.json paths.artifactRoot" {
		parseT.Fatalf("expected explicit runner-config release tracing, got %#v", parseConfig.resolution)
	}
}

func TestReleaseStartupProbeHelpers(parseT *testing.T) {
	parseT.Run("transport encoding priority", func(parseT2 *testing.T) {
		if parseGot := releaseStartupTransportEncoding(map[string]releaseArtifactRecord{"gzip": {Path: "app.wasm.gz"}}); parseGot != "gzip" {
			parseT2.Fatalf("expected gzip transport priority, got %q", parseGot)
		}
		if parseGot2 := releaseStartupTransportEncoding(map[string]releaseArtifactRecord{"brotli": {Path: "app.wasm.br"}}); parseGot2 != "br" {
			parseT2.Fatalf("expected brotli transport, got %q", parseGot2)
		}
		if parseGot3 := releaseStartupTransportEncoding(nil); parseGot3 != "identity" {
			parseT2.Fatalf("expected identity fallback transport, got %q", parseGot3)
		}
	})

	parseT.Run("serve startup wasm by encoding", func(parseT3 *testing.T) {
		parseOutDir := parseT3.TempDir()
		parseBinaryName := "app.wasm"
		if parseErr := os.WriteFile(filepath.Join(parseOutDir, parseBinaryName), []byte("identity"), 0644); parseErr != nil {
			parseT3.Fatalf("write wasm: %v", parseErr)
		}
		if parseErr2 := os.WriteFile(filepath.Join(parseOutDir, parseBinaryName+".gz"), []byte("gzip"), 0644); parseErr2 != nil {
			parseT3.Fatalf("write wasm gzip: %v", parseErr2)
		}
		if parseErr3 := os.WriteFile(filepath.Join(parseOutDir, parseBinaryName+".br"), []byte("br"), 0644); parseErr3 != nil {
			parseT3.Fatalf("write wasm br: %v", parseErr3)
		}

		parseRec := httptest.NewRecorder()
		parseReq := httptest.NewRequest(http.MethodGet, "/"+parseBinaryName, nil)
		releaseServeStartupWasm(parseRec, parseReq, parseOutDir, parseBinaryName, "gzip")
		if parseRec.Code != http.StatusOK || parseRec.Header().Get("Content-Encoding") != "gzip" {
			parseT3.Fatalf("expected gzip response, status=%d headers=%v", parseRec.Code, parseRec.Header())
		}

		parseRec = httptest.NewRecorder()
		parseReq = httptest.NewRequest(http.MethodGet, "/"+parseBinaryName, nil)
		releaseServeStartupWasm(parseRec, parseReq, parseOutDir, parseBinaryName, "br")
		if parseRec.Code != http.StatusOK || parseRec.Header().Get("Content-Encoding") != "br" {
			parseT3.Fatalf("expected brotli response, status=%d headers=%v", parseRec.Code, parseRec.Header())
		}

		parseRec = httptest.NewRecorder()
		parseReq = httptest.NewRequest(http.MethodGet, "/"+parseBinaryName, nil)
		releaseServeStartupWasm(parseRec, parseReq, parseOutDir, parseBinaryName, "identity")
		if parseRec.Code != http.StatusOK || parseRec.Header().Get("Content-Encoding") != "" {
			parseT3.Fatalf("expected identity response, status=%d headers=%v", parseRec.Code, parseRec.Header())
		}
	})

	parseT.Run("render startup probe html", func(parseT4 *testing.T) {
		parseRendered := renderReleaseStartupProbeHTML("app.wasm", "gzip", 4321)
		for _, parseWant := range []string{
			"GWC Release Startup Probe",
			"transportEncoding:",
			"gzip",
			"const gzipProbePath =",
			"/app.wasm.gz",
			"setTimeout(() => markReady('timeout'), 4321);",
		} {
			if !strings.Contains(parseRendered, parseWant) {
				parseT4.Fatalf("expected probe html to contain %q", parseWant)
			}
		}
		parseIdentity := renderReleaseStartupProbeHTML("app.wasm", "identity", 100)
		if !strings.Contains(parseIdentity, "const gzipProbePath = null;") {
			parseT4.Fatalf("expected identity probe html to skip gzip probe path")
		}
	})

	parseT.Run("smoke fetch wasm headers", func(parseT5 *testing.T) {
		parseOutDir2 := parseT5.TempDir()
		parseBinaryName2 := "app.wasm"
		if parseErr4 := os.WriteFile(filepath.Join(parseOutDir2, parseBinaryName2), []byte("wasm"), 0644); parseErr4 != nil {
			parseT5.Fatalf("write wasm: %v", parseErr4)
		}
		if parseErr5 := os.WriteFile(filepath.Join(parseOutDir2, parseBinaryName2+".gz"), []byte("gzip"), 0644); parseErr5 != nil {
			parseT5.Fatalf("write wasm gzip: %v", parseErr5)
		}

		parseContentType, parseContentEncoding, parseErr6 := releaseSmokeFetchWasmHeaders(parseOutDir2, parseBinaryName2, map[string]releaseArtifactRecord{
			"gzip": {Path: parseBinaryName2 + ".gz"},
		})
		if parseErr6 != nil {
			parseT5.Fatalf("release smoke fetch wasm headers: %v", parseErr6)
		}
		if !strings.Contains(strings.ToLower(parseContentType), "application/wasm") {
			parseT5.Fatalf("expected wasm content type, got %q", parseContentType)
		}
		if parseContentEncoding != "gzip" {
			parseT5.Fatalf("expected gzip encoding, got %q", parseContentEncoding)
		}

		if _, _, parseErr7 := releaseSmokeFetchWasmHeaders(parseOutDir2, "missing.wasm", nil); parseErr7 == nil {
			parseT5.Fatalf("expected fetch failure for missing wasm")
		}
	})

	parseT.Run("startup probe server routes", func(parseT6 *testing.T) {
		parseOutDir3 := parseT6.TempDir()
		parseBinaryName3 := "app.wasm"
		parseWasmExecPath := filepath.Join(parseOutDir3, "wasm_exec.js")
		if parseErr8 := os.WriteFile(filepath.Join(parseOutDir3, parseBinaryName3), []byte("wasm"), 0644); parseErr8 != nil {
			parseT6.Fatalf("write wasm: %v", parseErr8)
		}
		if parseErr9 := os.WriteFile(filepath.Join(parseOutDir3, parseBinaryName3+".br"), []byte("brotli"), 0644); parseErr9 != nil {
			parseT6.Fatalf("write wasm br: %v", parseErr9)
		}
		if parseErr10 := os.WriteFile(parseWasmExecPath, []byte("console.log('exec');"), 0644); parseErr10 != nil {
			parseT6.Fatalf("write wasm_exec.js: %v", parseErr10)
		}

		parseProbeURL, parseEncoding, parseShutdown, parseErr11 := startReleaseStartupProbeServer(parseOutDir3, parseBinaryName3, parseWasmExecPath, map[string]releaseArtifactRecord{
			"brotli": {Path: parseBinaryName3 + ".br"},
		}, 2500)
		if parseErr11 != nil {
			parseT6.Fatalf("start startup probe server: %v", parseErr11)
		}
		defer parseShutdown()
		if parseEncoding != "br" {
			parseT6.Fatalf("expected brotli transport encoding, got %q", parseEncoding)
		}
		if !strings.Contains(parseProbeURL, "/__gwc/startup-probe.html") {
			parseT6.Fatalf("unexpected probe url %q", parseProbeURL)
		}

		parseResp, parseErr11 := http.Get(parseProbeURL) // #nosec G107 -- local ephemeral test server URL
		if parseErr11 != nil {
			parseT6.Fatalf("fetch probe html: %v", parseErr11)
		}
		parseProbeBody, _ := io.ReadAll(parseResp.Body)
		_ = parseResp.Body.Close()
		if parseResp.StatusCode != http.StatusOK || !strings.Contains(string(parseProbeBody), "GWC Release Startup Probe") {
			parseT6.Fatalf("unexpected probe html response: status=%d body=%q", parseResp.StatusCode, string(parseProbeBody))
		}

		parseBaseURL := strings.TrimSuffix(parseProbeURL, "/__gwc/startup-probe.html")
		parseNoFollow := &http.Client{
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		parseResp, parseErr11 = parseNoFollow.Get(parseBaseURL + "/")
		if parseErr11 != nil {
			parseT6.Fatalf("fetch startup probe root redirect: %v", parseErr11)
		}
		_ = parseResp.Body.Close()
		if parseResp.StatusCode != http.StatusTemporaryRedirect {
			parseT6.Fatalf("expected startup probe root redirect, got %d", parseResp.StatusCode)
		}

		parseResp, parseErr11 = http.Get(parseBaseURL + "/__gwc/wasm_exec.js") // #nosec G107 -- local ephemeral test server URL
		if parseErr11 != nil {
			parseT6.Fatalf("fetch wasm_exec.js: %v", parseErr11)
		}
		parseExecBody, _ := io.ReadAll(parseResp.Body)
		_ = parseResp.Body.Close()
		if parseResp.StatusCode != http.StatusOK || !strings.Contains(string(parseExecBody), "exec") {
			parseT6.Fatalf("unexpected wasm_exec response: status=%d body=%q", parseResp.StatusCode, string(parseExecBody))
		}

		parseResp, parseErr11 = http.Get(parseBaseURL + "/" + parseBinaryName3) // #nosec G107 -- local ephemeral test server URL
		if parseErr11 != nil {
			parseT6.Fatalf("fetch startup wasm: %v", parseErr11)
		}
		_ = parseResp.Body.Close()
		if parseResp.StatusCode != http.StatusOK || parseResp.Header.Get("Content-Encoding") != "br" {
			parseT6.Fatalf("unexpected startup wasm response: status=%d headers=%v", parseResp.StatusCode, parseResp.Header)
		}
	})
}

func TestDerefInt64(parseT *testing.T) {
	if parseGot := derefInt64(nil); parseGot != 0 {
		parseT.Fatalf("derefInt64(nil) = %d, want 0", parseGot)
	}
	parseValue := int64(42)
	if parseGot2 := derefInt64(&parseValue); parseGot2 != 42 {
		parseT.Fatalf("derefInt64(&42) = %d, want 42", parseGot2)
	}
}

func TestRunReleaseHandlesHelpAndInvalidFlags(parseT *testing.T) {
	if parseErr := (launcher{}).runRelease([]string{"-help"}); parseErr != nil {
		parseT.Fatalf("expected release help to succeed, got %v", parseErr)
	}
	if parseErr2 := (launcher{}).runRelease([]string{"-definitely-invalid"}); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "flag provided but not defined") {
		parseT.Fatalf("expected invalid release flag error, got %v", parseErr2)
	}
}

func TestRunReleaseEnforcesEnterprisePolicy(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	if parseErr := os.WriteFile(filepath.Join(parseTempApp, "go.mod"), []byte("module example.com/gwcreleasepolicy\n\ngo 1.25.0\n"), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}

	parseOriginalPolicy := launcherActiveEnterpriseConfig
	parseOriginalRunCommand := launcherRunCommand
	parseT.Cleanup(func() {
		launcherActiveEnterpriseConfig = parseOriginalPolicy
		launcherRunCommand = parseOriginalRunCommand
	})

	parseT.Run("requires budgets when policy enabled", func(parseT2 *testing.T) {
		isParseRequired := true
		launcherActiveEnterpriseConfig = launcherEnterpriseConfig{
			Policy: launcherEnterprisePolicy{
				RequireReleaseBudgets: &isParseRequired,
			},
		}
		parseErr3 := (launcher{}).runRelease([]string{"-app", parseMainPath, "-root", parseTempApp, "-out-dir", filepath.Join(parseTempApp, "dist")})
		if parseErr3 == nil || !strings.Contains(parseErr3.Error(), "requires release budgets") {
			parseT2.Fatalf("expected required budgets policy failure, got %v", parseErr3)
		}
	})

	parseT.Run("requires release compression policy match", func(parseT3 *testing.T) {
		launcherActiveEnterpriseConfig = launcherEnterpriseConfig{
			Policy: launcherEnterprisePolicy{
				RequiredReleaseCompression: "gzip+brotli",
			},
		}
		parseErr4 := (launcher{}).runRelease([]string{"-app", parseMainPath, "-root", parseTempApp, "-out-dir", filepath.Join(parseTempApp, "dist"), "-compression", "gzip"})
		if parseErr4 == nil || !strings.Contains(parseErr4.Error(), "does not satisfy enterprise requirement") {
			parseT3.Fatalf("expected compression policy failure, got %v", parseErr4)
		}
	})

	parseT.Run("requires artifact naming patterns", func(parseT4 *testing.T) {
		launcherActiveEnterpriseConfig = launcherEnterpriseConfig{
			Policy: launcherEnterprisePolicy{
				ReleaseBinaryPattern:   "^corp-[a-z0-9._-]+\\.wasm$",
				ReleaseManifestPattern: "^corp-manifest\\.json$",
			},
		}
		parseErr5 := (launcher{}).runRelease([]string{"-app", parseMainPath, "-root", parseTempApp, "-out-dir", filepath.Join(parseTempApp, "dist"), "-binary-name", "app.wasm", "-manifest-name", "manifest.json", "-compression", "none"})
		if parseErr5 == nil || !strings.Contains(parseErr5.Error(), "release binary name") {
			parseT4.Fatalf("expected binary naming policy failure, got %v", parseErr5)
		}
	})

	parseT.Run("requires approved go toolchain", func(parseT5 *testing.T) {
		launcherActiveEnterpriseConfig = launcherEnterpriseConfig{
			Policy: launcherEnterprisePolicy{
				ApprovedGoToolchains: []string{"go1.26.x"},
			},
		}
		launcherRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
			if parseCommand == "go" && len(parseArgs) == 2 && parseArgs[0] == "env" && parseArgs[1] == "GOVERSION" {
				return "go1.25.2", nil
			}
			return "", nil
		}
		parseErr6 := (launcher{}).runRelease([]string{"-app", parseMainPath, "-root", parseTempApp, "-out-dir", filepath.Join(parseTempApp, "dist"), "-compression", "none"})
		if parseErr6 == nil || !strings.Contains(parseErr6.Error(), "not approved") {
			parseT5.Fatalf("expected approved-go-toolchain policy failure, got %v", parseErr6)
		}
	})
}

func TestRunReleaseJSONBuildsManifestAndCompressedSidecars(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseGoModPath := filepath.Join(parseTempApp, "go.mod")
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	parseOutDir := filepath.Join(parseTempApp, "dist", "release")
	if parseErr := os.WriteFile(parseGoModPath, []byte("module example.com/gwcreleasetest\n\ngo 1.25.0\n"), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	parseLauncher := launcher{}
	if parseErr4 := parseLauncher.run([]string{"release", "-app", parseMainPath, "-root", parseTempApp, "-out-dir", parseOutDir, "-binary-name", "app.wasm", "-json"}); parseErr4 != nil {
		parseT.Fatalf("run release: %v", parseErr4)
	}

	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr3)
	}
	var parseSummary releaseSummary
	if parseErr5 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr5 != nil {
		parseT.Fatalf("unmarshal release summary: %v\n%s", parseErr5, parseOutput)
	}
	if !parseSummary.OK {
		parseT.Fatalf("expected successful release summary, got %#v", parseSummary)
	}
	parseWasmArtifact, parseOk := parseSummary.Artifacts["wasm"]
	if !parseOk || parseWasmArtifact.Bytes <= 0 {
		parseT.Fatalf("expected wasm artifact in release summary, got %#v", parseSummary)
	}
	parseGzipArtifact, parseOk := parseSummary.Artifacts["gzip"]
	if !parseOk || parseGzipArtifact.Bytes <= 0 {
		parseT.Fatalf("expected gzip artifact in release summary, got %#v", parseSummary)
	}
	parseBrotliArtifact, parseOk := parseSummary.Artifacts["brotli"]
	if !parseOk || parseBrotliArtifact.Bytes <= 0 {
		parseT.Fatalf("expected brotli artifact in release summary, got %#v", parseSummary)
	}
	if _, parseErr6 := os.Stat(filepath.Join(parseOutDir, "app.wasm")); parseErr6 != nil {
		parseT.Fatalf("expected raw release artifact: %v", parseErr6)
	}
	if _, parseErr7 := os.Stat(filepath.Join(parseOutDir, "app.wasm.gz")); parseErr7 != nil {
		parseT.Fatalf("expected gzip release artifact: %v", parseErr7)
	}
	if _, parseErr8 := os.Stat(filepath.Join(parseOutDir, "app.wasm.br")); parseErr8 != nil {
		parseT.Fatalf("expected brotli release artifact: %v", parseErr8)
	}
	parseManifestPath := filepath.Join(parseOutDir, "wasm-release-manifest.json")
	parseManifestBytes, parseErr3 := os.ReadFile(parseManifestPath)
	if parseErr3 != nil {
		parseT.Fatalf("read release manifest: %v", parseErr3)
	}
	var parseManifest map[string]interface{}
	if parseErr9 := json.Unmarshal(parseManifestBytes, &parseManifest); parseErr9 != nil {
		parseT.Fatalf("unmarshal release manifest: %v\n%s", parseErr9, string(parseManifestBytes))
	}
	if parseManifest["profile"] != "release" {
		parseT.Fatalf("expected release manifest profile, got %#v", parseManifest)
	}
	parseFlags, parseOk := parseManifest["flags"].(map[string]interface{})
	if !parseOk || parseFlags["brotli"] != true {
		parseT.Fatalf("expected release manifest to report brotli packaging, got %#v", parseManifest)
	}
	if parseFlags["compressionPolicy"] != "gzip+brotli" {
		parseT.Fatalf("expected release manifest compression policy gzip+brotli, got %#v", parseManifest)
	}
}

func TestRunReleaseCompressionBrotliOnly(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseGoModPath := filepath.Join(parseTempApp, "go.mod")
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	parseOutDir := filepath.Join(parseTempApp, "dist", "release")
	if parseErr := os.WriteFile(parseGoModPath, []byte("module example.com/gwcreleasebrotli\n\ngo 1.25.0\n"), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	parseLauncher := launcher{}
	if parseErr4 := parseLauncher.run([]string{"release", "-app", parseMainPath, "-root", parseTempApp, "-out-dir", parseOutDir, "-binary-name", "app.wasm", "-compression", "brotli", "-json"}); parseErr4 != nil {
		parseT.Fatalf("run release: %v", parseErr4)
	}

	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr3)
	}
	var parseSummary releaseSummary
	if parseErr5 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr5 != nil {
		parseT.Fatalf("unmarshal release summary: %v\n%s", parseErr5, parseOutput)
	}
	if _, parseOk := parseSummary.Artifacts["brotli"]; !parseOk {
		parseT.Fatalf("expected brotli artifact in release summary, got %#v", parseSummary)
	}
	if _, parseOk2 := parseSummary.Artifacts["gzip"]; parseOk2 {
		parseT.Fatalf("expected gzip artifact to be omitted for brotli-only release, got %#v", parseSummary)
	}
	if _, parseErr6 := os.Stat(filepath.Join(parseOutDir, "app.wasm.br")); parseErr6 != nil {
		parseT.Fatalf("expected brotli release artifact: %v", parseErr6)
	}
	if _, parseErr7 := os.Stat(filepath.Join(parseOutDir, "app.wasm.gz")); !os.IsNotExist(parseErr7) {
		parseT.Fatalf("expected gzip artifact to be absent for brotli-only release, stat err=%v", parseErr7)
	}
	if parseSummary.Flags["compressionPolicy"] != "brotli" {
		parseT.Fatalf("expected compression policy brotli, got %#v", parseSummary)
	}
}

func TestRunReleaseRejectsConflictingCompressionFlags(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	if parseErr := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr != nil {
		parseT.Fatalf("write main.go: %v", parseErr)
	}

	parseLauncher := launcher{}
	parseErr2 := parseLauncher.run([]string{"release", "-app", parseMainPath, "-root", parseTempApp, "-compression", "brotli", "-skip-compression"})
	if parseErr2 == nil {
		parseT.Fatal("expected conflicting compression flags to fail")
	}
	if !strings.Contains(parseErr2.Error(), "use either -compression or -skip-compression") {
		parseT.Fatalf("expected conflicting compression flag error, got %v", parseErr2)
	}
}

func TestLoadReleaseBudgetsParsesNumericValues(parseT *testing.T) {
	parseBudgetsPath := filepath.Join(parseT.TempDir(), "budgets.json")
	if parseErr := os.WriteFile(parseBudgetsPath, []byte(`{"raw_bytes":1234,"gzip_bytes":567,"brotli_bytes":345}`), 0644); parseErr != nil {
		parseT.Fatalf("write budgets file: %v", parseErr)
	}

	parseBudgets, parseErr2 := loadReleaseBudgets(parseBudgetsPath)
	if parseErr2 != nil {
		parseT.Fatalf("load release budgets: %v", parseErr2)
	}
	if parseBudgets["raw_bytes"] != 1234 || parseBudgets["gzip_bytes"] != 567 || parseBudgets["brotli_bytes"] != 345 {
		parseT.Fatalf("unexpected budgets map: %#v", parseBudgets)
	}
}

func TestLoadReleaseBudgetsRejectsInvalidFiles(parseT *testing.T) {
	parseT.Run("parse error", func(parseT2 *testing.T) {
		parseBudgetsPath := filepath.Join(parseT2.TempDir(), "budgets.json")
		if parseErr := os.WriteFile(parseBudgetsPath, []byte(`{"raw_bytes":`), 0644); parseErr != nil {
			parseT2.Fatalf("write invalid budgets file: %v", parseErr)
		}
		if _, parseErr2 := loadReleaseBudgets(parseBudgetsPath); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "parse budgets file") {
			parseT2.Fatalf("expected parse error, got %v", parseErr2)
		}
	})

	parseT.Run("non numeric", func(parseT3 *testing.T) {
		parseBudgetsPath2 := filepath.Join(parseT3.TempDir(), "budgets.json")
		if parseErr3 := os.WriteFile(parseBudgetsPath2, []byte(`{"raw_bytes":"huge"}`), 0644); parseErr3 != nil {
			parseT3.Fatalf("write nonnumeric budgets file: %v", parseErr3)
		}
		if _, parseErr4 := loadReleaseBudgets(parseBudgetsPath2); parseErr4 == nil || !strings.Contains(parseErr4.Error(), `budget "raw_bytes" must be numeric`) {
			parseT3.Fatalf("expected numeric validation error, got %v", parseErr4)
		}
	})
}

func TestAssertReleaseBudgetsPassesAndFails(parseT *testing.T) {
	parseArtifacts := map[string]releaseArtifactRecord{
		"wasm":   {Path: "app.wasm", Bytes: 100},
		"gzip":   {Path: "app.wasm.gz", Bytes: 50},
		"brotli": {Path: "app.wasm.br", Bytes: 40},
	}

	if parseErr := assertReleaseBudgets(map[string]int64{"raw_bytes": 100, "gzip_bytes": 50, "brotli_bytes": 40}, parseArtifacts); parseErr != nil {
		parseT.Fatalf("expected matching budgets to pass, got %v", parseErr)
	}
	if parseErr2 := assertReleaseBudgets(map[string]int64{"raw_bytes": 99}, parseArtifacts); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "artifact budget exceeded for raw wasm") {
		parseT.Fatalf("expected raw wasm budget failure, got %v", parseErr2)
	}
	if parseErr3 := assertReleaseBudgets(map[string]int64{"gzip_bytes": 49}, parseArtifacts); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "artifact budget exceeded for gzip sidecar") {
		parseT.Fatalf("expected gzip budget failure, got %v", parseErr3)
	}
	if parseErr4 := assertReleaseBudgets(map[string]int64{"brotli_bytes": 39}, parseArtifacts); parseErr4 == nil || !strings.Contains(parseErr4.Error(), "artifact budget exceeded for brotli sidecar") {
		parseT.Fatalf("expected brotli budget failure, got %v", parseErr4)
	}
	if parseErr5 := assertReleaseBudgets(map[string]int64{"gzip_bytes": 1}, map[string]releaseArtifactRecord{"wasm": {Path: "app.wasm", Bytes: 100}}); parseErr5 != nil {
		parseT.Fatalf("expected missing artifact budget check to be skipped, got %v", parseErr5)
	}
}

func TestRunReleaseHelpReturnsNil(parseT *testing.T) {
	parseLauncher := launcher{}
	if parseErr := parseLauncher.run([]string{"release", "-help"}); parseErr != nil {
		parseT.Fatalf("expected release help to succeed, got %v", parseErr)
	}
}

func TestRunReleasePrintsSummaryWithoutJSON(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	parseOutDir := filepath.Join(parseTempApp, "dist", "release")
	if parseErr := os.WriteFile(filepath.Join(parseTempApp, "go.mod"), []byte("module example.com/gwcreleasetext\n\ngo 1.25.0\n"), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	if parseErr4 := (launcher{}).run([]string{"release", "-app", parseMainPath, "-root", parseTempApp, "-out-dir", parseOutDir, "-compression", "none"}); parseErr4 != nil {
		parseT.Fatalf("run release: %v", parseErr4)
	}
	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read stdout: %v", parseErr3)
	}
	for _, parseExpected := range []string{"GWC release", "out dir:      " + parseOutDir, "artifact[wasm]: app.wasm"} {
		if !strings.Contains(parseOutput, parseExpected) {
			parseT.Fatalf("expected release output to contain %q, got:\n%s", parseExpected, parseOutput)
		}
	}
}

func TestRunReleaseSkipCompressionProducesOnlyRawArtifact(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseGoModPath := filepath.Join(parseTempApp, "go.mod")
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	parseOutDir := filepath.Join(parseTempApp, "dist", "release")
	if parseErr := os.WriteFile(parseGoModPath, []byte("module example.com/gwcreleasenone\n\ngo 1.25.0\n"), 0644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseErr2)
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	parseLauncher := launcher{}
	if parseErr4 := parseLauncher.run([]string{"release", "-app", parseMainPath, "-root", parseTempApp, "-out-dir", parseOutDir, "-skip-compression", "-json"}); parseErr4 != nil {
		parseT.Fatalf("run release with skip-compression: %v", parseErr4)
	}

	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr3)
	}
	var parseSummary releaseSummary
	if parseErr5 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr5 != nil {
		parseT.Fatalf("unmarshal release summary: %v\n%s", parseErr5, parseOutput)
	}
	if _, parseOk := parseSummary.Artifacts["wasm"]; !parseOk {
		parseT.Fatalf("expected raw wasm artifact, got %#v", parseSummary)
	}
	if _, parseOk2 := parseSummary.Artifacts["gzip"]; parseOk2 {
		parseT.Fatalf("expected gzip artifact to be omitted, got %#v", parseSummary)
	}
	if _, parseOk3 := parseSummary.Artifacts["brotli"]; parseOk3 {
		parseT.Fatalf("expected brotli artifact to be omitted, got %#v", parseSummary)
	}
	if parseSummary.Flags["compressionPolicy"] != "none" || parseSummary.Flags["compression"] != false {
		parseT.Fatalf("expected no-compression flags, got %#v", parseSummary.Flags)
	}
	if _, parseErr6 := os.Stat(filepath.Join(parseOutDir, "app.wasm.gz")); !os.IsNotExist(parseErr6) {
		parseT.Fatalf("expected gzip sidecar to be absent, stat err=%v", parseErr6)
	}
	if _, parseErr7 := os.Stat(filepath.Join(parseOutDir, "app.wasm.br")); !os.IsNotExist(parseErr7) {
		parseT.Fatalf("expected brotli sidecar to be absent, stat err=%v", parseErr7)
	}
}

func TestResolveReleaseConfigRejectsInvalidNames(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseTempApp, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); parseErr != nil {
		parseT.Fatalf("write main.go: %v", parseErr)
	}
	parseOriginalGetwd := buildGetwd
	parseT.Cleanup(func() { buildGetwd = parseOriginalGetwd })
	buildGetwd = func() (string, error) { return parseTempApp, nil }

	_, parseErr2 := resolveReleaseConfig(releaseConfig{binaryName: "."})
	if parseErr2 == nil || !strings.Contains(parseErr2.Error(), "release binary name is required") {
		parseT.Fatalf("expected invalid binary name error, got %v", parseErr2)
	}

	_, parseErr2 = resolveReleaseConfig(releaseConfig{manifestName: "."})
	if parseErr2 == nil || !strings.Contains(parseErr2.Error(), "release manifest name is required") {
		parseT.Fatalf("expected invalid manifest name error, got %v", parseErr2)
	}
}

func TestResolveReleaseConfigDirectoryAppPathAndInvalidMetadata(parseT *testing.T) {
	parseT.Run("directory app path uses app directory defaults", func(parseT2 *testing.T) {
		parseRoot := parseT2.TempDir()
		parseAppDir := filepath.Join(parseRoot, "cmd", "web")
		if parseErr := os.MkdirAll(parseAppDir, 0755); parseErr != nil {
			parseT2.Fatalf("mkdir app dir: %v", parseErr)
		}
		if parseErr2 := os.WriteFile(filepath.Join(parseAppDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
			parseT2.Fatalf("write main.go: %v", parseErr2)
		}

		parseOriginalGetwd := buildGetwd
		parseT2.Cleanup(func() { buildGetwd = parseOriginalGetwd })
		buildGetwd = func() (string, error) { return parseRoot, nil }

		parseConfig, parseErr3 := resolveReleaseConfig(releaseConfig{appPath: parseAppDir, compression: "gzip", compressionSet: true})
		if parseErr3 != nil {
			parseT2.Fatalf("resolve release config: %v", parseErr3)
		}
		if parseConfig.rootPath != parseAppDir || parseConfig.outDir != filepath.Join(parseAppDir, "bin", "wasm-release") || parseConfig.binaryName != "app.wasm" {
			parseT2.Fatalf("expected directory app path defaults, got %#v", parseConfig)
		}
	})

	parseT.Run("invalid metadata bubbles parse error", func(parseT3 *testing.T) {
		parseRoot2 := parseT3.TempDir()
		if parseErr4 := os.WriteFile(filepath.Join(parseRoot2, "gwc-start.json"), []byte(`{"tooling":`), 0644); parseErr4 != nil {
			parseT3.Fatalf("write invalid metadata: %v", parseErr4)
		}
		parseOriginalGetwd2 := buildGetwd
		parseT3.Cleanup(func() { buildGetwd = parseOriginalGetwd2 })
		buildGetwd = func() (string, error) { return parseRoot2, nil }
		if _, parseErr5 := resolveReleaseConfig(releaseConfig{}); parseErr5 == nil || !strings.Contains(parseErr5.Error(), "parse scaffold metadata") {
			parseT3.Fatalf("expected invalid metadata error, got %v", parseErr5)
		}
	})
}

func TestResolveReleaseConfigRejectsInvalidMetadataCompressionAndErrors(parseT *testing.T) {
	parseT.Run("invalid metadata compression policy", func(parseT2 *testing.T) {
		parseRoot := parseT2.TempDir()
		if parseErr := os.WriteFile(filepath.Join(parseRoot, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); parseErr != nil {
			parseT2.Fatalf("write main.go: %v", parseErr)
		}
		parseMetadata := `{
  "projectName": "metadata-release-app",
  "modulePath": "example.com/metadata-release-app",
  "tooling": {
    "appPath": "main.go",
    "releaseCompression": "mystery"
  }
}
`
		if parseErr2 := os.WriteFile(filepath.Join(parseRoot, "gwc-start.json"), []byte(parseMetadata), 0644); parseErr2 != nil {
			parseT2.Fatalf("write gwc-start.json: %v", parseErr2)
		}
		parseOriginalGetwd := buildGetwd
		parseT2.Cleanup(func() { buildGetwd = parseOriginalGetwd })
		buildGetwd = func() (string, error) { return parseRoot, nil }

		if _, parseErr3 := resolveReleaseConfig(releaseConfig{}); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "unknown release compression policy") {
			parseT2.Fatalf("expected metadata compression error, got %v", parseErr3)
		}
	})

	parseT.Run("cwd error", func(parseT3 *testing.T) {
		parseOriginalGetwd2 := buildGetwd
		parseT3.Cleanup(func() { buildGetwd = parseOriginalGetwd2 })
		buildGetwd = func() (string, error) { return "", errors.New("cwd failed") }

		if _, parseErr4 := resolveReleaseConfig(releaseConfig{}); parseErr4 == nil || !strings.Contains(parseErr4.Error(), "cwd failed") {
			parseT3.Fatalf("expected cwd error, got %v", parseErr4)
		}
	})

	parseT.Run("missing explicit app path", func(parseT4 *testing.T) {
		parseRoot2 := parseT4.TempDir()
		parseOriginalGetwd3 := buildGetwd
		parseT4.Cleanup(func() { buildGetwd = parseOriginalGetwd3 })
		buildGetwd = func() (string, error) { return parseRoot2, nil }

		_, parseErr5 := resolveReleaseConfig(releaseConfig{appPath: filepath.Join(parseRoot2, "missing.go")})
		if parseErr5 == nil || !strings.Contains(parseErr5.Error(), "resolve app path") {
			parseT4.Fatalf("expected missing app path error, got %v", parseErr5)
		}
	})
}

func TestNormalizeReleaseCompressionPolicyAliases(parseT *testing.T) {
	parseTests := map[string]string{
		"":            "gzip",
		"gzip":        "gzip",
		"br":          "brotli",
		"both":        "gzip+brotli",
		"off":         "none",
		"disabled":    "none",
		"brotli+gzip": "gzip+brotli",
	}
	for parseInput, parseWant := range parseTests {
		parseGot, parseErr := normalizeReleaseCompressionPolicy(parseInput)
		if parseErr != nil {
			parseT.Fatalf("normalize compression %q: %v", parseInput, parseErr)
		}
		if parseGot != parseWant {
			parseT.Fatalf("expected compression %q -> %q, got %q", parseInput, parseWant, parseGot)
		}
	}
	if _, parseErr2 := normalizeReleaseCompressionPolicy("mystery"); parseErr2 == nil {
		parseT.Fatal("expected unknown compression policy to fail")
	}
}

func TestNormalizeReleasePostLinkOptimizationAliases(parseT *testing.T) {
	parseTests := map[string]string{
		"":         "none",
		"none":     "none",
		"off":      "none",
		"wasm-opt": "wasm-opt",
		"size":     "wasm-opt",
	}
	for parseInput, parseWant := range parseTests {
		parseGot, parseErr := normalizeReleasePostLinkOptimization(parseInput)
		if parseErr != nil {
			parseT.Fatalf("normalize post-link optimization %q: %v", parseInput, parseErr)
		}
		if parseGot != parseWant {
			parseT.Fatalf("expected post-link optimization %q -> %q, got %q", parseInput, parseWant, parseGot)
		}
	}
	if _, parseErr2 := normalizeReleasePostLinkOptimization("mystery"); parseErr2 == nil {
		parseT.Fatal("expected unknown post-link optimization to fail")
	}
}

func TestNormalizeReleaseSizeAttributionModeAliases(parseT *testing.T) {
	parseTests := map[string]string{
		"":            "none",
		"none":        "none",
		"off":         "none",
		"packages":    "packages",
		"per-package": "packages",
	}
	for parseInput, parseWant := range parseTests {
		parseGot, parseErr := normalizeReleaseSizeAttributionMode(parseInput)
		if parseErr != nil {
			parseT.Fatalf("normalize size attribution mode %q: %v", parseInput, parseErr)
		}
		if parseGot != parseWant {
			parseT.Fatalf("expected size attribution mode %q -> %q, got %q", parseInput, parseWant, parseGot)
		}
	}
	if _, parseErr2 := normalizeReleaseSizeAttributionMode("mystery"); parseErr2 == nil {
		parseT.Fatal("expected unknown size attribution mode to fail")
	}
}

func TestNormalizeReleaseStartupMeasureModeAliases(parseT *testing.T) {
	parseTests := map[string]string{
		"":           "none",
		"none":       "none",
		"off":        "none",
		"browser":    "browser",
		"playwright": "browser",
	}
	for parseInput, parseWant := range parseTests {
		parseGot, parseErr := normalizeReleaseStartupMeasureMode(parseInput)
		if parseErr != nil {
			parseT.Fatalf("normalize startup measure mode %q: %v", parseInput, parseErr)
		}
		if parseGot != parseWant {
			parseT.Fatalf("expected startup measure mode %q -> %q, got %q", parseInput, parseWant, parseGot)
		}
	}
	if _, parseErr2 := normalizeReleaseStartupMeasureMode("mystery"); parseErr2 == nil {
		parseT.Fatal("expected unknown startup measure mode to fail")
	}
}

func TestReleaseArtifactAndSidecarHelpers(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseWasmPath := filepath.Join(parseRoot, "app.wasm")
	if parseErr := os.WriteFile(parseWasmPath, []byte("wasm-bytes"), 0644); parseErr != nil {
		parseT.Fatalf("write wasm artifact: %v", parseErr)
	}
	parseRecord, parseErr2 := releaseArtifactRecordForPath(parseRoot, parseWasmPath)
	if parseErr2 != nil {
		parseT.Fatalf("release artifact record: %v", parseErr2)
	}
	if parseRecord.Path != "app.wasm" || parseRecord.Bytes <= 0 || len(parseRecord.SHA256) != 64 {
		parseT.Fatalf("unexpected release artifact record: %#v", parseRecord)
	}

	parseGzipPath := parseWasmPath + ".gz"
	if parseErr3 := writeGzipSidecar(parseWasmPath, parseGzipPath); parseErr3 != nil {
		parseT.Fatalf("write gzip sidecar: %v", parseErr3)
	}
	if parseInfo, parseErr4 := os.Stat(parseGzipPath); parseErr4 != nil || parseInfo.Size() <= 0 {
		parseT.Fatalf("expected gzip sidecar to exist, stat err=%v info=%v", parseErr4, parseInfo)
	}

	parseBrotliPath := parseWasmPath + ".br"
	if parseErr5 := writeBrotliSidecar(parseWasmPath, parseBrotliPath); parseErr5 != nil {
		parseT.Fatalf("write brotli sidecar: %v", parseErr5)
	}
	if parseInfo2, parseErr6 := os.Stat(parseBrotliPath); parseErr6 != nil || parseInfo2.Size() <= 0 {
		parseT.Fatalf("expected brotli sidecar to exist, stat err=%v info=%v", parseErr6, parseInfo2)
	}

	if _, parseErr7 := releaseArtifactRecordForPath(parseRoot, filepath.Join(parseRoot, "missing.wasm")); parseErr7 == nil || !strings.Contains(parseErr7.Error(), "read release artifact") {
		parseT.Fatalf("expected missing artifact read error, got %v", parseErr7)
	}
	if parseErr8 := writeGzipSidecar(filepath.Join(parseRoot, "missing.wasm"), filepath.Join(parseRoot, "missing.gz")); parseErr8 == nil || !strings.Contains(parseErr8.Error(), "read source artifact for gzip") {
		parseT.Fatalf("expected gzip read error, got %v", parseErr8)
	}
	if parseErr9 := writeBrotliSidecar(filepath.Join(parseRoot, "missing.wasm"), filepath.Join(parseRoot, "missing.br")); parseErr9 == nil || !strings.Contains(parseErr9.Error(), "read source artifact for brotli") {
		parseT.Fatalf("expected brotli read error, got %v", parseErr9)
	}
}

func TestReleaseSidecarCreateErrorsAndGzipOnlyRelease(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseWasmPath := filepath.Join(parseRoot, "app.wasm")
	if parseErr := os.WriteFile(parseWasmPath, []byte("wasm-bytes"), 0644); parseErr != nil {
		parseT.Fatalf("write wasm artifact: %v", parseErr)
	}
	if parseErr2 := writeGzipSidecar(parseWasmPath, filepath.Join(parseRoot, "missing-dir", "app.wasm.gz")); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "create gzip sidecar") {
		parseT.Fatalf("expected gzip sidecar create error, got %v", parseErr2)
	}
	if parseErr3 := writeBrotliSidecar(parseWasmPath, filepath.Join(parseRoot, "missing-dir", "app.wasm.br")); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "create brotli sidecar") {
		parseT.Fatalf("expected brotli sidecar create error, got %v", parseErr3)
	}

	parseAppRoot := parseT.TempDir()
	parseMainPath := filepath.Join(parseAppRoot, "main.go")
	if parseErr4 := os.WriteFile(filepath.Join(parseAppRoot, "go.mod"), []byte("module example.com/gwcreleasegzip\n\ngo 1.25.0\n"), 0644); parseErr4 != nil {
		parseT.Fatalf("write go.mod: %v", parseErr4)
	}
	if parseErr5 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr5 != nil {
		parseT.Fatalf("write main.go: %v", parseErr5)
	}

	parseSummary, parseErr6 := executeRelease(releaseConfig{appPath: parseMainPath, rootPath: parseAppRoot, outDir: filepath.Join(parseAppRoot, "dist"), binaryName: "app.wasm", manifestName: "manifest.json", profile: "release", compression: "gzip"})
	if parseErr6 != nil {
		parseT.Fatalf("execute gzip-only release: %v", parseErr6)
	}
	if _, parseOk := parseSummary.Artifacts["gzip"]; !parseOk {
		parseT.Fatalf("expected gzip artifact, got %#v", parseSummary)
	}
	if _, parseOk2 := parseSummary.Artifacts["brotli"]; parseOk2 {
		parseT.Fatalf("expected brotli artifact to be omitted, got %#v", parseSummary)
	}
	if parseSummary.Flags["compressionPolicy"] != "gzip" {
		parseT.Fatalf("expected gzip compression policy, got %#v", parseSummary.Flags)
	}
}

func TestExecuteReleaseAppliesPostLinkOptimization(parseT *testing.T) {
	parseOriginalReleaseExecuteBuild := releaseExecuteBuild
	parseOriginalReleaseRunCommand := releaseRunCommand
	parseOriginalReleaseLookPath := releaseLookPath
	parseT.Cleanup(func() {
		releaseExecuteBuild = parseOriginalReleaseExecuteBuild
		releaseRunCommand = parseOriginalReleaseRunCommand
		releaseLookPath = parseOriginalReleaseLookPath
	})

	parseRoot := parseT.TempDir()
	parseOutDir := filepath.Join(parseRoot, "dist")
	parseWasmPath := filepath.Join(parseOutDir, "app.wasm")
	releaseExecuteBuild = func(parseConfig buildConfig) (buildSummary, error) {
		if parseErr := os.MkdirAll(filepath.Dir(parseConfig.outputPath), 0755); parseErr != nil {
			return buildSummary{}, parseErr
		}
		if parseErr2 := os.WriteFile(parseConfig.outputPath, []byte("raw-wasm"), 0644); parseErr2 != nil {
			return buildSummary{}, parseErr2
		}
		return buildSummary{
			OK:          true,
			Profile:     buildProfile{Name: "release"},
			AppPath:     parseConfig.appPath,
			ProjectRoot: parseConfig.rootPath,
			PackageDir:  parseRoot,
			OutputPath:  parseConfig.outputPath,
		}, nil
	}
	releaseLookPath = func(parseFile string) (string, error) {
		if parseFile == "wasm-opt" {
			return filepath.Join(parseRoot, "bin", "wasm-opt"), nil
		}
		return "", errors.New("not found")
	}
	releaseRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		if parseCommand != "wasm-opt" {
			parseT.Fatalf("expected wasm-opt command, got %q", parseCommand)
		}
		if parseCwd != parseOutDir {
			parseT.Fatalf("expected optimizer cwd %q, got %q", parseOutDir, parseCwd)
		}
		if len(parseArgs) != 4 || parseArgs[0] != parseWasmPath || parseArgs[1] != "-Oz" || parseArgs[2] != "-o" {
			parseT.Fatalf("unexpected optimizer args: %#v", parseArgs)
		}
		if parseErr3 := os.WriteFile(parseArgs[3], []byte("optimized-wasm"), 0644); parseErr3 != nil {
			parseT.Fatalf("write optimized wasm artifact: %v", parseErr3)
		}
		return "", nil
	}

	parseSummary, parseErr4 := executeRelease(releaseConfig{
		appPath:      filepath.Join(parseRoot, "main.go"),
		rootPath:     parseRoot,
		outDir:       parseOutDir,
		binaryName:   "app.wasm",
		manifestName: "manifest.json",
		profile:      "release",
		compression:  "none",
		postLinkOpt:  "wasm-opt",
	})
	if parseErr4 != nil {
		parseT.Fatalf("execute release with post-link optimization: %v", parseErr4)
	}
	if parseSummary.Optimizer == nil || parseSummary.Optimizer.Mode != "wasm-opt" {
		parseT.Fatalf("expected optimizer summary, got %#v", parseSummary)
	}
	if parseGot := parseSummary.Flags["postLinkOptimization"]; parseGot != "wasm-opt" {
		parseT.Fatalf("expected post-link optimization flag, got %#v", parseGot)
	}
	parseArtifactBytes, parseErr4 := os.ReadFile(parseWasmPath)
	if parseErr4 != nil {
		parseT.Fatalf("read optimized release artifact: %v", parseErr4)
	}
	if string(parseArtifactBytes) != "optimized-wasm" {
		parseT.Fatalf("expected optimized artifact bytes, got %q", string(parseArtifactBytes))
	}
	parseManifestBytes, parseErr4 := os.ReadFile(filepath.Join(parseOutDir, "manifest.json"))
	if parseErr4 != nil {
		parseT.Fatalf("read release manifest: %v", parseErr4)
	}
	var parseManifest struct {
		Optimizer *releaseOptimizerRecord `json:"optimizer"`
	}
	if parseErr5 := json.Unmarshal(parseManifestBytes, &parseManifest); parseErr5 != nil {
		parseT.Fatalf("unmarshal manifest: %v", parseErr5)
	}
	if parseManifest.Optimizer == nil || parseManifest.Optimizer.Mode != "wasm-opt" {
		parseT.Fatalf("expected manifest optimizer record, got %#v", parseManifest)
	}
}

func TestExecuteReleaseWritesPackageSizeAttribution(parseT *testing.T) {
	parseOriginalReleaseExecuteBuild := releaseExecuteBuild
	parseOriginalReleaseRunCommand := releaseRunCommand
	parseT.Cleanup(func() {
		releaseExecuteBuild = parseOriginalReleaseExecuteBuild
		releaseRunCommand = parseOriginalReleaseRunCommand
	})

	parseRoot := parseT.TempDir()
	parseOutDir := filepath.Join(parseRoot, "dist")
	parseExportDir := filepath.Join(parseRoot, "exports")
	if parseErr := os.MkdirAll(parseExportDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir exports: %v", parseErr)
	}
	parseLibSourceDir := filepath.Join(parseRoot, "lib")
	parseAppSourceDir := filepath.Join(parseRoot, "app")
	if parseErr2 := os.MkdirAll(parseLibSourceDir, 0755); parseErr2 != nil {
		parseT.Fatalf("mkdir lib dir: %v", parseErr2)
	}
	if parseErr3 := os.MkdirAll(parseAppSourceDir, 0755); parseErr3 != nil {
		parseT.Fatalf("mkdir app dir: %v", parseErr3)
	}
	parseLibSource := filepath.Join(parseLibSourceDir, "lib.go")
	parseAppSource := filepath.Join(parseAppSourceDir, "main.go")
	parseLibExport := filepath.Join(parseExportDir, "lib.a")
	if parseErr4 := os.WriteFile(parseLibSource, []byte("package lib\n"), 0644); parseErr4 != nil {
		parseT.Fatalf("write lib source: %v", parseErr4)
	}
	if parseErr5 := os.WriteFile(parseAppSource, []byte("package main\n"), 0644); parseErr5 != nil {
		parseT.Fatalf("write app source: %v", parseErr5)
	}
	if parseErr6 := os.WriteFile(parseLibExport, []byte("compiled-library-archive"), 0644); parseErr6 != nil {
		parseT.Fatalf("write export archive: %v", parseErr6)
	}

	releaseExecuteBuild = func(parseConfig buildConfig) (buildSummary, error) {
		if parseErr7 := os.MkdirAll(filepath.Dir(parseConfig.outputPath), 0755); parseErr7 != nil {
			return buildSummary{}, parseErr7
		}
		if parseErr8 := os.WriteFile(parseConfig.outputPath, []byte("raw-wasm"), 0644); parseErr8 != nil {
			return buildSummary{}, parseErr8
		}
		return buildSummary{
			OK:          true,
			Profile:     buildProfile{Name: "release"},
			AppPath:     parseConfig.appPath,
			ProjectRoot: parseConfig.rootPath,
			PackageDir:  parseAppSourceDir,
			OutputPath:  parseConfig.outputPath,
		}, nil
	}
	releaseRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		if parseCommand != "go" {
			parseT.Fatalf("expected go command, got %q", parseCommand)
		}
		if parseCwd != parseAppSourceDir {
			parseT.Fatalf("expected go list cwd %q, got %q", parseAppSourceDir, parseCwd)
		}
		if len(parseArgs) != 5 || parseArgs[0] != "list" || parseArgs[1] != "-deps" || parseArgs[2] != "-json" || parseArgs[3] != "-export" || parseArgs[4] != "." {
			parseT.Fatalf("unexpected go list args: %#v", parseArgs)
		}
		return `{"ImportPath":"example.com/lib","Dir":"` + filepath.ToSlash(parseLibSourceDir) + `","Export":"` + filepath.ToSlash(parseLibExport) + `","GoFiles":["lib.go"]}
{"ImportPath":"example.com/app","Dir":"` + filepath.ToSlash(parseAppSourceDir) + `","GoFiles":["main.go"]}
`, nil
	}

	parseSummary, parseErr9 := executeRelease(releaseConfig{
		appPath:         parseAppSource,
		rootPath:        parseRoot,
		outDir:          parseOutDir,
		binaryName:      "app.wasm",
		manifestName:    "manifest.json",
		profile:         "release",
		compression:     "none",
		sizeAttribution: "packages",
	})
	if parseErr9 != nil {
		parseT.Fatalf("execute release with size attribution: %v", parseErr9)
	}
	if parseSummary.Attribution == nil || parseSummary.Attribution.Mode != "packages" {
		parseT.Fatalf("expected size attribution summary, got %#v", parseSummary)
	}
	if _, parseOk := parseSummary.Artifacts["size_attribution"]; !parseOk {
		parseT.Fatalf("expected size attribution artifact, got %#v", parseSummary.Artifacts)
	}
	parseAttributionBytes, parseErr9 := os.ReadFile(filepath.Join(parseOutDir, "wasm-package-size-attribution.json"))
	if parseErr9 != nil {
		parseT.Fatalf("read size attribution artifact: %v", parseErr9)
	}
	var parsePayload struct {
		Mode     string                     `json:"mode"`
		Packages []releasePackageSizeRecord `json:"packages"`
	}
	if parseErr10 := json.Unmarshal(parseAttributionBytes, &parsePayload); parseErr10 != nil {
		parseT.Fatalf("unmarshal size attribution artifact: %v", parseErr10)
	}
	if parsePayload.Mode != "packages" || len(parsePayload.Packages) != 2 {
		parseT.Fatalf("unexpected size attribution payload: %#v", parsePayload)
	}
	if parsePayload.Packages[0].ImportPath != "example.com/lib" || parsePayload.Packages[0].ArchiveBytes <= 0 {
		parseT.Fatalf("expected library package to lead attribution, got %#v", parsePayload.Packages)
	}
}

func TestExecuteReleaseWritesDiffReport(parseT *testing.T) {
	parseOriginalReleaseExecuteBuild := releaseExecuteBuild
	parseOriginalReleaseRunCommand := releaseRunCommand
	parseT.Cleanup(func() {
		releaseExecuteBuild = parseOriginalReleaseExecuteBuild
		releaseRunCommand = parseOriginalReleaseRunCommand
	})

	parseRoot := parseT.TempDir()
	parseOutDir := filepath.Join(parseRoot, "dist")
	parseExportDir := filepath.Join(parseRoot, "exports")
	parseBaselineDir := filepath.Join(parseRoot, "baseline")
	parseLibSourceDir := filepath.Join(parseRoot, "lib")
	parseAppSourceDir := filepath.Join(parseRoot, "app")
	for _, parseDir := range []string{parseExportDir, parseBaselineDir, parseLibSourceDir, parseAppSourceDir} {
		if parseErr := os.MkdirAll(parseDir, 0755); parseErr != nil {
			parseT.Fatalf("mkdir %s: %v", parseDir, parseErr)
		}
	}
	parseLibSource := filepath.Join(parseLibSourceDir, "lib.go")
	parseAppSource := filepath.Join(parseAppSourceDir, "main.go")
	parseLibExport := filepath.Join(parseExportDir, "lib.a")
	if parseErr2 := os.WriteFile(parseLibSource, []byte("package lib\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write lib source: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(parseAppSource, []byte("package main\n"), 0644); parseErr3 != nil {
		parseT.Fatalf("write app source: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(parseLibExport, []byte("compiled-library-archive-that-grew"), 0644); parseErr4 != nil {
		parseT.Fatalf("write export archive: %v", parseErr4)
	}
	parseBaselineAttributionPath := filepath.Join(parseBaselineDir, "wasm-package-size-attribution.json")
	if parseErr5 := os.WriteFile(parseBaselineAttributionPath, []byte(`{
  "mode": "packages",
  "packages": [
    {"importPath":"example.com/lib","archiveBytes":8,"sourceBytes":12,"fileCount":1},
    {"importPath":"example.com/app","archiveBytes":4,"sourceBytes":13,"fileCount":1}
  ]
}`), 0644); parseErr5 != nil {
		parseT.Fatalf("write baseline attribution: %v", parseErr5)
	}
	parseBaselineManifestPath := filepath.Join(parseBaselineDir, "wasm-release-manifest.json")
	if parseErr6 := os.WriteFile(parseBaselineManifestPath, []byte(`{
  "package": "example.com/app",
  "profile": "release",
  "goos": "js",
  "goarch": "wasm",
  "artifacts": {
    "wasm": {"path":"app.wasm","bytes":4,"sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
  },
  "attribution": {
    "mode": "packages",
    "path": "wasm-package-size-attribution.json",
    "packageCount": 2
  }
}`), 0644); parseErr6 != nil {
		parseT.Fatalf("write baseline manifest: %v", parseErr6)
	}

	releaseExecuteBuild = func(parseConfig buildConfig) (buildSummary, error) {
		if parseErr7 := os.MkdirAll(filepath.Dir(parseConfig.outputPath), 0755); parseErr7 != nil {
			return buildSummary{}, parseErr7
		}
		if parseErr8 := os.WriteFile(parseConfig.outputPath, []byte("raw-wasm-that-is-larger"), 0644); parseErr8 != nil {
			return buildSummary{}, parseErr8
		}
		return buildSummary{
			OK:          true,
			Profile:     buildProfile{Name: "release"},
			AppPath:     parseConfig.appPath,
			ProjectRoot: parseConfig.rootPath,
			PackageDir:  parseAppSourceDir,
			OutputPath:  parseConfig.outputPath,
		}, nil
	}
	releaseRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		if parseCommand != "go" {
			parseT.Fatalf("expected go command, got %q", parseCommand)
		}
		return `{"ImportPath":"example.com/lib","Dir":"` + filepath.ToSlash(parseLibSourceDir) + `","Export":"` + filepath.ToSlash(parseLibExport) + `","GoFiles":["lib.go"]}
{"ImportPath":"example.com/app","Dir":"` + filepath.ToSlash(parseAppSourceDir) + `","GoFiles":["main.go"]}
`, nil
	}

	parseSummary, parseErr9 := executeRelease(releaseConfig{
		appPath:         parseAppSource,
		rootPath:        parseRoot,
		outDir:          parseOutDir,
		binaryName:      "app.wasm",
		manifestName:    "manifest.json",
		compareManifest: parseBaselineManifestPath,
		profile:         "release",
		compression:     "none",
		sizeAttribution: "packages",
	})
	if parseErr9 != nil {
		parseT.Fatalf("execute release with diff report: %v", parseErr9)
	}
	if parseSummary.Diff == nil || parseSummary.Diff.Path != "wasm-release-size-diff.json" {
		parseT.Fatalf("expected diff report summary, got %#v", parseSummary)
	}
	if _, parseOk := parseSummary.Artifacts["diff_report"]; !parseOk {
		parseT.Fatalf("expected diff report artifact, got %#v", parseSummary.Artifacts)
	}
	if len(parseSummary.Diff.ArtifactChanges) == 0 {
		parseT.Fatalf("expected artifact changes in diff summary, got %#v", parseSummary.Diff)
	}
	if len(parseSummary.Diff.LikelyCulprits) == 0 || parseSummary.Diff.LikelyCulprits[0].ImportPath != "example.com/lib" {
		parseT.Fatalf("expected lib package as likely culprit, got %#v", parseSummary.Diff)
	}
	parseDiffBytes, parseErr9 := os.ReadFile(filepath.Join(parseOutDir, "wasm-release-size-diff.json"))
	if parseErr9 != nil {
		parseT.Fatalf("read diff report: %v", parseErr9)
	}
	var parseDiffPayload struct {
		ArtifactChanges []releaseArtifactDiffRecord `json:"artifactChanges"`
		LikelyCulprits  []releasePackageDiffRecord  `json:"likelyCulprits"`
	}
	if parseErr10 := json.Unmarshal(parseDiffBytes, &parseDiffPayload); parseErr10 != nil {
		parseT.Fatalf("unmarshal diff report: %v", parseErr10)
	}
	if len(parseDiffPayload.ArtifactChanges) == 0 || len(parseDiffPayload.LikelyCulprits) == 0 {
		parseT.Fatalf("expected populated diff report, got %#v", parseDiffPayload)
	}
}

func TestExecuteReleaseIncludesStartupReport(parseT *testing.T) {
	parseOriginalReleaseExecuteBuild := releaseExecuteBuild
	parseOriginalReleaseMeasureStartup := releaseMeasureStartup
	parseT.Cleanup(func() {
		releaseExecuteBuild = parseOriginalReleaseExecuteBuild
		releaseMeasureStartup = parseOriginalReleaseMeasureStartup
	})

	parseRoot := parseT.TempDir()
	parseOutDir := filepath.Join(parseRoot, "dist")
	releaseExecuteBuild = func(parseConfig buildConfig) (buildSummary, error) {
		if parseErr := os.MkdirAll(filepath.Dir(parseConfig.outputPath), 0755); parseErr != nil {
			return buildSummary{}, parseErr
		}
		if parseErr2 := os.WriteFile(parseConfig.outputPath, []byte("raw-wasm"), 0644); parseErr2 != nil {
			return buildSummary{}, parseErr2
		}
		return buildSummary{
			OK:          true,
			Profile:     buildProfile{Name: "release"},
			AppPath:     parseConfig.appPath,
			ProjectRoot: parseConfig.rootPath,
			PackageDir:  parseRoot,
			OutputPath:  parseConfig.outputPath,
		}, nil
	}
	releaseMeasureStartup = func(parseConfig2 releaseConfig, parseArtifacts map[string]releaseArtifactRecord) (*releaseStartupRecord, error) {
		parseReportPath := filepath.Join(parseConfig2.outDir, "wasm-startup-report.json")
		if parseErr3 := os.WriteFile(parseReportPath, []byte(`{"startup":{"readyMs":12}}`), 0644); parseErr3 != nil {
			return nil, parseErr3
		}
		return &releaseStartupRecord{
			Mode:              "browser",
			Path:              "wasm-startup-report.json",
			ProbeURL:          "http://127.0.0.1:9999/__gwc/startup-probe.html",
			TransportEncoding: "gzip",
		}, nil
	}

	parseSummary, parseErr4 := executeRelease(releaseConfig{
		appPath:          filepath.Join(parseRoot, "main.go"),
		rootPath:         parseRoot,
		outDir:           parseOutDir,
		binaryName:       "app.wasm",
		manifestName:     "manifest.json",
		profile:          "release",
		compression:      "none",
		startupMeasure:   "browser",
		startupTimeoutMs: 15000,
	})
	if parseErr4 != nil {
		parseT.Fatalf("execute release with startup report: %v", parseErr4)
	}
	if parseSummary.Startup == nil || parseSummary.Startup.Path != "wasm-startup-report.json" {
		parseT.Fatalf("expected startup report summary, got %#v", parseSummary)
	}
	if _, parseOk := parseSummary.Artifacts["startup_report"]; !parseOk {
		parseT.Fatalf("expected startup report artifact, got %#v", parseSummary.Artifacts)
	}
}

func TestExecuteReleaseIncludesValidationReport(parseT *testing.T) {
	parseOriginalReleaseExecuteBuild := releaseExecuteBuild
	parseOriginalReleaseMeasureStartup := releaseMeasureStartup
	parseOriginalReleaseValidateSmoke := releaseValidateSmoke
	parseT.Cleanup(func() {
		releaseExecuteBuild = parseOriginalReleaseExecuteBuild
		releaseMeasureStartup = parseOriginalReleaseMeasureStartup
		releaseValidateSmoke = parseOriginalReleaseValidateSmoke
	})

	parseRoot := parseT.TempDir()
	parseOutDir := filepath.Join(parseRoot, "dist")
	releaseExecuteBuild = func(parseConfig buildConfig) (buildSummary, error) {
		if parseErr := os.MkdirAll(filepath.Dir(parseConfig.outputPath), 0755); parseErr != nil {
			return buildSummary{}, parseErr
		}
		if parseErr2 := os.WriteFile(parseConfig.outputPath, []byte("raw-wasm"), 0644); parseErr2 != nil {
			return buildSummary{}, parseErr2
		}
		return buildSummary{
			OK:          true,
			Profile:     buildProfile{Name: "release"},
			AppPath:     parseConfig.appPath,
			ProjectRoot: parseConfig.rootPath,
			PackageDir:  parseRoot,
			OutputPath:  parseConfig.outputPath,
		}, nil
	}
	releaseMeasureStartup = func(parseConfig2 releaseConfig, parseArtifacts map[string]releaseArtifactRecord) (*releaseStartupRecord, error) {
		parseReportPath := filepath.Join(parseConfig2.outDir, "wasm-startup-report.json")
		if parseErr3 := os.WriteFile(parseReportPath, []byte(`{"startup":{"readyMs":12}}`), 0644); parseErr3 != nil {
			return nil, parseErr3
		}
		return &releaseStartupRecord{
			Mode:              "browser",
			Path:              "wasm-startup-report.json",
			ProbeURL:          "http://127.0.0.1:9999/__gwc/startup-probe.html",
			TransportEncoding: "identity",
		}, nil
	}
	releaseValidateSmoke = func(parseConfig3 releaseConfig, parseManifestPath string, parseArtifacts2 map[string]releaseArtifactRecord, parseStartupReport *releaseStartupRecord) (*releaseValidationRecord, error) {
		parseReportPath2 := filepath.Join(parseConfig3.outDir, "wasm-release-validation.json")
		if parseErr4 := os.WriteFile(parseReportPath2, []byte(`{"checks":["ok"]}`), 0644); parseErr4 != nil {
			return nil, parseErr4
		}
		return &releaseValidationRecord{
			Path:                "wasm-release-validation.json",
			Checks:              []string{"ok"},
			StartupReportPath:   parseStartupReport.Path,
			WasmContentType:     "application/wasm",
			WasmContentEncoding: "identity",
		}, nil
	}

	parseSummary, parseErr5 := executeRelease(releaseConfig{
		appPath:          filepath.Join(parseRoot, "main.go"),
		rootPath:         parseRoot,
		outDir:           parseOutDir,
		binaryName:       "app.wasm",
		manifestName:     "manifest.json",
		profile:          "release",
		compression:      "none",
		validateSmoke:    true,
		startupTimeoutMs: 15000,
	})
	if parseErr5 != nil {
		parseT.Fatalf("execute release with validation report: %v", parseErr5)
	}
	if parseSummary.Validation == nil || parseSummary.Validation.Path != "wasm-release-validation.json" {
		parseT.Fatalf("expected validation report summary, got %#v", parseSummary)
	}
	if _, parseOk := parseSummary.Artifacts["validation_report"]; !parseOk {
		parseT.Fatalf("expected validation report artifact, got %#v", parseSummary.Artifacts)
	}
}

func TestExecuteReleaseErrorPaths(parseT *testing.T) {
	parseOriginalReleaseExecuteBuild := releaseExecuteBuild
	parseOriginalArtifactRecord := releaseArtifactRecordForPathFunc
	parseOriginalWriteGzip := releaseWriteGzipSidecar
	parseOriginalWriteBrotli := releaseWriteBrotliSidecar
	parseOriginalMarshalIndent := releaseMarshalIndent
	parseOriginalReleaseRunCommand := releaseRunCommand
	parseOriginalReleaseLookPath := releaseLookPath
	parseOriginalReleaseMeasureStartup := releaseMeasureStartup
	parseOriginalReleaseValidateSmoke := releaseValidateSmoke
	parseT.Cleanup(func() {
		releaseExecuteBuild = parseOriginalReleaseExecuteBuild
		releaseArtifactRecordForPathFunc = parseOriginalArtifactRecord
		releaseWriteGzipSidecar = parseOriginalWriteGzip
		releaseWriteBrotliSidecar = parseOriginalWriteBrotli
		releaseMarshalIndent = parseOriginalMarshalIndent
		releaseRunCommand = parseOriginalReleaseRunCommand
		releaseLookPath = parseOriginalReleaseLookPath
		releaseMeasureStartup = parseOriginalReleaseMeasureStartup
		releaseValidateSmoke = parseOriginalReleaseValidateSmoke
	})

	parseT.Run("outdir is file", func(parseT2 *testing.T) {
		parseRoot := parseT2.TempDir()
		parseOutPath := filepath.Join(parseRoot, "release-file")
		if parseErr := os.WriteFile(parseOutPath, []byte("occupied"), 0644); parseErr != nil {
			parseT2.Fatalf("write occupied out path: %v", parseErr)
		}
		_, parseErr2 := executeRelease(releaseConfig{outDir: parseOutPath})
		if parseErr2 == nil || !strings.Contains(parseErr2.Error(), "create release output directory") {
			parseT2.Fatalf("expected out dir creation error, got %v", parseErr2)
		}
	})

	parseT.Run("invalid budgets file", func(parseT3 *testing.T) {
		parseRoot2 := parseT3.TempDir()
		parseMainPath := filepath.Join(parseRoot2, "main.go")
		if parseErr3 := os.WriteFile(filepath.Join(parseRoot2, "go.mod"), []byte("module example.com/gwcreleaseinvalidbudget\n\ngo 1.25.0\n"), 0644); parseErr3 != nil {
			parseT3.Fatalf("write go.mod: %v", parseErr3)
		}
		if parseErr4 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr4 != nil {
			parseT3.Fatalf("write main.go: %v", parseErr4)
		}
		parseBudgetsPath := filepath.Join(parseRoot2, "budgets.json")
		if parseErr5 := os.WriteFile(parseBudgetsPath, []byte(`{"raw_bytes":`), 0644); parseErr5 != nil {
			parseT3.Fatalf("write budgets: %v", parseErr5)
		}
		_, parseErr6 := executeRelease(releaseConfig{appPath: parseMainPath, rootPath: parseRoot2, outDir: filepath.Join(parseRoot2, "dist"), binaryName: "app.wasm", manifestName: "manifest.json", profile: "release", compression: "none", skipCompression: true, budgetsPath: parseBudgetsPath})
		if parseErr6 == nil || !strings.Contains(parseErr6.Error(), "parse budgets file") {
			parseT3.Fatalf("expected invalid budgets error, got %v", parseErr6)
		}
	})

	parseT.Run("artifact budgets exceeded", func(parseT4 *testing.T) {
		parseRoot3 := parseT4.TempDir()
		parseMainPath2 := filepath.Join(parseRoot3, "main.go")
		if parseErr7 := os.WriteFile(filepath.Join(parseRoot3, "go.mod"), []byte("module example.com/gwcreleasebudgetfail\n\ngo 1.25.0\n"), 0644); parseErr7 != nil {
			parseT4.Fatalf("write go.mod: %v", parseErr7)
		}
		if parseErr8 := os.WriteFile(parseMainPath2, []byte("package main\nfunc main() {}\n"), 0644); parseErr8 != nil {
			parseT4.Fatalf("write main.go: %v", parseErr8)
		}
		parseBudgetsPath2 := filepath.Join(parseRoot3, "budgets.json")
		if parseErr9 := os.WriteFile(parseBudgetsPath2, []byte(`{"raw_bytes":1}`), 0644); parseErr9 != nil {
			parseT4.Fatalf("write budgets: %v", parseErr9)
		}
		_, parseErr10 := executeRelease(releaseConfig{appPath: parseMainPath2, rootPath: parseRoot3, outDir: filepath.Join(parseRoot3, "dist"), binaryName: "app.wasm", manifestName: "manifest.json", profile: "release", compression: "none", skipCompression: true, budgetsPath: parseBudgetsPath2})
		if parseErr10 == nil || !strings.Contains(parseErr10.Error(), "artifact budget exceeded for raw wasm") {
			parseT4.Fatalf("expected artifact budget failure, got %v", parseErr10)
		}
	})

	parseT.Run("manifest write failure", func(parseT5 *testing.T) {
		parseRoot4 := parseT5.TempDir()
		parseMainPath3 := filepath.Join(parseRoot4, "main.go")
		if parseErr11 := os.WriteFile(filepath.Join(parseRoot4, "go.mod"), []byte("module example.com/gwcreleasemanifestfail\n\ngo 1.25.0\n"), 0644); parseErr11 != nil {
			parseT5.Fatalf("write go.mod: %v", parseErr11)
		}
		if parseErr12 := os.WriteFile(parseMainPath3, []byte("package main\nfunc main() {}\n"), 0644); parseErr12 != nil {
			parseT5.Fatalf("write main.go: %v", parseErr12)
		}
		_, parseErr13 := executeRelease(releaseConfig{appPath: parseMainPath3, rootPath: parseRoot4, outDir: filepath.Join(parseRoot4, "dist"), binaryName: "app.wasm", manifestName: ".", profile: "release", compression: "none", skipCompression: true})
		if parseErr13 == nil || !strings.Contains(parseErr13.Error(), "write release manifest") {
			parseT5.Fatalf("expected manifest write failure, got %v", parseErr13)
		}
	})

	parseT.Run("build failure", func(parseT6 *testing.T) {
		parseRoot5 := parseT6.TempDir()
		parseMainPath4 := filepath.Join(parseRoot5, "main.go")
		if parseErr14 := os.WriteFile(parseMainPath4, []byte("package main\nfunc main() {}\n"), 0644); parseErr14 != nil {
			parseT6.Fatalf("write main.go: %v", parseErr14)
		}
		parseBinDir := filepath.Join(parseT6.TempDir(), "bin")
		if parseErr15 := os.MkdirAll(parseBinDir, 0755); parseErr15 != nil {
			parseT6.Fatalf("mkdir fake bin: %v", parseErr15)
		}
		writeFakeGoBuildCommand(parseT6, parseBinDir)
		parseT6.Setenv("PATH", parseBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		parseT6.Setenv("FAKE_GO_MODE", "fail-empty")

		_, parseErr16 := executeRelease(releaseConfig{appPath: parseMainPath4, rootPath: parseRoot5, outDir: filepath.Join(parseRoot5, "dist"), binaryName: "app.wasm", manifestName: "manifest.json", compression: "none", skipCompression: true, profile: "release"})
		if parseErr16 == nil || !strings.Contains(parseErr16.Error(), "go build failed") {
			parseT6.Fatalf("expected build failure, got %v", parseErr16)
		}
	})

	parseT.Run("propagates built artifact read failure from executeBuild", func(parseT7 *testing.T) {
		parseRoot6 := parseT7.TempDir()
		parseMainPath5 := filepath.Join(parseRoot6, "main.go")
		if parseErr17 := os.WriteFile(parseMainPath5, []byte("package main\nfunc main() {}\n"), 0644); parseErr17 != nil {
			parseT7.Fatalf("write main.go: %v", parseErr17)
		}
		parseBinDir2 := filepath.Join(parseT7.TempDir(), "bin")
		if parseErr18 := os.MkdirAll(parseBinDir2, 0755); parseErr18 != nil {
			parseT7.Fatalf("mkdir fake bin: %v", parseErr18)
		}
		writeFakeGoBuildCommand(parseT7, parseBinDir2)
		parseT7.Setenv("PATH", parseBinDir2+string(os.PathListSeparator)+os.Getenv("PATH"))
		parseT7.Setenv("FAKE_GO_MODE", "write-dir")

		_, parseErr19 := executeRelease(releaseConfig{appPath: parseMainPath5, rootPath: parseRoot6, outDir: filepath.Join(parseRoot6, "dist"), binaryName: "app.wasm", manifestName: "manifest.json", compression: "none", skipCompression: true, profile: "release"})
		if parseErr19 == nil || !strings.Contains(parseErr19.Error(), "read built wasm artifact") {
			parseT7.Fatalf("expected executeBuild artifact read failure, got %v", parseErr19)
		}
	})

	parseT.Run("raw artifact helper failure", func(parseT8 *testing.T) {
		parseRoot7 := parseT8.TempDir()
		releaseExecuteBuild = func(parseConfig buildConfig) (buildSummary, error) {
			return buildSummary{OK: true, Profile: buildProfile{Name: "release"}, AppPath: parseConfig.appPath, ProjectRoot: parseConfig.rootPath, PackageDir: parseRoot7}, nil
		}
		releaseArtifactRecordForPathFunc = func(parseBaseDir string, parseArtifactPath3 string) (releaseArtifactRecord, error) {
			return releaseArtifactRecord{}, errors.New("artifact failed")
		}
		defer func() {
			releaseExecuteBuild = parseOriginalReleaseExecuteBuild
			releaseArtifactRecordForPathFunc = parseOriginalArtifactRecord
		}()

		_, parseErr20 := executeRelease(releaseConfig{appPath: filepath.Join(parseRoot7, "main.go"), rootPath: parseRoot7, outDir: filepath.Join(parseRoot7, "dist"), binaryName: "app.wasm", manifestName: "manifest.json", compression: "none", skipCompression: true, profile: "release"})
		if parseErr20 == nil || !strings.Contains(parseErr20.Error(), "artifact failed") {
			parseT8.Fatalf("expected raw artifact helper failure, got %v", parseErr20)
		}
	})

	parseT.Run("gzip sidecar helper failure", func(parseT9 *testing.T) {
		parseRoot8 := parseT9.TempDir()
		releaseExecuteBuild = func(parseConfig2 buildConfig) (buildSummary, error) {
			parseArtifactPath := filepath.Join(parseConfig2.rootPath, "dist", "app.wasm")
			if parseErr21 := os.MkdirAll(filepath.Dir(parseArtifactPath), 0755); parseErr21 != nil {
				return buildSummary{}, parseErr21
			}
			if parseErr22 := os.WriteFile(parseArtifactPath, []byte("wasm"), 0644); parseErr22 != nil {
				return buildSummary{}, parseErr22
			}
			return buildSummary{OK: true, Profile: buildProfile{Name: "release"}, AppPath: parseConfig2.appPath, ProjectRoot: parseConfig2.rootPath, PackageDir: parseRoot8}, nil
		}
		releaseWriteGzipSidecar = func(parseSourcePath string, parseTargetPath string) error { return errors.New("gzip failed") }
		defer func() {
			releaseExecuteBuild = parseOriginalReleaseExecuteBuild
			releaseWriteGzipSidecar = parseOriginalWriteGzip
		}()

		_, parseErr23 := executeRelease(releaseConfig{appPath: filepath.Join(parseRoot8, "main.go"), rootPath: parseRoot8, outDir: filepath.Join(parseRoot8, "dist"), binaryName: "app.wasm", manifestName: "manifest.json", compression: "gzip", profile: "release"})
		if parseErr23 == nil || !strings.Contains(parseErr23.Error(), "gzip failed") {
			parseT9.Fatalf("expected gzip sidecar helper failure, got %v", parseErr23)
		}
	})

	parseT.Run("brotli artifact helper and manifest encoding failures", func(parseT10 *testing.T) {
		parseRoot9 := parseT10.TempDir()
		releaseExecuteBuild = func(parseConfig3 buildConfig) (buildSummary, error) {
			parseArtifactPath2 := filepath.Join(parseConfig3.rootPath, "dist", "app.wasm")
			if parseErr24 := os.MkdirAll(filepath.Dir(parseArtifactPath2), 0755); parseErr24 != nil {
				return buildSummary{}, parseErr24
			}
			if parseErr25 := os.WriteFile(parseArtifactPath2, []byte("wasm"), 0644); parseErr25 != nil {
				return buildSummary{}, parseErr25
			}
			return buildSummary{OK: true, Profile: buildProfile{Name: "release"}, AppPath: parseConfig3.appPath, ProjectRoot: parseConfig3.rootPath, PackageDir: parseRoot9}, nil
		}
		parseCallCount := 0
		releaseArtifactRecordForPathFunc = func(parseBaseDir2 string, parseArtifactPath4 string) (releaseArtifactRecord, error) {
			parseCallCount++
			if parseCallCount == 2 {
				return releaseArtifactRecord{}, errors.New("brotli artifact failed")
			}
			return releaseArtifactRecord{Path: filepath.Base(parseArtifactPath4), Bytes: 4, SHA256: strings.Repeat("a", 64)}, nil
		}
		defer func() {
			releaseExecuteBuild = parseOriginalReleaseExecuteBuild
			releaseArtifactRecordForPathFunc = parseOriginalArtifactRecord
			releaseMarshalIndent = parseOriginalMarshalIndent
		}()

		_, parseErr26 := executeRelease(releaseConfig{appPath: filepath.Join(parseRoot9, "main.go"), rootPath: parseRoot9, outDir: filepath.Join(parseRoot9, "dist"), binaryName: "app.wasm", manifestName: "manifest.json", compression: "brotli", profile: "release"})
		if parseErr26 == nil || !strings.Contains(parseErr26.Error(), "brotli artifact failed") {
			parseT10.Fatalf("expected brotli artifact helper failure, got %v", parseErr26)
		}

		releaseArtifactRecordForPathFunc = func(parseBaseDir3 string, parseArtifactPath5 string) (releaseArtifactRecord, error) {
			return releaseArtifactRecord{Path: filepath.Base(parseArtifactPath5), Bytes: 4, SHA256: strings.Repeat("a", 64)}, nil
		}
		releaseMarshalIndent = func(parseV interface{}, parsePrefix string, parseIndent string) ([]byte, error) {
			return nil, errors.New("marshal failed")
		}
		_, parseErr26 = executeRelease(releaseConfig{appPath: filepath.Join(parseRoot9, "main.go"), rootPath: parseRoot9, outDir: filepath.Join(parseRoot9, "dist-2"), binaryName: "app.wasm", manifestName: "manifest.json", compression: "none", skipCompression: true, profile: "release"})
		if parseErr26 == nil || !strings.Contains(parseErr26.Error(), "encode release manifest") {
			parseT10.Fatalf("expected manifest encoding failure, got %v", parseErr26)
		}
	})

	parseT.Run("post-link optimizer unavailable or fails", func(parseT11 *testing.T) {
		parseRoot10 := parseT11.TempDir()
		releaseExecuteBuild = func(parseConfig4 buildConfig) (buildSummary, error) {
			if parseErr27 := os.MkdirAll(filepath.Dir(parseConfig4.outputPath), 0755); parseErr27 != nil {
				return buildSummary{}, parseErr27
			}
			if parseErr28 := os.WriteFile(parseConfig4.outputPath, []byte("wasm"), 0644); parseErr28 != nil {
				return buildSummary{}, parseErr28
			}
			return buildSummary{OK: true, Profile: buildProfile{Name: "release"}, AppPath: parseConfig4.appPath, ProjectRoot: parseConfig4.rootPath, PackageDir: parseRoot10}, nil
		}
		releaseLookPath = func(parseFile string) (string, error) { return "", errors.New("not found") }
		_, parseErr29 := executeRelease(releaseConfig{
			appPath:      filepath.Join(parseRoot10, "main.go"),
			rootPath:     parseRoot10,
			outDir:       filepath.Join(parseRoot10, "dist"),
			binaryName:   "app.wasm",
			manifestName: "manifest.json",
			compression:  "none",
			postLinkOpt:  "wasm-opt",
			profile:      "release",
		})
		if parseErr29 == nil || !strings.Contains(parseErr29.Error(), "wasm-opt is unavailable") {
			parseT11.Fatalf("expected unavailable optimizer error, got %v", parseErr29)
		}

		releaseLookPath = func(parseFile2 string) (string, error) { return "wasm-opt", nil }
		releaseRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
			return "", errors.New("optimizer failed")
		}
		_, parseErr29 = executeRelease(releaseConfig{
			appPath:      filepath.Join(parseRoot10, "main.go"),
			rootPath:     parseRoot10,
			outDir:       filepath.Join(parseRoot10, "dist-2"),
			binaryName:   "app.wasm",
			manifestName: "manifest.json",
			compression:  "none",
			postLinkOpt:  "wasm-opt",
			profile:      "release",
		})
		if parseErr29 == nil || !strings.Contains(parseErr29.Error(), "run post-link optimizer") {
			parseT11.Fatalf("expected optimizer execution failure, got %v", parseErr29)
		}
	})

	parseT.Run("size attribution go list failure", func(parseT12 *testing.T) {
		parseRoot11 := parseT12.TempDir()
		releaseExecuteBuild = func(parseConfig5 buildConfig) (buildSummary, error) {
			if parseErr30 := os.MkdirAll(filepath.Dir(parseConfig5.outputPath), 0755); parseErr30 != nil {
				return buildSummary{}, parseErr30
			}
			if parseErr31 := os.WriteFile(parseConfig5.outputPath, []byte("wasm"), 0644); parseErr31 != nil {
				return buildSummary{}, parseErr31
			}
			return buildSummary{OK: true, Profile: buildProfile{Name: "release"}, AppPath: parseConfig5.appPath, ProjectRoot: parseConfig5.rootPath, PackageDir: parseRoot11}, nil
		}
		releaseRunCommand = func(parseCommand2 string, parseArgs2 []string, parseCwd2 string, parseEnv2 []string) (string, error) {
			return "", errors.New("go list failed")
		}
		_, parseErr32 := executeRelease(releaseConfig{
			appPath:         filepath.Join(parseRoot11, "main.go"),
			rootPath:        parseRoot11,
			outDir:          filepath.Join(parseRoot11, "dist"),
			binaryName:      "app.wasm",
			manifestName:    "manifest.json",
			compression:     "none",
			sizeAttribution: "packages",
			profile:         "release",
		})
		if parseErr32 == nil || !strings.Contains(parseErr32.Error(), "collect release package attribution") {
			parseT12.Fatalf("expected size attribution collection failure, got %v", parseErr32)
		}
	})

	parseT.Run("compare manifest read failure", func(parseT13 *testing.T) {
		parseRoot12 := parseT13.TempDir()
		releaseExecuteBuild = func(parseConfig6 buildConfig) (buildSummary, error) {
			if parseErr33 := os.MkdirAll(filepath.Dir(parseConfig6.outputPath), 0755); parseErr33 != nil {
				return buildSummary{}, parseErr33
			}
			if parseErr34 := os.WriteFile(parseConfig6.outputPath, []byte("wasm"), 0644); parseErr34 != nil {
				return buildSummary{}, parseErr34
			}
			return buildSummary{OK: true, Profile: buildProfile{Name: "release"}, AppPath: parseConfig6.appPath, ProjectRoot: parseConfig6.rootPath, PackageDir: parseRoot12}, nil
		}
		releaseRunCommand = func(parseCommand3 string, parseArgs3 []string, parseCwd3 string, parseEnv3 []string) (string, error) {
			return `{"ImportPath":"example.com/app","Dir":"` + filepath.ToSlash(parseRoot12) + `","GoFiles":[]}` + "\n", nil
		}
		_, parseErr35 := executeRelease(releaseConfig{
			appPath:         filepath.Join(parseRoot12, "main.go"),
			rootPath:        parseRoot12,
			outDir:          filepath.Join(parseRoot12, "dist"),
			binaryName:      "app.wasm",
			manifestName:    "manifest.json",
			compression:     "none",
			sizeAttribution: "none",
			compareManifest: filepath.Join(parseRoot12, "missing-manifest.json"),
			profile:         "release",
		})
		if parseErr35 == nil || !strings.Contains(parseErr35.Error(), "read compare manifest") {
			parseT13.Fatalf("expected compare manifest read failure, got %v", parseErr35)
		}
	})

	parseT.Run("startup measurement failure", func(parseT14 *testing.T) {
		parseRoot13 := parseT14.TempDir()
		releaseExecuteBuild = func(parseConfig7 buildConfig) (buildSummary, error) {
			if parseErr36 := os.MkdirAll(filepath.Dir(parseConfig7.outputPath), 0755); parseErr36 != nil {
				return buildSummary{}, parseErr36
			}
			if parseErr37 := os.WriteFile(parseConfig7.outputPath, []byte("wasm"), 0644); parseErr37 != nil {
				return buildSummary{}, parseErr37
			}
			return buildSummary{OK: true, Profile: buildProfile{Name: "release"}, AppPath: parseConfig7.appPath, ProjectRoot: parseConfig7.rootPath, PackageDir: parseRoot13}, nil
		}
		releaseMeasureStartup = func(parseConfig8 releaseConfig, parseArtifacts map[string]releaseArtifactRecord) (*releaseStartupRecord, error) {
			return nil, errors.New("startup probe failed")
		}
		_, parseErr38 := executeRelease(releaseConfig{
			appPath:        filepath.Join(parseRoot13, "main.go"),
			rootPath:       parseRoot13,
			outDir:         filepath.Join(parseRoot13, "dist"),
			binaryName:     "app.wasm",
			manifestName:   "manifest.json",
			compression:    "none",
			startupMeasure: "browser",
			profile:        "release",
		})
		if parseErr38 == nil || !strings.Contains(parseErr38.Error(), "startup probe failed") {
			parseT14.Fatalf("expected startup measurement failure, got %v", parseErr38)
		}
	})

	parseT.Run("smoke validation failure", func(parseT15 *testing.T) {
		parseRoot14 := parseT15.TempDir()
		releaseExecuteBuild = func(parseConfig9 buildConfig) (buildSummary, error) {
			if parseErr39 := os.MkdirAll(filepath.Dir(parseConfig9.outputPath), 0755); parseErr39 != nil {
				return buildSummary{}, parseErr39
			}
			if parseErr40 := os.WriteFile(parseConfig9.outputPath, []byte("wasm"), 0644); parseErr40 != nil {
				return buildSummary{}, parseErr40
			}
			return buildSummary{OK: true, Profile: buildProfile{Name: "release"}, AppPath: parseConfig9.appPath, ProjectRoot: parseConfig9.rootPath, PackageDir: parseRoot14}, nil
		}
		releaseMeasureStartup = func(parseConfig10 releaseConfig, parseArtifacts2 map[string]releaseArtifactRecord) (*releaseStartupRecord, error) {
			parseReportPath := filepath.Join(parseConfig10.outDir, "wasm-startup-report.json")
			if parseErr41 := os.WriteFile(parseReportPath, []byte(`{"startup":{"readyMs":12}}`), 0644); parseErr41 != nil {
				return nil, parseErr41
			}
			return &releaseStartupRecord{Mode: "browser", Path: "wasm-startup-report.json", ProbeURL: "http://127.0.0.1:9999/__gwc/startup-probe.html"}, nil
		}
		releaseValidateSmoke = func(parseConfig11 releaseConfig, parseManifestPath string, parseArtifacts3 map[string]releaseArtifactRecord, parseStartupReport *releaseStartupRecord) (*releaseValidationRecord, error) {
			return nil, errors.New("smoke validation failed")
		}
		_, parseErr42 := executeRelease(releaseConfig{
			appPath:       filepath.Join(parseRoot14, "main.go"),
			rootPath:      parseRoot14,
			outDir:        filepath.Join(parseRoot14, "dist"),
			binaryName:    "app.wasm",
			manifestName:  "manifest.json",
			compression:   "none",
			validateSmoke: true,
			profile:       "release",
		})
		if parseErr42 == nil || !strings.Contains(parseErr42.Error(), "smoke validation failed") {
			parseT15.Fatalf("expected smoke validation failure, got %v", parseErr42)
		}
	})
}
