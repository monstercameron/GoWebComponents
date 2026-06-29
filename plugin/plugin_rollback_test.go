package plugin

import (
	"errors"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/ui"
)

func TestRequireCapabilityBranches(parseT *testing.T) {
	var parseNilHost *Host
	if parseErr := parseNilHost.AddRouteGuard(func(RouteRequest) GuardDecision { return Allow("ok") }); parseErr == nil || !strings.Contains(parseErr.Error(), "host is nil") {
		parseT.Fatalf("expected nil host capability error, got %v", parseErr)
	}

	parseHost := NewHost(HostOptions{})
	if parseErr2 := parseHost.AddRouteGuard(func(RouteRequest) GuardDecision { return Allow("ok") }); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "capability \"router\" is not enabled") {
		parseT.Fatalf("expected missing router capability error, got %v", parseErr2)
	}
	if parseErr3 := parseHost.AddHeadProvider(func() ui.Node { return nil }); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "capability \"ssr\" is not enabled") {
		parseT.Fatalf("expected missing ssr capability error, got %v", parseErr3)
	}
}

func TestRegisterRollbackTrimsAddedValuesAndContributions(parseT *testing.T) {
	parseHost := NewHost(HostOptions{Capabilities: []Capability{CapabilityRouter}})
	parseHost.SetValue("baseline", "yes")
	if parseErr := parseHost.AddRouteGuard(func(parseRequest RouteRequest) GuardDecision {
		if parseRequest.Path == "/blocked" {
			return Block("baseline block")
		}
		return Allow("baseline allow")
	}); parseErr != nil {
		parseT.Fatalf("unexpected baseline route guard error: %v", parseErr)
	}

	parseErr2 := parseHost.Register(Define(Manifest{
		ID:          "rollback-demo",
		Version:     "0.1.0",
		Description: "exercise rollback",
		Tier:        TierExperimental,
		Requires:    []Capability{CapabilityRouter},
	}, func(parseHost2 *Host) (CleanupFunc, error) {
		_ = parseHost2.AddRouteGuard(func(RouteRequest) GuardDecision { return Block("failed guard") })
		parseHost2.SetValue("failed-value-1", "x")
		parseHost2.SetValue("failed-value-2", "y")
		return nil, errors.New("force setup failure")
	}))
	if parseErr2 == nil || !strings.Contains(parseErr2.Error(), "setup failed for \"rollback-demo\"") {
		parseT.Fatalf("expected setup failure error, got %v", parseErr2)
	}

	// Route guards should roll back to baseline contributions only.
	if parseDecision := parseHost.EvaluateRoute(RouteRequest{Path: "/blocked"}); parseDecision.Outcome != GuardBlock || parseDecision.Reason != "baseline block" {
		parseT.Fatalf("expected baseline guard to remain after rollback, got %+v", parseDecision)
	}
	if parseDecision2 := parseHost.EvaluateRoute(RouteRequest{Path: "/allowed"}); parseDecision2.Outcome != GuardAllow {
		parseT.Fatalf("expected baseline allow decision after rollback, got %+v", parseDecision2)
	}

	// Failed plugin values should be trimmed back to snapshot size.
	if len(parseHost.values) != 1 {
		parseT.Fatalf("expected rollback to trim value map to snapshot size, got len=%d values=%+v", len(parseHost.values), parseHost.values)
	}
	if _, parseOk := parseHost.Value("failed-value-1"); parseOk {
		parseT.Fatalf("expected failed plugin value to be rolled back, got %+v", parseHost.values)
	}
	if _, parseOk2 := parseHost.Value("failed-value-2"); parseOk2 {
		parseT.Fatalf("expected failed plugin value to be rolled back, got %+v", parseHost.values)
	}
	if parseValue, parseOk3 := parseHost.Value("baseline"); !parseOk3 || parseValue != "yes" {
		parseT.Fatalf("expected baseline value to remain after rollback, got value=%v ok=%t map=%+v", parseValue, parseOk3, parseHost.values)
	}

	if len(parseHost.Plugins()) != 0 {
		parseT.Fatalf("expected failed plugin not to be registered, got %+v", parseHost.Plugins())
	}
}

func TestRegisterRollbackTrimsAddedDevtoolsContributions(parseT *testing.T) {
	parseHost := NewHost(HostOptions{Capabilities: []Capability{CapabilityDevtools}})

	parseErr := parseHost.Register(Define(Manifest{
		ID:          "rollback-devtools",
		Version:     "0.1.0",
		Description: "exercise devtools rollback",
		Tier:        TierExperimental,
		Requires:    []Capability{CapabilityDevtools},
	}, func(parseHost2 *Host) (CleanupFunc, error) {
		_ = parseHost2.AddDevtoolsSectionProvider(func() DevtoolsSection {
			return DevtoolsSection{Name: "Companion", Summary: map[string]string{"state": "ready"}}
		})
		_ = parseHost2.AddDevtoolsActionProvider(func() []DevtoolsAction {
			return []DevtoolsAction{{
				Label:        "Retry",
				MatchCodes:   []string{"GWC-ROUTER-FAILED"},
				MatchSources: []string{"router"},
				Run:          func(DevtoolsActionContext) {},
			}}
		})
		return nil, errors.New("force devtools setup failure")
	}))
	if parseErr == nil || !strings.Contains(parseErr.Error(), "setup failed for \"rollback-devtools\"") {
		parseT.Fatalf("expected setup failure error, got %v", parseErr)
	}

	if parseSections := parseHost.DevtoolsSections(); len(parseSections) != 0 {
		parseT.Fatalf("expected devtools sections to roll back, got %+v", parseSections)
	}
	if parseActions := parseHost.DevtoolsActions(); len(parseActions) != 0 {
		parseT.Fatalf("expected devtools actions to roll back, got %+v", parseActions)
	}
	if len(parseHost.Plugins()) != 0 {
		parseT.Fatalf("expected failed plugin not to be registered, got %+v", parseHost.Plugins())
	}
}
