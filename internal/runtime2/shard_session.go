package runtime2

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
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
	storeReceivedPayloads  [][]byte
	storePendingPatchReady ControlEnvelope
	getQueueLimit          int
	getPendingPatchMisses  uint64
	getPortBindVersion     uint64
	getQueueDropWarnedAt   time.Time
	isShardSessionReady    bool
	hasShardSessionCaps    bool
	hasPendingPatchReady   bool
	hasLoggedStaleInbound  bool
	isShardSessionClosed   bool
	getSessionMutex        sync.Mutex
}

const getShardSessionDefaultQueueLimit = 256

var (
	matchShardSessionControlEnvelopeProtocolKey = []byte(`"protocol_version"`)
	matchShardSessionControlEnvelopeKindKey     = []byte(`"kind"`)
)

// resetShardSessionHandshake clears handshake readiness and capability flags.
func (parseSession *ShardSession) resetShardSessionHandshake() {
	if parseSession == nil {
		return
	}
	parseSession.isShardSessionReady = false
	parseSession.hasShardSessionCaps = false
}

// parseHasShardSessionControlEnvelopeMarkers reports whether one payload appears to be a control-envelope JSON object.
func parseHasShardSessionControlEnvelopeMarkers(parsePayload []byte) bool {
	parseTrimmedPayload := bytes.TrimLeft(parsePayload, " \t\r\n")
	if len(parseTrimmedPayload) == 0 {
		return false
	}
	if parseTrimmedPayload[0] != '{' {
		return false
	}
	if !bytes.Contains(parseTrimmedPayload, matchShardSessionControlEnvelopeProtocolKey) {
		return false
	}
	if !bytes.Contains(parseTrimmedPayload, matchShardSessionControlEnvelopeKindKey) {
		return false
	}
	return true
}

// bindShardSessionPortHandler binds one inbound message handler to the currently active session port.
func (parseSession *ShardSession) bindShardSessionPortHandler() {
	if parseSession == nil {
		return
	}
	parseSession.getSessionMutex.Lock()
	if parseSession.getSessionPort == nil {
		parseSession.getSessionMutex.Unlock()
		return
	}
	parseSession.getPortBindVersion++
	getPortBindVersion := parseSession.getPortBindVersion
	parseSession.hasLoggedStaleInbound = false
	getSessionPort := parseSession.getSessionPort
	parseSession.getSessionMutex.Unlock()
	getSessionPort.BindMessageHandler(func(parsePayload []byte) {
		parseSession.getSessionMutex.Lock()
		defer parseSession.getSessionMutex.Unlock()
		if parseSession.isShardSessionClosed {
			return
		}
		if getPortBindVersion != parseSession.getPortBindVersion {
			if !parseSession.hasLoggedStaleInbound {
				log.Printf("runtime2: warn shard session ignored stale inbound payload on shard=%q", parseSession.getShardID)
				parseSession.hasLoggedStaleInbound = true
			}
			return
		}
		buildPayloadCopy := append([]byte(nil), parsePayload...)
		if parseSession.getQueueLimit > 0 && len(parseSession.storeReceivedPayloads) >= parseSession.getQueueLimit {
			getNow := time.Now()
			if parseSession.getQueueDropWarnedAt.IsZero() || getNow.Sub(parseSession.getQueueDropWarnedAt) >= time.Second {
				log.Printf(
					"runtime2: warn shard session inbound queue reached limit=%d on shard=%q; dropping oldest payload (queue_depth=%d)",
					parseSession.getQueueLimit,
					parseSession.getShardID,
					len(parseSession.storeReceivedPayloads),
				)
				parseSession.getQueueDropWarnedAt = getNow
			}
			copy(parseSession.storeReceivedPayloads, parseSession.storeReceivedPayloads[1:])
			parseSession.storeReceivedPayloads[len(parseSession.storeReceivedPayloads)-1] = buildPayloadCopy
			return
		}
		parseSession.storeReceivedPayloads = append(parseSession.storeReceivedPayloads, buildPayloadCopy)
	})
}

