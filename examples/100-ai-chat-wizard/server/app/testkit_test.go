package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/server/provider"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/proto"
)

type stubAddr string

func (parseA stubAddr) Network() string { return "test" }

func (parseA stubAddr) String() string { return string(parseA) }

type fakeProvider struct {
	id                    string
	defaultModel          string
	supportedModels       map[string]provider.ModelCapabilities
	streamChat            func(context.Context, provider.ChatRequest, func(provider.ChatEvent) error) (provider.ChatResult, error)
	generateTitle         func(context.Context, provider.TitleRequest) (string, error)
	extractUserMemories   func(context.Context, provider.MemoryExtractionRequest) ([]provider.UserMemoryCandidate, error)
	synthesizeSpeech      func(context.Context, provider.SpeechRequest, func(provider.SpeechChunk) error) (provider.SpeechResult, error)
	modelOptions          []provider.ModelOption
	available             bool
	lastGenerateTitleReq  provider.TitleRequest
	lastMemoryExtractReq  provider.MemoryExtractionRequest
	lastSynthesizeReq     provider.SpeechRequest
	lastStreamChatRequest provider.ChatRequest
}

func (parseP *fakeProvider) ParseID() string { return parseP.id }

func (parseP *fakeProvider) ParseAvailable() bool { return parseP != nil && parseP.available }

func (parseP *fakeProvider) ParseInfo() provider.ProviderInfo {
	return provider.ProviderInfo{
		ID:                 parseP.id,
		Label:              "Fake",
		AuthConfigured:     parseP.available,
		Available:          parseP.available,
		StreamingSupported: true,
		ReasoningSupported: true,
		ToolUseSupported:   true,
	}
}

func (parseP *fakeProvider) ParseDefaultModel() string { return parseP.defaultModel }

func (parseP *fakeProvider) ParseSupportsModel(parseModel string) bool {
	_, parseOk := parseP.supportedModels[parseNormalizeSelectedModelID(parseModel)]
	return parseOk
}

func (parseP *fakeProvider) ParseModelOptions() []provider.ModelOption { return parseP.modelOptions }

func (parseP *fakeProvider) ParseModelMetadata(parseModel string) (provider.ModelMetadata, bool) {
	parseCapabilities := parseP.ParseCapabilities(parseModel)
	for _, parseOption := range parseP.modelOptions {
		if parseOption.ID != parseNormalizeSelectedModelID(parseModel) {
			continue
		}
		return provider.ModelMetadata{
			ID:                 parseOption.ID,
			DisplayName:        parseOption.Label,
			Description:        parseOption.Note,
			ProviderID:         parseP.id,
			ProviderLabel:      "Fake",
			ProviderFamily:     "fake",
			Capabilities:       parseCapabilities,
			StreamingSupported: true,
			ReasoningSupported: parseCapabilities.SupportsThinking,
			ToolUseSupported:   true,
			OnboardingReady:    true,
		}, true
	}
	return provider.ModelMetadata{}, false
}

func (parseP *fakeProvider) ParseCapabilities(parseModel string) provider.ModelCapabilities {
	if parseCapabilities, parseOk := parseP.supportedModels[parseNormalizeSelectedModelID(parseModel)]; parseOk {
		return parseCapabilities
	}
	return provider.ModelCapabilities{ProviderID: parseP.id, ProviderLabel: parseP.id}
}

func (parseP *fakeProvider) ParseHealth() provider.ProviderHealth {
	parseStatus := provider.ProviderHealthUnavailable
	if parseP.ParseAvailable() {
		parseStatus = provider.ProviderHealthUnknown
	}
	return provider.ProviderHealth{ProviderID: parseP.id, Status: parseStatus}
}

func (parseP *fakeProvider) ParseCurrentRateLimits() provider.RateLimitSnapshot {
	return provider.RateLimitSnapshot{}
}

func (parseP *fakeProvider) ParseStreamChat(parseCtx context.Context, parseReq provider.ChatRequest, parseEmit func(provider.ChatEvent) error) (provider.ChatResult, error) {
	parseP.lastStreamChatRequest = parseReq
	return parseP.streamChat(parseCtx, parseReq, parseEmit)
}

func (parseP *fakeProvider) ParseGenerateTitle(parseCtx context.Context, parseReq provider.TitleRequest) (string, error) {
	parseP.lastGenerateTitleReq = parseReq
	return parseP.generateTitle(parseCtx, parseReq)
}

func (parseP *fakeProvider) ParseExtractUserMemories(parseCtx context.Context, parseReq provider.MemoryExtractionRequest) ([]provider.UserMemoryCandidate, error) {
	parseP.lastMemoryExtractReq = parseReq
	return parseP.extractUserMemories(parseCtx, parseReq)
}

