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

var examplesListen = net.Listen

var examplesServe = func(server *http.Server, listener net.Listener) error {
	return server.Serve(listener)
}

var renderExamplesUIBootstrapScript = ui.RenderBootstrapScript

var renderExamplesBootstrapScriptFunc = renderExamplesBootstrapScript

var examplesCatalogMarshalIndent = json.MarshalIndent

var renderExamplesToString = ui.RenderToString

func (l launcher) runExamples(args []string) error {
	fs := flag.NewFlagSet("examples", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	host := fs.String("host", defaultHost, "Host to bind")
	port := fs.String("port", defaultPort, "Port to bind")
	exportStaticCatalog := fs.String("export-static-catalog", "", "Write a static examples catalog JSON file for static hosting and exit")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if _, err := os.Stat(l.examplesDir); err != nil {
		return fmt.Errorf("examples directory not found: %w", err)
	}

	if strings.TrimSpace(*exportStaticCatalog) != "" {
		if err := l.writeStaticExamplesCatalogFile(*exportStaticCatalog); err != nil {
			return err
		}
		fmt.Printf("Wrote static examples catalog to %s\n", *exportStaticCatalog)
		return nil
	}

	addr := joinHostPort(*host, *port)
	listener, err := examplesListen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	server := &http.Server{
		Addr:    addr,
		Handler: l.newExamplesHandler(*host, *port),
	}

	fmt.Printf("GWC examples server listening on http://%s\n", addr)
	fmt.Printf("Examples: http://%s\n", addr)
	fmt.Printf("Counter:  http://%s/examples/01-counter/\n", addr)

	if err := examplesServe(server, listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (l launcher) newExamplesHandler(host string, port string) http.Handler {
	examplesServer := http.StripPrefix("/examples/", http.FileServer(http.Dir(l.examplesDir)))
	staticServer := http.StripPrefix("/static/", http.FileServer(http.Dir(l.staticDir)))
	wasmServer := http.StripPrefix("/static/bin/", http.FileServer(http.Dir(l.resolvedExamplesWasmDir())))

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"ok":      true,
			"service": "gowebcomponents-gwc-examples",
			"time":    time.Now().UTC().Format(time.RFC3339),
			"root":    l.repoRoot,
			"host":    host,
			"port":    port,
		})
	})
	mux.HandleFunc("/examples/list", func(w http.ResponseWriter, r *http.Request) {
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		links, err := l.buildExamplesListing()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "examples_listing_failed",
				"hint":  err.Error(),
			})
			return
		}
		links = filterExampleLinks(links, query)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(renderExamplesListingHTML(links, query)))
	})
	mux.HandleFunc("/examples/catalog.json", func(w http.ResponseWriter, r *http.Request) {
		catalog, err := l.buildExamplesCatalog()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "examples_catalog_failed",
				"hint":  err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, catalog)
	})
	mux.HandleFunc("/examples", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/examples" {
			examplesServer.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(renderExamplesAppShellHTML("/examples/", "/examples/")))
	})
	mux.HandleFunc("/examples/static/index.html", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/examples/static/index.html" {
			examplesServer.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(renderExamplesAppShellHTML("/examples/", "/examples/")))
	})
	mux.HandleFunc("/examples/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/examples/" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(renderExamplesAppShellHTML("/examples/", "/examples/")))
			return
		}
		trimmed := strings.TrimPrefix(filepath.ToSlash(r.URL.Path), "/examples/")
		trimmed = strings.TrimSpace(trimmed)
		if trimmed == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(renderExamplesAppShellHTML("/examples/", "/examples/")))
			return
		}
		if strings.HasSuffix(strings.ToLower(r.URL.Path), ".html") {
			dirName := strings.TrimSpace(strings.Split(trimmed, "/")[0])
			if dirName != "" {
				http.Redirect(w, r, "/examples/"+dirName+"/", http.StatusFound)
				return
			}
			examplePage, ok, err := l.resolveGeneratedExamplePage(r.URL.Path)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{
					"error": "examples_html_route_failed",
					"hint":  err.Error(),
				})
				return
			}
			if ok {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(renderGeneratedExampleHTML(examplePage)))
				return
			}
		}
		if !strings.Contains(trimmed, "/") {
			http.Redirect(w, r, r.URL.Path+"/", http.StatusFound)
			return
		}
		if strings.Count(strings.Trim(trimmed, "/"), "/") == 0 {
			examplePage, ok, err := l.resolveGeneratedExamplePage(r.URL.Path)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{
					"error": "examples_route_failed",
					"hint":  err.Error(),
				})
				return
			}
			if ok {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(renderGeneratedExampleHTML(examplePage)))
				return
			}
		}
		examplesServer.ServeHTTP(w, r)
	})
	mux.Handle("/static/bin/", wasmServer)
	mux.Handle("/static/", staticServer)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(renderExamplesAppShellHTML("/", "/")))
			return
		}
		examplesServer.ServeHTTP(w, r)
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		applyDevHeaders(wrapped, r)
		mux.ServeHTTP(wrapped, r)
		fmt.Printf("%d %s %s %dms\n", wrapped.status, r.Method, r.URL.RequestURI(), time.Since(start).Milliseconds())
	})
}

