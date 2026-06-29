//go:build !js || !wasm

package main

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"net/mail"
	"net/url"
	"strings"

	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

type quoteForm struct {
	Name     string
	Email    string
	Company  string
	Timeline string
	Notes    string
}

type uploadForm struct {
	Label string
}

type uploadedAsset struct {
	Name        string
	SizeBytes   int64
	ContentType string
}

type pageState struct {
	CSRF         string
	Notice       string
	Quote        quoteForm
	QuoteErrors  map[string]string
	Upload       uploadForm
	UploadErrors map[string]string
	Uploaded     uploadedAsset
}

func defaultQuoteForm() quoteForm {
	return quoteForm{
		Timeline: "30_days",
		Notes:    "Need a request-time rendered quote path with project timing preserved.",
	}
}

func defaultUploadForm() uploadForm {
	return uploadForm{Label: "Project board photo"}
}

func newPageState(parseCsrfToken string, parseQuery url.Values) pageState {
	parseState := pageState{
		CSRF:         parseCsrfToken,
		Notice:       strings.TrimSpace(parseQuery.Get("notice")),
		Quote:        defaultQuoteForm(),
		QuoteErrors:  map[string]string{},
		Upload:       defaultUploadForm(),
		UploadErrors: map[string]string{},
	}
	if parseAssetName := strings.TrimSpace(parseQuery.Get("asset")); parseAssetName != "" {
		parseState.Uploaded = uploadedAsset{
			Name:        parseAssetName,
			SizeBytes:   0,
			ContentType: strings.TrimSpace(parseQuery.Get("type")),
		}
	}
	return parseState
}

func validateQuote(parseInput quoteForm) map[string]string {
	parseErrors := map[string]string{}
	if strings.TrimSpace(parseInput.Name) == "" {
		parseErrors["name"] = "Name is required."
	}
	if strings.TrimSpace(parseInput.Company) == "" {
		parseErrors["company"] = "Company is required."
	}
	if parseEmail := strings.TrimSpace(parseInput.Email); parseEmail == "" {
		parseErrors["email"] = "Email is required."
	} else if _, parseErr := mail.ParseAddress(parseEmail); parseErr != nil {
		parseErrors["email"] = "Enter a valid email address."
	}
	if parseTimeline := strings.TrimSpace(parseInput.Timeline); parseTimeline != "14_days" && parseTimeline != "30_days" && parseTimeline != "quarter" {
		parseErrors["timeline"] = "Choose a supported timeline."
	}
	return parseErrors
}

func validateUpload(parseInput uploadForm, parseHeader *multipart.FileHeader, parseSize int64, parseContentType string) map[string]string {
	parseErrors := map[string]string{}
	if strings.TrimSpace(parseInput.Label) == "" {
		parseErrors["label"] = "Asset label is required."
	}
	if parseHeader == nil {
		parseErrors["asset"] = "Choose a PNG or JPEG file."
		return parseErrors
	}
	if parseSize <= 0 {
		parseErrors["asset"] = "Upload a non-empty file."
		return parseErrors
	}
	if parseSize > maxUploadBytes {
		parseErrors["asset"] = "Upload a file that is 2 MB or smaller."
		return parseErrors
	}
	if parseContentType != "image/png" && parseContentType != "image/jpeg" {
		parseErrors["asset"] = "Upload a PNG or JPEG file."
	}
	return parseErrors
}

