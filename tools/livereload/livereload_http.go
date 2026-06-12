package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/monstercameron/GoWebComponents/diagnostics"
)

func (parseLrs *LiveReloadServer) newHTTPHandler() http.Handler {
	parseMux := http.NewServeMux()
	parseFileServer := http.FileServer(http.Dir(parseLrs.projectRoot))
	parseMux.HandleFunc("/__gwc/status", parseLrs.handleStatus)
	parseMux.HandleFunc("/__gwc/clients/disconnect", parseLrs.handleClientDisconnect)
	if parseLrs.staticDir != "" {
		parseStaticFileServer := http.StripPrefix("/static/", http.FileServer(http.Dir(parseLrs.staticDir)))
		parseMux.Handle("/static/", parseStaticFileServer)
	}
	parseServedWASMPath := parseLrs.servedWASMPath()
	if parseServedWASMPath != "" {
		parseMux.HandleFunc(parseServedWASMPath, parseLrs.handleWASMArtifact)
	}
	parseMux.HandleFunc("/", func(parseW2 http.ResponseWriter, parseR2 *http.Request) {
		if parseR2.URL.Path == "/" {
			parseLrs.handleHTML(parseW2, parseR2, parseLrs.indexPath)
			return
		}
		if strings.HasSuffix(parseR2.URL.Path, ".html") {
			parseRelPath := strings.TrimPrefix(parseR2.URL.Path, "/")
			parseLrs.handleHTML(parseW2, parseR2, filepath.Join(parseLrs.projectRoot, filepath.FromSlash(parseRelPath)))
			return
		}
		parseFileServer.ServeHTTP(parseW2, parseR2)
	})
	parseMux.HandleFunc("/ws", parseLrs.handleWebSocketManaged)
	return parseMux
}

// handleWASMArtifact serves the build output, gzip-compressed when the client
// accepts it.  The compressed bytes are cached and keyed by artifact mtime, so
// each rebuild pays compression once instead of per-request.  This shrinks the
// dominant transfer (the multi-MB wasm) roughly 4-5x, which matters for
// tunneled/remote dev sessions and multi-client teams.
func (parseLrs *LiveReloadServer) handleWASMArtifact(parseW http.ResponseWriter, parseR *http.Request) {
	parseInfo, parseErr := os.Stat(parseLrs.outputPath)
	if parseErr != nil || !strings.Contains(parseR.Header.Get("Accept-Encoding"), "gzip") {
		http.ServeFile(parseW, parseR, parseLrs.outputPath)
		return
	}

	if parseLrs.precompressWASMArtifact(parseInfo.ModTime()) == 0 {
		http.ServeFile(parseW, parseR, parseLrs.outputPath)
		return
	}
	parseLrs.wasmGzipMu.Lock()
	parseCompressed := parseLrs.wasmGzipCache
	parseLrs.wasmGzipMu.Unlock()

	parseW.Header().Set("Content-Type", "application/wasm")
	parseW.Header().Set("Content-Encoding", "gzip")
	parseW.Header().Set("Content-Length", fmt.Sprintf("%d", len(parseCompressed)))
	parseW.Header().Set("Cache-Control", "no-cache")
	_, _ = parseW.Write(parseCompressed)
}

func (parseLrs *LiveReloadServer) statusURL() string {
	return "http://" + netAddr(parseLrs.host, parseLrs.port) + "/__gwc/status"
}

func (parseLrs *LiveReloadServer) websocketURL() string {
	return "ws://" + netAddr(parseLrs.host, parseLrs.port) + "/ws"
}

func (parseLrs *LiveReloadServer) currentStatus() LiveReloadStatus {
	parseLrs.mutex.Lock()
	parseLastClassification := parseLrs.lastClassification
	var parseLastBuild *BuildStatus
	var parseCurrentError *BuildStatus
	if parseLrs.lastBuildStatus != nil {
		buildCopy := *parseLrs.lastBuildStatus
		parseLastBuild = &buildCopy
		if !buildCopy.Success {
			parseErrorCopy := buildCopy
			parseCurrentError = &parseErrorCopy
		}
	}
	parseLrs.mutex.Unlock()

	parseClients := parseLrs.currentClientSessions()
	parseClientCount := len(parseClients)

	parseStatus := LiveReloadStatus{
		Mode:               "livereload-wasm",
		ListeningURL:       "http://" + netAddr(parseLrs.host, parseLrs.port),
		StatusURL:          parseLrs.statusURL(),
		WebSocketURL:       parseLrs.websocketURL(),
		ProjectRoot:        parseLrs.projectRoot,
		WatchRoot:          parseLrs.watchRoot,
		BuildDir:           parseLrs.buildDir,
		ServedWASMPath:     parseLrs.servedWASMPath(),
		HotReloadEnabled:   parseLrs.alwaysHotReload,
		HotReloadEligible:  parseLrs.alwaysHotReload || parseLastClassification.ReloadType == "hot",
		LastClassification: parseLastClassification,
		LastBuild:          parseLastBuild,
		CurrentError:       parseCurrentError,
		ClientCount:        parseClientCount,
		Clients:            parseClients,
	}
	return parseStatus
}

