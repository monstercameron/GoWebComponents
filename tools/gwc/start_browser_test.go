//go:build playwrightgo

package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	playwright "github.com/mxschmitt/playwright-go"
)

func TestDefaultStarterTemplatesMountInHeadlessBrowser(parseT *testing.T) {
	if testing.Short() {
		parseT.Skip("skipping starter browser first-pixel smoke in short mode")
	}

	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseLauncher := launcher{repoRoot: parseRepoRoot}
	parsePresets := defaultStartPresets()
	if len(parsePresets) == 0 {
		parseT.Fatal("expected at least one starter preset")
	}

	for _, parsePreset := range parsePresets {
		parsePreset := parsePreset
		parseT.Run(parsePreset.Key, func(parseT2 *testing.T) {
			parseProjectName := "browser-starter-" + parsePreset.Key
			parseTargetDir := filepath.Join(parseT2.TempDir(), parseProjectName)
			parseResult, parseErr2 := parseLauncher.generateStartScaffold(startSelection{
				Preset:        parsePreset,
				ProjectMode:   scaffoldProjectModeContributorLinked,
				ProjectName:   parseProjectName,
				ModulePath:    "github.com/example/" + parseProjectName,
				Author:        "Test Author",
				Version:       "1.2.3",
				Description:   parsePreset.Summary,
				TargetDir:     parseTargetDir,
				SkipGoModTidy: false,
			})
			if parseErr2 != nil {
				parseT2.Fatalf("generate starter scaffold for preset %q: %v", parsePreset.Key, parseErr2)
			}

			parseBuildErr := parseLauncher.runBuild([]string{
				"-app", parseResult.AppPath,
				"-root", parseResult.TargetDir,
				"-out", filepath.Join(parseResult.TargetDir, filepath.FromSlash(scaffoldWASMOutputPath())),
				"-profile", "development",
			})
			if parseBuildErr != nil {
				parseT2.Fatalf("gwc build failed for starter preset %q: %v", parsePreset.Key, parseBuildErr)
			}

			parseServer := httptest.NewServer(starterStaticHandler(parseTargetDir))
			parseT2.Cleanup(parseServer.Close)
			withLiveReloadClientPage(parseT2, func(parsePage playwright.Page) {
				parsePage.OnConsole(func(parseMsg playwright.ConsoleMessage) {
					if parseMsg.Type() == "error" || strings.Contains(parseMsg.Text(), "[gwc]") {
						parseT2.Logf("browser console [%s]: %s", parseMsg.Type(), parseMsg.Text())
					}
				})

				if _, parseErr3 := parsePage.Goto(parseServer.URL+"/", playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); parseErr3 != nil {
					parseT2.Fatalf("goto starter preset %q: %v", parsePreset.Key, parseErr3)
				}
				parseInitialText, parseErr4 := parsePage.TextContent("body")
				if parseErr4 != nil {
					parseT2.Fatalf("read first-pixel app text for preset %q: %v", parsePreset.Key, parseErr4)
				}
				if strings.TrimSpace(parseInitialText) == "" {
					parseT2.Fatalf("starter preset %q rendered an empty document before wasm mount", parsePreset.Key)
				}
				if _, parseErr5 := parsePage.WaitForSelector("#app .counter", playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(120000)}); parseErr5 != nil {
					parseT2.Fatalf("wait for mounted wasm tree for starter preset %q: %v", parsePreset.Key, parseErr5)
				}
				parseMountedText, parseErr6 := parsePage.TextContent("#app")
				if parseErr6 != nil {
					parseT2.Fatalf("read mounted app text for preset %q: %v", parsePreset.Key, parseErr6)
				}
				if !strings.Contains(parseMountedText, parseProjectName) || !strings.Contains(parseMountedText, "Count: 0") {
					parseT2.Fatalf("starter preset %q mounted unexpected app text: %q", parsePreset.Key, parseMountedText)
				}
				if _, parseErr7 := parsePage.WaitForSelector("#gwc-first-pixel", playwright.PageWaitForSelectorOptions{
					State:   playwright.WaitForSelectorStateHidden,
					Timeout: playwright.Float(120000),
				}); parseErr7 != nil {
					parseT2.Fatalf("starter preset %q did not remove first-pixel fallback after mount: %v", parsePreset.Key, parseErr7)
				}
			})
		})
	}
}

func starterStaticHandler(parseRoot string) http.Handler {
	parseFileServer := http.FileServer(http.Dir(parseRoot))
	return http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		if strings.HasSuffix(parseR.URL.Path, ".wasm") {
			parseW.Header().Set("Content-Type", "application/wasm")
		}
		if parseR.URL.Path == "" || parseR.URL.Path == "/" {
			parseW.Header().Set("Cache-Control", "no-store")
		}
		parseFileServer.ServeHTTP(parseW, parseR)
	})
}
