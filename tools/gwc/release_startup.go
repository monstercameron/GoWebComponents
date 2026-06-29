package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/pwa"
	playwright "github.com/playwright-community/playwright-go"
)

func measureReleaseStartup(parseConfig releaseConfig, parseArtifacts map[string]releaseArtifactRecord) (*releaseStartupRecord, error) {
	parseMode, parseErr := normalizeReleaseStartupMeasureMode(parseConfig.startupMeasure)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseMode == "none" {
		return nil, nil
	}
	parseWasmExecPath, parseErr := releaseResolveWasmExec()
	if parseErr != nil {
		return nil, fmt.Errorf("resolve wasm_exec.js for startup measurement: %w", parseErr)
	}
	parseReleaseWasmPath := filepath.Join(parseConfig.outDir, parseConfig.binaryName)
	parseProbeURL, parseTransportEncoding, parseShutdown, parseErr := startReleaseStartupProbeServer(parseConfig.outDir, parseConfig.binaryName, parseWasmExecPath, parseArtifacts, parseConfig.startupTimeoutMs)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseShutdown()

	parseReportPath := filepath.Join(parseConfig.outDir, "wasm-startup-report.json")
	if parseErr2 := releaseRunStartupProbeWithPlaywright(parseProbeURL, parseReportPath, parseConfig.startupTimeoutMs); parseErr2 != nil {
		return nil, fmt.Errorf("measure release startup: %w", parseErr2)
	}
	if !fileExists(parseReportPath) {
		return nil, fmt.Errorf("startup measurement did not produce a report for %s", parseReleaseWasmPath)
	}
	return &releaseStartupRecord{
		Mode:              parseMode,
		Path:              "wasm-startup-report.json",
		ProbeURL:          parseProbeURL,
		TransportEncoding: parseTransportEncoding,
	}, nil
}

func runReleaseStartupProbeWithPlaywright(parseProbeURL string, parseReportPath string, parseTimeoutMs int) error {
	parseRunOptions := &playwright.RunOptions{
		Browsers: []string{"chromium"},
		Verbose:  false,
	}
	if parseErr := releasePlaywrightInstall(parseRunOptions); parseErr != nil {
		return fmt.Errorf("install playwright-go runtime: %w", parseErr)
	}
	parsePw, parseErr2 := releasePlaywrightRun(parseRunOptions)
	if parseErr2 != nil {
		return fmt.Errorf("run playwright-go runtime: %w", parseErr2)
	}
	defer func() {
		_ = parsePw.Stop()
	}()
	parseBrowser, parseErr2 := parsePw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: new(true),
	})
	if parseErr2 != nil {
		return fmt.Errorf("launch chromium: %w", parseErr2)
	}
	defer func() {
		_ = parseBrowser.Close()
	}()
	parsePage, parseErr2 := parseBrowser.NewPage()
	if parseErr2 != nil {
		return fmt.Errorf("create probe page: %w", parseErr2)
	}
	if parseTimeoutMs > 0 {
		parsePage.SetDefaultTimeout(float64(parseTimeoutMs))
	}
	parseResponse, parseErr2 := parsePage.Goto(parseProbeURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if parseErr2 != nil {
		return fmt.Errorf("open startup probe page: %w", parseErr2)
	}
	if parseResponse == nil {
		return errors.New("startup probe navigation returned no response")
	}
	if parseResponse.Status() >= 400 {
		return fmt.Errorf("startup probe navigation failed: %d", parseResponse.Status())
	}
	if _, parseErr3 := parsePage.WaitForFunction("() => window.__gwcStartupProbe && window.__gwcStartupProbe.readyMs !== null", nil); parseErr3 != nil {
		return fmt.Errorf("wait for ready probe: %w", parseErr3)
	}
	if parseErr4 := parsePage.Locator("#__gwc_probe_button").Click(); parseErr4 != nil {
		return fmt.Errorf("trigger startup probe interaction: %w", parseErr4)
	}
	if _, parseErr5 := parsePage.WaitForFunction("() => window.__gwcStartupProbe && window.__gwcStartupProbe.interactionMs !== null", nil); parseErr5 != nil {
		return fmt.Errorf("wait for interaction probe: %w", parseErr5)
	}
	parseReport, parseErr2 := parsePage.Evaluate(`() => ({
		userAgent: navigator.userAgent,
		startup: window.__gwcStartupProbe || null,
		timestamp: new Date().toISOString()
	})`)
	if parseErr2 != nil {
		return fmt.Errorf("collect startup probe report: %w", parseErr2)
	}
	parseEncoded, parseErr2 := json.MarshalIndent(parseReport, "", "  ")
	if parseErr2 != nil {
		return fmt.Errorf("encode startup probe report: %w", parseErr2)
	}
	parseEncoded = append(parseEncoded, '\n')
	if parseErr6 := os.WriteFile(parseReportPath, parseEncoded, 0644); parseErr6 != nil {
		return fmt.Errorf("write startup probe report: %w", parseErr6)
	}
	return nil
}

