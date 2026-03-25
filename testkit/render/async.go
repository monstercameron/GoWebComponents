package render

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

const (
	FailureCodeHydrationMismatch = "hydration_mismatch"
	FailureCodeLoaderFailure     = "loader_failure"
	FailureCodeRouteGuardFailure = "route_guard_failure"
	FailureCodeCacheConflict     = "cache_conflict"
	FailureCodeOfflineReplay     = "offline_replay"
)

// FailureError represents one deterministic failure-injection error payload.
type FailureError struct {
	Code    string
	Message string
}

// Error returns the printable failure-injection error message.
func (parseE FailureError) Error() string {
	parseCode := strings.TrimSpace(parseE.Code)
	parseMessage := strings.TrimSpace(parseE.Message)
	if parseCode == "" {
		parseCode = "failure"
	}
	if parseMessage == "" {
		return parseCode
	}
	return fmt.Sprintf("%s: %s", parseCode, parseMessage)
}

type ResourceAttempt struct {
	Index     int
	Cancelled bool
}

type resourceOutcome[T any] struct {
	value T
	err   error
}

type resourcePending[T any] struct {
	index int
	done  chan resourceOutcome[T]
}

// ResourceController provides deterministic control over one async resource loader in tests.
type ResourceController[T any] struct {
	mu       sync.Mutex
	attempts []ResourceAttempt
	pending  *resourcePending[T]
	started  chan int
}

// NewResourceController creates one async controller for fetch/resource tests.
func NewResourceController[T any]() *ResourceController[T] {
	return &ResourceController[T]{
		started: make(chan int, 16),
	}
}

// Loader returns a loader function compatible with fetch.UseResource or fetch.UseCachedResource.
func (parseC *ResourceController[T]) Loader() func(context.Context) (T, error) {
	return parseC.Await
}

// Await blocks until the current attempt is resolved, rejected, or cancelled.
func (parseC *ResourceController[T]) Await(parseCtx context.Context) (T, error) {
	var parseZero T
	if parseC == nil {
		return parseZero, nil
	}

	parsePending := &resourcePending[T]{done: make(chan resourceOutcome[T], 1)}
	parseC.mu.Lock()
	parsePending.index = len(parseC.attempts) + 1
	parseC.attempts = append(parseC.attempts, ResourceAttempt{Index: parsePending.index})
	parseC.pending = parsePending
	parseC.mu.Unlock()

	select {
	case parseC.started <- parsePending.index:
	default:
	}

	select {
	case parseOutcome := <-parsePending.done:
		return parseOutcome.value, parseOutcome.err
	case <-parseCtx.Done():
		parseC.mu.Lock()
		if parseC.pending == parsePending {
			parseC.pending = nil
			parseC.attempts[parsePending.index-1].Cancelled = true
		}
		parseC.mu.Unlock()
		return parseZero, parseCtx.Err()
	}
}

// Resolve completes the current pending attempt successfully.
func (parseC *ResourceController[T]) Resolve(parseValue T) {
	if parseC == nil {
		return
	}
	parseC.finish(resourceOutcome[T]{value: parseValue})
}

// Reject completes the current pending attempt with an error.
func (parseC *ResourceController[T]) Reject(parseErr error) {
	if parseC == nil {
		return
	}
	parseC.finish(resourceOutcome[T]{err: parseErr})
}

// RejectCacheConflict completes the current pending attempt with a cache-conflict failure.
func (parseC *ResourceController[T]) RejectCacheConflict(parseEntity string) {
	if parseC == nil {
		return
	}
	parseC.Reject(BuildCacheConflictError(parseEntity))
}

// RejectOfflineReplay completes the current pending attempt with an offline-replay failure.
func (parseC *ResourceController[T]) RejectOfflineReplay(parseEntity string, parseReason string) {
	if parseC == nil {
		return
	}
	parseC.Reject(BuildOfflineReplayError(parseEntity, parseReason))
}

// Cancel completes the current pending attempt with context cancellation.
func (parseC *ResourceController[T]) Cancel() {
	if parseC == nil {
		return
	}
	parseC.finish(resourceOutcome[T]{err: context.Canceled}, true)
}

// Pending reports whether one attempt is currently stalled.
func (parseC *ResourceController[T]) Pending() bool {
	if parseC == nil {
		return false
	}
	parseC.mu.Lock()
	defer parseC.mu.Unlock()
	return parseC.pending != nil
}

