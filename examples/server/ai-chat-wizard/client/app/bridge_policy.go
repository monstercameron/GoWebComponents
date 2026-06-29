//go:build js && wasm

package app

import (
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type bridgeState string

const (
	bridgeStateBooting      bridgeState = "booting"
	bridgeStateReady        bridgeState = "ready"
	bridgeStateReconnecting bridgeState = "reconnecting"
	bridgeStateDegraded     bridgeState = "degraded"
	bridgeStateSleeping     bridgeState = "sleeping"
	bridgeStateOffline      bridgeState = "offline"
)

type bridgeRPCFamily string

const (
	bridgeRPCChatStream         bridgeRPCFamily = "chat_send_stream"
	bridgeRPCConversationList   bridgeRPCFamily = "conversation_list_refresh"
	bridgeRPCSettingsWrite      bridgeRPCFamily = "settings_write"
	bridgeRPCAuthSessionRefresh bridgeRPCFamily = "auth_session_refresh"
	bridgeRPCAdminDashboardLoad bridgeRPCFamily = "admin_dashboard_load"
	bridgeRPCTelemetryRelay     bridgeRPCFamily = "telemetry_relay"
	bridgeRPCDiagnosticsPolling bridgeRPCFamily = "diagnostics_polling"
	bridgeRPCBackgroundMaint    bridgeRPCFamily = "background_maintenance"
)

type bridgeRPCPolicy struct {
	Family                 bridgeRPCFamily
	TrafficClass           string
	UserCriticality        string
	Idempotent             bool
	RetrySafe              bool
	TimeoutBudget          time.Duration
	AllowWhileReconnecting bool
	QueueBestEffort        bool
}

type bridgeRetryPolicy struct {
	MaxAttempts       int
	PerAttemptTimeout time.Duration
	BackoffBase       time.Duration
	BackoffMax        time.Duration
}

func parseBridgeStateFromTransition(isReady bool, parseReason string) bridgeState {
	parseReason = strings.ToLower(strings.TrimSpace(parseReason))
	if isReady {
		switch {
		case strings.Contains(parseReason, "reconnecting") || strings.Contains(parseReason, "connecting"):
			return bridgeStateReconnecting
		case strings.Contains(parseReason, "unavailable"):
			return bridgeStateDegraded
		default:
			return bridgeStateReady
		}
	}
	switch {
	case strings.Contains(parseReason, "offline"):
		return bridgeStateOffline
	case strings.Contains(parseReason, "sleep") || strings.Contains(parseReason, "hidden"):
		return bridgeStateSleeping
	case strings.Contains(parseReason, "dial failed") || strings.Contains(parseReason, "stalled") || strings.Contains(parseReason, "transient"):
		return bridgeStateReconnecting
	case strings.Contains(parseReason, "runtime stopped") || strings.Contains(parseReason, "canceled"):
		return bridgeStateOffline
	case strings.Contains(parseReason, "unavailable") || strings.Contains(parseReason, "shutdown"):
		return bridgeStateDegraded
	default:
		return bridgeStateReconnecting
	}
}

func parseBridgeReadyFromState(parseState bridgeState) bool {
	return parseState == bridgeStateReady
}

func parseBridgeRPCPolicyFor(parseFamily bridgeRPCFamily) bridgeRPCPolicy {
	switch parseFamily {
	case bridgeRPCChatStream:
		return bridgeRPCPolicy{Family: parseFamily, TrafficClass: "foreground_stream", UserCriticality: "critical", Idempotent: false, RetrySafe: false, TimeoutBudget: 0}
	case bridgeRPCConversationList:
		return bridgeRPCPolicy{Family: parseFamily, TrafficClass: "foreground_read", UserCriticality: "high", Idempotent: true, RetrySafe: true, TimeoutBudget: 6 * time.Second, AllowWhileReconnecting: true}
	case bridgeRPCSettingsWrite:
		return bridgeRPCPolicy{Family: parseFamily, TrafficClass: "foreground_mutation", UserCriticality: "high", Idempotent: false, RetrySafe: false, TimeoutBudget: 8 * time.Second}
	case bridgeRPCAuthSessionRefresh:
		return bridgeRPCPolicy{Family: parseFamily, TrafficClass: "foreground_read", UserCriticality: "critical", Idempotent: true, RetrySafe: true, TimeoutBudget: 6 * time.Second, AllowWhileReconnecting: true}
	case bridgeRPCAdminDashboardLoad:
		return bridgeRPCPolicy{Family: parseFamily, TrafficClass: "foreground_read", UserCriticality: "medium", Idempotent: true, RetrySafe: true, TimeoutBudget: 8 * time.Second, AllowWhileReconnecting: true}
	case bridgeRPCTelemetryRelay:
		return bridgeRPCPolicy{Family: parseFamily, TrafficClass: "best_effort", UserCriticality: "low", Idempotent: true, RetrySafe: true, TimeoutBudget: 3 * time.Second, AllowWhileReconnecting: true, QueueBestEffort: true}
	case bridgeRPCDiagnosticsPolling:
		return bridgeRPCPolicy{Family: parseFamily, TrafficClass: "background_read", UserCriticality: "low", Idempotent: true, RetrySafe: true, TimeoutBudget: 4 * time.Second, QueueBestEffort: true}
	case bridgeRPCBackgroundMaint:
		return bridgeRPCPolicy{Family: parseFamily, TrafficClass: "background_maintenance", UserCriticality: "low", Idempotent: true, RetrySafe: true, TimeoutBudget: 4 * time.Second, QueueBestEffort: true}
	default:
		return bridgeRPCPolicy{Family: parseFamily, TrafficClass: "foreground", UserCriticality: "medium", TimeoutBudget: 5 * time.Second}
	}
}

func parseBridgeRetryPolicyFor(parseFamily bridgeRPCFamily) bridgeRetryPolicy {
	parsePolicy := parseBridgeRPCPolicyFor(parseFamily)
	if !parsePolicy.RetrySafe {
		return bridgeRetryPolicy{MaxAttempts: 1, PerAttemptTimeout: parsePolicy.TimeoutBudget}
	}
	return bridgeRetryPolicy{
		MaxAttempts:       3,
		PerAttemptTimeout: parsePolicy.TimeoutBudget,
		BackoffBase:       250 * time.Millisecond,
		BackoffMax:        2 * time.Second,
	}
}

func parseBridgeRetryDelay(parsePolicy bridgeRetryPolicy, parseAttempt int) time.Duration {
	if parsePolicy.BackoffBase <= 0 {
		return 0
	}
	parseDelay := parsePolicy.BackoffBase
	for parseI := 1; parseI < parseAttempt; parseI++ {
		parseDelay *= 2
		if parsePolicy.BackoffMax > 0 && parseDelay >= parsePolicy.BackoffMax {
			return parsePolicy.BackoffMax
		}
	}
	if parsePolicy.BackoffMax > 0 && parseDelay > parsePolicy.BackoffMax {
		return parsePolicy.BackoffMax
	}
	return parseDelay
}

func parseBridgeRetryableStatus(parseErr error) bool {
	if parseErr == nil {
		return false
	}
	parseStatusErr, parseOk := status.FromError(parseErr)
	if !parseOk {
		return true
	}
	switch parseStatusErr.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled, codes.Unknown, codes.ResourceExhausted:
		return true
	default:
		return false
	}
}

