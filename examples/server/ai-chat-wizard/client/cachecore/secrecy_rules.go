package cachecore

import (
	"encoding/json"
	"errors"
	"regexp"
	"slices"
	"strings"
)

var errLocalSecrecyDenied = errors.New("local secrecy rules denied persistence")

var parseForbiddenSecrecyKeyFragments = []string{
	"authorization",
	"bearer",
	"password_reset",
	"reset_token",
	"access_token",
	"refresh_token",
	"id_token",
	"api_key",
	"secret",
	"credential",
	"provider_key",
	"provider_secret",
	"webhook_secret",
	"hidden_email",
	"private_email",
	"operator_only",
}

var parseForbiddenSecrecyValuePattern = regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9\-\._~\+/]+=*`)

// EnforceLocalSecrecyForCacheRecord validates cache-record payloads against local secrecy persistence rules.
func EnforceLocalSecrecyForCacheRecord(parseRecord CacheRecordEnvelope) error {
	return enforceLocalSecrecyForPayload(parseRecord.Payload)
}

// EnforceLocalSecrecyForOutboxRecord validates outbox-record payloads against local secrecy persistence rules.
func EnforceLocalSecrecyForOutboxRecord(parseRecord OutboxRecordEnvelope) error {
	return enforceLocalSecrecyForPayload(parseRecord.Payload)
}

// enforceLocalSecrecyForPayload checks one JSON payload for forbidden secret fields and patterns.
func enforceLocalSecrecyForPayload(parsePayload []byte) error {
	if len(parsePayload) == 0 {
		return nil
	}
	var parseDecoded any
	if parseErr := json.Unmarshal(parsePayload, &parseDecoded); parseErr != nil {
		parseText := strings.ToLower(strings.TrimSpace(string(parsePayload)))
		if parseForbiddenSecrecyValuePattern.MatchString(parseText) {
			return errLocalSecrecyDenied
		}
		return nil
	}
	if hasForbiddenSecrecy(parseDecoded) {
		return errLocalSecrecyDenied
	}
	return nil
}

// hasForbiddenSecrecy recursively checks one decoded payload node for forbidden secrecy keys and values.
func hasForbiddenSecrecy(parseNode any) bool {
	switch parseValue := parseNode.(type) {
	case map[string]any:
		for parseKey, parseChild := range parseValue {
			if isForbiddenSecrecyKey(parseKey) {
				return true
			}
			if hasForbiddenSecrecy(parseChild) {
				return true
			}
		}
		return false
	case []any:
		return slices.ContainsFunc(parseValue, hasForbiddenSecrecy)
	case string:
		return parseForbiddenSecrecyValuePattern.MatchString(parseValue)
	default:
		return false
	}
}

// isForbiddenSecrecyKey reports whether one key name is forbidden for local persistence.
func isForbiddenSecrecyKey(parseKey string) bool {
	parseKey = strings.TrimSpace(strings.ToLower(parseKey))
	if parseKey == "" {
		return false
	}
	for _, parseFragment := range parseForbiddenSecrecyKeyFragments {
		if strings.Contains(parseKey, parseFragment) {
			return true
		}
	}
	return false
}
