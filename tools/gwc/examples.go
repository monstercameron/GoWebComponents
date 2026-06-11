package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	gwchtml "github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

type exampleLink struct {
	Name string
	Href string
}

type exampleCatalogEntry struct {
	Name        string   `json:"name"`
	Href        string   `json:"href"`
	HTMLFile    string   `json:"htmlFile"`
	Title       string   `json:"title,omitempty"`
	UsesWasm    bool     `json:"usesWasm"`
	WasmBinary  string   `json:"wasmBinary,omitempty"`
	MultiClient bool     `json:"multiClient"`
	Tags        []string `json:"tags,omitempty"`
}

type generatedExamplePage struct {
	RoutePath     string
	DirName       string
	HTMLFile      string
	Title         string
	WasmBinary    string
	ManifestHref  string
	Description   string
	GeneratedFrom string
}

type examplesCatalogPayload struct {
	GeneratedAt         string                `json:"generatedAt"`
	TotalExamples       int                   `json:"totalExamples"`
	WasmExamples        int                   `json:"wasmExamples"`
	MultiClientExamples int                   `json:"multiClientExamples"`
	Examples            []exampleCatalogEntry `json:"examples"`
}

type exampleRouteInfo struct {
	Name      string
	RoutePath string
}

var examplesListen = net.Listen

var examplesServe = func(server *http.Server, listener net.Listener) error {
	return server.Serve(listener)
}

var renderExamplesUIBootstrapScript = ui.RenderBootstrapScript

var renderExamplesBootstrapScriptFunc = renderExamplesBootstrapScript

var examplesCatalogMarshalIndent = json.MarshalIndent

var renderExamplesToString = ui.RenderToString

var exampleCatalogRouteAliasPaths = map[string]string{
	"09-atoms":                "public/state-atoms",
	"17-ssr-routing":          "public/static-server-side-rendering-routing",
	"18-ssr-server-routing":   "server/server-side-rendering-routing",
	"33-lazy":                 "public/lazy-loading",
	"71-hydrate":              "public/hydration",
	"73-ssr-bootstrap":        "server/server-side-rendering-bootstrap",
	"74-ssr-route-data-reuse": "public/server-side-rendering-route-data-reuse",
	"84-ssr-i18n-bootstrap":   "public/server-side-rendering-internationalization-bootstrap",
	"93-ssr-cache-bootstrap":  "public/server-side-rendering-cache-bootstrap",
	"97-pwa-installability":   "public/progressive-web-app-installability",
	"97-pwa-multi-client":     "public/progressive-web-app-multi-client",
	"97-pwa-offline-cache":    "public/progressive-web-app-offline-cache",
}

var executeExamplesBuild = executeBuild

