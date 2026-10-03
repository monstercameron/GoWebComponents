package plugin

import (
	"errors"
	"strings"
	"testing"
)

func reviewManifest(parseID string, parseRequires ...Capability) Manifest {
	return Manifest{ID: parseID, Version: "1.0.0", Tier: TierStable, Requires: parseRequires}
}

// TestRegisterRejectsDuplicateIDWithWhitespace pins that IDs are compared after trimming on both
// sides. validateManifest trims only a local copy, so " dup " was stored untrimmed and a second
// " dup " (or "dup") slipped past the duplicate check, installing the plugin twice.
func TestRegisterRejectsDuplicateIDWithWhitespace(parseT *testing.T) {
	parseHost := NewHost(HostOptions{})
	if parseErr := parseHost.Register(Define(reviewManifest(" dup "), nil)); parseErr != nil {
		parseT.Fatalf("first register: %v", parseErr)
	}
	for _, parseID := range []string{" dup ", "dup", "dup\t"} {
		if parseErr := parseHost.Register(Define(reviewManifest(parseID), nil)); parseErr == nil {
			parseT.Errorf("duplicate ID %q was accepted", parseID)
		}
	}
	if parseCount := len(parseHost.Plugins()); parseCount != 1 {
		parseT.Fatalf("plugins = %d, want 1", parseCount)
	}
}

// TestRegisterIsolatesSetupPanic pins that a plugin whose Setup panics after registering
// callbacks is rolled back and reported as an error, not allowed to crash the host or leave the
// half-installed guard behind.
func TestRegisterIsolatesSetupPanic(parseT *testing.T) {
	parseOldHandler := PluginPanicHandler
	parseReported := 0
	PluginPanicHandler = func(string, any) { parseReported++ }
	defer func() { PluginPanicHandler = parseOldHandler }()

	parseHost := NewHost(HostOptions{Capabilities: []Capability{CapabilityRouter}})
	parsePlugin := Define(reviewManifest("boom", CapabilityRouter), func(parseH *Host) (CleanupFunc, error) {
		if parseErr := parseH.AddRouteGuard(func(RouteRequest) GuardDecision { return Block("installed by a panicking setup") }); parseErr != nil {
			return nil, parseErr
		}
		parseH.SetValue("leak", true)
		panic("setup exploded")
	})

	var parseErr error
	func() {
		defer func() {
			if parseRecovered := recover(); parseRecovered != nil {
				parseT.Fatalf("Register propagated the plugin panic: %v", parseRecovered)
			}
		}()
		parseErr = parseHost.Register(parsePlugin)
	}()
	if parseErr == nil || !strings.Contains(parseErr.Error(), "setup exploded") {
		parseT.Fatalf("Register error = %v, want it to carry the panic value", parseErr)
	}
	if parseReported != 1 {
		parseT.Errorf("PluginPanicHandler calls = %d, want 1", parseReported)
	}
	if parseDecision := parseHost.EvaluateRoute(RouteRequest{Path: "/"}); parseDecision.Outcome != GuardAllow {
		parseT.Errorf("guard from the panicked plugin survived: %#v", parseDecision)
	}
	if _, parseOk := parseHost.Value("leak"); parseOk {
		parseT.Errorf("host value from the panicked plugin survived")
	}
	if len(parseHost.Plugins()) != 0 {
		parseT.Errorf("panicked plugin is listed as registered")
	}
	// The host must stay usable, and the ID must be free for a fixed version.
	if parseErr := parseHost.Register(Define(reviewManifest("boom"), nil)); parseErr != nil {
		parseT.Errorf("re-register after rollback: %v", parseErr)
	}
}

// TestCloseRunsEveryCleanupDespitePanicAndResetsState pins error isolation on shutdown: one
// panicking cleanup must not skip the remaining cleanups or leave the host unreset.
func TestCloseRunsEveryCleanupDespitePanicAndResetsState(parseT *testing.T) {
	parseOldHandler := PluginPanicHandler
	PluginPanicHandler = func(string, any) {}
	defer func() { PluginPanicHandler = parseOldHandler }()

	parseHost := NewHost(HostOptions{})
	var parseOrder []string
	for _, parseID := range []string{"a", "b", "c"} {
		parseCaptured := parseID
		if parseErr := parseHost.Register(Define(reviewManifest(parseCaptured), func(*Host) (CleanupFunc, error) {
			return func() error {
				parseOrder = append(parseOrder, parseCaptured)
				switch parseCaptured {
				case "b":
					panic("cleanup exploded")
				case "c":
					return errors.New("c failed")
				}
				return nil
			}, nil
		})); parseErr != nil {
			parseT.Fatal(parseErr)
		}
	}

	var parseCloseErr error
	func() {
		defer func() {
			if parseRecovered := recover(); parseRecovered != nil {
				parseT.Fatalf("Close propagated a cleanup panic: %v", parseRecovered)
			}
		}()
		parseCloseErr = parseHost.Close()
	}()
	if strings.Join(parseOrder, ",") != "c,b,a" {
		parseT.Fatalf("cleanup order = %v, want reverse registration c,b,a", parseOrder)
	}
	if parseCloseErr == nil || !strings.Contains(parseCloseErr.Error(), "c failed") || !strings.Contains(parseCloseErr.Error(), "cleanup exploded") {
		parseT.Fatalf("Close error = %v, want both failures joined", parseCloseErr)
	}
	if len(parseHost.Plugins()) != 0 {
		parseT.Fatalf("plugins not cleared after Close")
	}
}