func validateReleaseSmoke(parseConfig releaseConfig, parseManifestPath string, parseArtifacts map[string]releaseArtifactRecord, parseStartupReport *releaseStartupRecord) (*releaseValidationRecord, error) {
	if !parseConfig.validateSmoke {
		return nil, nil
	}
	parseManifestBytes, parseErr := os.ReadFile(parseManifestPath)
	if parseErr != nil {
		return nil, fmt.Errorf("read release manifest for smoke validation: %w", parseErr)
	}
	parseManifest, parseErr := pwa.ParseWasmReleaseManifestJSON(parseManifestBytes)
	if parseErr != nil {
		return nil, fmt.Errorf("validate release manifest for smoke validation: %w", parseErr)
	}
	parseChecks := []string{"manifest parses as a valid js/wasm release record"}
	for parseName, parseArtifact := range parseManifest.Artifacts {
		parseArtifactPath := filepath.Join(parseConfig.outDir, filepath.FromSlash(parseArtifact.Path))
		parseRecord, parseErr2 := releaseArtifactRecordForPathFunc(parseConfig.outDir, parseArtifactPath)
		if parseErr2 != nil {
			return nil, fmt.Errorf("validate release artifact %q: %w", parseName, parseErr2)
		}
		if parseRecord.Bytes != parseArtifact.Bytes || !strings.EqualFold(parseRecord.SHA256, parseArtifact.SHA256) {
			return nil, fmt.Errorf("validate release artifact %q: manifest record does not match on-disk artifact", parseName)
		}
		parseChecks = append(parseChecks, fmt.Sprintf("artifact %s exists and matches manifest bytes and sha256", parseName))
	}
	if parseStartupReport == nil {
		return nil, errors.New("release smoke validation requires a startup probe result")
	}
	parseWasmContentType, parseWasmContentEncoding, parseErr := releaseSmokeFetchWasmHeaders(parseConfig.outDir, parseConfig.binaryName, parseArtifacts)
	if parseErr != nil {
		return nil, parseErr
	}
	parseChecks = append(parseChecks, "wasm asset serves with application/wasm content type")
	parseChecks = append(parseChecks, "boot-time startup probe completed")

	// product-quality scan: flag placeholder copy and missing trust-route wiring
	parseClientDir, parseHasClientDir := resolveReleaseSmokeClientDir(parseConfig)
	if parseHasClientDir {
		parseQualityChecks, parseQualityViolations, parseQErr := releaseSmokeScanProductQuality(parseClientDir)
		if parseQErr != nil {
			return nil, parseQErr
		}
		if len(parseQualityViolations) > 0 {
			return nil, fmt.Errorf("release smoke product-quality scan found %d issue(s):\n%s",
				len(parseQualityViolations), strings.Join(parseQualityViolations, "\n"))
		}
		parseChecks = append(parseChecks, parseQualityChecks...)
	} else {
		parseChecks = append(parseChecks, "product-quality: skipped because no app path or project root was provided")
	}

	parseRecord2 := &releaseValidationRecord{
		Path:                "wasm-release-validation.json",
		Checks:              parseChecks,
		StartupReportPath:   parseStartupReport.Path,
		WasmContentType:     parseWasmContentType,
		WasmContentEncoding: parseWasmContentEncoding,
	}
	parseEncoded, parseErr := releaseMarshalIndent(parseRecord2, "", "  ")
	if parseErr != nil {
		return nil, fmt.Errorf("encode release validation report: %w", parseErr)
	}
	parseEncoded = append(parseEncoded, '\n')
	if parseErr3 := os.WriteFile(filepath.Join(parseConfig.outDir, parseRecord2.Path), parseEncoded, 0644); parseErr3 != nil {
		return nil, fmt.Errorf("write release validation report: %w", parseErr3)
	}
	return parseRecord2, nil
}

