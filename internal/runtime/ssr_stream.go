package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"strings"
)

const (
	// SSRStreamChunkShell identifies the initial shell chunk.
	SSRStreamChunkShell = "shell"
	// SSRStreamChunkBoundary identifies an out-of-order async boundary chunk.
	SSRStreamChunkBoundary = "boundary"
)

// SSRStreamChunk describes one emitted streaming SSR chunk.
type SSRStreamChunk struct {
	Kind       string
	BoundaryID string
	HTML       string
	Err        error
}

// SSRStreamOptions configures streaming SSR output.
type SSRStreamOptions struct {
	BoundaryIDPrefix      string
	DisableBoundaryScript bool
	ScriptNonce           string
	OnChunk               func(SSRStreamChunk)
	Flush                 func()
	// InitialAtoms seeds this stream's per-render atom scope before the shell walk
	// starts, so request state is an INPUT to the render rather than something a
	// previous request left in a shared registry. Seeded ids beat the component's
	// UseAtom initial (InitAtom is init-if-absent); ignored in the browser, where
	// SSR keeps using the live page's atoms. See ssr_atom_scope.go.
	InitialAtoms map[string]any
}

type ssrStreamPendingBoundary struct {
	id            string
	content       *Element
	suspension    *Suspension
	contextValues map[int64]any
}

type ssrStreamState struct {
	nextBoundaryID int
	pending        []ssrStreamPendingBoundary
	options        SSRStreamOptions
	// contextValues carries the inherited context (descriptor ID -> value) for the
	// subtree currently being rendered, so a streamed component's GoUseContextValue
	// resolves to the nearest provider. It is derived at each ContextProvider
	// boundary and restored when that boundary's children finish.
	contextValues map[int64]any
}

// RenderToStream writes an SSR shell immediately and then flushes async
// boundary replacement chunks as their render-time suspensions resolve.
func RenderToStream(parseCtx context.Context, parseWriter io.Writer, parseElement *Element, parseOptions SSRStreamOptions) (parseErr error) {
	if parseWriter == nil {
		return fmt.Errorf("ssr stream: writer is required")
	}
	if parseCtx == nil {
		parseCtx = context.Background()
	}

	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			if parseOriginal, parseSuppressed := finalizeUnhandledPanicContext("runtime", PanicPhaseSSR, "RenderToStream", "", nil, parseRecovered); parseSuppressed {
				parseErr = recoveredAsError(parseOriginal)
			}
		}
	}()

	// One atom scope per stream, minted here and owned by this request only. Every
	// pending boundary copies these context values, so a boundary that resolves
	// later — on its own goroutine — writes into the SAME request's registry and
	// never into a neighbouring request's. See ssr_atom_scope.go.
	parseState := &ssrStreamState{
		options:       parseOptions,
		contextValues: withSSRAtomScope(ssrHookOwnerContext(nil), ssrAtomScopeForRender(parseOptions.InitialAtoms)),
	}
	var parseShell strings.Builder
	if parseErr2 := renderElementToStreamShell(&parseShell, parseElement, parseState); parseErr2 != nil {
		return parseErr2
	}
	parseShellHTML := parseShell.String()
	if parseErr2 := writeSSRStreamChunk(parseWriter, parseOptions, SSRStreamChunk{Kind: SSRStreamChunkShell, HTML: parseShellHTML}); parseErr2 != nil {
		return parseErr2
	}
	flushSSRStream(parseWriter, parseOptions)

	if len(parseState.pending) == 0 {
		return nil
	}

	parseChunks := make(chan SSRStreamChunk, len(parseState.pending))
	for _, parseBoundary := range parseState.pending {
		parseBoundaryCopy := parseBoundary
		go func() {
			parseChunks <- safeRenderSSRStreamBoundaryChunk(parseCtx, parseBoundaryCopy, parseOptions)
		}()
	}

	for parseRemaining := len(parseState.pending); parseRemaining > 0; parseRemaining-- {
		select {
		case <-parseCtx.Done():
			return parseCtx.Err()
		case parseChunk := <-parseChunks:
			if parseChunk.Err != nil {
				return parseChunk.Err
			}
			if parseErr2 := writeSSRStreamChunk(parseWriter, parseOptions, parseChunk); parseErr2 != nil {
				return parseErr2
			}
			flushSSRStream(parseWriter, parseOptions)
		}
	}
	return nil
}

func writeSSRStreamChunk(parseWriter io.Writer, parseOptions SSRStreamOptions, parseChunk SSRStreamChunk) error {
	if _, parseErr := io.WriteString(parseWriter, parseChunk.HTML); parseErr != nil {
		return parseErr
	}
	if parseOptions.OnChunk != nil {
		parseOptions.OnChunk(parseChunk)
	}
	return nil
}

