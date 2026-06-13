//go:build playwrightgo

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// captureEnvelope runs fn (which writes one agentic envelope to stdout) and
// returns the parsed envelope.
func captureEnvelope(t *testing.T, parseFn func() error) map[string]any {
	t.Helper()
	parseOrig := os.Stdout
	parseR, parseW, parseErr := os.Pipe()
	if parseErr != nil {
		t.Fatalf("pipe: %v", parseErr)
	}
	os.Stdout = parseW
	parseRunErr := parseFn()
	_ = parseW.Close()
	os.Stdout = parseOrig
	parseBuf := make([]byte, 0, 4096)
	parseTmp := make([]byte, 4096)
	for {
		parseN, parseReadErr := parseR.Read(parseTmp)
		parseBuf = append(parseBuf, parseTmp[:parseN]...)
		if parseReadErr != nil {
			break
		}
	}
	if parseRunErr != nil {
		t.Fatalf("command returned error: %v (output: %s)", parseRunErr, string(parseBuf))
	}
	parseStr := string(parseBuf)
	parseIdx := strings.Index(parseStr, "{")
	if parseIdx < 0 {
		t.Fatalf("no JSON envelope in output: %s", parseStr)
	}
	var parseEnv map[string]any
	if parseErr := json.Unmarshal([]byte(parseStr[parseIdx:]), &parseEnv); parseErr != nil {
		t.Fatalf("parse envelope: %v (output: %s)", parseErr, parseStr)
	}
	return parseEnv
}

const proxyTestPage = `<!doctype html><html><head><meta charset="utf-8"><title>proxy-test</title></head>
<body>
<h1 id="headline" data-kind="title">Proxy Test</h1>
<button id="btn" onclick="document.getElementById('out').textContent='clicked'">Go</button>
<div id="out">idle</div>
<input id="field" />
<script>
  window.__answer = 42;
  console.log("boot-log-marker");
  console.error("boot-error-marker");
  fetch("/data-ok").catch(function(){});
  fetch("/data-missing").catch(function(){});
</script>
</body></html>`

func newProxyServer() *httptest.Server {
	parseMux := http.NewServeMux()
	parseMux.HandleFunc("/", func(parseW http.ResponseWriter, parseR *http.Request) {
		if parseR.URL.Path == "/data-ok" {
			parseW.WriteHeader(200)
			_, _ = parseW.Write([]byte("ok"))
			return
		}
		if parseR.URL.Path == "/data-missing" {
			parseW.WriteHeader(404)
			return
		}
		parseW.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = parseW.Write([]byte(proxyTestPage))
	})
	return httptest.NewServer(parseMux)
}

// TestInputMechanicsMutateDOM pins that real input (click/fill/press) drives the
// live DOM. It drives one launch-mode page directly through the shared helper
// (the command wrappers are thin flag-parsers over these exact calls), so the
// mutation is observable via Evaluate on the same page.
func TestInputMechanicsMutateDOM(t *testing.T) {
	parseSrv := newProxyServer()
	defer parseSrv.Close()

	parsePage, parseErr := openProxyPage("", parseSrv.URL, 1024, 768)
	if parseErr != nil {
		t.Fatalf("openProxyPage: %v", parseErr)
	}
	defer parsePage.close()

	if parseErr := parsePage.page.Locator("#btn").Click(); parseErr != nil {
		t.Fatalf("click: %v", parseErr)
	}
	parseOut, parseErr := parsePage.page.Evaluate("() => document.getElementById('out').textContent")
	if parseErr != nil {
		t.Fatalf("evaluate out: %v", parseErr)
	}
	if parseOut != "clicked" {
		t.Fatalf("after click, #out = %v, want 'clicked'", parseOut)
	}

	if parseErr := parsePage.page.Locator("#field").Fill("hello world"); parseErr != nil {
		t.Fatalf("fill: %v", parseErr)
	}
	parseVal, _ := parsePage.page.Evaluate("() => document.getElementById('field').value")
	if parseVal != "hello world" {
		t.Fatalf("after fill, #field value = %v, want 'hello world'", parseVal)
	}

	// Press updates the value via keyboard (append a char).
	if parseErr := parsePage.page.Locator("#field").Press("!"); parseErr != nil {
		t.Fatalf("press: %v", parseErr)
	}
	parseVal2, _ := parsePage.page.Evaluate("() => document.getElementById('field').value")
	if parseVal2 != "hello world!" {
		t.Fatalf("after press '!', #field value = %v, want 'hello world!'", parseVal2)
	}
}

