//go:build !js || !wasm

package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/monstercameron/GoWebComponents/diagnostics"
	"github.com/monstercameron/GoWebComponents/ui"
)

const (
	secureFormsCSRFCookie  = "gwc_secure_forms_csrf"
	secureFormsRequestDocs = "ACTIONABLE_ERRORS.md#gwc-example-server-request"
	secureFormsStartupDocs = "ACTIONABLE_ERRORS.md#gwc-example-server-startup"
)

type secureFormsServer struct{}

func secureFormsRequestReport(parseSubject string, parsePath string, parseErr error, parseConsequence string, parseNext string) diagnostics.Report {
	return diagnostics.NewReport(diagnostics.Options{
		Summary:  parseErr.Error(),
		Code:     "GWC-EXAMPLE-SERVER-REQUEST",
		Headline: "server failure in " + strings.TrimSpace(parseSubject),
		Path:     strings.TrimSpace(parsePath),
		Runtime:  strings.TrimSpace(parseConsequence),
		Next:     strings.TrimSpace(parseNext),
		Docs:     secureFormsRequestDocs,
	})
}

func fatalSecureFormsStartup(parsePath string, parseErr error) {
	diagnostics.Emit(diagnostics.NewReport(diagnostics.Options{
		Summary:  parseErr.Error(),
		Code:     "GWC-EXAMPLE-SERVER-STARTUP",
		Headline: "server startup failure in secure forms demo",
		Path:     strings.TrimSpace(parsePath),
		Runtime:  "the secure forms example did not start, so no requests can be served.",
		Next:     "Free the configured port or update PORT before restarting the secure forms example.",
		Docs:     secureFormsStartupDocs,
	}))
	os.Exit(1)
}

func (parseS *secureFormsServer) routes() http.Handler {
	parseMux := http.NewServeMux()
	parseMux.HandleFunc("GET /", parseS.handleIndex)
	parseMux.HandleFunc("POST /quote", parseS.handleQuote)
	parseMux.HandleFunc("POST /upload", parseS.handleUpload)
	return parseMux
}

func (parseS *secureFormsServer) handleIndex(parseW http.ResponseWriter, parseR *http.Request) {
	parseState := newPageState(ensureCSRFCookie(parseW, parseR), parseR.URL.Query())
	parseS.renderHTML(parseW, http.StatusOK, parseState)
}

func (parseS *secureFormsServer) handleQuote(parseW http.ResponseWriter, parseR *http.Request) {
	if !validateCSRFFromForm(parseR) {
		diagnostics.WriteHTTPError(parseW, http.StatusForbidden, secureFormsRequestReport(
			"secureFormsServer.handleQuote.csrf",
			parseR.URL.Path,
			errors.New("csrf validation failed"),
			"the quote submission was rejected before any mutation because the CSRF proof was missing or invalid.",
			"Reload the form to refresh the CSRF token and resubmit from the same origin.",
		))
		return
	}
	if parseErr := parseR.ParseForm(); parseErr != nil {
		diagnostics.WriteHTTPError(parseW, http.StatusBadRequest, secureFormsRequestReport(
			"secureFormsServer.handleQuote.ParseForm",
			parseR.URL.Path,
			parseErr,
			"the quote submission body could not be parsed, so the request ended before validation or redirect.",
			"Inspect the form encoding and request payload for this submission.",
		))
		return
	}
	parseInput := quoteForm{
		Name:     strings.TrimSpace(parseR.Form.Get("name")),
		Email:    strings.TrimSpace(parseR.Form.Get("email")),
		Company:  strings.TrimSpace(parseR.Form.Get("company")),
		Timeline: strings.TrimSpace(parseR.Form.Get("timeline")),
		Notes:    strings.TrimSpace(parseR.Form.Get("notes")),
	}
	parseState := newPageState(ensureCSRFCookie(parseW, parseR), parseR.URL.Query())
	parseState.Quote = parseInput
	parseState.QuoteErrors = validateQuote(parseInput)
	if len(parseState.QuoteErrors) > 0 {
		parseState.Notice = "Fix the highlighted quote fields and resubmit."
		parseS.renderHTML(parseW, http.StatusBadRequest, parseState)
		return
	}
	redirectWithNotice(parseW, parseR, "Quote request captured with CSRF validation and a 303 redirect.")
}

