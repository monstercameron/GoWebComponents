//go:build js && wasm && gwcagent

package app

import (
	"fmt"
	"io"
	"net/url"
	"strings"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/agentbridge"
)

type parseDogfoodAgentSocket struct {
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

// parseEnableDogfoodAgentBridge enables the ai-chat-wizard dogfood bridge for
// wasm builds launched with gwc-dev=agent.
func parseEnableDogfoodAgentBridge() {
	parseDev, parseToken, parseAppID, parseBuildID, parseErr := parseDogfoodAgentQueryParams()
	if parseErr != nil || parseDev != "agent" || parseToken == "" {
		return
	}
	parseHost := parseDogfoodLocationHost()
	if parseHost == "" {
		return
	}
	parseWSURL := "ws://" + parseHost + "/gwc-agent?token=" + url.QueryEscape(parseToken)
	agentbridge.RegisterReadCommands()
	agentbridge.RegisterWriteCommands()
	agentbridge.RegisterControlCommands()
	agentbridge.SetAgentModeActive(true)
	go parseRunDogfoodAgentBridge(agentbridge.NewBridgeClient(parseAppID, parseBuildID), parseWSURL)
}

// parseRunDogfoodAgentBridge keeps one browser WebSocket connected to the
// localhost hub, reconnecting after reloads or hub restarts.
func parseRunDogfoodAgentBridge(parseClient *agentbridge.BridgeClient, parseWSURL string) {
	parseInitialBackoff := 250 * time.Millisecond
	parseBackoff := parseInitialBackoff
	parseCapBackoff := 5 * time.Second
	for {
		parseSock, parseErr := parseDialDogfoodAgentSocket(parseWSURL)
		if parseErr != nil {
			time.Sleep(parseBackoff)
			parseBackoff = parseDogfoodMinDuration(parseBackoff*2, parseCapBackoff)
			continue
		}
		parseBackoff = parseInitialBackoff
		parseClient.RunLoop(parseSock)
		time.Sleep(parseBackoff)
		parseBackoff = parseDogfoodMinDuration(parseBackoff*2, parseCapBackoff)
	}
}

// parseDialDogfoodAgentSocket constructs a browser WebSocket and waits until
// it is open before returning it to the bridge client.
func parseDialDogfoodAgentSocket(parseURL string) (*parseDogfoodAgentSocket, error) {
	parseCtor := js.Global().Get("WebSocket")
	if parseCtor.Type() != js.TypeFunction {
		return nil, fmt.Errorf("dogfood agent bridge: WebSocket is unavailable")
	}
	parseSock := &parseDogfoodAgentSocket{
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
		parseErr := fmt.Errorf("dogfood agent bridge: WebSocket error")
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
		return nil, fmt.Errorf("dogfood agent bridge: WebSocket open timed out")
	}
}

// ReadFrame returns one inbound text frame.
func (parseSock *parseDogfoodAgentSocket) ReadFrame() (string, error) {
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
func (parseSock *parseDogfoodAgentSocket) WriteFrame(parseFrame string) (parseErr error) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseErr = fmt.Errorf("dogfood agent bridge: WebSocket send panic: %v", parseRecovered)
		}
	}()
	parseSend := parseSock.raw.Get("send")
	if parseSend.Type() != js.TypeFunction {
		return fmt.Errorf("dogfood agent bridge: WebSocket send unavailable")
	}
	parseSock.raw.Call("send", parseFrame)
	return nil
}

// Close closes the browser WebSocket and releases callback functions.
func (parseSock *parseDogfoodAgentSocket) Close() error {
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

// parseDogfoodAgentQueryParams reads the bridge activation query and bootstrap
// globals from the browser.
func parseDogfoodAgentQueryParams() (parseDev string, parseToken string, parseAppID string, parseBuildID string, parseErr error) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseErr = fmt.Errorf("dogfood agent bridge: read query params: %v", parseRecovered)
		}
	}()
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
		parseToken = parseDogfoodBootstrapString(parseBootstrap, "token")
	}
	if parseAppID == "" && parseBootstrap.Type() == js.TypeObject {
		parseAppID = parseDogfoodBootstrapString(parseBootstrap, "appId")
	}
	if parseBuildID == "" && parseBootstrap.Type() == js.TypeObject {
		parseBuildID = parseDogfoodBootstrapString(parseBootstrap, "buildId")
	}
	return parseQuery.Get("gwc-dev"), parseToken, parseAppID, parseBuildID, nil
}

// parseDogfoodBootstrapString reads one string field from the bootstrap global.
func parseDogfoodBootstrapString(parseValue js.Value, parseName string) string {
	parseField := parseValue.Get(parseName)
	if parseField.Type() != js.TypeString {
		return ""
	}
	return parseField.String()
}

// parseDogfoodLocationHost returns location.host from the browser.
func parseDogfoodLocationHost() string {
	defer func() { recover() }()
	return js.Global().Get("location").Get("host").String()
}

// parseDogfoodMinDuration returns the smaller duration.
func parseDogfoodMinDuration(parseLeft time.Duration, parseRight time.Duration) time.Duration {
	if parseLeft < parseRight {
		return parseLeft
	}
	return parseRight
}
