package cachecore

import (
	"encoding/json"
	"strings"
	"time"
)

// CacheRecordEnvelope stores one persistent cache row with canonical persistence shape.
type CacheRecordEnvelope struct {
	ScopeKey    string `json:"scope_key"`
	ResourceKey string `json:"resource_key"`
	Version     string `json:"version"`
	ETagOrHash  string `json:"etag_or_hash"`
	FetchedAt   string `json:"fetched_at"`
	StaleAt     string `json:"stale_at"`
	ExpiresAt   string `json:"expires_at"`
	Payload     []byte `json:"parsePayload"`
	Status      string `json:"parseStatus"`
	Error       string `json:"parseError"`
}

// OutboxRecordEnvelope stores one queued-write row for retryable mutation delivery.
type OutboxRecordEnvelope struct {
	QueueKey      string `json:"queue_key"`
	ResourceKey   string `json:"resource_key"`
	OperationKey  string `json:"operation_key"`
	CreatedAt     string `json:"created_at"`
	LastAttemptAt string `json:"last_attempt_at"`
	AttemptCount  int    `json:"attempt_count"`
	NextRetryAt   string `json:"next_retry_at"`
	Payload       []byte `json:"parsePayload"`
	Status        string `json:"parseStatus"`
	AckKey        string `json:"parseAckKey"`
	LastError     string `json:"parseLastError"`
}

const (
	CacheStatusReady = "ready"
	CacheStatusStale = "stale"
	CacheStatusError = "error"
)

const (
	OutboxStatusQueued  = "queued"
	OutboxStatusSending = "sending"
	OutboxStatusAcked   = "acked"
	OutboxStatusFailed  = "failed"
)

// BuildCacheRecordEnvelope builds one canonical cache-record envelope using one freshness policy.
func BuildCacheRecordEnvelope(
	parseScopeKey string,
	parseResourceKey string,
	parseVersion string,
	parseETagOrHash string,
	parseFetchedAt time.Time,
	parseStaleAfter time.Duration,
	parseExpiresAfter time.Duration,
	parsePayload []byte,
	parseStatus string,
	parseError string,
) CacheRecordEnvelope {
	parseFetchedAtText, parseStaleAtText, parseExpiresAtText := BuildFreshnessWindow(parseFetchedAt, parseStaleAfter, parseExpiresAfter)
	return NormalizeCacheRecordEnvelope(CacheRecordEnvelope{
		ScopeKey:    parseScopeKey,
		ResourceKey: parseResourceKey,
		Version:     parseVersion,
		ETagOrHash:  parseETagOrHash,
		FetchedAt:   parseFetchedAtText,
		StaleAt:     parseStaleAtText,
		ExpiresAt:   parseExpiresAtText,
		Payload:     append([]byte(nil), parsePayload...),
		Status:      parseStatus,
		Error:       parseError,
	})
}

// NormalizeCacheRecordEnvelope normalizes one cache envelope into one canonical contract-safe shape.
func NormalizeCacheRecordEnvelope(parseRecord CacheRecordEnvelope) CacheRecordEnvelope {
	parseRecord.ScopeKey = strings.TrimSpace(parseRecord.ScopeKey)
	parseRecord.ResourceKey = strings.TrimSpace(parseRecord.ResourceKey)
	parseRecord.Version = strings.TrimSpace(parseRecord.Version)
	parseRecord.ETagOrHash = strings.TrimSpace(parseRecord.ETagOrHash)
	parseRecord.FetchedAt = strings.TrimSpace(parseRecord.FetchedAt)
	parseRecord.StaleAt = strings.TrimSpace(parseRecord.StaleAt)
	parseRecord.ExpiresAt = strings.TrimSpace(parseRecord.ExpiresAt)
	parseRecord.Status = parseNormalizeCacheStatus(parseRecord.Status)
	parseRecord.Error = strings.TrimSpace(parseRecord.Error)
	if parseRecord.Payload == nil {
		parseRecord.Payload = []byte{}
	} else {
		parseRecord.Payload = append([]byte(nil), parseRecord.Payload...)
	}
	return parseRecord
}

// BuildCacheRecordEnvelopeJSON marshals one normalized cache-record envelope into JSON.
func BuildCacheRecordEnvelopeJSON(parseRecord CacheRecordEnvelope) ([]byte, error) {
	return json.Marshal(NormalizeCacheRecordEnvelope(parseRecord))
}