func (parseL launcher) runExamples(parseArgs []string) error {
	if isExamplesManagedAction(parseArgs) {
		return parseL.runExamplesManaged(parseArgs)
	}
	if len(parseArgs) >= 2 && isExamplesManagedPathCommand(parseArgs[0], parseArgs[1:]) {
		return parseL.runExamplesManaged(buildExamplesManagedPathCommandArgs(parseArgs[0], parseArgs[1:]))
	}
	if len(parseArgs) > 0 && strings.EqualFold(strings.TrimSpace(parseArgs[0]), "build-public-site") {
		return parseL.runExamplesBuildPublicSite(parseArgs[1:])
	}

	parseFs := flag.NewFlagSet("examples", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseHost := parseFs.String("host", defaultHost, "Host to bind")
	parsePort := parseFs.String("port", defaultPort, "Port to bind")
	parseExportStaticCatalog := parseFs.String("export-static-catalog", "", "Write a static examples catalog JSON file for static hosting and exit")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	if _, parseErr2 := os.Stat(parseL.examplesDir); parseErr2 != nil {
		return fmt.Errorf("examples directory not found: %w", parseErr2)
	}

	if strings.TrimSpace(*parseExportStaticCatalog) != "" {
		if parseErr3 := parseL.writeStaticExamplesCatalogFile(*parseExportStaticCatalog); parseErr3 != nil {
			return parseErr3
		}
		fmt.Printf("Wrote static examples catalog to %s\n", *parseExportStaticCatalog)
		return nil
	}

	parseAddr := joinHostPort(*parseHost, *parsePort)
	parseListener, parseErr4 := examplesListen("tcp", parseAddr)
	if parseErr4 != nil {
		return parseErr4
	}
	defer parseListener.Close()

	parseServer := &http.Server{
		Addr:    parseAddr,
		Handler: parseL.newExamplesHandler(*parseHost, *parsePort),
	}

	fmt.Printf("GWC examples server listening on http://%s\n", parseAddr)
	fmt.Printf("Examples: http://%s\n", parseAddr)
	fmt.Printf("Counter:  http://%s/examples/public/counter/\n", parseAddr)

	if parseErr5 := examplesServe(parseServer, parseListener); parseErr5 != nil && !errors.Is(parseErr5, http.ErrServerClosed) {
		return parseErr5
	}
	return nil
}

func (parseL launcher) newExamplesHandler(parseHost string, parsePort string) http.Handler {
	parseExamplesServer := http.StripPrefix("/examples/", http.FileServer(http.Dir(parseL.examplesDir)))
	parseStaticServer := http.StripPrefix("/static/", http.FileServer(http.Dir(parseL.staticDir)))
	// Wasm artifacts are multi-megabyte and gzip ~4-5x smaller; serving them
	// uncompressed makes cold start network-bound on anything but localhost.
	parseWasmServer := newGzipWASMHandler(http.StripPrefix("/static/bin/", http.FileServer(http.Dir(parseL.resolvedExamplesWasmDir()))))

	parseMux := http.NewServeMux()
	parseMux.HandleFunc("/healthz", func(parseW http.ResponseWriter, parseR *http.Request) {
		writeJSON(parseW, http.StatusOK, map[string]interface{}{
			"ok":      true,
			"service": "gowebcomponents-gwc-examples",
			"time":    time.Now().UTC().Format(time.RFC3339),
			"root":    parseL.repoRoot,
			"host":    parseHost,
			"port":    parsePort,
		})
	})
	parseMux.HandleFunc("/examples/list", func(parseW2 http.ResponseWriter, parseR2 *http.Request) {
		parseQuery := strings.TrimSpace(parseR2.URL.Query().Get("q"))
		parseLinks, parseErr := parseL.buildExamplesListing()
		if parseErr != nil {
			writeJSON(parseW2, http.StatusInternalServerError, map[string]string{
				"error": "examples_listing_failed",
				"hint":  parseErr.Error(),
			})
			return
		}
		parseLinks = filterExampleLinks(parseLinks, parseQuery)
		parseW2.Header().Set("Content-Type", "text/html; charset=utf-8")
		parseW2.WriteHeader(http.StatusOK)
		_, _ = parseW2.Write([]byte(renderExamplesListingHTML(parseLinks, parseQuery)))
	})
	parseMux.HandleFunc("/examples/catalog.json", func(parseW3 http.ResponseWriter, parseR3 *http.Request) {
		parseCatalog, parseErr2 := parseL.buildExamplesCatalog()
		if parseErr2 != nil {
			writeJSON(parseW3, http.StatusInternalServerError, map[string]string{
				"error": "examples_catalog_failed",
				"hint":  parseErr2.Error(),
			})
			return
		}
		writeJSON(parseW3, http.StatusOK, parseCatalog)
	})
	parseMux.HandleFunc("/examples", func(parseW4 http.ResponseWriter, parseR4 *http.Request) {
		if parseR4.URL.Path != "/examples" {
			parseExamplesServer.ServeHTTP(parseW4, parseR4)
			return
		}
		parseW4.Header().Set("Content-Type", "text/html; charset=utf-8")
		parseW4.WriteHeader(http.StatusOK)
		_, _ = parseW4.Write([]byte(renderExamplesAppShellHTML("/examples/", "/examples/")))
	})
	parseMux.HandleFunc("/examples/static/index.html", func(parseW5 http.ResponseWriter, parseR5 *http.Request) {
		if parseR5.URL.Path != "/examples/static/index.html" {
			parseExamplesServer.ServeHTTP(parseW5, parseR5)
			return
		}
		parseW5.Header().Set("Content-Type", "text/html; charset=utf-8")
		parseW5.WriteHeader(http.StatusOK)
		_, _ = parseW5.Write([]byte(renderExamplesAppShellHTML("/examples/", "/examples/")))
	})
	parseMux.HandleFunc("/examples/", func(parseW6 http.ResponseWriter, parseR6 *http.Request) {
		if parseR6.URL.Path == "/examples/" {
			parseW6.Header().Set("Content-Type", "text/html; charset=utf-8")
			parseW6.WriteHeader(http.StatusOK)
			_, _ = parseW6.Write([]byte(renderExamplesAppShellHTML("/examples/", "/examples/")))
			return
		}
		parseTrimmed := strings.TrimPrefix(filepath.ToSlash(parseR6.URL.Path), "/examples/")
		parseTrimmed = strings.TrimSpace(parseTrimmed)
		if parseTrimmed == "" {
			parseW6.Header().Set("Content-Type", "text/html; charset=utf-8")
			parseW6.WriteHeader(http.StatusOK)
			_, _ = parseW6.Write([]byte(renderExamplesAppShellHTML("/examples/", "/examples/")))
			return
		}
		parseRouteInfo, parseHasRoute, parseErr3 := parseL.resolveExampleRouteInfo(parseR6.URL.Path)
		if parseErr3 != nil {
			parseErrorCode := "examples_route_failed"
			if strings.HasSuffix(strings.ToLower(parseR6.URL.Path), ".html") {
				parseErrorCode = "examples_html_route_failed"
			}
			writeJSON(parseW6, http.StatusInternalServerError, map[string]string{
				"error": parseErrorCode,
				"hint":  parseErr3.Error(),
			})
			return
		}
		if parseHasRoute && filepath.ToSlash(parseR6.URL.Path) != parseRouteInfo.RoutePath {
			http.Redirect(parseW6, parseR6, parseRouteInfo.RoutePath, http.StatusFound)
			return
		}
		parseExamplePage, parseOk, parseErr4 := parseL.resolveGeneratedExamplePage(parseR6.URL.Path)
		if parseErr4 != nil {
			writeJSON(parseW6, http.StatusInternalServerError, map[string]string{
				"error": "examples_route_failed",
				"hint":  parseErr4.Error(),
			})
			return
		}
		if parseOk {
			parseW6.Header().Set("Content-Type", "text/html; charset=utf-8")
			parseW6.WriteHeader(http.StatusOK)
			_, _ = parseW6.Write([]byte(renderGeneratedExampleHTML(parseExamplePage)))
			return
		}
		parseAssetAliasPath, parseHasAssetAlias, parseErr5 := parseL.resolveExampleAssetAliasPath(parseR6.URL.Path)
		if parseErr5 != nil {
			writeJSON(parseW6, http.StatusInternalServerError, map[string]string{
				"error": "examples_route_failed",
				"hint":  parseErr5.Error(),
			})
			return
		}
		if parseHasAssetAlias {
			parseRewritten := parseR6.Clone(parseR6.Context())
			parseRewritten.URL.Path = parseAssetAliasPath
			parseExamplesServer.ServeHTTP(parseW6, parseRewritten)
			return
		}
		parseExamplesServer.ServeHTTP(parseW6, parseR6)
	})
	parseMux.Handle("/static/bin/", parseWasmServer)
	parseMux.Handle("/static/", parseStaticServer)
	parseMux.HandleFunc("/", func(parseW7 http.ResponseWriter, parseR7 *http.Request) {
		if parseR7.URL.Path == "/" {
			parseW7.Header().Set("Content-Type", "text/html; charset=utf-8")
			parseW7.WriteHeader(http.StatusOK)
			_, _ = parseW7.Write([]byte(renderExamplesAppShellHTML("/", "/")))
			return
		}
		parseExamplesServer.ServeHTTP(parseW7, parseR7)
	})

	return http.HandlerFunc(func(parseW8 http.ResponseWriter, parseR8 *http.Request) {
		parseStart := time.Now()
		parseWrapped := &statusWriter{ResponseWriter: parseW8, status: http.StatusOK}
		applyDevHeaders(parseWrapped, parseR8)
		parseMux.ServeHTTP(parseWrapped, parseR8)
		fmt.Printf("%d %s %s %dms\n", parseWrapped.status, parseR8.Method, parseR8.URL.RequestURI(), time.Since(parseStart).Milliseconds())
	})
}

func (parseL launcher) buildExamplesListing() ([]exampleLink, error) {
	parseCatalog, parseErr := parseL.buildExamplesCatalog()
	if parseErr != nil {
		return nil, parseErr
	}
	parseLinks := make([]exampleLink, 0, len(parseCatalog.Examples))
	for _, parseEntry := range parseCatalog.Examples {
		parseLinks = append(parseLinks, exampleLink{Name: parseEntry.Name, Href: parseEntry.Href})
	}
	return parseLinks, nil
}

func (parseL launcher) buildExamplesCatalog() (examplesCatalogPayload, error) {
	parseCatalog, isParseLoaded, parseErr := parseL.buildExampleManifestCatalog(false)
	if parseErr != nil || isParseLoaded {
		return parseCatalog, parseErr
	}
	return parseL.buildExamplesCatalogWithHref(func(parseDirPath string, parseDirName string, parseHtmlFile string) string {
		return "/examples/" + parseDirName + "/"
	})
}

func (parseL launcher) buildStaticExamplesCatalog() (examplesCatalogPayload, error) {
	parseCatalog, isParseLoaded, parseErr := parseL.buildExampleManifestCatalog(true)
	if parseErr != nil || isParseLoaded {
		return parseCatalog, parseErr
	}
	return parseL.buildExamplesCatalogWithHref(func(parseDirPath string, parseDirName string, parseHtmlFile string) string {
		return "../" + parseDirName + "/" + parseHtmlFile
	})
}

// loadExampleCatalogManifest loads the committed examples catalog manifest when it is available.
func (parseL launcher) loadExampleCatalogManifest() (examplesCatalogPayload, bool, error) {
	parseManifestPath := filepath.Join(parseL.staticDir, "catalog.json")
	if !fileExists(parseManifestPath) {
		return examplesCatalogPayload{}, false, nil
	}
	parseContent, parseErr := os.ReadFile(parseManifestPath)
	if parseErr != nil {
		return examplesCatalogPayload{}, false, fmt.Errorf("read examples catalog manifest: %w", parseErr)
	}
	var parseCatalog examplesCatalogPayload
	if parseErr2 := json.Unmarshal(parseContent, &parseCatalog); parseErr2 != nil {
		return examplesCatalogPayload{}, false, fmt.Errorf("decode examples catalog manifest: %w", parseErr2)
	}
	return parseCatalog, true, nil
}

// resolveExampleCurrentDir maps one legacy catalog name onto the current grouped examples directory.
func (parseL launcher) resolveExampleCurrentDir(parseLegacyName string) (string, bool, error) {
	parseLegacyName = strings.TrimSpace(parseLegacyName)
	if parseLegacyName == "" {
		return "", false, nil
	}
	parseCheckDir := func(parseRelativeDir string) (string, bool, error) {
		parseRelativeDir = strings.TrimSpace(parseRelativeDir)
		if parseRelativeDir == "" {
			return "", false, nil
		}
		parseDirPath := filepath.Join(parseL.examplesDir, filepath.FromSlash(parseRelativeDir))
		parseInfo, parseErr := os.Stat(parseDirPath)
		if parseErr != nil {
			if os.IsNotExist(parseErr) {
				return "", false, nil
			}
			return "", false, parseErr
		}
		if !parseInfo.IsDir() {
			return "", false, nil
		}
		return filepath.ToSlash(parseRelativeDir), true, nil
	}

	if parseRelativeDir, parseOk, parseErr := parseCheckDir(parseLegacyName); parseErr != nil || parseOk {
		return parseRelativeDir, parseOk, parseErr
	}
	if parseAliasDir, ok := exampleCatalogRouteAliasPaths[parseLegacyName]; ok {
		if parseRelativeDir, parseOk, parseErr := parseCheckDir(parseAliasDir); parseErr != nil || parseOk {
			return parseRelativeDir, parseOk, parseErr
		}
	}

	parseSlug := parseLegacyName
	if parseParts := strings.SplitN(parseLegacyName, "-", 2); len(parseParts) == 2 {
		parseSlug = parseParts[1]
	}
	for _, parseGroup := range []string{"public", "server", "testing"} {
		parseRelativeDir := filepath.ToSlash(filepath.Join(parseGroup, parseSlug))
		if parseResolvedDir, parseOk, parseErr := parseCheckDir(parseRelativeDir); parseErr != nil || parseOk {
			return parseResolvedDir, parseOk, parseErr
		}
	}
	return "", false, nil
}

// buildExampleCatalogEntryFromManifest builds one catalog entry from the committed examples manifest plus current filesystem state.
func (parseL launcher) buildExampleCatalogEntryFromManifest(parseManifestEntry exampleCatalogEntry, isParseStaticHref bool) (exampleCatalogEntry, bool, error) {
	parseRelativeDir, parseOk, parseErr := parseL.resolveExampleCurrentDir(parseManifestEntry.Name)
	if parseErr != nil || !parseOk {
		return exampleCatalogEntry{}, parseOk, parseErr
	}
	parseDirPath := filepath.Join(parseL.examplesDir, filepath.FromSlash(parseRelativeDir))
	parseHTMLFile := strings.TrimSpace(parseManifestEntry.HTMLFile)
	if parseHTMLFile == "" {
		parseHTMLFile2, parseOk2, parseErr2 := firstHTMLFileName(parseDirPath)
		if parseErr2 != nil || !parseOk2 {
			return exampleCatalogEntry{}, parseOk2, parseErr2
		}
		parseHTMLFile = parseHTMLFile2
	}

	parseWasmBinary := strings.TrimSpace(parseManifestEntry.WasmBinary)
	isParseUsesWasm := false
	if parseWasmBinary != "" {
		if fileExists(filepath.Join(parseL.resolvedExamplesWasmDir(), parseWasmBinary)) {
			isParseUsesWasm = true
		} else {
			parseWasmBinary = ""
		}
	}
	if !isParseUsesWasm && parseWasmBinary == "" {
		parseHTMLPath := filepath.Join(parseDirPath, parseHTMLFile)
		if fileExists(parseHTMLPath) {
			parseDetectedWasmBinary, isParseDetectedWasm, parseErr3 := detectAvailableExampleWasmBinary(parseL.resolvedExamplesWasmDir(), parseHTMLPath)
			if parseErr3 != nil {
				return exampleCatalogEntry{}, false, parseErr3
			}
			parseWasmBinary = parseDetectedWasmBinary
			isParseUsesWasm = isParseDetectedWasm
		}
	}

	parseHref := "/examples/" + parseRelativeDir + "/"
	if isParseStaticHref {
		parseHref = "../" + parseManifestEntry.Name + "/" + parseHTMLFile
	}
	parseTags := exampleCatalogTags(parseManifestEntry.Name, parseWasmBinary, isParseUsesWasm)
	return exampleCatalogEntry{
		Name:        parseManifestEntry.Name,
		Href:        parseHref,
		HTMLFile:    parseHTMLFile,
		Title:       firstNonEmpty(strings.TrimSpace(parseManifestEntry.Title), defaultExampleTitle(parseManifestEntry.Name, parseHTMLFile)),
		UsesWasm:    isParseUsesWasm,
		WasmBinary:  parseWasmBinary,
		MultiClient: hasAnyTag(parseTags, "multi-client", "cross-tab", "multi-window"),
		Tags:        parseTags,
	}, true, nil
}

// buildExampleManifestCatalog builds one examples catalog from the committed manifest when it is available.
func (parseL launcher) buildExampleManifestCatalog(isParseStaticHref bool) (examplesCatalogPayload, bool, error) {
	parseManifest, isParseLoaded, parseErr := parseL.loadExampleCatalogManifest()
	if parseErr != nil || !isParseLoaded {
		return examplesCatalogPayload{}, isParseLoaded, parseErr
	}

	parseCatalogEntries := make([]exampleCatalogEntry, 0, len(parseManifest.Examples))
	for _, parseManifestEntry := range parseManifest.Examples {
		parseCatalogEntry, parseOk, parseErr2 := parseL.buildExampleCatalogEntryFromManifest(parseManifestEntry, isParseStaticHref)
		if parseErr2 != nil {
			return examplesCatalogPayload{}, true, parseErr2
		}
		if !parseOk {
			return examplesCatalogPayload{}, true, fmt.Errorf("resolve current example directory for %s", parseManifestEntry.Name)
		}
		parseCatalogEntries = append(parseCatalogEntries, parseCatalogEntry)
	}
	sort.Slice(parseCatalogEntries, func(parseI, parseJ int) bool {
		return parseCatalogEntries[parseI].Name < parseCatalogEntries[parseJ].Name
	})

	parseWasmCount := 0
	parseMultiClientCount := 0
	for _, parseCatalogEntry := range parseCatalogEntries {
		if parseCatalogEntry.UsesWasm {
			parseWasmCount++
		}
		if parseCatalogEntry.MultiClient {
			parseMultiClientCount++
		}
	}

	return examplesCatalogPayload{
		GeneratedAt:         time.Now().UTC().Format(time.RFC3339),
		TotalExamples:       len(parseCatalogEntries),
		WasmExamples:        parseWasmCount,
		MultiClientExamples: parseMultiClientCount,
		Examples:            parseCatalogEntries,
	}, true, nil
}

func (parseL launcher) buildExamplesCatalogWithHref(parseResolveHref func(dirPath string, dirName string, htmlFile string) string) (examplesCatalogPayload, error) {
	parseEntries, parseErr := os.ReadDir(parseL.examplesDir)
	if parseErr != nil {
		return examplesCatalogPayload{}, parseErr
	}

	parsePattern := regexp.MustCompile(`^\d+-`)
	parseCatalogEntries := make([]exampleCatalogEntry, 0)
	for _, parseEntry := range parseEntries {
		if !parseEntry.IsDir() || !parsePattern.MatchString(parseEntry.Name()) {
			continue
		}
		parseDirPath := filepath.Join(parseL.examplesDir, parseEntry.Name())
		parseCatalogEntry, parseOk, parseErr2 := parseL.buildExampleCatalogEntry(parseDirPath, parseEntry.Name(), parseResolveHref)
		if parseErr2 != nil {
			return examplesCatalogPayload{}, parseErr2
		}
		if parseOk {
			parseCatalogEntries = append(parseCatalogEntries, parseCatalogEntry)
		}
	}
	sort.Slice(parseCatalogEntries, func(parseI, parseJ int) bool {
		return parseCatalogEntries[parseI].Name < parseCatalogEntries[parseJ].Name
	})

	parseWasmCount := 0
	parseMultiClientCount := 0
	for _, parseEntry2 := range parseCatalogEntries {
		if parseEntry2.UsesWasm {
			parseWasmCount++
		}
		if parseEntry2.MultiClient {
			parseMultiClientCount++
		}
	}

	return examplesCatalogPayload{
		GeneratedAt:         time.Now().UTC().Format(time.RFC3339),
		TotalExamples:       len(parseCatalogEntries),
		WasmExamples:        parseWasmCount,
		MultiClientExamples: parseMultiClientCount,
		Examples:            parseCatalogEntries,
	}, nil
}

func (parseL launcher) writeStaticExamplesCatalogFile(parseTargetPath string) error {
	parseCatalog, parseErr := parseL.buildStaticExamplesCatalog()
	if parseErr != nil {
		return parseErr
	}
	parseTargetPath = strings.TrimSpace(parseTargetPath)
	if parseTargetPath == "" {
		return errors.New("static catalog output path is required")
	}
	if !filepath.IsAbs(parseTargetPath) {
		parseResolved, parseErr2 := filepath.Abs(parseTargetPath)
		if parseErr2 != nil {
			return fmt.Errorf("resolve static catalog path: %w", parseErr2)
		}
		parseTargetPath = parseResolved
	}
	if parseErr3 := os.MkdirAll(filepath.Dir(parseTargetPath), 0755); parseErr3 != nil {
		return fmt.Errorf("create static catalog directory: %w", parseErr3)
	}
	parseEncoded, parseErr := examplesCatalogMarshalIndent(parseCatalog, "", "  ")
	if parseErr != nil {
		return fmt.Errorf("encode static catalog: %w", parseErr)
	}
	parseEncoded = append(parseEncoded, '\n')
	if parseErr4 := os.WriteFile(parseTargetPath, parseEncoded, 0644); parseErr4 != nil {
		return fmt.Errorf("write static catalog: %w", parseErr4)
	}
	return nil
}

func filterExampleLinks(parseLinks []exampleLink, parseQuery string) []exampleLink {
	parseQuery = strings.TrimSpace(strings.ToLower(parseQuery))
	if parseQuery == "" {
		return parseLinks
	}
	parseTerms := strings.Fields(parseQuery)
	parseFiltered := make([]exampleLink, 0, len(parseLinks))
	for _, parseLink := range parseLinks {
		parseHaystack := strings.ToLower(parseLink.Name + " " + parseLink.Href)
		isParseMatchesAll := true
		for _, parseTerm := range parseTerms {
			if !strings.Contains(parseHaystack, parseTerm) {
				isParseMatchesAll = false
				break
			}
		}
		if isParseMatchesAll {
			parseFiltered = append(parseFiltered, parseLink)
		}
	}
	return parseFiltered
}

func firstHTMLFileName(parseDirPath string) (string, bool, error) {
	parseEntries, parseErr := os.ReadDir(parseDirPath)
	if parseErr != nil {
		return "", false, parseErr
	}
	for _, parseEntry := range parseEntries {
		if parseEntry.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(parseEntry.Name()), ".html") {
			return parseEntry.Name(), true, nil
		}
	}
	return "", false, nil
}

