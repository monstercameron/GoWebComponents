//go:build js && wasm

package main

import (
	"fmt"
	"strings"
	"syscall/js"

	. "github.com/monstercameron/GoWebComponents/v5/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

func initialPlaygroundSource() string {
	parseSource, parseFound, parseErr := decodePlaygroundSource(getPageQueryValue(playgroundSnippetParam))
	if parseFound && parseErr == nil {
		return parseSource
	}
	return defaultPlaygroundSnippet
}

func currentPlaygroundHref() string {
	parseWindow := js.Global().Get("window")
	if parseWindow.IsUndefined() || parseWindow.IsNull() {
		return "http://localhost/"
	}
	parseLocation := parseWindow.Get("location")
	if parseLocation.IsUndefined() || parseLocation.IsNull() {
		return "http://localhost/"
	}
	parseHref := parseLocation.Get("href")
	if parseHref.IsUndefined() || parseHref.IsNull() {
		return "http://localhost/"
	}
	return parseHref.String()
}

func replacePlaygroundSourceURL(parseSource string) string {
	parseShareURL := buildPlaygroundShareURL(currentPlaygroundHref(), parseSource)
	if parseShareURL == "" {
		return parseShareURL
	}
	parseWindow := js.Global().Get("window")
	if parseWindow.IsUndefined() || parseWindow.IsNull() {
		return parseShareURL
	}
	parseHistory := parseWindow.Get("history")
	if parseHistory.IsUndefined() || parseHistory.IsNull() {
		return parseShareURL
	}
	parseReplaceState := parseHistory.Get("replaceState")
	if parseReplaceState.Type() == js.TypeFunction {
		parseHistory.Call("replaceState", nil, "", parseShareURL)
	}
	return parseShareURL
}

func copyPlaygroundShareURL(parseShareURL string) bool {
	parseNavigator := js.Global().Get("navigator")
	if parseNavigator.IsUndefined() || parseNavigator.IsNull() {
		return false
	}
	parseClipboard := parseNavigator.Get("clipboard")
	if parseClipboard.IsUndefined() || parseClipboard.IsNull() {
		return false
	}
	parseWriteText := parseClipboard.Get("writeText")
	if parseWriteText.Type() != js.TypeFunction {
		return false
	}
	parseWriteText.Invoke(parseShareURL)
	return true
}

func playgroundURLForSource(parseSource string) string {
	if strings.TrimSpace(parseSource) == "" {
		return "#playground"
	}
	parseShareURL := buildPlaygroundShareURL(currentPlaygroundHref(), parseSource)
	if strings.TrimSpace(parseShareURL) == "" {
		return "#playground"
	}
	return parseShareURL
}

func renderPlaygroundDiagnostic(parseDiagnostic playgroundDiagnostic) ui.Node {
	parseLocation := "source"
	if parseDiagnostic.Line > 0 {
		parseLocation = fmt.Sprintf("line %d", parseDiagnostic.Line)
		if parseDiagnostic.Column > 0 {
			parseLocation = fmt.Sprintf("%s:%d", parseLocation, parseDiagnostic.Column)
		}
	}
	return Div(
		ID("playground-diagnostics"),
		Attr("data-diagnostic-code", parseDiagnostic.Code),
		Attr("data-diagnostic-severity", parseDiagnostic.Severity),
		Attr("aria-live", "polite"),
		ClassStr("rounded-[20px] border border-rose-400/25 bg-rose-500/10 p-4 text-sm text-rose-50"),
		Div(ClassStr("flex flex-wrap items-center justify-between gap-2"),
			Span(ClassStr("rounded-full border border-rose-300/30 bg-rose-300/10 px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.16em] text-rose-100"), Text(parseDiagnostic.Code)),
			Span(ClassStr("text-xs uppercase tracking-[0.16em] text-rose-200/80"), Text(parseLocation)),
		),
		P(ClassStr("mt-3 leading-6"), Text(parseDiagnostic.Message)),
	)
}

func renderPlaygroundPreview(parseResult playgroundCompileResult) ui.Node {
	if !parseResult.OK {
		return renderPlaygroundDiagnostic(parseResult.Diagnostic)
	}
	return Iframe(
		ID("playground-sandbox"),
		Title("GoWebComponents playground preview"),
		Attr("sandbox", ""),
		Attr("srcdoc", playgroundSandboxHTML(parseResult)),
		Attr("data-playground-sandbox", "true"),
		ClassStr("h-[340px] w-full rounded-[20px] border border-white/10 bg-white shadow-2xl shadow-black/25"),
	)
}

func renderPlaygroundPanel() ui.Node {
	parseInitialSource := initialPlaygroundSource()
	parseSource := ui.UseState(parseInitialSource)
	parseCompiledSource := ui.UseState(parseInitialSource)
	parseShareStatus := ui.UseState("Share URL ready")
	parseCompileResult := ui.UseMemo(func() playgroundCompileResult {
		return compilePlaygroundSnippet(parseCompiledSource.Get())
	}, parseCompiledSource.Get())
	parseShareURL := ui.UseMemo(func() string {
		return buildPlaygroundShareURL(currentPlaygroundHref(), parseSource.Get())
	}, parseSource.Get())
	parseUpdateSource := ui.UseEvent(func(parseEvent ui.InputEvent) {
		parseNextSource := parseEvent.GetValue()
		parseSource.Set(parseNextSource)
		replacePlaygroundSourceURL(parseNextSource)
		parseShareStatus.Set("Share URL updated; run to refresh preview")
	})
	parseRun := ui.UseEvent(func() {
		parseCompiledSource.Set(parseSource.Get())
		parseShareStatus.Set("Snippet compiled into the sandbox")
	})
	parseReset := ui.UseEvent(func() {
		parseSource.Set(defaultPlaygroundSnippet)
		parseCompiledSource.Set(defaultPlaygroundSnippet)
		replacePlaygroundSourceURL(defaultPlaygroundSnippet)
		parseShareStatus.Set("Default snippet restored")
	})
	parseCopy := ui.UseEvent(func() {
		if copyPlaygroundShareURL(parseShareURL) {
			parseShareStatus.Set("Share URL copied")
			return
		}
		parseShareStatus.Set("Share URL is selected below")
	})
	parseCompileTone := "border-rose-400/25 bg-rose-400/10 text-rose-100"
	parseCompileLabel := "Compile error"
	if parseCompileResult.OK {
		parseCompileTone = "border-emerald-400/25 bg-emerald-400/10 text-emerald-100"
		parseCompileLabel = "Compiled"
	}

	return Section(ID("playground"), ClassStr("mt-2 grid gap-2 rounded-[24px] border border-white/10 bg-slate-950/45 p-3 shadow-2xl shadow-black/20 lg:grid-cols-[minmax(0,1fr)_minmax(360px,0.85fr)]"),
		Div(ClassStr("min-w-0"),
			Div(ClassStr("flex flex-wrap items-center justify-between gap-3"),
				Div(
					Div(ClassStr("text-xs uppercase tracking-[0.18em] text-cyan-200"), Text("Snippet playground")),
					H2(ClassStr("mt-1 text-xl font-semibold tracking-tight text-white"), Text("Edit, compile, share")),
				),
				Div(ClassStr("flex flex-wrap gap-2"),
					Button(ID("playground-run"), Type("button"), OnClick(parseRun), ClassStr("rounded-xl border border-emerald-300/25 bg-emerald-400/15 px-3 py-2 text-xs font-medium text-emerald-100 transition hover:bg-emerald-400/20"), Text("Run")),
					Button(Type("button"), OnClick(parseReset), ClassStr("rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-xs font-medium text-slate-200 transition hover:bg-white/10"), Text("Reset")),
					Button(ID("playground-copy"), Type("button"), OnClick(parseCopy), ClassStr("rounded-xl border border-cyan-300/25 bg-cyan-400/15 px-3 py-2 text-xs font-medium text-cyan-100 transition hover:bg-cyan-400/20"), Text("Copy URL")),
				),
			),
			Textarea(
				ID("playground-source"),
				Value(parseSource.Get()),
				OnInput(parseUpdateSource),
				Rows(17),
				Attr("spellcheck", "false"),
				Attr("data-playground-editor", "true"),
				ClassStr("mt-3 min-h-[340px] w-full resize-y rounded-[18px] border border-white/10 bg-[#050b14] p-4 font-mono text-sm leading-6 text-cyan-50 outline-none shadow-inner shadow-black/25 placeholder:text-slate-600 focus:border-cyan-300/40"),
			),
			Div(ClassStr("mt-2 flex min-w-0 items-center gap-2 rounded-[16px] border border-white/10 bg-black/20 px-3 py-2 text-xs text-slate-300"),
				Span(ClassStr("shrink-0 text-slate-500"), Text("Share")),
				A(ID("playground-share"), Href(parseShareURL), ClassStr("min-w-0 truncate text-cyan-200 underline decoration-cyan-300/40 underline-offset-4"), Text(parseShareURL)),
				Span(ID("playground-status"), ClassStr("shrink-0 text-slate-500"), Attr("aria-live", "polite"), Text(parseShareStatus.Get())),
			),
		),
		Div(ClassStr("min-w-0"),
			Div(ClassStr("mb-3 flex flex-wrap items-center justify-between gap-2"),
				Div(
					Div(ClassStr("text-xs uppercase tracking-[0.18em] text-slate-500"), Text("Sandbox preview")),
					Div(ClassStr("text-sm text-slate-300"), Text("Isolated iframe output")),
				),
				Span(ID("playground-compile-status"), Attr("data-playground-status", parseCompileLabel), Attr("aria-live", "polite"), ClassStr("rounded-full border px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.16em] "+parseCompileTone), Text(parseCompileLabel)),
			),
			Div(ID("playground-preview"), renderPlaygroundPreview(parseCompileResult)),
		),
	)
}