func (l launcher) buildExamplesListing() ([]exampleLink, error) {
	catalog, err := l.buildExamplesCatalog()
	if err != nil {
		return nil, err
	}
	links := make([]exampleLink, 0, len(catalog.Examples))
	for _, entry := range catalog.Examples {
		links = append(links, exampleLink{Name: entry.Name, Href: entry.Href})
	}
	return links, nil
}

func (l launcher) buildExamplesCatalog() (examplesCatalogPayload, error) {
	return l.buildExamplesCatalogWithHref(func(dirPath string, dirName string, htmlFile string) string {
		return "/examples/" + dirName + "/"
	})
}

func (l launcher) buildStaticExamplesCatalog() (examplesCatalogPayload, error) {
	return l.buildExamplesCatalogWithHref(func(dirPath string, dirName string, htmlFile string) string {
		return "../" + dirName + "/" + htmlFile
	})
}

func (l launcher) buildExamplesCatalogWithHref(resolveHref func(dirPath string, dirName string, htmlFile string) string) (examplesCatalogPayload, error) {
	entries, err := os.ReadDir(l.examplesDir)
	if err != nil {
		return examplesCatalogPayload{}, err
	}

	pattern := regexp.MustCompile(`^\d{2}-`)
	catalogEntries := make([]exampleCatalogEntry, 0)
	for _, entry := range entries {
		if !entry.IsDir() || !pattern.MatchString(entry.Name()) {
			continue
		}
		dirPath := filepath.Join(l.examplesDir, entry.Name())
		catalogEntry, ok, err := l.buildExampleCatalogEntry(dirPath, entry.Name(), resolveHref)
		if err != nil {
			return examplesCatalogPayload{}, err
		}
		if ok {
			catalogEntries = append(catalogEntries, catalogEntry)
		}
	}
	sort.Slice(catalogEntries, func(i, j int) bool { return catalogEntries[i].Name < catalogEntries[j].Name })

	wasmCount := 0
	multiClientCount := 0
	for _, entry := range catalogEntries {
		if entry.UsesWasm {
			wasmCount++
		}
		if entry.MultiClient {
			multiClientCount++
		}
	}

	return examplesCatalogPayload{
		GeneratedAt:         time.Now().UTC().Format(time.RFC3339),
		TotalExamples:       len(catalogEntries),
		WasmExamples:        wasmCount,
		MultiClientExamples: multiClientCount,
		Examples:            catalogEntries,
	}, nil
}

func (l launcher) writeStaticExamplesCatalogFile(targetPath string) error {
	catalog, err := l.buildStaticExamplesCatalog()
	if err != nil {
		return err
	}
	targetPath = strings.TrimSpace(targetPath)
	if targetPath == "" {
		return errors.New("static catalog output path is required")
	}
	if !filepath.IsAbs(targetPath) {
		resolved, err := filepath.Abs(targetPath)
		if err != nil {
			return fmt.Errorf("resolve static catalog path: %w", err)
		}
		targetPath = resolved
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("create static catalog directory: %w", err)
	}
	encoded, err := examplesCatalogMarshalIndent(catalog, "", "  ")
	if err != nil {
		return fmt.Errorf("encode static catalog: %w", err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(targetPath, encoded, 0644); err != nil {
		return fmt.Errorf("write static catalog: %w", err)
	}
	return nil
}

func filterExampleLinks(links []exampleLink, query string) []exampleLink {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return links
	}
	terms := strings.Fields(query)
	filtered := make([]exampleLink, 0, len(links))
	for _, link := range links {
		haystack := strings.ToLower(link.Name + " " + link.Href)
		matchesAll := true
		for _, term := range terms {
			if !strings.Contains(haystack, term) {
				matchesAll = false
				break
			}
		}
		if matchesAll {
			filtered = append(filtered, link)
		}
	}
	return filtered
}

func firstHTMLFileName(dirPath string) (string, bool, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "", false, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".html") {
			return entry.Name(), true, nil
		}
	}
	return "", false, nil
}

