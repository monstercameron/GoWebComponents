//go:build !js || !wasm

package fetch

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestFetchNativeTransportAndResultHelpers(parseT *testing.T) {
	parseResult := <-Fetch("/api/profile", Options{Method: "POST"})
	if parseResult.Err == nil || parseResult.Err.Error() != "fetch API unavailable in this environment" {
		parseT.Fatalf("expected native fetch fallback error, got %+v", parseResult)
	}

	parseCtx, parseCancel := context.WithCancel(context.Background())
	parseCancel()
	parseUploadCh := Upload(parseCtx, "/api/upload", Options{})
	parseUpdate, parseOk := <-parseUploadCh
	if !parseOk || !parseUpdate.Done || parseUpdate.Result.Err != context.Canceled {
		parseT.Fatalf("expected canceled native upload result, got update=%+v ok=%t", parseUpdate, parseOk)
	}
	if _, parseStillOpen := <-parseUploadCh; parseStillOpen {
		parseT.Fatal("expected native upload fallback channel to close after the terminal update")
	}

	if parseText := (Result{Data: 42}).Text(); parseText != "42" {
		parseT.Fatalf("expected non-string result text to format as 42, got %q", parseText)
	}
	if parseText := (Result{}).Text(); parseText != "" {
		parseT.Fatalf("expected nil result text to be empty, got %q", parseText)
	}
	if parseText := (Result{Data: "ready"}).Text(); parseText != "ready" {
		parseT.Fatalf("expected string-backed result text, got %q", parseText)
	}

	var parseTarget struct {
		Name string `json:"name"`
	}
	if parseErr := (Result{Data: `{"name":"Ada"}`}).DecodeJSON(&parseTarget); parseErr != nil || parseTarget.Name != "Ada" {
		parseT.Fatalf("expected DecodeJSON success, target=%+v err=%v", parseTarget, parseErr)
	}
	if parseErr := (Result{Data: `{"name":"Ada"}`}).DecodeJSON(nil); parseErr == nil || !strings.Contains(parseErr.Error(), "decode target is nil") {
		parseT.Fatalf("expected nil-target DecodeJSON error, got %v", parseErr)
	}
	if parseErr := (Result{}).DecodeJSON(&parseTarget); parseErr == nil || !strings.Contains(parseErr.Error(), "response body is empty") {
		parseT.Fatalf("expected empty-body DecodeJSON error, got %v", parseErr)
	}
	if parseErr := (Result{Data: "{"}).DecodeJSON(&parseTarget); parseErr == nil || !strings.Contains(parseErr.Error(), "decode response json") {
		parseT.Fatalf("expected invalid-json DecodeJSON error, got %v", parseErr)
	}

	if parseMessage := (HTTPError{}).Error(); parseMessage != "request failed" {
		parseT.Fatalf("expected zero HTTPError message, got %q", parseMessage)
	}
	if parseMessage := (HTTPError{Status: 422, StatusText: "Unprocessable Entity"}).Error(); parseMessage != "request failed with status 422 Unprocessable Entity" {
		parseT.Fatalf("unexpected HTTPError message: %q", parseMessage)
	}
	if parseMessage := (HTTPError{Status: 503}).Error(); parseMessage != "request failed with status 503" {
		parseT.Fatalf("unexpected HTTPError status-only message: %q", parseMessage)
	}

	parseHeaders := parseRawHeaders("Content-Type: application/json\nX-Request-ID: abc123\nbroken-header\n")
	if parseHeaders["Content-Type"] != "application/json" || parseHeaders["X-Request-ID"] != "abc123" || len(parseHeaders) != 2 {
		parseT.Fatalf("unexpected parsed headers: %#v", parseHeaders)
	}
	if parseRawHeaders("") != nil {
		parseT.Fatal("expected empty raw headers to return nil")
	}

	ReturnChannel(make(chan Result))
}

func TestFetchNativeResourceHandles(parseT *testing.T) {
	installFetchTestHookContext(parseT)

	parseFetchResource := UseFetch("/api/profile")
	parseFetchState := parseFetchResource.Get()
	if parseFetchState.Error != "fetch API unavailable in this environment" || parseFetchState.Loading {
		parseT.Fatalf("unexpected native UseFetch stub state: %+v", parseFetchState)
	}
	parseFetchResource.Refetch()

	parseResource := UseResource(func(parseCtx context.Context) (string, error) {
		_ = parseCtx
		return "ready", nil
	}, "dep")
	parseState := parseResource.Get()
	if parseState.Loading || parseState.Ready || parseState.Error != nil || parseState.Value != "" {
		parseT.Fatalf("unexpected initial native resource state: %+v", parseState)
	}
	parseResource.Reload()
	parseResource.Cancel()

	var parseZero AsyncResource[string]
	if parseZeroState := parseZero.Get(); parseZeroState.Loading || parseZeroState.Ready || parseZeroState.Error != nil || parseZeroState.Value != "" {
		parseT.Fatalf("unexpected zero-value async resource state: %+v", parseZeroState)
	}
	parseZero.Reload()
	parseZero.Cancel()
}