// AttemptCount reports how many attempts have started.
func (parseC *ResourceController[T]) AttemptCount() int {
	if parseC == nil {
		return 0
	}
	parseC.mu.Lock()
	defer parseC.mu.Unlock()
	return len(parseC.attempts)
}

// Attempts returns a snapshot of all started attempts.
func (parseC *ResourceController[T]) Attempts() []ResourceAttempt {
	if parseC == nil {
		return nil
	}
	parseC.mu.Lock()
	defer parseC.mu.Unlock()
	return append([]ResourceAttempt(nil), parseC.attempts...)
}

// Started returns a channel that receives each started attempt index.
func (parseC *ResourceController[T]) Started() <-chan int {
	if parseC == nil {
		return nil
	}
	return parseC.started
}

// BuildFailureError constructs one typed failure-injection error.
func BuildFailureError(parseCode string, parseMessage string) error {
	parseTrimmedCode := strings.TrimSpace(parseCode)
	if parseTrimmedCode == "" {
		parseTrimmedCode = "failure"
	}
	return FailureError{
		Code:    parseTrimmedCode,
		Message: strings.TrimSpace(parseMessage),
	}
}

// BuildHydrationMismatchError constructs one hydration mismatch failure error.
func BuildHydrationMismatchError(parsePath string, parseReason string) error {
	parseTrimmedPath := strings.TrimSpace(parsePath)
	parseTrimmedReason := strings.TrimSpace(parseReason)
	parseMessage := parseTrimmedReason
	if parseTrimmedPath != "" {
		if parseMessage == "" {
			parseMessage = "path=" + parseTrimmedPath
		} else {
			parseMessage = "path=" + parseTrimmedPath + " reason=" + parseMessage
		}
	}
	return BuildFailureError(FailureCodeHydrationMismatch, parseMessage)
}

// BuildLoaderFailureError constructs one loader failure error.
func BuildLoaderFailureError(parsePath string, parseReason string) error {
	parseTrimmedPath := strings.TrimSpace(parsePath)
	parseTrimmedReason := strings.TrimSpace(parseReason)
	parseMessage := parseTrimmedReason
	if parseTrimmedPath != "" {
		if parseMessage == "" {
			parseMessage = "path=" + parseTrimmedPath
		} else {
			parseMessage = "path=" + parseTrimmedPath + " reason=" + parseMessage
		}
	}
	return BuildFailureError(FailureCodeLoaderFailure, parseMessage)
}

// BuildRouteGuardFailureError constructs one route-guard failure error.
func BuildRouteGuardFailureError(parsePath string, parseReason string) error {
	parseTrimmedPath := strings.TrimSpace(parsePath)
	parseTrimmedReason := strings.TrimSpace(parseReason)
	parseMessage := parseTrimmedReason
	if parseTrimmedPath != "" {
		if parseMessage == "" {
			parseMessage = "path=" + parseTrimmedPath
		} else {
			parseMessage = "path=" + parseTrimmedPath + " reason=" + parseMessage
		}
	}
	return BuildFailureError(FailureCodeRouteGuardFailure, parseMessage)
}

// BuildCacheConflictError constructs one cache-conflict failure error.
func BuildCacheConflictError(parseEntity string) error {
	parseTrimmedEntity := strings.TrimSpace(parseEntity)
	if parseTrimmedEntity == "" {
		parseTrimmedEntity = "cache"
	}
	return BuildFailureError(FailureCodeCacheConflict, "entity="+parseTrimmedEntity)
}

// BuildOfflineReplayError constructs one offline-replay failure error.
func BuildOfflineReplayError(parseEntity string, parseReason string) error {
	parseTrimmedEntity := strings.TrimSpace(parseEntity)
	if parseTrimmedEntity == "" {
		parseTrimmedEntity = "replay"
	}
	parseTrimmedReason := strings.TrimSpace(parseReason)
	parseMessage := "entity=" + parseTrimmedEntity
	if parseTrimmedReason != "" {
		parseMessage += " reason=" + parseTrimmedReason
	}
	return BuildFailureError(FailureCodeOfflineReplay, parseMessage)
}

func (parseC *ResourceController[T]) finish(parseOutcome resourceOutcome[T], parseMarkCancelled ...bool) {
	parseC.mu.Lock()
	parsePending := parseC.pending
	if parsePending != nil {
		parseC.pending = nil
		if len(parseMarkCancelled) > 0 && parseMarkCancelled[0] {
			parseC.attempts[parsePending.index-1].Cancelled = true
		}
	}
	parseC.mu.Unlock()
	if parsePending != nil {
		parsePending.done <- parseOutcome
	}
}