func flushSSRStream(parseWriter io.Writer, parseOptions SSRStreamOptions) {
	if parseOptions.Flush != nil {
		parseOptions.Flush()
		return
	}
	if parseFlusher, parseOk := parseWriter.(interface{ Flush() }); parseOk {
		parseFlusher.Flush()
	}
}

// safeRenderSSRStreamBoundaryChunk wraps renderSSRStreamBoundaryChunk so that a
// panic escaping a boundary's deferred resolution can NEVER crash the server
// process. Each pending boundary resolves on its own goroutine, and a panic on
// a goroutine is not recoverable by its parent — so without this guard a single
// panicking suspended component (or the deliberate re-panic in
// renderSSRStreamBoundaryChunk's inner recover for non-suspension panics) takes
// down the whole process AND deadlocks the collector, which is waiting for a
// chunk the dead goroutine never sends. The recover degrades the panic to an
// error chunk, consistent with how re-suspension is already reported, so the
// stream aborts with a recoverable error instead of a crash.
func safeRenderSSRStreamBoundaryChunk(parseCtx context.Context, parseBoundary ssrStreamPendingBoundary, parseOptions SSRStreamOptions) (parseChunk SSRStreamChunk) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseChunk = SSRStreamChunk{
				Kind:       SSRStreamChunkBoundary,
				BoundaryID: parseBoundary.id,
				Err:        fmt.Errorf("ssr stream: boundary %s panicked during resolution: %v", parseBoundary.id, parseRecovered),
			}
		}
	}()
	return renderSSRStreamBoundaryChunk(parseCtx, parseBoundary, parseOptions)
}

func renderSSRStreamBoundaryChunk(parseCtx context.Context, parseBoundary ssrStreamPendingBoundary, parseOptions SSRStreamOptions) SSRStreamChunk {
	if parseBoundary.suspension == nil || parseBoundary.suspension.Done == nil {
		return SSRStreamChunk{
			Kind:       SSRStreamChunkBoundary,
			BoundaryID: parseBoundary.id,
			Err:        fmt.Errorf("ssr stream: boundary %s suspended without a completion channel", parseBoundary.id),
		}
	}

	select {
	case <-parseCtx.Done():
		return SSRStreamChunk{Kind: SSRStreamChunkBoundary, BoundaryID: parseBoundary.id, Err: parseCtx.Err()}
	case <-parseBoundary.suspension.Done:
	}

	var parseContent strings.Builder
	parseErr := func() (parseErr error) {
		defer func() {
			if parseRecovered := recover(); parseRecovered != nil {
				if _, parseOk := AsSuspension(parseRecovered); parseOk {
					parseErr = fmt.Errorf("ssr stream: boundary %s suspended again after completion", parseBoundary.id)
					return
				}
				panic(parseRecovered)
			}
		}()
		// Boundary chunks render on their own goroutine, so the shell's cached
		// hook-owner id must be replaced with this goroutine's before rendering.
		return renderElementToString(&parseContent, parseBoundary.content, ssrHookOwnerContext(parseBoundary.contextValues))
	}()
	if parseErr != nil {
		return SSRStreamChunk{Kind: SSRStreamChunkBoundary, BoundaryID: parseBoundary.id, Err: parseErr}
	}

	parseHTML := renderSSRStreamBoundaryPatch(parseBoundary.id, parseContent.String(), !parseOptions.DisableBoundaryScript, parseOptions.ScriptNonce)
	return SSRStreamChunk{Kind: SSRStreamChunkBoundary, BoundaryID: parseBoundary.id, HTML: parseHTML}
}

