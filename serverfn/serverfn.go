// Package serverfn is the runtime for GoWebComponents server functions — the //gwc:server
// keystone. A server function is an ordinary, type-safe Go function
//
//	func(context.Context, Req) (Resp, error)
//
// that runs only on the server. `gwc server gen` generates a server registration (which
// wires the function into an HTTP mux via Handle) and a matching client stub (which calls
// it via Call). The two sides share one Go signature, so a server function is invoked from
// the browser with full compile-time type safety and no hand-written fetch/JSON glue.
//
// Both Handle and Call use net/http + encoding/json with no build tags: on the server they
// use real sockets, and in the browser Go's net/http transport is backed by the Fetch API,
// so the exact same code path is exercised — and the whole package is unit-testable
// natively with httptest.
package serverfn

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"
	"sync"
)

// RoutePrefix is the URL path prefix every server function is served under.
const RoutePrefix = "/_gwc/fn/"

// Endpoint returns the URL path a server function with the given name is served at.
func Endpoint(parseName string) string {
	return RoutePrefix + parseName
}

var (
	configMu      sync.RWMutex
	clientBaseURL string
	httpClient    = http.DefaultClient
	// maxRequestBytes caps the request body Handle will read. Server functions
	// are the trust boundary — request bodies are attacker-controlled — so an
	// uncapped io.ReadAll is a trivial memory-exhaustion DoS. Configurable via
	// SetMaxRequestBytes; default 10 MiB.
	maxRequestBytes int64 = 10 << 20
	// csrfProtection, when true (the default), makes Handle require a JSON
	// Content-Type and reject browser-flagged cross-site requests. See
	// SetCSRFProtection for the rationale.
	csrfProtection = true
)

// errorLoggerMu guards errorLogger independently of configMu so logging a server
// error never contends with request-path config reads.
var (
	errorLoggerMu sync.RWMutex
	// errorLogger receives the REAL (un-sanitized) error behind every 5xx — the
	// error a server function returned or a recovered panic — so operators keep
	// full diagnostics server-side even though the client only ever sees a generic
	// "internal server error". Defaults to a stderr line; set to nil to silence,
	// or override to route into a structured logger. The name is the server
	// function's registered name.
	errorLogger = func(parseName string, parseErr error) {
		fmt.Fprintf(os.Stderr, "serverfn: %s failed: %v\n", parseName, parseErr)
	}
)

// SetErrorLogger overrides the sink for the real server-side error behind a 5xx.
// Passing nil disables server-side error logging (the client still only sees a
// generic message). Enterprise deployments should route this into their logger.
func SetErrorLogger(parseLogger func(parseName string, parseErr error)) {
	errorLoggerMu.Lock()
	defer errorLoggerMu.Unlock()
	errorLogger = parseLogger
}

// logServerError surfaces the real error to the configured sink (if any).
func logServerError(parseName string, parseErr error) {
	errorLoggerMu.RLock()
	parseLogger := errorLogger
	errorLoggerMu.RUnlock()
	if parseLogger != nil {
		parseLogger(parseName, parseErr)
	}
}

// SetCSRFProtection toggles Handle's cross-site-request defenses (default on).
//
// When enabled, Handle requires the request's Content-Type to be application/json
// and rejects any request a browser has flagged (via Sec-Fetch-Site) as
// cross-site. This is the package's CSRF defense:
//
//   - application/json is NOT a CORS-safelisted Content-Type, so a cross-origin
//     browser request carrying it forces a CORS preflight, which an attacker's
//     forged page cannot satisfy for a cookie-authenticated victim; and an HTML
//     <form> — which can only send the three safelisted content types — cannot
//     forge a call at all.
//   - Sec-Fetch-Site is a browser-set, JS-unspoofable request header; when it is
//     present and reports "cross-site" the request is rejected outright. It is
//     absent for non-browser clients, so this never rejects a legitimate
//     server-to-server or curl caller.
//
// The generated client always sends application/json, so this is transparent to
// generated code. Disable ONLY if you must accept raw non-JSON callers and have
// another CSRF defense in place.
func SetCSRFProtection(parseEnabled bool) {
	configMu.Lock()
	defer configMu.Unlock()
	csrfProtection = parseEnabled
}

