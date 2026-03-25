package chatpb

import (
	"reflect"
	"strings"
	"testing"
)

func TestGeneratedMessagesExposeNoArgMethodsAndNilSafeGetters(parseT *testing.T) {
	parseMessages := []interface{}{
		&ChatMessage{},
		&SendRequest{},
		&ChatChunk{},
		&ModelCapabilities{},
		&ModelPricing{},
		&ModelOption{},
		&ListModelOptionsRequest{},
		&ListModelOptionsResponse{},
		&ConversationSummary{},
		&ListConversationsRequest{},
		&ListConversationsResponse{},
		&ResolveConversationRouteRequest{},
		&ResolveConversationRouteResponse{},
		&LoadConversationRequest{},
		&LoadConversationResponse{},
		&DeleteConversationRequest{},
		&DeleteConversationResponse{},
		&SetUserNameRequest{},
		&SetUserNameResponse{},
		&GetUserNameRequest{},
		&GetUserNameResponse{},
		&UserMemory{},
		&ListUserMemoriesRequest{},
		&ListUserMemoriesResponse{},
		&UpsertUserMemoryRequest{},
		&DeleteUserMemoryRequest{},
		&SignupRequest{},
		&LoginRequest{},
		&AuthResponse{},
		&GetSessionResponse{},
		&SynthesizeSpeechRequest{},
		&SynthesizeSpeechChunk{},
	}

	for _, parseMessage := range parseMessages {
		parseCallNoArgMethods(parseT, parseMessage)
		parseCallNilGetters(parseT, reflect.TypeOf(parseMessage))
	}
}

func parseCallNoArgMethods(parseT *testing.T, parseMessage interface{}) {
	parseT.Helper()
	parseValue := reflect.ValueOf(parseMessage)
	parseTyp := parseValue.Type()
	for parseIndex := 0; parseIndex < parseValue.NumMethod(); parseIndex++ {
		parseMethod := parseValue.Method(parseIndex)
		parseMethodType := parseMethod.Type()
		if parseMethodType.NumIn() != 0 {
			continue
		}
		// Skip two-return Descriptor signatures and other non-trivial call shapes.
		if parseMethodType.NumOut() > 1 {
			continue
		}
		if parseTyp.Method(parseIndex).Name == "ProtoMessage" {
			continue
		}
		parseMethod.Call(nil)
	}
}

func parseCallNilGetters(parseT *testing.T, parsePointerType reflect.Type) {
	parseT.Helper()
	parseNilValue := reflect.Zero(parsePointerType)
	for parseIndex := 0; parseIndex < parseNilValue.NumMethod(); parseIndex++ {
		parseMethodInfo := parsePointerType.Method(parseIndex)
		parseMethodType := parseMethodInfo.Type
		if !strings.HasPrefix(parseMethodInfo.Name, "Get") {
			continue
		}
		// Receiver + no parameters.
		if parseMethodType.NumIn() != 1 || parseMethodType.NumOut() != 1 {
			continue
		}
		parseNilValue.Method(parseIndex).Call(nil)
	}
}