type bridgeBestEffortDecision string

const (
	bridgeBestEffortRun   bridgeBestEffortDecision = "run"
	bridgeBestEffortDefer bridgeBestEffortDecision = "defer"
	bridgeBestEffortSkip  bridgeBestEffortDecision = "skip"
)

func parseBridgeBestEffortDecision(parseState bridgeState, parsePolicy bridgeRPCPolicy) bridgeBestEffortDecision {
	if parsePolicy.TrafficClass != "best_effort" && !parsePolicy.QueueBestEffort {
		if parseState == bridgeStateReady || (parseState == bridgeStateReconnecting && parsePolicy.AllowWhileReconnecting) {
			return bridgeBestEffortRun
		}
		return bridgeBestEffortSkip
	}
	switch parseState {
	case bridgeStateReady:
		return bridgeBestEffortRun
	case bridgeStateReconnecting, bridgeStateSleeping:
		if parsePolicy.QueueBestEffort {
			return bridgeBestEffortDefer
		}
		return bridgeBestEffortSkip
	case bridgeStateDegraded, bridgeStateOffline, bridgeStateBooting:
		if parsePolicy.QueueBestEffort {
			return bridgeBestEffortDefer
		}
		return bridgeBestEffortSkip
	default:
		return bridgeBestEffortSkip
	}
}

func parseBridgeStatusLabel(parseState bridgeState) string {
	switch parseState {
	case bridgeStateReconnecting:
		return "Retrying connection"
	case bridgeStateDegraded:
		return "Connection degraded"
	case bridgeStateSleeping:
		return "Connection paused while this tab sleeps"
	case bridgeStateOffline:
		return "Offline"
	case bridgeStateBooting:
		return "Starting connection"
	default:
		return ""
	}
}