// unbindShardSessionPortHandler detaches the currently bound inbound callback from the active session port.
func (parseSession *ShardSession) unbindShardSessionPortHandler() {
	if parseSession == nil {
		return
	}
	parseSession.getSessionMutex.Lock()
	if parseSession.getSessionPort == nil {
		parseSession.getSessionMutex.Unlock()
		return
	}
	parseSession.getPortBindVersion++
	getSessionPort := parseSession.getSessionPort
	parseSession.getSessionMutex.Unlock()
	getSessionPort.BindMessageHandler(func(parsePayload []byte) {})
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
		getShardID:            parseShardID,
		getSessionPort:        parseSessionPort,
		storeReceivedPayloads: make([][]byte, 0),
		getQueueLimit:         parseQueueLimit,
	}
	buildSession.bindShardSessionPortHandler()
	return buildSession, nil
}

// GetShardSessionShardID reports the shard identity owned by this session.
func (parseSession *ShardSession) GetShardSessionShardID() SchedulerShardID {
	if parseSession == nil {
		return ""
	}
	parseSession.getSessionMutex.Lock()
	defer parseSession.getSessionMutex.Unlock()
	return parseSession.getShardID
}

// HandleShardSessionSendPayload posts one payload through the backing session port.
func (parseSession *ShardSession) HandleShardSessionSendPayload(parsePayload []byte) error {
	if parseSession == nil {
		return fmt.Errorf("runtime2: shard session is nil")
	}
	parseSession.getSessionMutex.Lock()
	if parseSession.isShardSessionClosed {
		parseSession.getSessionMutex.Unlock()
		return fmt.Errorf("runtime2: shard session is closed")
	}
	if len(parsePayload) == 0 {
		parseSession.getSessionMutex.Unlock()
		return fmt.Errorf("runtime2: shard session payload is required")
	}
	if parseSession.getSessionPort == nil {
		parseSession.getSessionMutex.Unlock()
		return fmt.Errorf("runtime2: shard session port is required")
	}
	getSessionPort := parseSession.getSessionPort
	parseSession.getSessionMutex.Unlock()
	return getSessionPort.PostMessage(parsePayload)
}

// HandleShardSessionReceivePayload drains one queued inbound payload from the session port handler.
func (parseSession *ShardSession) HandleShardSessionReceivePayload() ([]byte, bool) {
	if parseSession == nil {
		return nil, false
	}
	parseSession.getSessionMutex.Lock()
	defer parseSession.getSessionMutex.Unlock()
	if parseSession.isShardSessionClosed || len(parseSession.storeReceivedPayloads) == 0 {
		return nil, false
	}
	getPayload := parseSession.storeReceivedPayloads[0]
	parseSession.storeReceivedPayloads[0] = nil
	parseSession.storeReceivedPayloads = parseSession.storeReceivedPayloads[1:]
	if len(parseSession.storeReceivedPayloads) == 0 {
		parseSession.storeReceivedPayloads = nil
	}
	return getPayload, true
}

// HasShardSessionHandshakeComplete reports whether ready and capabilities handshake messages were both accepted.
func (parseSession *ShardSession) HasShardSessionHandshakeComplete() bool {
	if parseSession == nil {
		return false
	}
	parseSession.getSessionMutex.Lock()
	defer parseSession.getSessionMutex.Unlock()
	return parseSession.isShardSessionReady && parseSession.hasShardSessionCaps
}