func (parseL launcher) buildExampleCatalogEntry(parseDirPath string, parseDirName string, parseResolveHref func(dirPath string, dirName string, htmlFile string) string) (exampleCatalogEntry, bool, error) {
	parseHtmlFile, parseOk, parseErr := firstHTMLFileName(parseDirPath)
	if parseErr != nil || !parseOk {
		return exampleCatalogEntry{}, parseOk, parseErr
	}
	parseHtmlPath := filepath.Join(parseDirPath, parseHtmlFile)
	parseWasmBinary, parseUsesWasm, parseErr := detectAvailableExampleWasmBinary(parseL.resolvedExamplesWasmDir(), parseHtmlPath)
	if parseErr != nil {
		return exampleCatalogEntry{}, false, parseErr
	}
	parseTitle, _ := detectHTMLTitle(parseHtmlPath)
	parseTags := exampleCatalogTags(parseDirName, parseWasmBinary, parseUsesWasm)
	parseHref := "/examples/" + parseDirName + "/"
	if parseResolveHref != nil {
		parseHref = parseResolveHref(parseDirPath, parseDirName, parseHtmlFile)
	}
	return exampleCatalogEntry{
		Name:        parseDirName,
		Href:        parseHref,
		HTMLFile:    parseHtmlFile,
		Title:       parseTitle,
		UsesWasm:    parseUsesWasm,
		WasmBinary:  parseWasmBinary,
		MultiClient: hasAnyTag(parseTags, "multi-client", "cross-tab", "multi-window"),
		Tags:        parseTags,
	}, true, nil
}

