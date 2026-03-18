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

	"github.com/monstercameron/GoWebComponents/ui"
)

const (
	secureFormsCSRFCookie = "gwc_secure_forms_csrf"
)

type secureFormsServer struct{}

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
		http.Error(w, "csrf validation failed", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
		http.Error(w, "csrf validation failed", http.StatusForbidden)
		return
	}
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	state := newPageState(ensureCSRFCookie(w, r), r.URL.Query())
	state.Upload = uploadForm{Label: strings.TrimSpace(r.Form.Get("label"))}

	file, header, err := r.FormFile("asset")
	if err != nil && !errors.Is(err, http.ErrMissingFile) {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
			http.Error(w, readErr.Error(), http.StatusBadRequest)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		panic(err)
	}
}