func (l launcher) buildExampleCatalogEntry(dirPath string, dirName string, resolveHref func(dirPath string, dirName string, htmlFile string) string) (exampleCatalogEntry, bool, error) {
	htmlFile, ok, err := firstHTMLFileName(dirPath)
	if err != nil || !ok {
		return exampleCatalogEntry{}, ok, err
	}
	htmlPath := filepath.Join(dirPath, htmlFile)
	wasmBinary, usesWasm, err := detectAvailableExampleWasmBinary(l.resolvedExamplesWasmDir(), htmlPath)
	if err != nil {
		return exampleCatalogEntry{}, false, err
	}
	title, _ := detectHTMLTitle(htmlPath)
	tags := exampleCatalogTags(dirName, wasmBinary, usesWasm)
	href := "/examples/" + dirName + "/"
	if resolveHref != nil {
		href = resolveHref(dirPath, dirName, htmlFile)
	}
	return exampleCatalogEntry{
		Name:        dirName,
		Href:        href,
		HTMLFile:    htmlFile,
		Title:       title,
		UsesWasm:    usesWasm,
		WasmBinary:  wasmBinary,
		MultiClient: hasAnyTag(tags, "multi-client", "cross-tab", "multi-window"),
		Tags:        tags,
	}, true, nil
}

func (l launcher) resolveExampleCatalogEntry(dirName string) (exampleCatalogEntry, bool, error) {
	dirName = strings.TrimSpace(dirName)
	if dirName == "" {
		return exampleCatalogEntry{}, false, nil
	}
	dirPath := filepath.Join(l.examplesDir, dirName)
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return exampleCatalogEntry{}, false, nil
		}
		return exampleCatalogEntry{}, false, err
	}
	if !info.IsDir() {
		return exampleCatalogEntry{}, false, nil
	}
	return l.buildExampleCatalogEntry(dirPath, dirName, func(dirPath string, dirName string, htmlFile string) string {
		return "/examples/" + dirName + "/"
	})
}

func (l launcher) resolveGeneratedExamplePage(routePath string) (generatedExamplePage, bool, error) {
	trimmed := strings.Trim(strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(routePath)), "/examples/"), "/")
	if trimmed == "" {
		return generatedExamplePage{}, false, nil
	}
	parts := strings.Split(trimmed, "/")
	dirName := strings.TrimSpace(parts[0])
	if dirName == "" {
		return generatedExamplePage{}, false, nil
	}
	entry, ok, err := l.resolveExampleCatalogEntry(dirName)
	if err != nil || !ok {
		return generatedExamplePage{}, ok, err
	}
	if !entry.UsesWasm {
		return generatedExamplePage{}, false, nil
	}
	manifestHref := ""
	if fileExists(filepath.Join(l.examplesDir, dirName, "manifest.webmanifest")) {
		manifestHref = "./manifest.webmanifest"
	}

	return generatedExamplePage{
		RoutePath:     routePath,
		DirName:       dirName,
		HTMLFile:      entry.HTMLFile,
		Title:         firstNonEmpty(entry.Title, defaultExampleTitle(dirName, entry.HTMLFile)),
		WasmBinary:    entry.WasmBinary,
		ManifestHref:  manifestHref,
		Description:   fmt.Sprintf("Generated wasm host page for %s. The Go examples server sends the compiled wasm bundle and a #app mount container to the browser.", dirName),
		GeneratedFrom: entry.HTMLFile,
	}, true, nil
}

func detectExampleWasmBinary(htmlPath string) (string, bool, error) {
	content, err := os.ReadFile(htmlPath)
	if err != nil {
		return "", false, err
	}
	matches := regexp.MustCompile(`static/bin/([A-Za-z0-9._-]+\.wasm)`).FindSubmatch(content)
	if len(matches) < 2 {
		return "", false, nil
	}
	return string(matches[1]), true, nil
}

