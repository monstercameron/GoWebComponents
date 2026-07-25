//go:build !js || !wasm

package main

import (
	"bytes"
	"crypto/tls"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/ui"
)

func TestPageHelpersAndValidationBranches(parseT *testing.T) {
	parseState := newPageState("csrf-token", url.Values{
		"notice": {"  Upload complete  "},
		"asset":  {" board.png "},
		"type":   {" image/png "},
	})
	if parseState.CSRF != "csrf-token" || parseState.Notice != "Upload complete" {
		parseT.Fatalf("newPageState() = %+v", parseState)
	}
	if parseState.Quote.Timeline != "30_days" || !strings.Contains(parseState.Quote.Notes, "request-time rendered") {
		parseT.Fatalf("default quote state = %+v", parseState.Quote)
	}
	if parseState.Upload.Label != "Project board photo" {
		parseT.Fatalf("default upload state = %+v", parseState.Upload)
	}
	if parseState.Uploaded.Name != "board.png" || parseState.Uploaded.ContentType != "image/png" {
		parseT.Fatalf("uploaded asset state = %+v", parseState.Uploaded)
	}

	parseQuoteErrors := validateQuote(quoteForm{
		Name:     " ",
		Email:    "bad-address",
		Company:  "",
		Timeline: "tomorrow",
	})
	for _, parseKey := range []string{"name", "email", "company", "timeline"} {
		if _, parseOk := parseQuoteErrors[parseKey]; !parseOk {
			parseT.Fatalf("validateQuote() missing %q error: %+v", parseKey, parseQuoteErrors)
		}
	}
	if parseErrs := validateQuote(quoteForm{Name: "Ada", Email: "ada@example.com", Company: "Atlas", Timeline: "quarter"}); len(parseErrs) != 0 {
		parseT.Fatalf("validateQuote(valid) = %+v, want no errors", parseErrs)
	}

	if parseErrs2 := validateUpload(uploadForm{Label: ""}, nil, 0, ""); parseErrs2["label"] == "" || parseErrs2["asset"] == "" {
		parseT.Fatalf("validateUpload(nil file) = %+v, want label and asset errors", parseErrs2)
	}
	if parseErrs3 := validateUpload(uploadForm{Label: "Board"}, &multipart.FileHeader{Filename: "board.png"}, 0, "image/png"); parseErrs3["asset"] != "Upload a non-empty file." {
		parseT.Fatalf("validateUpload(empty file) = %+v", parseErrs3)
	}
	if parseErrs4 := validateUpload(uploadForm{Label: "Board"}, &multipart.FileHeader{Filename: "board.png"}, maxUploadBytes+1, "image/png"); parseErrs4["asset"] != "Upload a file that is 2 MB or smaller." {
		parseT.Fatalf("validateUpload(oversize) = %+v", parseErrs4)
	}
	if parseErrs5 := validateUpload(uploadForm{Label: "Board"}, &multipart.FileHeader{Filename: "board.gif"}, 12, "image/gif"); parseErrs5["asset"] != "Upload a PNG or JPEG file." {
		parseT.Fatalf("validateUpload(wrong type) = %+v", parseErrs5)
	}
	if parseErrs6 := validateUpload(uploadForm{Label: "Board"}, &multipart.FileHeader{Filename: "board.png"}, 12, "image/png"); len(parseErrs6) != 0 {
		parseT.Fatalf("validateUpload(valid) = %+v, want no errors", parseErrs6)
	}
}

