//go:build !js || !wasm
// +build !js !wasm

package main

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"net/mail"
	"net/url"
	"strings"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
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

func newPageState(csrfToken string, query url.Values) pageState {
	state := pageState{
		CSRF:         csrfToken,
		Notice:       strings.TrimSpace(query.Get("notice")),
		Quote:        defaultQuoteForm(),
		QuoteErrors:  map[string]string{},
		Upload:       defaultUploadForm(),
		UploadErrors: map[string]string{},
	}
	if assetName := strings.TrimSpace(query.Get("asset")); assetName != "" {
		state.Uploaded = uploadedAsset{
			Name:        assetName,
			SizeBytes:   0,
			ContentType: strings.TrimSpace(query.Get("type")),
		}
	}
	return state
}

func validateQuote(input quoteForm) map[string]string {
	errors := map[string]string{}
	if strings.TrimSpace(input.Name) == "" {
		errors["name"] = "Name is required."
	}
	if strings.TrimSpace(input.Company) == "" {
		errors["company"] = "Company is required."
	}
	if email := strings.TrimSpace(input.Email); email == "" {
		errors["email"] = "Email is required."
	} else if _, err := mail.ParseAddress(email); err != nil {
		errors["email"] = "Enter a valid email address."
	}
	if timeline := strings.TrimSpace(input.Timeline); timeline != "14_days" && timeline != "30_days" && timeline != "quarter" {
		errors["timeline"] = "Choose a supported timeline."
	}
	return errors
}

func validateUpload(input uploadForm, header *multipart.FileHeader, size int64, contentType string) map[string]string {
	errors := map[string]string{}
	if strings.TrimSpace(input.Label) == "" {
		errors["label"] = "Asset label is required."
	}
	if header == nil {
		errors["asset"] = "Choose a PNG or JPEG file."
		return errors
	}
	if size <= 0 {
		errors["asset"] = "Upload a non-empty file."
		return errors
	}
	if size > maxUploadBytes {
		errors["asset"] = "Upload a file that is 2 MB or smaller."
		return errors
	}
	if contentType != "image/png" && contentType != "image/jpeg" {
		errors["asset"] = "Upload a PNG or JPEG file."
	}
	return errors
}

func renderPage(state pageState) ui.Node {
	fieldName, fieldValue := ui.NewCSRFToken(state.CSRF).FormField()
	return html.Main(html.Props{Class: "mx-auto flex min-h-screen max-w-6xl flex-col gap-8 bg-stone-950 px-6 py-10 text-stone-100"},
		html.Section(html.Props{Class: "grid gap-4"},
			html.P(html.Props{Class: "text-sm uppercase tracking-[0.22em] text-emerald-300"}, html.Text("SSR Secure Forms")),
			html.H1(html.Props{Class: "text-4xl font-semibold"}, html.Text("Server-validated forms with CSRF and multipart uploads")),
			html.P(html.Props{Class: "max-w-3xl text-sm text-stone-300"}, html.Text("This page keeps the forms request-time rendered, preserves submitted values on validation failure, and uses 303 redirects after successful quote and upload submissions.")),
			renderNotice(state),
		),
		html.Div(html.Props{Class: "grid gap-6 lg:grid-cols-2"},
			html.Section(html.Props{Class: "grid gap-4 rounded-3xl border border-white/10 bg-white/5 p-6"},
				html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.22em] text-cyan-300"}, html.Text("Quote Request")),
				html.P(html.Props{Class: "text-sm text-stone-300"}, html.Text("Demonstrates form defaults, CSRF token injection, server validation, and redirect-after-submit semantics.")),
				html.Form(html.Props{Action: "/quote", Method: http.MethodPost, Class: "grid gap-4"},
					html.HiddenInput(fieldName, fieldValue),
					labeledInput("name", "Name", state.Quote.Name, state.QuoteErrors["name"]),
					labeledInput("email", "Email", state.Quote.Email, state.QuoteErrors["email"]),
					labeledInput("company", "Company", state.Quote.Company, state.QuoteErrors["company"]),
					timelineField(state.Quote.Timeline, state.QuoteErrors["timeline"]),
					labeledTextarea("notes", "Notes", state.Quote.Notes, state.QuoteErrors["notes"]),
					html.Button(html.Props{Type: "submit", Class: submitButtonClass}, html.Text("Request pricing")),
				),
			),
			html.Section(html.Props{Class: "grid gap-4 rounded-3xl border border-white/10 bg-white/5 p-6"},
				html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.22em] text-amber-300"}, html.Text("Secure Upload")),
				html.P(html.Props{Class: "text-sm text-stone-300"}, html.Text("Demonstrates `multipart/form-data`, server-side file validation, CSRF pairing, and redirect-after-submit success handling.")),
				renderUploadedSummary(state.Uploaded),
				html.Form(html.Props{Action: "/upload", Method: http.MethodPost, EncType: "multipart/form-data", Class: "grid gap-4"},
					html.HiddenInput(fieldName, fieldValue),
					labeledInput("label", "Asset Label", state.Upload.Label, state.UploadErrors["label"]),
					fileField(state.UploadErrors["asset"]),
					html.Button(html.Props{Type: "submit", Class: submitButtonClass}, html.Text("Upload asset")),
				),
			),
		),
	)
}

