//go:build !js || !wasm
// +build !js !wasm

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

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestPageHelpersAndValidationBranches(t *testing.T) {
	state := newPageState("csrf-token", url.Values{
		"notice": {"  Upload complete  "},
		"asset":  {" board.png "},
		"type":   {" image/png "},
	})
	if state.CSRF != "csrf-token" || state.Notice != "Upload complete" {
		t.Fatalf("newPageState() = %+v", state)
	}
	if state.Quote.Timeline != "30_days" || !strings.Contains(state.Quote.Notes, "request-time rendered") {
		t.Fatalf("default quote state = %+v", state.Quote)
	}
	if state.Upload.Label != "Project board photo" {
		t.Fatalf("default upload state = %+v", state.Upload)
	}
	if state.Uploaded.Name != "board.png" || state.Uploaded.ContentType != "image/png" {
		t.Fatalf("uploaded asset state = %+v", state.Uploaded)
	}

	quoteErrors := validateQuote(quoteForm{
		Name:     " ",
		Email:    "bad-address",
		Company:  "",
		Timeline: "tomorrow",
	})
	for _, key := range []string{"name", "email", "company", "timeline"} {
		if _, ok := quoteErrors[key]; !ok {
			t.Fatalf("validateQuote() missing %q error: %+v", key, quoteErrors)
		}
	}
	if errs := validateQuote(quoteForm{Name: "Ada", Email: "ada@example.com", Company: "Atlas", Timeline: "quarter"}); len(errs) != 0 {
		t.Fatalf("validateQuote(valid) = %+v, want no errors", errs)
	}

	if errs := validateUpload(uploadForm{Label: ""}, nil, 0, ""); errs["label"] == "" || errs["asset"] == "" {
		t.Fatalf("validateUpload(nil file) = %+v, want label and asset errors", errs)
	}
	if errs := validateUpload(uploadForm{Label: "Board"}, &multipart.FileHeader{Filename: "board.png"}, 0, "image/png"); errs["asset"] != "Upload a non-empty file." {
		t.Fatalf("validateUpload(empty file) = %+v", errs)
	}
	if errs := validateUpload(uploadForm{Label: "Board"}, &multipart.FileHeader{Filename: "board.png"}, maxUploadBytes+1, "image/png"); errs["asset"] != "Upload a file that is 2 MB or smaller." {
		t.Fatalf("validateUpload(oversize) = %+v", errs)
	}
	if errs := validateUpload(uploadForm{Label: "Board"}, &multipart.FileHeader{Filename: "board.gif"}, 12, "image/gif"); errs["asset"] != "Upload a PNG or JPEG file." {
		t.Fatalf("validateUpload(wrong type) = %+v", errs)
	}
	if errs := validateUpload(uploadForm{Label: "Board"}, &multipart.FileHeader{Filename: "board.png"}, 12, "image/png"); len(errs) != 0 {
		t.Fatalf("validateUpload(valid) = %+v, want no errors", errs)
	}
}