func (parseLrs *LiveReloadServer) currentClientSessions() []ClientSession {
	parseLrs.clientsMutex.RLock()
	defer parseLrs.clientsMutex.RUnlock()

	if len(parseLrs.clients) == 0 {
		return nil
	}

	parseSessions := make([]ClientSession, 0, len(parseLrs.clients))
	for _, parseSession := range parseLrs.clients {
		parseSessions = append(parseSessions, parseSession)
	}
	sort.Slice(parseSessions, func(parseI, parseJ int) bool {
		if parseSessions[parseI].ConnectedAt.Equal(parseSessions[parseJ].ConnectedAt) {
			return parseSessions[parseI].ID < parseSessions[parseJ].ID
		}
		return parseSessions[parseI].ConnectedAt.Before(parseSessions[parseJ].ConnectedAt)
	})
	return parseSessions
}

func (parseLrs *LiveReloadServer) handleStatus(parseW http.ResponseWriter, parseR *http.Request) {
	parseStatus := parseLrs.currentStatus()
	parseW.Header().Set("Content-Type", "application/json; charset=utf-8")
	parseEncoded, parseErr := json.MarshalIndent(parseStatus, "", "  ")
	if parseErr != nil {
		diagnostics.WriteHTTPError(parseW, http.StatusInternalServerError, livereloadErrReport(
			"LiveReloadServer.handleStatus.marshal",
			parseR.URL.Path,
			parseErr,
			"the machine-readable dev status endpoint could not serialize the current livereload state.",
			"Inspect the status payload fields and marshalling assumptions before relying on editor or tooling integrations.",
		))
		return
	}
	_, _ = parseW.Write(parseEncoded)
}

