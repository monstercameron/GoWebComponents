//go:build js && wasm

package main

import (
	"fmt"
	"io"
	"net/url"
	"strings"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/agentbridge"
)

// enableDemoAgentBridge connects the demo app to the localhost agent hub when
// the page is opened with ?gwc-dev=agent. The hub endpoint and token are
// provided by `gwc dev` (it mounts the hub and injects window.__GWC_AGENT_BRIDGE).
func enableDemoAgentBridge() {
	parseDev, parseToken, parseAppID, parseBuildID := readDemoAgentParams()
	if parseDev != "agent" || parseToken == "" {
		return
	}
	parseHost := demoLocationHost()
	if parseHost == "" {
		return
	}
	parseWSURL := "ws://" + parseHost + "/gwc-agent?token=" + url.QueryEscape(parseToken)
	agentbridge.RegisterReadCommands()
	agentbridge.RegisterWriteCommands()
	agentbridge.RegisterControlCommands()
	agentbridge.SetAgentModeActive(true)
	go runDemoAgentBridge(agentbridge.NewBridgeClient(parseAppID, parseBuildID), parseWSURL)
}

// runDemoAgentBridge keeps one browser WebSocket connected to the hub,
// reconnecting after reloads or hub restarts.
func runDemoAgentBridge(parseClient *agentbridge.BridgeClient, parseWSURL string) {
	parseInitial := 250 * time.Millisecond
	parseBackoff := parseInitial
	parseCap := 5 * time.Second
	for {
		parseSock, parseErr := dialDemoAgentSocket(parseWSURL)
		if parseErr != nil {
			time.Sleep(parseBackoff)
			parseBackoff = demoMinDuration(parseBackoff*2, parseCap)
			continue
		}
		parseBackoff = parseInitial
		parseClient.RunLoop(parseSock)
		time.Sleep(parseBackoff)
		parseBackoff = demoMinDuration(parseBackoff*2, parseCap)
	}
}

// demoAgentSocket adapts a browser WebSocket to agentbridge.AgentSocket.
type demoAgentSocket struct {
	raw       js.Value
	incoming  chan string
	errc      chan error
	done      chan struct{}
	openReady chan error

	openFn    js.Func
	messageFn js.Func
	errorFn   js.Func
	closeFn   js.Func
}

// dialDemoAgentSocket opens a browser WebSocket and waits until it is open.
func dialDemoAgentSocket(parseURL string) (*demoAgentSocket, error) {
	parseCtor := js.Global().Get("WebSocket")
	if parseCtor.Type() != js.TypeFunction {
		return nil, fmt.Errorf("demo agent bridge: WebSocket is unavailable")
	}
	parseSock := &demoAgentSocket{
		incoming:  make(chan string, 64),
		errc:      make(chan error, 4),
		done:      make(chan struct{}),
		openReady: make(chan error, 1),
	}
	parseSock.raw = parseCtor.New(parseURL)
	parseSock.openFn = js.FuncOf(func(js.Value, []js.Value) any {
		select {
		case parseSock.openReady <- nil:
		default:
		}
		return nil
	})
	parseSock.messageFn = js.FuncOf(func(_ js.Value, parseArgs []js.Value) any {
		parseData := ""
		if len(parseArgs) > 0 {
			parseData = parseArgs[0].Get("data").String()
		}
		select {
		case parseSock.incoming <- parseData:
		default:
		}
		return nil
	})
	parseSock.errorFn = js.FuncOf(func(js.Value, []js.Value) any {
		parseErr := fmt.Errorf("demo agent bridge: WebSocket error")
		select {
		case parseSock.errc <- parseErr:
		default:
		}
		select {
		case parseSock.openReady <- parseErr:
		default:
		}
		return nil
	})
	parseSock.closeFn = js.FuncOf(func(js.Value, []js.Value) any {
		select {
		case <-parseSock.done:
		default:
			close(parseSock.done)
		}
		select {
		case parseSock.openReady <- io.EOF:
		default:
		}
		return nil
	})
	parseSock.raw.Set("onopen", parseSock.openFn)
	parseSock.raw.Set("onmessage", parseSock.messageFn)
	parseSock.raw.Set("onerror", parseSock.errorFn)
	parseSock.raw.Set("onclose", parseSock.closeFn)
	select {
	case parseErr := <-parseSock.openReady:
		if parseErr != nil {
			_ = parseSock.Close()
			return nil, parseErr
		}
		return parseSock, nil
	case <-time.After(5 * time.Second):
		_ = parseSock.Close()
		return nil, fmt.Errorf("demo agent bridge: WebSocket open timed out")
	}
}

// ReadFrame returns one inbound text frame.
func (parseSock *demoAgentSocket) ReadFrame() (string, error) {
	select {
	case parseFrame, parseOK := <-parseSock.incoming:
		if !parseOK {
			return "", io.EOF
		}
		return parseFrame, nil
	case parseErr := <-parseSock.errc:
		return "", parseErr
	case <-parseSock.done:
		return "", io.EOF
	}
}

// WriteFrame sends one outbound text frame.
func (parseSock *demoAgentSocket) WriteFrame(parseFrame string) (parseErr error) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseErr = fmt.Errorf("demo agent bridge: WebSocket send panic: %v", parseRecovered)
		}
	}()
	parseSend := parseSock.raw.Get("send")
	if parseSend.Type() != js.TypeFunction {
		return fmt.Errorf("demo agent bridge: WebSocket send unavailable")
	}
	parseSock.raw.Call("send", parseFrame)
	return nil
}

// Close closes the WebSocket and releases its callbacks.
func (parseSock *demoAgentSocket) Close() error {
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

// readDemoAgentParams reads gwc-dev / token / app / build from the query string
// or the window.__GWC_AGENT_BRIDGE bootstrap global injected by gwc dev.
func readDemoAgentParams() (parseDev string, parseToken string, parseAppID string, parseBuildID string) {
	defer func() { recover() }()
	parseSearch := strings.TrimPrefix(js.Global().Get("location").Get("search").String(), "?")
	parseQuery, parseErr := url.ParseQuery(parseSearch)
	if parseErr != nil {
		return "", "", "", ""
	}
	parseToken = parseQuery.Get("gwc-agent-token")
	parseAppID = parseQuery.Get("gwc-agent-app")
	parseBuildID = parseQuery.Get("gwc-agent-build")
	parseBootstrap := js.Global().Get("__GWC_AGENT_BRIDGE")
	if parseBootstrap.Type() == js.TypeObject {
		if parseToken == "" {
			parseToken = demoBootstrapString(parseBootstrap, "token")
		}
		if parseAppID == "" {
			parseAppID = demoBootstrapString(parseBootstrap, "appId")
		}
		if parseBuildID == "" {
			parseBuildID = demoBootstrapString(parseBootstrap, "buildId")
		}
	}
	return parseQuery.Get("gwc-dev"), parseToken, parseAppID, parseBuildID
}

// demoBootstrapString reads one string field from the bootstrap global.
func demoBootstrapString(parseValue js.Value, parseName string) string {
	parseField := parseValue.Get(parseName)
	if parseField.Type() != js.TypeString {
		return ""
	}
	return parseField.String()
}

// demoLocationHost returns location.host from the browser.
func demoLocationHost() string {
	defer func() { recover() }()
	return js.Global().Get("location").Get("host").String()
}

// demoMinDuration returns the smaller duration.
func demoMinDuration(parseLeft time.Duration, parseRight time.Duration) time.Duration {
	if parseLeft < parseRight {
		return parseLeft
	}
	return parseRight
}
