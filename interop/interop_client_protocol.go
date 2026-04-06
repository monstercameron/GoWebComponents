package interop

import (
	"errors"
	"strings"
	"time"
)

func DecodeClientMessage(parseValue any) (ClientMessage, error) {
	parseMessage, parseErr := decodeClientMessageValue(parseValue)
	if parseErr != nil {
		return ClientMessage{}, parseErr
	}
	if parseErr2 := validateClientMessage("DecodeClientMessage", parseMessage.Topic, parseMessage); parseErr2 != nil {
		return ClientMessage{}, parseErr2
	}
	return parseMessage, nil
}

// PublishClientMessage encodes and sends a ClientMessage over a CrossTabChannel.
func PublishClientMessage(parseChannel CrossTabChannel, parseMessage ClientMessage) error {
	parsePrepared, parseErr := prepareClientMessage("PublishClientMessage", parseChannel.Name(), parseMessage)
	if parseErr != nil {
		return parseErr
	}
	return parseChannel.Publish(parsePrepared)
}

// PublishClientWindowMessage encodes and sends a ClientMessage over a WindowChannel.
func PublishClientWindowMessage(parseChannel WindowChannel, parseMessage ClientMessage) error {
	parsePrepared, parseErr := prepareClientMessage("PublishClientWindowMessage", parseChannel.Name(), parseMessage)
	if parseErr != nil {
		return parseErr
	}
	return parseChannel.Publish(parsePrepared)
}

// SubscribeClientMessages decodes incoming CrossTabChannel envelopes as ClientMessages and invokes handler.
func SubscribeClientMessages(parseChannel CrossTabChannel, parseHandler func(ClientMessage, error)) (Subscription, error) {
	if parseHandler == nil {
		return Subscription{}, wrapError("SubscribeClientMessages", parseChannel.Name(), CodeInvalid, errors.New("handler is nil"))
	}
	return parseChannel.Subscribe(func(parseMessage CrossTabEnvelope, parseErr error) {
		if parseErr != nil {
			parseHandler(ClientMessage{}, parseErr)
			return
		}
		parseDecoded, parseDecodeErr := DecodeClientMessage(parseMessage.Payload)
		parseHandler(parseDecoded, parseDecodeErr)
	})
}

// SubscribeClientWindowMessages decodes incoming WindowChannel envelopes as ClientMessages and invokes handler.
func SubscribeClientWindowMessages(parseChannel WindowChannel, parseHandler func(ClientMessage, error)) (Subscription, error) {
	if parseHandler == nil {
		return Subscription{}, wrapError("SubscribeClientWindowMessages", parseChannel.Name(), CodeInvalid, errors.New("handler is nil"))
	}
	return parseChannel.Subscribe(func(parseMessage WindowEnvelope, parseErr error) {
		if parseErr != nil {
			parseHandler(ClientMessage{}, parseErr)
			return
		}
		parseDecoded, parseDecodeErr := DecodeClientMessage(parseMessage.Payload)
		parseHandler(parseDecoded, parseDecodeErr)
	})
}

// PublishClientHello sends a ClientHello presence message on a CrossTabChannel with default capabilities.
func PublishClientHello(parseChannel CrossTabChannel, parseSelf ClientIdentity) error {
	parseCapabilities := defaultCrossTabClientCapabilities(parseChannel)
	return PublishClientHelloWithCapabilities(parseChannel, parseSelf, parseCapabilities)
}

// PublishClientHelloWithCapabilities sends a ClientHello presence message on a CrossTabChannel with explicit capabilities.
func PublishClientHelloWithCapabilities(parseChannel CrossTabChannel, parseSelf ClientIdentity, parseCapabilities ClientCapabilities) error {
	return PublishClientMessage(parseChannel, ClientMessage{
		Kind:         ClientHello,
		Topic:        ClientPresenceTopic,
		Source:       parseSelf,
		Capabilities: &parseCapabilities,
	})
}

// PublishClientHelloWindow sends a ClientHello presence message on a WindowChannel with default capabilities.
func PublishClientHelloWindow(parseChannel WindowChannel, parseSelf ClientIdentity) error {
	parseCapabilities := defaultWindowClientCapabilities(parseChannel)
	return PublishClientHelloWindowWithCapabilities(parseChannel, parseSelf, parseCapabilities)
}