func detectAvailableExampleWasmBinary(wasmDir string, htmlPath string) (string, bool, error) {
	wasmBinary, usesWasm, err := detectExampleWasmBinary(htmlPath)
	if err != nil || !usesWasm {
		return wasmBinary, usesWasm, err
	}
	wasmBinary = strings.TrimSpace(wasmBinary)
	if wasmBinary == "" {
		return "", false, nil
	}
	if !fileExists(filepath.Join(wasmDir, wasmBinary)) {
		return "", false, nil
	}
	return wasmBinary, true, nil
}

func detectHTMLTitle(htmlPath string) (string, error) {
	content, err := os.ReadFile(htmlPath)
	if err != nil {
		return "", err
	}
	matches := regexp.MustCompile(`(?is)<title>(.*?)</title>`).FindSubmatch(content)
	if len(matches) < 2 {
		return "", nil
	}
	title := strings.TrimSpace(string(matches[1]))
	title = strings.ReplaceAll(title, "\n", " ")
	title = strings.Join(strings.Fields(title), " ")
	return title, nil
}

func defaultExampleTitle(dirName string, htmlFile string) string {
	base := strings.TrimSuffix(htmlFile, filepath.Ext(htmlFile))
	if base == "" {
		base = dirName
	}
	base = strings.ReplaceAll(base, "-", " ")
	base = strings.ReplaceAll(base, "_", " ")
	parts := strings.Fields(base)
	for index, part := range parts {
		parts[index] = strings.ToUpper(part[:1]) + part[1:]
	}
	if len(parts) == 0 {
		return dirName
	}
	return strings.Join(parts, " ") + " - GoWebComponents"
}

