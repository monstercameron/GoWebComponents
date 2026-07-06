package plugin

import (
	"fmt"
	"sync"
	"testing"
)

// TestHostConcurrentRegisterAndDispatchNoDeadlock exercises Register (which holds
// registerMu across the Setup callback while Setup's Add* calls take stateMu) racing
// with concurrent dispatch and reads. It confirms the two-lock design does not
// deadlock and survives concurrent access. (The Go race detector is unavailable on
// this platform, so correctness w.r.t. data races is by construction — all mutable
// Host state is guarded by stateMu and no lock is held across a plugin callback.)
func TestHostConcurrentRegisterAndDispatchNoDeadlock(parseT *testing.T) {
	parseHost := NewHost(HostOptions{Capabilities: []Capability{CapabilityRouter, CapabilityAsyncData}})
	var parseWg sync.WaitGroup

	parseWg.Add(1)
	go func() {
		defer parseWg.Done()
		for parseI := 0; parseI < 50; parseI++ {
			_ = parseHost.Register(Define(Manifest{
				ID:       fmt.Sprintf("p%d", parseI),
				Version:  "1.0.0",
				Tier:     TierExperimental,
				Requires: []Capability{CapabilityRouter, CapabilityAsyncData},
			}, func(parseHost2 *Host) (CleanupFunc, error) {
				_ = parseHost2.AddRouteGuard(func(RouteRequest) GuardDecision { return GuardDecision{} })
				_ = parseHost2.AddCacheKeyDecorator(func(parseK string) string { return parseK })
				return nil, nil
			}))
		}
	}()

	for parseD := 0; parseD < 4; parseD++ {
		parseWg.Add(1)
		go func() {
			defer parseWg.Done()
			for parseI := 0; parseI < 200; parseI++ {
				_ = parseHost.EvaluateRoute(RouteRequest{Path: "/x"})
				_ = parseHost.DecorateCacheKey("k")
				parseHost.NotifyNavigation(NavigationEvent{Path: "/x"})
				_ = parseHost.Plugins()
			}
		}()
	}

	parseWg.Wait()
}