// PublishClientHelloWindowWithCapabilities sends a ClientHello presence message on a WindowChannel with explicit capabilities.
func PublishClientHelloWindowWithCapabilities(parseChannel WindowChannel, parseSelf ClientIdentity, parseCapabilities ClientCapabilities) error {
	return PublishClientWindowMessage(parseChannel, ClientMessage{
		Kind:         ClientHello,
		Topic:        ClientPresenceTopic,
		Source:       parseSelf,
		Capabilities: &parseCapabilities,
	})
}

// PublishClientGoodbye sends a ClientGoodbye presence message on a CrossTabChannel.
func PublishClientGoodbye(parseChannel CrossTabChannel, parseSelf ClientIdentity) error {
	return PublishClientMessage(parseChannel, ClientMessage{
		Kind:   ClientGoodbye,
		Topic:  ClientPresenceTopic,
		Source: parseSelf,
	})
}

// PublishClientGoodbyeWindow sends a ClientGoodbye presence message on a WindowChannel.
func PublishClientGoodbyeWindow(parseChannel WindowChannel, parseSelf ClientIdentity) error {
	return PublishClientWindowMessage(parseChannel, ClientMessage{
		Kind:   ClientGoodbye,
		Topic:  ClientPresenceTopic,
		Source: parseSelf,
	})
}

// PublishClientEvent sends a ClientEvent message on a CrossTabChannel to a named topic.
func PublishClientEvent(parseChannel CrossTabChannel, parseTopic string, parseSelf ClientIdentity, parsePayload any) error {
	return PublishClientMessage(parseChannel, ClientMessage{
		Kind:    ClientEvent,
		Topic:   strings.TrimSpace(parseTopic),
		Source:  parseSelf,
		Payload: parsePayload,
	})
}

// PublishClientIntent sends a directed ClientIntent message on a WindowChannel.
func PublishClientIntent(parseChannel WindowChannel, parseTopic string, parseSelf ClientIdentity, parseTarget string, parsePayload any) error {
	parseTrimmedTarget := strings.TrimSpace(parseTarget)
	if parseTrimmedTarget == "" {
		return wrapError("PublishClientIntent", parseChannel.Name(), CodeInvalid, errors.New("target is empty"))
	}
	return PublishClientWindowMessage(parseChannel, ClientMessage{
		Kind:    ClientIntent,
		Topic:   strings.TrimSpace(parseTopic),
		Source:  parseSelf,
		Target:  parseTrimmedTarget,
		Payload: parsePayload,
	})
}

// PublishClientInvalidation sends a ClientInvalidate message on a CrossTabChannel for a given topic and revision.
func PublishClientInvalidation(parseChannel CrossTabChannel, parseTopic string, parseSelf ClientIdentity, parseRevision string) error {
	parseTrimmedRevision := strings.TrimSpace(parseRevision)
	if parseTrimmedRevision == "" {
		return wrapError("PublishClientInvalidation", parseChannel.Name(), CodeInvalid, errors.New("revision is empty"))
	}
	return PublishClientMessage(parseChannel, ClientMessage{
		Kind:     ClientInvalidate,
		Topic:    strings.TrimSpace(parseTopic),
		Source:   parseSelf,
		Revision: parseTrimmedRevision,
	})
}

// PublishClientQuery sends a ClientQuery message on a CrossTabChannel for a given topic.
func PublishClientQuery(parseChannel CrossTabChannel, parseTopic string, parseSelf ClientIdentity) error {
	return PublishClientMessage(parseChannel, ClientMessage{
		Kind:   ClientQuery,
		Topic:  strings.TrimSpace(parseTopic),
		Source: parseSelf,
	})
}

// PublishClientResult sends a directed ClientResult message on a CrossTabChannel.
func PublishClientResult(parseChannel CrossTabChannel, parseTopic string, parseSelf ClientIdentity, parseTarget string, parsePayload any) error {
	parseTrimmedTarget := strings.TrimSpace(parseTarget)
	if parseTrimmedTarget == "" {
		return wrapError("PublishClientResult", parseChannel.Name(), CodeInvalid, errors.New("target is empty"))
	}
	return PublishClientMessage(parseChannel, ClientMessage{
		Kind:    ClientResult,
		Topic:   strings.TrimSpace(parseTopic),
		Source:  parseSelf,
		Target:  parseTrimmedTarget,
		Payload: parsePayload,
	})
}