// HandleShardSessionReplacePort swaps the backing session port and resets handshake state for one renegotiation pass.
func (parseSession *ShardSession) HandleShardSessionReplacePort(parseSessionPort ShardSessionPort) error {
	if parseSession == nil {
		return fmt.Errorf("runtime2: shard session is nil")
	}
	parseSession.getSessionMutex.Lock()
	if parseSession.isShardSessionClosed {
		parseSession.getSessionMutex.Unlock()
		return fmt.Errorf("runtime2: shard session is closed")
	}
	if parseSessionPort == nil {
		parseSession.getSessionMutex.Unlock()
		return fmt.Errorf("runtime2: shard session replacement port is required")
	}
	getOriginalSessionPort := parseSession.getSessionPort
	parseSession.getSessionPort = parseSessionPort
	parseSession.storeReceivedPayloads = nil
	parseSession.storePendingPatchReady = ControlEnvelope{}
	parseSession.hasPendingPatchReady = false
	parseSession.getPendingPatchMisses = 0
	parseSession.getQueueDropWarnedAt = time.Time{}
	parseSession.hasLoggedStaleInbound = false
	parseSession.resetShardSessionHandshake()
	parseSession.getPortBindVersion++
	parseSession.getSessionMutex.Unlock()
	if getOriginalSessionPort != nil {
		getOriginalSessionPort.BindMessageHandler(func(parsePayload []byte) {})
	}
	parseSession.bindShardSessionPortHandler()
	return nil
}

// HandleShardSessionTeardown closes one shard session, unbinds inbound handlers, and clears queued state.
func (parseSession *ShardSession) HandleShardSessionTeardown() error {
	if parseSession == nil {
		return fmt.Errorf("runtime2: shard session is nil")
	}
	parseSession.getSessionMutex.Lock()
	if parseSession.isShardSessionClosed {
		parseSession.getSessionMutex.Unlock()
		return nil
	}
	getSessionPort := parseSession.getSessionPort
	parseSession.storeReceivedPayloads = nil
	parseSession.storePendingPatchReady = ControlEnvelope{}
	parseSession.hasPendingPatchReady = false
	parseSession.getPendingPatchMisses = 0
	parseSession.getQueueDropWarnedAt = time.Time{}
	parseSession.hasLoggedStaleInbound = false
	parseSession.resetShardSessionHandshake()
	parseSession.isShardSessionClosed = true
	parseSession.getPortBindVersion++
	parseSession.getSessionMutex.Unlock()
	if getSessionPort != nil {
		getSessionPort.BindMessageHandler(func(parsePayload []byte) {})
	}
	return nil
}