func renderElementToStreamShell(parseBuilder *strings.Builder, parseElement *Element, parseState *ssrStreamState) error {
	if parseElement == nil {
		return nil
	}

	if parseTyp, parseOk := parseElement.Type.(string); parseOk {
		switch parseTyp {
		case "TEXT_ELEMENT":
			parseBuilder.WriteString(html.EscapeString(parseElement.TextContent))
			return nil
		case "FRAGMENT":
			return renderChildrenToStreamShell(parseBuilder, parseElement.Children, parseState)
		default:
			return renderHostElementToStreamShell(parseBuilder, parseTyp, parseElement, parseState)
		}
	}

	if parseProvider, parseOk2 := parseElement.Type.(*ContextProviderType); parseOk2 {
		if parseProvider.Descriptor != nil {
			parsePrevCtx := parseState.contextValues
			parseState.contextValues = deriveContextValues(parsePrevCtx, parseProvider.Descriptor.ID, parseElement.Props["value"])
			parseErr := renderChildrenToStreamShell(parseBuilder, parseElement.Children, parseState)
			parseState.contextValues = parsePrevCtx
			return parseErr
		}
		return renderChildrenToStreamShell(parseBuilder, parseElement.Children, parseState)
	}
	if _, parseOk3 := parseElement.Type.(*PortalElementType); parseOk3 {
		return renderChildrenToStreamShell(parseBuilder, parseElement.Children, parseState)
	}
	if _, parseOk4 := parseElement.Type.(*ReactiveTextElementType); parseOk4 {
		parseGetter, _ := parseElement.Props[reactiveTextGetterProp].(func() string)
		if parseGetter == nil {
			parseBuilder.WriteString(html.EscapeString(parseElement.TextContent))
			return nil
		}
		parseBuilder.WriteString(html.EscapeString(parseGetter()))
		return nil
	}
	if _, parseOk5 := parseElement.Type.(*ReactiveRegionElementType); parseOk5 {
		render, _ := parseElement.Props[reactiveRegionRenderProp].(func() *Element)
		if render == nil {
			return nil
		}
		return renderElementToStreamShell(parseBuilder, render(), parseState)
	}
	if _, parseOk6 := parseElement.Type.(*ErrorBoundaryType); parseOk6 {
		return renderErrorBoundaryToStreamShell(parseBuilder, parseElement, parseState)
	}
	if _, parseOk7 := parseElement.Type.(*AsyncBoundaryElementType); parseOk7 {
		return renderAsyncBoundaryToStreamShell(parseBuilder, parseElement, parseState)
	}

	parseResolved, parseErr := resolveComponentElement(parseElement, parseState.contextValues)
	if parseErr != nil {
		return parseErr
	}
	if parseResolved == nil {
		return nil
	}
	return renderElementToStreamShell(parseBuilder, parseResolved, parseState)
}

func renderErrorBoundaryToStreamShell(parseBuilder *strings.Builder, parseElement *Element, parseState *ssrStreamState) (parseErr error) {
	if parseElement == nil {
		return nil
	}

	// Children render into a scratch builder (mirroring the buffered serializer):
	// a panic mid-subtree must not leave partial, unclosed markup ahead of the
	// fallback. Pending stream boundaries registered by the discarded children
	// are rolled back too — their markers died with the scratch output, so
	// streaming their patches later would be wasted work targeting nothing.
	var parseChildBuilder strings.Builder
	parsePendingLen := len(parseState.pending)
	parseNextBoundaryID := parseState.nextBoundaryID
	defer func() {
		parseRecovered := recover()
		if parseRecovered == nil {
			parseBuilder.WriteString(parseChildBuilder.String())
			return
		}
		parseState.pending = parseState.pending[:parsePendingLen]
		parseState.nextBoundaryID = parseNextBoundaryID

		parseBoundaryErr := normalizeBoundaryError(parseRecovered)
		if parseOnError, _ := parseElement.Props["onError"].(func(error)); parseOnError != nil {
			func() {
				defer func() { _ = recover() }()
				parseOnError(parseBoundaryErr)
			}()
		}

		if parseFallbackFn, _ := parseElement.Props["errorFallback"].(func(error, func()) *Element); parseFallbackFn != nil {
			parseFallback := parseFallbackFn(parseBoundaryErr, func() {})
			parseErr = renderElementToStreamShell(parseBuilder, parseFallback, parseState)
			return
		}
		if parseFallback2, _ := parseElement.Props["fallback"].(*Element); parseFallback2 != nil {
			parseErr = renderElementToStreamShell(parseBuilder, parseFallback2, parseState)
			return
		}
		parseErr = nil
	}()

	return renderChildrenToStreamShell(&parseChildBuilder, parseElement.Children, parseState)
}