// PublishClientBinaryWindow sends a binary ClientEvent on a WindowChannel to a specific target.
func PublishClientBinaryWindow(parseChannel WindowChannel, parseTopic string, parseSelf ClientIdentity, parseTarget string, parsePayload ClientBinaryPayload) error {
	parseTrimmedTarget := strings.TrimSpace(parseTarget)
	if parseTrimmedTarget == "" {
		return wrapError("PublishClientBinaryWindow", parseChannel.Name(), CodeInvalid, errors.New("target is empty"))
	}
	parsePrepared, parseErr := prepareClientBinaryMessage("PublishClientBinaryWindow", parseChannel.Name(), ClientMessage{
		Kind:        ClientEvent,
		Topic:       strings.TrimSpace(parseTopic),
		Source:      parseSelf,
		Target:      parseTrimmedTarget,
		Encoding:    ClientPayloadBinary,
		ContentType: strings.TrimSpace(parsePayload.ContentType),
		Payload:     append([]byte(nil), parsePayload.Bytes...),
	})
	if parseErr != nil {
		return parseErr
	}
	if parseChannel.publishClientBinary != nil {
		return parseChannel.publishClientBinary(parsePrepared)
	}
	return PublishClientWindowMessage(parseChannel, parsePrepared)
}

// PublishClientBinaryCrossTab sends a binary ClientEvent on a CrossTabChannel.
func PublishClientBinaryCrossTab(parseChannel CrossTabChannel, parseTopic string, parseSelf ClientIdentity, parsePayload ClientBinaryPayload) error {
	parsePrepared, parseErr := prepareClientBinaryMessage("PublishClientBinaryCrossTab", parseChannel.Name(), ClientMessage{
		Kind:        ClientEvent,
		Topic:       strings.TrimSpace(parseTopic),
		Source:      parseSelf,
		Encoding:    ClientPayloadBinary,
		ContentType: strings.TrimSpace(parsePayload.ContentType),
		Payload:     append([]byte(nil), parsePayload.Bytes...),
	})
	if parseErr != nil {
		return parseErr
	}
	if parseChannel.publishClientBinary != nil {
		return parseChannel.publishClientBinary(parsePrepared)
	}
	return PublishClientMessage(parseChannel, parsePrepared)
}

func prepareClientMessage(parseOp string, parseTarget string, parseMessage ClientMessage) (ClientMessage, error) {
	if parseErr := validateClientMessage(parseOp, parseTarget, parseMessage); parseErr != nil {
		return ClientMessage{}, parseErr
	}
	if parseMessage.SentAt.IsZero() {
		parseMessage.SentAt = time.Now().UTC()
	}
	return parseMessage, nil
}

func prepareClientBinaryMessage(parseOp string, parseTarget string, parseMessage ClientMessage) (ClientMessage, error) {
	parsePrepared, parseErr := prepareClientMessage(parseOp, parseTarget, parseMessage)
	if parseErr != nil {
		return ClientMessage{}, parseErr
	}
	parsePrepared.Encoding = ClientPayloadBinary
	parseBytes, parseOk := parsePrepared.Payload.([]byte)
	if !parseOk {
		return ClientMessage{}, wrapError(parseOp, parseTarget, CodeInvalid, errors.New("binary payload must be []byte"))
	}
	if len(parseBytes) == 0 {
		return ClientMessage{}, wrapError(parseOp, parseTarget, CodeInvalid, errors.New("binary payload is empty"))
	}
	if strings.TrimSpace(parsePrepared.ContentType) == "" {
		return ClientMessage{}, wrapError(parseOp, parseTarget, CodeInvalid, errors.New("binary content type is empty"))
	}
	parsePrepared.Payload = append([]byte(nil), parseBytes...)
	return parsePrepared, nil
}

