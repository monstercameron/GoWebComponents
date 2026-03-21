//go:build !js || !wasm
// +build !js !wasm

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

func secureFormsRequestReport(subject string, path string, err error, consequence string, next string) diagnostics.Report {
	return diagnostics.Build(diagnostics.Options{
		Summary:  err.Error(),
		Code:     "GWC-EXAMPLE-SERVER-REQUEST",
		Headline: "server failure in " + strings.TrimSpace(subject),
		Path:     strings.TrimSpace(path),
		Runtime:  strings.TrimSpace(consequence),
		Next:     strings.TrimSpace(next),
		Docs:     secureFormsRequestDocs,
	})
}

func fatalSecureFormsStartup(path string, err error) {
	diagnostics.Emit(diagnostics.Build(diagnostics.Options{
		Summary:  err.Error(),
		Code:     "GWC-EXAMPLE-SERVER-STARTUP",
		Headline: "server startup failure in secure forms demo",
		Path:     strings.TrimSpace(path),
		Runtime:  "the secure forms example did not start, so no requests can be served.",
		Next:     "Free the configured port or update PORT before restarting the secure forms example.",
		Docs:     secureFormsStartupDocs,
	}))
	os.Exit(1)
}

func (s *secureFormsServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("POST /quote", s.handleQuote)
	mux.HandleFunc("POST /upload", s.handleUpload)
	return mux
}

func (s *secureFormsServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	state := newPageState(ensureCSRFCookie(w, r), r.URL.Query())
	s.renderHTML(w, http.StatusOK, state)
}

func (s *secureFormsServer) handleQuote(w http.ResponseWriter, r *http.Request) {
	if !validateCSRFFromForm(r) {
		diagnostics.WriteHTTPError(w, http.StatusForbidden, secureFormsRequestReport(
			"secureFormsServer.handleQuote.csrf",
			r.URL.Path,
			errors.New("csrf validation failed"),
			"the quote submission was rejected before any mutation because the CSRF proof was missing or invalid.",
			"Reload the form to refresh the CSRF token and resubmit from the same origin.",
		))
		return
	}
	if err := r.ParseForm(); err != nil {
		diagnostics.WriteHTTPError(w, http.StatusBadRequest, secureFormsRequestReport(
			"secureFormsServer.handleQuote.ParseForm",
			r.URL.Path,
			err,
			"the quote submission body could not be parsed, so the request ended before validation or redirect.",
			"Inspect the form encoding and request payload for this submission.",
		))
		return
	}
	input := quoteForm{
		Name:     strings.TrimSpace(r.Form.Get("name")),
		Email:    strings.TrimSpace(r.Form.Get("email")),
		Company:  strings.TrimSpace(r.Form.Get("company")),
		Timeline: strings.TrimSpace(r.Form.Get("timeline")),
		Notes:    strings.TrimSpace(r.Form.Get("notes")),
	}
	state := newPageState(ensureCSRFCookie(w, r), r.URL.Query())
	state.Quote = input
	state.QuoteErrors = validateQuote(input)
	if len(state.QuoteErrors) > 0 {
		state.Notice = "Fix the highlighted quote fields and resubmit."
		s.renderHTML(w, http.StatusBadRequest, state)
		return
	}
	redirectWithNotice(w, r, "Quote request captured with CSRF validation and a 303 redirect.")
}

