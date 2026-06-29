package cachecore

import "time"

// PolicyClass identifies one cache policy profile category.
type PolicyClass string

const (
	PolicyClassStatic   PolicyClass = "static"
	PolicyClassSession  PolicyClass = "session"
	PolicyClassVolatile PolicyClass = "volatile"
	PolicyClassQueued   PolicyClass = "queued"
)

// CachePolicy describes freshness, background-refresh, and mutation-gating behavior.
type CachePolicy struct {
	Class                       PolicyClass
	StaleAfter                  time.Duration
	ExpiresAfter                time.Duration
	CanBackgroundRefresh        bool
	ShouldDisplayOnlyWhenStale  bool
	ShouldRefetchBeforeMutation bool
}

// PolicyReadDecision describes one read-path decision after applying one cache policy.
type PolicyReadDecision struct {
	CanReturnCached             bool
	IsStale                     bool
	ShouldRefreshInBackground   bool
	ShouldRefetchBeforeMutation bool
}

// BuildCachePolicy builds one normalized cache policy from explicit configuration.
func BuildCachePolicy(
	parseClass PolicyClass,
	parseStaleAfter time.Duration,
	parseExpiresAfter time.Duration,
	isParseBackgroundRefreshEnabled bool,
	isParseDisplayOnlyWhenStale bool,
	isParseMustRefetchBeforeMutation bool,
) CachePolicy {
	return NormalizeCachePolicy(CachePolicy{
		Class:                       parseClass,
		StaleAfter:                  parseStaleAfter,
		ExpiresAfter:                parseExpiresAfter,
		CanBackgroundRefresh:        isParseBackgroundRefreshEnabled,
		ShouldDisplayOnlyWhenStale:  isParseDisplayOnlyWhenStale,
		ShouldRefetchBeforeMutation: isParseMustRefetchBeforeMutation,
	})
}

// NormalizeCachePolicy normalizes one policy to valid class/duration bounds.
func NormalizeCachePolicy(parsePolicy CachePolicy) CachePolicy {
	parsePolicy.Class = normalizePolicyClass(parsePolicy.Class)
	if parsePolicy.StaleAfter < 0 {
		parsePolicy.StaleAfter = 0
	}
	if parsePolicy.ExpiresAfter < parsePolicy.StaleAfter {
		parsePolicy.ExpiresAfter = parsePolicy.StaleAfter
	}
	return parsePolicy
}

// ResolveCachePolicyDefaults returns one default policy for one class.
func ResolveCachePolicyDefaults(parseClass PolicyClass) CachePolicy {
	switch normalizePolicyClass(parseClass) {
	case PolicyClassStatic:
		return BuildCachePolicy(PolicyClassStatic, 30*time.Minute, 24*time.Hour, true, true, false)
	case PolicyClassSession:
		return BuildCachePolicy(PolicyClassSession, 2*time.Minute, 30*time.Minute, true, true, false)
	case PolicyClassQueued:
		return BuildCachePolicy(PolicyClassQueued, 0, 7*24*time.Hour, false, true, true)
	default:
		return BuildCachePolicy(PolicyClassVolatile, 15*time.Second, 2*time.Minute, true, false, true)
	}
}

// ResolvePolicyReadDecision applies one cache policy to one freshness state.
func ResolvePolicyReadDecision(parsePolicy CachePolicy, parseState FreshnessState) PolicyReadDecision {
	parsePolicy = NormalizeCachePolicy(parsePolicy)
	parseDecision := PolicyReadDecision{}
	switch parseState {
	case FreshnessFresh:
		parseDecision.CanReturnCached = true
	case FreshnessStale:
		parseDecision.CanReturnCached = true
		parseDecision.IsStale = true
		parseDecision.ShouldRefreshInBackground = parsePolicy.CanBackgroundRefresh
	case FreshnessExpired:
		parseDecision.IsStale = true
		parseDecision.CanReturnCached = parsePolicy.ShouldDisplayOnlyWhenStale
		parseDecision.ShouldRefreshInBackground = parsePolicy.CanBackgroundRefresh
	case FreshnessMissing:
		parseDecision.CanReturnCached = false
	default:
		parseDecision.CanReturnCached = false
	}
	parseDecision.ShouldRefetchBeforeMutation = parsePolicy.ShouldRefetchBeforeMutation
	return parseDecision
}

// normalizePolicyClass resolves one canonical policy class fallback.
func normalizePolicyClass(parseClass PolicyClass) PolicyClass {
	switch parseClass {
	case PolicyClassStatic, PolicyClassSession, PolicyClassVolatile, PolicyClassQueued:
		return parseClass
	default:
		return PolicyClassVolatile
	}
}
