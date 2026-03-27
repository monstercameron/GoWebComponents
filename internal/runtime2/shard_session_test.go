package runtime2

import (
	"bytes"
	"testing"
)

type fakeShardSessionPort struct {
	storePostPayloads [][]byte
	getOnMessage      func(parsePayload []byte)
}

// PostMessage stores one payload posted through the fake shard session port.
func (parsePort *fakeShardSessionPort) PostMessage(parsePayload []byte) error {
	parsePort.storePostPayloads = append(parsePort.storePostPayloads, append([]byte(nil), parsePayload...))
	return nil
}

// BindMessageHandler stores one handler callback for fake inbound message delivery.
func (parsePort *fakeShardSessionPort) BindMessageHandler(parseHandler func(parsePayload []byte)) {
	parsePort.getOnMessage = parseHandler
}

// TestBuildShardSessionBindsInboundPortHandler verifies shard sessions bind inbound handlers and queue received payloads.
func TestBuildShardSessionBindsInboundPortHandler(parseT *testing.T) {
	parsePort := &fakeShardSessionPort{}
	parseSession, parseSessionErr := BuildShardSession("shard-a", parsePort)
	if parseSessionErr != nil {
		parseT.Fatalf("BuildShardSession returned error: %v", parseSessionErr)
	}
	if parsePort.getOnMessage == nil {
		parseT.Fatal("expected shard session to bind port message handler")
	}
	parsePort.getOnMessage([]byte("inbound"))
	getPayload, hasPayload := parseSession.HandleShardSessionReceivePayload()
	if !hasPayload {
		parseT.Fatal("expected inbound payload in shard session receive queue")
	}
	if !bytes.Equal(getPayload, []byte("inbound")) {
		parseT.Fatalf("expected inbound payload %q, got %q", "inbound", string(getPayload))
	}
}

// TestHandleShardSessionSendPayloadUsesPort verifies shard-session send delegates through the backing port.
func TestHandleShardSessionSendPayloadUsesPort(parseT *testing.T) {
	parsePort := &fakeShardSessionPort{}
	parseSession, parseSessionErr := BuildShardSession("shard-a", parsePort)
	if parseSessionErr != nil {
		parseT.Fatalf("BuildShardSession returned error: %v", parseSessionErr)
	}
	if parseSendErr := parseSession.HandleShardSessionSendPayload([]byte("outbound")); parseSendErr != nil {
		parseT.Fatalf("HandleShardSessionSendPayload returned error: %v", parseSendErr)
	}
	if len(parsePort.storePostPayloads) != 1 {
		parseT.Fatalf("expected one posted payload, got %d", len(parsePort.storePostPayloads))
	}
	if !bytes.Equal(parsePort.storePostPayloads[0], []byte("outbound")) {
		parseT.Fatalf("expected outbound payload %q, got %q", "outbound", string(parsePort.storePostPayloads[0]))
	}
}

// TestBuildShardSessionWithQueueLimitCapsInboundPayloadQueue verifies burst inbound payload traffic is capped deterministically by the configured queue limit.
func TestBuildShardSessionWithQueueLimitCapsInboundPayloadQueue(parseT *testing.T) {
	parsePort := &fakeShardSessionPort{}
	parseSession, parseSessionErr := BuildShardSessionWithQueueLimit("shard-a", parsePort, 2)
	if parseSessionErr != nil {
		parseT.Fatalf("BuildShardSessionWithQueueLimit returned error: %v", parseSessionErr)
	}
	parsePort.getOnMessage([]byte("payload-1"))
	parsePort.getOnMessage([]byte("payload-2"))
	parsePort.getOnMessage([]byte("payload-3"))
	parseFirstPayload, hasFirstPayload := parseSession.HandleShardSessionReceivePayload()
	if !hasFirstPayload {
		parseT.Fatal("expected first bounded payload")
	}
	if !bytes.Equal(parseFirstPayload, []byte("payload-2")) {
		parseT.Fatalf("expected first bounded payload %q, got %q", "payload-2", string(parseFirstPayload))
	}
	parseSecondPayload, hasSecondPayload := parseSession.HandleShardSessionReceivePayload()
	if !hasSecondPayload {
		parseT.Fatal("expected second bounded payload")
	}
	if !bytes.Equal(parseSecondPayload, []byte("payload-3")) {
		parseT.Fatalf("expected second bounded payload %q, got %q", "payload-3", string(parseSecondPayload))
	}
	if _, hasThirdPayload := parseSession.HandleShardSessionReceivePayload(); hasThirdPayload {
		parseT.Fatal("expected inbound queue to remain capped at configured limit")
	}
}

