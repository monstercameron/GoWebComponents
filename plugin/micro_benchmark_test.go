package plugin

import "testing"

func BenchmarkEvaluateRouteMicro(b *testing.B) {
	host := NewHost(HostOptions{Capabilities: []Capability{CapabilityRouter}})
	err := host.AddRouteGuard(func(request RouteRequest) GuardDecision {
		if request.Path == "/restricted" {
			return Block("restricted route")
		}
		return Allow("ok")
	})
	if err != nil {
		b.Fatal(err)
	}

	request := RouteRequest{Path: "/dashboard"}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		decision := host.EvaluateRoute(request)
		if decision.Outcome != GuardAllow {
			b.Fatalf("unexpected outcome: %s", decision.Outcome)
		}
	}
}
