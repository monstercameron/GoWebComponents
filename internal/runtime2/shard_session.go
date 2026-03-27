package runtime2

import (
	"fmt"
	"strings"
)

// ShardSessionPort abstracts one interop worker or MessagePort-like primitive for shard session traffic.
type ShardSessionPort interface {
	PostMessage(parsePayload []byte) error
	BindMessageHandler(parseHandler func(parsePayload []byte))
}

// ShardSession stores one shard-scoped control and payload channel backed by an interop or MessagePort primitive.
type ShardSession struct {
	getShardID             SchedulerShardID
	getSessionPort         ShardSessionPort
	storeReceivedPayloads [][]byte
	getQueueLimit          int
	isShardSessionReady    bool
	hasShardSessionCaps    bool
	isShardSessionClosed   bool
}

const getShardSessionDefaultQueueLimit = 256

// resetShardSessionHandshake clears handshake readiness and capability flags.
func (parseSession *ShardSession) resetShardSessionHandshake() {
	if parseSession == nil {
		return
	}
	parseSession.isShardSessionReady = false
	parseSession.hasShardSessionCaps = false
}

// bindShardSessionPortHandler binds one inbound message handler to the currently active session port.
func (parseSession *ShardSession) bindShardSessionPortHandler() {
	if parseSession == nil || parseSession.getSessionPort == nil {
		return
	}
	parseSession.getSessionPort.BindMessageHandler(func(parsePayload []byte) {
		if parseSession.isShardSessionClosed {
			return
		}
		buildPayloadCopy := append([]byte(nil), parsePayload...)
		if parseSession.getQueueLimit > 0 && len(parseSession.storeReceivedPayloads) >= parseSession.getQueueLimit {
			parseSession.storeReceivedPayloads = append(parseSession.storeReceivedPayloads[1:], buildPayloadCopy)
			return
		}
		parseSession.storeReceivedPayloads = append(parseSession.storeReceivedPayloads, buildPayloadCopy)
	})
}

// unbindShardSessionPortHandler detaches the currently bound inbound callback from the active session port.
func (parseSession *ShardSession) unbindShardSessionPortHandler() {
	if parseSession == nil || parseSession.getSessionPort == nil {
		return
	}
	parseSession.getSessionPort.BindMessageHandler(func(parsePayload []byte) {})
}

// BuildShardSession creates one shard session backed by one interop or MessagePort-like primitive.
func BuildShardSession(parseShardID SchedulerShardID, parseSessionPort ShardSessionPort) (*ShardSession, error) {
	return BuildShardSessionWithQueueLimit(parseShardID, parseSessionPort, getShardSessionDefaultQueueLimit)
}

// BuildShardSessionWithQueueLimit creates one shard session with one explicit inbound queue bound.
func BuildShardSessionWithQueueLimit(parseShardID SchedulerShardID, parseSessionPort ShardSessionPort, parseQueueLimit int) (*ShardSession, error) {
	if strings.TrimSpace(string(parseShardID)) == "" {
		return nil, fmt.Errorf("runtime2: shard session shard ID is required")
	}
	if parseSessionPort == nil {
		return nil, fmt.Errorf("runtime2: shard session port is required")
	}
	if parseQueueLimit <= 0 {
		return nil, fmt.Errorf("runtime2: shard session queue limit must be greater than zero")
	}
	buildSession := &ShardSession{
		getShardID:             parseShardID,
		getSessionPort:         parseSessionPort,
		storeReceivedPayloads: make([][]byte, 0),
		getQueueLimit:          parseQueueLimit,
	}
	buildSession.bindShardSessionPortHandler()
	return buildSession, nil
}

// GetShardSessionShardID reports the shard identity owned by this session.
func (parseSession *ShardSession) GetShardSessionShardID() SchedulerShardID {
	if parseSession == nil {
		return ""
	}
	return parseSession.getShardID
}

// HandleShardSessionSendPayload posts one payload through the backing session port.
func (parseSession *ShardSession) HandleShardSessionSendPayload(parsePayload []byte) error {
	if parseSession == nil {
		return fmt.Errorf("runtime2: shard session is nil")
	}
	if parseSession.isShardSessionClosed {
		return fmt.Errorf("runtime2: shard session is closed")
	}
	if len(parsePayload) == 0 {
		return fmt.Errorf("runtime2: shard session payload is required")
	}
	buildPayloadCopy := append([]byte(nil), parsePayload...)
	return parseSession.getSessionPort.PostMessage(buildPayloadCopy)
}

