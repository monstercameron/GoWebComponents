package plugin

import (
	"strings"
	"testing"
	"time"
)

// TestRegisterFromInsideSetupFailsInsteadOfDeadlocking pins that a plugin calling Register or Close
// from its own Setup gets a deterministic error: registerMu is held across Setup, so the nested
// call used to wait on itself forever.
func TestRegisterFromInsideSetupFailsInsteadOfDeadlocking(parseT *testing.T) {
	parseHost := NewHost(HostOptions{})
	var parseNestedErr, parseCloseErr error
	parseDone := make(chan error, 1)
	go func() {
		parseDone <- parseHost.Register(Define(reviewManifest("outer"), func(parseInner *Host) (CleanupFunc, error) {
			parseNestedErr = parseInner.Register(Define(reviewManifest("inner"), func(*Host) (CleanupFunc, error) { return nil, nil }))
			parseCloseErr = parseInner.Close()
			return nil, nil
		}))
	}()
	select {
	case parseErr := <-parseDone:
		if parseErr != nil {
			parseT.Fatalf("outer Register failed: %v", parseErr)
		}
	case <-time.After(5 * time.Second):
		parseT.Fatal("Register from inside Setup deadlocked")
	}
	if parseNestedErr == nil || !strings.Contains(parseNestedErr.Error(), "inside a plugin Setup") {
		parseT.Errorf("nested Register error = %v", parseNestedErr)
	}
	if parseCloseErr == nil {
		parseT.Error("nested Close should fail")
	}
	if parsePlugins := parseHost.Plugins(); len(parsePlugins) != 1 || parsePlugins[0].ID != "outer" {
		parseT.Errorf("plugins = %v, want only outer", parsePlugins)
	}
	// The scope is cleared afterwards: a later Register from this goroutine works.
	if parseErr := parseHost.Register(Define(reviewManifest("later"), func(*Host) (CleanupFunc, error) { return nil, nil })); parseErr != nil {
		parseT.Errorf("Register after Setup: %v", parseErr)
	}
}
