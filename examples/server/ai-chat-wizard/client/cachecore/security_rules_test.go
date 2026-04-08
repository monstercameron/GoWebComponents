package cachecore

import (
	"errors"
	"testing"
)

// TestResolveSecurityDecisionDisplayOnly verifies display-only resources deny cache and queue writes.
func TestResolveSecurityDecisionDisplayOnly(parseT *testing.T) {
	parseRules := BuildDefaultSecurityRules()
	parseCacheWriteDecision := ResolveSecurityDecision("billing.summary", SecurityIntentCacheWrite, parseRules)
	if parseCacheWriteDecision.IsAllowed {
		parseT.Fatalf("expected display-only cache write deny, decision=%+v", parseCacheWriteDecision)
	}
	parseQueueWriteDecision := ResolveSecurityDecision("admin.dashboard", SecurityIntentQueueWrite, parseRules)
	if parseQueueWriteDecision.IsAllowed || parseQueueWriteDecision.Reason != "queue_write_not_allowed" {
		parseT.Fatalf("expected display-only queue write deny, decision=%+v", parseQueueWriteDecision)
	}
}

// TestResolveSecurityDecisionGenericResource verifies generic resources allow writes with server-authorization requirement.
func TestResolveSecurityDecisionGenericResource(parseT *testing.T) {
	parseRules := BuildDefaultSecurityRules()
	parseDecision := ResolveSecurityDecision("chat.thread", SecurityIntentQueueWrite, parseRules)
	if !parseDecision.IsAllowed {
		parseT.Fatalf("expected generic queue write allowed, decision=%+v", parseDecision)
	}
	if !parseDecision.MustUseServerAuthorizationGate {
		parseT.Fatalf("expected queue write to require server-authorization gate, decision=%+v", parseDecision)
	}
}

// TestEnforceSecurityDecision verifies denied decisions return one stable error.
func TestEnforceSecurityDecision(parseT *testing.T) {
	parseErr := EnforceSecurityDecision(SecurityDecision{IsAllowed: false})
	if !errors.Is(parseErr, errCacheSecurityDenied) {
		parseT.Fatalf("expected errCacheSecurityDenied, got %v", parseErr)
	}
	if parseErr2 := EnforceSecurityDecision(SecurityDecision{IsAllowed: true}); parseErr2 != nil {
		parseT.Fatalf("expected nil for allowed decision, got %v", parseErr2)
	}
}