// HandleShardSessionReceivePayload drains one queued inbound payload from the session port handler.
func (parseSession *ShardSession) HandleShardSessionReceivePayload() ([]byte, bool) {
	if parseSession != nil && parseSession.isShardSessionClosed {
		return nil, false
	}
	if parseSession == nil || len(parseSession.storeReceivedPayloads) == 0 {
		return nil, false
	}
	getPayload := parseSession.storeReceivedPayloads[0]
	parseSession.storeReceivedPayloads = parseSession.storeReceivedPayloads[1:]
	return getPayload, true
}

// HasShardSessionHandshakeComplete reports whether ready and capabilities handshake messages were both accepted.
func (parseSession *ShardSession) HasShardSessionHandshakeComplete() bool {
	if parseSession == nil {
		return false
	}
	return parseSession.isShardSessionReady && parseSession.hasShardSessionCaps
}

// HandleShardSessionReplacePort swaps the backing session port and resets handshake state for one renegotiation pass.
func (parseSession *ShardSession) HandleShardSessionReplacePort(parseSessionPort ShardSessionPort) error {
	if parseSession == nil {
		return fmt.Errorf("runtime2: shard session is nil")
	}
	if parseSession.isShardSessionClosed {
		return fmt.Errorf("runtime2: shard session is closed")
	}
	if parseSessionPort == nil {
		return fmt.Errorf("runtime2: shard session replacement port is required")
	}
	parseSession.unbindShardSessionPortHandler()
	parseSession.getSessionPort = parseSessionPort
	parseSession.storeReceivedPayloads = make([][]byte, 0)
	parseSession.resetShardSessionHandshake()
	parseSession.bindShardSessionPortHandler()
	return nil
}

// HandleShardSessionTeardown closes one shard session, unbinds inbound handlers, and clears queued state.
func (parseSession *ShardSession) HandleShardSessionTeardown() error {
	if parseSession == nil {
		return fmt.Errorf("runtime2: shard session is nil")
	}
	if parseSession.isShardSessionClosed {
		return nil
	}
	parseSession.unbindShardSessionPortHandler()
	parseSession.storeReceivedPayloads = make([][]byte, 0)
	parseSession.resetShardSessionHandshake()
	parseSession.isShardSessionClosed = true
	return nil
}

// HandleShardSessionAcceptControlEnvelope validates one control envelope and enforces ready/capabilities handshake before mount/update acceptance.
func (parseSession *ShardSession) HandleShardSessionAcceptControlEnvelope(parseEnvelope ControlEnvelope) error {
	if parseSession == nil {
		return fmt.Errorf("runtime2: shard session is nil")
	}
	if parseSession.isShardSessionClosed {
		return fmt.Errorf("runtime2: shard session is closed")
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		return parseErr
	}
	switch parseEnvelope.Kind {
	case ControlKindReady:
		parseSession.isShardSessionReady = true
		return nil
	case ControlKindCapabilities:
		parseSession.hasShardSessionCaps = true
		return nil
	case ControlKindRestart:
		parseSession.resetShardSessionHandshake()
		return nil
	case ControlKindMount, ControlKindUpdate:
		if !parseSession.HasShardSessionHandshakeComplete() {
			return fmt.Errorf("runtime2: shard session handshake must complete before %q control traffic is accepted", parseEnvelope.Kind)
		}
		return nil
	default:
		return nil
	}
}

// HandleShardSessionSendControlEnvelope serializes and posts one control envelope through this shard session.
func (parseSession *ShardSession) HandleShardSessionSendControlEnvelope(parseEnvelope ControlEnvelope) error {
	if parseSession == nil {
		return fmt.Errorf("runtime2: shard session is nil")
	}
	if parseEnvelope.Kind == ControlKindMount || parseEnvelope.Kind == ControlKindUpdate {
		if !parseSession.HasShardSessionHandshakeComplete() {
			return fmt.Errorf("runtime2: shard session handshake must complete before %q control traffic is sent", parseEnvelope.Kind)
		}
	}
	parsePayload, parsePayloadErr := BuildControlEnvelopeJSON(parseEnvelope)
	if parsePayloadErr != nil {
		return parsePayloadErr
	}
	return parseSession.HandleShardSessionSendPayload(parsePayload)
}

// HandleShardSessionSendMountControlEnvelope builds and sends one mount control envelope.
func (parseSession *ShardSession) HandleShardSessionSendMountControlEnvelope(parseRendererID RendererID, parseSnapshot SnapshotEnvelope) error {
	parseEnvelope, parseEnvelopeErr := BuildControlMountEnvelope(parseRendererID, parseSnapshot)
	if parseEnvelopeErr != nil {
		return parseEnvelopeErr
	}
	return parseSession.HandleShardSessionSendControlEnvelope(parseEnvelope)
}

