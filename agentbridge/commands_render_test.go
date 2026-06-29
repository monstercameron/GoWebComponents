package agentbridge

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

func TestRenderBuildElementAllowsSafeTreeAndSanitizesAttrs(parseT *testing.T) {
	parseElement, parseErr := renderBuildElement(renderTreeNode{
		Tag: " SECTION ",
		Attrs: map[string]string{
			"id":      "hero",
			"onclick": "alert(1)",
			"href":    " javascript:alert(1) ",
			"data-ok": "yes",
		},
		Text: "Hello",
		Children: []renderTreeNode{{
			Tag:   "a",
			Attrs: map[string]string{"href": "/docs"},
			Text:  "Docs",
		}},
	})
	if parseErr != nil {
		parseT.Fatalf("renderBuildElement returned error: %v", parseErr)
	}
	if parseElement.Type != "section" {
		parseT.Fatalf("tag was not normalized: %#v", parseElement.Type)
	}
	if parseElement.Props["id"] != "hero" || parseElement.Props["data-ok"] != "yes" {
		parseT.Fatalf("safe attributes missing: %#v", parseElement.Props)
	}
	if _, parseHasOnclick := parseElement.Props["onclick"]; parseHasOnclick {
		parseT.Fatalf("event attribute was not stripped: %#v", parseElement.Props)
	}
	if _, parseHasHref := parseElement.Props["href"]; parseHasHref {
		parseT.Fatalf("unsafe href was not stripped: %#v", parseElement.Props)
	}
	if len(parseElement.Children) != 2 {
		parseT.Fatalf("expected text plus child element, got %#v", parseElement.Children)
	}
	parseText, parseOk := parseElement.Children[0].(*runtime.Element)
	if !parseOk || parseText.Type != "TEXT_ELEMENT" || parseText.Props["nodeValue"] != "Hello" {
		parseT.Fatalf("text child was not built correctly: %#v", parseElement.Children[0])
	}
}

func TestRenderBuildElementRejectsDisallowedTags(parseT *testing.T) {
	_, parseErr := renderBuildElement(renderTreeNode{Tag: "script", Text: "alert(1)"})
	if parseErr == nil {
		parseT.Fatal("expected disallowed script tag to fail")
	}
	if parseErr.Code != ErrorCodeBadPayload || !strings.Contains(parseErr.Message, "not allowed") {
		parseT.Fatalf("unexpected disallowed-tag error: %#v", parseErr)
	}
}

func TestRenderUnsafeURLRecognizesScriptBearingSchemes(parseT *testing.T) {
	parseUnsafe := []string{"javascript:alert(1)", " DATA:text/html,x ", "vbscript:msgbox(1)"}
	for _, parseURL := range parseUnsafe {
		if !renderUnsafeURL(parseURL) {
			parseT.Fatalf("expected unsafe URL %q", parseURL)
		}
	}
	parseSafe := []string{"/relative", "https://example.test", "mailto:team@example.test"}
	for _, parseURL := range parseSafe {
		if renderUnsafeURL(parseURL) {
			parseT.Fatalf("expected safe URL %q", parseURL)
		}
	}
}