func TestRenderHelpersAndDocumentBranches(parseT *testing.T) {
	parseNoticeMarkup, parseErr := ui.RenderToString(renderNotice(pageState{Notice: "Saved"}))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(renderNotice) error = %v", parseErr)
	}
	if !strings.Contains(parseNoticeMarkup, "Saved") {
		parseT.Fatalf("renderNotice() markup = %q, want notice", parseNoticeMarkup)
	}
	parseEmptyNoticeMarkup, parseErr := ui.RenderToString(renderNotice(pageState{}))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(renderNotice empty) error = %v", parseErr)
	}
	if parseEmptyNoticeMarkup != "" {
		parseT.Fatalf("renderNotice(empty) = %q, want empty markup", parseEmptyNoticeMarkup)
	}

	parseNoUploadMarkup, parseErr := ui.RenderToString(renderUploadedSummary(uploadedAsset{}))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(renderUploadedSummary empty) error = %v", parseErr)
	}
	if !strings.Contains(parseNoUploadMarkup, "No file has been accepted yet") {
		parseT.Fatalf("renderUploadedSummary(empty) = %q", parseNoUploadMarkup)
	}
	parseUploadMarkup, parseErr := ui.RenderToString(renderUploadedSummary(uploadedAsset{Name: "board.png", ContentType: "image/png", SizeBytes: 128}))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(renderUploadedSummary full) error = %v", parseErr)
	}
	for _, parseExpected := range []string{"board.png", "image/png", "128 bytes"} {
		if !strings.Contains(parseUploadMarkup, parseExpected) {
			parseT.Fatalf("renderUploadedSummary(full) missing %q\n%s", parseExpected, parseUploadMarkup)
		}
	}

	parseInputMarkup, parseErr := ui.RenderToString(labeledInput("email", "Email", "ada@example.com", "Required"))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(labeledInput) error = %v", parseErr)
	}
	if !strings.Contains(parseInputMarkup, "ada@example.com") || !strings.Contains(parseInputMarkup, "Required") {
		parseT.Fatalf("labeledInput markup = %q", parseInputMarkup)
	}
	parseTextareaMarkup, parseErr := ui.RenderToString(labeledTextarea("notes", "Notes", "Need pricing", ""))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(labeledTextarea) error = %v", parseErr)
	}
	if !strings.Contains(parseTextareaMarkup, "Need pricing") {
		parseT.Fatalf("labeledTextarea markup = %q", parseTextareaMarkup)
	}
	parseTimelineMarkup, parseErr := ui.RenderToString(timelineField("quarter", "Choose one"))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(timelineField) error = %v", parseErr)
	}
	if !strings.Contains(parseTimelineMarkup, `selected value="quarter"`) || !strings.Contains(parseTimelineMarkup, "Choose one") {
		parseT.Fatalf("timelineField markup = %q", parseTimelineMarkup)
	}
	parseFileMarkup, parseErr := ui.RenderToString(fileField("Upload failed"))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(fileField) error = %v", parseErr)
	}
	if !strings.Contains(parseFileMarkup, "image/png,image/jpeg") || !strings.Contains(parseFileMarkup, "Upload failed") {
		parseT.Fatalf("fileField markup = %q", parseFileMarkup)
	}

	parseDocument, parseErr := renderDocument(pageState{
		CSRF:   "csrf-token",
		Notice: "Saved",
		Quote: quoteForm{
			Name:     "Ada",
			Email:    "ada@example.com",
			Company:  "Atlas",
			Timeline: "30_days",
		},
		QuoteErrors:  map[string]string{},
		Upload:       uploadForm{Label: "Board"},
		UploadErrors: map[string]string{},
	})
	if parseErr != nil {
		parseT.Fatalf("renderDocument() error = %v", parseErr)
	}
	for _, parseExpected2 := range []string{"<!DOCTYPE html>", "SSR Secure Forms Demo", "csrf_token", "Request pricing", "Upload asset"} {
		if !strings.Contains(parseDocument, parseExpected2) {
			parseT.Fatalf("renderDocument() missing %q\n%s", parseExpected2, parseDocument)
		}
	}
}

