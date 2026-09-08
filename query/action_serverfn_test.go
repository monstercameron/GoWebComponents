package query_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/query"
	"github.com/monstercameron/GoWebComponents/v6/serverfn"
)

type likeReq struct {
	Delta int `json:"delta"`
}
type likeResp struct {
	Total int `json:"total"`
}

// TestOptimisticActionOverServerFunction is the FB2 one-liner end-to-end: a server function
// (the authority for the like count) is invoked through query.MutateAsync, so the UI sees
// the optimistic value immediately and the cache commits the server's authoritative result
// when the call settles — the "Action" pattern (optimistic update + server reconcile +
// rollback) composed from the shipped query + serverfn primitives, no new glue.
func TestOptimisticActionOverServerFunction(parseT *testing.T) {
	parseServerTotal := 41
	parseMux := http.NewServeMux()
	serverfn.Handle(parseMux, "Like", func(parseCtx context.Context, parseReq likeReq) (likeResp, error) {
		parseServerTotal += parseReq.Delta
		return likeResp{Total: parseServerTotal}, nil
	})
	parseServer := httptest.NewServer(parseMux)
	defer parseServer.Close()
	serverfn.Configure(parseServer.URL)
	defer serverfn.Configure("")

	parseCache := query.New(query.WithStaleTime(time.Hour))
	parseCache.Set("likes", 41)

	parseSettled := make(chan query.Result[int], 1)
	query.MutateAsync(parseCache, "likes", 42, func() (int, error) {
		parseResp, parseErr := serverfn.Call[likeReq, likeResp](context.Background(), "Like", likeReq{Delta: 1})
		return parseResp.Total, parseErr
	}, func(parseRes query.Result[int]) { parseSettled <- parseRes })

	// Optimistic value is on screen before the round-trip completes.
	if parsePeek, _ := parseCache.Peek("likes"); parsePeek != 42 {
		parseT.Fatalf("expected optimistic 42 immediately, got %v", parsePeek)
	}

	select {
	case parseRes := <-parseSettled:
		if parseRes.Err != nil || parseRes.Data != 42 {
			parseT.Fatalf("expected the server's authoritative 42, got %+v", parseRes)
		}
	case <-time.After(2 * time.Second):
		parseT.Fatal("the optimistic action never settled")
	}
	if parsePeek, _ := parseCache.Peek("likes"); parsePeek != 42 {
		parseT.Fatalf("expected the committed server value 42, got %v", parsePeek)
	}
}