func currentCSRFProtection() bool {
	configMu.RLock()
	defer configMu.RUnlock()
	return csrfProtection
}

// SetMaxRequestBytes sets the maximum server-function request body size in bytes.
// A non-positive value restores the 10 MiB default. Applies to subsequently
// served requests.
func SetMaxRequestBytes(parseLimit int64) {
	configMu.Lock()
	defer configMu.Unlock()
	if parseLimit <= 0 {
		parseLimit = 10 << 20
	}
	maxRequestBytes = parseLimit
}

func currentMaxRequestBytes() int64 {
	configMu.RLock()
	defer configMu.RUnlock()
	return maxRequestBytes
}

// Configure sets the base URL the client uses to reach server functions (e.g. an httptest
// server URL in tests, or an absolute origin for a cross-origin API). An empty base URL —
// the default — targets the same origin, which is what a browser-hosted app wants.
func Configure(parseBaseURL string) {
	configMu.Lock()
	defer configMu.Unlock()
	clientBaseURL = parseBaseURL
}

// SetClient overrides the *http.Client used by Call (for custom timeouts, auth transports,
// or a test double). Passing nil restores http.DefaultClient.
func SetClient(parseClient *http.Client) {
	configMu.Lock()
	defer configMu.Unlock()
	if parseClient == nil {
		parseClient = http.DefaultClient
	}
	httpClient = parseClient
}

func currentClient() (*http.Client, string) {
	configMu.RLock()
	defer configMu.RUnlock()
	return httpClient, clientBaseURL
}

// errorEnvelope is the JSON body returned for a failed server function.
type errorEnvelope struct {
	Error string `json:"error"`
}

// ServerError is returned by Call when the server function failed: it carries the HTTP
// status and the server-provided message, so the client sees a typed error with the real
// reason rather than a bare transport failure.
type ServerError struct {
	Status  int
	Message string
}

// Error implements error.
func (parseE *ServerError) Error() string {
	return fmt.Sprintf("server function failed (%d): %s", parseE.Status, parseE.Message)
}

// StatusError is an error a server function returns to control the HTTP status of the failure.
// A plain error still maps to 500; returning a *StatusError (or wrapping one) makes Handle reply
// with the chosen status and message. The status round-trips to the caller as ServerError.Status,
// so the client can branch on 401/404/409/… instead of treating every failure as a 500.
//
//	func getUser(ctx context.Context, req Req) (User, error) {
//	    u, ok := store.Find(req.ID)
//	    if !ok { return User{}, serverfn.NotFound("no such user") }
//	    return u, nil
//	}
type StatusError struct {
	Status  int
	Message string
}

// Error implements error. It is nil-safe so a mistakenly-returned nil *StatusError cannot panic.
func (parseE *StatusError) Error() string {
	if parseE == nil {
		return "internal server error"
	}
	return parseE.Message
}

// NewStatusError builds a StatusError with an explicit HTTP status code.
func NewStatusError(parseStatus int, parseMessage string) *StatusError {
	return &StatusError{Status: parseStatus, Message: parseMessage}
}

// Convenience constructors for the common client-error statuses.
func BadRequest(parseMessage string) *StatusError {
	return NewStatusError(http.StatusBadRequest, parseMessage)
}
func Unauthorized(parseMessage string) *StatusError {
	return NewStatusError(http.StatusUnauthorized, parseMessage)
}
func Forbidden(parseMessage string) *StatusError {
	return NewStatusError(http.StatusForbidden, parseMessage)
}
func NotFound(parseMessage string) *StatusError {
	return NewStatusError(http.StatusNotFound, parseMessage)
}
func Conflict(parseMessage string) *StatusError {
	return NewStatusError(http.StatusConflict, parseMessage)
}
func UnprocessableEntity(parseMessage string) *StatusError {
	return NewStatusError(http.StatusUnprocessableEntity, parseMessage)
}