func resolveReleaseSmokeClientDir(parseConfig releaseConfig) (string, bool) {
	parseAppPath := strings.TrimSpace(parseConfig.appPath)
	if parseAppPath != "" {
		parseInfo, parseErr := os.Stat(parseAppPath)
		if parseErr == nil {
			if parseInfo.IsDir() {
				return parseAppPath, true
			}
			return filepath.Dir(parseAppPath), true
		}
		return filepath.Dir(parseAppPath), true
	}

	parseRootPath := strings.TrimSpace(parseConfig.rootPath)
	if parseRootPath != "" {
		return parseRootPath, true
	}

	return "", false
}

// releaseSmokeScanProductQuality walks Go source files in parseClientDir and checks for
// known placeholder copy patterns and missing trust-route references before a release is accepted.
// It returns a list of passing checks and a (possibly empty) list of violations.
func releaseSmokeScanProductQuality(parseClientDir string) ([]string, []string, error) {
	// literal Text() call bodies that indicate unfinished or placeholder UI copy
	parsePlaceholderPatterns := []string{
		`Text("Coming soon")`,
		`Text("coming soon")`,
		`Text("TODO")`,
		`Text("Placeholder")`,
		`Text("placeholder")`,
		`Text("Lorem ipsum")`,
	}
	// source-level identifiers that must appear somewhere in the client when trust routes are wired
	parseTrustSignals := []string{
		"marketingSecurityRoute",
		"marketingStatusRoute",
		"marketingPrivacyRoute",
		"marketingTermsRoute",
	}

	var parseAllSource strings.Builder
	parseViolations := []string{}

	parseWalkErr := filepath.WalkDir(parseClientDir, func(parsePath string, parseD fs.DirEntry, parseErr error) error {
		if parseErr != nil {
			return parseErr
		}
		if parseD.IsDir() || !strings.HasSuffix(parsePath, ".go") {
			return nil
		}
		parseContents, parseReadErr := os.ReadFile(parsePath)
		if parseReadErr != nil {
			return parseReadErr
		}
		parseSource := string(parseContents)
		parseAllSource.WriteString(parseSource)
		parseRel, _ := filepath.Rel(parseClientDir, parsePath)
		for _, parsePattern := range parsePlaceholderPatterns {
			if strings.Contains(parseSource, parsePattern) {
				parseViolations = append(parseViolations,
					fmt.Sprintf("placeholder copy %q in %s", parsePattern, parseRel))
			}
		}
		return nil
	})
	if parseWalkErr != nil {
		return nil, nil, fmt.Errorf("scan client source for product quality: %w", parseWalkErr)
	}

	parseChecks := []string{}
	parseAggregate := parseAllSource.String()
	for _, parseSignal := range parseTrustSignals {
		if strings.Contains(parseAggregate, parseSignal) {
			parseChecks = append(parseChecks,
				fmt.Sprintf("product-quality: trust route %q referenced in client source", parseSignal))
		} else {
			parseViolations = append(parseViolations,
				fmt.Sprintf("product-quality: trust route %q not found in client source", parseSignal))
		}
	}
	if len(parseViolations) == 0 {
		parseChecks = append(parseChecks, "product-quality: no placeholder copy detected in client source")
	}
	return parseChecks, parseViolations, nil
}

func releaseSmokeFetchWasmHeaders(parseOutDir string, parseBinaryName string, parseArtifacts map[string]releaseArtifactRecord) (string, string, error) {
	parseListener, parseErr := net.Listen("tcp", joinHostPort(defaultHost, "0"))
	if parseErr != nil {
		return "", "", fmt.Errorf("release smoke validation listen: %w", parseErr)
	}
	defer parseListener.Close()
	parseMux := http.NewServeMux()
	parseTransportEncoding := releaseStartupTransportEncoding(parseArtifacts)
	parseMux.HandleFunc("/"+parseBinaryName, func(parseW http.ResponseWriter, parseR *http.Request) {
		releaseServeStartupWasm(parseW, parseR, parseOutDir, parseBinaryName, parseTransportEncoding)
	})
	parseServer := &http.Server{Handler: parseMux}
	defer parseServer.Close()
	go func() {
		_ = parseServer.Serve(parseListener)
	}()
	parseClient := &http.Client{
		Transport: &http.Transport{DisableCompression: true},
		Timeout:   15 * time.Second,
	}
	parseResp, parseErr := parseClient.Get("http://" + parseListener.Addr().String() + "/" + strings.TrimLeft(parseBinaryName, "/"))
	if parseErr != nil {
		return "", "", fmt.Errorf("release smoke validation fetch wasm asset: %w", parseErr)
	}
	defer parseResp.Body.Close()
	if parseResp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("release smoke validation expected wasm asset status 200, got %d", parseResp.StatusCode)
	}
	parseContentType := strings.TrimSpace(parseResp.Header.Get("Content-Type"))
	if !strings.Contains(strings.ToLower(parseContentType), "application/wasm") {
		return parseContentType, strings.TrimSpace(parseResp.Header.Get("Content-Encoding")), fmt.Errorf("release smoke validation expected application/wasm content type, got %q", parseContentType)
	}
	return parseContentType, strings.TrimSpace(parseResp.Header.Get("Content-Encoding")), nil
}