func renderAsyncBoundaryToStreamShell(parseBuilder *strings.Builder, parseElement *Element, parseState *ssrStreamState) error {
	if parseElement == nil {
		return nil
	}
	if parseErr, _ := parseElement.Props["error"].(error); parseErr != nil {
		return renderAsyncBoundaryFallbackToStreamShell(parseBuilder, parseElement, parseState, parseErr, "")
	}
	if parsePending, _ := parseElement.Props["pending"].(bool); parsePending {
		return renderAsyncBoundaryFallbackToStreamShell(parseBuilder, parseElement, parseState, nil, "")
	}

	parseContent, _ := parseElement.Props["content"].(*Element)
	if parseContent == nil {
		return renderChildrenToStreamShell(parseBuilder, parseElement.Children, parseState)
	}

	var parseContentBuilder strings.Builder
	var parseSuspension *Suspension
	parsePendingLen := len(parseState.pending)
	parseNextBoundaryID := parseState.nextBoundaryID
	parseRollbackPending := func() {
		parseState.pending = parseState.pending[:parsePendingLen]
		parseState.nextBoundaryID = parseNextBoundaryID
	}
	parseErr := func() (parseErr error) {
		defer func() {
			if parseRecovered := recover(); parseRecovered != nil {
				if parseRecoveredSuspension, parseOk := AsSuspension(parseRecovered); parseOk {
					parseSuspension = parseRecoveredSuspension
					parseErr = nil
					return
				}
				parseRollbackPending()
				panic(parseRecovered)
			}
		}()
		return renderElementToStreamShell(&parseContentBuilder, parseContent, parseState)
	}()
	if parseErr != nil {
		parseRollbackPending()
		return parseErr
	}
	if parseSuspension == nil {
		parseBuilder.WriteString(parseContentBuilder.String())
		return nil
	}

	parseRollbackPending()
	parseBoundaryID := parseState.nextSSRStreamBoundaryID()
	parseState.pending = append(parseState.pending, ssrStreamPendingBoundary{
		id:            parseBoundaryID,
		content:       parseContent,
		suspension:    parseSuspension,
		contextValues: parseState.contextValues,
	})
	return renderAsyncBoundaryFallbackToStreamShell(parseBuilder, parseElement, parseState, nil, parseBoundaryID)
}

func renderAsyncBoundaryFallbackToStreamShell(parseBuilder *strings.Builder, parseElement *Element, parseState *ssrStreamState, parseErr error, parseBoundaryID string) error {
	if parseElement == nil || parseElement.Props == nil {
		return nil
	}
	if parseBoundaryID != "" {
		writeSSRStreamBoundaryStart(parseBuilder, parseBoundaryID)
	}
	if parseErr != nil {
		if parseFallbackFn, _ := parseElement.Props["errorFallback"].(func(error) *Element); parseFallbackFn != nil {
			if parseErr2 := renderElementToStreamShell(parseBuilder, parseFallbackFn(parseErr), parseState); parseErr2 != nil {
				return parseErr2
			}
			if parseBoundaryID != "" {
				writeSSRStreamBoundaryEnd(parseBuilder, parseBoundaryID)
			}
			return nil
		}
	}
	if parseFallback, _ := parseElement.Props["fallback"].(*Element); parseFallback != nil {
		if parseErr2 := renderElementToStreamShell(parseBuilder, parseFallback, parseState); parseErr2 != nil {
			return parseErr2
		}
	}
	if parseBoundaryID != "" {
		writeSSRStreamBoundaryEnd(parseBuilder, parseBoundaryID)
	}
	return nil
}

func renderHostElementToStreamShell(parseBuilder *strings.Builder, parseTag string, parseElement *Element, parseState *ssrStreamState) error {
	parseProps := normalizeFormValueProps(parseElement.Props)

	// Mirror the buffered serializer: a controlled <textarea value="x"> renders
	// its value as text content, not as a browser-ignored value attribute.
	if parseTextareaValue, parseIsTextarea := textareaControlledValue(parseTag, parseProps); parseIsTextarea {
		parseBuilder.WriteByte('<')
		parseBuilder.WriteString(parseTag)
		writeSSRProps(parseBuilder, parseProps, "value")
		parseBuilder.WriteByte('>')
		parseBuilder.WriteString(html.EscapeString(parseTextareaValue))
		parseBuilder.WriteString("</")
		parseBuilder.WriteString(parseTag)
		parseBuilder.WriteByte('>')
		return nil
	}

	// A controlled <select value="x"> marks the matching <option> selected and
	// drops the value attribute from the select (options/optgroups don't suspend,
	// so the buffered select-children renderer is reused here).
	if parseSelectValue, parseIsSelect := selectControlledValue(parseTag, parseProps); parseIsSelect {
		parseBuilder.WriteByte('<')
		parseBuilder.WriteString(parseTag)
		writeSSRProps(parseBuilder, parseProps, "value")
		parseBuilder.WriteByte('>')
		if parseErr := renderSelectChildrenToString(parseBuilder, getElementChildren(parseElement), parseSelectValue, parseState.contextValues); parseErr != nil {
			return parseErr
		}
		parseBuilder.WriteString("</")
		parseBuilder.WriteString(parseTag)
		parseBuilder.WriteByte('>')
		return nil
	}

	parseBuilder.WriteByte('<')
	parseBuilder.WriteString(parseTag)
	if parseElement.isCompactHostProps && parseProps == nil {
		writeSSRCompactAttrs(parseBuilder, parseElement.getHostAttrs)
	} else {
		writeSSRProps(parseBuilder, parseProps)
	}
	parseBuilder.WriteByte('>')

	if isVoidElement(parseTag) {
		return nil
	}

	if parseElement.hasDirectText {
		parseBuilder.WriteString(html.EscapeString(parseElement.TextContent))
	} else if parseErr := renderChildrenToStreamShell(parseBuilder, getElementChildren(parseElement), parseState); parseErr != nil {
		return parseErr
	}

	parseBuilder.WriteString("</")
	parseBuilder.WriteString(parseTag)
	parseBuilder.WriteByte('>')
	return nil
}