func (parseP *fakeProvider) ParseSynthesizeSpeech(parseCtx context.Context, parseReq provider.SpeechRequest, parseEmit func(provider.SpeechChunk) error) (provider.SpeechResult, error) {
	parseP.lastSynthesizeReq = parseReq
	return parseP.synthesizeSpeech(parseCtx, parseReq, parseEmit)
}

type fakeChatSendStream struct {
	ctx    context.Context
	chunks []*chatpb.ChatChunk
}

func (parseS *fakeChatSendStream) Send(parseChunk *chatpb.ChatChunk) error {
	if parseChunk == nil {
		parseS.chunks = append(parseS.chunks, nil)
		return nil
	}
	parseS.chunks = append(parseS.chunks, proto.Clone(parseChunk).(*chatpb.ChatChunk))
	return nil
}

func (parseS *fakeChatSendStream) SetHeader(metadata.MD) error { return nil }

func (parseS *fakeChatSendStream) SendHeader(metadata.MD) error { return nil }

func (parseS *fakeChatSendStream) SetTrailer(metadata.MD) {}

func (parseS *fakeChatSendStream) Context() context.Context { return parseS.ctx }

func (parseS *fakeChatSendStream) SendMsg(any) error { return nil }

func (parseS *fakeChatSendStream) RecvMsg(any) error { return nil }

type fakeSpeechStream struct {
	ctx    context.Context
	chunks []*chatpb.SynthesizeSpeechChunk
}

func (parseS *fakeSpeechStream) Send(parseChunk *chatpb.SynthesizeSpeechChunk) error {
	if parseChunk == nil {
		parseS.chunks = append(parseS.chunks, nil)
		return nil
	}
	parseS.chunks = append(parseS.chunks, proto.Clone(parseChunk).(*chatpb.SynthesizeSpeechChunk))
	return nil
}

func (parseS *fakeSpeechStream) SetHeader(metadata.MD) error { return nil }

func (parseS *fakeSpeechStream) SendHeader(metadata.MD) error { return nil }

func (parseS *fakeSpeechStream) SetTrailer(metadata.MD) {}

func (parseS *fakeSpeechStream) Context() context.Context { return parseS.ctx }

func (parseS *fakeSpeechStream) SendMsg(any) error { return nil }

func (parseS *fakeSpeechStream) RecvMsg(any) error { return nil }

func parseNewTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func parseNewTestStore(parseT *testing.T) *Store {
	parseT.Helper()
	store, parseErr := parseOpenChatStore(filepath.Join(parseT.TempDir(), "chat.db"))
	if parseErr != nil {
		parseT.Fatalf("openChatStore: %v", parseErr)
	}
	parseT.Cleanup(store.parseClose)
	return store
}

func parseNewAuthenticatedContext(parsePeerName string) context.Context {
	return peer.NewContext(context.Background(), &peer.Peer{Addr: stubAddr(parsePeerName)})
}

func parseBindAuthUser(parseServer *chatServer, parsePeerName string, parseUserID int64, parseEmail string) context.Context {
	parseCtx := parseNewAuthenticatedContext(parsePeerName)
	parseServer.parseBindAuthenticatedPeer(parsePeerName, authUser{ID: parseUserID, Email: parseEmail})
	return parseCtx
}