func startReleaseStartupProbeServer(parseOutDir string, parseBinaryName string, parseWasmExecPath string, parseArtifacts map[string]releaseArtifactRecord, parseTimeoutMs int) (string, string, func(), error) {
	parseListener, parseErr := net.Listen("tcp", joinHostPort(defaultHost, "0"))
	if parseErr != nil {
		return "", "", nil, fmt.Errorf("listen for startup measurement probe: %w", parseErr)
	}
	parseTransportEncoding := releaseStartupTransportEncoding(parseArtifacts)
	parseMux := http.NewServeMux()
	parseProbeHTML := renderReleaseStartupProbeHTML(parseBinaryName, parseTransportEncoding, parseTimeoutMs)
	parseMux.HandleFunc("/", func(parseW http.ResponseWriter, parseR *http.Request) {
		switch parseR.URL.Path {
		case "/":
			http.Redirect(parseW, parseR, "/__gwc/startup-probe.html", http.StatusTemporaryRedirect)
			return
		case "/__gwc/startup-probe.html":
			parseW.Header().Set("Content-Type", "text/html; charset=utf-8")
			parseW.Header().Set("Cache-Control", "no-store")
			_, _ = parseW.Write([]byte(parseProbeHTML))
			return
		case "/__gwc/wasm_exec.js":
			http.ServeFile(parseW, parseR, parseWasmExecPath)
			return
		case "/" + parseBinaryName:
			releaseServeStartupWasm(parseW, parseR, parseOutDir, parseBinaryName, parseTransportEncoding)
			return
		default:
			applyDevHeaders(parseW, parseR)
			http.FileServer(http.Dir(parseOutDir)).ServeHTTP(parseW, parseR)
			return
		}
	})
	parseServer := &http.Server{Handler: parseMux}
	go func() {
		_ = parseServer.Serve(parseListener)
	}()
	parseShutdown := func() {
		_ = parseServer.Close()
		_ = parseListener.Close()
	}
	return "http://" + parseListener.Addr().String() + "/__gwc/startup-probe.html", parseTransportEncoding, parseShutdown, nil
}

func releaseStartupTransportEncoding(parseArtifacts map[string]releaseArtifactRecord) string {
	if _, parseOk := parseArtifacts["gzip"]; parseOk {
		return "gzip"
	}
	if _, parseOk2 := parseArtifacts["brotli"]; parseOk2 {
		return "br"
	}
	return "identity"
}

func releaseServeStartupWasm(parseW http.ResponseWriter, parseR *http.Request, parseOutDir string, parseBinaryName string, parseTransportEncoding string) {
	applyDevHeaders(parseW, parseR)
	parseW.Header().Set("Cache-Control", "no-store")
	switch parseTransportEncoding {
	case "gzip":
		parseW.Header().Set("Content-Encoding", "gzip")
		http.ServeFile(parseW, parseR, filepath.Join(parseOutDir, parseBinaryName+".gz"))
	case "br":
		parseW.Header().Set("Content-Encoding", "br")
		http.ServeFile(parseW, parseR, filepath.Join(parseOutDir, parseBinaryName+".br"))
	default:
		http.ServeFile(parseW, parseR, filepath.Join(parseOutDir, parseBinaryName))
	}
}

