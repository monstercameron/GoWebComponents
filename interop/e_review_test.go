package interop

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestBuildCookieStringStripsAttributeInjection verifies Path, Domain and
// SameSite values cannot smuggle extra directives into the cookie string.
func TestBuildCookieStringStripsAttributeInjection(parseT *testing.T) {
	parseGot := buildCookieString("k", "v", CookieOptions{
		Path:     "/a; Domain=evil.example",
		Domain:   "ok.example\r\nSet-Cookie: x=1; Secure",
		SameSite: SameSite("Lax; Domain=evil.example"),
	})
	for _, parsePart := range strings.Split(parseGot, "; ") {
		if strings.HasPrefix(parsePart, "Domain=evil") || strings.Contains(parsePart, "\n") || strings.HasPrefix(parsePart, "Set-Cookie") {
			parseT.Fatalf("attribute injection survived: %q", parseGot)
		}
	}
	if strings.Count(parseGot, "; Domain=") != 1 {
		parseT.Fatalf("expected exactly one Domain directive: %q", parseGot)
	}
	if strings.Contains(parseGot, "SameSite=") {
		parseT.Fatalf("bad SameSite survived: %q", parseGot)
	}
}

// TestSetCookieRejectsNameThatSanitizesToEmpty verifies a name made only of
// illegal bytes is rejected instead of writing "=value".
func TestSetCookieRejectsNameThatSanitizesToEmpty(parseT *testing.T) {
	if parseErr := SetCookie(";=", "v"); !IsCode(parseErr, CodeInvalid) {
		parseT.Fatalf("expected CodeInvalid, got %v", parseErr)
	}
}

// TestWorkerPoolDrainDoesNotHangWhenWorkerDisposedMidDrain verifies a queued
// request does not wait forever on a slot lost to a disposed worker once the
// pool is draining.
func TestWorkerPoolDrainDoesNotHangWhenWorkerDisposedMidDrain(parseT *testing.T) {
	parseStarted := make(chan struct{})
	parseRelease := make(chan struct{})
	parsePool, parseErr := OpenWorkerPool(context.Background(), WorkerPoolOptions{
		Size:       1,
		QueueLimit: 1,
		OpenWorker: func(context.Context) (Worker, error) {
			return Worker{
				request: func(context.Context, string, any, func(WorkerMessage, error)) (WorkerMessage, error) {
					select {
					case parseStarted <- struct{}{}:
					default:
					}
					<-parseRelease
					return WorkerMessage{}, wrapError("Worker.Request", "t", CodeDisposed, context.Canceled)
				},
				terminate: func() error { return nil },
			}, nil
		},
	})
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseDone := make(chan struct{}, 2)
	for parseI := 0; parseI < 2; parseI++ {
		go func() {
			_, _ = parsePool.Request(context.Background(), "t", nil, nil)
			parseDone <- struct{}{}
		}()
	}
	<-parseStarted
	time.Sleep(50 * time.Millisecond)
	parseDrainErr := make(chan error, 1)
	go func() { parseDrainErr <- parsePool.Drain(context.Background()) }()
	time.Sleep(50 * time.Millisecond)
	close(parseRelease)
	select {
	case <-parseDrainErr:
	case <-time.After(3 * time.Second):
		parseT.Fatal("Drain hung: queued request never completed after worker disposal")
	}
}