func TestRenderHandleTreeValidationContracts(parseT *testing.T) {
	SetAgentModeActive(false)
	if _, parseErr := renderHandleTree(json.RawMessage(`{}`)); parseErr == nil || parseErr.Code != ErrorCodeForbidden {
		parseT.Fatalf("inactive render-tree error = %#v, want forbidden", parseErr)
	}

	writeActivateAgentMode(parseT)
	parseAdapter := mockdom.NewMockDOMAdapter()
	parseContainer := parseAdapter.CreateElement("div")
	parseAdapter.SetAttribute(parseContainer, "id", "render-validation-root")
	runtime.InitGlobalRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})

	parseCases := []struct {
		name       string
		payload    string
		messageSub string
	}{
		{"malformed", `not json`, "malformed payload"},
		{"missing selector", `{"tree":{"tag":"div"}}`, "missing required field \"selector\""},
		{"missing container", `{"selector":"#missing-render-target","tree":{"tag":"div"}}`, "did not resolve"},
		{"disallowed tag", `{"selector":"#render-validation-root","tree":{"tag":"script","text":"alert(1)"}}`, "not allowed"},
	}
	for _, parseCase := range parseCases {
		parseCase := parseCase
		parseT.Run(parseCase.name, func(parseT *testing.T) {
			_, parseErr := renderHandleTree(json.RawMessage(parseCase.payload))
			if parseErr == nil {
				parseT.Fatal("expected render-tree validation error")
			}
			if parseErr.Code != ErrorCodeBadPayload {
				parseT.Fatalf("error code = %q, want %q (%s)", parseErr.Code, ErrorCodeBadPayload, parseErr.Message)
			}
			if !strings.Contains(parseErr.Message, parseCase.messageSub) {
				parseT.Fatalf("error message = %q, want substring %q", parseErr.Message, parseCase.messageSub)
			}
		})
	}
}

func TestRenderHandleTreeRendersSanitizedDOMAndAdvancesVersion(parseT *testing.T) {
	writeActivateAgentMode(parseT)
	resetAgentAudit()

	parseAdapter := mockdom.NewMockDOMAdapter()
	parseContainer := parseAdapter.CreateElement("div")
	parseAdapter.SetAttribute(parseContainer, "id", "agent-render-success")
	runtime.InitGlobalRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})
	parseBeforeVersion := runtime.GetGlobalRuntime().AgentStateVersion()

	parseRaw, parseErr := renderHandleTree(json.RawMessage(`{
		"selector":"#agent-render-success",
		"tree":{
			"tag":"a",
			"attrs":{"id":"safe-link","href":" javascript:alert(1) ","onclick":"alert(1)","data-kind":"doc"},
			"text":"Open"
		}
	}`))
	if parseErr != nil {
		parseT.Fatalf("render-tree returned error: %v", parseErr)
	}
	if !strings.Contains(string(parseRaw), `"ok":true`) {
		parseT.Fatalf("render-tree result = %s", string(parseRaw))
	}
	if parseAfterVersion := runtime.GetGlobalRuntime().AgentStateVersion(); parseAfterVersion <= parseBeforeVersion {
		parseT.Fatalf("state version did not advance: before=%d after=%d", parseBeforeVersion, parseAfterVersion)
	}

	parseChildren := parseAdapter.GetChildren(parseContainer)
	if len(parseChildren) != 1 {
		parseT.Fatalf("container children = %d, want 1", len(parseChildren))
	}
	parseLink, parseOK := parseChildren[0].(*mockdom.MockDOMNode)
	if !parseOK || parseLink.Tag != "a" {
		parseT.Fatalf("rendered child = %#v, want anchor node", parseChildren[0])
	}
	if parseGot := parseAdapter.GetAttribute(parseLink, "id"); parseGot != "safe-link" {
		parseT.Fatalf("safe id attr = %q, want safe-link", parseGot)
	}
	if parseGot := parseAdapter.GetAttribute(parseLink, "data-kind"); parseGot != "doc" {
		parseT.Fatalf("safe data attr = %q, want doc", parseGot)
	}
	if parseGot := parseAdapter.GetAttribute(parseLink, "href"); parseGot != "" {
		parseT.Fatalf("unsafe href was rendered: %q", parseGot)
	}
	if parseGot := parseAdapter.GetAttribute(parseLink, "onclick"); parseGot != "" {
		parseT.Fatalf("event handler attr was rendered: %q", parseGot)
	}
	parseTextChildren := parseAdapter.GetChildren(parseLink)
	if len(parseTextChildren) != 1 {
		parseT.Fatalf("link text children = %d, want 1", len(parseTextChildren))
	}
	parseText, parseOK := parseTextChildren[0].(*mockdom.MockDOMNode)
	if !parseOK || parseText.TextContent != "Open" {
		parseT.Fatalf("text child = %#v, want Open", parseTextChildren[0])
	}
}