func renderPage(parseState pageState) ui.Node {
	parseFieldName, parseFieldValue := ui.NewCSRFToken(parseState.CSRF).FormField()
	return html.Main(html.Props{Class: "mx-auto flex min-h-screen max-w-6xl flex-col gap-8 bg-stone-950 px-6 py-10 text-stone-100"},
		html.Section(html.Props{Class: "grid gap-4"},
			html.P(html.Props{Class: "text-sm uppercase tracking-[0.22em] text-emerald-300"}, html.Text("SSR Secure Forms")),
			html.H1(html.Props{Class: "text-4xl font-semibold"}, html.Text("Server-validated forms with CSRF and multipart uploads")),
			html.P(html.Props{Class: "max-w-3xl text-sm text-stone-300"}, html.Text("This page keeps the forms request-time rendered, preserves submitted values on validation failure, and uses 303 redirects after successful quote and upload submissions.")),
			renderNotice(parseState),
		),
		html.Div(html.Props{Class: "grid gap-6 lg:grid-cols-2"},
			html.Section(html.Props{Class: "grid gap-4 rounded-3xl border border-white/10 bg-white/5 p-6"},
				html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text("Quote Request")),
				html.P(html.Props{Class: "text-sm text-stone-300"}, html.Text("Demonstrates form defaults, CSRF token injection, server validation, and redirect-after-submit semantics.")),
				html.Form(html.Props{Action: "/quote", Method: http.MethodPost, Class: "grid gap-4"},
					html.HiddenInput(parseFieldName, parseFieldValue),
					labeledInput("name", "Name", parseState.Quote.Name, parseState.QuoteErrors["name"]),
					labeledInput("email", "Email", parseState.Quote.Email, parseState.QuoteErrors["email"]),
					labeledInput("company", "Company", parseState.Quote.Company, parseState.QuoteErrors["company"]),
					timelineField(parseState.Quote.Timeline, parseState.QuoteErrors["timeline"]),
					labeledTextarea("notes", "Notes", parseState.Quote.Notes, parseState.QuoteErrors["notes"]),
					html.Button(html.Props{Type: "submit", Class: submitButtonClass}, html.Text("Request pricing")),
				),
			),
			html.Section(html.Props{Class: "grid gap-4 rounded-3xl border border-white/10 bg-white/5 p-6"},
				html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.22em] text-amber-300"}, html.Text("Secure Upload")),
				html.P(html.Props{Class: "text-sm text-stone-300"}, html.Text("Demonstrates `multipart/form-data`, server-side file validation, CSRF pairing, and redirect-after-submit success handling.")),
				renderUploadedSummary(parseState.Uploaded),
				html.Form(html.Props{Action: "/upload", Method: http.MethodPost, EncType: "multipart/form-data", Class: "grid gap-4"},
					html.HiddenInput(parseFieldName, parseFieldValue),
					labeledInput("label", "Asset Label", parseState.Upload.Label, parseState.UploadErrors["label"]),
					fileField(parseState.UploadErrors["asset"]),
					html.Button(html.Props{Type: "submit", Class: submitButtonClass}, html.Text("Upload asset")),
				),
			),
		),
	)
}

func renderDocument(parseState pageState) (string, error) {
	parseBody, parseErr := ui.RenderToString(renderPage(parseState))
	if parseErr != nil {
		return "", parseErr
	}
	return fmt.Sprintf("<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"><title>SSR Secure Forms Demo</title></head><body style=\"margin:0;background:#0c0a09;font-family:Georgia,serif;\">%s</body></html>", parseBody), nil
}

func renderNotice(parseState pageState) ui.Node {
	if strings.TrimSpace(parseState.Notice) == "" {
		return nil
	}
	return html.P(html.Props{Class: "rounded-2xl border border-emerald-400/35 bg-emerald-400/10 px-4 py-3 text-sm text-emerald-100"}, html.Text(parseState.Notice))
}

func renderUploadedSummary(parseAsset uploadedAsset) ui.Node {
	if strings.TrimSpace(parseAsset.Name) == "" {
		return html.P(html.Props{Class: "rounded-2xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-stone-400"}, html.Text("No file has been accepted yet. Successful uploads redirect back here with a notice and summary."))
	}
	parseSummary := parseAsset.Name
	if parseAsset.ContentType != "" {
		parseSummary += " · " + parseAsset.ContentType
	}
	if parseAsset.SizeBytes > 0 {
		parseSummary += fmt.Sprintf(" · %d bytes", parseAsset.SizeBytes)
	}
	return html.P(html.Props{Class: "rounded-2xl border border-amber-400/35 bg-amber-400/10 px-4 py-3 text-sm text-amber-100"}, html.Text("Last accepted asset: "+parseSummary))
}

