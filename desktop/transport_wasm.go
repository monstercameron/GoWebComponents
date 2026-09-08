//go:build js && wasm

package desktop

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v5/interop"
)

type getJSTransport struct{ parseValue js.Value }

// Connect discovers the explicitly installed external bootstrap transport.
func Connect() (Client, error) {
	parseValue := js.Global().Get("__gwcDesktop")
	if parseValue.Type() != js.TypeObject || parseValue.IsNull() {
		return Client{}, getError("connect", interop.CodeUnavailable, fmt.Errorf("desktop bootstrap unavailable"))
	}
	parseClient := NewClient(getJSTransport{parseValue: parseValue})
	_, parseErr := parseClient.GetCapabilities()
	return parseClient, parseErr
}

// invoke decodes a synchronous envelope: no Go callbacks are retained by promises.
func (parseTransport getJSTransport) invoke(parseMethod string, parseTarget any, parseArgs ...any) (parseErr error) {
	defer func() {
		if parsePanic := recover(); parsePanic != nil {
			parseErr = getError(parseMethod, interop.CodeRemote, fmt.Errorf("transport exception: %v", parsePanic))
		}
	}()
	parseValue := parseTransport.parseValue.Call(parseMethod, parseArgs...)
	var parseEnvelope struct {
		Data    json.RawMessage   `json:"data"`
		Code    interop.ErrorCode `json:"code"`
		Message string            `json:"message"`
	}
	if parseErr = json.Unmarshal([]byte(parseValue.String()), &parseEnvelope); parseErr != nil {
		return getError(parseMethod, interop.CodeDecode, parseErr)
	}
	if parseEnvelope.Code != "" {
		return getError(parseMethod, parseEnvelope.Code, fmt.Errorf("%s", parseEnvelope.Message))
	}
	if parseTarget == nil {
		return nil
	}
	if parseErr = json.Unmarshal(parseEnvelope.Data, parseTarget); parseErr != nil {
		return getError(parseMethod, interop.CodeDecode, parseErr)
	}
	return nil
}

// Capabilities reads the installed protocol and method allowlist.
func (parseTransport getJSTransport) Capabilities() (Capabilities, error) {
	var parseValue Capabilities
	parseErr := parseTransport.invoke("capabilities", &parseValue)
	return parseValue, parseErr
}

// Start retains a cancellable native request in the bounded JS registry.
func (parseTransport getJSTransport) Start(parseMethod string, parseArgs json.RawMessage) (string, error) {
	var parseValue string
	parseErr := parseTransport.invoke("start", &parseValue, parseMethod, string(parseArgs))
	return parseValue, parseErr
}

// Poll returns a request result without installing promise callbacks in Go.
func (parseTransport getJSTransport) Poll(parseID string) (Reply, error) {
	var parseValue Reply
	parseErr := parseTransport.invoke("poll", &parseValue, parseID)
	return parseValue, parseErr
}

// Cancel forwards cancellation and forgets the request.
func (parseTransport getJSTransport) Cancel(parseID string) error {
	return parseTransport.invoke("cancel", nil, parseID)
}

// Listen installs a latest-value native event listener.
func (parseTransport getJSTransport) Listen(parseTopic string) (string, error) {
	var parseValue string
	parseErr := parseTransport.invoke("listen", &parseValue, parseTopic)
	return parseValue, parseErr
}

// Next consumes the latest event envelope.
func (parseTransport getJSTransport) Next(parseID string) (Reply, error) {
	var parseValue Reply
	parseErr := parseTransport.invoke("next", &parseValue, parseID)
	return parseValue, parseErr
}

// Unlisten releases the native runtime listener.
func (parseTransport getJSTransport) Unlisten(parseID string) error {
	return parseTransport.invoke("unlisten", nil, parseID)
}