func renderDocument(state pageState) (string, error) {
	body, err := ui.RenderToString(renderPage(state))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"><title>SSR Secure Forms Demo</title></head><body style=\"margin:0;background:#0c0a09;font-family:Georgia,serif;\">%s</body></html>", body), nil
}

func renderNotice(state pageState) ui.Node {
	if strings.TrimSpace(state.Notice) == "" {
		return nil
	}
	return html.P(html.Props{Class: "rounded-2xl border border-emerald-400/35 bg-emerald-400/10 px-4 py-3 text-sm text-emerald-100"}, html.Text(state.Notice))
}

func renderUploadedSummary(asset uploadedAsset) ui.Node {
	if strings.TrimSpace(asset.Name) == "" {
		return html.P(html.Props{Class: "rounded-2xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-stone-400"}, html.Text("No file has been accepted yet. Successful uploads redirect back here with a notice and summary."))
	}
	summary := asset.Name
	if asset.ContentType != "" {
		summary += " · " + asset.ContentType
	}
	if asset.SizeBytes > 0 {
		summary += fmt.Sprintf(" · %d bytes", asset.SizeBytes)
	}
	return html.P(html.Props{Class: "rounded-2xl border border-amber-400/35 bg-amber-400/10 px-4 py-3 text-sm text-amber-100"}, html.Text("Last accepted asset: "+summary))
}

func labeledInput(name string, label string, value string, fieldError string) ui.Node {
	nodes := []ui.Node{
		html.Label(html.Props{For: name, Class: "text-sm font-semibold text-stone-200"}, html.Text(label)),
		html.Input(html.Props{ID: name, Name: name, Value: value, Class: inputClass}),
	}
	if strings.TrimSpace(fieldError) != "" {
		nodes = append(nodes, html.P(html.Props{Class: errorClass}, html.Text(fieldError)))
	}
	return html.Div(html.Props{Class: "grid gap-2"}, nodes...)
}

func labeledTextarea(name string, label string, value string, fieldError string) ui.Node {
	nodes := []ui.Node{
		html.Label(html.Props{For: name, Class: "text-sm font-semibold text-stone-200"}, html.Text(label)),
		html.Textarea(html.Props{ID: name, Name: name, Rows: 4, Class: inputClass}, html.Text(value)),
	}
	if strings.TrimSpace(fieldError) != "" {
		nodes = append(nodes, html.P(html.Props{Class: errorClass}, html.Text(fieldError)))
	}
	return html.Div(html.Props{Class: "grid gap-2"}, nodes...)
}

func timelineField(selected string, fieldError string) ui.Node {
	nodes := []ui.Node{
		html.Label(html.Props{For: "timeline", Class: "text-sm font-semibold text-stone-200"}, html.Text("Project timeline")),
		html.Select(html.Props{ID: "timeline", Name: "timeline", Class: inputClass},
			html.Option(html.Props{Value: "14_days", Selected: selected == "14_days"}, html.Text("Need pricing in 14 days")),
			html.Option(html.Props{Value: "30_days", Selected: selected == "30_days"}, html.Text("Need pricing in 30 days")),
			html.Option(html.Props{Value: "quarter", Selected: selected == "quarter"}, html.Text("Pricing request for this quarter")),
		),
	}
	if strings.TrimSpace(fieldError) != "" {
		nodes = append(nodes, html.P(html.Props{Class: errorClass}, html.Text(fieldError)))
	}
	return html.Div(html.Props{Class: "grid gap-2"}, nodes...)
}

func fileField(fieldError string) ui.Node {
	nodes := []ui.Node{
		html.Label(html.Props{For: "asset", Class: "text-sm font-semibold text-stone-200"}, html.Text("Reference image")),
		html.Input(html.Props{ID: "asset", Name: "asset", Type: "file", Accept: "image/png,image/jpeg", Class: inputClass}),
		html.P(html.Props{Class: "text-xs text-stone-400"}, html.Text("Accepted: PNG or JPEG up to 2 MB.")),
	}
	if strings.TrimSpace(fieldError) != "" {
		nodes = append(nodes, html.P(html.Props{Class: errorClass}, html.Text(fieldError)))
	}
	return html.Div(html.Props{Class: "grid gap-2"}, nodes...)
}

const (
	maxUploadBytes    = 2 << 20
	inputClass        = "rounded-2xl border border-white/10 bg-stone-900/80 px-4 py-3 text-sm text-stone-100"
	errorClass        = "text-xs font-semibold text-rose-300"
	submitButtonClass = "rounded-full border border-white/10 bg-emerald-300 px-4 py-3 text-sm font-semibold text-stone-950"
)