func TestCSRFAndOriginHelpers(parseT *testing.T) {
	parseServer := newTestServer()
	parseReq := httptest.NewRequest(http.MethodGet, "/", nil)
	parseReq.Host = "example.com"
	parseReq.AddCookie(&http.Cookie{Name: secureFormsCSRFCookie, Value: "existing-token"})
	parseRes := httptest.NewRecorder()
	parseToken := ensureCSRFCookie(parseRes, parseReq)
	if parseToken != "existing-token" {
		parseT.Fatalf("ensureCSRFCookie(existing) = %q, want existing-token", parseToken)
	}
	if len(parseRes.Result().Cookies()) != 0 {
		parseT.Fatalf("ensureCSRFCookie(existing) unexpectedly set cookies: %+v", parseRes.Result().Cookies())
	}

	parseFreshReq := httptest.NewRequest(http.MethodGet, "/", nil)
	parseFreshReq.Host = "example.com"
	parseFreshRes := httptest.NewRecorder()
	parseFreshToken := ensureCSRFCookie(parseFreshRes, parseFreshReq)
	if strings.TrimSpace(parseFreshToken) == "" {
		parseT.Fatal("ensureCSRFCookie(fresh) returned empty token")
	}
	parseCookies := parseFreshRes.Result().Cookies()
	if len(parseCookies) != 1 || parseCookies[0].Name != secureFormsCSRFCookie || parseCookies[0].Value != parseFreshToken || parseCookies[0].Path != "/" || !parseCookies[0].HttpOnly {
		parseT.Fatalf("ensureCSRFCookie(fresh) cookies = %+v, token = %q", parseCookies, parseFreshToken)
	}

	parseCsrfToken, parseCsrfCookie := loadCSRF(parseT, parseServer)
	parseValidForm := strings.NewReader("csrf_token=" + parseCsrfToken + "&name=Ada")
	parseValidReq := httptest.NewRequest(http.MethodPost, "/quote", parseValidForm)
	parseValidReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseValidReq.Header.Set("Origin", "http://example.com")
	parseValidReq.Host = "example.com"
	parseValidReq.AddCookie(parseCsrfCookie)
	if !validateCSRFFromForm(parseValidReq) {
		parseT.Fatal("validateCSRFFromForm(valid form) = false, want true")
	}

	parseInvalidOriginReq := httptest.NewRequest(http.MethodPost, "/quote", strings.NewReader("csrf_token="+parseCsrfToken))
	parseInvalidOriginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseInvalidOriginReq.Header.Set("Origin", "https://example.com")
	parseInvalidOriginReq.Host = "example.com"
	parseInvalidOriginReq.AddCookie(parseCsrfCookie)
	if validateCSRFFromForm(parseInvalidOriginReq) {
		parseT.Fatal("validateCSRFFromForm(cross origin) = true, want false")
	}

	parseBadTokenReq := httptest.NewRequest(http.MethodPost, "/quote", strings.NewReader("csrf_token=wrong"))
	parseBadTokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseBadTokenReq.Header.Set("Origin", "http://example.com")
	parseBadTokenReq.Host = "example.com"
	parseBadTokenReq.AddCookie(parseCsrfCookie)
	if validateCSRFFromForm(parseBadTokenReq) {
		parseT.Fatal("validateCSRFFromForm(bad token) = true, want false")
	}

	parseMultipartBody := &bytes.Buffer{}
	parseWriter := multipart.NewWriter(parseMultipartBody)
	_ = parseWriter.WriteField("csrf_token", parseCsrfToken)
	_ = parseWriter.Close()
	parseMultipartReq := httptest.NewRequest(http.MethodPost, "/upload", parseMultipartBody)
	parseMultipartReq.Header.Set("Content-Type", parseWriter.FormDataContentType())
	parseMultipartReq.Header.Set("Origin", "http://example.com")
	parseMultipartReq.Host = "example.com"
	parseMultipartReq.AddCookie(parseCsrfCookie)
	if !validateCSRFFromForm(parseMultipartReq) {
		parseT.Fatal("validateCSRFFromForm(multipart) = false, want true")
	}

	parseNoOriginReq := httptest.NewRequest(http.MethodPost, "http://example.com/quote", nil)
	parseNoOriginReq.Host = "example.com"
	if sameOriginRequest(parseNoOriginReq) {
		parseT.Fatal("sameOriginRequest(no headers) = true, want false")
	}
	parseRefererReq := httptest.NewRequest(http.MethodPost, "http://example.com/quote", nil)
	parseRefererReq.Host = "example.com"
	parseRefererReq.Header.Set("Referer", "http://example.com/form")
	if !sameOriginRequest(parseRefererReq) {
		parseT.Fatal("sameOriginRequest(referer) = false, want true")
	}
	parseCrossReq := httptest.NewRequest(http.MethodPost, "http://example.com/quote", nil)
	parseCrossReq.Host = "example.com"
	parseCrossReq.Header.Set("Origin", "http://evil.example")
	if sameOriginRequest(parseCrossReq) {
		parseT.Fatal("sameOriginRequest(cross origin) = true, want false")
	}

	parseHttpsReq := httptest.NewRequest(http.MethodPost, "https://example.com/quote", nil)
	parseHttpsReq.Host = "example.com"
	parseHttpsReq.TLS = &tls.ConnectionState{}
	if sameOrigin("https://example.com/path", parseHttpsReq) != true {
		parseT.Fatal("sameOrigin(https) = false, want true")
	}
	if sameOrigin("notaurl", parseHttpsReq) {
		parseT.Fatal("sameOrigin(invalid) = true, want false")
	}
	if sameOrigin("/relative", parseHttpsReq) {
		parseT.Fatal("sameOrigin(relative) = true, want false")
	}
}

