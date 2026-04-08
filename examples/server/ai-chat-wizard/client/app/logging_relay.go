//go:build js && wasm

package app

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type chatRelayLogger struct {
	parseLogger logging.Logger
}

type parseClientLogRelayState struct {
	parseMutex    sync.RWMutex
	parseClient   chatpb.ChatServiceClient
	parseClientID string
	isParseReady  bool
}

var parseClientLogRelay = &parseClientLogRelayState{}
var parseRelayLogger = logging.New("chat-wizard-client-relay")

const parseClientLogRelayMaxAttempts = 3
const parseClientLogRelayAttemptTimeout = 3 * time.Second
const parseClientLogRelayBackoffBase = 250 * time.Millisecond

// parseNewChatRelayLogger builds a scoped logger that mirrors entries locally and forwards them to the server.
func parseNewChatRelayLogger(parseScope string) chatRelayLogger {
	return chatRelayLogger{parseLogger: logging.New(parseScope)}
}

// Scope returns the wrapped logger scope.
func (parseRelay chatRelayLogger) Scope() string {
	return parseRelay.parseLogger.Scope()
}

// Log writes a structured entry locally and attempts async forwarding to the server bridge.
func (parseRelay chatRelayLogger) Log(parseLevel string, parseMessage string, parseFields logging.Fields) {
	parseRelay.parseLogger.Log(parseLevel, parseMessage, parseFields)
	parseForwardClientLogAsync(parseLevel, parseRelay.parseLogger.Scope(), parseMessage, parseFields)
}

// Debug writes a debug-level entry.
func (parseRelay chatRelayLogger) Debug(parseMessage string, parseFields logging.Fields) {
	parseRelay.Log("debug", parseMessage, parseFields)
}

// Info writes an info-level entry.
func (parseRelay chatRelayLogger) Info(parseMessage string, parseFields logging.Fields) {
	parseRelay.Log("info", parseMessage, parseFields)
}

// Warn writes a warning-level entry.
func (parseRelay chatRelayLogger) Warn(parseMessage string, parseFields logging.Fields) {
	parseRelay.Log("warn", parseMessage, parseFields)
}

// Error writes an error-level entry.
func (parseRelay chatRelayLogger) Error(parseMessage string, parseFields logging.Fields) {
	parseRelay.Log("error", parseMessage, parseFields)
}

// parseSetClientLogRelay updates the active relay target for client log forwarding.
func parseSetClientLogRelay(parseClient chatpb.ChatServiceClient, parseClientID string, isParseReady bool) {
	parseClientLogRelay.parseMutex.Lock()
	defer parseClientLogRelay.parseMutex.Unlock()
	parseClientLogRelay.parseClient = parseClient
	parseClientLogRelay.parseClientID = parseNormalizeClientIdentity(parseClientID)
	parseClientLogRelay.isParseReady = isParseReady && parseClient != nil
}

// parseForwardClientLogAsync dispatches one non-blocking client log forward attempt.
func parseForwardClientLogAsync(parseLevel string, parseScope string, parseMessage string, parseFields logging.Fields) {
	parseMessage = strings.TrimSpace(parseMessage)
	if parseMessage == "" {
		return
	}
	go parseSendClientLog(parseLevel, parseScope, parseMessage, parseFields)
}

// parseSendClientLog sends one structured client log record to the server over gRPC.
func parseSendClientLog(parseLevel string, parseScope string, parseMessage string, parseFields logging.Fields) {
	parseStructuredFields, parseErr := structpb.NewStruct(parseBuildClientLogFields(parseFields))
	if parseErr != nil {
		parseStructuredFields = nil
	}
	parseReq := &chatpb.ReportClientLogRequest{
		Level:      strings.TrimSpace(parseLevel),
		Scope:      strings.TrimSpace(parseScope),
		Message:    parseMessage,
		Fields:     parseStructuredFields,
		ObservedAt: timestamppb.Now(),
	}
	var parseLastErr error
	var parseClientID string
	for parseAttempt := 1; parseAttempt <= parseClientLogRelayMaxAttempts; parseAttempt++ {
		parseClient, parseAttemptClientID, isParseReady := parseSnapshotClientLogRelay()
		if !isParseReady || parseClient == nil || parseAttemptClientID == "" {
			return
		}
		parseClientID = parseAttemptClientID
		parseReq.ClientId = parseClientID
		parseCtx, parseCancel := context.WithTimeout(context.Background(), parseClientLogRelayAttemptTimeout)
		_, parseErr2 := parseClient.ReportClientLog(parseAuthContextWithMetadata(parseCtx), parseReq)
		parseCancel()
		if parseErr2 == nil {
			return
		}
		parseLastErr = parseErr2
		if !parseShouldRetryClientLogRelay(parseErr2) || parseAttempt == parseClientLogRelayMaxAttempts {
			break
		}
		time.Sleep(parseClientLogRelayBackoff(parseAttempt))
	}
	if parseLastErr != nil {
		parseRelayLogger.Warn(parseFormatClientLogRelayFailureMessage(parseLevel, parseScope, parseClientID), logging.Fields{"error": parseLastErr})
	}
}

