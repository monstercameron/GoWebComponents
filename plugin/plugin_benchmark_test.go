package plugin

import "testing"

func BenchmarkNewHostWithCapabilities(parseB *testing.B) {
	parseB.ReportAllocs()
	parseOptions := HostOptions{
		Capabilities: []Capability{
			CapabilityRouter,
			CapabilityAsyncData,
			CapabilityDevtools,
		},
	}
	for parseB.Loop() {
		parseHost := NewHost(parseOptions)
		if parseHost == nil {
			parseB.Fatal("NewHost returned nil")
		}
	}
}

func BenchmarkEvaluateRouteWithSingleAllowGuard(parseB *testing.B) {
	parseB.ReportAllocs()
	parseHost := NewHost(HostOptions{Capabilities: []Capability{CapabilityRouter}})
	if parseErr := parseHost.AddRouteGuard(func(parseRequest2 RouteRequest) GuardDecision {
		if parseRequest2.Path == "" {
			return Block("missing path")
		}
		return Allow("ok")
	}); parseErr != nil {
		parseB.Fatalf("AddRouteGuard: %v", parseErr)
	}
	parseRequest := RouteRequest{Path: "/app/thread/bench"}
	for parseB.Loop() {
		parseDecision := parseHost.EvaluateRoute(parseRequest)
		if parseDecision.Outcome == GuardBlock {
			parseB.Fatal("unexpected blocked route")
		}
	}
}
