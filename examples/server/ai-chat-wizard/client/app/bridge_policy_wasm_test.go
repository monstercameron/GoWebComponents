//go:build js && wasm

package app

import (
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestParseBridgeStateFromTransition(parseT *testing.T) {
	parseTests := []struct {
		name   string
		ready  bool
		reason string
		want   bridgeState
	}{
		{name: "ready", ready: true, reason: "bridge ready", want: bridgeStateReady},
		{name: "reconnecting", ready: true, reason: "bridge reconnecting", want: bridgeStateReconnecting},
		{name: "sleeping", ready: false, reason: "bridge sleeping", want: bridgeStateSleeping},
		{name: "offline", ready: false, reason: "browser offline", want: bridgeStateOffline},
		{name: "degraded", ready: false, reason: "bridge shutdown", want: bridgeStateDegraded},
	}
	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			if parseGot := parseBridgeStateFromTransition(parseTest.ready, parseTest.reason); parseGot != parseTest.want {
				parseT2.Fatalf("state = %q, want %q", parseGot, parseTest.want)
			}
		})
	}
}

func TestParseBridgeRPCPolicyTableCoversTargetFamilies(parseT *testing.T) {
	parseFamilies := []bridgeRPCFamily{
		bridgeRPCChatStream,
		bridgeRPCConversationList,
		bridgeRPCSettingsWrite,
		bridgeRPCAuthSessionRefresh,
		bridgeRPCAdminDashboardLoad,
		bridgeRPCTelemetryRelay,
		bridgeRPCDiagnosticsPolling,
		bridgeRPCBackgroundMaint,
	}
	for _, parseFamily := range parseFamilies {
		parsePolicy := parseBridgeRPCPolicyFor(parseFamily)
		if parsePolicy.Family != parseFamily {
			parseT.Fatalf("policy family = %q, want %q", parsePolicy.Family, parseFamily)
		}
		if parsePolicy.TrafficClass == "" || parsePolicy.UserCriticality == "" {
			parseT.Fatalf("policy %q missing class/criticality: %+v", parseFamily, parsePolicy)
		}
	}
	if parseBridgeRPCPolicyFor(bridgeRPCSettingsWrite).RetrySafe {
		parseT.Fatal("settings writes must remain single-shot until idempotency keys exist")
	}
	if !parseBridgeRPCPolicyFor(bridgeRPCTelemetryRelay).QueueBestEffort {
		parseT.Fatal("telemetry relay should be queued as bounded best-effort work")
	}
}

func TestParseBridgeBestEffortDecision(parseT *testing.T) {
	parsePolicy := parseBridgeRPCPolicyFor(bridgeRPCTelemetryRelay)
	if parseBridgeBestEffortDecision(bridgeStateReady, parsePolicy) != bridgeBestEffortRun {
		parseT.Fatal("expected telemetry to run while ready")
	}
	if parseBridgeBestEffortDecision(bridgeStateReconnecting, parsePolicy) != bridgeBestEffortDefer {
		parseT.Fatal("expected telemetry to defer while reconnecting")
	}
}

func TestParseBridgeRetryPolicy(parseT *testing.T) {
	parsePolicy := parseBridgeRetryPolicyFor(bridgeRPCTelemetryRelay)
	if parsePolicy.MaxAttempts != 3 || parsePolicy.PerAttemptTimeout != 3*time.Second {
		parseT.Fatalf("unexpected telemetry retry policy: %+v", parsePolicy)
	}
	if !parseBridgeRetryableStatus(status.Error(codes.Unavailable, "down")) {
		parseT.Fatal("unavailable should be retryable")
	}
	if parseBridgeRetryableStatus(status.Error(codes.PermissionDenied, "denied")) {
		parseT.Fatal("permission denied should not be retryable")
	}
}

func TestBridgeDeferredLogQueueBoundsAndDrops(parseT *testing.T) {
	parseMetricsBefore := parseSnapshotBridgeChurnMetrics()
	parseQueue := parseNewBridgeDeferredLogQueue(2, 10*time.Second)
	parseNow := time.Unix(100, 0)
	parseQueue.enqueue(bridgeDeferredLogWork{Message: "one"}, parseNow)
	parseQueue.enqueue(bridgeDeferredLogWork{Message: "two"}, parseNow)
	parseQueue.enqueue(bridgeDeferredLogWork{Message: "three"}, parseNow)
	if parseGot := parseQueue.len(parseNow); parseGot != 2 {
		parseT.Fatalf("queue len = %d, want 2", parseGot)
	}
	parseDrained := parseQueue.drain(parseNow, 10)
	if len(parseDrained) != 2 || parseDrained[0].Message != "two" || parseDrained[1].Message != "three" {
		parseT.Fatalf("unexpected drained queue: %+v", parseDrained)
	}
	parseMetricsAfter := parseSnapshotBridgeChurnMetrics()
	if parseMetricsAfter[string(bridgeRPCTelemetryRelay)+":dropped"] <= parseMetricsBefore[string(bridgeRPCTelemetryRelay)+":dropped"] {
		parseT.Fatal("expected bounded queue overflow to increment dropped telemetry metric")
	}
}
