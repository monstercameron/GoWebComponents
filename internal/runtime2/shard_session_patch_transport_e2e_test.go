package runtime2

import "testing"

type loopbackShardSessionPort struct {
	storePortPostedPayloads [][]byte
	getPortMessageHandler   func(parsePayload []byte)
}

// PostMessage stores one outbound payload and immediately loops it back as inbound traffic.
func (parsePort *loopbackShardSessionPort) PostMessage(parsePayload []byte) error {
	buildPayloadCopy := append([]byte(nil), parsePayload...)
	parsePort.storePortPostedPayloads = append(parsePort.storePortPostedPayloads, buildPayloadCopy)
	if parsePort.getPortMessageHandler != nil {
		parsePort.getPortMessageHandler(buildPayloadCopy)
	}
	return nil
}

// BindMessageHandler stores one inbound callback for session loopback tests.
func (parsePort *loopbackShardSessionPort) BindMessageHandler(parseHandler func(parsePayload []byte)) {
	parsePort.getPortMessageHandler = parseHandler
}

// buildWorkerRuntimeForShardSessionPatchTransportTest creates one mounted worker runtime that emits patch-ready output on changed input versions.
func buildWorkerRuntimeForShardSessionPatchTransportTest(parseT *testing.T) *WorkerRegionRuntime {
	parseT.Helper()
	buildRuntime := BuildWorkerRegionRuntime()
	if parseRegisterErr := buildRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
		return map[string]any{
			"version": parseMount.InputVersion,
		}, nil
	}); parseRegisterErr != nil {
		parseT.Fatalf("RegisterWorkerRegionRenderer returned error: %v", parseRegisterErr)
	}
	if _, parseMountErr := buildRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
		RegionID:     "region-1",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 1,
	}); parseMountErr != nil {
		parseT.Fatalf("HandleWorkerRegionMount returned error: %v", parseMountErr)
	}
	return buildRuntime
}

// handleShardSessionPatchTransportRoundTrip sends one patch-ready envelope and payload over one shard session and returns the host-decoded transport tier and patch stream.
func handleShardSessionPatchTransportRoundTrip(
	parseT *testing.T,
	parsePatchReadyEnvelope ControlEnvelope,
	parsePatchPayload []byte,
	parseSharedPatchPage *SharedPatchPage,
) (TransportTier, PatchStreamRaw) {
	parseT.Helper()
	parsePort := &loopbackShardSessionPort{}
	parseSession, parseSessionErr := BuildShardSession("shard-a", parsePort)
	if parseSessionErr != nil {
		parseT.Fatalf("BuildShardSession returned error: %v", parseSessionErr)
	}
	if parseSendErr := parseSession.HandleShardSessionSendControlEnvelope(parsePatchReadyEnvelope); parseSendErr != nil {
		parseT.Fatalf("HandleShardSessionSendControlEnvelope returned error: %v", parseSendErr)
	}
	if parseSendErr := parseSession.HandleShardSessionSendPayload(parsePatchPayload); parseSendErr != nil {
		parseT.Fatalf("HandleShardSessionSendPayload returned error: %v", parseSendErr)
	}
	parseReceivedEnvelope, hasReceivedEnvelope, parseReceivedEnvelopeErr := parseSession.HandleShardSessionReceiveControlEnvelope()
	if parseReceivedEnvelopeErr != nil {
		parseT.Fatalf("HandleShardSessionReceiveControlEnvelope returned error: %v", parseReceivedEnvelopeErr)
	}
	if !hasReceivedEnvelope {
		parseT.Fatal("expected one looped-back patch-ready control envelope")
	}
	parseReceivedPayload, hasReceivedPayload := parseSession.HandleShardSessionReceivePayload()
	if !hasReceivedPayload {
		parseT.Fatal("expected one looped-back patch payload")
	}
	parseTransportTier, parsePatchStream, parsePatchStreamErr := ParseHostPatchPayloadWithFallback(
		parseReceivedEnvelope,
		parseReceivedPayload,
		parseSharedPatchPage,
	)
	if parsePatchStreamErr != nil {
		parseT.Fatalf("ParseHostPatchPayloadWithFallback returned error: %v", parsePatchStreamErr)
	}
	return parseTransportTier, parsePatchStream
}