func (parseS *secureFormsServer) handleUpload(parseW http.ResponseWriter, parseR *http.Request) {
	if !validateCSRFFromForm(parseR) {
		diagnostics.WriteHTTPError(parseW, http.StatusForbidden, secureFormsRequestReport(
			"secureFormsServer.handleUpload.csrf",
			parseR.URL.Path,
			errors.New("csrf validation failed"),
			"the upload submission was rejected before any file processing because the CSRF proof was missing or invalid.",
			"Reload the form to refresh the CSRF token and resubmit from the same origin.",
		))
		return
	}
	if parseErr := parseR.ParseMultipartForm(maxUploadBytes); parseErr != nil {
		diagnostics.WriteHTTPError(parseW, http.StatusBadRequest, secureFormsRequestReport(
			"secureFormsServer.handleUpload.ParseMultipartForm",
			parseR.URL.Path,
			parseErr,
			"the multipart request could not be parsed, so the upload was rejected before validation.",
			"Inspect the multipart form encoding and upload size limits for this request.",
		))
		return
	}
	parseState := newPageState(ensureCSRFCookie(parseW, parseR), parseR.URL.Query())
	parseState.Upload = uploadForm{Label: strings.TrimSpace(parseR.Form.Get("label"))}

	parseFile, parseHeader, parseErr2 := parseR.FormFile("asset")
	if parseErr2 != nil && !errors.Is(parseErr2, http.ErrMissingFile) {
		diagnostics.WriteHTTPError(parseW, http.StatusBadRequest, secureFormsRequestReport(
			"secureFormsServer.handleUpload.FormFile",
			parseR.URL.Path,
			parseErr2,
			"the upload could not read the submitted file field, so validation stopped before redirect.",
			"Inspect the multipart asset field name and browser submission payload for this request.",
		))
		return
	}
	if parseFile != nil {
		defer parseFile.Close()
	}

	var (
		parseSizeBytes   int64
		parseContentType string
	)
	if parseFile != nil {
		parseData, parseReadErr := io.ReadAll(io.LimitReader(parseFile, maxUploadBytes+1))
		if parseReadErr != nil {
			diagnostics.WriteHTTPError(parseW, http.StatusBadRequest, secureFormsRequestReport(
				"secureFormsServer.handleUpload.ReadAll",
				parseR.URL.Path,
				parseReadErr,
				"the upload body could not be read completely, so file validation stopped before redirect.",
				"Inspect the uploaded file stream and size limits for this request.",
			))
			return
		}
		parseSizeBytes = int64(len(parseData))
		if parseSizeBytes > 0 {
			parseContentType = http.DetectContentType(parseData)
		}
	}

	parseState.UploadErrors = validateUpload(parseState.Upload, parseHeader, parseSizeBytes, parseContentType)
	if len(parseState.UploadErrors) > 0 {
		parseState.Notice = "Fix the upload fields and try again."
		parseS.renderHTML(parseW, http.StatusBadRequest, parseState)
		return
	}

	parseQuery := url.Values{}
	parseQuery.Set("notice", "Asset accepted with CSRF validation and a 303 redirect.")
	parseQuery.Set("asset", parseHeader.Filename)
	parseQuery.Set("type", parseContentType)
	http.Redirect(parseW, parseR, "/?"+parseQuery.Encode(), http.StatusSeeOther)
}

