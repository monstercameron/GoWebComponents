package plugin

import "testing"

func BenchmarkEvaluateRouteMicro(parseB *testing.B) {
	parseHost := NewHost(HostOptions{Capabilities: []Capability{CapabilityRouter}})
	parseErr := parseHost.AddRouteGuard(func(parseRequest2 RouteRequest) GuardDecision {
		if parseRequest2.Path == "/restricted" {
			return Block("restricted route")
		}
		return Allow("ok")
	})
	if parseErr != nil {
		parseB.Fatal(parseErr)
	}

	parseRequest := RouteRequest{Path: "/dashboard"}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseDecision := parseHost.EvaluateRoute(parseRequest)
		if parseDecision.Outcome != GuardAllow {
			parseB.Fatalf("unexpected outcome: %s", parseDecision.Outcome)
		}
	}
}
