package cachepolicy

import (
	"context"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/client/cachecore"
)

const settingsSnapshotResourcePrefix = "settings.snapshot"

const (
	settingsSnapshotSectionProfile     = "profile"
	settingsSnapshotSectionTone        = "tone"
	settingsSnapshotSectionPrompt      = "prompt"
	settingsSnapshotSectionReasoning   = "reasoning"
	settingsSnapshotSectionTTS         = "tts"
	settingsSnapshotSectionPreferences = "remembered_preferences"
)

// BuildSettingsSnapshotPolicy returns one settings/profile snapshot cache policy.
func BuildSettingsSnapshotPolicy() cachecore.CachePolicy {
	return cachecore.BuildCachePolicy(cachecore.PolicyClassSession, 30*time.Second, 45*time.Minute, true, true, false)
}

// BuildSettingsSnapshotResourceKey builds one settings snapshot key for one section.
func BuildSettingsSnapshotResourceKey(parseSection string) string {
	parseSection = normalizeSettingsSnapshotSection(parseSection)
	return settingsSnapshotResourcePrefix + "|section=" + parseSection
}

// StoreSettingsSnapshot stores one settings/profile snapshot payload for one section.
func StoreSettingsSnapshot(parseCtx context.Context, parseStorage cachecore.Storage, parseScopeKey string, parseSection string, parseVersion string, parsePayload []byte) error {
	if parseStorage == nil {
		return nil
	}
	parsePolicy := BuildSettingsSnapshotPolicy()
	parseRecord := cachecore.BuildCacheRecordEnvelope(
		strings.TrimSpace(parseScopeKey),
		BuildSettingsSnapshotResourceKey(parseSection),
		strings.TrimSpace(parseVersion),
		"",
		time.Now().UTC(),
		parsePolicy.StaleAfter,
		parsePolicy.ExpiresAfter,
		append([]byte(nil), parsePayload...),
		cachecore.CacheStatusReady,
		"",
	)
	return parseStorage.Set(parseCtx, parseRecord)
}

// ReadSettingsSnapshot reads one settings/profile snapshot with snapshot-first SWR behavior.
func ReadSettingsSnapshot(
	parseCtx context.Context,
	parseAPI *cachecore.UIAPI,
	parseScopeKey string,
	parseSection string,
	parseRefresh func(context.Context, string, string) (cachecore.CacheRecordEnvelope, error),
) (cachecore.CachedResourceView, error) {
	if parseAPI == nil {
		return cachecore.CachedResourceView{}, nil
	}
	return parseAPI.ReadCachedResource(
		parseCtx,
		strings.TrimSpace(parseScopeKey),
		BuildSettingsSnapshotResourceKey(parseSection),
		BuildSettingsSnapshotPolicy(),
		parseRefresh,
	)
}

// ApplySettingsSnapshotWriteSuccess merges one successful write payload into the cached settings snapshot.
func ApplySettingsSnapshotWriteSuccess(parseCtx context.Context, parseStorage cachecore.Storage, parseSnapshot *cachecore.SnapshotStore, parseScopeKey string, parseSection string, parseVersion string, parsePayload []byte) error {
	if parseStorage == nil {
		return nil
	}
	parsePolicy := BuildSettingsSnapshotPolicy()
	parseRecord := cachecore.BuildCacheRecordEnvelope(
		strings.TrimSpace(parseScopeKey),
		BuildSettingsSnapshotResourceKey(parseSection),
		strings.TrimSpace(parseVersion),
		"",
		time.Now().UTC(),
		parsePolicy.StaleAfter,
		parsePolicy.ExpiresAfter,
		append([]byte(nil), parsePayload...),
		cachecore.CacheStatusReady,
		"",
	)
	if parseErr := parseStorage.Set(parseCtx, parseRecord); parseErr != nil {
		return parseErr
	}
	if parseSnapshot != nil {
		parseSnapshot.SetSync(parseRecord)
	}
	return nil
}

// ApplySettingsSnapshotWriteFailure marks one settings snapshot as stale/unavailable while retaining the last payload.
func ApplySettingsSnapshotWriteFailure(parseCtx context.Context, parseStorage cachecore.Storage, parseSnapshot *cachecore.SnapshotStore, parseScopeKey string, parseSection string, parseUnavailableReason string) error {
	if parseStorage == nil {
		return nil
	}
	parseScopeKey = strings.TrimSpace(parseScopeKey)
	parseResourceKey := BuildSettingsSnapshotResourceKey(parseSection)
	parseScopedResourceKey := cachecore.BuildScopedResourceKey(parseScopeKey, parseResourceKey)
	parseRecord, isParseFound, parseErr := parseStorage.Get(parseCtx, parseScopedResourceKey)
	if parseErr != nil {
		return parseErr
	}
	if !isParseFound {
		parseRecord = cachecore.BuildCacheRecordEnvelope(parseScopeKey, parseResourceKey, "unavailable", "", time.Now().UTC(), 0, BuildSettingsSnapshotPolicy().ExpiresAfter, []byte{}, cachecore.CacheStatusStale, strings.TrimSpace(parseUnavailableReason))
	} else {
		parseRecord = cachecore.NormalizeCacheRecordEnvelope(parseRecord)
		parseRecord.Status = cachecore.CacheStatusStale
		parseRecord.Error = strings.TrimSpace(parseUnavailableReason)
	}
	if parseErr = parseStorage.Set(parseCtx, parseRecord); parseErr != nil {
		return parseErr
	}
	if parseSnapshot != nil {
		parseSnapshot.SetSync(parseRecord)
	}
	return nil
}

// normalizeSettingsSnapshotSection normalizes one settings snapshot section token.
func normalizeSettingsSnapshotSection(parseSection string) string {
	parseSection = strings.TrimSpace(strings.ToLower(parseSection))
	switch parseSection {
	case settingsSnapshotSectionProfile,
		settingsSnapshotSectionTone,
		settingsSnapshotSectionPrompt,
		settingsSnapshotSectionReasoning,
		settingsSnapshotSectionTTS,
		settingsSnapshotSectionPreferences:
		return parseSection
	default:
		return settingsSnapshotSectionProfile
	}
}