// statusForError maps a server-function error to an HTTP status and the
// CLIENT-FACING message. A *StatusError anywhere in the error chain selects its
// status and its (author-chosen, safe-to-expose) message. ANY OTHER error maps
// to a generic 500: a plain fmt.Errorf can wrap a DSN, SQL fragment, or
// filesystem path, so its text must never reach the caller — the real error is
// surfaced to operators via the ErrorLogger sink instead.
func statusForError(parseErr error) (int, string) {
	var parseStatusErr *StatusError
	if errors.As(parseErr, &parseStatusErr) {
		// errors.As reports true even for a nil *StatusError wrapped in the error interface; treat
		// that (and a zero status) as a plain 500 rather than dereferencing nil or emitting status 0.
		if parseStatusErr == nil {
			return http.StatusInternalServerError, "internal server error"
		}
		parseStatus := parseStatusErr.Status
		if parseStatus == 0 {
			parseStatus = http.StatusInternalServerError
		}
		return parseStatus, parseStatusErr.Message
	}
	return http.StatusInternalServerError, "internal server error"
}

// Handle registers fn as a JSON POST endpoint at Endpoint(name) on mux. It decodes the
// request body into Req, invokes fn with the request's context, and writes fn's Resp as
// JSON (200) or, on error, a {"error":...} body with a 4xx/5xx status. A non-POST method
// is rejected with 405 and a malformed body with 400.
func Handle[Req, Resp any](parseMux *http.ServeMux, parseName string, parseFn func(context.Context, Req) (Resp, error)) {
	parseMux.HandleFunc(Endpoint(parseName), func(parseW http.ResponseWriter, parseR *http.Request) {
		// A server function is arbitrary user code invoked with attacker-shaped
		// input; a nil-deref/index panic inside it (e.g. an omitted optional
		// field left as a nil slice) would otherwise reset the connection with no
		// clean error. Recover into a generic 500 so every failure has a uniform
		// JSON surface and the process/connection is not disturbed.
		defer func() {
			if parseRec := recover(); parseRec != nil {
				// Keep the real panic value server-side; the client only sees a
				// generic 500 so a panic message cannot leak internal detail.
				logServerError(parseName, fmt.Errorf("panic: %v", parseRec))
				writeError(parseW, http.StatusInternalServerError, "internal server error")
			}
		}()
		if parseR.Method != http.MethodPost {
			writeError(parseW, http.StatusMethodNotAllowed, "server functions require POST")
			return
		}
		// CSRF defense (see SetCSRFProtection): require a JSON Content-Type and
		// reject browser-flagged cross-site requests. Runs before the body is read.
		if currentCSRFProtection() {
			if !isJSONContentType(parseR.Header.Get("Content-Type")) {
				writeError(parseW, http.StatusUnsupportedMediaType, "server functions require Content-Type: application/json")
				return
			}
			if strings.EqualFold(parseR.Header.Get("Sec-Fetch-Site"), "cross-site") {
				writeError(parseW, http.StatusForbidden, "cross-site request rejected")
				return
			}
		}
		var parseReq Req
		// Cap the body: this is the trust boundary, so an uncapped io.ReadAll is a
		// memory-exhaustion DoS. MaxBytesReader also aborts chunked streams.
		parseR.Body = http.MaxBytesReader(parseW, parseR.Body, currentMaxRequestBytes())
		parseBody, parseErr := io.ReadAll(parseR.Body)
		if parseErr != nil {
			var parseMaxErr *http.MaxBytesError
			if errors.As(parseErr, &parseMaxErr) {
				writeError(parseW, http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			// Generic message: the read error's text can name Go internals.
			writeError(parseW, http.StatusBadRequest, "could not read request body")
			return
		}
		// An empty body is a valid zero-value request (a no-argument call).
		if len(bytes.TrimSpace(parseBody)) > 0 {
			if parseErr := json.Unmarshal(parseBody, &parseReq); parseErr != nil {
				// Generic message: a decode error names Go types/fields.
				writeError(parseW, http.StatusBadRequest, "invalid request body")
				return
			}
		}
		parseResp, parseErr := parseFn(parseR.Context(), parseReq)
		if parseErr != nil {
			parseStatus, parseMessage := statusForError(parseErr)
			// A 5xx means either a plain (message-suppressed) error or an explicit
			// server-fault StatusError: log the real error so it is not lost.
			if parseStatus >= http.StatusInternalServerError {
				logServerError(parseName, parseErr)
			}
			writeError(parseW, parseStatus, parseMessage)
			return
		}
		// Marshal BEFORE writing: json.Encoder writes the header implicitly on its
		// first byte, so a marshal failure mid-stream would leave a misleading
		// empty 200. Buffer first so an encode error becomes a real 500.
		parseOut, parseMarshalErr := json.Marshal(parseResp)
		if parseMarshalErr != nil {
			writeError(parseW, http.StatusInternalServerError, "encode response")
			return
		}
		parseW.Header().Set("Content-Type", "application/json")
		_, _ = parseW.Write(parseOut)
	})
}

// Call invokes the named server function over HTTP and decodes its typed response. On
// failure it returns a *ServerError (when the server reported one) or the underlying
// transport/decoding error. Works identically on the server and in the browser.
func Call[Req, Resp any](parseCtx context.Context, parseName string, parseReq Req) (Resp, error) {
	var parseResp Resp
	parseBody, parseErr := json.Marshal(parseReq)
	if parseErr != nil {
		return parseResp, fmt.Errorf("encode request: %w", parseErr)
	}

	parseClient, parseBaseURL := currentClient()
	parseHTTPReq, parseErr := http.NewRequestWithContext(parseCtx, http.MethodPost, parseBaseURL+Endpoint(parseName), bytes.NewReader(parseBody))
	if parseErr != nil {
		return parseResp, fmt.Errorf("build request: %w", parseErr)
	}
	parseHTTPReq.Header.Set("Content-Type", "application/json")

	parseHTTPResp, parseErr := parseClient.Do(parseHTTPReq)
	if parseErr != nil {
		return parseResp, fmt.Errorf("call %q: %w", parseName, parseErr)
	}
	defer parseHTTPResp.Body.Close()

	if parseHTTPResp.StatusCode != http.StatusOK {
		return parseResp, decodeServerError(parseHTTPResp)
	}
	if parseErr := json.NewDecoder(parseHTTPResp.Body).Decode(&parseResp); parseErr != nil {
		return parseResp, fmt.Errorf("decode response: %w", parseErr)
	}
	return parseResp, nil
}

// isJSONContentType reports whether a Content-Type header names application/json
// (ignoring any charset/boundary parameters and case). An unparseable or absent
// header is not JSON. This is the gate that forces a CORS preflight on
// cross-origin browser calls — see SetCSRFProtection.
func isJSONContentType(parseHeader string) bool {
	if parseHeader == "" {
		return false
	}
	parseMediaType, _, parseErr := mime.ParseMediaType(parseHeader)
	if parseErr != nil {
		return false
	}
	return strings.EqualFold(parseMediaType, "application/json")
}

// writeError sends a JSON error envelope with the given status.
func writeError(parseW http.ResponseWriter, parseStatus int, parseMessage string) {
	parseW.Header().Set("Content-Type", "application/json")
	parseW.WriteHeader(parseStatus)
	_ = json.NewEncoder(parseW).Encode(errorEnvelope{Error: parseMessage})
}

// decodeServerError reads a non-200 response into a *ServerError, falling back to the raw
// body when it is not a JSON error envelope.
func decodeServerError(parseResp *http.Response) error {
	parseBody, _ := io.ReadAll(parseResp.Body)
	var parseEnvelope errorEnvelope
	if parseErr := json.Unmarshal(parseBody, &parseEnvelope); parseErr == nil && parseEnvelope.Error != "" {
		return &ServerError{Status: parseResp.StatusCode, Message: parseEnvelope.Error}
	}
	return &ServerError{Status: parseResp.StatusCode, Message: string(bytes.TrimSpace(parseBody))}
}