func (parseL launcher) resolveExampleCatalogEntry(parseDirName string) (exampleCatalogEntry, bool, error) {
	parseDirName = strings.TrimSpace(parseDirName)
	if parseDirName == "" {
		return exampleCatalogEntry{}, false, nil
	}
	parseManifest, isParseLoaded, parseErr := parseL.loadExampleCatalogManifest()
	if parseErr != nil {
		return exampleCatalogEntry{}, false, parseErr
	}
	if isParseLoaded {
		for _, parseManifestEntry := range parseManifest.Examples {
			if parseManifestEntry.Name != parseDirName {
				continue
			}
			return parseL.buildExampleCatalogEntryFromManifest(parseManifestEntry, false)
		}
	}
	parseDirPath := filepath.Join(parseL.examplesDir, parseDirName)
	parseInfo, parseErr := os.Stat(parseDirPath)
	if parseErr != nil {
		if os.IsNotExist(parseErr) {
			return exampleCatalogEntry{}, false, nil
		}
		return exampleCatalogEntry{}, false, parseErr
	}
	if !parseInfo.IsDir() {
		return exampleCatalogEntry{}, false, nil
	}
	return parseL.buildExampleCatalogEntry(parseDirPath, parseDirName, func(parseDirPath2 string, parseDirName2 string, parseHtmlFile string) string {
		return "/examples/" + parseDirName2 + "/"
	})
}

// resolveExampleLegacyNameFromGroupedRoute maps one grouped route slug back to one numbered catalog directory when only flat directories are available.
func (parseL launcher) resolveExampleLegacyNameFromGroupedRoute(parseGroup string, parseSlug string) (string, bool, error) {
	parseGroup = strings.TrimSpace(parseGroup)
	parseSlug = strings.TrimSpace(parseSlug)
	if parseGroup == "" || parseSlug == "" {
		return "", false, nil
	}
	parseInfo, parseErr := os.Stat(parseL.examplesDir)
	if parseErr != nil {
		if os.IsNotExist(parseErr) {
			return "", false, nil
		}
		return "", false, parseErr
	}
	if !parseInfo.IsDir() {
		return "", false, nil
	}
	parseEntries, parseErr := os.ReadDir(parseL.examplesDir)
	if parseErr != nil {
		if os.IsNotExist(parseErr) {
			return "", false, nil
		}
		return "", false, parseErr
	}
	parsePattern := regexp.MustCompile(`^\d+-`)
	parseExpectedRoute := filepath.ToSlash(filepath.Join(parseGroup, parseSlug))
	for _, parseEntry := range parseEntries {
		if !parseEntry.IsDir() || !parsePattern.MatchString(parseEntry.Name()) {
			continue
		}
		parseLegacyName := parseEntry.Name()
		if filepath.ToSlash(exampleCatalogRouteAliasPaths[parseLegacyName]) == parseExpectedRoute {
			return parseLegacyName, true, nil
		}
		if parseParts := strings.SplitN(parseLegacyName, "-", 2); len(parseParts) == 2 && parseParts[1] == parseSlug {
			return parseLegacyName, true, nil
		}
	}
	return "", false, nil
}

// resolveExampleAssetAliasPath maps grouped asset routes onto flat numbered example directories when the current filesystem still uses legacy flat paths.
func (parseL launcher) resolveExampleAssetAliasPath(parseRoutePath string) (string, bool, error) {
	parseManifest, isParseLoaded, parseErr := parseL.loadExampleCatalogManifest()
	if parseErr != nil {
		return "", false, parseErr
	}
	if isParseLoaded && len(parseManifest.Examples) > 0 {
		return "", false, nil
	}
	parseTrimmed := strings.Trim(strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(parseRoutePath)), "/examples/"), "/")
	if parseTrimmed == "" {
		return "", false, nil
	}
	parseParts := strings.Split(parseTrimmed, "/")
	if len(parseParts) < 3 {
		return "", false, nil
	}
	parseGroup := strings.TrimSpace(parseParts[0])
	if parseGroup != "public" && parseGroup != "server" && parseGroup != "testing" {
		return "", false, nil
	}
	parseLegacyName, parseOk, parseErr2 := parseL.resolveExampleLegacyNameFromGroupedRoute(parseGroup, strings.TrimSpace(parseParts[1]))
	if parseErr2 != nil || !parseOk {
		return "", parseOk, parseErr2
	}
	return "/examples/" + parseLegacyName + "/" + strings.Join(parseParts[2:], "/"), true, nil
}