func TestRoutesAndRedirectHelpers(parseT *testing.T) {
	parseServer := newTestServer()
	parseHandler := parseServer.routes()

	parseIndex := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseIndex, httptest.NewRequest(http.MethodGet, "/", nil))
	if parseIndex.Code != http.StatusOK {
		parseT.Fatalf("routes GET / = %d, want 200", parseIndex.Code)
	}

	parseNotAllowed := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseNotAllowed, httptest.NewRequest(http.MethodGet, "/quote", nil))
	if parseNotAllowed.Code != http.StatusOK || !strings.Contains(parseNotAllowed.Body.String(), "SSR Secure Forms") {
		parseT.Fatalf("routes GET /quote = code %d body %q, want index fallback", parseNotAllowed.Code, parseNotAllowed.Body.String())
	}

	parseRedirectReq := httptest.NewRequest(http.MethodPost, "/quote", nil)
	parseRedirectRes := httptest.NewRecorder()
	redirectWithNotice(parseRedirectRes, parseRedirectReq, "Saved with redirect")
	if parseRedirectRes.Code != http.StatusSeeOther || !strings.Contains(parseRedirectRes.Header().Get("Location"), "notice=Saved+with+redirect") {
		parseT.Fatalf("redirectWithNotice() = code %d location %q", parseRedirectRes.Code, parseRedirectRes.Header().Get("Location"))
	}

	parseReport := secureFormsRequestReport("handleQuote", "/quote", http.ErrMissingFile, "upload failed", "retry")
	if parseReport.Code != "GWC-EXAMPLE-SERVER-REQUEST" || parseReport.Path != "/quote" || !strings.Contains(parseReport.Headline, "handleQuote") {
		parseT.Fatalf("secureFormsRequestReport() = %+v", parseReport)
	}
}

func TestUploadErrorBranches(parseT *testing.T) {
	parseServer := newTestServer()
	parseCsrfToken, parseCsrfCookie := loadCSRF(parseT, parseServer)

	parseUrlEncodedReq := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("csrf_token="+parseCsrfToken+"&label=Board"))
	parseUrlEncodedReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseUrlEncodedReq.Header.Set("Origin", "http://example.com")
	parseUrlEncodedReq.Host = "example.com"
	parseUrlEncodedReq.AddCookie(parseCsrfCookie)
	parseUrlEncodedRes := httptest.NewRecorder()
	parseServer.handleUpload(parseUrlEncodedRes, parseUrlEncodedReq)
	if parseUrlEncodedRes.Code != http.StatusBadRequest {
		parseT.Fatalf("handleUpload(urlencoded) = %d, want 400", parseUrlEncodedRes.Code)
	}
	if !strings.Contains(strings.ToLower(parseUrlEncodedRes.Body.String()), "multipart") {
		parseT.Fatalf("handleUpload(urlencoded) body = %q, want multipart parse failure", parseUrlEncodedRes.Body.String())
	}

	parseMissingCSRFReq := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("label=Board"))
	parseMissingCSRFReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseMissingCSRFReq.Header.Set("Origin", "http://example.com")
	parseMissingCSRFReq.Host = "example.com"
	parseMissingCSRFRes := httptest.NewRecorder()
	parseServer.handleUpload(parseMissingCSRFRes, parseMissingCSRFReq)
	if parseMissingCSRFRes.Code != http.StatusForbidden {
		parseT.Fatalf("handleUpload(missing csrf) = %d, want 403", parseMissingCSRFRes.Code)
	}
	if !strings.Contains(strings.ToLower(parseMissingCSRFRes.Body.String()), "csrf") {
		parseT.Fatalf("handleUpload(missing csrf) body = %q, want csrf failure", parseMissingCSRFRes.Body.String())
	}
}