// TestHandleShardSessionAcceptControlEnvelopeRequiresHandshakeBeforeMountOrUpdate verifies mount and update control traffic is blocked until ready/capabilities handshake completes.
func TestHandleShardSessionAcceptControlEnvelopeRequiresHandshakeBeforeMountOrUpdate(parseT *testing.T) {
	parsePort := &fakeShardSessionPort{}
	parseSession, parseSessionErr := BuildShardSession("shard-a", parsePort)
	if parseSessionErr != nil {
		parseT.Fatalf("BuildShardSession returned error: %v", parseSessionErr)
	}
	parseSnapshotEnvelope, parseSnapshotErr := BuildSnapshotEnvelope(
		"region-1",
		1,
		1,
		map[string]any{"title": "Orders"},
		nil,
		nil,
		nil,
	)
	if parseSnapshotErr != nil {
		parseT.Fatalf("BuildSnapshotEnvelope returned error: %v", parseSnapshotErr)
	}
	parseMountEnvelope, parseMountErr := BuildControlMountEnvelope("dashboard.hot-panel", parseSnapshotEnvelope)
	if parseMountErr != nil {
		parseT.Fatalf("BuildControlMountEnvelope returned error: %v", parseMountErr)
	}
	parseUpdateSnapshotEnvelope, parseUpdateSnapshotErr := BuildSnapshotEnvelope(
		"region-1",
		1,
		2,
		map[string]any{"title": "Orders"},
		nil,
		nil,
		nil,
	)
	if parseUpdateSnapshotErr != nil {
		parseT.Fatalf("BuildSnapshotEnvelope(update) returned error: %v", parseUpdateSnapshotErr)
	}
	parseUpdateEnvelope, parseUpdateErr := BuildControlUpdateEnvelope(parseUpdateSnapshotEnvelope)
	if parseUpdateErr != nil {
		parseT.Fatalf("BuildControlUpdateEnvelope returned error: %v", parseUpdateErr)
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(parseUpdateEnvelope); parseAcceptErr == nil {
		parseT.Fatal("expected update control envelope to fail before handshake")
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(parseMountEnvelope); parseAcceptErr == nil {
		parseT.Fatal("expected mount control envelope to fail before handshake")
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(BuildControlReadyEnvelope()); parseAcceptErr != nil {
		parseT.Fatalf("HandleShardSessionAcceptControlEnvelope(ready) returned error: %v", parseAcceptErr)
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(parseUpdateEnvelope); parseAcceptErr == nil {
		parseT.Fatal("expected update control envelope to fail before capabilities handshake")
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(parseMountEnvelope); parseAcceptErr == nil {
		parseT.Fatal("expected mount control envelope to fail before capabilities handshake")
	}
	parseCapabilitiesEnvelope, parseCapabilitiesErr := BuildControlCapabilitiesEnvelope(BuildCapabilityReport(CapabilitySource{
		HasWorkerSupport:          true,
		HasMessagePortSupport:     true,
		HasStructuredCloneSupport: true,
	}))
	if parseCapabilitiesErr != nil {
		parseT.Fatalf("BuildControlCapabilitiesEnvelope returned error: %v", parseCapabilitiesErr)
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(parseCapabilitiesEnvelope); parseAcceptErr != nil {
		parseT.Fatalf("HandleShardSessionAcceptControlEnvelope(capabilities) returned error: %v", parseAcceptErr)
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(parseMountEnvelope); parseAcceptErr != nil {
		parseT.Fatalf("HandleShardSessionAcceptControlEnvelope(mount) returned error after handshake: %v", parseAcceptErr)
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(parseUpdateEnvelope); parseAcceptErr != nil {
		parseT.Fatalf("HandleShardSessionAcceptControlEnvelope(update) returned error after handshake: %v", parseAcceptErr)
	}
}

// TestHandleShardSessionSendMountControlEnvelopeRequiresHandshake verifies mount sends are blocked until ready/capabilities handshake completes.
func TestHandleShardSessionSendMountControlEnvelopeRequiresHandshake(parseT *testing.T) {
	parsePort := &fakeShardSessionPort{}
	parseSession, parseSessionErr := BuildShardSession("shard-a", parsePort)
	if parseSessionErr != nil {
		parseT.Fatalf("BuildShardSession returned error: %v", parseSessionErr)
	}
	parseSnapshotEnvelope, parseSnapshotErr := BuildSnapshotEnvelope(
		"region-1",
		1,
		1,
		map[string]any{"title": "Orders"},
		nil,
		nil,
		nil,
	)
	if parseSnapshotErr != nil {
		parseT.Fatalf("BuildSnapshotEnvelope returned error: %v", parseSnapshotErr)
	}
	if parseSendErr := parseSession.HandleShardSessionSendMountControlEnvelope("dashboard.hot-panel", parseSnapshotEnvelope); parseSendErr == nil {
		parseT.Fatal("expected mount control send to fail before handshake")
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(BuildControlReadyEnvelope()); parseAcceptErr != nil {
		parseT.Fatalf("HandleShardSessionAcceptControlEnvelope(ready) returned error: %v", parseAcceptErr)
	}
	parseCapabilitiesEnvelope, parseCapabilitiesErr := BuildControlCapabilitiesEnvelope(BuildCapabilityReport(CapabilitySource{
		HasWorkerSupport:          true,
		HasMessagePortSupport:     true,
		HasStructuredCloneSupport: true,
	}))
	if parseCapabilitiesErr != nil {
		parseT.Fatalf("BuildControlCapabilitiesEnvelope returned error: %v", parseCapabilitiesErr)
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(parseCapabilitiesEnvelope); parseAcceptErr != nil {
		parseT.Fatalf("HandleShardSessionAcceptControlEnvelope(capabilities) returned error: %v", parseAcceptErr)
	}
	if parseSendErr := parseSession.HandleShardSessionSendMountControlEnvelope("dashboard.hot-panel", parseSnapshotEnvelope); parseSendErr != nil {
		parseT.Fatalf("HandleShardSessionSendMountControlEnvelope returned error after handshake: %v", parseSendErr)
	}
}

// TestHandleShardSessionSendLifecycleControlHelpersPostEnvelopes verifies host-side lifecycle helper sends post serialized control envelopes over one shard session.
func TestHandleShardSessionSendLifecycleControlHelpersPostEnvelopes(parseT *testing.T) {
	parsePort := &fakeShardSessionPort{}
	parseSession, parseSessionErr := BuildShardSession("shard-a", parsePort)
	if parseSessionErr != nil {
		parseT.Fatalf("BuildShardSession returned error: %v", parseSessionErr)
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(BuildControlReadyEnvelope()); parseAcceptErr != nil {
		parseT.Fatalf("HandleShardSessionAcceptControlEnvelope(ready) returned error: %v", parseAcceptErr)
	}
	parseCapabilitiesEnvelope, parseCapabilitiesErr := BuildControlCapabilitiesEnvelope(BuildCapabilityReport(CapabilitySource{
		HasWorkerSupport:          true,
		HasMessagePortSupport:     true,
		HasStructuredCloneSupport: true,
	}))
	if parseCapabilitiesErr != nil {
		parseT.Fatalf("BuildControlCapabilitiesEnvelope returned error: %v", parseCapabilitiesErr)
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(parseCapabilitiesEnvelope); parseAcceptErr != nil {
		parseT.Fatalf("HandleShardSessionAcceptControlEnvelope(capabilities) returned error: %v", parseAcceptErr)
	}
	parseMountSnapshotEnvelope, parseMountSnapshotErr := BuildSnapshotEnvelope(
		"region-1",
		1,
		1,
		map[string]any{"title": "Orders"},
		nil,
		nil,
		nil,
	)
	if parseMountSnapshotErr != nil {
		parseT.Fatalf("BuildSnapshotEnvelope(mount) returned error: %v", parseMountSnapshotErr)
	}
	parseUpdateSnapshotEnvelope, parseUpdateSnapshotErr := BuildSnapshotEnvelope(
		"region-1",
		1,
		2,
		map[string]any{"title": "Orders"},
		nil,
		nil,
		nil,
	)
	if parseUpdateSnapshotErr != nil {
		parseT.Fatalf("BuildSnapshotEnvelope(update) returned error: %v", parseUpdateSnapshotErr)
	}
	if parseSendErr := parseSession.HandleShardSessionSendMountControlEnvelope("dashboard.hot-panel", parseMountSnapshotEnvelope); parseSendErr != nil {
		parseT.Fatalf("HandleShardSessionSendMountControlEnvelope returned error: %v", parseSendErr)
	}
	if parseSendErr := parseSession.HandleShardSessionSendUpdateControlEnvelope(parseUpdateSnapshotEnvelope); parseSendErr != nil {
		parseT.Fatalf("HandleShardSessionSendUpdateControlEnvelope returned error: %v", parseSendErr)
	}
	if parseSendErr := parseSession.HandleShardSessionSendCancelControlEnvelope("region-1"); parseSendErr != nil {
		parseT.Fatalf("HandleShardSessionSendCancelControlEnvelope returned error: %v", parseSendErr)
	}
	if parseSendErr := parseSession.HandleShardSessionSendDisposeControlEnvelope("region-1"); parseSendErr != nil {
		parseT.Fatalf("HandleShardSessionSendDisposeControlEnvelope returned error: %v", parseSendErr)
	}
	if parseSendErr := parseSession.HandleShardSessionSendRestartControlEnvelope("region-1", 2); parseSendErr != nil {
		parseT.Fatalf("HandleShardSessionSendRestartControlEnvelope returned error: %v", parseSendErr)
	}
	if len(parsePort.storePostPayloads) != 5 {
		parseT.Fatalf("expected 5 posted control envelopes, got %d", len(parsePort.storePostPayloads))
	}
	parseExpectedKinds := []ControlKind{
		ControlKindMount,
		ControlKindUpdate,
		ControlKindCancel,
		ControlKindDispose,
		ControlKindRestart,
	}
	for parseIndex, parsePayload := range parsePort.storePostPayloads {
		parseEnvelope, parseEnvelopeErr := ParseControlEnvelopeJSON(parsePayload)
		if parseEnvelopeErr != nil {
			parseT.Fatalf("ParseControlEnvelopeJSON(payload %d) returned error: %v", parseIndex, parseEnvelopeErr)
		}
		if parseEnvelope.Kind != parseExpectedKinds[parseIndex] {
			parseT.Fatalf("expected posted kind %q at index %d, got %q", parseExpectedKinds[parseIndex], parseIndex, parseEnvelope.Kind)
		}
	}
}

// TestHandleShardSessionReceiveControlEnvelopeParsesInboundControlTraffic verifies inbound payloads parse into control envelopes through one shard session.
func TestHandleShardSessionReceiveControlEnvelopeParsesInboundControlTraffic(parseT *testing.T) {
	parsePort := &fakeShardSessionPort{}
	parseSession, parseSessionErr := BuildShardSession("shard-a", parsePort)
	if parseSessionErr != nil {
		parseT.Fatalf("BuildShardSession returned error: %v", parseSessionErr)
	}
	parsePayload, parsePayloadErr := BuildControlEnvelopeJSON(BuildControlReadyEnvelope())
	if parsePayloadErr != nil {
		parseT.Fatalf("BuildControlEnvelopeJSON returned error: %v", parsePayloadErr)
	}
	parsePort.getOnMessage(parsePayload)
	parseEnvelope, hasEnvelope, parseEnvelopeErr := parseSession.HandleShardSessionReceiveControlEnvelope()
	if parseEnvelopeErr != nil {
		parseT.Fatalf("HandleShardSessionReceiveControlEnvelope returned error: %v", parseEnvelopeErr)
	}
	if !hasEnvelope {
		parseT.Fatal("expected inbound control envelope")
	}
	if parseEnvelope.Kind != ControlKindReady {
		parseT.Fatalf("expected inbound ready kind, got %q", parseEnvelope.Kind)
	}
}

// TestHandleShardSessionSendWorkerControlHelpersPostEnvelopes verifies worker-side control helper sends post expected envelope kinds over one shard session.
func TestHandleShardSessionSendWorkerControlHelpersPostEnvelopes(parseT *testing.T) {
	parsePort := &fakeShardSessionPort{}
	parseSession, parseSessionErr := BuildShardSession("shard-a", parsePort)
	if parseSessionErr != nil {
		parseT.Fatalf("BuildShardSession returned error: %v", parseSessionErr)
	}
	parseCapabilityReport := BuildCapabilityReport(CapabilitySource{
		HasWorkerSupport:          true,
		HasMessagePortSupport:     true,
		HasStructuredCloneSupport: true,
	})
	if parseSendErr := parseSession.HandleShardSessionSendWorkerPatchReadyControlEnvelope("region-1", 4, 4, TransportTierStructuredClone); parseSendErr != nil {
		parseT.Fatalf("HandleShardSessionSendWorkerPatchReadyControlEnvelope returned error: %v", parseSendErr)
	}
	if parseSendErr := parseSession.HandleShardSessionSendWorkerDiagnosticControlEnvelope("region-1", ControlDiagnosticEnvelopeSpec{
		DiagnosticType: DiagnosticEventKindUpdate,
		DiagnosticText: "update complete",
	}); parseSendErr != nil {
		parseT.Fatalf("HandleShardSessionSendWorkerDiagnosticControlEnvelope returned error: %v", parseSendErr)
	}
	if parseSendErr := parseSession.HandleShardSessionSendWorkerRestartControlEnvelope("region-1", 2); parseSendErr != nil {
		parseT.Fatalf("HandleShardSessionSendWorkerRestartControlEnvelope returned error: %v", parseSendErr)
	}
	if parseSendErr := parseSession.HandleShardSessionSendWorkerReadyControlEnvelope(); parseSendErr != nil {
		parseT.Fatalf("HandleShardSessionSendWorkerReadyControlEnvelope returned error: %v", parseSendErr)
	}
	if parseSendErr := parseSession.HandleShardSessionSendWorkerCapabilitiesControlEnvelope(parseCapabilityReport); parseSendErr != nil {
		parseT.Fatalf("HandleShardSessionSendWorkerCapabilitiesControlEnvelope returned error: %v", parseSendErr)
	}
	if parseSendErr := parseSession.HandleShardSessionSendWorkerPongControlEnvelope("shard-a", 8); parseSendErr != nil {
		parseT.Fatalf("HandleShardSessionSendWorkerPongControlEnvelope returned error: %v", parseSendErr)
	}
	if len(parsePort.storePostPayloads) != 6 {
		parseT.Fatalf("expected 6 posted worker control envelopes, got %d", len(parsePort.storePostPayloads))
	}
	parseExpectedKinds := []ControlKind{
		ControlKindPatchReady,
		ControlKindDiagnostic,
		ControlKindRestart,
		ControlKindReady,
		ControlKindCapabilities,
		ControlKindPong,
	}
	for parseIndex, parsePayload := range parsePort.storePostPayloads {
		parseEnvelope, parseEnvelopeErr := ParseControlEnvelopeJSON(parsePayload)
		if parseEnvelopeErr != nil {
			parseT.Fatalf("ParseControlEnvelopeJSON(worker payload %d) returned error: %v", parseIndex, parseEnvelopeErr)
		}
		if parseEnvelope.Kind != parseExpectedKinds[parseIndex] {
			parseT.Fatalf("expected worker posted kind %q at index %d, got %q", parseExpectedKinds[parseIndex], parseIndex, parseEnvelope.Kind)
		}
	}
}

// TestHandleShardSessionAcceptControlEnvelopeRestartResetsHandshake verifies restart control traffic resets session handshake state and requires ready/capabilities renegotiation before mount acceptance resumes.
func TestHandleShardSessionAcceptControlEnvelopeRestartResetsHandshake(parseT *testing.T) {
	parsePort := &fakeShardSessionPort{}
	parseSession, parseSessionErr := BuildShardSession("shard-a", parsePort)
	if parseSessionErr != nil {
		parseT.Fatalf("BuildShardSession returned error: %v", parseSessionErr)
	}
	parseSnapshotEnvelope, parseSnapshotErr := BuildSnapshotEnvelope(
		"region-1",
		1,
		1,
		map[string]any{"title": "Orders"},
		nil,
		nil,
		nil,
	)
	if parseSnapshotErr != nil {
		parseT.Fatalf("BuildSnapshotEnvelope returned error: %v", parseSnapshotErr)
	}
	parseMountEnvelope, parseMountErr := BuildControlMountEnvelope("dashboard.hot-panel", parseSnapshotEnvelope)
	if parseMountErr != nil {
		parseT.Fatalf("BuildControlMountEnvelope returned error: %v", parseMountErr)
	}
	parseCapabilitiesEnvelope, parseCapabilitiesErr := BuildControlCapabilitiesEnvelope(BuildCapabilityReport(CapabilitySource{
		HasWorkerSupport:          true,
		HasMessagePortSupport:     true,
		HasStructuredCloneSupport: true,
	}))
	if parseCapabilitiesErr != nil {
		parseT.Fatalf("BuildControlCapabilitiesEnvelope returned error: %v", parseCapabilitiesErr)
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(BuildControlReadyEnvelope()); parseAcceptErr != nil {
		parseT.Fatalf("HandleShardSessionAcceptControlEnvelope(ready) returned error: %v", parseAcceptErr)
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(parseCapabilitiesEnvelope); parseAcceptErr != nil {
		parseT.Fatalf("HandleShardSessionAcceptControlEnvelope(capabilities) returned error: %v", parseAcceptErr)
	}
	if !parseSession.HasShardSessionHandshakeComplete() {
		parseT.Fatal("expected shard session handshake to be complete before restart")
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(parseMountEnvelope); parseAcceptErr != nil {
		parseT.Fatalf("HandleShardSessionAcceptControlEnvelope(mount) returned error before restart: %v", parseAcceptErr)
	}
	parseRestartEnvelope, parseRestartErr := BuildControlRestartEnvelope("region-1", 2)
	if parseRestartErr != nil {
		parseT.Fatalf("BuildControlRestartEnvelope returned error: %v", parseRestartErr)
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(parseRestartEnvelope); parseAcceptErr != nil {
		parseT.Fatalf("HandleShardSessionAcceptControlEnvelope(restart) returned error: %v", parseAcceptErr)
	}
	if parseSession.HasShardSessionHandshakeComplete() {
		parseT.Fatal("expected restart control envelope to reset session handshake")
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(parseMountEnvelope); parseAcceptErr == nil {
		parseT.Fatal("expected mount control envelope to fail before renegotiated handshake completes")
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(BuildControlReadyEnvelope()); parseAcceptErr != nil {
		parseT.Fatalf("HandleShardSessionAcceptControlEnvelope(renegotiated ready) returned error: %v", parseAcceptErr)
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(parseMountEnvelope); parseAcceptErr == nil {
		parseT.Fatal("expected mount control envelope to fail before renegotiated capabilities completes")
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(parseCapabilitiesEnvelope); parseAcceptErr != nil {
		parseT.Fatalf("HandleShardSessionAcceptControlEnvelope(renegotiated capabilities) returned error: %v", parseAcceptErr)
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(parseMountEnvelope); parseAcceptErr != nil {
		parseT.Fatalf("HandleShardSessionAcceptControlEnvelope(mount after renegotiation) returned error: %v", parseAcceptErr)
	}
}

// TestHandleShardSessionReplacePortResetsHandshakeAndRebindsInboundHandler verifies channel replacement rebinds inbound handling and forces one fresh handshake renegotiation before mount sends resume.
func TestHandleShardSessionReplacePortResetsHandshakeAndRebindsInboundHandler(parseT *testing.T) {
	parseOriginalPort := &fakeShardSessionPort{}
	parseSession, parseSessionErr := BuildShardSession("shard-a", parseOriginalPort)
	if parseSessionErr != nil {
		parseT.Fatalf("BuildShardSession returned error: %v", parseSessionErr)
	}
	parseSnapshotEnvelope, parseSnapshotErr := BuildSnapshotEnvelope(
		"region-1",
		1,
		1,
		map[string]any{"title": "Orders"},
		nil,
		nil,
		nil,
	)
	if parseSnapshotErr != nil {
		parseT.Fatalf("BuildSnapshotEnvelope returned error: %v", parseSnapshotErr)
	}
	parseCapabilitiesEnvelope, parseCapabilitiesErr := BuildControlCapabilitiesEnvelope(BuildCapabilityReport(CapabilitySource{
		HasWorkerSupport:          true,
		HasMessagePortSupport:     true,
		HasStructuredCloneSupport: true,
	}))
	if parseCapabilitiesErr != nil {
		parseT.Fatalf("BuildControlCapabilitiesEnvelope returned error: %v", parseCapabilitiesErr)
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(BuildControlReadyEnvelope()); parseAcceptErr != nil {
		parseT.Fatalf("HandleShardSessionAcceptControlEnvelope(ready) returned error: %v", parseAcceptErr)
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(parseCapabilitiesEnvelope); parseAcceptErr != nil {
		parseT.Fatalf("HandleShardSessionAcceptControlEnvelope(capabilities) returned error: %v", parseAcceptErr)
	}
	if parseSendErr := parseSession.HandleShardSessionSendMountControlEnvelope("dashboard.hot-panel", parseSnapshotEnvelope); parseSendErr != nil {
		parseT.Fatalf("HandleShardSessionSendMountControlEnvelope returned error before channel replacement: %v", parseSendErr)
	}
	if len(parseOriginalPort.storePostPayloads) != 1 {
		parseT.Fatalf("expected one posted payload on original port, got %d", len(parseOriginalPort.storePostPayloads))
	}

	parseReplacementPort := &fakeShardSessionPort{}
	if parseReplaceErr := parseSession.HandleShardSessionReplacePort(parseReplacementPort); parseReplaceErr != nil {
		parseT.Fatalf("HandleShardSessionReplacePort returned error: %v", parseReplaceErr)
	}
	if parseReplacementPort.getOnMessage == nil {
		parseT.Fatal("expected replacement port to receive one bound inbound handler")
	}
	if parseSession.HasShardSessionHandshakeComplete() {
		parseT.Fatal("expected channel replacement to reset session handshake")
	}
	if parseSendErr := parseSession.HandleShardSessionSendMountControlEnvelope("dashboard.hot-panel", parseSnapshotEnvelope); parseSendErr == nil {
		parseT.Fatal("expected mount control send to fail before renegotiated handshake")
	}
	parsePayload, parsePayloadErr := BuildControlEnvelopeJSON(BuildControlReadyEnvelope())
	if parsePayloadErr != nil {
		parseT.Fatalf("BuildControlEnvelopeJSON(ready) returned error: %v", parsePayloadErr)
	}
	parseReplacementPort.getOnMessage(parsePayload)
	parseEnvelope, hasEnvelope, parseEnvelopeErr := parseSession.HandleShardSessionReceiveControlEnvelope()
	if parseEnvelopeErr != nil {
		parseT.Fatalf("HandleShardSessionReceiveControlEnvelope returned error: %v", parseEnvelopeErr)
	}
	if !hasEnvelope {
		parseT.Fatal("expected control envelope received from replacement port")
	}
	if parseEnvelope.Kind != ControlKindReady {
		parseT.Fatalf("expected replacement-port control kind %q, got %q", ControlKindReady, parseEnvelope.Kind)
	}
	if _, hasDuplicateEnvelope, parseDuplicateErr := parseSession.HandleShardSessionReceiveControlEnvelope(); parseDuplicateErr != nil {
		parseT.Fatalf("HandleShardSessionReceiveControlEnvelope(duplicate check) returned error: %v", parseDuplicateErr)
	} else if hasDuplicateEnvelope {
		parseT.Fatal("expected replacement-port ready message to deliver once without duplicate handler delivery")
	}
	parseOriginalPort.getOnMessage([]byte("late-original-inbound"))
	if _, hasLateOriginalPayload := parseSession.HandleShardSessionReceivePayload(); hasLateOriginalPayload {
		parseT.Fatal("expected original-port late inbound payload to be ignored after replacement")
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(parseCapabilitiesEnvelope); parseAcceptErr != nil {
		parseT.Fatalf("HandleShardSessionAcceptControlEnvelope(capabilities after replacement) returned error: %v", parseAcceptErr)
	}
	if parseSendErr := parseSession.HandleShardSessionSendMountControlEnvelope("dashboard.hot-panel", parseSnapshotEnvelope); parseSendErr != nil {
		parseT.Fatalf("HandleShardSessionSendMountControlEnvelope returned error after renegotiated replacement handshake: %v", parseSendErr)
	}
	if len(parseReplacementPort.storePostPayloads) != 1 {
		parseT.Fatalf("expected one posted payload on replacement port, got %d", len(parseReplacementPort.storePostPayloads))
	}
	if len(parseOriginalPort.storePostPayloads) != 1 {
		parseT.Fatalf("expected original port payload count to remain 1 after replacement sends, got %d", len(parseOriginalPort.storePostPayloads))
	}
}

// TestHandleShardSessionTeardownRejectsLateSendsAndClearsQueuedPayloads verifies session teardown unbinds inbound handlers, clears queued payloads, and rejects late sends.
func TestHandleShardSessionTeardownRejectsLateSendsAndClearsQueuedPayloads(parseT *testing.T) {
	parsePort := &fakeShardSessionPort{}
	parseSession, parseSessionErr := BuildShardSession("shard-a", parsePort)
	if parseSessionErr != nil {
		parseT.Fatalf("BuildShardSession returned error: %v", parseSessionErr)
	}
	parsePort.getOnMessage([]byte("queued-before-teardown"))
	if parseTeardownErr := parseSession.HandleShardSessionTeardown(); parseTeardownErr != nil {
		parseT.Fatalf("HandleShardSessionTeardown returned error: %v", parseTeardownErr)
	}
	if parseTeardownErr := parseSession.HandleShardSessionTeardown(); parseTeardownErr != nil {
		parseT.Fatalf("HandleShardSessionTeardown(second call) returned error: %v", parseTeardownErr)
	}
	if _, hasPayload := parseSession.HandleShardSessionReceivePayload(); hasPayload {
		parseT.Fatal("expected queued payloads to clear during teardown")
	}
	if parseSendErr := parseSession.HandleShardSessionSendPayload([]byte("late-send")); parseSendErr == nil {
		parseT.Fatal("expected late payload send to fail after teardown")
	}
	parsePort.getOnMessage([]byte("late-inbound"))
	if _, hasPayload := parseSession.HandleShardSessionReceivePayload(); hasPayload {
		parseT.Fatal("expected late inbound payload to be ignored after teardown")
	}
}

// TestHandleShardSessionSendAndReceivePatchReadyWithPayload verifies shard-session patch helpers send and receive one patch-ready control envelope with one paired raw patch payload.
func TestHandleShardSessionSendAndReceivePatchReadyWithPayload(parseT *testing.T) {
	parsePort := &loopbackShardSessionPort{}
	parseSession, parseSessionErr := BuildShardSession("shard-a", parsePort)
	if parseSessionErr != nil {
		parseT.Fatalf("BuildShardSession returned error: %v", parseSessionErr)
	}
	parsePatchReadyEnvelope, parsePatchReadyEnvelopeErr := BuildControlPatchReadyEnvelope(
		"region-1",
		2,
		2,
		TransportTierBinary,
	)
	if parsePatchReadyEnvelopeErr != nil {
		parseT.Fatalf("BuildControlPatchReadyEnvelope returned error: %v", parsePatchReadyEnvelopeErr)
	}
	parsePatchPayload := []byte(`{"patch":"payload"}`)
	if parseSendErr := parseSession.HandleShardSessionSendPatchReadyWithPayload(parsePatchReadyEnvelope, parsePatchPayload); parseSendErr != nil {
		parseT.Fatalf("HandleShardSessionSendPatchReadyWithPayload returned error: %v", parseSendErr)
	}
	parseReceivedEnvelope, parseReceivedPayload, hasReceivedEnvelope, parseReceiveErr := parseSession.HandleShardSessionReceivePatchReadyWithPayload()
	if parseReceiveErr != nil {
		parseT.Fatalf("HandleShardSessionReceivePatchReadyWithPayload returned error: %v", parseReceiveErr)
	}
	if !hasReceivedEnvelope {
		parseT.Fatal("expected one received patch-ready envelope")
	}
	if parseReceivedEnvelope.Kind != ControlKindPatchReady {
		parseT.Fatalf("expected received kind %q, got %q", ControlKindPatchReady, parseReceivedEnvelope.Kind)
	}
	if !bytes.Equal(parseReceivedPayload, parsePatchPayload) {
		parseT.Fatalf("expected received payload %q, got %q", string(parsePatchPayload), string(parseReceivedPayload))
	}
}

// TestHandleShardSessionSendPatchReadyWithPayloadRejectsNonPatchReadyEnvelope verifies patch-helper send rejects non patch-ready control envelopes.
func TestHandleShardSessionSendPatchReadyWithPayloadRejectsNonPatchReadyEnvelope(parseT *testing.T) {
	parsePort := &loopbackShardSessionPort{}
	parseSession, parseSessionErr := BuildShardSession("shard-a", parsePort)
	if parseSessionErr != nil {
		parseT.Fatalf("BuildShardSession returned error: %v", parseSessionErr)
	}
	if parseSendErr := parseSession.HandleShardSessionSendPatchReadyWithPayload(BuildControlReadyEnvelope(), []byte("payload")); parseSendErr == nil {
		parseT.Fatal("expected non patch-ready control envelope send to fail")
	}
}