// resolveExampleRouteInfo maps one examples request path onto one legacy catalog name plus its canonical route.
func (parseL launcher) resolveExampleRouteInfo(parseRoutePath string) (exampleRouteInfo, bool, error) {
	parseTrimmed := strings.Trim(strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(parseRoutePath)), "/examples/"), "/")
	if parseTrimmed == "" {
		return exampleRouteInfo{}, false, nil
	}
	parseParts := strings.Split(parseTrimmed, "/")

	parseManifest, isParseLoaded, parseErr := parseL.loadExampleCatalogManifest()
	if parseErr != nil {
		return exampleRouteInfo{}, false, parseErr
	}
	if isParseLoaded {
		for _, parseManifestEntry := range parseManifest.Examples {
			parseRelativeDir, parseOk, parseErr2 := parseL.resolveExampleCurrentDir(parseManifestEntry.Name)
			if parseErr2 != nil {
				return exampleRouteInfo{}, false, parseErr2
			}
			if !parseOk {
				continue
			}
			parseRouteRelative := strings.Trim(parseRelativeDir, "/")
			parseHTMLRoute := parseRouteRelative
			if strings.TrimSpace(parseManifestEntry.HTMLFile) != "" {
				parseHTMLRoute += "/" + parseManifestEntry.HTMLFile
			}
			switch parseTrimmed {
			case parseRouteRelative, parseHTMLRoute, parseManifestEntry.Name, parseManifestEntry.Name + "/" + parseManifestEntry.HTMLFile:
				return exampleRouteInfo{Name: parseManifestEntry.Name, RoutePath: "/examples/" + parseRouteRelative + "/"}, true, nil
			}
		}
	}

	if len(parseParts) == 1 || (len(parseParts) == 2 && strings.HasSuffix(strings.ToLower(parseParts[1]), ".html")) {
		parseLegacyName := strings.TrimSpace(parseParts[0])
		if parseLegacyName != "" {
			parseEntry, parseOk, parseErr2 := parseL.resolveExampleCatalogEntry(parseLegacyName)
			if parseErr2 != nil {
				return exampleRouteInfo{}, false, parseErr2
			}
			if parseOk {
				return exampleRouteInfo{Name: parseEntry.Name, RoutePath: "/examples/" + parseLegacyName + "/"}, true, nil
			}
		}
	}

	if len(parseParts) == 2 || (len(parseParts) == 3 && strings.HasSuffix(strings.ToLower(parseParts[2]), ".html")) {
		parseGroup := strings.TrimSpace(parseParts[0])
		if parseGroup == "public" || parseGroup == "server" || parseGroup == "testing" {
			parseInfo, parseErr2 := os.Stat(parseL.examplesDir)
			if parseErr2 != nil {
				if os.IsNotExist(parseErr2) {
					return exampleRouteInfo{}, false, parseErr2
				}
				return exampleRouteInfo{}, false, parseErr2
			}
			if !parseInfo.IsDir() {
				return exampleRouteInfo{}, false, nil
			}
			parseSlug := strings.TrimSpace(parseParts[1])
			parseLegacyName, parseOk, parseErr3 := parseL.resolveExampleLegacyNameFromGroupedRoute(parseGroup, parseSlug)
			if parseErr3 != nil {
				return exampleRouteInfo{}, false, parseErr3
			}
			if parseOk {
				return exampleRouteInfo{Name: parseLegacyName, RoutePath: "/examples/" + parseGroup + "/" + parseSlug + "/"}, true, nil
			}
		}
	}
	return exampleRouteInfo{}, false, nil
}

func (parseL launcher) resolveGeneratedExamplePage(parseRoutePath string) (generatedExamplePage, bool, error) {
	parseRouteInfo, parseOk, parseErr := parseL.resolveExampleRouteInfo(parseRoutePath)
	if parseErr != nil || !parseOk {
		return generatedExamplePage{}, parseOk, parseErr
	}
	parseNormalizedPath := filepath.ToSlash(strings.TrimSpace(parseRoutePath))
	if parseNormalizedPath != parseRouteInfo.RoutePath {
		return generatedExamplePage{}, false, nil
	}
	parseEntry, parseOk, parseErr := parseL.resolveExampleCatalogEntry(parseRouteInfo.Name)
	if parseErr != nil || !parseOk {
		return generatedExamplePage{}, parseOk, parseErr
	}
	if !parseEntry.UsesWasm {
		return generatedExamplePage{}, false, nil
	}
	parseManifestHref := ""
	parseRelativeDir, parseOk2, parseErr2 := parseL.resolveExampleCurrentDir(parseEntry.Name)
	if parseErr2 != nil {
		return generatedExamplePage{}, false, parseErr2
	}
	if parseOk2 && fileExists(filepath.Join(parseL.examplesDir, filepath.FromSlash(parseRelativeDir), "manifest.webmanifest")) {
		parseManifestHref = "./manifest.webmanifest"
	}

	return generatedExamplePage{
		RoutePath:     parseRouteInfo.RoutePath,
		DirName:       parseEntry.Name,
		HTMLFile:      parseEntry.HTMLFile,
		Title:         firstNonEmpty(parseEntry.Title, defaultExampleTitle(parseEntry.Name, parseEntry.HTMLFile)),
		WasmBinary:    parseEntry.WasmBinary,
		ManifestHref:  parseManifestHref,
		Description:   fmt.Sprintf("Generated wasm host page for %s. The Go examples server sends the compiled wasm bundle and a #app mount container to the browser.", parseEntry.Name),
		GeneratedFrom: parseEntry.HTMLFile,
	}, true, nil
}

func detectExampleWasmBinary(parseHtmlPath string) (string, bool, error) {
	parseContent, parseErr := os.ReadFile(parseHtmlPath)
	if parseErr != nil {
		return "", false, parseErr
	}
	parseMatches := regexp.MustCompile(`static/bin/([A-Za-z0-9._-]+\.wasm)`).FindSubmatch(parseContent)
	if len(parseMatches) < 2 {
		return "", false, nil
	}
	return string(parseMatches[1]), true, nil
}

func detectAvailableExampleWasmBinary(parseWasmDir string, parseHtmlPath string) (string, bool, error) {
	parseWasmBinary, parseUsesWasm, parseErr := detectExampleWasmBinary(parseHtmlPath)
	if parseErr != nil || !parseUsesWasm {
		return parseWasmBinary, parseUsesWasm, parseErr
	}
	parseWasmBinary = strings.TrimSpace(parseWasmBinary)
	if parseWasmBinary == "" {
		return "", false, nil
	}
	if !fileExists(filepath.Join(parseWasmDir, parseWasmBinary)) {
		return "", false, nil
	}
	return parseWasmBinary, true, nil
}

func detectHTMLTitle(parseHtmlPath string) (string, error) {
	parseContent, parseErr := os.ReadFile(parseHtmlPath)
	if parseErr != nil {
		return "", parseErr
	}
	parseMatches := regexp.MustCompile(`(?is)<title>(.*?)</title>`).FindSubmatch(parseContent)
	if len(parseMatches) < 2 {
		return "", nil
	}
	parseTitle := strings.TrimSpace(string(parseMatches[1]))
	parseTitle = strings.ReplaceAll(parseTitle, "\n", " ")
	parseTitle = strings.Join(strings.Fields(parseTitle), " ")
	return parseTitle, nil
}

func defaultExampleTitle(parseDirName string, parseHtmlFile string) string {
	parseBase := strings.TrimSuffix(parseHtmlFile, filepath.Ext(parseHtmlFile))
	if parseBase == "" {
		parseBase = parseDirName
	}
	parseBase = strings.ReplaceAll(parseBase, "-", " ")
	parseBase = strings.ReplaceAll(parseBase, "_", " ")
	parseParts := strings.Fields(parseBase)
	for parseIndex, parsePart := range parseParts {
		parseParts[parseIndex] = strings.ToUpper(parsePart[:1]) + parsePart[1:]
	}
	if len(parseParts) == 0 {
		return parseDirName
	}
	return strings.Join(parseParts, " ") + " - GoWebComponents"
}

