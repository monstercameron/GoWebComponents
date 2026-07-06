package serverfn

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type echoReq struct {
	Name string `json:"name"`
}

type echoResp struct {
	Greeting string `json:"greeting"`
}

// newTestServer registers the given handler-builder on a fresh mux behind an httptest
// server, points the client at it, and returns a cleanup.
func newTestServer(parseT *testing.T, parseRegister func(*http.ServeMux)) {
	parseT.Helper()
	parseMux := http.NewServeMux()
	parseRegister(parseMux)
	parseServer := httptest.NewServer(parseMux)
	Configure(parseServer.URL)
	parseT.Cleanup(func() {
		parseServer.Close()
		Configure("")
	})
}

// TestHandleAndCallRoundTrip proves a typed server function is invoked end-to-end over
// HTTP: the request struct is decoded, the function runs, and the typed response comes
// back — the same path the browser exercises via Fetch-backed net/http.
func TestHandleAndCallRoundTrip(parseT *testing.T) {
	newTestServer(parseT, func(parseMux *http.ServeMux) {
		Handle(parseMux, "Greet", func(parseCtx context.Context, parseReq echoReq) (echoResp, error) {
			return echoResp{Greeting: "Hello, " + parseReq.Name}, nil
		})
	})

	parseResp, parseErr := Call[echoReq, echoResp](context.Background(), "Greet", echoReq{Name: "Ada"})
	if parseErr != nil {
		parseT.Fatalf("unexpected error: %v", parseErr)
	}
	if parseResp.Greeting != "Hello, Ada" {
		parseT.Fatalf("expected greeting, got %q", parseResp.Greeting)
	}
}

// TestCallPropagatesServerError proves a server function's DELIBERATE message reaches the
// client as a typed *ServerError with its status. Under the error-disclosure policy an
// author-chosen message must travel via a StatusError (a plain error is suppressed to a
// generic 500 — see TestHandlePlainErrorIsGenericToClient); NewStatusError(500, …) is how a
// server-fault exposes a safe message.
func TestCallPropagatesServerError(parseT *testing.T) {
	newTestServer(parseT, func(parseMux *http.ServeMux) {
		Handle(parseMux, "Fail", func(parseCtx context.Context, parseReq echoReq) (echoResp, error) {
			return echoResp{}, NewStatusError(http.StatusInternalServerError, "name is taken")
		})
	})

	_, parseErr := Call[echoReq, echoResp](context.Background(), "Fail", echoReq{Name: "Ada"})
	if parseErr == nil {
		parseT.Fatal("expected an error")
	}
	var parseServerErr *ServerError
	if !errors.As(parseErr, &parseServerErr) {
		parseT.Fatalf("expected a *ServerError, got %T: %v", parseErr, parseErr)
	}
	if parseServerErr.Status != http.StatusInternalServerError || parseServerErr.Message != "name is taken" {
		parseT.Fatalf("unexpected server error: %+v", parseServerErr)
	}
}

// TestHandleMapsStatusErrorToChosenStatus proves a *StatusError selects the HTTP status, which
// round-trips to the caller as ServerError.Status (so the client can branch on 404 vs 500).
func TestHandleMapsStatusErrorToChosenStatus(parseT *testing.T) {
	newTestServer(parseT, func(parseMux *http.ServeMux) {
		Handle(parseMux, "Missing", func(parseCtx context.Context, parseReq echoReq) (echoResp, error) {
			return echoResp{}, NotFound("no such user")
		})
	})

	_, parseErr := Call[echoReq, echoResp](context.Background(), "Missing", echoReq{Name: "Ada"})
	var parseServerErr *ServerError
	if !errors.As(parseErr, &parseServerErr) {
		parseT.Fatalf("expected a *ServerError, got %T: %v", parseErr, parseErr)
	}
	if parseServerErr.Status != http.StatusNotFound {
		parseT.Fatalf("expected 404 round-tripped, got %d", parseServerErr.Status)
	}
	if parseServerErr.Message != "no such user" {
		parseT.Fatalf("expected the StatusError message preserved, got %q", parseServerErr.Message)
	}
}

// TestHandleMapsWrappedStatusError proves the status is found anywhere in the error chain (the
// handler can wrap a StatusError with context via fmt.Errorf("...: %w", ...)).
func TestHandleMapsWrappedStatusError(parseT *testing.T) {
	newTestServer(parseT, func(parseMux *http.ServeMux) {
		Handle(parseMux, "Wrapped", func(parseCtx context.Context, parseReq echoReq) (echoResp, error) {
			return echoResp{}, fmt.Errorf("create user: %w", Conflict("already exists"))
		})
	})

	_, parseErr := Call[echoReq, echoResp](context.Background(), "Wrapped", echoReq{})
	var parseServerErr *ServerError
	if !errors.As(parseErr, &parseServerErr) || parseServerErr.Status != http.StatusConflict {
		parseT.Fatalf("expected a wrapped StatusError to map to 409, got %+v (%v)", parseServerErr, parseErr)
	}
	// Only the StatusError's own message is transmitted — the fmt.Errorf wrapper text must not leak.
	if parseServerErr.Message != "already exists" {
		parseT.Fatalf("expected the StatusError message, got %q", parseServerErr.Message)
	}
}

