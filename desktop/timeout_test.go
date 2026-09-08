package desktop

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/interop"
)

// TestCallTimeoutBounds rejects invalid overrides before touching the transport.
func TestCallTimeoutBounds(parseTest *testing.T) {
	parseTransport := getTestTransport()
	parseTransport.parseStart = func(string, json.RawMessage) (string, error) {
		parseTest.Fatal("invalid timeout started native work")
		return "", nil
	}
	for _, parseTimeout := range []time.Duration{0, -time.Nanosecond, MaximumRequestTimeout + time.Nanosecond} {
		if _, parseErr := CallWithTimeout[any](context.Background(), NewClient(parseTransport), parseTimeout, "value"); !interop.IsCode(parseErr, interop.CodeInvalid) {
			parseTest.Fatalf("timeout %v: %v", parseTimeout, parseErr)
		}
	}
}

// TestCallTimeoutContextPolicy checks the unchanged default and opt-in deadline without long sleeps.
func TestCallTimeoutContextPolicy(parseTest *testing.T) {
	if RequestTimeout != 30*time.Second || MaximumRequestTimeout != 10*time.Minute {
		parseTest.Fatal("call timeout policy changed")
	}
	for _, parseTimeout := range []time.Duration{RequestTimeout, 5 * time.Minute, MaximumRequestTimeout} {
		parseBefore := time.Now()
		parseContext, parseCancel := getCallContext(nil, parseTimeout) //nolint:staticcheck // Exercise the documented nil-context fallback.
		parseDeadline, hasDeadline := parseContext.Deadline()
		parseCancel()
		if !hasDeadline || parseDeadline.Before(parseBefore.Add(parseTimeout)) || parseDeadline.After(time.Now().Add(parseTimeout)) {
			parseTest.Fatalf("unexpected deadline for %v: %v", parseTimeout, parseDeadline)
		}
	}
	parseParent, parseStop := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
	defer parseStop()
	parseContext, parseCancel := getCallContext(parseParent, 5*time.Minute)
	defer parseCancel()
	parseParentDeadline, _ := parseParent.Deadline()
	parseDeadline, _ := parseContext.Deadline()
	if !parseDeadline.Equal(parseParentDeadline) {
		parseTest.Fatal("interactive override extended caller deadline")
	}
}

// TestCallTimeoutAndEarlierDeadline checks both call paths against a never-settling transport.
func TestCallTimeoutAndEarlierDeadline(parseTest *testing.T) {
	for _, parseDefault := range []bool{false, true} {
		parseTransport := getTestTransport()
		parseTransport.parsePoll = func(string) (Reply, error) { return Reply{}, nil }
		parseContext, parseCancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		var parseErr error
		if parseDefault {
			_, parseErr = Call[any](parseContext, NewClient(parseTransport), "value")
		} else {
			_, parseErr = CallWithTimeout[any](parseContext, NewClient(parseTransport), 5*time.Minute, "value")
		}
		parseCancel()
		if !interop.IsCode(parseErr, interop.CodeTimeout) || parseTransport.parseCancelCount != 1 {
			parseTest.Fatalf("default=%t: error=%v cancels=%d", parseDefault, parseErr, parseTransport.parseCancelCount)
		}
	}
	parseTransport := getTestTransport()
	parseTransport.parsePoll = func(string) (Reply, error) { return Reply{}, nil }
	if _, parseErr := CallWithTimeout[any](nil, NewClient(parseTransport), 20*time.Millisecond, "value"); !interop.IsCode(parseErr, interop.CodeTimeout) || parseTransport.parseCancelCount != 1 { //nolint:staticcheck // Exercise the backwards-compatible nil-context fallback.
		parseTest.Fatalf("explicit short timeout: %v cancels=%d", parseErr, parseTransport.parseCancelCount)
	}
}