func exampleCatalogTags(parseDirName string, parseWasmBinary string, isUsesWasm bool) []string {
	parseTrimmedName := parseDirName
	if parseParts := strings.SplitN(parseDirName, "-", 2); len(parseParts) == 2 {
		parseTrimmedName = parseParts[1]
	}
	parseNameLower := strings.ToLower(parseTrimmedName)

	parseSeen := map[string]struct{}{}
	parseTags := make([]string, 0, 8)
	parseAddTag := func(parseTag string) {
		parseTag = strings.TrimSpace(strings.ToLower(parseTag))
		if parseTag == "" {
			return
		}
		if _, parseOk := parseSeen[parseTag]; parseOk {
			return
		}
		parseSeen[parseTag] = struct{}{}
		parseTags = append(parseTags, parseTag)
	}

	if isUsesWasm {
		parseAddTag("wasm")
	}
	if strings.TrimSpace(parseWasmBinary) != "" {
		parseAddTag(strings.TrimSuffix(strings.ToLower(parseWasmBinary), ".wasm"))
	}

	for _, parseToken := range strings.FieldsFunc(strings.ToLower(parseTrimmedName), func(parseR rune) bool {
		return parseR == '-' || parseR == '_' || parseR == ' '
	}) {
		parseAddTag(parseToken)
	}

	parseAddFrameworkTags := func() {
		switch {
		case strings.HasPrefix(parseDirName, "21-") || strings.HasPrefix(parseDirName, "22-") || strings.HasPrefix(parseDirName, "23-") || strings.HasPrefix(parseDirName, "24-") || strings.HasPrefix(parseDirName, "25-") || strings.HasPrefix(parseDirName, "26-") || strings.HasPrefix(parseDirName, "27-") || strings.HasPrefix(parseDirName, "28-") || strings.HasPrefix(parseDirName, "29-") || strings.HasPrefix(parseDirName, "30-") || strings.HasPrefix(parseDirName, "31-") || strings.HasPrefix(parseDirName, "32-") || strings.HasPrefix(parseDirName, "33-") || strings.HasPrefix(parseDirName, "34-") || strings.HasPrefix(parseDirName, "35-") || strings.HasPrefix(parseDirName, "36-") || strings.HasPrefix(parseDirName, "46-") || strings.HasPrefix(parseDirName, "47-") || strings.HasPrefix(parseDirName, "48-") || strings.HasPrefix(parseDirName, "49-") || strings.HasPrefix(parseDirName, "50-") || strings.HasPrefix(parseDirName, "51-") || strings.HasPrefix(parseDirName, "75-") || strings.HasPrefix(parseDirName, "76-") || strings.HasPrefix(parseDirName, "77-") || strings.HasPrefix(parseDirName, "78-") || strings.HasPrefix(parseDirName, "79-") || strings.HasPrefix(parseDirName, "80-") || strings.HasPrefix(parseDirName, "81-") || strings.HasPrefix(parseDirName, "82-"):
			parseAddTag("ui")
		case strings.HasPrefix(parseDirName, "01-") || strings.HasPrefix(parseDirName, "02-") || strings.HasPrefix(parseDirName, "03-") || strings.HasPrefix(parseDirName, "04-") || strings.HasPrefix(parseDirName, "05-") || strings.HasPrefix(parseDirName, "06-") || strings.HasPrefix(parseDirName, "07-") || strings.HasPrefix(parseDirName, "08-") || strings.HasPrefix(parseDirName, "09-") || strings.HasPrefix(parseDirName, "10-") || strings.HasPrefix(parseDirName, "11-") || strings.HasPrefix(parseDirName, "12-") || strings.HasPrefix(parseDirName, "13-") || strings.HasPrefix(parseDirName, "14-") || strings.HasPrefix(parseDirName, "15-") || strings.HasPrefix(parseDirName, "16-") || strings.HasPrefix(parseDirName, "17-") || strings.HasPrefix(parseDirName, "18-") || strings.HasPrefix(parseDirName, "19-") || strings.HasPrefix(parseDirName, "20-"):
			parseAddTag("ui")
		}

		switch {
		case strings.HasPrefix(parseDirName, "37-") || strings.HasPrefix(parseDirName, "38-") || strings.HasPrefix(parseDirName, "39-") || strings.HasPrefix(parseDirName, "40-") || strings.HasPrefix(parseDirName, "41-"):
			parseAddTag("state")
		}

		switch {
		case strings.HasPrefix(parseDirName, "42-") || strings.HasPrefix(parseDirName, "43-") || strings.HasPrefix(parseDirName, "44-") || strings.HasPrefix(parseDirName, "45-") || strings.HasPrefix(parseDirName, "93-"):
			parseAddTag("fetch")
		}

		switch {
		case strings.HasPrefix(parseDirName, "52-") || strings.HasPrefix(parseDirName, "53-") || strings.HasPrefix(parseDirName, "54-") || strings.HasPrefix(parseDirName, "88-") || strings.HasPrefix(parseDirName, "89-"):
			parseAddTag("html")
		}

		switch {
		case strings.HasPrefix(parseDirName, "55-") || strings.HasPrefix(parseDirName, "56-") || strings.HasPrefix(parseDirName, "57-") || strings.HasPrefix(parseDirName, "58-") || strings.HasPrefix(parseDirName, "59-") || strings.HasPrefix(parseDirName, "60-") || strings.HasPrefix(parseDirName, "61-") || strings.HasPrefix(parseDirName, "62-") || strings.HasPrefix(parseDirName, "63-") || strings.HasPrefix(parseDirName, "64-") || strings.HasPrefix(parseDirName, "65-") || strings.HasPrefix(parseDirName, "92-") || strings.HasPrefix(parseDirName, "96-"):
			parseAddTag("router")
		}

		switch {
		case strings.HasPrefix(parseDirName, "66-") || strings.HasPrefix(parseDirName, "67-") || strings.HasPrefix(parseDirName, "68-") || strings.HasPrefix(parseDirName, "69-") || strings.HasPrefix(parseDirName, "98-"):
			parseAddTag("devtools")
		}

		switch {
		case strings.HasPrefix(parseDirName, "70-") || strings.HasPrefix(parseDirName, "71-") || strings.HasPrefix(parseDirName, "72-") || strings.HasPrefix(parseDirName, "73-") || strings.HasPrefix(parseDirName, "74-") || strings.HasPrefix(parseDirName, "84-") || strings.HasPrefix(parseDirName, "87-"):
			parseAddTag("ssr")
		}

		switch {
		case strings.HasPrefix(parseDirName, "71-") || strings.HasPrefix(parseDirName, "72-") || strings.HasPrefix(parseDirName, "74-") || strings.HasPrefix(parseDirName, "84-"):
			parseAddTag("hydration")
		}

		switch {
		case strings.HasPrefix(parseDirName, "83-") || strings.HasPrefix(parseDirName, "84-") || strings.HasPrefix(parseDirName, "85-"):
			parseAddTag("i18n")
		}

		switch {
		case strings.HasPrefix(parseDirName, "90-") || strings.HasPrefix(parseDirName, "91-") || strings.HasPrefix(parseDirName, "94-") || strings.HasPrefix(parseDirName, "95-") || strings.Contains(parseNameLower, "multi-client") || strings.Contains(parseNameLower, "cross-tab") || strings.Contains(parseNameLower, "multi-window"):
			parseAddTag("interop")
		}

		if strings.Contains(parseNameLower, "pwa") {
			parseAddTag("pwa")
		}
		if strings.Contains(parseNameLower, "offline") {
			parseAddTag("offline")
		}
		if strings.Contains(parseNameLower, "form") {
			parseAddTag("forms")
		}
		if strings.Contains(parseNameLower, "worker") {
			parseAddTag("workers")
		}
		if strings.Contains(parseNameLower, "overlay") || strings.Contains(parseNameLower, "portal") {
			parseAddTag("overlays")
		}
		if strings.Contains(parseNameLower, "accessible") || strings.Contains(parseNameLower, "accessibility") {
			parseAddTag("accessibility")
		}
		if strings.Contains(parseNameLower, "web-components") || strings.Contains(parseNameLower, "custom-element") || strings.Contains(parseNameLower, "custom-elements") {
			parseAddTag("custom-elements")
		}
		if strings.Contains(parseNameLower, "transition") || strings.Contains(parseNameLower, "deferred") || strings.Contains(parseNameLower, "debounced") || strings.Contains(parseNameLower, "throttled") || strings.Contains(parseNameLower, "goroutines") {
			parseAddTag("scheduling")
		}
		if strings.Contains(parseNameLower, "code-splitting") {
			parseAddTag("code-splitting")
		}
	}

	parseAddFrameworkTags()

	parseJoined := parseNameLower
	if strings.Contains(parseJoined, "multi-client") {
		parseAddTag("multi-client")
	}
	if strings.Contains(parseJoined, "cross-tab") {
		parseAddTag("cross-tab")
	}
	if strings.Contains(parseJoined, "multi-window") {
		parseAddTag("multi-window")
	}
	if strings.Contains(parseJoined, "pwa") {
		parseAddTag("pwa")
	}
	if strings.Contains(parseJoined, "ssr") {
		parseAddTag("ssr")
	}
	if strings.Contains(parseJoined, "router") {
		parseAddTag("router")
	}
	if strings.Contains(parseJoined, "state") {
		parseAddTag("state")
	}
	if strings.Contains(parseJoined, "fetch") {
		parseAddTag("fetch")
	}
	if strings.Contains(parseJoined, "devtools") {
		parseAddTag("devtools")
	}

	return parseTags
}

