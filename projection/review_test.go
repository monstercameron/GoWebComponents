package projection_test

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v7/projection"
)

// TestInsertOfResidentKeyIsRefused pins that re-delivering an insert for a key
// that is already resident cannot create a second row under the same key: Get
// and Update would then address only one of the two and Remove would leave the
// other behind.
func TestInsertOfResidentKeyIsRefused(parseT *testing.T) {
	parseProjection := buildProjection(parseT, projection.Options{})
	parseInsert := projection.Op{Kind: projection.OpInsert, Key: "a", Payload: encodeItem(item{Name: "one", Price: 1})}
	if parseErr := parseProjection.Apply([]projection.Op{parseInsert}); parseErr != nil {
		parseT.Fatalf("first insert: %v", parseErr)
	}
	if parseErr := parseProjection.Apply([]projection.Op{parseInsert}); parseErr == nil {
		parseT.Fatal("a second insert of the same key was accepted")
	}
	if parseProjection.Len() != 1 {
		parseT.Fatalf("Len = %d after a duplicate insert, want 1", parseProjection.Len())
	}
	if parseErr := parseProjection.Apply([]projection.Op{{Kind: projection.OpRemove, Key: "a"}}); parseErr != nil {
		parseT.Fatalf("remove: %v", parseErr)
	}
	if parseProjection.Len() != 0 {
		parseT.Fatalf("Len = %d after removing the only key, want 0", parseProjection.Len())
	}
}

// TestRetryBackoffSaturatesInsteadOfOverflowing pins that doubling a very large
// backoff never wraps negative, which sleepers treat as "retry immediately".
func TestRetryBackoffSaturatesInsteadOfOverflowing(parseT *testing.T) {
	parseClient := &scriptedFailureClient{outcomes: []error{
		projection.WorkerDeath("a"), projection.WorkerDeath("b"), projection.WorkerDeath("c"), projection.WorkerDeath("d"),
	}}
	var parseSleeps []time.Duration
	parseResilient, parseErr := projection.NewResilient(parseClient, projection.Policy{
		MaxAttempts:         4,
		Backoff:             time.Duration(math.MaxInt64/2 + 1),
		RetryMayHaveApplied: true,
		Sleep: func(_ context.Context, parseDuration time.Duration) error {
			parseSleeps = append(parseSleeps, parseDuration)
			return nil
		},
	})
	if parseErr != nil {
		parseT.Fatalf("NewResilient: %v", parseErr)
	}
	_, _ = parseResilient.Send(context.Background(), "cmd", nil)
	for parseIndex, parseSleep := range parseSleeps {
		if parseSleep <= 0 {
			parseT.Fatalf("sleep %d = %v; backoff wrapped negative: %v", parseIndex, parseSleep, parseSleeps)
		}
	}
}

// TestRetryStopsWhenCallerDeadlineExpired pins that an expired caller context
// ends the retry loop even when the backoff is zero (sleepWithContext does not
// look at the context for a zero delay).
func TestRetryStopsWhenCallerDeadlineExpired(parseT *testing.T) {
	parseClient := &scriptedFailureClient{outcomes: []error{
		fmt.Errorf("slow: %w", context.DeadlineExceeded),
		fmt.Errorf("slow: %w", context.DeadlineExceeded),
		fmt.Errorf("slow: %w", context.DeadlineExceeded),
		fmt.Errorf("slow: %w", context.DeadlineExceeded),
	}}
	parseResilient, _ := projection.NewResilient(parseClient, projection.Policy{MaxAttempts: 4, RetryMayHaveApplied: true})
	parseCtx, parseCancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer parseCancel()
	<-parseCtx.Done()
	_, _ = parseResilient.Send(parseCtx, "cmd", nil)
	if parseClient.calls != 1 {
		parseT.Fatalf("command sent %d times on an already-expired caller context, want 1", parseClient.calls)
	}
}
