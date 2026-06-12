//go:build js && wasm && gwcagent

package agentbridge

import (
	"fmt"
	"io"
	"net/url"
	"strings"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

var activeConsoleCaptureRelease consoleCaptureRelease

// wasmSocket is an AgentSocket backed by a browser WebSocket via syscall/js.
// All registered js.Func values are released when the socket closes to prevent
// memory leaks.
type wasmSocket struct {
	raw      js.Value
	incoming chan string
	errc     chan error
	done     chan struct{}

	openFn    js.Func
	messageFn js.Func
	errorFn   js.Func
	closeFn   js.Func
}

// dialWASMSocket constructs a wasmSocket that connects to the given ws:// URL
// and returns it. The socket is ready when the first message or error is
// received; the open event unblocks ReadFrame only after it fires.
func dialWASMSocket(parseURL string) (*wasmSocket, error) {
	parseCtor := js.Global().Get("WebSocket")
	if parseCtor.Type() != js.TypeFunction {
		return nil, fmt.Errorf("agentbridge: WebSocket is not available")
	}

	parseSock := &wasmSocket{
		incoming: make(chan string, 64),
		errc:     make(chan error, 4),
		done:     make(chan struct{}),
	}

	parseRaw := parseCtor.New(parseURL)
	parseSock.raw = parseRaw

	parseSock.openFn = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		return nil
	})
	parseSock.messageFn = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseData := ""
		if len(parseArgs) > 0 {
			parseData = parseArgs[0].Get("data").String()
		}
		select {
		case parseSock.incoming <- parseData:
		default:
			// Inbound buffer full: report the drop rather than losing the
			// frame silently. A dropped command would otherwise strand the
			// hub waiting until its context times out with no signal.
			runtime.ReportDiagnostic("agentbridge", runtime.DiagnosticWarning,
				"bridge client: inbound frame dropped (buffer full)")
		}
		return nil
	})
	parseSock.errorFn = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		select {
		case parseSock.errc <- fmt.Errorf("agentbridge: WebSocket error"):
		default:
		}
		return nil
	})
	parseSock.closeFn = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		select {
		case <-parseSock.done:
		default:
			close(parseSock.done)
		}
		return nil
	})

	parseRaw.Set("onopen", parseSock.openFn)
	parseRaw.Set("onmessage", parseSock.messageFn)
	parseRaw.Set("onerror", parseSock.errorFn)
	parseRaw.Set("onclose", parseSock.closeFn)

	return parseSock, nil
}

// ReadFrame blocks until a text frame arrives, the socket closes, or an error
// occurs.
func (parseSock *wasmSocket) ReadFrame() (string, error) {
	select {
	case parseFrame, parseOk := <-parseSock.incoming:
		if !parseOk {
			return "", io.EOF
		}
		return parseFrame, nil
	case parseErr := <-parseSock.errc:
		return "", parseErr
	case <-parseSock.done:
		return "", io.EOF
	}
}

// WriteFrame sends one text frame over the browser WebSocket.
func (parseSock *wasmSocket) WriteFrame(parseFrame string) (parseErr error) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseErr = fmt.Errorf("agentbridge: WebSocket send panic: %v", parseRecovered)
		}
	}()
	parseSend := parseSock.raw.Get("send")
	if parseSend.Type() != js.TypeFunction {
		return fmt.Errorf("agentbridge: WebSocket send unavailable")
	}
	parseSock.raw.Call("send", parseFrame)
	return nil
}

// Close terminates the browser WebSocket and releases all js.Func handles.
func (parseSock *wasmSocket) Close() error {
	select {
	case <-parseSock.done:
	default:
		close(parseSock.done)
	}
	defer func() { recover() }()
	parseSock.raw.Set("onopen", js.Null())
	parseSock.raw.Set("onmessage", js.Null())
	parseSock.raw.Set("onerror", js.Null())
	parseSock.raw.Set("onclose", js.Null())
	parseSock.openFn.Release()
	parseSock.messageFn.Release()
	parseSock.errorFn.Release()
	parseSock.closeFn.Release()
	parseClose := parseSock.raw.Get("close")
	if parseClose.Type() == js.TypeFunction {
		parseSock.raw.Call("close")
	}
	return nil
}

