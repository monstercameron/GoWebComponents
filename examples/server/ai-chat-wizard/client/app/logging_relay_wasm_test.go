//go:build js && wasm

package app

import (
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestParseShouldRetryClientLogRelay(parseT *testing.T) {
	parseTests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "unavailable retries",
			err:  status.Error(codes.Unavailable, "bridge unavailable"),
			want: true,
		},
		{
			name: "deadline exceeded retries",
			err:  status.Error(codes.DeadlineExceeded, "timeout"),
			want: true,
		},
		{
			name: "permission denied does not retry",
			err:  status.Error(codes.PermissionDenied, "forbidden"),
			want: false,
		},
	}

	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			if parseGot := parseShouldRetryClientLogRelay(parseTest.err); parseGot != parseTest.want {
				parseT2.Fatalf("parseShouldRetryClientLogRelay(%v) = %v, want %v", parseTest.err, parseGot, parseTest.want)
			}
		})
	}
}

func TestParseClientLogRelayBackoff(parseT *testing.T) {
	parseTests := []struct {
		attempt int
		want    time.Duration
	}{
		{attempt: 0, want: 250 * time.Millisecond},
		{attempt: 1, want: 250 * time.Millisecond},
		{attempt: 2, want: 500 * time.Millisecond},
		{attempt: 3, want: 1 * time.Second},
	}

	for _, parseTest := range parseTests {
		if parseGot := parseClientLogRelayBackoff(parseTest.attempt); parseGot != parseTest.want {
			parseT.Fatalf("parseClientLogRelayBackoff(%d) = %s, want %s", parseTest.attempt, parseGot, parseTest.want)
		}
	}
}