func exampleCatalogTags(dirName string, wasmBinary string, usesWasm bool) []string {
	trimmedName := dirName
	if parts := strings.SplitN(dirName, "-", 2); len(parts) == 2 {
		trimmedName = parts[1]
	}
	nameLower := strings.ToLower(trimmedName)

	seen := map[string]struct{}{}
	tags := make([]string, 0, 8)
	addTag := func(tag string) {
		tag = strings.TrimSpace(strings.ToLower(tag))
		if tag == "" {
			return
		}
		if _, ok := seen[tag]; ok {
			return
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
	}

	if usesWasm {
		addTag("wasm")
	}
	if strings.TrimSpace(wasmBinary) != "" {
		addTag(strings.TrimSuffix(strings.ToLower(wasmBinary), ".wasm"))
	}

	for _, token := range strings.FieldsFunc(strings.ToLower(trimmedName), func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	}) {
		addTag(token)
	}

	addFrameworkTags := func() {
		switch {
		case strings.HasPrefix(dirName, "21-") || strings.HasPrefix(dirName, "22-") || strings.HasPrefix(dirName, "23-") || strings.HasPrefix(dirName, "24-") || strings.HasPrefix(dirName, "25-") || strings.HasPrefix(dirName, "26-") || strings.HasPrefix(dirName, "27-") || strings.HasPrefix(dirName, "28-") || strings.HasPrefix(dirName, "29-") || strings.HasPrefix(dirName, "30-") || strings.HasPrefix(dirName, "31-") || strings.HasPrefix(dirName, "32-") || strings.HasPrefix(dirName, "33-") || strings.HasPrefix(dirName, "34-") || strings.HasPrefix(dirName, "35-") || strings.HasPrefix(dirName, "36-") || strings.HasPrefix(dirName, "46-") || strings.HasPrefix(dirName, "47-") || strings.HasPrefix(dirName, "48-") || strings.HasPrefix(dirName, "49-") || strings.HasPrefix(dirName, "50-") || strings.HasPrefix(dirName, "51-") || strings.HasPrefix(dirName, "75-") || strings.HasPrefix(dirName, "76-") || strings.HasPrefix(dirName, "77-") || strings.HasPrefix(dirName, "78-") || strings.HasPrefix(dirName, "79-") || strings.HasPrefix(dirName, "80-") || strings.HasPrefix(dirName, "81-") || strings.HasPrefix(dirName, "82-"):
			addTag("ui")
		case strings.HasPrefix(dirName, "01-") || strings.HasPrefix(dirName, "02-") || strings.HasPrefix(dirName, "03-") || strings.HasPrefix(dirName, "04-") || strings.HasPrefix(dirName, "05-") || strings.HasPrefix(dirName, "06-") || strings.HasPrefix(dirName, "07-") || strings.HasPrefix(dirName, "08-") || strings.HasPrefix(dirName, "09-") || strings.HasPrefix(dirName, "10-") || strings.HasPrefix(dirName, "11-") || strings.HasPrefix(dirName, "12-") || strings.HasPrefix(dirName, "13-") || strings.HasPrefix(dirName, "14-") || strings.HasPrefix(dirName, "15-") || strings.HasPrefix(dirName, "16-") || strings.HasPrefix(dirName, "17-") || strings.HasPrefix(dirName, "18-") || strings.HasPrefix(dirName, "19-") || strings.HasPrefix(dirName, "20-"):
			addTag("ui")
		}

		switch {
		case strings.HasPrefix(dirName, "37-") || strings.HasPrefix(dirName, "38-") || strings.HasPrefix(dirName, "39-") || strings.HasPrefix(dirName, "40-") || strings.HasPrefix(dirName, "41-"):
			addTag("state")
		}

		switch {
		case strings.HasPrefix(dirName, "42-") || strings.HasPrefix(dirName, "43-") || strings.HasPrefix(dirName, "44-") || strings.HasPrefix(dirName, "45-") || strings.HasPrefix(dirName, "93-"):
			addTag("fetch")
		}

		switch {
		case strings.HasPrefix(dirName, "52-") || strings.HasPrefix(dirName, "53-") || strings.HasPrefix(dirName, "54-") || strings.HasPrefix(dirName, "88-") || strings.HasPrefix(dirName, "89-"):
			addTag("html")
		}

		switch {
		case strings.HasPrefix(dirName, "55-") || strings.HasPrefix(dirName, "56-") || strings.HasPrefix(dirName, "57-") || strings.HasPrefix(dirName, "58-") || strings.HasPrefix(dirName, "59-") || strings.HasPrefix(dirName, "60-") || strings.HasPrefix(dirName, "61-") || strings.HasPrefix(dirName, "62-") || strings.HasPrefix(dirName, "63-") || strings.HasPrefix(dirName, "64-") || strings.HasPrefix(dirName, "65-") || strings.HasPrefix(dirName, "92-") || strings.HasPrefix(dirName, "96-"):
			addTag("router")
		}

		switch {
		case strings.HasPrefix(dirName, "66-") || strings.HasPrefix(dirName, "67-") || strings.HasPrefix(dirName, "68-") || strings.HasPrefix(dirName, "69-") || strings.HasPrefix(dirName, "98-"):
			addTag("devtools")
		}

		switch {
		case strings.HasPrefix(dirName, "70-") || strings.HasPrefix(dirName, "71-") || strings.HasPrefix(dirName, "72-") || strings.HasPrefix(dirName, "73-") || strings.HasPrefix(dirName, "74-") || strings.HasPrefix(dirName, "84-") || strings.HasPrefix(dirName, "87-"):
			addTag("ssr")
		}

		switch {
		case strings.HasPrefix(dirName, "71-") || strings.HasPrefix(dirName, "72-") || strings.HasPrefix(dirName, "74-") || strings.HasPrefix(dirName, "84-"):
			addTag("hydration")
		}

		switch {
		case strings.HasPrefix(dirName, "83-") || strings.HasPrefix(dirName, "84-") || strings.HasPrefix(dirName, "85-"):
			addTag("i18n")
		}

		switch {
		case strings.HasPrefix(dirName, "90-") || strings.HasPrefix(dirName, "91-") || strings.HasPrefix(dirName, "94-") || strings.HasPrefix(dirName, "95-") || strings.Contains(nameLower, "multi-client") || strings.Contains(nameLower, "cross-tab") || strings.Contains(nameLower, "multi-window"):
			addTag("interop")
		}

		if strings.Contains(nameLower, "pwa") {
			addTag("pwa")
		}
		if strings.Contains(nameLower, "offline") {
			addTag("offline")
		}
		if strings.Contains(nameLower, "form") {
			addTag("forms")
		}
		if strings.Contains(nameLower, "worker") {
			addTag("workers")
		}
		if strings.Contains(nameLower, "overlay") || strings.Contains(nameLower, "portal") {
			addTag("overlays")
		}
		if strings.Contains(nameLower, "accessible") || strings.Contains(nameLower, "accessibility") {
			addTag("accessibility")
		}
		if strings.Contains(nameLower, "web-components") || strings.Contains(nameLower, "custom-element") || strings.Contains(nameLower, "custom-elements") {
			addTag("custom-elements")
		}
		if strings.Contains(nameLower, "transition") || strings.Contains(nameLower, "deferred") || strings.Contains(nameLower, "debounced") || strings.Contains(nameLower, "throttled") || strings.Contains(nameLower, "goroutines") {
			addTag("scheduling")
		}
		if strings.Contains(nameLower, "code-splitting") {
			addTag("code-splitting")
		}
	}

	addFrameworkTags()

	joined := nameLower
	if strings.Contains(joined, "multi-client") {
		addTag("multi-client")
	}
	if strings.Contains(joined, "cross-tab") {
		addTag("cross-tab")
	}
	if strings.Contains(joined, "multi-window") {
		addTag("multi-window")
	}
	if strings.Contains(joined, "pwa") {
		addTag("pwa")
	}
	if strings.Contains(joined, "ssr") {
		addTag("ssr")
	}
	if strings.Contains(joined, "router") {
		addTag("router")
	}
	if strings.Contains(joined, "state") {
		addTag("state")
	}
	if strings.Contains(joined, "fetch") {
		addTag("fetch")
	}
	if strings.Contains(joined, "devtools") {
		addTag("devtools")
	}

	return tags
}

