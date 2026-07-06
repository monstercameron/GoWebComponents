package serverfn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHandleRejectsOversizedBody pins that a request body over the configured
// limit is rejected with 413 rather than read unbounded into memory.
func TestHandleRejectsOversizedBody(parseT *testing.T) {
	SetMaxRequestBytes(64)
	parseT.Cleanup(func() { SetMaxRequestBytes(0) })

	parseMux := http.NewServeMux()
	Handle(parseMux, "Echo", func(parseCtx context.Context, parseReq echoReq) (echoResp, error) {
		return echoResp{Greeting: parseReq.Name}, nil
	})
	parseServer := httptest.NewServer(parseMux)
	parseT.Cleanup(parseServer.Close)

	parseBig := `{"name":"` + strings.Repeat("x", 4096) + `"}`
	parseResp, parseErr := http.Post(parseServer.URL+Endpoint("Echo"), "application/json", strings.NewReader(parseBig))
	if parseErr != nil {
		parseT.Fatalf("post: %v", parseErr)
	}
	defer parseResp.Body.Close()
	if parseResp.StatusCode != http.StatusRequestEntityTooLarge {
		parseT.Fatalf("status = %d, want 413", parseResp.StatusCode)
	}
}

// TestHandleRecoversPanic pins that a panic inside a server function becomes a
// clean 500 rather than resetting the connection.
func TestHandleRecoversPanic(parseT *testing.T) {
	newTestServer(parseT, func(parseMux *http.ServeMux) {
		Handle(parseMux, "Boom", func(parseCtx context.Context, parseReq echoReq) (echoResp, error) {
			panic("kaboom")
		})
	})

	_, parseErr := Call[echoReq, echoResp](context.Background(), "Boom", echoReq{Name: "x"})
	if parseErr == nil {
		parseT.Fatal("expected error from panicking server function")
	}
	var parseServerErr *ServerError
	if !asServerError(parseErr, &parseServerErr) || parseServerErr.Status != http.StatusInternalServerError {
		parseT.Fatalf("expected 500 ServerError, got %v", parseErr)
	}
}

func asServerError(parseErr error, parseTarget **ServerError) bool {
	parseServerErr, parseOk := parseErr.(*ServerError)
	if parseOk {
		*parseTarget = parseServerErr
	}
	return parseOk
}