func (parseS *secureFormsServer) renderHTML(parseW http.ResponseWriter, parseStatus int, parseState pageState) {
	parseDocument, parseErr := renderDocument(parseState)
	if parseErr != nil {
		diagnostics.WriteHTTPError(parseW, http.StatusInternalServerError, secureFormsRequestReport(
			"secureFormsServer.renderHTML",
			"/",
			parseErr,
			"the secure forms page could not be rendered, so the request returned HTTP 500 without HTML.",
			"Inspect the secure forms document render path and the page state being serialized.",
		))
		return
	}
	parseW.Header().Set("Content-Type", "text/html; charset=utf-8")
	parseW.WriteHeader(parseStatus)
	_, _ = parseW.Write([]byte(parseDocument))
}

func redirectWithNotice(parseW http.ResponseWriter, parseR *http.Request, parseNotice string) {
	parseQuery := url.Values{}
	parseQuery.Set("notice", parseNotice)
	http.Redirect(parseW, parseR, "/?"+parseQuery.Encode(), http.StatusSeeOther)
}

func ensureCSRFCookie(parseW http.ResponseWriter, parseR *http.Request) string {
	if parseCookie, parseErr := parseR.Cookie(secureFormsCSRFCookie); parseErr == nil && strings.TrimSpace(parseCookie.Value) != "" {
		return parseCookie.Value
	}
	parseRaw := make([]byte, 32)
	if _, parseErr2 := rand.Read(parseRaw); parseErr2 != nil {
		return base64.RawURLEncoding.EncodeToString([]byte("gwc-secure-forms-fallback-token"))
	}
	parseToken := base64.RawURLEncoding.EncodeToString(parseRaw)
	http.SetCookie(parseW, &http.Cookie{
		Name:     secureFormsCSRFCookie,
		Value:    parseToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	return parseToken
}

func validateCSRFFromForm(parseR *http.Request) bool {
	if !sameOriginRequest(parseR) {
		return false
	}
	parseCookie, parseErr := parseR.Cookie(secureFormsCSRFCookie)
	if parseErr != nil || strings.TrimSpace(parseCookie.Value) == "" {
		return false
	}
	parseContentType := strings.ToLower(strings.TrimSpace(parseR.Header.Get("Content-Type")))
	if strings.Contains(parseContentType, "multipart/form-data") {
		if parseErr2 := parseR.ParseMultipartForm(maxUploadBytes); parseErr2 != nil {
			return false
		}
	} else {
		if parseErr3 := parseR.ParseForm(); parseErr3 != nil {
			return false
		}
	}
	parseFieldName, _ := ui.NewCSRFToken(parseCookie.Value).FormField()
	parseToken := strings.TrimSpace(parseR.Form.Get(parseFieldName))
	return parseToken != "" && subtle.ConstantTimeCompare([]byte(parseCookie.Value), []byte(parseToken)) == 1
}

func sameOriginRequest(parseR *http.Request) bool {
	if parseOrigin := strings.TrimSpace(parseR.Header.Get("Origin")); parseOrigin != "" {
		return sameOrigin(parseOrigin, parseR)
	}
	if parseReferer := strings.TrimSpace(parseR.Header.Get("Referer")); parseReferer != "" {
		return sameOrigin(parseReferer, parseR)
	}
	return false
}

func sameOrigin(parseRaw string, parseR *http.Request) bool {
	parseParsed, parseErr := url.Parse(parseRaw)
	if parseErr != nil {
		return false
	}
	if parseParsed.Scheme == "" || parseParsed.Host == "" {
		return false
	}
	parseScheme := "http"
	if parseR.TLS != nil {
		parseScheme = "https"
	}
	return strings.EqualFold(parseParsed.Scheme, parseScheme) && strings.EqualFold(parseParsed.Host, parseR.Host)
}

func main() {
	parsePort := strings.TrimSpace(os.Getenv("PORT"))
	if parsePort == "" {
		parsePort = "8087"
	}
	parseServer := &secureFormsServer{}
	fmt.Printf("SSR secure forms demo listening on http://127.0.0.1:%s\n", parsePort)
	if parseErr := http.ListenAndServe("127.0.0.1:"+parsePort, parseServer.routes()); parseErr != nil {
		fatalSecureFormsStartup("127.0.0.1:"+parsePort, parseErr)
	}
}