func validateClientMessage(parseOp string, parseTarget string, parseMessage ClientMessage) error {
	if parseErr := validateClientIdentity(parseOp, parseTarget, parseMessage.Source); parseErr != nil {
		return parseErr
	}
	if strings.TrimSpace(parseMessage.Topic) == "" {
		return wrapError(parseOp, parseTarget, CodeInvalid, errors.New("client topic is empty"))
	}
	switch parseMessage.Kind {
	case ClientHello, ClientGoodbye, ClientEvent, ClientIntent, ClientQuery, ClientResult, ClientInvalidate, ClientError:
	default:
		return wrapError(parseOp, parseTarget, CodeInvalid, errors.New("client message kind is empty or unknown"))
	}
	switch parseMessage.Encoding {
	case "", ClientPayloadJSON, ClientPayloadBinary:
	default:
		return wrapError(parseOp, parseTarget, CodeInvalid, errors.New("client payload encoding is empty or unknown"))
	}
	if parseMessage.Encoding == ClientPayloadBinary {
		if _, parseOk := parseMessage.Payload.([]byte); !parseOk {
			return wrapError(parseOp, parseTarget, CodeInvalid, errors.New("binary client payload must be []byte"))
		}
	}
	if parseErr2 := validateClientTopicAuthorization(parseOp, parseTarget, parseMessage); parseErr2 != nil {
		return parseErr2
	}
	return nil
}

func validateClientTopicAuthorization(parseOp string, parseTarget string, parseMessage ClientMessage) error {
	parseTopic := strings.ToLower(strings.TrimSpace(parseMessage.Topic))
	if !isPrivilegedClientTopic(parseTopic) {
		return nil
	}
	parseRole := normalizeClientRole(parseMessage.Source.Role)
	if clientRoleMayUsePrivilegedTopic(parseRole) {
		return nil
	}
	if parseMessage.Kind == ClientIntent {
		return wrapError(parseOp, parseTarget, CodeUnauthorized, errors.New("client role is not authorized to publish privileged intent topic"))
	}
	return wrapError(parseOp, parseTarget, CodeUnauthorized, errors.New("client role is not authorized for privileged topic"))
}

func isPrivilegedClientTopic(parseTopic string) bool {
	switch {
	case strings.HasPrefix(parseTopic, "session:"), strings.HasPrefix(parseTopic, "operator:"), strings.HasPrefix(parseTopic, "intent:session"), strings.HasPrefix(parseTopic, "intent:operator"):
		return true
	default:
		return false
	}
}

func normalizeClientRole(parseRole string) string {
	return strings.ToLower(strings.TrimSpace(parseRole))
}

func clientRoleMayUsePrivilegedTopic(parseRole string) bool {
	switch parseRole {
	case "operator", "admin", "system":
		return true
	default:
		return false
	}
}

func decodeClientMessageValue(parseValue any) (ClientMessage, error) {
	switch parseTyped := parseValue.(type) {
	case ClientMessage:
		return parseTyped, nil
	case map[string]any:
		return decodeClientMessageMap(parseTyped)
	default:
		var parseMessage ClientMessage
		if parseErr := Decode(parseValue, &parseMessage); parseErr != nil {
			return ClientMessage{}, parseErr
		}
		return parseMessage, nil
	}
}

func decodeClientMessageMap(parseData map[string]any) (ClientMessage, error) {
	parseMessage := ClientMessage{
		ID:          stringField(parseData, "id"),
		Kind:        ClientMessageKind(stringField(parseData, "kind")),
		Topic:       stringField(parseData, "topic"),
		Target:      stringField(parseData, "target"),
		Revision:    stringField(parseData, "revision"),
		Error:       stringField(parseData, "error"),
		Encoding:    ClientPayloadEncoding(stringField(parseData, "encoding")),
		ContentType: stringField(parseData, "contentType"),
	}
	if parseSource, parseOk := parseData["source"].(map[string]any); parseOk {
		parseMessage.Source = ClientIdentity{
			ID:      stringField(parseSource, "id"),
			App:     stringField(parseSource, "app"),
			Surface: stringField(parseSource, "surface"),
			Role:    stringField(parseSource, "role"),
			Version: stringField(parseSource, "version"),
		}
	}
	if parseCapabilities, parseOk2 := parseData["capabilities"].(map[string]any); parseOk2 {
		parseMessage.Capabilities = decodeClientCapabilitiesMap(parseCapabilities)
	}
	if parsePayload, parseOk3 := parseData["payload"]; parseOk3 {
		parseMessage.Payload = parsePayload
	}
	if parseSentAt, parseOk4 := clientTimeField(parseData["sentAt"]); parseOk4 {
		parseMessage.SentAt = parseSentAt
	}
	return parseMessage, nil
}

