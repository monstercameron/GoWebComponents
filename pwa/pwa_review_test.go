package pwa

import (
	"errors"
	"fmt"
	"math"
	"testing"
)

func reviewManifestFor(parseBuild string) WasmReleaseManifest {
	return WasmReleaseManifest{
		Package: "example/app", GOOS: "js", GOARCH: "wasm", BuildID: parseBuild,
		Artifacts: map[string]WasmReleaseArtifact{"wasm": {Path: "app-" + parseBuild + ".wasm", SHA256: "abc" + parseBuild}},
	}
}

// TestWasmRolloutCanaryShareMatchesPercent pins that CanaryPercent is the share of clients routed
// to the canary. The bucket was `int(sum[0]) % 100` over a BYTE (0..255), so buckets 0..55 were
// hit three times as often as 56..99 and "10%" canaried ~11.7% of clients ("50%" ~58.6%).
func TestWasmRolloutCanaryShareMatchesPercent(parseT *testing.T) {
	const parseClients = 40000
	for _, parsePercent := range []int{1, 10, 25, 50, 75, 99} {
		parseConfig := WasmRolloutConfig{
			Stable: reviewManifestFor("stable"), Canary: reviewManifestFor("canary"),
			CanaryPercent: parsePercent, Salt: "release-1",
		}
		parseCanary := 0
		for parseIndex := range parseClients {
			parseDecision, parseErr := ChooseWasmRollout(parseConfig, fmt.Sprintf("client-%d", parseIndex))
			if parseErr != nil {
				parseT.Fatal(parseErr)
			}
			if parseDecision.Cohort == "canary" {
				parseCanary++
			}
		}
		parseShare := 100 * float64(parseCanary) / parseClients
		if math.Abs(parseShare-float64(parsePercent)) > 1.0 {
			parseT.Errorf("CanaryPercent=%d routed %.2f%% of clients to the canary", parsePercent, parseShare)
		}
	}
}

// TestWasmRolloutIsMonotonicInPercent pins that raising the percentage only ever adds clients: a
// client already on the canary at 10% must stay on it at 20%.
func TestWasmRolloutIsMonotonicInPercent(parseT *testing.T) {
	parseConfig := WasmRolloutConfig{Stable: reviewManifestFor("stable"), Canary: reviewManifestFor("canary"), Salt: "s"}
	for parseIndex := range 2000 {
		parseClient := fmt.Sprintf("c%d", parseIndex)
		parseWasOn := false
		for _, parsePercent := range []int{0, 5, 10, 20, 50, 100} {
			parseConfig.CanaryPercent = parsePercent
			parseDecision, parseErr := ChooseWasmRollout(parseConfig, parseClient)
			if parseErr != nil {
				parseT.Fatal(parseErr)
			}
			isOn := parseDecision.Cohort == "canary"
			if parseWasOn && !isOn {
				parseT.Fatalf("client %s left the canary when the percentage rose to %d", parseClient, parsePercent)
			}
			parseWasOn = isOn
		}
		if !parseWasOn {
			parseT.Fatalf("client %s not on the canary at 100%%", parseClient)
		}
	}
}

// TestUpdatePromptStartRetriesAfterSubscribeFailure pins that a failed lifecycle subscription does
// not leave the prompt permanently "started" (Start would then return nil forever without ever
// listening, so an update banner could never appear).
func TestUpdatePromptStartRetriesAfterSubscribeFailure(parseT *testing.T) {
	parseAttempts := 0
	var parseDeliver func(ServiceWorkerSnapshot)
	parseRegistration := ServiceWorkerRegistration{
		snapshot: func() ServiceWorkerSnapshot { return ServiceWorkerSnapshot{} },
		subscribeLifecycle: func(parseHandler func(ServiceWorkerSnapshot)) (ServiceWorkerSubscription, error) {
			parseAttempts++
			if parseAttempts == 1 {
				return ServiceWorkerSubscription{}, errors.New("transient")
			}
			parseDeliver = parseHandler
			return ServiceWorkerSubscription{}, nil
		},
	}
	parsePrompt := NewUpdatePrompt(parseRegistration)
	if parseErr := parsePrompt.Start(); parseErr == nil {
		parseT.Fatal("first Start should report the subscription failure")
	}
	if parseErr := parsePrompt.Start(); parseErr != nil {
		parseT.Fatalf("second Start: %v", parseErr)
	}
	if parseAttempts != 2 || parseDeliver == nil {
		parseT.Fatalf("second Start did not re-subscribe (attempts=%d)", parseAttempts)
	}
	parseDeliver(ServiceWorkerSnapshot{Waiting: ServiceWorkerVersion{ScriptURL: "/sw.js"}})
	if !parsePrompt.Available() {
		parseT.Fatal("update not surfaced after a successful retry")
	}
}