// parseGrantSuperuserRole grants the canonical superuser role to one test user id.
func parseGrantSuperuserRole(parseT *testing.T, parseStore *Store, parseUserID int64) {
	parseT.Helper()
	if parseErr := parseStore.parseUpsertSURole(parseSURoleWrite{
		RoleKey:     "su",
		Label:       "Superuser",
		Description: "Test-only superuser role",
		IsSystem:    true,
		IsEnabled:   true,
		Permissions: []parseSURolePermissionRow{
			{PermissionKey: "control_plane.*", PermissionValue: "allow"},
		},
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSURole: %v", parseErr)
	}
	if parseErr := parseStore.parseUpsertSUUserRole(parseUserID, "su", parseUserID); parseErr != nil {
		parseT.Fatalf("parseUpsertSUUserRole: %v", parseErr)
	}
}

func parseMustCreateUser(parseT *testing.T, store *Store, parseEmail string) authUser {
	parseT.Helper()
	parseUserID, parseErr := store.parseCreateUser(parseEmail, "hash", "Demo User")
	if parseErr != nil {
		parseT.Fatalf("createUser: %v", parseErr)
	}
	return authUser{ID: parseUserID, Email: parseNormalizeAuthEmail(parseEmail)}
}

// parseMustAssignBillingPlan seeds one active billing customer+subscription for a user.
func parseMustAssignBillingPlan(parseT *testing.T, parseStore *Store, parseUserID int64, parsePlanCode string) {
	parseT.Helper()
	parseNow := time.Now().UTC()
	parseCustomer, parseErr := parseStore.parseUpsertBillingCustomer(parseBillingCustomerWrite{
		UserID:             parseUserID,
		ProviderID:         "stripe",
		ProviderCustomerID: fmt.Sprintf("cus-test-%d-%s", parseUserID, strings.TrimSpace(parsePlanCode)),
		DefaultCurrency:    "usd",
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertBillingCustomer: %v", parseErr)
	}
	if _, parseErr2 := parseStore.parseUpsertBillingSubscription(parseBillingSubscriptionWrite{
		CustomerID:             parseCustomer.ID,
		ProviderID:             "stripe",
		ProviderSubscriptionID: fmt.Sprintf("sub-test-%d-%s", parseUserID, strings.TrimSpace(parsePlanCode)),
		PlanCode:               strings.TrimSpace(parsePlanCode),
		Status:                 "active",
		BillingInterval:        "month",
		CurrentPeriodStart:     parseNow.Format(time.RFC3339),
		CurrentPeriodEnd:       parseNow.Add(30 * 24 * time.Hour).Format(time.RFC3339),
	}); parseErr2 != nil {
		parseT.Fatalf("parseUpsertBillingSubscription: %v", parseErr2)
	}
}

// parseMustEnsureWorkspaceMembership seeds one active workspace membership for one user and returns the workspace id.
func parseMustEnsureWorkspaceMembership(parseT *testing.T, parseStore *Store, parseUserID int64, parseWorkspaceKey string) int64 {
	parseT.Helper()
	parseWorkspaceKey = parseNormalizeSUKey(parseWorkspaceKey)
	if parseWorkspaceKey == "" {
		parseWorkspaceKey = fmt.Sprintf("ws-user-%d", parseUserID)
	}
	if parseErr := parseStore.parseUpsertWorkspace(parseWorkspaceWrite{
		WorkspaceKey: parseWorkspaceKey,
		Slug:         parseWorkspaceKey,
		Name:         "Workspace " + parseWorkspaceKey,
		PlanCode:     "free",
		Status:       "active",
		OwnerUserID:  parseUserID,
		SettingsJSON: "{}",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspace: %v", parseErr)
	}
	parseWorkspaceRows, parseErr := parseStore.parseListWorkspaces(200)
	if parseErr != nil {
		parseT.Fatalf("parseListWorkspaces: %v", parseErr)
	}
	parseWorkspaceID := int64(0)
	for _, parseWorkspaceRow := range parseWorkspaceRows {
		if parseWorkspaceRow.WorkspaceKey != parseWorkspaceKey {
			continue
		}
		parseWorkspaceID = parseWorkspaceRow.ID
		break
	}
	if parseWorkspaceID <= 0 {
		parseT.Fatalf("workspace %q not found after upsert", parseWorkspaceKey)
	}
	if parseErr2 := parseStore.parseUpsertWorkspaceMembership(parseWorkspaceMembershipWrite{
		WorkspaceID:     parseWorkspaceID,
		UserID:          parseUserID,
		RoleKey:         "owner",
		Status:          "active",
		InvitedByUserID: parseUserID,
	}); parseErr2 != nil {
		parseT.Fatalf("parseUpsertWorkspaceMembership: %v", parseErr2)
	}
	return parseWorkspaceID
}

func parseNewFakeProvider() *fakeProvider {
	parseCapabilities := provider.ModelCapabilities{
		ProviderID:       "fake",
		ProviderLabel:    "Fake",
		SupportsThinking: true,
		SupportsSpeech:   true,
	}
	return &fakeProvider{
		id:           "fake",
		defaultModel: modelGPT54Mini,
		supportedModels: map[string]provider.ModelCapabilities{
			modelGPT54Mini: parseCapabilities,
			modelGPT54:     parseCapabilities,
		},
		modelOptions: []provider.ModelOption{{
			ID:           modelGPT54Mini,
			Label:        "GPT-5.4 mini",
			Note:         "Fast",
			Capabilities: parseCapabilities,
		}},
		available: true,
		extractUserMemories: func(context.Context, provider.MemoryExtractionRequest) ([]provider.UserMemoryCandidate, error) {
			return nil, nil
		},
	}
}

func parseNewFakeChatServer(store *Store, parseFake *fakeProvider) *chatServer {
	parseServer := &chatServer{
		providerRegistry:      provider.ParseNewRegistry(parseFake),
		defaultModel:          modelGPT54Mini,
		memoryExtractionModel: modelGPT54,
		store:                 store,
		logger:                parseNewTestLogger(),
		sessions:              map[string]*sessionState{},
		authUsers:             map[string]authUser{},
	}
	return parseServer
}

var _ net.Addr = stubAddr("")