func decodeClientCapabilitiesMap(parseData map[string]any) *ClientCapabilities {
	parseCapabilities := &ClientCapabilities{
		ProtocolVersion: stringField(parseData, "protocolVersion"),
		Transports:      stringSliceField(parseData["transports"]),
		Encodings:       stringSliceField(parseData["encodings"]),
		Topics:          stringSliceField(parseData["topics"]),
		MaxJSONBytes:    intField(parseData["maxJsonBytes"]),
		MaxBinaryBytes:  intField(parseData["maxBinaryBytes"]),
	}
	return parseCapabilities
}

func stringField(parseData map[string]any, parseKey string) string {
	parseValue, parseOk := parseData[parseKey]
	if !parseOk {
		return ""
	}
	switch parseTyped := parseValue.(type) {
	case string:
		return parseTyped
	default:
		return ""
	}
}

func stringSliceField(parseValue any) []string {
	switch parseTyped := parseValue.(type) {
	case []string:
		return append([]string(nil), parseTyped...)
	case []any:
		parseValues := make([]string, 0, len(parseTyped))
		for _, parseEntry := range parseTyped {
			parseText, parseOk := parseEntry.(string)
			if parseOk && strings.TrimSpace(parseText) != "" {
				parseValues = append(parseValues, parseText)
			}
		}
		return parseValues
	default:
		return nil
	}
}

func intField(parseValue any) int {
	switch parseTyped := parseValue.(type) {
	case int:
		return parseTyped
	case int64:
		return int(parseTyped)
	case float64:
		return int(parseTyped)
	default:
		return 0
	}
}

func clientTimeField(parseValue any) (time.Time, bool) {
	parseText, parseOk := parseValue.(string)
	if !parseOk || strings.TrimSpace(parseText) == "" {
		return time.Time{}, false
	}
	parseParsed, parseErr := time.Parse(time.RFC3339Nano, parseText)
	if parseErr != nil {
		return time.Time{}, false
	}
	return parseParsed, true
}

func defaultCrossTabClientCapabilities(parseChannel CrossTabChannel) ClientCapabilities {
	parseEncodings := []string{string(ClientPayloadJSON)}
	if parseChannel.Transport() == "broadcast-channel" {
		parseEncodings = append(parseEncodings, string(ClientPayloadBinary))
	}
	parseTransports := []string{}
	if parseTransport := strings.TrimSpace(parseChannel.Transport()); parseTransport != "" {
		parseTransports = append(parseTransports, parseTransport)
	}
	return ClientCapabilities{
		ProtocolVersion: "v1",
		Transports:      parseTransports,
		Encodings:       parseEncodings,
	}
}

func defaultWindowClientCapabilities(parseChannel WindowChannel) ClientCapabilities {
	parseTransports := []string{"window-message"}
	if parseTarget := strings.TrimSpace(parseChannel.Name()); parseTarget != "" {
		_ = parseTarget
	}
	return ClientCapabilities{
		ProtocolVersion: "v1",
		Transports:      parseTransports,
		Encodings:       []string{string(ClientPayloadJSON), string(ClientPayloadBinary)},
	}
}

// ClientProtocolCompatible reports whether two capability sets share a compatible protocol major version.
func ClientProtocolCompatible(parseLocal ClientCapabilities, parsePeer ClientCapabilities) bool {
	parseLocalVersion := normalizeProtocolVersion(parseLocal.ProtocolVersion)
	parsePeerVersion := normalizeProtocolVersion(parsePeer.ProtocolVersion)
	if parseLocalVersion == "" || parsePeerVersion == "" {
		return false
	}
	return protocolMajor(parseLocalVersion) == protocolMajor(parsePeerVersion)
}

