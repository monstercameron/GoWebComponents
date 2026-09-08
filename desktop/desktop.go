// Package desktop provides an optional, host-independent desktop frontend bridge.
// Native Wails packages belong in the application's host module, never here.
package desktop

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/interop"
)

// ProtocolVersion identifies the synchronous transport envelope contract.
const ProtocolVersion = 1

// RequestTimeout is the default call ceiling, including context.Background calls.
const RequestTimeout = 30 * time.Second

// MaximumRequestTimeout bounds explicitly extended interactive calls.
const MaximumRequestTimeout = 10 * time.Minute

// BootstrapSource is the external ES module to serve as desktop.js before Wasm.
//
//go:embed desktop.js
var BootstrapSource string

// Capabilities describes an installed bridge, not an authorization credential.
type Capabilities struct {
	Protocol    int      `json:"protocol"`
	Platform    string   `json:"platform"`
	HostVersion string   `json:"hostVersion"`
	Methods     []string `json:"methods"`
	Topics      []string `json:"topics"`
	// Features advertises non-RPC capabilities such as installed native menus.
	// It is feature discovery, never authorization; RPC features still require methods.
	Features []Feature `json:"features,omitempty"`
}

// Reply is a synchronously polled request/event envelope; Data is JSON, not JS handles.
type Reply struct {
	Done    bool              `json:"done"`
	Data    json.RawMessage   `json:"data"`
	Code    interop.ErrorCode `json:"code,omitempty"`
	Message string            `json:"message,omitempty"`
}

// Transport can be injected in native tests. Methods must return promptly and
// serialize access internally. Cancel must forget requests even if native work stalls.
type Transport interface {
	Capabilities() (Capabilities, error)
	Start(string, json.RawMessage) (string, error)
	Poll(string) (Reply, error)
	Cancel(string) error
	Listen(string) (string, error)
	Next(string) (Reply, error)
	Unlisten(string) error
}

// Client owns calls and subscriptions; its zero value is predictably unavailable.
type Client struct{ parseTransport Transport }

// NewClient constructs a client with an explicit transport for testing or embedding.
func NewClient(parseTransport Transport) Client { return Client{parseTransport: parseTransport} }

// GetCapabilities validates the installed protocol before callers enable controls.
func (parseClient Client) GetCapabilities() (Capabilities, error) {
	if parseClient.parseTransport == nil {
		return Capabilities{}, getError("capabilities", interop.CodeUnavailable, errors.New("desktop host unavailable"))
	}
	parseCaps, parseErr := parseClient.parseTransport.Capabilities()
	if parseErr == nil && parseCaps.Protocol != ProtocolVersion {
		parseErr = getError("capabilities", interop.CodeInvalid, errors.New("desktop protocol mismatch"))
	}
	return parseCaps, parseErr
}

// Call decodes one registered method's JSON response. Use strings for wide integer
// identifiers, base64 strings for bytes and RFC3339 strings for time values.
// Cancellation is cooperative and cannot undo completed native writes.
func Call[T any](parseContext context.Context, parseClient Client, parseMethod string, parseArgs ...any) (T, error) {
	return CallWithTimeout[T](parseContext, parseClient, RequestTimeout, parseMethod, parseArgs...)
}

// CallWithTimeout opts one call into a positive ceiling of at most ten minutes.
// Earlier context deadlines still win. Use longer waits only for interactive native
// operations; cancellation cannot dismiss every OS dialog or undo completed writes.
func CallWithTimeout[T any](parseContext context.Context, parseClient Client, parseTimeout time.Duration, parseMethod string, parseArgs ...any) (T, error) {
	var parseZero T
	if parseTimeout <= 0 || parseTimeout > MaximumRequestTimeout {
		return parseZero, getError(parseMethod, interop.CodeInvalid, errors.New("call timeout must be positive and at most ten minutes"))
	}
	parseContext, parseCancel := getCallContext(parseContext, parseTimeout)
	defer parseCancel()
	if parseErr := parseContext.Err(); parseErr != nil {
		return parseZero, getContextError(parseMethod, parseErr)
	}
	parseCaps, parseErr := parseClient.GetCapabilities()
	if parseErr != nil {
		return parseZero, parseErr
	}
	if !hasName(parseCaps.Methods, parseMethod) {
		return parseZero, getError(parseMethod, interop.CodeMissingExport, errors.New("method not registered"))
	}
	if parseArgs == nil {
		parseArgs = []any{}
	}
	parseData, parseErr := json.Marshal(parseArgs)
	if parseErr != nil {
		return parseZero, getError(parseMethod, interop.CodeEncode, parseErr)
	}
	if parseErr = validateWireNumbers(parseData); parseErr != nil {
		return parseZero, getError(parseMethod, interop.CodeEncode, parseErr)
	}
	if parseErr = parseContext.Err(); parseErr != nil {
		return parseZero, getContextError(parseMethod, parseErr)
	}
	parseID, parseErr := parseClient.parseTransport.Start(parseMethod, parseData)
	if parseErr != nil {
		return parseZero, parseErr
	}
	// Cleanup forgets the request even when forwarding native cancellation fails.
	defer func() { _ = parseClient.parseTransport.Cancel(parseID) }()
	parseTicker := time.NewTicker(5 * time.Millisecond)
	defer parseTicker.Stop()
	for {
		if parseErr = parseContext.Err(); parseErr != nil {
			return parseZero, getContextError(parseMethod, parseErr)
		}
		parseReply, parsePollErr := parseClient.parseTransport.Poll(parseID)
		// A transport can synchronously trigger cancellation while producing its final reply.
		if parseErr = parseContext.Err(); parseErr != nil {
			return parseZero, getContextError(parseMethod, parseErr)
		}
		if parsePollErr != nil {
			return parseZero, parsePollErr
		}
		if parseReply.Done {
			if parseReply.Code != "" {
				return parseZero, getError(parseMethod, parseReply.Code, errors.New(parseReply.Message))
			}
			if parseErr = json.Unmarshal(parseReply.Data, &parseZero); parseErr != nil {
				return parseZero, getError(parseMethod, interop.CodeDecode, parseErr)
			}
			return parseZero, nil
		}
		select {
		case <-parseContext.Done():
			return parseZero, getContextError(parseMethod, parseContext.Err())
		case <-parseTicker.C:
		}
	}
}