// HandleShardSessionSendUpdateControlEnvelope builds and sends one update control envelope.
func (parseSession *ShardSession) HandleShardSessionSendUpdateControlEnvelope(parseSnapshot SnapshotEnvelope) error {
	parseEnvelope, parseEnvelopeErr := BuildControlUpdateEnvelope(parseSnapshot)
	if parseEnvelopeErr != nil {
		return parseEnvelopeErr
	}
	return parseSession.HandleShardSessionSendControlEnvelope(parseEnvelope)
}

// HandleShardSessionSendCancelControlEnvelope builds and sends one cancel control envelope.
func (parseSession *ShardSession) HandleShardSessionSendCancelControlEnvelope(parseRegionInstanceID RegionInstanceID) error {
	parseEnvelope, parseEnvelopeErr := BuildControlCancelEnvelope(parseRegionInstanceID)
	if parseEnvelopeErr != nil {
		return parseEnvelopeErr
	}
	return parseSession.HandleShardSessionSendControlEnvelope(parseEnvelope)
}

// HandleShardSessionSendDisposeControlEnvelope builds and sends one dispose control envelope.
func (parseSession *ShardSession) HandleShardSessionSendDisposeControlEnvelope(parseRegionInstanceID RegionInstanceID) error {
	parseEnvelope, parseEnvelopeErr := BuildControlDisposeEnvelope(parseRegionInstanceID)
	if parseEnvelopeErr != nil {
		return parseEnvelopeErr
	}
	return parseSession.HandleShardSessionSendControlEnvelope(parseEnvelope)
}

// HandleShardSessionSendRestartControlEnvelope builds and sends one restart control envelope.
func (parseSession *ShardSession) HandleShardSessionSendRestartControlEnvelope(parseRegionInstanceID RegionInstanceID, parseEpoch uint64) error {
	parseEnvelope, parseEnvelopeErr := BuildControlRestartEnvelope(parseRegionInstanceID, parseEpoch)
	if parseEnvelopeErr != nil {
		return parseEnvelopeErr
	}
	if parseSendErr := parseSession.HandleShardSessionSendControlEnvelope(parseEnvelope); parseSendErr != nil {
		return parseSendErr
	}
	parseSession.resetShardSessionHandshake()
	return nil
}

// HandleShardSessionReceiveControlEnvelope drains one inbound payload, parses one control envelope, and applies handshake acceptance rules.
func (parseSession *ShardSession) HandleShardSessionReceiveControlEnvelope() (ControlEnvelope, bool, error) {
	parsePayload, hasPayload := parseSession.HandleShardSessionReceivePayload()
	if !hasPayload {
		return ControlEnvelope{}, false, nil
	}
	parseEnvelope, parseEnvelopeErr := ParseControlEnvelopeJSON(parsePayload)
	if parseEnvelopeErr != nil {
		return ControlEnvelope{}, true, parseEnvelopeErr
	}
	if parseAcceptErr := parseSession.HandleShardSessionAcceptControlEnvelope(parseEnvelope); parseAcceptErr != nil {
		return ControlEnvelope{}, true, parseAcceptErr
	}
	return parseEnvelope, true, nil
}

// HandleShardSessionSendPatchPayload posts one raw patch payload through this shard session.
func (parseSession *ShardSession) HandleShardSessionSendPatchPayload(parsePayload []byte) error {
	return parseSession.HandleShardSessionSendPayload(parsePayload)
}

// HandleShardSessionReceivePatchPayload drains one raw patch payload from this shard session.
func (parseSession *ShardSession) HandleShardSessionReceivePatchPayload() ([]byte, bool) {
	return parseSession.HandleShardSessionReceivePayload()
}

// HandleShardSessionSendPatchReadyWithPayload sends one patch-ready control envelope and one paired raw patch payload through this shard session.
func (parseSession *ShardSession) HandleShardSessionSendPatchReadyWithPayload(parsePatchReadyEnvelope ControlEnvelope, parsePatchPayload []byte) error {
	if parsePatchReadyEnvelope.Kind != ControlKindPatchReady {
		return fmt.Errorf("runtime2: patch-ready control envelope is required")
	}
	if parseSendEnvelopeErr := parseSession.HandleShardSessionSendControlEnvelope(parsePatchReadyEnvelope); parseSendEnvelopeErr != nil {
		return parseSendEnvelopeErr
	}
	return parseSession.HandleShardSessionSendPatchPayload(parsePatchPayload)
}