// TestStatusForErrorEdgeCases pins the mapping's defensive branches: a zero-status StatusError and
// a mistakenly-nil *StatusError both degrade to 500 (no panic, no status-0 header), and a plain
// error keeps its message.
func TestStatusForErrorEdgeCases(parseT *testing.T) {
	if parseStatus, _ := statusForError(NewStatusError(0, "no status")); parseStatus != http.StatusInternalServerError {
		parseT.Fatalf("zero-status StatusError must map to 500, got %d", parseStatus)
	}
	if parseStatus, parseMsg := statusForError(NotFound("missing")); parseStatus != http.StatusNotFound || parseMsg != "missing" {
		parseT.Fatalf("NotFound must map to 404/missing, got %d/%q", parseStatus, parseMsg)
	}
	// Disclosure policy: a plain error's own text is NEVER exposed — it maps to a generic 500.
	if parseStatus, parseMsg := statusForError(errors.New("dsn=postgres://secret")); parseStatus != http.StatusInternalServerError || parseMsg != "internal server error" {
		parseT.Fatalf("plain error must map to a GENERIC 500, got %d/%q", parseStatus, parseMsg)
	}

	// A nil *StatusError typed as error (the Go nil-interface anti-pattern) must not panic.
	func() {
		defer func() {
			if parseR := recover(); parseR != nil {
				parseT.Fatalf("statusForError panicked on a nil *StatusError: %v", parseR)
			}
		}()
		var parseNilTyped error = (*StatusError)(nil)
		if parseStatus, _ := statusForError(parseNilTyped); parseStatus != http.StatusInternalServerError {
			parseT.Fatalf("nil *StatusError must map to 500, got %d", parseStatus)
		}
	}()
}

// TestHandlePlainErrorIsGenericToClient pins the error-disclosure policy: a plain error's
// text (which can wrap a DSN/SQL/path) is NEVER sent to the caller — it maps to 500 with a
// generic message — while the REAL error is still delivered to the server-side ErrorLogger.
func TestHandlePlainErrorIsGenericToClient(parseT *testing.T) {
	parseOldLogger := errorLogger
	var parseLogged error
	SetErrorLogger(func(parseName string, parseErr error) { parseLogged = parseErr })
	parseT.Cleanup(func() { SetErrorLogger(parseOldLogger) })

	newTestServer(parseT, func(parseMux *http.ServeMux) {
		Handle(parseMux, "Boom", func(parseCtx context.Context, parseReq echoReq) (echoResp, error) {
			return echoResp{}, errors.New("dsn=postgres://user:pw@db/secret")
		})
	})

	_, parseErr := Call[echoReq, echoResp](context.Background(), "Boom", echoReq{})
	var parseServerErr *ServerError
	if !errors.As(parseErr, &parseServerErr) || parseServerErr.Status != http.StatusInternalServerError {
		parseT.Fatalf("expected a plain error to stay 500, got %+v", parseServerErr)
	}
	// The secret-bearing text must NOT reach the client.
	if strings.Contains(parseServerErr.Message, "postgres") || strings.Contains(parseServerErr.Message, "secret") {
		parseT.Fatalf("plain error text leaked to client: %q", parseServerErr.Message)
	}
	if parseServerErr.Message != "internal server error" {
		parseT.Fatalf("expected a generic client message, got %q", parseServerErr.Message)
	}
	// ...but the operator-side logger DID receive the real error.
	if parseLogged == nil || !strings.Contains(parseLogged.Error(), "postgres") {
		parseT.Fatalf("expected the real error delivered to ErrorLogger, got %v", parseLogged)
	}
}

// TestHandleRejectsNonPost proves the endpoint only accepts POST.
func TestHandleRejectsNonPost(parseT *testing.T) {
	parseMux := http.NewServeMux()
	Handle(parseMux, "Greet", func(parseCtx context.Context, parseReq echoReq) (echoResp, error) {
		return echoResp{}, nil
	})
	parseServer := httptest.NewServer(parseMux)
	parseT.Cleanup(parseServer.Close)

	parseResp, parseErr := http.Get(parseServer.URL + Endpoint("Greet"))
	if parseErr != nil {
		parseT.Fatalf("GET failed: %v", parseErr)
	}
	defer parseResp.Body.Close()
	if parseResp.StatusCode != http.StatusMethodNotAllowed {
		parseT.Fatalf("expected 405 for GET, got %d", parseResp.StatusCode)
	}
}

// TestCallContextCancelled proves a cancelled context aborts the call rather than hanging.
func TestCallContextCancelled(parseT *testing.T) {
	newTestServer(parseT, func(parseMux *http.ServeMux) {
		Handle(parseMux, "Greet", func(parseCtx context.Context, parseReq echoReq) (echoResp, error) {
			return echoResp{Greeting: "hi"}, nil
		})
	})

	parseCtx, parseCancel := context.WithCancel(context.Background())
	parseCancel() // already cancelled
	if _, parseErr := Call[echoReq, echoResp](parseCtx, "Greet", echoReq{}); parseErr == nil {
		parseT.Fatal("expected a context-cancelled error")
	}
}

// TestEndpoint proves the route layout is stable and prefixed.
func TestEndpoint(parseT *testing.T) {
	if parseGot := Endpoint("CreateOrder"); parseGot != "/_gwc/fn/CreateOrder" {
		parseT.Fatalf("unexpected endpoint %q", parseGot)
	}
}
