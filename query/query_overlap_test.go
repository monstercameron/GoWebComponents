package query

import (
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

// TestMutateOverlappingFailuresRollBackToPreMutationState pins that when two
// overlapping optimistic mutations both fail, the cache returns to the value
// from before the first one, not to the first mutation's optimistic guess.
func TestMutateOverlappingFailuresRollBackToPreMutationState(parseT *testing.T) {
	synctest.Test(parseT, func(parseT *testing.T) {
		parseCache := New()
		parseCache.Set("count", 10)

		parseReleaseA := make(chan struct{})
		parseReleaseB := make(chan struct{})
		parseDoneA := make(chan Result[int], 1)
		parseDoneB := make(chan Result[int], 1)
		MutateAsync(parseCache, "count", 11, func() (int, error) {
			<-parseReleaseA
			return 0, errors.New("A failed")
		}, func(parseRes Result[int]) { parseDoneA <- parseRes })
		MutateAsync(parseCache, "count", 12, func() (int, error) {
			<-parseReleaseB
			return 0, errors.New("B failed")
		}, func(parseRes Result[int]) { parseDoneB <- parseRes })
		synctest.Wait()
		if parseGot := Snapshot[int](parseCache, "count").Data; parseGot != 12 {
			parseT.Fatalf("expected newest optimistic value 12, got %d", parseGot)
		}

		close(parseReleaseA)
		<-parseDoneA
		if parseGot := Snapshot[int](parseCache, "count").Data; parseGot != 12 {
			parseT.Fatalf("B is still pending; expected its optimistic 12 to stay, got %d", parseGot)
		}
		close(parseReleaseB)
		<-parseDoneB
		if parseGot := Snapshot[int](parseCache, "count").Data; parseGot != 10 {
			parseT.Fatalf("both mutations failed; expected rollback to pre-mutation 10, got %d", parseGot)
		}
	})
}

// TestMutateOverlappingEarlierSuccessSurvivesLaterFailure pins that a later
// failure rolls back to the earlier mutation's committed value.
func TestMutateOverlappingEarlierSuccessSurvivesLaterFailure(parseT *testing.T) {
	synctest.Test(parseT, func(parseT *testing.T) {
		parseCache := New()
		parseCache.Set("count", 10)

		parseReleaseA := make(chan struct{})
		parseReleaseB := make(chan struct{})
		parseDoneA := make(chan Result[int], 1)
		parseDoneB := make(chan Result[int], 1)
		MutateAsync(parseCache, "count", 11, func() (int, error) {
			<-parseReleaseA
			return 21, nil
		}, func(parseRes Result[int]) { parseDoneA <- parseRes })
		MutateAsync(parseCache, "count", 12, func() (int, error) {
			<-parseReleaseB
			return 0, errors.New("B failed")
		}, func(parseRes Result[int]) { parseDoneB <- parseRes })
		synctest.Wait()

		close(parseReleaseA)
		<-parseDoneA
		close(parseReleaseB)
		<-parseDoneB
		if parseGot := Snapshot[int](parseCache, "count").Data; parseGot != 21 {
			parseT.Fatalf("expected A's committed 21 after B failed, got %d", parseGot)
		}
	})
}

// TestInvalidateDiscardsInFlightFetchResult pins that a fetch started before an
// Invalidate does not commit its pre-invalidation data as fresh.
func TestInvalidateDiscardsInFlightFetchResult(parseT *testing.T) {
	synctest.Test(parseT, func(parseT *testing.T) {
		parseCache := New(WithStaleTime(time.Hour))

		parseRelease := make(chan struct{})
		parseStarted := make(chan struct{})
		parseDone := make(chan struct{})
		go func() {
			Fetch(parseCache, "k", func() (string, error) {
				close(parseStarted)
				<-parseRelease
				return "old", nil
			})
			close(parseDone)
		}()
		<-parseStarted

		parseCache.Invalidate("k")
		close(parseRelease)
		<-parseDone

		parseSnap := Snapshot[string](parseCache, "k")
		if parseSnap.Data == "old" && !parseSnap.Stale {
			parseT.Fatalf("pre-invalidation data committed as fresh: %+v", parseSnap)
		}
	})
}