func labeledInput(parseName string, parseLabel string, parseValue string, parseFieldError string) ui.Node {
	parseNodes := []ui.Node{
		html.Label(html.Props{For: parseName, Class: "text-sm font-semibold text-stone-200"}, html.Text(parseLabel)),
		html.Input(html.Props{ID: parseName, Name: parseName, Value: parseValue, Class: inputClass}),
	}
	if strings.TrimSpace(parseFieldError) != "" {
		parseNodes = append(parseNodes, html.P(html.Props{Class: errorClass}, html.Text(parseFieldError)))
	}
	return html.Div(html.Props{Class: "grid gap-2"}, parseNodes...)
}

func labeledTextarea(parseName string, parseLabel string, parseValue string, parseFieldError string) ui.Node {
	parseNodes := []ui.Node{
		html.Label(html.Props{For: parseName, Class: "text-sm font-semibold text-stone-200"}, html.Text(parseLabel)),
		html.Textarea(html.Props{ID: parseName, Name: parseName, Rows: 4, Class: inputClass}, html.Text(parseValue)),
	}
	if strings.TrimSpace(parseFieldError) != "" {
		parseNodes = append(parseNodes, html.P(html.Props{Class: errorClass}, html.Text(parseFieldError)))
	}
	return html.Div(html.Props{Class: "grid gap-2"}, parseNodes...)
}

func timelineField(parseSelected string, parseFieldError string) ui.Node {
	parseNodes := []ui.Node{
		html.Label(html.Props{For: "timeline", Class: "text-sm font-semibold text-stone-200"}, html.Text("Project timeline")),
		html.Select(html.Props{ID: "timeline", Name: "timeline", Class: inputClass},
			html.Option(html.Props{Value: "14_days", Selected: parseSelected == "14_days"}, html.Text("Need pricing in 14 days")),
			html.Option(html.Props{Value: "30_days", Selected: parseSelected == "30_days"}, html.Text("Need pricing in 30 days")),
			html.Option(html.Props{Value: "quarter", Selected: parseSelected == "quarter"}, html.Text("Pricing request for this quarter")),
		),
	}
	if strings.TrimSpace(parseFieldError) != "" {
		parseNodes = append(parseNodes, html.P(html.Props{Class: errorClass}, html.Text(parseFieldError)))
	}
	return html.Div(html.Props{Class: "grid gap-2"}, parseNodes...)
}

func fileField(parseFieldError string) ui.Node {
	parseNodes := []ui.Node{
		html.Label(html.Props{For: "asset", Class: "text-sm font-semibold text-stone-200"}, html.Text("Reference image")),
		html.Input(html.Props{ID: "asset", Name: "asset", Type: "file", Accept: "image/png,image/jpeg", Class: inputClass}),
		html.P(html.Props{Class: "text-xs text-stone-400"}, html.Text("Accepted: PNG or JPEG up to 2 MB.")),
	}
	if strings.TrimSpace(parseFieldError) != "" {
		parseNodes = append(parseNodes, html.P(html.Props{Class: errorClass}, html.Text(parseFieldError)))
	}
	return html.Div(html.Props{Class: "grid gap-2"}, parseNodes...)
}

const (
	maxUploadBytes    = 2 << 20
	inputClass        = "rounded-2xl border border-white/10 bg-stone-900/80 px-4 py-3 text-sm text-stone-100"
	errorClass        = "text-xs font-semibold text-rose-300"
	submitButtonClass = "rounded-full border border-white/10 bg-emerald-300 px-4 py-3 text-sm font-semibold text-stone-950"
)