// TestHandleShardSessionPatchTransportRoundTripStructuredClone verifies structured-clone patch-ready selection and host decode over one shard session.
func TestHandleShardSessionPatchTransportRoundTripStructuredClone(parseT *testing.T) {
	parseRuntime := buildWorkerRuntimeForShardSessionPatchTransportTest(parseT)
	parseResult, parseResultErr := parseRuntime.HandleWorkerRegionUpdateWithPatchTransport(
		WorkerRegionUpdateSpec{
			RegionID:     "region-1",
			Epoch:        1,
			InputVersion: 2,
		},
		BuildCapabilityReport(CapabilitySource{
			HasWorkerSupport:          true,
			HasMessagePortSupport:     true,
			HasStructuredCloneSupport: true,
		}),
	)
	if parseResultErr != nil {
		parseT.Fatalf("HandleWorkerRegionUpdateWithPatchTransport returned error: %v", parseResultErr)
	}
	if !parseResult.HasPatchReadyEnvelope || !parseResult.HasPatchPayload {
		parseT.Fatalf("expected patch-ready envelope and payload, got %+v", parseResult)
	}
	if parseResult.GetTransportTier != TransportTierStructuredClone {
		parseT.Fatalf("expected structured-clone patch transport, got %q", parseResult.GetTransportTier)
	}
	parseTransportTier, parsePatchStream := handleShardSessionPatchTransportRoundTrip(
		parseT,
		parseResult.GetPatchReadyEnvelope,
		parseResult.GetPatchPayload,
		nil,
	)
	if parseTransportTier != TransportTierStructuredClone {
		parseT.Fatalf("expected structured-clone host decode tier, got %q", parseTransportTier)
	}
	if parsePatchStream.GetHeader.RegionID != "region-1" || parsePatchStream.GetHeader.PatchVersion != 2 {
		parseT.Fatalf("unexpected decoded patch stream header: %+v", parsePatchStream.GetHeader)
	}
}

// TestHandleShardSessionPatchTransportRoundTripBinary verifies binary patch-ready selection and host decode over one shard session.
func TestHandleShardSessionPatchTransportRoundTripBinary(parseT *testing.T) {
	parseRuntime := buildWorkerRuntimeForShardSessionPatchTransportTest(parseT)
	parseResult, parseResultErr := parseRuntime.HandleWorkerRegionUpdateWithPatchTransport(
		WorkerRegionUpdateSpec{
			RegionID:     "region-1",
			Epoch:        1,
			InputVersion: 2,
		},
		BuildCapabilityReport(CapabilitySource{
			HasWorkerSupport:          true,
			HasMessagePortSupport:     true,
			HasStructuredCloneSupport: true,
			HasBinaryTransportSupport: true,
		}),
	)
	if parseResultErr != nil {
		parseT.Fatalf("HandleWorkerRegionUpdateWithPatchTransport returned error: %v", parseResultErr)
	}
	if !parseResult.HasPatchReadyEnvelope || !parseResult.HasPatchPayload {
		parseT.Fatalf("expected patch-ready envelope and payload, got %+v", parseResult)
	}
	if parseResult.GetTransportTier != TransportTierBinary {
		parseT.Fatalf("expected binary patch transport, got %q", parseResult.GetTransportTier)
	}
	parseTransportTier, parsePatchStream := handleShardSessionPatchTransportRoundTrip(
		parseT,
		parseResult.GetPatchReadyEnvelope,
		parseResult.GetPatchPayload,
		nil,
	)
	if parseTransportTier != TransportTierBinary {
		parseT.Fatalf("expected binary host decode tier, got %q", parseTransportTier)
	}
	if parsePatchStream.GetHeader.RegionID != "region-1" || parsePatchStream.GetHeader.PatchVersion != 2 {
		parseT.Fatalf("unexpected decoded patch stream header: %+v", parsePatchStream.GetHeader)
	}
}

