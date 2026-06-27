package serverfn

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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

// TestCallPropagatesServerError proves a server function error reaches the client as a
// typed *ServerError carrying the server's message and status.
func TestCallPropagatesServerError(parseT *testing.T) {
	newTestServer(parseT, func(parseMux *http.ServeMux) {
		Handle(parseMux, "Fail", func(parseCtx context.Context, parseReq echoReq) (echoResp, error) {
			return echoResp{}, errors.New("name is taken")
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