// parseAgentQueryParams reads location.search from the browser and returns the
// gwc-dev and gwc-agent-token values.
func parseAgentQueryParams() (parseDev string, parseToken string, parseAppID string, parseBuildID string, parseErr error) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseErr = fmt.Errorf("agentbridge: read location.search: %v", parseRecovered)
		}
	}()
	// location.search includes the leading "?"; url.ParseQuery does not strip
	// it, so without TrimPrefix the first key parses as "?gwc-dev" and the
	// bridge never activates from a normal browser URL.
	parseSearch := strings.TrimPrefix(js.Global().Get("location").Get("search").String(), "?")
	parseQuery, parseParseErr := url.ParseQuery(parseSearch)
	if parseParseErr != nil {
		return "", "", "", "", parseParseErr
	}
	parseToken = parseQuery.Get("gwc-agent-token")
	parseAppID = parseQuery.Get("gwc-agent-app")
	parseBuildID = parseQuery.Get("gwc-agent-build")
	parseBootstrap := js.Global().Get("__GWC_AGENT_BRIDGE")
	if parseToken == "" && parseBootstrap.Type() == js.TypeObject {
		parseToken = parseBootstrapString(parseBootstrap, "token")
	}
	if parseAppID == "" && parseBootstrap.Type() == js.TypeObject {
		parseAppID = parseBootstrapString(parseBootstrap, "appId")
	}
	if parseBuildID == "" && parseBootstrap.Type() == js.TypeObject {
		parseBuildID = parseBootstrapString(parseBootstrap, "buildId")
	}
	return parseQuery.Get("gwc-dev"), parseToken, parseAppID, parseBuildID, nil
}

func parseBootstrapString(parseValue js.Value, parseName string) string {
	parseField := parseValue.Get(parseName)
	if parseField.Type() != js.TypeString {
		return ""
	}
	return parseField.String()
}

// parseLocationHost returns location.host from the browser.
func parseLocationHost() string {
	defer func() { recover() }()
	return js.Global().Get("location").Get("host").String()
}

// enableAgentBridgeLoop runs the reconnect loop forever; called from a
// goroutine. A dev bridge stays in agent mode for the life of the page and
// reconnects across hub restarts and livereload rebuilds, so the loop never
// returns (and agent mode is never deactivated while the page lives).
func enableAgentBridgeLoop(parseClient *BridgeClient, parseWSURL string) {
	parseInitialBackoff := 250 * time.Millisecond
	parseCapBackoff := 5 * time.Second
	parseBackoff := parseInitialBackoff

	for {
		parseSock, parseDialErr := dialWASMSocket(parseWSURL)
		if parseDialErr != nil {
			time.Sleep(parseBackoff)
			parseBackoff = min(parseBackoff*2, parseCapBackoff)
			continue
		}
		parseBackoff = parseInitialBackoff
		parseClient.RunLoop(parseSock)

		// Socket dropped: back off, then reconnect.
		time.Sleep(parseBackoff)
		parseBackoff = min(parseBackoff*2, parseCapBackoff)
	}
}

// EnableAgentBridge checks the page URL for gwc-dev=agent and gwc-agent-token.
// If both are present it dials the agent hub WebSocket, marks agent mode active,
// and runs a reconnecting client loop on a background goroutine. If the URL
// parameters are absent the function is a no-op so release builds compile it
// out entirely.
func EnableAgentBridge() {
	parseDev, parseToken, parseAppID, parseBuildID, parseQueryErr := parseAgentQueryParams()
	if parseQueryErr != nil || parseDev != "agent" || parseToken == "" {
		return
	}

	parseHost := parseLocationHost()
	if parseHost == "" {
		return
	}

	parseWSURL := "ws://" + parseHost + "/gwc-agent?token=" + url.QueryEscape(parseToken)
	RegisterReadCommands()
	RegisterWriteCommands()
	RegisterControlCommands()
	parseClient := NewBridgeClient(parseAppID, parseBuildID)
	SetAgentModeActive(true)
	activeConsoleCaptureRelease = installConsoleCapture(parseClient)

	// The reconnect loop runs for the life of the page; agent mode stays
	// active throughout (see enableAgentBridgeLoop).
	go enableAgentBridgeLoop(parseClient, parseWSURL)
}
