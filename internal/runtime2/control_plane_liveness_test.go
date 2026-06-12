package runtime2

import (
	"testing"
	"time"
)

// TestHandleShardSessionReceiveControlEnvelopeReturnsWithoutBlockingWait verifies empty receive paths return quickly and do not block waiting for control traffic.
func TestHandleShardSessionReceiveControlEnvelopeReturnsWithoutBlockingWait(parseT *testing.T) {
	parsePort := &fakeShardSessionPort{}
	parseSession, parseSessionErr := BuildShardSession("shard-a", parsePort)
	if parseSessionErr != nil {
		parseT.Fatalf("BuildShardSession returned error: %v", parseSessionErr)
	}
	parseStartedAt := time.Now()
	parseEnvelope, hasEnvelope, parseEnvelopeErr := parseSession.HandleShardSessionReceiveControlEnvelope()
	parseElapsed := time.Since(parseStartedAt)
	if parseEnvelopeErr != nil {
		parseT.Fatalf("HandleShardSessionReceiveControlEnvelope returned error: %v", parseEnvelopeErr)
	}
	if hasEnvelope {
		parseT.Fatalf("expected no envelope when inbound queue is empty, got %+v", parseEnvelope)
	}
	if parseElapsed > 50*time.Millisecond {
		parseT.Fatalf("expected non-blocking empty receive path, elapsed=%s", parseElapsed)
	}
}

// TestHandleShardSessionReceiveControlEnvelopeDrainsQueueWithoutBlockingWait verifies queued control traffic drains promptly without polling sleeps.
func TestHandleShardSessionReceiveControlEnvelopeDrainsQueueWithoutBlockingWait(parseT *testing.T) {
	parsePort := &fakeShardSessionPort{}
	parseSession, parseSessionErr := BuildShardSession("shard-a", parsePort)
	if parseSessionErr != nil {
		parseT.Fatalf("BuildShardSession returned error: %v", parseSessionErr)
	}
	parsePayload, parsePayloadErr := BuildControlEnvelopeJSON(BuildControlReadyEnvelope())
	if parsePayloadErr != nil {
		parseT.Fatalf("BuildControlEnvelopeJSON returned error: %v", parsePayloadErr)
	}
	const getQueuedEnvelopeCount = 128
	for range getQueuedEnvelopeCount {
		parsePort.getOnMessage(parsePayload)
	}
	parseStartedAt := time.Now()
	for parseIndex := range getQueuedEnvelopeCount {
		parseEnvelope, hasEnvelope, parseEnvelopeErr := parseSession.HandleShardSessionReceiveControlEnvelope()
		if parseEnvelopeErr != nil {
			parseT.Fatalf("HandleShardSessionReceiveControlEnvelope(queue index %d) returned error: %v", parseIndex, parseEnvelopeErr)
		}
		if !hasEnvelope {
			parseT.Fatalf("expected queued envelope at index %d", parseIndex)
		}
		if parseEnvelope.Kind != ControlKindReady {
			parseT.Fatalf("expected queued ready envelope at index %d, got %q", parseIndex, parseEnvelope.Kind)
		}
	}
	parseElapsed := time.Since(parseStartedAt)
	if parseElapsed > 200*time.Millisecond {
		parseT.Fatalf("expected queued receive drain to remain event-driven, elapsed=%s", parseElapsed)
	}
}