// ParseCacheRecordEnvelopeJSON unmarshals one persisted cache-record envelope JSON payload.
func ParseCacheRecordEnvelopeJSON(parseRaw []byte) (CacheRecordEnvelope, error) {
	parseRecord := CacheRecordEnvelope{}
	if parseErr := json.Unmarshal(parseRaw, &parseRecord); parseErr != nil {
		return CacheRecordEnvelope{}, parseErr
	}
	return NormalizeCacheRecordEnvelope(parseRecord), nil
}

// parseNormalizeCacheStatus resolves one stable cache status value.
func parseNormalizeCacheStatus(parseStatus string) string {
	switch strings.TrimSpace(strings.ToLower(parseStatus)) {
	case CacheStatusStale:
		return CacheStatusStale
	case CacheStatusError:
		return CacheStatusError
	default:
		return CacheStatusReady
	}
}

// BuildOutboxRecordEnvelope builds one canonical queued-write envelope for retry orchestration.
func BuildOutboxRecordEnvelope(
	parseQueueKey string,
	parseResourceKey string,
	parseOperationKey string,
	parseCreatedAt time.Time,
	parseNextRetryAt time.Time,
	parsePayload []byte,
	parseStatus string,
	parseAckKey string,
	parseLastError string,
) OutboxRecordEnvelope {
	if parseCreatedAt.IsZero() {
		parseCreatedAt = time.Now().UTC()
	}
	return NormalizeOutboxRecordEnvelope(OutboxRecordEnvelope{
		QueueKey:      parseQueueKey,
		ResourceKey:   parseResourceKey,
		OperationKey:  parseOperationKey,
		CreatedAt:     parseCreatedAt.UTC().Format(time.RFC3339),
		LastAttemptAt: "",
		AttemptCount:  0,
		NextRetryAt:   parseNextRetryAt.UTC().Format(time.RFC3339),
		Payload:       append([]byte(nil), parsePayload...),
		Status:        parseStatus,
		AckKey:        parseAckKey,
		LastError:     parseLastError,
	})
}

// NormalizeOutboxRecordEnvelope normalizes one outbox envelope into one canonical contract-safe shape.
func NormalizeOutboxRecordEnvelope(parseRecord OutboxRecordEnvelope) OutboxRecordEnvelope {
	parseRecord.QueueKey = strings.TrimSpace(parseRecord.QueueKey)
	parseRecord.ResourceKey = strings.TrimSpace(parseRecord.ResourceKey)
	parseRecord.OperationKey = strings.TrimSpace(parseRecord.OperationKey)
	parseRecord.CreatedAt = strings.TrimSpace(parseRecord.CreatedAt)
	parseRecord.LastAttemptAt = strings.TrimSpace(parseRecord.LastAttemptAt)
	parseRecord.NextRetryAt = strings.TrimSpace(parseRecord.NextRetryAt)
	parseRecord.Status = parseNormalizeOutboxStatus(parseRecord.Status)
	parseRecord.AckKey = strings.TrimSpace(parseRecord.AckKey)
	parseRecord.LastError = strings.TrimSpace(parseRecord.LastError)
	if parseRecord.AttemptCount < 0 {
		parseRecord.AttemptCount = 0
	}
	if parseRecord.Payload == nil {
		parseRecord.Payload = []byte{}
	} else {
		parseRecord.Payload = append([]byte(nil), parseRecord.Payload...)
	}
	return parseRecord
}

// BuildOutboxRecordEnvelopeJSON marshals one normalized outbox-record envelope into JSON.
func BuildOutboxRecordEnvelopeJSON(parseRecord OutboxRecordEnvelope) ([]byte, error) {
	return json.Marshal(NormalizeOutboxRecordEnvelope(parseRecord))
}

// ParseOutboxRecordEnvelopeJSON unmarshals one persisted outbox-record envelope JSON payload.
func ParseOutboxRecordEnvelopeJSON(parseRaw []byte) (OutboxRecordEnvelope, error) {
	parseRecord := OutboxRecordEnvelope{}
	if parseErr := json.Unmarshal(parseRaw, &parseRecord); parseErr != nil {
		return OutboxRecordEnvelope{}, parseErr
	}
	return NormalizeOutboxRecordEnvelope(parseRecord), nil
}

// parseNormalizeOutboxStatus resolves one stable outbox status value.
func parseNormalizeOutboxStatus(parseStatus string) string {
	switch strings.TrimSpace(strings.ToLower(parseStatus)) {
	case OutboxStatusSending:
		return OutboxStatusSending
	case OutboxStatusAcked:
		return OutboxStatusAcked
	case OutboxStatusFailed:
		return OutboxStatusFailed
	default:
		return OutboxStatusQueued
	}
}