func hasAnyTag(tags []string, expected ...string) bool {
	for _, tag := range tags {
		for _, candidate := range expected {
			if tag == candidate {
				return true
			}
		}
	}
	return false
}

func renderExamplesAppShellHTML(routePath string, catalogHref string) string {
	routePath = strings.TrimSpace(routePath)
	if routePath == "" {
		routePath = "/examples/"
	}
	catalogHref = strings.TrimSpace(catalogHref)
	if catalogHref == "" {
		catalogHref = "/examples/"
	}
	return renderExamplesShellHTML(examplesShellDocument{
		Title:             "GoWebComponents Examples",
		Description:       "GoWebComponents examples catalog powered by a Go server and a wasm-first, multi-client catalog app.",
		BodyClass:         "example-shell bg-[#08111d] text-white min-h-screen",
		RoutePath:         routePath,
		CatalogHref:       catalogHref,
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

func renderGeneratedExampleHTML(page generatedExamplePage) string {
	return renderExamplesShellHTML(examplesShellDocument{
		Title:            page.Title,
		Description:      page.Description,
		BodyClass:        "example-shell bg-[#08111d] text-white min-h-screen",
		BodyData:         map[string]string{"gwc-example": page.DirName, "gwc-entry": page.HTMLFile},
		RoutePath:        page.RoutePath,
		ExampleSlug:      page.DirName,
		ManifestHref:     page.ManifestHref,
		WasmURL:          "/static/bin/" + page.WasmBinary,
		FailureTitle:     page.Title,
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

func renderExamplesShellHTML(document examplesShellDocument) string {
	bootstrapScript := renderExamplesBootstrapDataScript(document)
	loaderScript := renderExamplesLoaderScriptTag(document.WasmURL, document.FailureTitle, document.FailureMessage, document.FailureHref, document.FailureLinkLabel)
	headChildren := []ui.Node{
		gwchtml.Meta(gwchtml.Props{Raw: map[string]interface{}{"charset": "utf-8"}}),
		gwchtml.Meta(gwchtml.Props{Raw: map[string]interface{}{"name": "viewport", "content": "width=device-width, initial-scale=1"}}),
		gwchtml.Meta(gwchtml.Props{Raw: map[string]interface{}{"name": "description", "content": document.Description}}),
		gwchtml.Tag("title", gwchtml.Props{}, gwchtml.Text(document.Title)),
		gwchtml.Link(gwchtml.Props{Rel: "stylesheet", Href: "/static/css/tailwind.css"}),
		gwchtml.Link(gwchtml.Props{Rel: "stylesheet", Href: "/static/css/example-shell.css"}),
		gwchtml.Script(gwchtml.Props{Src: "/static/script/wasm_exec.js"}),
		gwchtml.Script(gwchtml.Props{Src: "/static/script/example-logger.js"}),
	}
	if strings.TrimSpace(document.ManifestHref) != "" {
		headChildren = append(headChildren, gwchtml.Link(gwchtml.Props{Rel: "manifest", Href: document.ManifestHref}))
	}

	bodyChildren := []ui.Node{gwchtml.Div(gwchtml.Props{ID: "app"})}
	if strings.TrimSpace(document.NoScriptMessage) != "" {
		bodyChildren = append(bodyChildren,
			gwchtml.NoScript(gwchtml.Props{},
				gwchtml.Main(gwchtml.Props{Style: map[string]string{"max-width": "72rem", "margin": "0 auto", "padding": "2rem", "font-family": "'Segoe UI Variable', 'Segoe UI', sans-serif"}},
					gwchtml.H1(gwchtml.Props{}, gwchtml.Text(document.FailureTitle)),
					gwchtml.P(gwchtml.Props{}, gwchtml.Text(document.NoScriptMessage)),
					gwchtml.P(gwchtml.Props{}, gwchtml.A(gwchtml.Props{Href: document.NoScriptHref}, gwchtml.Text(document.NoScriptLinkLabel))),
				),
			),
		)
	}

	bodyProps := gwchtml.Props{Class: document.BodyClass, Data: document.BodyData}
	markup, err := renderExamplesToString(gwchtml.Html(gwchtml.Props{Raw: map[string]interface{}{"lang": "en"}},
		gwchtml.Head(gwchtml.Props{}, headChildren...),
		gwchtml.Body(bodyProps, bodyChildren...),
	))
	if err != nil {
		return renderExamplesShellHTMLFallback(document, bootstrapScript)
	}
	if bootstrapScript != "" {
		markup = strings.Replace(markup, `<div id="app"></div>`, `<div id="app"></div>`+bootstrapScript, 1)
	}
	if loaderScript != "" {
		markup = strings.Replace(markup, `</body>`, loaderScript+`</body>`, 1)
	}
	return "<!doctype html>\n" + markup
}

func renderExamplesBootstrapDataScript(document examplesShellDocument) string {
	bootstrap := ui.SSRBootstrap{
		Route: ui.SSRRouteBootstrap{Path: document.RoutePath},
		Data: map[string]interface{}{
			"examples": map[string]interface{}{
				"mode":        "server",
				"catalogURL":  "/examples/catalog.json",
				"assetBase":   "/static/",
				"wasmBase":    "/static/bin/",
				"catalogHref": document.CatalogHref,
				"slug":        document.ExampleSlug,
			},
		},
	}
	script, err := renderExamplesUIBootstrapScript(bootstrap, "")
	if err != nil {
		return ""
	}
	return script
}

func renderExamplesBootstrapScript(wasmURL string, failureTitle string, failureMessage string, failureHref string, failureLinkLabel string) string {
	return "const GWC_EXAMPLES_CACHE = 'gwc-examples-runtime-v1';\n" +
		"async function loadCachedWasm(url, importObject) {\n" +
		"  if (!('caches' in globalThis)) {\n" +
		"    const response = await fetch(url, { cache: 'no-store' });\n" +
		"    if (!response.ok) {\n" +
		"      throw new Error('Failed to fetch wasm: ' + response.status + ' ' + response.statusText);\n" +
		"    }\n" +
		"    const bytes = await response.arrayBuffer();\n" +
		"    console.info('[gwc examples] wasm source: network (no Cache Storage)', url);\n" +
		"    return WebAssembly.instantiate(bytes, importObject);\n" +
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
		"  const bytes = await response.arrayBuffer();\n" +
		"  console.info('[gwc examples] wasm source: ' + source, url);\n" +
		"  return WebAssembly.instantiate(bytes, importObject);\n" +
		"}\n" +
		"const go = new Go();\n" +
		"loadCachedWasm(" + jsStringLiteral(wasmURL) + ", go.importObject)\n" +
		"  .then(result => go.run(result.instance))\n" +
		"  .catch(error => {\n" +
		"    const root = document.getElementById('app');\n" +
		"    if (root) {\n" +
		"      root.innerHTML = '<main style=\"max-width:72rem;margin:0 auto;padding:2rem;font-family:\\'Segoe UI Variable\\',\\'Segoe UI\\',sans-serif;\"><h1>" + escapeHTML(jsSingleQuoted(failureTitle)) + "</h1><p>" + escapeHTML(jsSingleQuoted(failureMessage)) + "</p><p><a href=\"" + escapeHTML(failureHref) + "\">" + escapeHTML(jsSingleQuoted(failureLinkLabel)) + "</a></p></main>';\n" +
		"    }\n" +
		"    console.error(error);\n" +
		"  });"
}

func renderExamplesLoaderScriptTag(wasmURL string, failureTitle string, failureMessage string, failureHref string, failureLinkLabel string) string {
	scriptBody := renderExamplesBootstrapScriptFunc(wasmURL, failureTitle, failureMessage, failureHref, failureLinkLabel)
	if strings.TrimSpace(scriptBody) == "" {
		return ""
	}
	return `<script>` + scriptBody + `</script>`
}

func renderExamplesShellHTMLFallback(document examplesShellDocument, bootstrapScript string) string {
	manifestLink := ""
	if strings.TrimSpace(document.ManifestHref) != "" {
		manifestLink = "\n  <link rel=\"manifest\" href=\"" + escapeHTML(document.ManifestHref) + "\">"
	}
	noscript := ""
	if strings.TrimSpace(document.NoScriptMessage) != "" {
		noscript = "\n  <noscript><main style=\"max-width:72rem;margin:0 auto;padding:2rem;font-family:'Segoe UI Variable','Segoe UI',sans-serif;\"><h1>" + escapeHTML(document.FailureTitle) + "</h1><p>" + escapeHTML(document.NoScriptMessage) + "</p><p><a href=\"" + escapeHTML(document.NoScriptHref) + "\">" + escapeHTML(document.NoScriptLinkLabel) + "</a></p></main></noscript>"
	}
	bodyAttrs := " class=\"" + escapeHTML(document.BodyClass) + "\""
	for key, value := range document.BodyData {
		bodyAttrs += " data-" + escapeHTML(key) + "=\"" + escapeHTML(value) + "\""
	}
	return "<!doctype html>\n<html lang=\"en\">\n<head>\n  <meta charset=\"utf-8\">\n  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n  <meta name=\"description\" content=\"" + escapeHTML(document.Description) + "\">" + manifestLink + "\n  <title>" + escapeHTML(document.Title) + "</title>\n  <link rel=\"stylesheet\" href=\"/static/css/tailwind.css\">\n  <link rel=\"stylesheet\" href=\"/static/css/example-shell.css\">\n  <script src=\"/static/script/wasm_exec.js\"></script>\n  <script src=\"/static/script/example-logger.js\"></script>\n</head>\n<body" + bodyAttrs + ">\n  <div id=\"app\"></div>" + noscript + bootstrapScript + "\n  <script>\n" + renderExamplesBootstrapScript(document.WasmURL, document.FailureTitle, document.FailureMessage, document.FailureHref, document.FailureLinkLabel) + "\n  </script>\n</body>\n</html>"
}

func jsStringLiteral(value string) string {
	return "'" + jsSingleQuoted(value) + "'"
}

func jsSingleQuoted(text string) string {
	text = strings.ReplaceAll(text, `\`, `\\`)
	text = strings.ReplaceAll(text, `'`, `\'`)
	return text
}

func renderExamplesListingHTML(links []exampleLink, query string) string {
	var items strings.Builder
	for _, link := range links {
		items.WriteString(`<li><a href="` + link.Href + `">` + link.Name + `</a></li>`)
	}
	if items.Len() == 0 {
		items.WriteString(`<li>No examples matched this search yet.</li>`)
	}
	metaText := "Generated from example folders under /examples."
	if strings.TrimSpace(query) != "" {
		metaText = fmt.Sprintf("Filtered examples for %q.", query)
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
	<p class="meta">` + metaText + `</p>
  <p><a href="/examples/static/index.html">Open styled showcase page</a></p>
	<form class="search" method="get" action="/examples/list">
		<input type="search" name="q" value="` + escapeHTML(query) + `" placeholder="Search examples by keyword">
		<button type="submit">Filter</button>
	</form>
  <ul>` + items.String() + `</ul>
</body>
</html>`
}

func escapeHTML(text string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		`"`, "&quot;",
		"<", "&lt;",
		">", "&gt;",
	)
	return replacer.Replace(text)
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(data)
}

var _ http.ResponseWriter = (*statusWriter)(nil)