func (s *secureFormsServer) handleUpload(w http.ResponseWriter, r *http.Request) {
	if !validateCSRFFromForm(r) {
		diagnostics.WriteHTTPError(w, http.StatusForbidden, secureFormsRequestReport(
			"secureFormsServer.handleUpload.csrf",
			r.URL.Path,
			errors.New("csrf validation failed"),
			"the upload submission was rejected before any file processing because the CSRF proof was missing or invalid.",
			"Reload the form to refresh the CSRF token and resubmit from the same origin.",
		))
		return
	}
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		diagnostics.WriteHTTPError(w, http.StatusBadRequest, secureFormsRequestReport(
			"secureFormsServer.handleUpload.ParseMultipartForm",
			r.URL.Path,
			err,
			"the multipart request could not be parsed, so the upload was rejected before validation.",
			"Inspect the multipart form encoding and upload size limits for this request.",
		))
		return
	}
	state := newPageState(ensureCSRFCookie(w, r), r.URL.Query())
	state.Upload = uploadForm{Label: strings.TrimSpace(r.Form.Get("label"))}

	file, header, err := r.FormFile("asset")
	if err != nil && !errors.Is(err, http.ErrMissingFile) {
		diagnostics.WriteHTTPError(w, http.StatusBadRequest, secureFormsRequestReport(
			"secureFormsServer.handleUpload.FormFile",
			r.URL.Path,
			err,
			"the upload could not read the submitted file field, so validation stopped before redirect.",
			"Inspect the multipart asset field name and browser submission payload for this request.",
		))
		return
	}
	if file != nil {
		defer file.Close()
	}

	var (
		sizeBytes   int64
		contentType string
	)
	if file != nil {
		data, readErr := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
		if readErr != nil {
			diagnostics.WriteHTTPError(w, http.StatusBadRequest, secureFormsRequestReport(
				"secureFormsServer.handleUpload.ReadAll",
				r.URL.Path,
				readErr,
				"the upload body could not be read completely, so file validation stopped before redirect.",
				"Inspect the uploaded file stream and size limits for this request.",
			))
			return
		}
		sizeBytes = int64(len(data))
		if sizeBytes > 0 {
			contentType = http.DetectContentType(data)
		}
	}

	state.UploadErrors = validateUpload(state.Upload, header, sizeBytes, contentType)
	if len(state.UploadErrors) > 0 {
		state.Notice = "Fix the upload fields and try again."
		s.renderHTML(w, http.StatusBadRequest, state)
		return
	}

	query := url.Values{}
	query.Set("notice", "Asset accepted with CSRF validation and a 303 redirect.")
	query.Set("asset", header.Filename)
	query.Set("type", contentType)
	http.Redirect(w, r, "/?"+query.Encode(), http.StatusSeeOther)
}

func (s *secureFormsServer) renderHTML(w http.ResponseWriter, status int, state pageState) {
	document, err := renderDocument(state)
	if err != nil {
		diagnostics.WriteHTTPError(w, http.StatusInternalServerError, secureFormsRequestReport(
			"secureFormsServer.renderHTML",
			"/",
			err,
			"the secure forms page could not be rendered, so the request returned HTTP 500 without HTML.",
			"Inspect the secure forms document render path and the page state being serialized.",
		))
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(document))
}

func redirectWithNotice(w http.ResponseWriter, r *http.Request, notice string) {
	query := url.Values{}
	query.Set("notice", notice)
	http.Redirect(w, r, "/?"+query.Encode(), http.StatusSeeOther)
}

func ensureCSRFCookie(w http.ResponseWriter, r *http.Request) string {
	if cookie, err := r.Cookie(secureFormsCSRFCookie); err == nil && strings.TrimSpace(cookie.Value) != "" {
		return cookie.Value
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return base64.RawURLEncoding.EncodeToString([]byte("gwc-secure-forms-fallback-token"))
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	http.SetCookie(w, &http.Cookie{
		Name:     secureFormsCSRFCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	return token
}

func validateCSRFFromForm(r *http.Request) bool {
	if !sameOriginRequest(r) {
		return false
	}
	cookie, err := r.Cookie(secureFormsCSRFCookie)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return false
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
			return false
		}
	} else {
		if err := r.ParseForm(); err != nil {
			return false
		}
	}
	fieldName, _ := ui.NewCSRFToken(cookie.Value).FormField()
	token := strings.TrimSpace(r.Form.Get(fieldName))
	return token != "" && subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(token)) == 1
}

func sameOriginRequest(r *http.Request) bool {
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		return sameOrigin(origin, r)
	}
	if referer := strings.TrimSpace(r.Header.Get("Referer")); referer != "" {
		return sameOrigin(referer, r)
	}
	return false
}

func sameOrigin(raw string, r *http.Request) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return strings.EqualFold(parsed.Scheme, scheme) && strings.EqualFold(parsed.Host, r.Host)
}

func main() {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8087"
	}
	server := &secureFormsServer{}
	fmt.Printf("SSR secure forms demo listening on http://127.0.0.1:%s\n", port)
	if err := http.ListenAndServe("127.0.0.1:"+port, server.routes()); err != nil {
		fatalSecureFormsStartup("127.0.0.1:"+port, err)
	}
}
