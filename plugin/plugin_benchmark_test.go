package plugin

import "testing"

func BenchmarkNewHostWithCapabilities(b *testing.B) {
	b.ReportAllocs()
	options := HostOptions{
		Capabilities: []Capability{
			CapabilityRouter,
			CapabilityAsyncData,
			CapabilityDevtools,
		},
	}
	for b.Loop() {
		host := NewHost(options)
		if host == nil {
			b.Fatal("NewHost returned nil")
		}
	}
}

func BenchmarkEvaluateRouteWithSingleAllowGuard(b *testing.B) {
	b.ReportAllocs()
	host := NewHost(HostOptions{Capabilities: []Capability{CapabilityRouter}})
	if err := host.AddRouteGuard(func(request RouteRequest) GuardDecision {
		if request.Path == "" {
			return Block("missing path")
		}
		return Allow("ok")
	}); err != nil {
		b.Fatalf("AddRouteGuard: %v", err)
	}
	request := RouteRequest{Path: "/app/thread/bench"}
	for b.Loop() {
		decision := host.EvaluateRoute(request)
		if decision.Outcome == GuardBlock {
			b.Fatal("unexpected blocked route")
		}
	}
}
