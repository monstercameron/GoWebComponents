package cachecore

import (
	"errors"
	"testing"
)

// TestEnforceLocalSecrecyForCacheRecord verifies forbidden cache payload fields are denied.
func TestEnforceLocalSecrecyForCacheRecord(parseT *testing.T) {
	parseErr := EnforceLocalSecrecyForCacheRecord(CacheRecordEnvelope{
		Payload: []byte(`{"authorization":"Bearer abc.def.ghi"}`),
	})
	if !errors.Is(parseErr, errLocalSecrecyDenied) {
		parseT.Fatalf("expected local secrecy denial for bearer token payload, got %v", parseErr)
	}
}

// TestEnforceLocalSecrecyForOutboxRecord verifies forbidden outbox payload fields are denied.
func TestEnforceLocalSecrecyForOutboxRecord(parseT *testing.T) {
	parseErr := EnforceLocalSecrecyForOutboxRecord(OutboxRecordEnvelope{
		Payload: []byte(`{"provider_secret":"top-secret"}`),
	})
	if !errors.Is(parseErr, errLocalSecrecyDenied) {
		parseT.Fatalf("expected local secrecy denial for provider_secret payload, got %v", parseErr)
	}
}

// TestEnforceLocalSecrecyAllowsSafePayload verifies ordinary payloads remain allowed.
func TestEnforceLocalSecrecyAllowsSafePayload(parseT *testing.T) {
	parseErr := EnforceLocalSecrecyForCacheRecord(CacheRecordEnvelope{
		Payload: []byte(`{"conversation_id":"conv-1","draft":"hello"}`),
	})
	if parseErr != nil {
		parseT.Fatalf("expected safe payload to pass secrecy checks, got %v", parseErr)
	}
}