func hasAnyTag(parseTags []string, parseExpected ...string) bool {
	for _, parseTag := range parseTags {
		for _, parseCandidate := range parseExpected {
			if parseTag == parseCandidate {
				return true
			}
		}
	}
	return false
}

func renderExamplesAppShellHTML(parseRoutePath string, parseCatalogHref string) string {
	parseRoutePath = strings.TrimSpace(parseRoutePath)
	if parseRoutePath == "" {
		parseRoutePath = "/examples/"
	}
	parseCatalogHref = strings.TrimSpace(parseCatalogHref)
	if parseCatalogHref == "" {
		parseCatalogHref = "/examples/"
	}
	return renderExamplesShellHTML(examplesShellDocument{
		Title:             "GoWebComponents Examples",
		Description:       "GoWebComponents examples catalog powered by a Go server and a wasm-first, multi-client catalog app.",
		BodyClass:         "example-shell bg-[#08111d] text-white min-h-screen",
		RoutePath:         parseRoutePath,
		CatalogHref:       parseCatalogHref,
		WasmURL:           "/static/bin/gwc-examples-site.wasm",
		FailureTitle:      "GoWebComponents Examples",
		FailureMessage:    "The wasm catalog failed to start.",
		FailureHref:       "/examples/list",
		FailureLinkLabel:  "Open raw examples listing",
		NoScriptMessage:   "The primary catalog now boots from a Go/wasm app. Enable JavaScript to browse the interactive multi-client catalog, or use the raw listing below.",
		NoScriptHref:      "/examples/list",
		NoScriptLinkLabel: "Open raw examples listing",
	})
}

func renderGeneratedExampleHTML(parsePage generatedExamplePage) string {
	return renderExamplesShellHTML(examplesShellDocument{
		Title:            parsePage.Title,
		Description:      parsePage.Description,
		BodyClass:        "example-shell bg-[#08111d] text-white min-h-screen",
		BodyData:         map[string]string{"gwc-example": parsePage.DirName, "gwc-entry": parsePage.HTMLFile},
		RoutePath:        parsePage.RoutePath,
		ExampleSlug:      parsePage.DirName,
		ManifestHref:     parsePage.ManifestHref,
		WasmURL:          "/static/bin/" + parsePage.WasmBinary,
		FailureTitle:     parsePage.Title,
		FailureMessage:   "Failed to start the example wasm bundle.",
		FailureHref:      "/examples/static/index.html",
		FailureLinkLabel: "Back to examples catalog",
	})
}

type examplesShellDocument struct {
	Title             string
	Description       string
	BodyClass         string
	BodyData          map[string]string
	RoutePath         string
	CatalogHref       string
	ExampleSlug       string
	ManifestHref      string
	WasmURL           string
	FailureTitle      string
	FailureMessage    string
	FailureHref       string
	FailureLinkLabel  string
	NoScriptMessage   string
	NoScriptHref      string
	NoScriptLinkLabel string
}

func renderExamplesShellHTML(parseDocument examplesShellDocument) string {
	parseBootstrapScript := renderExamplesBootstrapDataScript(parseDocument)
	parseLoaderScript := renderExamplesLoaderScriptTag(parseDocument.WasmURL, parseDocument.FailureTitle, parseDocument.FailureMessage, parseDocument.FailureHref, parseDocument.FailureLinkLabel)
	parseHeadChildren := []ui.Node{
		gwchtml.Meta(gwchtml.Props{Raw: map[string]interface{}{"charset": "utf-8"}}),
		gwchtml.Meta(gwchtml.Props{Raw: map[string]interface{}{"name": "viewport", "content": "width=device-width, initial-scale=1"}}),
		gwchtml.Meta(gwchtml.Props{Raw: map[string]interface{}{"name": "description", "content": parseDocument.Description}}),
		gwchtml.Tag("title", gwchtml.Props{}, gwchtml.Text(parseDocument.Title)),
		gwchtml.Link(gwchtml.Props{Rel: "stylesheet", Href: "/static/css/tailwind.css"}),
		gwchtml.Link(gwchtml.Props{Rel: "stylesheet", Href: "/static/css/example-shell.css"}),
		gwchtml.Script(gwchtml.Props{Src: "/static/script/wasm_exec.js"}),
		gwchtml.Script(gwchtml.Props{Src: "/static/script/example-logger.js"}),
	}
	if strings.TrimSpace(parseDocument.ManifestHref) != "" {
		parseHeadChildren = append(parseHeadChildren, gwchtml.Link(gwchtml.Props{Rel: "manifest", Href: parseDocument.ManifestHref}))
	}

	parseBodyChildren := []ui.Node{gwchtml.Div(gwchtml.Props{ID: "app"})}
	if strings.TrimSpace(parseDocument.NoScriptMessage) != "" {
		parseBodyChildren = append(parseBodyChildren,
			gwchtml.NoScript(gwchtml.Props{},
				gwchtml.Main(gwchtml.Props{Style: map[string]string{"max-width": "72rem", "margin": "0 auto", "padding": "2rem", "font-family": "'Segoe UI Variable', 'Segoe UI', sans-serif"}},
					gwchtml.H1(gwchtml.Props{}, gwchtml.Text(parseDocument.FailureTitle)),
					gwchtml.P(gwchtml.Props{}, gwchtml.Text(parseDocument.NoScriptMessage)),
					gwchtml.P(gwchtml.Props{}, gwchtml.A(gwchtml.Props{Href: parseDocument.NoScriptHref}, gwchtml.Text(parseDocument.NoScriptLinkLabel))),
				),
			),
		)
	}

	parseBodyProps := gwchtml.Props{Class: parseDocument.BodyClass, Data: parseDocument.BodyData}
	parseMarkup, parseErr := renderExamplesToString(gwchtml.Html(gwchtml.Props{Raw: map[string]interface{}{"lang": "en"}},
		gwchtml.Head(gwchtml.Props{}, parseHeadChildren...),
		gwchtml.Body(parseBodyProps, parseBodyChildren...),
	))
	if parseErr != nil {
		return renderExamplesShellHTMLFallback(parseDocument, parseBootstrapScript)
	}
	if parseBootstrapScript != "" {
		parseMarkup = strings.Replace(parseMarkup, `<div id="app"></div>`, `<div id="app"></div>`+parseBootstrapScript, 1)
	}
	if parseLoaderScript != "" {
		parseMarkup = strings.Replace(parseMarkup, `</body>`, parseLoaderScript+`</body>`, 1)
	}
	return "<!doctype html>\n" + parseMarkup
}

func renderExamplesBootstrapDataScript(parseDocument examplesShellDocument) string {
	parseBootstrap := ui.SSRBootstrap{
		Route: ui.SSRRouteBootstrap{Path: parseDocument.RoutePath},
		Data: map[string]interface{}{
			"examples": map[string]interface{}{
				"mode":        "server",
				"catalogURL":  "/examples/catalog.json",
				"assetBase":   "/static/",
				"wasmBase":    "/static/bin/",
				"catalogHref": parseDocument.CatalogHref,
				"slug":        parseDocument.ExampleSlug,
			},
		},
	}
	parseScript, parseErr := renderExamplesUIBootstrapScript(parseBootstrap, "")
	if parseErr != nil {
		return ""
	}
	return parseScript
}

