//go:build !js || !wasm

package router

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// SPAFallbackHandler serves a history-router app: a request that names an
// existing file under staticDir gets that file, and any other GET or HEAD request
// gets indexHTML so a deep-link refresh such as /app/users/9 boots the app and the
// router resolves the route client-side. basePath is the prefix passed to
// WithBasePath ("" when the app is mounted at the root); requests outside it get 404.
//
// Requests whose last path segment has a file extension but no matching file get a
// real 404 rather than the index page, so a missing asset does not masquerade as HTML.
func SPAFallbackHandler(parseIndexHTML, parseStaticDir, parseBasePath string) http.Handler {
	parseBase := buildSPABasePath(parseBasePath)
	return http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		if parseR.Method != http.MethodGet && parseR.Method != http.MethodHead {
			http.Error(parseW, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		parseRel, parseOk := trimSPABasePath(parseR.URL.Path, parseBase)
		if !parseOk {
			http.NotFound(parseW, parseR)
			return
		}
		// path.Clean on a rooted path removes ".." so the lookup cannot escape staticDir.
		parseClean := path.Clean("/" + parseRel)
		if parseClean != "/" && parseStaticDir != "" {
			parseFile := filepath.Join(parseStaticDir, filepath.FromSlash(parseClean))
			if parseInfo, parseErr := os.Stat(parseFile); parseErr == nil && !parseInfo.IsDir() {
				serveSPAFile(parseW, parseR, parseFile)
				return
			}
		}
		if path.Ext(parseClean) != "" {
			http.NotFound(parseW, parseR)
			return
		}
		serveSPAFile(parseW, parseR, parseIndexHTML)
	})
}

// trimSPABasePath removes the base path from a request path, reporting false
// when the request is outside the base.
func trimSPABasePath(parseRequestPath, parseBase string) (string, bool) {
	if parseBase == "" {
		return parseRequestPath, true
	}
	if parseRequestPath == parseBase {
		return "/", true
	}
	if strings.HasPrefix(parseRequestPath, parseBase+"/") {
		return parseRequestPath[len(parseBase):], true
	}
	return "", false
}

// buildSPABasePath normalizes a base path to "" (root) or "/segment" with no trailing slash.
func buildSPABasePath(parseBase string) string {
	parseTrimmed := strings.Trim(strings.TrimSpace(parseBase), "/")
	if parseTrimmed == "" {
		return ""
	}
	return "/" + parseTrimmed
}

// serveSPAFile writes one file with http.ServeContent. ServeFile is avoided on
// purpose: it rejects request paths containing ".." and redirects "/index.html",
// neither of which applies once the handler has resolved the file itself.
func serveSPAFile(parseW http.ResponseWriter, parseR *http.Request, parseFile string) {
	parseHandle, parseErr := os.Open(parseFile)
	if parseErr != nil {
		http.NotFound(parseW, parseR)
		return
	}
	defer parseHandle.Close()
	parseInfo, parseErr := parseHandle.Stat()
	if parseErr != nil {
		http.NotFound(parseW, parseR)
		return
	}
	http.ServeContent(parseW, parseR, parseInfo.Name(), parseInfo.ModTime(), parseHandle)
}
