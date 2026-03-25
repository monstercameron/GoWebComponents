package plugin

import (
	"errors"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestRequireCapabilityBranches(t *testing.T) {
	var nilHost *Host
	if err := nilHost.AddRouteGuard(func(RouteRequest) GuardDecision { return Allow("ok") }); err == nil || !strings.Contains(err.Error(), "host is nil") {
		t.Fatalf("expected nil host capability error, got %v", err)
	}

	host := NewHost(HostOptions{})
	if err := host.AddRouteGuard(func(RouteRequest) GuardDecision { return Allow("ok") }); err == nil || !strings.Contains(err.Error(), "capability \"router\" is not enabled") {
		t.Fatalf("expected missing router capability error, got %v", err)
	}
	if err := host.AddHeadProvider(func() ui.Node { return nil }); err == nil || !strings.Contains(err.Error(), "capability \"ssr\" is not enabled") {
		t.Fatalf("expected missing ssr capability error, got %v", err)
	}
}

func TestRegisterRollbackTrimsAddedValuesAndContributions(t *testing.T) {
	host := NewHost(HostOptions{Capabilities: []Capability{CapabilityRouter}})
	host.SetValue("baseline", "yes")
	if err := host.AddRouteGuard(func(request RouteRequest) GuardDecision {
		if request.Path == "/blocked" {
			return Block("baseline block")
		}
		return Allow("baseline allow")
	}); err != nil {
		t.Fatalf("unexpected baseline route guard error: %v", err)
	}

	err := host.Register(Define(Manifest{
		ID:          "rollback-demo",
		Version:     "0.1.0",
		Description: "exercise rollback",
		Tier:        TierExperimental,
		Requires:    []Capability{CapabilityRouter},
	}, func(host *Host) (CleanupFunc, error) {
		_ = host.AddRouteGuard(func(RouteRequest) GuardDecision { return Block("failed guard") })
		host.SetValue("failed-value-1", "x")
		host.SetValue("failed-value-2", "y")
		return nil, errors.New("force setup failure")
	}))
	if err == nil || !strings.Contains(err.Error(), "setup failed for \"rollback-demo\"") {
		t.Fatalf("expected setup failure error, got %v", err)
	}

	// Route guards should roll back to baseline contributions only.
	if decision := host.EvaluateRoute(RouteRequest{Path: "/blocked"}); decision.Outcome != GuardBlock || decision.Reason != "baseline block" {
		t.Fatalf("expected baseline guard to remain after rollback, got %+v", decision)
	}
	if decision := host.EvaluateRoute(RouteRequest{Path: "/allowed"}); decision.Outcome != GuardAllow {
		t.Fatalf("expected baseline allow decision after rollback, got %+v", decision)
	}

	// Failed plugin values should be trimmed back to snapshot size.
	if len(host.values) != 1 {
		t.Fatalf("expected rollback to trim value map to snapshot size, got len=%d values=%+v", len(host.values), host.values)
	}
	if _, ok := host.Value("failed-value-1"); ok {
		t.Fatalf("expected failed plugin value to be rolled back, got %+v", host.values)
	}
	if _, ok := host.Value("failed-value-2"); ok {
		t.Fatalf("expected failed plugin value to be rolled back, got %+v", host.values)
	}
	if value, ok := host.Value("baseline"); !ok || value != "yes" {
		t.Fatalf("expected baseline value to remain after rollback, got value=%v ok=%t map=%+v", value, ok, host.values)
	}

	if len(host.Plugins()) != 0 {
		t.Fatalf("expected failed plugin not to be registered, got %+v", host.Plugins())
	}
}
