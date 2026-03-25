package chatpb

import (
	"reflect"
	"strings"
	"testing"
)

func TestGeneratedMessagesExposeNoArgMethodsAndNilSafeGetters(t *testing.T) {
	messages := []interface{}{
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

	for _, message := range messages {
		callNoArgMethods(t, message)
		callNilGetters(t, reflect.TypeOf(message))
	}
}

func callNoArgMethods(t *testing.T, message interface{}) {
	t.Helper()
	value := reflect.ValueOf(message)
	typ := value.Type()
	for index := 0; index < value.NumMethod(); index++ {
		method := value.Method(index)
		methodType := method.Type()
		if methodType.NumIn() != 0 {
			continue
		}
		// Skip two-return Descriptor signatures and other non-trivial call shapes.
		if methodType.NumOut() > 1 {
			continue
		}
		if typ.Method(index).Name == "ProtoMessage" {
			continue
		}
		method.Call(nil)
	}
}

func callNilGetters(t *testing.T, pointerType reflect.Type) {
	t.Helper()
	nilValue := reflect.Zero(pointerType)
	for index := 0; index < nilValue.NumMethod(); index++ {
		methodInfo := pointerType.Method(index)
		methodType := methodInfo.Type
		if !strings.HasPrefix(methodInfo.Name, "Get") {
			continue
		}
		// Receiver + no parameters.
		if methodType.NumIn() != 1 || methodType.NumOut() != 1 {
			continue
		}
		nilValue.Method(index).Call(nil)
	}
}