// ClientSupportsEncoding reports whether the capabilities include the given payload encoding.
func ClientSupportsEncoding(parseCapabilities ClientCapabilities, parseEncoding ClientPayloadEncoding) bool {
	parseTrimmed := strings.TrimSpace(string(parseEncoding))
	if parseTrimmed == "" {
		parseTrimmed = string(ClientPayloadJSON)
	}
	for _, parseCandidate := range parseCapabilities.Encodings {
		if strings.EqualFold(strings.TrimSpace(parseCandidate), parseTrimmed) {
			return true
		}
	}
	return false
}

// ClientSupportsTopic reports whether the capabilities include the given topic (empty topic list means all).
func ClientSupportsTopic(parseCapabilities ClientCapabilities, parseTopic string) bool {
	parseTrimmed := strings.TrimSpace(parseTopic)
	if parseTrimmed == "" {
		return false
	}
	if len(parseCapabilities.Topics) == 0 {
		return true
	}
	for _, parseCandidate := range parseCapabilities.Topics {
		if strings.EqualFold(strings.TrimSpace(parseCandidate), parseTrimmed) {
			return true
		}
	}
	return false
}

// ClientCanExchange reports whether two clients can communicate on a topic with a shared encoding.
func ClientCanExchange(parseLocal ClientCapabilities, parsePeer ClientCapabilities, parseTopic string, parseEncoding ClientPayloadEncoding) bool {
	if !ClientProtocolCompatible(parseLocal, parsePeer) {
		return false
	}
	if !ClientSupportsEncoding(parseLocal, parseEncoding) || !ClientSupportsEncoding(parsePeer, parseEncoding) {
		return false
	}
	if !ClientSupportsTopic(parseLocal, parseTopic) || !ClientSupportsTopic(parsePeer, parseTopic) {
		return false
	}
	return true
}

func normalizeProtocolVersion(parseValue string) string {
	parseTrimmed := strings.TrimSpace(strings.ToLower(parseValue))
	parseTrimmed = strings.TrimPrefix(parseTrimmed, "v")
	return parseTrimmed
}

func protocolMajor(parseValue string) string {
	parseTrimmed := normalizeProtocolVersion(parseValue)
	if parseTrimmed == "" {
		return ""
	}
	if parseDot := strings.Index(parseTrimmed, "."); parseDot >= 0 {
		return parseTrimmed[:parseDot]
	}
	return parseTrimmed
}

func validateClientIdentity(parseOp string, parseTarget string, parseIdentity ClientIdentity) error {
	if strings.TrimSpace(parseIdentity.ID) == "" {
		return wrapError(parseOp, parseTarget, CodeInvalid, errors.New("client identity id is empty"))
	}
	if strings.TrimSpace(parseIdentity.App) == "" {
		return wrapError(parseOp, parseTarget, CodeInvalid, errors.New("client identity app is empty"))
	}
	if strings.TrimSpace(parseIdentity.Surface) == "" {
		return wrapError(parseOp, parseTarget, CodeInvalid, errors.New("client identity surface is empty"))
	}
	return nil
}

func SubscribeDecodedWindow[T any](parseChannel WindowChannel, parseHandler func(DecodedWindowEnvelope[T], error)) (Subscription, error) {
	if parseHandler == nil {
		return Subscription{}, wrapError("SubscribeDecodedWindow", parseChannel.Name(), CodeInvalid, errors.New("handler is nil"))
	}
	return parseChannel.Subscribe(func(parseMessage WindowEnvelope, parseErr error) {
		if parseErr != nil {
			parseHandler(DecodedWindowEnvelope[T]{}, parseErr)
			return
		}
		parseDecoded, parseDecodeErr := DecodeWindowEnvelope[T](parseMessage)
		parseHandler(parseDecoded, parseDecodeErr)
	})
}

// DecodeSurfaceSignal decodes a WindowEnvelope payload as a SurfaceSignal.
func DecodeSurfaceSignal(parseMessage WindowEnvelope) (DecodedWindowEnvelope[SurfaceSignal], error) {
	return DecodeWindowEnvelope[SurfaceSignal](parseMessage)
}

// SubscribeSurfaceSignals receives decoded SurfaceSignal messages on a WindowChannel.
func SubscribeSurfaceSignals(parseChannel WindowChannel, parseHandler func(DecodedWindowEnvelope[SurfaceSignal], error)) (Subscription, error) {
	if parseHandler == nil {
		return Subscription{}, wrapError("SubscribeSurfaceSignals", parseChannel.Name(), CodeInvalid, errors.New("handler is nil"))
	}
	return SubscribeDecodedWindow(parseChannel, parseHandler)
}