func renderReleaseStartupProbeHTML(parseBinaryName string, parseTransportEncoding string, parseTimeoutMs int) string {
	parseProbeGzipPath := "null"
	if parseTransportEncoding == "gzip" {
		parseProbeGzipPath = jsStringLiteral("/" + parseBinaryName + ".gz")
	}
	return "<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n  <meta charset=\"utf-8\">\n  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n  <title>GWC Release Startup Probe</title>\n  <script src=\"/__gwc/wasm_exec.js\"></script>\n</head>\n<body>\n  <div id=\"app\"></div>\n  <button id=\"__gwc_probe_button\" style=\"position:fixed;top:12px;right:12px;z-index:2147483647\">Probe interaction</button>\n  <script>\n" +
		"(() => {\n" +
		"  const startup = window.__gwcStartupProbe = {\n" +
		"    startedAt: performance.now(),\n" +
		"    transportEncoding: " + jsStringLiteral(parseTransportEncoding) + ",\n" +
		"    readyMs: null,\n" +
		"    readyReason: '',\n" +
		"    interactionMs: null,\n" +
		"    instantiate: null,\n" +
		"    decompression: null,\n" +
		"    error: null\n" +
		"  };\n" +
		"  const mount = document.getElementById('app');\n" +
		"  const button = document.getElementById('__gwc_probe_button');\n" +
		"  const markReady = (reason) => {\n" +
		"    if (startup.readyMs !== null) return;\n" +
		"    startup.readyMs = performance.now() - startup.startedAt;\n" +
		"    startup.readyReason = reason;\n" +
		"  };\n" +
		"  if ((mount.textContent || '').trim() !== '' || mount.childNodes.length > 0) {\n" +
		"    markReady('preexisting');\n" +
		"  }\n" +
		"  const observer = new MutationObserver(() => {\n" +
		"    if ((mount.textContent || '').trim() !== '' || mount.childNodes.length > 0) {\n" +
		"      observer.disconnect();\n" +
		"      requestAnimationFrame(() => markReady('mutation'));\n" +
		"    }\n" +
		"  });\n" +
		"  observer.observe(mount, { childList: true, subtree: true, characterData: true });\n" +
		"  setTimeout(() => markReady('timeout'), " + fmt.Sprintf("%d", parseTimeoutMs) + ");\n" +
		"  button.addEventListener('click', () => {\n" +
		"    const interactionStart = performance.now();\n" +
		"    requestAnimationFrame(() => {\n" +
		"      startup.interactionMs = performance.now() - interactionStart;\n" +
		"    });\n" +
		"  });\n" +
		"  const originalInstantiateStreaming = WebAssembly.instantiateStreaming.bind(WebAssembly);\n" +
		"  WebAssembly.instantiateStreaming = async (source, importObject) => {\n" +
		"    const start = performance.now();\n" +
		"    try {\n" +
		"      const result = await originalInstantiateStreaming(source, importObject);\n" +
		"      startup.instantiate = { mode: 'instantiateStreaming', durationMs: performance.now() - start };\n" +
		"      return result;\n" +
		"    } catch (error) {\n" +
		"      startup.instantiate = { mode: 'instantiateStreaming', durationMs: performance.now() - start, error: String(error) };\n" +
		"      throw error;\n" +
		"    }\n" +
		"  };\n" +
		"  const gzipProbePath = " + parseProbeGzipPath + ";\n" +
		"  if (gzipProbePath && typeof DecompressionStream === 'function') {\n" +
		"    fetch(gzipProbePath, { cache: 'no-store' })\n" +
		"      .then(async (response) => {\n" +
		"        const fetchStart = performance.now();\n" +
		"        const encoded = await response.arrayBuffer();\n" +
		"        const fetchDone = performance.now();\n" +
		"        const decoded = await new Response(new Blob([encoded]).stream().pipeThrough(new DecompressionStream('gzip'))).arrayBuffer();\n" +
		"        const decodedDone = performance.now();\n" +
		"        startup.decompression = {\n" +
		"          encoding: 'gzip',\n" +
		"          encodedBytes: encoded.byteLength,\n" +
		"          decodedBytes: decoded.byteLength,\n" +
		"          fetchMs: fetchDone - fetchStart,\n" +
		"          decompressMs: decodedDone - fetchDone\n" +
		"        };\n" +
		"      })\n" +
		"      .catch((error) => {\n" +
		"        startup.decompression = { encoding: 'gzip', error: String(error) };\n" +
		"      });\n" +
		"  }\n" +
		"  const go = new Go();\n" +
		"  WebAssembly.instantiateStreaming(fetch(" + jsStringLiteral("/"+parseBinaryName) + ", { cache: 'no-store' }), go.importObject)\n" +
		"    .then((result) => go.run(result.instance))\n" +
		"    .catch((error) => {\n" +
		"      startup.error = String(error);\n" +
		"      markReady('error');\n" +
		"    });\n" +
		"})();\n" +
		"  </script>\n</body>\n</html>\n"
}