func (parseLrs *LiveReloadServer) handleClientDisconnect(parseW http.ResponseWriter, parseR *http.Request) {
	if parseR.Method != http.MethodPost {
		parseW.Header().Set("Allow", http.MethodPost)
		http.Error(parseW, "disconnect endpoint only supports POST", http.StatusMethodNotAllowed)
		return
	}

	parseRequest := clientDisconnectRequest{
		ClientID: strings.TrimSpace(parseR.URL.Query().Get("clientID")),
		All:      strings.EqualFold(strings.TrimSpace(parseR.URL.Query().Get("all")), "true"),
	}
	if parseR.Body != nil && parseR.ContentLength != 0 {
		defer parseR.Body.Close()
		var parseDecoded clientDisconnectRequest
		if parseErr := json.NewDecoder(parseR.Body).Decode(&parseDecoded); parseErr != nil {
			http.Error(parseW, "invalid disconnect request payload", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(parseDecoded.ClientID) != "" {
			parseRequest.ClientID = strings.TrimSpace(parseDecoded.ClientID)
		}
		parseRequest.All = parseRequest.All || parseDecoded.All
	}
	if !parseRequest.All && parseRequest.ClientID == "" {
		http.Error(parseW, "specify clientID or all=true", http.StatusBadRequest)
		return
	}

	parseDisconnectedIDs, parseRemaining := parseLrs.disconnectClients(parseRequest.ClientID, parseRequest.All)
	parseResponse := clientDisconnectResponse{
		DisconnectedIDs: parseDisconnectedIDs,
		Disconnected:    len(parseDisconnectedIDs),
		Remaining:       parseRemaining,
	}
	parseW.Header().Set("Content-Type", "application/json; charset=utf-8")
	parseEncoded, parseErr2 := json.MarshalIndent(parseResponse, "", "  ")
	if parseErr2 != nil {
		http.Error(parseW, "failed to encode disconnect response", http.StatusInternalServerError)
		return
	}
	_, _ = parseW.Write(parseEncoded)
}

func (parseLrs *LiveReloadServer) disconnectClients(parseClientID string, isDisconnectAll bool) ([]string, int) {
	parseLrs.clientsMutex.Lock()
	parseTargets := make([]*websocket.Conn, 0, len(parseLrs.clients))
	parseDisconnectedIDs := make([]string, 0, len(parseLrs.clients))
	for parseConn, parseSession := range parseLrs.clients {
		if isDisconnectAll || parseSession.ID == parseClientID {
			parseTargets = append(parseTargets, parseConn)
			parseDisconnectedIDs = append(parseDisconnectedIDs, parseSession.ID)
			delete(parseLrs.clients, parseConn)
			if !isDisconnectAll {
				break
			}
		}
	}
	parseRemaining := len(parseLrs.clients)
	parseLrs.clientsMutex.Unlock()

	sort.Strings(parseDisconnectedIDs)
	for _, parseConn2 := range parseTargets {
		_ = parseConn2.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Disconnected by gwc dashboard"), time.Now().Add(time.Second))
		_ = parseConn2.Close()
	}
	return parseDisconnectedIDs, parseRemaining
}

func (parseLrs *LiveReloadServer) nextClientSession() ClientSession {
	parseNow := time.Now().UTC()
	return ClientSession{
		ID:          fmt.Sprintf("client-%d", atomic.AddUint64(&parseLrs.nextClientID, 1)),
		ConnectedAt: parseNow,
		LastSeenAt:  parseNow,
	}
}

func (parseLrs *LiveReloadServer) markClientSeen(parseConn *websocket.Conn) {
	parseLrs.clientsMutex.Lock()
	defer parseLrs.clientsMutex.Unlock()
	parseSession, parseOk := parseLrs.clients[parseConn]
	if !parseOk {
		return
	}
	parseSession.LastSeenAt = time.Now().UTC()
	parseLrs.clients[parseConn] = parseSession
}

func (parseLrs *LiveReloadServer) handleHTML(parseW http.ResponseWriter, parseR *http.Request, parseFilePath string) {
	parseHtmlContent, parseErr := os.ReadFile(parseFilePath)
	if parseErr != nil {
		http.NotFound(parseW, parseR)
		return
	}

	parseScriptContent, parseErr := livereloadClientScriptBytes(parseLrs.clientScriptPath)
	if parseErr != nil {
		parseScriptPath := strings.TrimSpace(parseLrs.clientScriptPath)
		if parseScriptPath == "" {
			parseScriptPath = "embedded launcher client script"
		}
		diagnostics.WriteHTTPError(parseW, http.StatusInternalServerError, livereloadErrReport(
			"LiveReloadServer.handleHTML.clientScript",
			parseScriptPath,
			parseErr,
			"the HTML response could not inject the livereload client script, so browser reload coordination is unavailable for this request.",
			"Verify the configured livereload client override path or restore the embedded launcher client asset before serving HTML through this tool.",
		))
		return
	}

	parseConfigScript := fmt.Sprintf("\n<script>\nwindow.__GWC_LIVERELOAD_CONFIG = Object.assign({}, window.__GWC_LIVERELOAD_CONFIG || {}, { wasmPath: %q, projectRoot: %q });\n</script>", parseLrs.servedWASMPath(), filepath.ToSlash(parseLrs.projectRoot))
	parseLiveReloadScript := fmt.Sprintf("%s\n<script>\n%s\n</script>", parseConfigScript, string(parseScriptContent))
	parseModifiedContent := strings.Replace(string(parseHtmlContent), "</body>", parseLiveReloadScript+"\n</body>", 1)

	parseW.Header().Set("Content-Type", "text/html; charset=utf-8")
	parseW.Write([]byte(parseModifiedContent))
}

func (parseLrs *LiveReloadServer) servedWASMPath() string {
	if parseLrs == nil {
		return "/main.wasm"
	}

	parseOutputPath := strings.TrimSpace(parseLrs.outputPath)
	if parseOutputPath == "" {
		return "/main.wasm"
	}
	if !filepath.IsAbs(parseOutputPath) {
		parseOutputPath = filepath.Join(parseLrs.buildDir, parseOutputPath)
	}

	parseRelPath, parseErr := filepath.Rel(parseLrs.projectRoot, parseOutputPath)
	if parseErr == nil && parseRelPath != "" && parseRelPath != "." && !strings.HasPrefix(parseRelPath, "..") && !filepath.IsAbs(parseRelPath) {
		return "/" + filepath.ToSlash(parseRelPath)
	}

	parseBase := strings.TrimSpace(filepath.Base(parseOutputPath))
	if parseBase == "" || parseBase == "." || parseBase == string(filepath.Separator) {
		return "/main.wasm"
	}
	return "/" + filepath.ToSlash(parseBase)
}

// precompressWASMArtifact ensures the gzip cache matches the artifact with the
// given mtime, compressing if needed, and returns the compressed size in bytes
// (0 on failure).  Called after each successful build (so the first reload
// fetch is served from cache) and lazily from the artifact handler.
func (parseLrs *LiveReloadServer) precompressWASMArtifact(parseModTime time.Time) int64 {
	parseLrs.wasmGzipMu.Lock()
	defer parseLrs.wasmGzipMu.Unlock()
	if parseLrs.wasmGzipModTime.Equal(parseModTime) && parseLrs.wasmGzipCache != nil {
		return int64(len(parseLrs.wasmGzipCache))
	}
	parseRaw, parseReadErr := os.ReadFile(parseLrs.outputPath)
	if parseReadErr != nil {
		return 0
	}
	var parseBuf bytes.Buffer
	parseWriter, _ := gzip.NewWriterLevel(&parseBuf, gzip.BestSpeed)
	if _, parseWriteErr := parseWriter.Write(parseRaw); parseWriteErr != nil {
		return 0
	}
	if parseCloseErr := parseWriter.Close(); parseCloseErr != nil {
		return 0
	}
	parseLrs.wasmGzipCache = parseBuf.Bytes()
	parseLrs.wasmGzipModTime = parseModTime
	return int64(len(parseLrs.wasmGzipCache))
}
