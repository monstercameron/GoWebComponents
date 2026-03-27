package app

import (
	"context"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/server/provider"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// TestAuthRPCsMapStoreUnavailableToClientErrors verifies auth-manager validation errors surface as client errors.
func TestAuthRPCsMapStoreUnavailableToClientErrors(parseT *testing.T) {
	parseServer := &chatServer{
		logger:      parseNewTestLogger(),
		sessions:    map[string]*sessionState{},
		authUsers:   map[string]authUser{},
		authManager: parseNewAuthManager("test-secret", nil, parseNewTestLogger()),
	}

	if _, parseErr := parseServer.Signup(context.Background(), &chatpb.SignupRequest{
		Email:    "signup@example.com",
		Password: "password123",
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("expected signup store-unavailable error to be InvalidArgument, got %v", status.Code(parseErr))
	}
	if _, parseErr := parseServer.Login(context.Background(), &chatpb.LoginRequest{
		Email:    "login@example.com",
		Password: "password123",
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("expected login store-unavailable error to be InvalidArgument, got %v", status.Code(parseErr))
	}
}

// TestGetSelectedModelRepairsUnsupportedPreference verifies unsupported stored models are repaired to a valid fallback.
func TestGetSelectedModelRepairsUnsupportedPreference(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "unsupported-model@example.com")
	parseFake := parseNewFakeProvider()
	parseFake.defaultModel = modelGPT54Mini
	parseFake.modelOptions = []provider.ModelOption{
		{
			ID:           modelGPT54,
			Label:        "GPT-5.4",
			Note:         "Best",
			Capabilities: parseFake.supportedModels[modelGPT54],
		},
	}
	parseServer := parseNewFakeChatServer(parseStore, parseFake)
	parseCtx := parseBindAuthUser(parseServer, "peer-unsupported-model", parseUser.ID, parseUser.Email)

	if parseErr := parseStore.setSelectedModel(parseUser.ID, "missing-model"); parseErr != nil {
		parseT.Fatalf("seed unsupported selected model: %v", parseErr)
	}
	parseSelectedModel, parseErr := parseServer.GetSelectedModel(parseCtx, &emptypb.Empty{})
	if parseErr != nil {
		parseT.Fatalf("GetSelectedModel unsupported repair: %v", parseErr)
	}
	if parseSelectedModel.GetValue() != modelGPT54 {
		parseT.Fatalf("expected repaired model %q, got %q", modelGPT54, parseSelectedModel.GetValue())
	}
	parsePersistedModel, parseErr := parseStore.getSelectedModel(parseUser.ID, "")
	if parseErr != nil {
		parseT.Fatalf("getSelectedModel after repair: %v", parseErr)
	}
	if parsePersistedModel != modelGPT54 {
		parseT.Fatalf("expected persisted repaired model %q, got %q", modelGPT54, parsePersistedModel)
	}
	parseServer.parseUnbindAuthenticatedPeer("peer-unsupported-model")
}

// TestPreferenceGettersReturnDefaultAndNormalizedValues verifies default and normalization getter branches.
func TestPreferenceGettersReturnDefaultAndNormalizedValues(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "preferences-defaults@example.com")
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseCtx := parseBindAuthUser(parseServer, "peer-preferences-defaults", parseUser.ID, parseUser.Email)

	parseEnabledResp, parseErr := parseServer.GetSelectedThinkingEnabled(parseCtx, &emptypb.Empty{})
	if parseErr != nil {
		parseT.Fatalf("GetSelectedThinkingEnabled default: %v", parseErr)
	}
	if parseEnabledResp.GetValue() != defaultThinkingEnabled {
		parseT.Fatalf("expected default thinking enabled %t, got %t", defaultThinkingEnabled, parseEnabledResp.GetValue())
	}

	if parseErr2 := parseStore.setSelectedThinkingEffort(parseUser.ID, "unsupported"); parseErr2 != nil {
		parseT.Fatalf("setSelectedThinkingEffort invalid seed: %v", parseErr2)
	}
	parseEffortResp, parseErr := parseServer.GetSelectedThinkingEffort(parseCtx, &emptypb.Empty{})
	if parseErr != nil {
		parseT.Fatalf("GetSelectedThinkingEffort normalized fallback: %v", parseErr)
	}
	if parseEffortResp.GetValue() != defaultThinkingEffort {
		parseT.Fatalf("expected normalized thinking effort %q, got %q", defaultThinkingEffort, parseEffortResp.GetValue())
	}

	parseServer.parseUnbindAuthenticatedPeer("peer-preferences-defaults")
}
