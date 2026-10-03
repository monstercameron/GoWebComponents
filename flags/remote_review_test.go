package flags

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestRefreshDiscardsStaleOutOfOrderResponse pins last-REQUEST-wins: when an older, slower fetch
// completes after a newer one, it must not overwrite the newer Set. Otherwise a kill switch that
// the newer response delivered is silently re-enabled by the stale response.
func TestRefreshDiscardsStaleOutOfOrderResponse(parseT *testing.T) {
	parseSlowStarted := make(chan struct{})
	parseSlowRelease := make(chan struct{})
	parseCalls := 0
	parseProvider := NewRemoteProvider(Set{}, func(context.Context) (Set, error) {
		parseCalls++
		if parseCalls == 1 {
			close(parseSlowStarted)
			<-parseSlowRelease
			return BuildSet(map[string]Flag{"feature": {Enabled: true}}, nil), nil // stale: before the kill
		}
		return BuildSet(map[string]Flag{"feature": {Enabled: false}}, nil), nil // newer: killed
	})

	parseDone := make(chan error, 1)
	go func() { parseDone <- parseProvider.Refresh(context.Background()) }()
	<-parseSlowStarted
	if parseErr := parseProvider.Refresh(context.Background()); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if !parseProvider.IsKilled("feature") {
		parseT.Fatal("newer refresh did not kill the feature")
	}
	close(parseSlowRelease)
	if parseErr := <-parseDone; parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if !parseProvider.IsKilled("feature") {
		parseT.Fatal("a stale, slower response re-enabled a killed feature")
	}
}

// TestPollReturnsSleepError pins that a failing sleep function (not a cancelled context) is
// reported instead of ending the poll loop with a nil error that looks like a clean shutdown.
func TestPollReturnsSleepError(parseT *testing.T) {
	parseProvider := NewRemoteProvider(Set{}, func(context.Context) (Set, error) { return Set{}, nil })
	parseSleepErr := errors.New("timer broke")
	parseGot := parseProvider.Poll(context.Background(), time.Second, func(context.Context, time.Duration) error { return parseSleepErr })
	if !errors.Is(parseGot, parseSleepErr) {
		parseT.Fatalf("Poll returned %v, want the sleep error", parseGot)
	}
}
