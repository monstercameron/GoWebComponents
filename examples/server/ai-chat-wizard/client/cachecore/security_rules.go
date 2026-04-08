package cachecore

import (
	"errors"
	"strings"
)

// SecurityIntent identifies one cache action under security policy evaluation.
type SecurityIntent string

const (
	SecurityIntentDisplayRead SecurityIntent = "display_read"
	SecurityIntentCacheWrite  SecurityIntent = "cache_write"
	SecurityIntentQueueWrite  SecurityIntent = "queue_write"
)

// SecurityRule stores one resource-prefix security rule.
type SecurityRule struct {
	ResourcePrefix                 string
	IsDisplayOnly                  bool
	IsQueueWriteAllowed            bool
	MustUseServerAuthorizationGate bool
}

// SecurityDecision stores one evaluated decision for one resource+intent.
type SecurityDecision struct {
	IsAllowed                      bool
	IsDisplayOnly                  bool
	IsQueueWriteAllowed            bool
	MustUseServerAuthorizationGate bool
	Reason                         string
}

var errCacheSecurityDenied = errors.New("cache security rule denied")

// BuildDefaultSecurityRules returns default security rules for sensitive display-only resources.
func BuildDefaultSecurityRules() []SecurityRule {
	return []SecurityRule{
		{ResourcePrefix: "auth.", IsDisplayOnly: true, IsQueueWriteAllowed: false, MustUseServerAuthorizationGate: true},
		{ResourcePrefix: "bootstrap.", IsDisplayOnly: true, IsQueueWriteAllowed: false, MustUseServerAuthorizationGate: true},
		{ResourcePrefix: "admin.", IsDisplayOnly: true, IsQueueWriteAllowed: false, MustUseServerAuthorizationGate: true},
		{ResourcePrefix: "billing.", IsDisplayOnly: true, IsQueueWriteAllowed: false, MustUseServerAuthorizationGate: true},
		{ResourcePrefix: "provider_health.", IsDisplayOnly: true, IsQueueWriteAllowed: false, MustUseServerAuthorizationGate: true},
	}
}

// ResolveSecurityDecision evaluates cache security behavior for one resource and intent.
func ResolveSecurityDecision(parseResourceKey string, parseIntent SecurityIntent, parseRules []SecurityRule) SecurityDecision {
	parseResourceKey = strings.TrimSpace(strings.ToLower(parseResourceKey))
	parseRule := resolveSecurityRule(parseResourceKey, parseRules)
	parseDecision := SecurityDecision{
		IsAllowed:                      true,
		IsDisplayOnly:                  parseRule.IsDisplayOnly,
		IsQueueWriteAllowed:            parseRule.IsQueueWriteAllowed,
		MustUseServerAuthorizationGate: parseRule.MustUseServerAuthorizationGate,
		Reason:                         "allowed",
	}
	switch parseIntent {
	case SecurityIntentDisplayRead:
		parseDecision.IsAllowed = true
	case SecurityIntentCacheWrite:
		if parseRule.IsDisplayOnly {
			parseDecision.IsAllowed = false
			parseDecision.Reason = "display_only_resource"
		}
	case SecurityIntentQueueWrite:
		parseDecision.MustUseServerAuthorizationGate = true
		if !parseRule.IsQueueWriteAllowed {
			parseDecision.IsAllowed = false
			parseDecision.Reason = "queue_write_not_allowed"
		}
	default:
		parseDecision.IsAllowed = false
		parseDecision.Reason = "unsupported_intent"
	}
	return parseDecision
}

// EnforceSecurityDecision returns one error when one security decision denies an action.
func EnforceSecurityDecision(parseDecision SecurityDecision) error {
	if parseDecision.IsAllowed {
		return nil
	}
	return errCacheSecurityDenied
}

// resolveSecurityRule resolves one best prefix-match rule for one resource key.
func resolveSecurityRule(parseResourceKey string, parseRules []SecurityRule) SecurityRule {
	parseResolved := SecurityRule{
		ResourcePrefix:                 "",
		IsDisplayOnly:                  false,
		IsQueueWriteAllowed:            true,
		MustUseServerAuthorizationGate: true,
	}
	parseLongestMatch := 0
	for _, parseRule := range parseRules {
		parsePrefix := strings.TrimSpace(strings.ToLower(parseRule.ResourcePrefix))
		if parsePrefix == "" || !strings.HasPrefix(parseResourceKey, parsePrefix) {
			continue
		}
		if len(parsePrefix) <= parseLongestMatch {
			continue
		}
		parseLongestMatch = len(parsePrefix)
		parseResolved = parseRule
	}
	return parseResolved
}
