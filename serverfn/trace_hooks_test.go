package serverfn

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubTracePropagator struct{}

func (stubTracePropagator) Extract(parseCtx context.Context, _ http.Header) context.Context {
	return parseCtx
}

func (stubTracePropagator) Inject(_ context.Context, parseHeader http.Header) {
	parseHeader.Set("Traceparent", "injected")
}

func TestTraceHooksAreNoOpsByDefault(parseT *testing.T) {
	parseMux := http.NewServeMux()
	Handle(parseMux, "plain", func(_ context.Context, parseReq map[string]string) (map[string]string, error) {
		return parseReq, nil
	})
	parseSrv := httptest.NewServer(parseMux)
	defer parseSrv.Close()
	Configure(parseSrv.URL)
	defer Configure("")
	if _, parseErr := Call[map[string]string, map[string]string](context.Background(), "plain", nil); parseErr != nil {
		parseT.Fatal(parseErr)
	}
}

func TestCallInjectsAndHandleReportsStatusAndError(parseT *testing.T) {
	SetTracePropagator(stubTracePropagator{})
	var parseGotStatus int
	var parseGotErr error
	var parseGotHeader string
	SetServerSpanHook(func(parseCtx context.Context, _ string, parseHeader http.Header) (context.Context, func(int, error)) {
		parseGotHeader = parseHeader.Get("Traceparent")
		return parseCtx, func(parseStatus int, parseErr error) { parseGotStatus, parseGotErr = parseStatus, parseErr }
	})
	defer SetTracePropagator(nil)
	defer SetServerSpanHook(nil)

	parseMux := http.NewServeMux()
	Handle(parseMux, "fail", func(context.Context, struct{}) (struct{}, error) { return struct{}{}, errors.New("kaboom") })
	parseSrv := httptest.NewServer(parseMux)
	defer parseSrv.Close()
	Configure(parseSrv.URL)
	defer Configure("")

	_, _ = Call[struct{}, struct{}](context.Background(), "fail", struct{}{})
	if parseGotHeader != "injected" || parseGotStatus != http.StatusInternalServerError || parseGotErr == nil {
		parseT.Fatalf("header=%q status=%d err=%v", parseGotHeader, parseGotStatus, parseGotErr)
	}
}