// getCallContext applies a call ceiling without extending an earlier caller deadline.
func getCallContext(parseContext context.Context, parseTimeout time.Duration) (context.Context, context.CancelFunc) {
	if parseContext == nil {
		parseContext = context.Background()
	}
	return context.WithTimeout(parseContext, parseTimeout)
}

// validateWireNumbers rejects numeric magnitudes that lose integer precision in JavaScript.
func validateWireNumbers(parseData []byte) error {
	parseDecoder := json.NewDecoder(bytes.NewReader(parseData))
	parseDecoder.UseNumber()
	for {
		parseToken, parseErr := parseDecoder.Token()
		if parseErr == io.EOF {
			return nil
		}
		if parseErr != nil {
			return parseErr
		}
		if parseNumber, isNumber := parseToken.(json.Number); isNumber {
			parseValue, parseErr := parseNumber.Float64()
			if parseErr != nil || math.Abs(parseValue) > 9007199254740991 {
				return errors.New("numeric payload exceeds JavaScript safe range; encode wide numbers as strings")
			}
		}
	}
}

// Subscribe delivers latest-value events at most once per polling tick (16 ms).
// Bursts coalesce per subscription, so this API is for state/progress, not audit logs.
// Cancel is idempotent; a handler already executing may finish after cancellation.
func Subscribe[T any](parseContext context.Context, parseClient Client, parseTopic string, parseHandler func(T, error)) (func(), error) {
	if parseContext == nil {
		parseContext = context.Background()
	}
	if parseErr := parseContext.Err(); parseErr != nil {
		return nil, getContextError(parseTopic, parseErr)
	}
	if parseHandler == nil || !strings.Contains(parseTopic, ".") {
		return nil, getError(parseTopic, interop.CodeInvalid, errors.New("namespaced topic and handler required"))
	}
	parseCaps, parseErr := parseClient.GetCapabilities()
	if parseErr != nil {
		return nil, parseErr
	}
	if !hasName(parseCaps.Topics, parseTopic) {
		return nil, getError(parseTopic, interop.CodeMissingExport, errors.New("topic not registered"))
	}
	parseID, parseErr := parseClient.parseTransport.Listen(parseTopic)
	if parseErr != nil {
		return nil, parseErr
	}
	parseContext, parseCancel := context.WithCancel(parseContext)
	var parseOnce sync.Once
	parseStop := func() { parseOnce.Do(func() { parseCancel(); _ = parseClient.parseTransport.Unlisten(parseID) }) }
	go func() {
		defer interop.RecoverContainedPanic("desktop subscription")
		defer parseStop()
		parseTicker := time.NewTicker(16 * time.Millisecond)
		defer parseTicker.Stop()
		for {
			select {
			case <-parseContext.Done():
				return
			case <-parseTicker.C:
			}
			parseReply, parsePollErr := parseClient.parseTransport.Next(parseID)
			if parseContext.Err() != nil {
				return
			}
			if parsePollErr != nil {
				var parseZero T
				parseHandler(parseZero, parsePollErr)
				return
			}
			if !parseReply.Done {
				continue
			}
			var parseValue T
			if parseReply.Code != "" {
				parsePollErr = getError(parseTopic, parseReply.Code, errors.New(parseReply.Message))
			} else {
				parsePollErr = json.Unmarshal(parseReply.Data, &parseValue)
				if parsePollErr != nil {
					parsePollErr = getError(parseTopic, interop.CodeDecode, parsePollErr)
				}
			}
			parseHandler(parseValue, parsePollErr)
		}
	}()
	return parseStop, nil
}

// hasName checks exact allowlisted names without wildcard expansion.
func hasName(parseNames []string, parseName string) bool {
	for _, parseItem := range parseNames {
		if parseItem == parseName {
			return true
		}
	}
	return false
}

// getError preserves the shared interop error classification.
func getError(parseTarget string, parseCode interop.ErrorCode, parseErr error) error {
	return &interop.Error{Op: "desktop", Target: parseTarget, Code: parseCode, Err: parseErr}
}

// getContextError distinguishes deadlines from explicit cancellation.
func getContextError(parseTarget string, parseErr error) error {
	parseCode := interop.CodeCancelled
	if errors.Is(parseErr, context.DeadlineExceeded) {
		parseCode = interop.CodeTimeout
	}
	return getError(parseTarget, parseCode, parseErr)
}