func renderExamplesBootstrapScript(parseWasmURL string, parseFailureTitle string, parseFailureMessage string, parseFailureHref string, parseFailureLinkLabel string) string {
	return "const GWC_EXAMPLES_CACHE = 'gwc-examples-runtime-v1';\n" +
		"async function instantiateCachedWasm(response, importObject) {\n" +
		"  const contentType = (response.headers.get('content-type') || '').toLowerCase();\n" +
		"  if (typeof WebAssembly.instantiateStreaming === 'function' && contentType.includes('application/wasm')) {\n" +
		"    return WebAssembly.instantiateStreaming(Promise.resolve(response), importObject);\n" +
		"  }\n" +
		"  const bytes = await response.arrayBuffer();\n" +
		"  return WebAssembly.instantiate(bytes, importObject);\n" +
		"}\n" +
		"async function loadCachedWasm(url, importObject) {\n" +
		"  if (!('caches' in globalThis)) {\n" +
		"    const response = await fetch(url, { cache: 'no-store' });\n" +
		"    if (!response.ok) {\n" +
		"      throw new Error('Failed to fetch wasm: ' + response.status + ' ' + response.statusText);\n" +
		"    }\n" +
		"    console.info('[gwc examples] wasm source: network (no Cache Storage)', url);\n" +
		"    return instantiateCachedWasm(response, importObject);\n" +
		"  }\n" +
		"  const cache = await caches.open(GWC_EXAMPLES_CACHE);\n" +
		"  let response = await cache.match(url);\n" +
		"  let source = 'Cache Storage';\n" +
		"  if (!response) {\n" +
		"    response = await fetch(url, { cache: 'no-store' });\n" +
		"    if (!response.ok) {\n" +
		"      throw new Error('Failed to fetch wasm: ' + response.status + ' ' + response.statusText);\n" +
		"    }\n" +
		"    await cache.put(url, response.clone());\n" +
		"    source = 'network';\n" +
		"  }\n" +
		"  console.info('[gwc examples] wasm source: ' + source, url);\n" +
		"  return instantiateCachedWasm(response, importObject);\n" +
		"}\n" +
		"const go = new Go();\n" +
		"loadCachedWasm(" + jsStringLiteral(parseWasmURL) + ", go.importObject)\n" +
		"  .then(result => go.run(result.instance))\n" +
		"  .catch(error => {\n" +
		"    const root = document.getElementById('app');\n" +
		"    if (root) {\n" +
		"      root.innerHTML = '<main style=\"max-width:72rem;margin:0 auto;padding:2rem;font-family:\\'Segoe UI Variable\\',\\'Segoe UI\\',sans-serif;\"><h1>" + escapeHTML(jsSingleQuoted(parseFailureTitle)) + "</h1><p>" + escapeHTML(jsSingleQuoted(parseFailureMessage)) + "</p><p><a href=\"" + escapeHTML(parseFailureHref) + "\">" + escapeHTML(jsSingleQuoted(parseFailureLinkLabel)) + "</a></p></main>';\n" +
		"    }\n" +
		"    console.error(error);\n" +
		"  });"
}

func renderExamplesLoaderScriptTag(parseWasmURL string, parseFailureTitle string, parseFailureMessage string, parseFailureHref string, parseFailureLinkLabel string) string {
	parseScriptBody := renderExamplesBootstrapScriptFunc(parseWasmURL, parseFailureTitle, parseFailureMessage, parseFailureHref, parseFailureLinkLabel)
	if strings.TrimSpace(parseScriptBody) == "" {
		return ""
	}
	return `<script>` + parseScriptBody + `</script>`
}

func renderExamplesShellHTMLFallback(parseDocument examplesShellDocument, parseBootstrapScript string) string {
	parseManifestLink := ""
	if strings.TrimSpace(parseDocument.ManifestHref) != "" {
		parseManifestLink = "\n  <link rel=\"manifest\" href=\"" + escapeHTML(parseDocument.ManifestHref) + "\">"
	}
	parseNoscript := ""
	if strings.TrimSpace(parseDocument.NoScriptMessage) != "" {
		parseNoscript = "\n  <noscript><main style=\"max-width:72rem;margin:0 auto;padding:2rem;font-family:'Segoe UI Variable','Segoe UI',sans-serif;\"><h1>" + escapeHTML(parseDocument.FailureTitle) + "</h1><p>" + escapeHTML(parseDocument.NoScriptMessage) + "</p><p><a href=\"" + escapeHTML(parseDocument.NoScriptHref) + "\">" + escapeHTML(parseDocument.NoScriptLinkLabel) + "</a></p></main></noscript>"
	}
	parseBodyAttrs := " class=\"" + escapeHTML(parseDocument.BodyClass) + "\""
	for parseKey, parseValue := range parseDocument.BodyData {
		parseBodyAttrs += " data-" + escapeHTML(parseKey) + "=\"" + escapeHTML(parseValue) + "\""
	}
	return "<!doctype html>\n<html lang=\"en\">\n<head>\n  <meta charset=\"utf-8\">\n  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n  <meta name=\"description\" content=\"" + escapeHTML(parseDocument.Description) + "\">" + parseManifestLink + "\n  <title>" + escapeHTML(parseDocument.Title) + "</title>\n  <link rel=\"stylesheet\" href=\"/static/css/tailwind.css\">\n  <link rel=\"stylesheet\" href=\"/static/css/example-shell.css\">\n  <script src=\"/static/script/wasm_exec.js\"></script>\n  <script src=\"/static/script/example-logger.js\"></script>\n</head>\n<body" + parseBodyAttrs + ">\n  <div id=\"app\"></div>" + parseNoscript + parseBootstrapScript + "\n  <script>\n" + renderExamplesBootstrapScript(parseDocument.WasmURL, parseDocument.FailureTitle, parseDocument.FailureMessage, parseDocument.FailureHref, parseDocument.FailureLinkLabel) + "\n  </script>\n</body>\n</html>"
}

func jsStringLiteral(parseValue string) string {
	return "'" + jsSingleQuoted(parseValue) + "'"
}

func jsSingleQuoted(parseText string) string {
	parseText = strings.ReplaceAll(parseText, `\`, `\\`)
	parseText = strings.ReplaceAll(parseText, `'`, `\'`)
	return parseText
}

func renderExamplesListingHTML(parseLinks []exampleLink, parseQuery string) string {
	var parseItems strings.Builder
	for _, parseLink := range parseLinks {
		parseItems.WriteString(`<li><a href="` + parseLink.Href + `">` + parseLink.Name + `</a></li>`)
	}
	if parseItems.Len() == 0 {
		parseItems.WriteString(`<li>No examples matched this search yet.</li>`)
	}
	parseMetaText := "Generated from example folders under /examples."
	if strings.TrimSpace(parseQuery) != "" {
		parseMetaText = fmt.Sprintf("Filtered examples for %q.", parseQuery)
	}
	return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>GoWebComponents Examples</title>
  <style>
    body { font-family: Segoe UI, Arial, sans-serif; margin: 2rem; line-height: 1.5; }
    h1 { margin-top: 0; }
    ul { padding-left: 1.2rem; }
    li { margin: 0.35rem 0; }
    a { color: #0b63ce; text-decoration: none; }
    a:hover { text-decoration: underline; }
    .meta { margin-bottom: 1rem; color: #555; }
		.search { display: flex; gap: 0.5rem; margin: 1rem 0 1.25rem; flex-wrap: wrap; }
		.search input { min-width: 18rem; max-width: 28rem; padding: 0.55rem 0.7rem; font: inherit; }
		.search button { padding: 0.55rem 0.85rem; font: inherit; cursor: pointer; }
  </style>
</head>
<body>
  <h1>GoWebComponents Examples</h1>
	<p class="meta">` + parseMetaText + `</p>
  <p><a href="/examples/static/index.html">Open styled showcase page</a></p>
	<form class="search" method="get" action="/examples/list">
		<input type="search" name="q" value="` + escapeHTML(parseQuery) + `" placeholder="Search examples by keyword">
		<button type="submit">Filter</button>
	</form>
  <ul>` + parseItems.String() + `</ul>
</body>
</html>`
}

func escapeHTML(parseText string) string {
	parseReplacer := strings.NewReplacer(
		"&", "&amp;",
		`"`, "&quot;",
		"<", "&lt;",
		">", "&gt;",
	)
	return parseReplacer.Replace(parseText)
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (parseW *statusWriter) WriteHeader(parseStatus int) {
	parseW.status = parseStatus
	parseW.ResponseWriter.WriteHeader(parseStatus)
}

func (parseW *statusWriter) Write(parseData []byte) (int, error) {
	if parseW.status == 0 {
		parseW.status = http.StatusOK
	}
	return parseW.ResponseWriter.Write(parseData)
}

var _ http.ResponseWriter = (*statusWriter)(nil)