func TestRenderHelpersAndDocumentBranches(t *testing.T) {
	noticeMarkup, err := ui.RenderToString(renderNotice(pageState{Notice: "Saved"}))
	if err != nil {
		t.Fatalf("RenderToString(renderNotice) error = %v", err)
	}
	if !strings.Contains(noticeMarkup, "Saved") {
		t.Fatalf("renderNotice() markup = %q, want notice", noticeMarkup)
	}
	emptyNoticeMarkup, err := ui.RenderToString(renderNotice(pageState{}))
	if err != nil {
		t.Fatalf("RenderToString(renderNotice empty) error = %v", err)
	}
	if emptyNoticeMarkup != "" {
		t.Fatalf("renderNotice(empty) = %q, want empty markup", emptyNoticeMarkup)
	}

	noUploadMarkup, err := ui.RenderToString(renderUploadedSummary(uploadedAsset{}))
	if err != nil {
		t.Fatalf("RenderToString(renderUploadedSummary empty) error = %v", err)
	}
	if !strings.Contains(noUploadMarkup, "No file has been accepted yet") {
		t.Fatalf("renderUploadedSummary(empty) = %q", noUploadMarkup)
	}
	uploadMarkup, err := ui.RenderToString(renderUploadedSummary(uploadedAsset{Name: "board.png", ContentType: "image/png", SizeBytes: 128}))
	if err != nil {
		t.Fatalf("RenderToString(renderUploadedSummary full) error = %v", err)
	}
	for _, expected := range []string{"board.png", "image/png", "128 bytes"} {
		if !strings.Contains(uploadMarkup, expected) {
			t.Fatalf("renderUploadedSummary(full) missing %q\n%s", expected, uploadMarkup)
		}
	}

	inputMarkup, err := ui.RenderToString(labeledInput("email", "Email", "ada@example.com", "Required"))
	if err != nil {
		t.Fatalf("RenderToString(labeledInput) error = %v", err)
	}
	if !strings.Contains(inputMarkup, "ada@example.com") || !strings.Contains(inputMarkup, "Required") {
		t.Fatalf("labeledInput markup = %q", inputMarkup)
	}
	textareaMarkup, err := ui.RenderToString(labeledTextarea("notes", "Notes", "Need pricing", ""))
	if err != nil {
		t.Fatalf("RenderToString(labeledTextarea) error = %v", err)
	}
	if !strings.Contains(textareaMarkup, "Need pricing") {
		t.Fatalf("labeledTextarea markup = %q", textareaMarkup)
	}
	timelineMarkup, err := ui.RenderToString(timelineField("quarter", "Choose one"))
	if err != nil {
		t.Fatalf("RenderToString(timelineField) error = %v", err)
	}
	if !strings.Contains(timelineMarkup, `selected value="quarter"`) || !strings.Contains(timelineMarkup, "Choose one") {
		t.Fatalf("timelineField markup = %q", timelineMarkup)
	}
	fileMarkup, err := ui.RenderToString(fileField("Upload failed"))
	if err != nil {
		t.Fatalf("RenderToString(fileField) error = %v", err)
	}
	if !strings.Contains(fileMarkup, "image/png,image/jpeg") || !strings.Contains(fileMarkup, "Upload failed") {
		t.Fatalf("fileField markup = %q", fileMarkup)
	}

	document, err := renderDocument(pageState{
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
	if err != nil {
		t.Fatalf("renderDocument() error = %v", err)
	}
	for _, expected := range []string{"<!DOCTYPE html>", "SSR Secure Forms Demo", "csrf_token", "Request pricing", "Upload asset"} {
		if !strings.Contains(document, expected) {
			t.Fatalf("renderDocument() missing %q\n%s", expected, document)
		}
	}
}

func TestCSRFAndOriginHelpers(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "example.com"
	req.AddCookie(&http.Cookie{Name: secureFormsCSRFCookie, Value: "existing-token"})
	res := httptest.NewRecorder()
	token := ensureCSRFCookie(res, req)
	if token != "existing-token" {
		t.Fatalf("ensureCSRFCookie(existing) = %q, want existing-token", token)
	}
	if len(res.Result().Cookies()) != 0 {
		t.Fatalf("ensureCSRFCookie(existing) unexpectedly set cookies: %+v", res.Result().Cookies())
	}

	freshReq := httptest.NewRequest(http.MethodGet, "/", nil)
	freshReq.Host = "example.com"
	freshRes := httptest.NewRecorder()
	freshToken := ensureCSRFCookie(freshRes, freshReq)
	if strings.TrimSpace(freshToken) == "" {
		t.Fatal("ensureCSRFCookie(fresh) returned empty token")
	}
	cookies := freshRes.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != secureFormsCSRFCookie || cookies[0].Value != freshToken || cookies[0].Path != "/" || !cookies[0].HttpOnly {
		t.Fatalf("ensureCSRFCookie(fresh) cookies = %+v, token = %q", cookies, freshToken)
	}

	csrfToken, csrfCookie := loadCSRF(t, server)
	validForm := strings.NewReader("csrf_token=" + csrfToken + "&name=Ada")
	validReq := httptest.NewRequest(http.MethodPost, "/quote", validForm)
	validReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	validReq.Header.Set("Origin", "http://example.com")
	validReq.Host = "example.com"
	validReq.AddCookie(csrfCookie)
	if !validateCSRFFromForm(validReq) {
		t.Fatal("validateCSRFFromForm(valid form) = false, want true")
	}

	invalidOriginReq := httptest.NewRequest(http.MethodPost, "/quote", strings.NewReader("csrf_token="+csrfToken))
	invalidOriginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	invalidOriginReq.Header.Set("Origin", "https://example.com")
	invalidOriginReq.Host = "example.com"
	invalidOriginReq.AddCookie(csrfCookie)
	if validateCSRFFromForm(invalidOriginReq) {
		t.Fatal("validateCSRFFromForm(cross origin) = true, want false")
	}

	badTokenReq := httptest.NewRequest(http.MethodPost, "/quote", strings.NewReader("csrf_token=wrong"))
	badTokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	badTokenReq.Header.Set("Origin", "http://example.com")
	badTokenReq.Host = "example.com"
	badTokenReq.AddCookie(csrfCookie)
	if validateCSRFFromForm(badTokenReq) {
		t.Fatal("validateCSRFFromForm(bad token) = true, want false")
	}

	multipartBody := &bytes.Buffer{}
	writer := multipart.NewWriter(multipartBody)
	_ = writer.WriteField("csrf_token", csrfToken)
	_ = writer.Close()
	multipartReq := httptest.NewRequest(http.MethodPost, "/upload", multipartBody)
	multipartReq.Header.Set("Content-Type", writer.FormDataContentType())
	multipartReq.Header.Set("Origin", "http://example.com")
	multipartReq.Host = "example.com"
	multipartReq.AddCookie(csrfCookie)
	if !validateCSRFFromForm(multipartReq) {
		t.Fatal("validateCSRFFromForm(multipart) = false, want true")
	}

	noOriginReq := httptest.NewRequest(http.MethodPost, "http://example.com/quote", nil)
	noOriginReq.Host = "example.com"
	if sameOriginRequest(noOriginReq) {
		t.Fatal("sameOriginRequest(no headers) = true, want false")
	}
	refererReq := httptest.NewRequest(http.MethodPost, "http://example.com/quote", nil)
	refererReq.Host = "example.com"
	refererReq.Header.Set("Referer", "http://example.com/form")
	if !sameOriginRequest(refererReq) {
		t.Fatal("sameOriginRequest(referer) = false, want true")
	}
	crossReq := httptest.NewRequest(http.MethodPost, "http://example.com/quote", nil)
	crossReq.Host = "example.com"
	crossReq.Header.Set("Origin", "http://evil.example")
	if sameOriginRequest(crossReq) {
		t.Fatal("sameOriginRequest(cross origin) = true, want false")
	}

	httpsReq := httptest.NewRequest(http.MethodPost, "https://example.com/quote", nil)
	httpsReq.Host = "example.com"
	httpsReq.TLS = &tls.ConnectionState{}
	if sameOrigin("https://example.com/path", httpsReq) != true {
		t.Fatal("sameOrigin(https) = false, want true")
	}
	if sameOrigin("notaurl", httpsReq) {
		t.Fatal("sameOrigin(invalid) = true, want false")
	}
	if sameOrigin("/relative", httpsReq) {
		t.Fatal("sameOrigin(relative) = true, want false")
	}
}

func TestRoutesAndRedirectHelpers(t *testing.T) {
	server := newTestServer()
	handler := server.routes()

	index := httptest.NewRecorder()
	handler.ServeHTTP(index, httptest.NewRequest(http.MethodGet, "/", nil))
	if index.Code != http.StatusOK {
		t.Fatalf("routes GET / = %d, want 200", index.Code)
	}

	notAllowed := httptest.NewRecorder()
	handler.ServeHTTP(notAllowed, httptest.NewRequest(http.MethodGet, "/quote", nil))
	if notAllowed.Code != http.StatusOK || !strings.Contains(notAllowed.Body.String(), "SSR Secure Forms") {
		t.Fatalf("routes GET /quote = code %d body %q, want index fallback", notAllowed.Code, notAllowed.Body.String())
	}

	redirectReq := httptest.NewRequest(http.MethodPost, "/quote", nil)
	redirectRes := httptest.NewRecorder()
	redirectWithNotice(redirectRes, redirectReq, "Saved with redirect")
	if redirectRes.Code != http.StatusSeeOther || !strings.Contains(redirectRes.Header().Get("Location"), "notice=Saved+with+redirect") {
		t.Fatalf("redirectWithNotice() = code %d location %q", redirectRes.Code, redirectRes.Header().Get("Location"))
	}

	report := secureFormsRequestReport("handleQuote", "/quote", http.ErrMissingFile, "upload failed", "retry")
	if report.Code != "GWC-EXAMPLE-SERVER-REQUEST" || report.Path != "/quote" || !strings.Contains(report.Headline, "handleQuote") {
		t.Fatalf("secureFormsRequestReport() = %+v", report)
	}
}

func TestUploadErrorBranches(t *testing.T) {
	server := newTestServer()
	csrfToken, csrfCookie := loadCSRF(t, server)

	urlEncodedReq := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("csrf_token="+csrfToken+"&label=Board"))
	urlEncodedReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	urlEncodedReq.Header.Set("Origin", "http://example.com")
	urlEncodedReq.Host = "example.com"
	urlEncodedReq.AddCookie(csrfCookie)
	urlEncodedRes := httptest.NewRecorder()
	server.handleUpload(urlEncodedRes, urlEncodedReq)
	if urlEncodedRes.Code != http.StatusBadRequest {
		t.Fatalf("handleUpload(urlencoded) = %d, want 400", urlEncodedRes.Code)
	}
	if !strings.Contains(strings.ToLower(urlEncodedRes.Body.String()), "multipart") {
		t.Fatalf("handleUpload(urlencoded) body = %q, want multipart parse failure", urlEncodedRes.Body.String())
	}

	missingCSRFReq := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("label=Board"))
	missingCSRFReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	missingCSRFReq.Header.Set("Origin", "http://example.com")
	missingCSRFReq.Host = "example.com"
	missingCSRFRes := httptest.NewRecorder()
	server.handleUpload(missingCSRFRes, missingCSRFReq)
	if missingCSRFRes.Code != http.StatusForbidden {
		t.Fatalf("handleUpload(missing csrf) = %d, want 403", missingCSRFRes.Code)
	}
	if !strings.Contains(strings.ToLower(missingCSRFRes.Body.String()), "csrf") {
		t.Fatalf("handleUpload(missing csrf) body = %q, want csrf failure", missingCSRFRes.Body.String())
	}
}