type testResourceStateSink[T any] struct {
	parseMu    sync.Mutex
	parseState ResourceState[T]
	parseCh    chan ResourceState[T]
}

func newTestResourceStateSink[T any]() *testResourceStateSink[T] {
	return &testResourceStateSink[T]{parseCh: make(chan ResourceState[T], 8)}
}

func (parseS *testResourceStateSink[T]) Set(parseState ResourceState[T]) {
	parseS.parseMu.Lock()
	parseS.parseState = parseState
	parseS.parseMu.Unlock()
	parseS.parseCh <- parseState
}

func (parseS *testResourceStateSink[T]) Update(parseFn func(ResourceState[T]) ResourceState[T]) {
	parseS.parseMu.Lock()
	parseS.parseState = parseFn(parseS.parseState)
	parseState := parseS.parseState
	parseS.parseMu.Unlock()
	parseS.parseCh <- parseState
}

type testResourceCancelRef struct {
	parseMu     sync.Mutex
	parseCancel context.CancelFunc
}

func (parseR *testResourceCancelRef) Get() context.CancelFunc {
	parseR.parseMu.Lock()
	defer parseR.parseMu.Unlock()
	return parseR.parseCancel
}

func (parseR *testResourceCancelRef) Set(parseCancel context.CancelFunc) {
	parseR.parseMu.Lock()
	parseR.parseCancel = parseCancel
	parseR.parseMu.Unlock()
}

func waitResourceState[T any](parseT *testing.T, parseSink *testResourceStateSink[T], parseWant func(ResourceState[T]) bool) ResourceState[T] {
	parseT.Helper()
	parseDeadline := time.After(time.Second)
	for {
		select {
		case parseState := <-parseSink.parseCh:
			if parseWant(parseState) {
				return parseState
			}
		case <-parseDeadline:
			parseSink.parseMu.Lock()
			parseState := parseSink.parseState
			parseSink.parseMu.Unlock()
			parseT.Fatalf("timed out waiting for resource state; last=%+v", parseState)
		}
	}
}

func TestStartResourceLoadPublishesSuccessAndErrors(parseT *testing.T) {
	parseSink := newTestResourceStateSink[string]()
	parseCancelRef := &testResourceCancelRef{}
	parseSeq := new(atomic.Int32)

	startResourceLoad(func(parseCtx context.Context) (string, error) {
		return "ready", nil
	}, parseSink, parseCancelRef, parseSeq)

	parseLoading := waitResourceState(parseT, parseSink, func(parseState ResourceState[string]) bool {
		return parseState.Loading
	})
	if parseLoading.Error != nil {
		parseT.Fatalf("loading state retained error: %+v", parseLoading)
	}
	parseReady := waitResourceState(parseT, parseSink, func(parseState ResourceState[string]) bool {
		return parseState.Ready && parseState.Value == "ready"
	})
	if parseReady.Loading || parseReady.Error != nil {
		parseT.Fatalf("ready state = %+v, want not loading without error", parseReady)
	}

	parseErr := errors.New("loader failed")
	startResourceLoad(func(parseCtx context.Context) (string, error) {
		return "", parseErr
	}, parseSink, parseCancelRef, parseSeq)
	parseFailed := waitResourceState(parseT, parseSink, func(parseState ResourceState[string]) bool {
		return parseState.Error == parseErr
	})
	if parseFailed.Ready || parseFailed.Loading {
		parseT.Fatalf("failed state = %+v, want not ready and not loading", parseFailed)
	}
}

func TestStartResourceLoadCancelsPreviousAndIgnoresStaleCompletion(parseT *testing.T) {
	parseSink := newTestResourceStateSink[string]()
	parseCancelRef := &testResourceCancelRef{}
	parseSeq := new(atomic.Int32)
	parseFirstRelease := make(chan struct{})
	parseFirstDone := make(chan struct{})

	startResourceLoad(func(parseCtx context.Context) (string, error) {
		<-parseFirstRelease
		close(parseFirstDone)
		return "stale", parseCtx.Err()
	}, parseSink, parseCancelRef, parseSeq)
	waitResourceState(parseT, parseSink, func(parseState ResourceState[string]) bool { return parseState.Loading })

	startResourceLoad(func(parseCtx context.Context) (string, error) {
		return "fresh", nil
	}, parseSink, parseCancelRef, parseSeq)
	parseFresh := waitResourceState(parseT, parseSink, func(parseState ResourceState[string]) bool {
		return parseState.Ready && parseState.Value == "fresh"
	})
	if parseFresh.Error != nil {
		parseT.Fatalf("fresh state = %+v, want no error", parseFresh)
	}

	close(parseFirstRelease)
	select {
	case <-parseFirstDone:
	case <-time.After(time.Second):
		parseT.Fatal("first loader did not unblock")
	}
	time.Sleep(10 * time.Millisecond)
	parseSink.parseMu.Lock()
	parseFinal := parseSink.parseState
	parseSink.parseMu.Unlock()
	if parseFinal.Value != "fresh" || !parseFinal.Ready {
		parseT.Fatalf("stale completion overwrote fresh state: %+v", parseFinal)
	}
}