// TestConsoleCaptureLaunch pins that console messages + uncaught errors are
// captured from a launched page (with -reload to catch boot output).
func TestConsoleCaptureLaunch(t *testing.T) {
	parseSrv := newProxyServer()
	defer parseSrv.Close()

	parseEnv := captureEnvelope(t, func() error {
		return runConsoleCommand(launcher{}, []string{"-url", parseSrv.URL, "-reload", "-duration", "1500ms"})
	})
	if parseEnv["ok"] != true {
		t.Fatalf("console ok != true: %v", parseEnv)
	}
	parseData, _ := parseEnv["data"].(map[string]any)
	parseEntries, _ := parseData["entries"].([]any)
	parseJoined := ""
	for _, parseE := range parseEntries {
		if parseM, parseOk := parseE.(map[string]any); parseOk {
			parseJoined += parseM["type"].(string) + ":" + parseM["text"].(string) + "\n"
		}
	}
	if !strings.Contains(parseJoined, "boot-log-marker") {
		t.Fatalf("console did not capture log marker; got:\n%s", parseJoined)
	}
	if !strings.Contains(parseJoined, "boot-error-marker") {
		t.Fatalf("console did not capture error marker; got:\n%s", parseJoined)
	}
}

// TestNetworkCaptureLaunch pins that requests are captured and the 404 is
// flagged as a failure.
func TestNetworkCaptureLaunch(t *testing.T) {
	parseSrv := newProxyServer()
	defer parseSrv.Close()

	parseEnv := captureEnvelope(t, func() error {
		return runNetworkCommand(launcher{}, []string{"-url", parseSrv.URL, "-reload", "-duration", "1500ms"})
	})
	if parseEnv["ok"] != true {
		t.Fatalf("network ok != true: %v", parseEnv)
	}
	parseData, _ := parseEnv["data"].(map[string]any)
	parseEntries, _ := parseData["entries"].([]any)
	parseSawOK, parseSaw404 := false, false
	for _, parseE := range parseEntries {
		parseM, _ := parseE.(map[string]any)
		parseURL, _ := parseM["url"].(string)
		if strings.HasSuffix(parseURL, "/data-ok") {
			parseSawOK = true
		}
		if strings.HasSuffix(parseURL, "/data-missing") {
			parseSaw404 = true
			if parseM["failed"] != true {
				t.Fatalf("/data-missing not flagged failed: %v", parseM)
			}
		}
	}
	if !parseSawOK || !parseSaw404 {
		t.Fatalf("network missed requests: sawOK=%v saw404=%v", parseSawOK, parseSaw404)
	}
}

// TestDomReadLaunch pins that dom returns the real element's text + attributes.
func TestDomReadLaunch(t *testing.T) {
	parseSrv := newProxyServer()
	defer parseSrv.Close()

	parseEnv := captureEnvelope(t, func() error {
		return runDomCommand(launcher{}, []string{"-url", parseSrv.URL, "-selector", "#headline"})
	})
	if parseEnv["ok"] != true {
		t.Fatalf("dom ok != true: %v", parseEnv)
	}
	parseData, _ := parseEnv["data"].(map[string]any)
	if parseData["text"] != "Proxy Test" {
		t.Fatalf("dom text = %v, want 'Proxy Test'", parseData["text"])
	}
	parseAttrs, _ := parseData["attrs"].(map[string]any)
	if parseAttrs["id"] != "headline" {
		t.Fatalf("dom attrs.id = %v, want 'headline'", parseAttrs["id"])
	}
}

// TestEvalLaunch pins that eval returns a JS value from the live page.
func TestEvalLaunch(t *testing.T) {
	parseSrv := newProxyServer()
	defer parseSrv.Close()

	parseEnv := captureEnvelope(t, func() error {
		return runEvalCommand(launcher{}, []string{"-url", parseSrv.URL, "-expr", "window.__answer"})
	})
	if parseEnv["ok"] != true {
		t.Fatalf("eval ok != true: %v", parseEnv)
	}
	parseData, _ := parseEnv["data"].(map[string]any)
	// JSON numbers decode as float64.
	if parseVal, _ := parseData["value"].(float64); parseVal != 42 {
		t.Fatalf("eval value = %v, want 42", parseData["value"])
	}
}

// TestProxyVerbsRequireTarget pins that the verbs error without -url or -cdp.
func TestProxyVerbsRequireTarget(t *testing.T) {
	for _, parseTC := range []struct {
		name string
		fn   func() error
	}{
		{"click", func() error { return runClickCommand(launcher{}, []string{"-selector", "#x"}) }},
		{"dom", func() error { return runDomCommand(launcher{}, []string{"-selector", "#x"}) }},
		{"eval", func() error { return runEvalCommand(launcher{}, []string{"-expr", "1"}) }},
		{"console", func() error { return runConsoleCommand(launcher{}, []string{}) }},
	} {
		parseEnv := captureEnvelope(t, parseTC.fn)
		if parseEnv["ok"] != false {
			t.Fatalf("%s without target: ok != false: %v", parseTC.name, parseEnv)
		}
	}
}