// HandleShardSessionReceivePatchReadyWithPayload drains one patch-ready control envelope and one paired raw patch payload from this shard session.
func (parseSession *ShardSession) HandleShardSessionReceivePatchReadyWithPayload() (ControlEnvelope, []byte, bool, error) {
	parseEnvelope, hasEnvelope, parseEnvelopeErr := parseSession.HandleShardSessionReceiveControlEnvelope()
	if parseEnvelopeErr != nil {
		return ControlEnvelope{}, nil, hasEnvelope, parseEnvelopeErr
	}
	if !hasEnvelope {
		return ControlEnvelope{}, nil, false, nil
	}
	if parseEnvelope.Kind != ControlKindPatchReady {
		return ControlEnvelope{}, nil, true, fmt.Errorf("runtime2: expected patch-ready control envelope, got %q", parseEnvelope.Kind)
	}
	parsePayload, hasPayload := parseSession.HandleShardSessionReceivePatchPayload()
	if !hasPayload {
		return ControlEnvelope{}, nil, true, fmt.Errorf("runtime2: patch-ready payload is missing")
	}
	return parseEnvelope, parsePayload, true, nil
}

// HandleShardSessionSendWorkerPatchReadyControlEnvelope builds and sends one worker patch-ready control envelope.
func (parseSession *ShardSession) HandleShardSessionSendWorkerPatchReadyControlEnvelope(
	parseRegionInstanceID RegionInstanceID,
	parsePatchVersion uint64,
	parseInputVersion uint64,
	parseTransportTier TransportTier,
) error {
	parseEnvelope, parseEnvelopeErr := BuildControlPatchReadyEnvelope(parseRegionInstanceID, parsePatchVersion, parseInputVersion, parseTransportTier)
	if parseEnvelopeErr != nil {
		return parseEnvelopeErr
	}
	return parseSession.HandleShardSessionSendControlEnvelope(parseEnvelope)
}

// HandleShardSessionSendWorkerDiagnosticControlEnvelope builds and sends one worker diagnostic control envelope.
func (parseSession *ShardSession) HandleShardSessionSendWorkerDiagnosticControlEnvelope(
	parseRegionInstanceID RegionInstanceID,
	parseDiagnosticSpec ControlDiagnosticEnvelopeSpec,
) error {
	parseEnvelope, parseEnvelopeErr := BuildControlDiagnosticEnvelope(parseRegionInstanceID, parseDiagnosticSpec)
	if parseEnvelopeErr != nil {
		return parseEnvelopeErr
	}
	return parseSession.HandleShardSessionSendControlEnvelope(parseEnvelope)
}

// HandleShardSessionSendWorkerRestartControlEnvelope builds and sends one worker restart control envelope.
func (parseSession *ShardSession) HandleShardSessionSendWorkerRestartControlEnvelope(parseRegionInstanceID RegionInstanceID, parseEpoch uint64) error {
	return parseSession.HandleShardSessionSendRestartControlEnvelope(parseRegionInstanceID, parseEpoch)
}

// HandleShardSessionSendWorkerReadyControlEnvelope builds and sends one worker ready control envelope.
func (parseSession *ShardSession) HandleShardSessionSendWorkerReadyControlEnvelope() error {
	return parseSession.HandleShardSessionSendControlEnvelope(BuildControlReadyEnvelope())
}

// HandleShardSessionSendWorkerCapabilitiesControlEnvelope builds and sends one worker capabilities control envelope.
func (parseSession *ShardSession) HandleShardSessionSendWorkerCapabilitiesControlEnvelope(parseCapabilityReport CapabilityReport) error {
	parseEnvelope, parseEnvelopeErr := BuildControlCapabilitiesEnvelope(parseCapabilityReport)
	if parseEnvelopeErr != nil {
		return parseEnvelopeErr
	}
	return parseSession.HandleShardSessionSendControlEnvelope(parseEnvelope)
}

// HandleShardSessionSendWorkerPongControlEnvelope builds and sends one worker pong control envelope.
func (parseSession *ShardSession) HandleShardSessionSendWorkerPongControlEnvelope(parseShardID SchedulerShardID, parseSequence uint64) error {
	parseEnvelope, parseEnvelopeErr := BuildControlPongEnvelope(parseShardID, parseSequence)
	if parseEnvelopeErr != nil {
		return parseEnvelopeErr
	}
	return parseSession.HandleShardSessionSendControlEnvelope(parseEnvelope)
}