// parseShouldRetryClientLogRelay reports whether one client-log relay failure is transient enough for another attempt.
func parseShouldRetryClientLogRelay(parseErr error) bool {
	if parseErr == nil {
		return false
	}
	parseStatusErr, parseOk := status.FromError(parseErr)
	if !parseOk {
		return true
	}
	switch parseStatusErr.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled, codes.Unknown, codes.ResourceExhausted:
		return true
	default:
		return false
	}
}

// parseClientLogRelayBackoff returns the delay before the next client-log relay retry attempt.
func parseClientLogRelayBackoff(parseAttempt int) time.Duration {
	if parseAttempt <= 0 {
		return parseClientLogRelayBackoffBase
	}
	parseDelay := parseClientLogRelayBackoffBase
	for parseIndex := 1; parseIndex < parseAttempt; parseIndex++ {
		parseDelay *= 2
	}
	return parseDelay
}

// parseSnapshotClientLogRelay reads one consistent relay snapshot for async forwarding.
func parseSnapshotClientLogRelay() (chatpb.ChatServiceClient, string, bool) {
	parseClientLogRelay.parseMutex.RLock()
	defer parseClientLogRelay.parseMutex.RUnlock()
	return parseClientLogRelay.parseClient, parseClientLogRelay.parseClientID, parseClientLogRelay.isParseReady
}

// parseEnsureClientIdentity fetches and stores a server-issued client identity when one is missing.
func parseEnsureClientIdentity(parseClient chatpb.ChatServiceClient) {
	if parseClient == nil {
		return
	}
	parsePersistedClientID := parseLoadPersistedClientIdentity()
	if parsePersistedClientID != "" {
		_ = parseEnsurePersistedCorrelationIdentity()
		parseSetClientLogRelay(parseClient, parsePersistedClientID, true)
		return
	}
	go func() {
		parseCtx, parseCancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer parseCancel()
		parseResp, parseErr := parseClient.GetClientIdentity(parseAuthContextWithMetadata(parseCtx), &emptypb.Empty{})
		if parseErr != nil {
			parseRelayLogger.Warn("client identity fetch failed: rpc=GetClientIdentity", logging.Fields{"error": parseErr})
			return
		}
		parseClientID := parseNormalizeClientIdentity(parseResp.GetClientId())
		if parseClientID == "" {
			parseRelayLogger.Warn("client identity fetch returned blank identity: rpc=GetClientIdentity", nil)
			return
		}
		parsePersistClientIdentity(parseClientID)
		_ = parseEnsurePersistedCorrelationIdentity()
		parseSetClientLogRelay(parseClient, parseClientID, true)
		parseRelayLogger.Info(fmt.Sprintf("client identity ready: client.id=%s", parseClientID), logging.Fields{"client_id": parseClientID})
	}()
}

// parseFormatClientLogRelayFailureMessage formats one actionable relay failure message with key dimensions.
func parseFormatClientLogRelayFailureMessage(parseLevel string, parseScope string, parseClientID string) string {
	parseLevel = strings.TrimSpace(parseLevel)
	if parseLevel == "" {
		parseLevel = "info"
	}
	parseScope = strings.TrimSpace(parseScope)
	if parseScope == "" {
		parseScope = "chat-wizard"
	}
	parseClientID = strings.TrimSpace(parseClientID)
	if parseClientID == "" {
		parseClientID = "unknown"
	}
	return fmt.Sprintf("client log relay failed: level=%s scope=%s client.id=%s", parseLevel, parseScope, parseClientID)
}

// parseLoadPersistedClientIdentity loads the cached server-issued client identity from local storage.
func parseLoadPersistedClientIdentity() string {
	parseStorage, parseErr := interop.GetLocalStorage()
	if parseErr != nil {
		return ""
	}
	parseValue, parseOk, parseErr2 := parseStorage.GetItem(storageKeyClientIdentity)
	if parseErr2 != nil || !parseOk {
		return ""
	}
	return parseNormalizeClientIdentity(parseValue)
}

// parsePersistClientIdentity stores one normalized server-issued client identity in local storage.
func parsePersistClientIdentity(parseClientID string) {
	parseClientID = parseNormalizeClientIdentity(parseClientID)
	if parseClientID == "" {
		return
	}
	parseStorage, parseErr := interop.GetLocalStorage()
	if parseErr != nil {
		return
	}
	_ = parseStorage.SetItem(storageKeyClientIdentity, parseClientID)
}