func renderChildrenToStreamShell(parseBuilder *strings.Builder, parseChildren []any, parseState *ssrStreamState) error {
	for _, parseChild := range parseChildren {
		switch parseValue := parseChild.(type) {
		case nil:
			continue
		case *Element:
			if parseErr := renderElementToStreamShell(parseBuilder, parseValue, parseState); parseErr != nil {
				return parseErr
			}
		case string:
			parseBuilder.WriteString(html.EscapeString(parseValue))
		default:
			parseBuilder.WriteString(html.EscapeString(fmt.Sprint(parseValue)))
		}
	}
	return nil
}

func (parseState *ssrStreamState) nextSSRStreamBoundaryID() string {
	parseState.nextBoundaryID++
	parsePrefix := strings.TrimSpace(parseState.options.BoundaryIDPrefix)
	if parsePrefix == "" {
		parsePrefix = "gwc-stream-"
	}
	return fmt.Sprintf("%s%d", parsePrefix, parseState.nextBoundaryID)
}

func writeSSRStreamBoundaryStart(parseBuilder *strings.Builder, parseBoundaryID string) {
	parseBuilder.WriteString("<!--gwc-stream-boundary:")
	parseBuilder.WriteString(parseBoundaryID)
	parseBuilder.WriteString(":start-->")
}

func writeSSRStreamBoundaryEnd(parseBuilder *strings.Builder, parseBoundaryID string) {
	parseBuilder.WriteString("<!--gwc-stream-boundary:")
	parseBuilder.WriteString(parseBoundaryID)
	parseBuilder.WriteString(":end-->")
}

func renderSSRStreamBoundaryPatch(parseBoundaryID string, parseHTML string, parseIncludeScript bool, parseScriptNonce string) string {
	var parseBuilder strings.Builder
	parseBuilder.WriteString(`<template data-gwc-stream-boundary="`)
	parseBuilder.WriteString(html.EscapeString(parseBoundaryID))
	parseBuilder.WriteString(`">`)
	parseBuilder.WriteString(parseHTML)
	parseBuilder.WriteString(`</template>`)
	if parseIncludeScript {
		writeSSRStreamBoundaryPatchScript(&parseBuilder, parseBoundaryID, parseScriptNonce)
	}
	return parseBuilder.String()
}

func writeSSRStreamBoundaryPatchScript(parseBuilder *strings.Builder, parseBoundaryID string, parseScriptNonce string) {
	parseEncodedID, _ := json.Marshal(parseBoundaryID)
	parseBuilder.WriteString(`<script data-gwc-stream-boundary="`)
	parseBuilder.WriteString(html.EscapeString(parseBoundaryID))
	writeSSRScriptNonceAttr(parseBuilder, parseScriptNonce)
	parseBuilder.WriteString(`>(function(){var id=`)
	parseBuilder.Write(parseEncodedID)
	parseBuilder.WriteString(`;var d=document;var s="gwc-stream-boundary:"+id+":start";var e="gwc-stream-boundary:"+id+":end";var t=d.currentScript&&d.currentScript.previousElementSibling;if(!t||t.tagName!=="TEMPLATE")return;var w=d.createTreeWalker(d,NodeFilter.SHOW_COMMENT);var a=null,b=null,n;while((n=w.nextNode())){if(n.nodeValue===s)a=n;if(a&&n.nodeValue===e){b=n;break}}if(!a||!b)return;var r=d.createRange();r.setStartAfter(a);r.setEndBefore(b);r.deleteContents();r.insertNode(t.content.cloneNode(true));})();</script>`)
}

func writeSSRScriptNonceAttr(parseBuilder *strings.Builder, parseScriptNonce string) {
	parseScriptNonce = strings.TrimSpace(parseScriptNonce)
	if parseScriptNonce == "" {
		return
	}
	parseBuilder.WriteString(` nonce="`)
	parseBuilder.WriteString(html.EscapeString(parseScriptNonce))
	parseBuilder.WriteByte('"')
}
