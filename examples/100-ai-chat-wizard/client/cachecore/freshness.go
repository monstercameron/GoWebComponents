package cachecore

import (
	"time"
)

// FreshnessState describes one record freshness decision at read time.
type FreshnessState string

const (
	FreshnessMissing FreshnessState = "missing"
	FreshnessFresh   FreshnessState = "fresh"
	FreshnessStale   FreshnessState = "stale"
	FreshnessExpired FreshnessState = "expired"
)

// BuildFreshnessWindow formats fetched/stale/expire timestamps from one fetched-at baseline and TTL policy.
func BuildFreshnessWindow(parseFetchedAt time.Time, parseStaleAfter time.Duration, parseExpiresAfter time.Duration) (string, string, string) {
	if parseFetchedAt.IsZero() {
		parseFetchedAt = time.Now().UTC()
	}
	if parseStaleAfter < 0 {
		parseStaleAfter = 0
	}
	if parseExpiresAfter < parseStaleAfter {
		parseExpiresAfter = parseStaleAfter
	}
	parseFetchedAt = parseFetchedAt.UTC()
	parseStaleAt := parseFetchedAt.Add(parseStaleAfter).UTC()
	parseExpiresAt := parseFetchedAt.Add(parseExpiresAfter).UTC()
	return parseFetchedAt.Format(time.RFC3339), parseStaleAt.Format(time.RFC3339), parseExpiresAt.Format(time.RFC3339)
}

// ResolveFreshnessState resolves one freshness state from one record envelope and one read timestamp.
func ResolveFreshnessState(parseRecord CacheRecordEnvelope, parseNow time.Time) FreshnessState {
	if parseNow.IsZero() {
		parseNow = time.Now().UTC()
	}
	parseExpiresAt, isParseExpiresValid := parseParseRFC3339(parseRecord.ExpiresAt)
	parseStaleAt, isParseStaleValid := parseParseRFC3339(parseRecord.StaleAt)
	if !isParseExpiresValid && !isParseStaleValid {
		return FreshnessMissing
	}
	if isParseExpiresValid && !parseExpiresAt.After(parseNow) {
		return FreshnessExpired
	}
	if isParseStaleValid && !parseStaleAt.After(parseNow) {
		return FreshnessStale
	}
	return FreshnessFresh
}

// parseParseRFC3339 parses one RFC3339 timestamp and reports parse success.
func parseParseRFC3339(parseValue string) (time.Time, bool) {
	parseTime, parseErr := time.Parse(time.RFC3339, parseValue)
	if parseErr != nil {
		return time.Time{}, false
	}
	return parseTime.UTC(), true
}
