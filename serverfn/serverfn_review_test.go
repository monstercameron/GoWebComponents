package serverfn

import (
	"context"
	"net/http"
	"testing"
)

// TestHandleClampsNonErrorStatusToServerFault pins that a server function
// returning a StatusError whose status is not a 4xx/5xx cannot turn a failure
// into a 2xx/3xx reply: Call decodes a 200 as the typed response, so the caller
// would receive a zero-value success for a failed call.
func TestHandleClampsNonErrorStatusToServerFault(parseT *testing.T) {
	for _, parseStatus := range []int{http.StatusOK, http.StatusNoContent, http.StatusFound, 99, 1000, -1} {
		newTestServer(parseT, func(parseMux *http.ServeMux) {
			Handle(parseMux, "Odd", func(parseCtx context.Context, parseReq echoReq) (echoResp, error) {
				return echoResp{}, NewStatusError(parseStatus, "boom")
			})
		})
		_, parseErr := Call[echoReq, echoResp](context.Background(), "Odd", echoReq{Name: "x"})
		parseServerErr, parseOk := parseErr.(*ServerError)
		if !parseOk {
			parseT.Fatalf("status %d: expected *ServerError, got %v", parseStatus, parseErr)
		}
		if parseServerErr.Status != http.StatusInternalServerError {
			parseT.Fatalf("status %d: reply status = %d, want 500", parseStatus, parseServerErr.Status)
		}
	}
}

// TestCallBoundsErrorBodyRead pins that a failed response with an enormous body
// does not make Call buffer all of it.
func TestCallBoundsErrorBodyRead(parseT *testing.T) {
	newTestServer(parseT, func(parseMux *http.ServeMux) {
		parseMux.HandleFunc(Endpoint("Huge"), func(parseW http.ResponseWriter, parseR *http.Request) {
			parseW.WriteHeader(http.StatusBadGateway)
			parseChunk := make([]byte, 1<<20)
			for parseI := 0; parseI < 8; parseI++ {
				if _, parseErr := parseW.Write(parseChunk); parseErr != nil {
					return
				}
			}
		})
	})
	_, parseErr := Call[echoReq, echoResp](context.Background(), "Huge", echoReq{})
	parseServerErr, parseOk := parseErr.(*ServerError)
	if !parseOk {
		parseT.Fatalf("expected *ServerError, got %v", parseErr)
	}
	if len(parseServerErr.Message) > maxErrorBodyBytes {
		parseT.Fatalf("error message holds %d bytes, want at most %d", len(parseServerErr.Message), maxErrorBodyBytes)
	}
}