// parseLoadPersistedCorrelationIdentity loads the cached client correlation identity from local storage.
func parseLoadPersistedCorrelationIdentity() string {
	parseStorage, parseErr := interop.GetLocalStorage()
	if parseErr != nil {
		return ""
	}
	parseValue, parseOk, parseErr2 := parseStorage.GetItem(storageKeyCorrelationIdentity)
	if parseErr2 != nil || !parseOk {
		return ""
	}
	return parseNormalizeCorrelationIdentity(parseValue)
}

// parsePersistCorrelationIdentity stores one normalized client correlation identity in local storage.
func parsePersistCorrelationIdentity(parseCorrelationID string) {
	parseCorrelationID = parseNormalizeCorrelationIdentity(parseCorrelationID)
	if parseCorrelationID == "" {
		return
	}
	parseStorage, parseErr := interop.GetLocalStorage()
	if parseErr != nil {
		return
	}
	_ = parseStorage.SetItem(storageKeyCorrelationIdentity, parseCorrelationID)
}

// parseEnsurePersistedCorrelationIdentity returns a stable correlation identity, generating one when needed.
func parseEnsurePersistedCorrelationIdentity() string {
	parsePersistedCorrelationID := parseLoadPersistedCorrelationIdentity()
	if parsePersistedCorrelationID != "" {
		return parsePersistedCorrelationID
	}
	parseGeneratedCorrelationID := parseNormalizeCorrelationIdentity(uuid.NewString())
	if parseGeneratedCorrelationID == "" {
		return ""
	}
	parsePersistCorrelationIdentity(parseGeneratedCorrelationID)
	return parseGeneratedCorrelationID
}

// parseNormalizeClientIdentity normalizes and validates one client identity candidate.
func parseNormalizeClientIdentity(parseRawClientID string) string {
	parseRawClientID = strings.TrimSpace(parseRawClientID)
	if parseRawClientID == "" {
		return ""
	}
	if _, parseErr := uuid.Parse(parseRawClientID); parseErr != nil {
		return ""
	}
	return parseRawClientID
}

// parseNormalizeCorrelationIdentity normalizes and validates one client correlation identity candidate.
func parseNormalizeCorrelationIdentity(parseRawCorrelationID string) string {
	parseRawCorrelationID = strings.TrimSpace(parseRawCorrelationID)
	if parseRawCorrelationID == "" {
		return ""
	}
	if _, parseErr := uuid.Parse(parseRawCorrelationID); parseErr != nil {
		return ""
	}
	return parseRawCorrelationID
}

// parseBuildClientLogFields sanitizes dynamic log fields into protobuf-struct safe values.
func parseBuildClientLogFields(parseFields logging.Fields) map[string]any {
	parseNormalized := map[string]any{}
	for parseKey, parseValue := range parseFields {
		parseKey = strings.TrimSpace(parseKey)
		if parseKey == "" {
			continue
		}
		parseNormalized[parseKey] = parseNormalizeClientLogFieldValue(parseValue)
	}
	return parseNormalized
}

// parseNormalizeClientLogFieldValue converts one arbitrary value into a protobuf struct-safe value.
func parseNormalizeClientLogFieldValue(parseValue any) any {
	switch parseTypedValue := parseValue.(type) {
	case nil:
		return nil
	case string:
		return parseTypedValue
	case bool:
		return parseTypedValue
	case float32:
		return float64(parseTypedValue)
	case float64:
		return parseTypedValue
	case int:
		return float64(parseTypedValue)
	case int8:
		return float64(parseTypedValue)
	case int16:
		return float64(parseTypedValue)
	case int32:
		return float64(parseTypedValue)
	case int64:
		return float64(parseTypedValue)
	case uint:
		return float64(parseTypedValue)
	case uint8:
		return float64(parseTypedValue)
	case uint16:
		return float64(parseTypedValue)
	case uint32:
		return float64(parseTypedValue)
	case uint64:
		return float64(parseTypedValue)
	case error:
		return parseTypedValue.Error()
	case time.Time:
		return parseTypedValue.UTC().Format(time.RFC3339Nano)
	case time.Duration:
		return parseTypedValue.String()
	case fmt.Stringer:
		return parseTypedValue.String()
	case []string:
		parseResult := make([]any, 0, len(parseTypedValue))
		for _, parseItem := range parseTypedValue {
			parseResult = append(parseResult, parseItem)
		}
		return parseResult
	case []any:
		parseResult := make([]any, 0, len(parseTypedValue))
		for _, parseItem := range parseTypedValue {
			parseResult = append(parseResult, parseNormalizeClientLogFieldValue(parseItem))
		}
		return parseResult
	case map[string]any:
		parseResult := map[string]any{}
		for parseKey, parseItem := range parseTypedValue {
			parseKey = strings.TrimSpace(parseKey)
			if parseKey == "" {
				continue
			}
			parseResult[parseKey] = parseNormalizeClientLogFieldValue(parseItem)
		}
		return parseResult
	default:
		return fmt.Sprint(parseValue)
	}
}