// PublishSurfaceSignal validates and sends a SurfaceSignal over a WindowChannel.
func PublishSurfaceSignal(parseChannel WindowChannel, parseSignal SurfaceSignal) error {
	switch parseSignal.Kind {
	case SurfaceSignalSession:
		if parseSignal.Session == nil {
			return wrapError("PublishSurfaceSignal", parseChannel.Name(), CodeInvalid, errors.New("session signal is missing session payload"))
		}
	case SurfaceSignalRoute:
		if parseSignal.Route == nil || strings.TrimSpace(parseSignal.Route.Path) == "" {
			return wrapError("PublishSurfaceSignal", parseChannel.Name(), CodeInvalid, errors.New("route signal is missing path"))
		}
	case SurfaceSignalSelection:
		if parseSignal.Selection == nil || strings.TrimSpace(parseSignal.Selection.ID) == "" {
			return wrapError("PublishSurfaceSignal", parseChannel.Name(), CodeInvalid, errors.New("selection signal is missing id"))
		}
	case SurfaceSignalIntent:
		if parseSignal.Intent == nil || strings.TrimSpace(string(parseSignal.Intent.Action)) == "" {
			return wrapError("PublishSurfaceSignal", parseChannel.Name(), CodeInvalid, errors.New("intent signal is missing action"))
		}
	default:
		return wrapError("PublishSurfaceSignal", parseChannel.Name(), CodeInvalid, errors.New("surface signal kind is empty or unknown"))
	}
	return parseChannel.Publish(parseSignal)
}

// PublishLogout sends a signed-out session signal on a WindowChannel.
func PublishLogout(parseChannel WindowChannel, parseReason string) error {
	return PublishSurfaceSignal(parseChannel, SurfaceSignal{
		Kind: SurfaceSignalSession,
		Session: &SurfaceSessionSignal{
			Status: "signed-out",
			Reason: strings.TrimSpace(parseReason),
		},
	})
}

// PublishSessionExpired sends a session-expired signal on a WindowChannel.
func PublishSessionExpired(parseChannel WindowChannel, parseReason string, parseReturnTo string, parseExpiresAt time.Time) error {
	return PublishSurfaceSignal(parseChannel, SurfaceSignal{
		Kind: SurfaceSignalSession,
		Session: &SurfaceSessionSignal{
			Status:    "expired",
			Reason:    strings.TrimSpace(parseReason),
			ReturnTo:  strings.TrimSpace(parseReturnTo),
			ExpiresAt: parseExpiresAt,
		},
	})
}

// PublishRouteFocus sends a route-focus surface signal on a WindowChannel.
func PublishRouteFocus(parseChannel WindowChannel, parsePath string, parseQuery string, parseFocusID string) error {
	return PublishSurfaceSignal(parseChannel, SurfaceSignal{
		Kind: SurfaceSignalRoute,
		Route: &SurfaceRouteSignal{
			Path:    strings.TrimSpace(parsePath),
			Query:   strings.TrimSpace(parseQuery),
			FocusID: strings.TrimSpace(parseFocusID),
		},
	})
}

// PublishSelection sends a selection surface signal on a WindowChannel.
func PublishSelection(parseChannel WindowChannel, parseScope string, parseId string, parseRevision string) error {
	return PublishSurfaceSignal(parseChannel, SurfaceSignal{
		Kind: SurfaceSignalSelection,
		Selection: &SurfaceSelectionSignal{
			Scope:    strings.TrimSpace(parseScope),
			ID:       strings.TrimSpace(parseId),
			Revision: strings.TrimSpace(parseRevision),
		},
	})
}

// PublishIntent sends an intent surface signal on a WindowChannel.
func PublishIntent(parseChannel WindowChannel, parseAction SurfaceIntentAction, parseTarget string, parseParams map[string]string) error {
	return PublishSurfaceSignal(parseChannel, SurfaceSignal{
		Kind: SurfaceSignalIntent,
		Intent: &SurfaceIntentSignal{
			Action: parseAction,
			Target: strings.TrimSpace(parseTarget),
			Params: parseParams,
		},
	})
}