// HandleShardSessionAcceptControlEnvelope validates one control envelope and enforces ready/capabilities handshake before mount/update acceptance.
func (parseSession *ShardSession) HandleShardSessionAcceptControlEnvelope(parseEnvelope ControlEnvelope) error {
	if parseSession == nil {
		return fmt.Errorf("runtime2: shard session is nil")
	}
	parseSession.getSessionMutex.Lock()
	defer parseSession.getSessionMutex.Unlock()
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
		if !(parseSession.isShardSessionReady && parseSession.hasShardSessionCaps) {
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
	parseSession.getSessionMutex.Lock()
	parseSession.resetShardSessionHandshake()
	parseSession.getSessionMutex.Unlock()
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
	if parseSendPayloadErr := parseSession.HandleShardSessionSendPatchPayload(parsePatchPayload); parseSendPayloadErr != nil {
		log.Printf(
			"runtime2: error shard session patch-ready payload send failed for region=%q patch_version=%d: %v",
			parsePatchReadyEnvelope.RegionInstanceID,
			parsePatchReadyEnvelope.PatchVersion,
			parseSendPayloadErr,
		)
		return fmt.Errorf(
			"runtime2: patch-ready payload send failed for region=%q patch_version=%d: %w",
			parsePatchReadyEnvelope.RegionInstanceID,
			parsePatchReadyEnvelope.PatchVersion,
			parseSendPayloadErr,
		)
	}
	return nil
}

// HandleShardSessionReceivePatchReadyWithPayload drains one patch-ready control envelope and one paired raw patch payload from this shard session.
func (parseSession *ShardSession) HandleShardSessionReceivePatchReadyWithPayload() (ControlEnvelope, []byte, bool, error) {
	if parseSession == nil {
		return ControlEnvelope{}, nil, false, fmt.Errorf("runtime2: shard session is nil")
	}
	parseSession.getSessionMutex.Lock()
	if parseSession.hasPendingPatchReady {
		if len(parseSession.storeReceivedPayloads) > 0 {
			getNextPayload := parseSession.storeReceivedPayloads[0]
			getPatchReadyEnvelope := parseSession.storePendingPatchReady
			if parseHasShardSessionControlEnvelopeMarkers(getNextPayload) {
				if getUnexpectedControlEnvelope, getUnexpectedControlEnvelopeErr := ParseControlEnvelopeJSON(getNextPayload); getUnexpectedControlEnvelopeErr == nil {
					parseSession.storePendingPatchReady = ControlEnvelope{}
					parseSession.hasPendingPatchReady = false
					parseSession.getPendingPatchMisses = 0
					parseSession.getSessionMutex.Unlock()
					log.Printf(
						"runtime2: error shard session missing patch payload for region=%q patch_version=%d before next control envelope kind=%q",
						getPatchReadyEnvelope.RegionInstanceID,
						getPatchReadyEnvelope.PatchVersion,
						getUnexpectedControlEnvelope.Kind,
					)
					return ControlEnvelope{}, nil, false, fmt.Errorf(
						"runtime2: patch-ready payload is missing for region=%q patch_version=%d before next control envelope kind=%q",
						getPatchReadyEnvelope.RegionInstanceID,
						getPatchReadyEnvelope.PatchVersion,
						getUnexpectedControlEnvelope.Kind,
					)
				}
			}
			getPatchPayload := getNextPayload
			parseSession.storeReceivedPayloads[0] = nil
			parseSession.storeReceivedPayloads = parseSession.storeReceivedPayloads[1:]
			if len(parseSession.storeReceivedPayloads) == 0 {
				parseSession.storeReceivedPayloads = nil
			}
			parseSession.storePendingPatchReady = ControlEnvelope{}
			parseSession.hasPendingPatchReady = false
			parseSession.getPendingPatchMisses = 0
			parseSession.getSessionMutex.Unlock()
			return getPatchReadyEnvelope, getPatchPayload, true, nil
		}
		parseSession.getPendingPatchMisses++
		getPendingEnvelope := parseSession.storePendingPatchReady
		getPendingMisses := parseSession.getPendingPatchMisses
		parseSession.getSessionMutex.Unlock()
		if getPendingMisses == 1 || getPendingMisses%128 == 0 {
			log.Printf(
				"runtime2: warn shard session waiting for patch-ready payload for region=%q patch_version=%d (polls=%d)",
				getPendingEnvelope.RegionInstanceID,
				getPendingEnvelope.PatchVersion,
				getPendingMisses,
			)
		}
		return ControlEnvelope{}, nil, false, nil
	}
	parseSession.getSessionMutex.Unlock()
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
	parseSession.getSessionMutex.Lock()
	if len(parseSession.storeReceivedPayloads) == 0 {
		parseSession.storePendingPatchReady = parseEnvelope
		parseSession.hasPendingPatchReady = true
		parseSession.getPendingPatchMisses = 0
		parseSession.getSessionMutex.Unlock()
		log.Printf(
			"runtime2: warn shard session received patch-ready envelope before payload for region=%q patch_version=%d",
			parseEnvelope.RegionInstanceID,
			parseEnvelope.PatchVersion,
		)
		return ControlEnvelope{}, nil, false, nil
	}
	parsePayload := parseSession.storeReceivedPayloads[0]
	parseSession.storeReceivedPayloads[0] = nil
	parseSession.storeReceivedPayloads = parseSession.storeReceivedPayloads[1:]
	if len(parseSession.storeReceivedPayloads) == 0 {
		parseSession.storeReceivedPayloads = nil
	}
	parseSession.getSessionMutex.Unlock()
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