// TestHandleShardSessionPatchTransportRoundTripSharedBuffer verifies shared-buffer patch transport publish and host decode over one shard session.
func TestHandleShardSessionPatchTransportRoundTripSharedBuffer(parseT *testing.T) {
	parseRuntime := buildWorkerRuntimeForShardSessionPatchTransportTest(parseT)
	parseUpdateResult, parseUpdateErr := parseRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID:     "region-1",
		Epoch:        1,
		InputVersion: 2,
	})
	if parseUpdateErr != nil {
		parseT.Fatalf("HandleWorkerRegionUpdate returned error: %v", parseUpdateErr)
	}
	if !parseUpdateResult.HasPatchReady {
		parseT.Fatalf("expected patch-ready update result, got %+v", parseUpdateResult)
	}
	parseRawPatchPayload, parseRawPatchPayloadErr := BuildStructuredClonePatchPayloadJSON(parseUpdateResult.PatchIR)
	if parseRawPatchPayloadErr != nil {
		parseT.Fatalf("BuildStructuredClonePatchPayloadJSON returned error: %v", parseRawPatchPayloadErr)
	}
	parseStructuredEnvelope := StructuredClonePatchEnvelope{
		RegionInstanceID: "region-1",
		Epoch:            1,
		PatchVersion:     parseUpdateResult.PatchIR.GetHeader.PatchVersion,
		PatchPayload:     parseRawPatchPayload,
	}
	parseSharedPatchPage, parseSharedPatchPageErr := BuildSharedPatchPage(64 * 1024)
	if parseSharedPatchPageErr != nil {
		parseT.Fatalf("BuildSharedPatchPage returned error: %v", parseSharedPatchPageErr)
	}
	parseSharedTransportResult, parseSharedTransportResultErr := BuildSharedPatchTransportResult(
		parseStructuredEnvelope,
		BuildCapabilityReport(CapabilitySource{
			HasWorkerSupport:                true,
			HasMessagePortSupport:           true,
			HasStructuredCloneSupport:       true,
			HasSharedBufferSupport:          true,
			HasSharedMemoryTransportSupport: true,
		}),
		parseSharedPatchPage,
	)
	if parseSharedTransportResultErr != nil {
		parseT.Fatalf("BuildSharedPatchTransportResult returned error: %v", parseSharedTransportResultErr)
	}
	if parseSharedTransportResult.GetTransportTier != TransportTierSharedBuffer {
		parseT.Fatalf("expected shared-buffer transport tier, got %q", parseSharedTransportResult.GetTransportTier)
	}
	parsePatchReadyEnvelope, parsePatchReadyEnvelopeErr := BuildControlPatchReadyEnvelope(
		"region-1",
		parseSharedTransportResult.GetPatchVersion,
		parseUpdateResult.PatchIR.GetHeader.InputVersion,
		TransportTierSharedBuffer,
	)
	if parsePatchReadyEnvelopeErr != nil {
		parseT.Fatalf("BuildControlPatchReadyEnvelope returned error: %v", parsePatchReadyEnvelopeErr)
	}
	parsePayload := parseSharedTransportResult.GetMessagePayload
	if len(parsePayload) == 0 {
		parsePayload = []byte(`{"fallback":"unused"}`)
	}
	parseTransportTier, parsePatchStream := handleShardSessionPatchTransportRoundTrip(
		parseT,
		parsePatchReadyEnvelope,
		parsePayload,
		parseSharedPatchPage,
	)
	if parseTransportTier != TransportTierSharedBuffer {
		parseT.Fatalf("expected shared-buffer host decode tier, got %q", parseTransportTier)
	}
	if parsePatchStream.GetHeader.RegionID != "region-1" || parsePatchStream.GetHeader.PatchVersion != 2 {
		parseT.Fatalf("unexpected decoded patch stream header: %+v", parsePatchStream.GetHeader)
	}
}
