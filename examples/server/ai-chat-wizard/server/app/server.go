package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/server/provider"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const defaultToneID = "balanced"
const defaultThinkingEffort = "medium"
const defaultThinkingEnabled = true
const maxTTSScriptRunes = 4096
const maxCustomSystemPromptRunes = 4000
const maxInjectedUserMemoryCount = 5
const maxInjectedUserMemoryRunes = 1200
const userMemoryUsefulnessThreshold = 60
const userMemoryConfidenceThreshold = 0.55
const userMemoryExtractionTimeout = 8 * time.Second
const defaultCustomSystemPromptTemplate = `Current runtime context:
- Date: {{date}}
- Time: {{time}}
- Remembered user context:
{{memories}}
Use this context when relevant, but do not invent facts not present in user messages or memories.`

const thoughtChunkModelPrefix = "__thought_delta__:"
const thoughtChunkModelDone = "__thought_done__"

const (
	modelGPT54     = "gpt-5.4"
	modelGPT54Mini = "gpt-5.4-mini"
	modelGPT54Nano = "gpt-5.4-nano"
)

// gRPC service implementation.

type chatServer struct {
	chatpb.UnimplementedChatServiceServer
	providerRegistry                *provider.Registry
	defaultModel                    string
	runtimeEnvironment              string
	isParseBypassBillingModelPolicy bool
	store                           *Store
	logger                          *slog.Logger
	clientLogger                    *slog.Logger
	sessionsMutex                   sync.Mutex
	sessions                        map[string]*sessionState
	authMutex                       sync.RWMutex
	authUsers                       map[string]authUser
	authManager                     *authManager
	activeTTSStreams                atomic.Int64
	memoryExtractionSlots           chan struct{}
	memoryExtractionModel           string
	trackUsageBudgetMutex           sync.Mutex
	trackUsageBudgetWindowByUser    map[int64][]time.Time
	trackUsageBudgetActiveByUser    map[int64]int
}

// sessionState tracks the active SQLite conversation for one WebSocket peer.
type sessionState struct {
	userID            int64
	conversationID    int64
	savedMessageCount int // number of messages already persisted in this conversation
}

func parseNewChatServiceServer(parseOpenAIAPIKey, parseAnthropicAPIKey, parseCerebrasAPIKey, parseDefaultModel string, store *Store, parseLogger *slog.Logger, parseStubProviders ...string) *chatServer {
	parseDefaultModel = parseNormalizeSelectedModelID(parseDefaultModel)
	parseCatalogConfig, parseErr := parseLoadModelCatalogConfig(store)
	if parseErr != nil {
		parseLogger.Warn("chat provider catalog load failed", slog.String("error", parseErr.Error()))
	}
	parseStubSet := parseNormalizeStubProviders(parseStubProviders)
	isParseBypassBillingModelPolicy := len(parseStubSet) > 0
	parseProviderRegistry := provider.ParseNewRegistry(
		parseSelectRuntimeProvider("openai", strings.TrimSpace(parseOpenAIAPIKey), parseStubSet, parseCatalogConfig.ProviderCatalogs["openai"]),
		parseSelectRuntimeProvider("anthropic", strings.TrimSpace(parseAnthropicAPIKey), parseStubSet, parseCatalogConfig.ProviderCatalogs["anthropic"]),
		parseSelectRuntimeProvider("cerebras", strings.TrimSpace(parseCerebrasAPIKey), parseStubSet, parseCatalogConfig.ProviderCatalogs["cerebras"]),
	)
	if parseDefaultModel == "" {
		parseDefaultModel = parseNormalizeSelectedModelID(parseCatalogConfig.DefaultModel)
	}
	if _, parseResolvedModel, parseErr2 := parseProviderRegistry.ParseResolve(parseDefaultModel); parseErr2 == nil {
		parseDefaultModel = parseNormalizeSelectedModelID(parseResolvedModel)
	} else if _, parseResolvedModel2, parseFallbackErr := parseProviderRegistry.ParseResolve(""); parseFallbackErr == nil {
		parseDefaultModel = parseNormalizeSelectedModelID(parseResolvedModel2)
		parseLogger.Warn("chat provider default model override",
			slog.String("requested_model", strings.TrimSpace(parseDefaultModel)),
			slog.String("fallback_model", parseResolvedModel2),
			slog.String("error", parseErr2.Error()),
		)
	}
	parseChatService := &chatServer{
		providerRegistry:                parseProviderRegistry,
		defaultModel:                    parseDefaultModel,
		isParseBypassBillingModelPolicy: isParseBypassBillingModelPolicy,
		store:                           store,
		logger:                          parseLogger,
		sessions:                        make(map[string]*sessionState),
		authUsers:                       make(map[string]authUser),
		authManager:                     parseNewAuthManager("", store, parseLogger.With(slog.String("component", "auth"))),
		memoryExtractionSlots:           make(chan struct{}, 2),
		memoryExtractionModel:           parseNormalizeSelectedModelID(parseCatalogConfig.MemoryExtractionModel),
		trackUsageBudgetWindowByUser:    make(map[int64][]time.Time),
		trackUsageBudgetActiveByUser:    make(map[int64]int),
	}
	if parseChatService.memoryExtractionModel == "" {
		parseChatService.memoryExtractionModel = parseDefaultModel
	}
	return parseChatService
}

func parseNormalizeStubProviders(parseValues []string) map[string]struct{} {
	parseNormalized := map[string]struct{}{}
	for _, parseValue := range parseValues {
		for parseToken := range strings.SplitSeq(parseValue, ",") {
			parseResolved := strings.TrimSpace(strings.ToLower(parseToken))
			if parseResolved == "" {
				continue
			}
			if parseResolved == "all" {
				parseNormalized["openai"] = struct{}{}
				parseNormalized["anthropic"] = struct{}{}
				parseNormalized["cerebras"] = struct{}{}
				continue
			}
			parseNormalized[parseResolved] = struct{}{}
		}
	}
	return parseNormalized
}

func parseSelectRuntimeProvider(parseProviderID string, parseApiKey string, parseStubProviders map[string]struct{}, parseCatalog provider.Catalog) provider.ChatProvider {
	if strings.TrimSpace(parseApiKey) != "" {
		switch parseProviderID {
		case "openai":
			return provider.ParseNewOpenAIProvider(parseApiKey, parseCatalog)
		case "anthropic":
			return provider.ParseNewAnthropicProvider(parseApiKey, parseCatalog)
		case "cerebras":
			return provider.ParseNewCerebrasProvider(parseApiKey, parseCatalog)
		default:
			return nil
		}
	}
	if _, parseOk := parseStubProviders[strings.TrimSpace(parseProviderID)]; parseOk {
		return provider.ParseNewStubProvider(parseProviderID, parseCatalog)
	}
	switch parseProviderID {
	case "openai":
		return provider.ParseNewOpenAIProvider("", parseCatalog)
	case "anthropic":
		return provider.ParseNewAnthropicProvider("", parseCatalog)
	case "cerebras":
		return provider.ParseNewCerebrasProvider("", parseCatalog)
	default:
		return nil
	}
}

// loadOrCreateConversationSession returns the conversation ID and the number of messages
// already persisted for this exchange.
//
//   - requestedConversationID > 0: the client wants to continue a specific conversation that
//     already exists in the DB; all historyLength messages are already saved there.
//   - requestedConversationID == 0: start a fresh conversation row.
//     This covers empty new chats and branch/fork drafts that intentionally send
//     prior history into a new thread lineage.
func (parseS *chatServer) parseLoadOrCreateConversationSession(parsePeerAddress string, parseUserID, parseRequestedConversationID int64, parseHistoryLength int) (int64, int, error) {
	parseS.sessionsMutex.Lock()
	defer parseS.sessionsMutex.Unlock()

	if parseRequestedConversationID > 0 {
		parseOwned, parseErr := parseS.store.parseConversationOwnedByUser(parseUserID, parseRequestedConversationID)
		if parseErr != nil {
			return 0, 0, parseErr
		}
		if !parseOwned {
			return 0, 0, status.Error(codes.NotFound, "conversation no longer exists for authenticated user")
		}
		parseS.sessions[parsePeerAddress] = &sessionState{userID: parseUserID, conversationID: parseRequestedConversationID, savedMessageCount: parseHistoryLength}
		return parseRequestedConversationID, parseHistoryLength, nil
	}

	parseConversationID, parseErr2 := parseS.store.parseCreateConversation(parseUserID)
	if parseErr2 != nil {
		return 0, 0, parseErr2
	}
	parseS.sessions[parsePeerAddress] = &sessionState{userID: parseUserID, conversationID: parseConversationID, savedMessageCount: 0}
	return parseConversationID, 0, nil
}

func (parseS *chatServer) parseMarkConversationMessagesSaved(parsePeerAddress string, parseSavedMessageCount int) {
	parseS.sessionsMutex.Lock()
	defer parseS.sessionsMutex.Unlock()
	if parseSession, parseOk := parseS.sessions[parsePeerAddress]; parseOk {
		parseSession.savedMessageCount = parseSavedMessageCount
	}
}

// parsePersistFailedConversationMessages saves one failed exchange so reconnects do not leave an empty conversation shell behind.
func (parseS *chatServer) parsePersistFailedConversationMessages(parseLogger *slog.Logger, parsePeerAddress string, parseUserID, parseRequestedConversationID, parseConversationID int64, parseSavedMessageCount int, parseHistory []*chatpb.ChatMessage, parseUserMessage, parseAssistantMessage, parseResolvedModel string) {
	parsePersistedMessageCount := parseSavedMessageCount
	for parseHistoryIndex := parseSavedMessageCount; parseHistoryIndex < len(parseHistory); parseHistoryIndex++ {
		parseHistoryMessage := parseHistory[parseHistoryIndex]
		if parseDbErr := parseS.store.parseSaveConversationMessage(parseUserID, parseConversationID, parseHistoryMessage.Role, parseHistoryMessage.Content, parseHistoryMessage.GetModelId(), parseHistoryMessage.GetPromptTokens(), parseHistoryMessage.GetCompletionTokens()); parseDbErr != nil {
			parseLogger.Error("rpc.Send: db: save failed-stream history message failed",
				slog.Int64("user_id", parseUserID),
				slog.Int64("requested_conv_id", parseRequestedConversationID),
				slog.Int64("conv_id", parseConversationID),
				slog.Int("index", parseHistoryIndex),
				slog.String("role", parseHistoryMessage.Role),
				slog.String("error", parseDbErr.Error()),
			)
			if errors.Is(parseDbErr, errStoreConversationMissing) {
				parseS.clearPeerSession(parsePeerAddress)
			}
			return
		}
		parsePersistedMessageCount++
	}
	if parseDbErr := parseS.store.parseSaveConversationMessage(parseUserID, parseConversationID, "user", parseUserMessage, "", 0, 0); parseDbErr != nil {
		parseLogger.Error("rpc.Send: db: save failed-stream user message failed",
			slog.Int64("user_id", parseUserID),
			slog.Int64("requested_conv_id", parseRequestedConversationID),
			slog.Int64("conv_id", parseConversationID),
			slog.String("error", parseDbErr.Error()),
		)
		if errors.Is(parseDbErr, errStoreConversationMissing) {
			parseS.clearPeerSession(parsePeerAddress)
		}
		return
	}
	parsePersistedMessageCount++
	if strings.TrimSpace(parseAssistantMessage) != "" {
		if parseDbErr := parseS.store.parseSaveConversationMessage(parseUserID, parseConversationID, "assistant", parseAssistantMessage, parseResolvedModel, 0, 0); parseDbErr != nil {
			parseLogger.Error("rpc.Send: db: save failed-stream assistant message failed",
				slog.Int64("user_id", parseUserID),
				slog.Int64("requested_conv_id", parseRequestedConversationID),
				slog.Int64("conv_id", parseConversationID),
				slog.String("resolved_model", parseResolvedModel),
				slog.String("error", parseDbErr.Error()),
			)
			if errors.Is(parseDbErr, errStoreConversationMissing) {
				parseS.clearPeerSession(parsePeerAddress)
			}
			return
		}
		parsePersistedMessageCount++
	}
	parseS.parseMarkConversationMessagesSaved(parsePeerAddress, parsePersistedMessageCount)
}

func (parseS *chatServer) parseBindAuthenticatedPeer(parsePeerAddress string, parseUser authUser) {
	parseS.authMutex.Lock()
	defer parseS.authMutex.Unlock()
	parseS.authUsers[parsePeerAddress] = parseUser
}

func (parseS *chatServer) parseUnbindAuthenticatedPeer(parsePeerAddress string) {
	parseS.authMutex.Lock()
	defer parseS.authMutex.Unlock()
	delete(parseS.authUsers, parsePeerAddress)

	parseS.clearPeerSession(parsePeerAddress)
}

func (parseS *chatServer) clearPeerSession(parsePeerAddress string) {
	parseS.sessionsMutex.Lock()
	defer parseS.sessionsMutex.Unlock()
	delete(parseS.sessions, parsePeerAddress)
}

// parsePeerAddressFromContext resolves one stable peer address from one RPC context when available.
func parsePeerAddressFromContext(parseCtx context.Context) (string, bool) {
	parsePeerInfo, parseOk := peer.FromContext(parseCtx)
	if !parseOk || parsePeerInfo.Addr == nil {
		return "", false
	}
	parsePeerAddress := strings.TrimSpace(parsePeerInfo.Addr.String())
	if parsePeerAddress == "" {
		return "", false
	}
	return parsePeerAddress, true
}

func (parseS *chatServer) parseAuthenticatedUserFromContext(parseCtx context.Context) (authUser, bool) {
	if parseS.authManager != nil {
		if parseUser, parseOk := parseS.authManager.parseAuthenticatedUserFromContext(parseCtx); parseOk && parseUser.ID > 0 {
			return parseUser, true
		}
		if _, parseTokenPresent := parseResolveAuthTokenFromContext(parseCtx); parseTokenPresent {
			if parsePeerAddress, parseHasPeerAddress := parsePeerAddressFromContext(parseCtx); parseHasPeerAddress {
				parseS.parseUnbindAuthenticatedPeer(parsePeerAddress)
			}
			return authUser{}, false
		}
	}
	parsePeerAddress, parseHasPeerAddress := parsePeerAddressFromContext(parseCtx)
	if !parseHasPeerAddress {
		return authUser{}, false
	}

	parseS.authMutex.RLock()
	parseUser2, parseFoundUser := parseS.authUsers[parsePeerAddress]
	parseS.authMutex.RUnlock()
	if !parseFoundUser || parseUser2.ID <= 0 {
		return authUser{}, false
	}
	return parseUser2, true
}

func (parseS *chatServer) parseRequireAuthenticatedUserID(parseCtx context.Context) (int64, error) {
	parseAuthUser, parseOk := parseS.parseAuthenticatedUserFromContext(parseCtx)
	if !parseOk || parseAuthUser.ID <= 0 {
		return 0, status.Error(codes.Unauthenticated, "authentication required")
	}
	return parseAuthUser.ID, nil
}

func parseStatusForStoreGuard(parseErr error) error {
	switch {
	case errors.Is(parseErr, errStoreUserMissing):
		return status.Error(codes.Unauthenticated, "authenticated user no longer exists; sign in again")
	case errors.Is(parseErr, errStoreConversationMissing):
		return status.Error(codes.FailedPrecondition, "conversation no longer exists; refresh and retry")
	case errors.Is(parseErr, errStoreUserDisabled), errors.Is(parseErr, errStoreUserAuthBlocked):
		return status.Error(codes.PermissionDenied, "user access is blocked")
	case errors.Is(parseErr, errStoreWorkspaceSuspended):
		return status.Error(codes.PermissionDenied, "workspace is suspended")
	default:
		return parseErr
	}
}

func isStoreGuardError(parseErr error) bool {
	return errors.Is(parseErr, errStoreUserMissing) ||
		errors.Is(parseErr, errStoreConversationMissing) ||
		errors.Is(parseErr, errStoreUserDisabled) ||
		errors.Is(parseErr, errStoreUserAuthBlocked) ||
		errors.Is(parseErr, errStoreWorkspaceSuspended)
}

// generateAndSaveConversationTitle turns the first exchange into a concise topic label so conversation history stays scannable.
func (parseS *chatServer) parseGenerateAndSaveConversationTitle(parseUserID, parseConversationID int64, parseModel, parseUserMessage, parseAssistantMessage string) {
	parseLogger := parseS.logger.With(slog.String("op", "generateTitle"), slog.Int64("conv_id", parseConversationID))
	if parseS.providerRegistry == nil || parseS.store == nil {
		return
	}

	// Truncate inputs so the title-gen call is fast and cheap.
	parseTruncateText := func(parseText string, parseMaxLength int) string {
		if len(parseText) <= parseMaxLength {
			return parseText
		}
		return parseText[:parseMaxLength] + "..."
	}
	parseTitlePrompt := "User: " + parseTruncateText(parseUserMessage, 400) + "\n\nAssistant: " + parseTruncateText(parseAssistantMessage, 400)

	parseCtx, parseCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer parseCancel()

	parseChatProvider, _, parseErr := parseS.providerRegistry.ParseResolve(parseModel)
	if parseErr != nil {
		parseLogger.Warn("generateTitle: provider unavailable", slog.String("model", parseModel), slog.String("error", parseErr.Error()))
		return
	}
	parseTitle, parseErr := parseChatProvider.ParseGenerateTitle(parseCtx, provider.TitleRequest{
		Model: parseModel,
		SystemPrompt: "You are a conversation title generator. " +
			"Given the first user message and assistant reply, produce a clear and concise title of 3-7 words " +
			"that captures the topic. Return only the title with no punctuation at the end, no quotes, and no explanation.",
		Prompt: parseTitlePrompt,
	})
	if parseErr != nil {
		parseLogger.Error("generateTitle: provider call failed", slog.String("model", parseModel), slog.String("error", parseErr.Error()))
		return
	}
	if parseTitle == "" {
		parseLogger.Warn("generateTitle: empty title returned")
		return
	}
	if parseErr2 := parseS.store.parseSaveConversationTitle(parseUserID, parseConversationID, parseTitle); parseErr2 != nil {
		parseLogger.Error("generateTitle: db save failed", slog.String("error", parseErr2.Error()))
		return
	}
	parseLogger.Info("generateTitle: saved", slog.String("title", parseTitle))
}

// Send keeps auth checks, billing policy, optimistic UI state, and provider streaming in one path so paid turns either complete cleanly or fail fast.
// gRPC client via the tunnel.
func (parseS *chatServer) Send(parseReq *chatpb.SendRequest, parseStream chatpb.ChatService_SendServer) error {
	parsePeerAddress := ""
	if parseP, parseOk := peer.FromContext(parseStream.Context()); parseOk {
		parsePeerAddress = parseP.Addr.String()
	}
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseStream.Context())
	if parseErr != nil {
		return parseErr
	}

	parseMessagePreview := parseReq.GetMessage()
	if len(parseMessagePreview) > 80 {
		parseMessagePreview = parseMessagePreview[:80] + "..."
	}

	parseLogger := parseS.logger.With(
		slog.String("peer", parsePeerAddress),
		slog.String("model", parseReq.GetModel()),
		slog.Int("history_len", len(parseReq.History)),
		slog.Int64("conv_id_req", parseReq.GetConversationId()),
	)
	parseLogger = parseLogger.With(parseBuildLogFieldAttrs(parseStream.Context(), parseLogFieldSpec{
		ParseRPC:         "Send",
		ParseActorScope:  "user",
		ParseActorUserID: parseUserID,
		ParseTargetScope: "conversation",
		ParseTargetID:    fmt.Sprintf("%d", parseReq.GetConversationId()),
	})...)
	parseLogger.Info("rpc.Send: started", slog.String("message_preview", parseMessagePreview))
	parseClientID := parseResolveClientIdentityFromContext(parseStream.Context())
	parseTraceID, parseSpanID, parseTraceState := parseExtractTraceContextFromContext(parseStream.Context())
	if parseS.store != nil {
		// Fail fast on stale auth or cross-conversation access before provider work starts.
		parseUserExists, parseUserErr := parseS.store.parseUserExists(parseUserID)
		if parseUserErr != nil {
			parseLogger.Error("rpc.Send: authenticated user lookup failed",
				slog.Int64("user_id", parseUserID),
				slog.String("error", parseUserErr.Error()),
			)
			return status.Errorf(codes.Internal, "lookup authenticated user: %v", parseUserErr)
		}
		if !parseUserExists {
			parseLogger.Warn("rpc.Send: authenticated user missing from store",
				slog.Int64("user_id", parseUserID),
				slog.String("peer", parsePeerAddress),
			)
			return status.Error(codes.Unauthenticated, "authenticated user no longer exists; sign in again")
		}
		if parseRuntimeErr := parseS.parseRequireChatRuntimeAccess(parseUserID); parseRuntimeErr != nil {
			parseLogger.Warn("rpc.Send: runtime access denied",
				slog.Int64("user_id", parseUserID),
				slog.String("code", status.Code(parseRuntimeErr).String()),
				slog.String("error", parseRuntimeErr.Error()),
			)
			return parseRuntimeErr
		}
		if parseReq.GetConversationId() > 0 {
			parseOwned, parseOwnedErr := parseS.store.parseConversationOwnedByUser(parseUserID, parseReq.GetConversationId())
			if parseOwnedErr != nil {
				parseLogger.Error("rpc.Send: requested conversation lookup failed",
					slog.Int64("user_id", parseUserID),
					slog.Int64("requested_conv_id", parseReq.GetConversationId()),
					slog.String("error", parseOwnedErr.Error()),
				)
				return status.Errorf(codes.Internal, "lookup conversation: %v", parseOwnedErr)
			}
			if !parseOwned {
				parseLogger.Warn("rpc.Send: requested conversation missing or inaccessible",
					slog.Int64("user_id", parseUserID),
					slog.Int64("requested_conv_id", parseReq.GetConversationId()),
				)
				return status.Error(codes.NotFound, "conversation no longer exists for authenticated user")
			}
		}
	}

	parseCustomSystemPrompt := defaultCustomSystemPromptTemplate
	var parseInjectedUserMemories []userMemoryRow
	if parseS.store != nil {
		parseStoredPrompt, parsePromptErr := parseS.store.getSelectedSystemPrompt(parseUserID, defaultCustomSystemPromptTemplate)
		if parsePromptErr != nil {
			parseLogger.Warn("rpc.Send: custom system prompt lookup failed", slog.String("error", parsePromptErr.Error()))
		} else {
			parseCustomSystemPrompt = parseStoredPrompt
		}
		parseStoredMemories, parseMemoryErr := parseS.store.parseListUserMemories(parseUserID)
		if parseMemoryErr != nil {
			parseLogger.Warn("rpc.Send: user memory lookup failed", slog.String("error", parseMemoryErr.Error()))
		} else {
			parseInjectedUserMemories = parseStoredMemories
		}
	}

	// Assemble the final prompt from defaults, user overrides, and remembered preferences.
	parseSystemPrompt := buildSystemPrompt(parseReq.GetTone(), parseCustomSystemPrompt, parseInjectedUserMemories)

	// Resolve plan limits up front so the streamed response never bypasses workspace billing rules.
	parseRequestedModel := parseNormalizeOptionalSelectedModelID(parseReq.GetModel())
	parseBillingPolicy, parseBillingPolicyErr := parseS.parseResolveBillingModelPolicy(parseUserID, time.Now().UTC())
	if parseBillingPolicyErr != nil {
		parseLogger.Error("rpc.Send: billing model policy lookup failed", slog.String("error", parseBillingPolicyErr.Error()))
		return parseBuildSanitizedInternalStatus(parseStream.Context(), "resolve billing model policy", parseBillingPolicyErr)
	}
	parseFallbackModel := parseNormalizeSelectedModelID(parseS.defaultModel)
	if parseRequestedModel == "" && parseBillingPolicy.DefaultModelID != "" {
		parseFallbackModel = parseNormalizeSelectedModelID(parseBillingPolicy.DefaultModelID)
	}
	parseResolvedModel, isParseRequestedModelDenied := parseResolvePlanAwareModel(parseRequestedModel, parseFallbackModel, parseBillingPolicy)
	if isParseRequestedModelDenied {
		parseDeniedMessage := parseFormatModelDeniedByPlan(parseRequestedModel, parseBillingPolicy)
		parseLogger.Warn("rpc.Send: requested model denied by billing policy",
			slog.Int64("user_id", parseUserID),
			slog.String("requested_model", parseRequestedModel),
			slog.String("plan_code", parseBillingPolicy.PlanCode),
			slog.String("message", parseDeniedMessage),
		)
		return status.Error(codes.FailedPrecondition, parseDeniedMessage)
	}
	if parseRequestedModel == "" && parseS.store != nil && parseResolvedModel != "" && parseResolvedModel != parseFallbackModel {
		if parseSetErr := parseS.store.setSelectedModel(parseUserID, parseResolvedModel); parseSetErr != nil {
			parseLogger.Warn("rpc.Send: failed to persist auto-selected billing model", slog.String("error", parseSetErr.Error()), slog.String("model", parseResolvedModel))
		}
	}

	// Open the provider stream only after caller, workspace, and plan checks pass.
	parseLogger.Debug("rpc.Send: opening OpenAI stream",
		slog.String("requested_model", parseRequestedModel),
		slog.String("resolved_model", parseResolvedModel),
		slog.String("plan_code", parseBillingPolicy.PlanCode),
		slog.String("tone", parseReq.GetTone()),
		slog.Bool("thinking_enabled", parseReq.GetThinkingEnabled()),
		slog.String("thinking_effort", parseReq.GetThinkingEffort()),
	)
	var parseChatProvider provider.ChatProvider
	if parseReq.GetThinkingEnabled() {
		parseChatProvider, parseResolvedModel, _, parseErr = parseS.providerRegistry.ParseRequireCapability(parseResolvedModel, provider.CapabilityThinking)
	} else {
		parseChatProvider, parseResolvedModel, parseErr = parseS.providerRegistry.ParseResolve(parseResolvedModel)
	}
	if parseErr != nil {
		parseLogger.Error("rpc.Send: provider resolution failed",
			slog.String("requested_model", parseRequestedModel),
			slog.String("resolved_model", parseResolvedModel),
			slog.String("error", parseErr.Error()),
		)
		if parseCapabilityErr := parseCapabilityStatusError(parseErr); parseCapabilityErr != nil {
			return parseCapabilityErr
		}
		if parseRequestedModel != "" {
			return status.Errorf(codes.InvalidArgument, "unsupported model %q", parseRequestedModel)
		}
		return status.Error(codes.Unavailable, "no configured model provider available")
	}
	parseLogger = parseLogger.With(parseBuildLogFieldAttrs(parseStream.Context(), parseLogFieldSpec{
		ParseProvider: parseChatProvider.ParseID(),
	})...)

	parseHistory := make([]provider.ChatMessage, 0, len(parseReq.History))
	for _, parseHistoryMessage := range parseReq.History {
		parseHistory = append(parseHistory, provider.ChatMessage{
			Role:    provider.ParseNormalizeRole(parseHistoryMessage.GetRole()),
			Content: parseHistoryMessage.GetContent(),
		})
	}
	if parseErr = parseS.parseRequireUserEntitlement(parseUserID, billingEntitlementChatSendEnabled); parseErr != nil {
		parseSendAccessErr := parseBuildSendAccessDeniedStatus(parseErr, parseBillingPolicy.PlanCode)
		parseLogger.Warn("rpc.Send: entitlement gate denied send",
			slog.Int64("user_id", parseUserID),
			slog.String("plan_code", parseBillingPolicy.PlanCode),
			slog.String("status_code", status.Code(parseSendAccessErr).String()),
			slog.String("error", parseSendAccessErr.Error()),
		)
		return parseSendAccessErr
	}
	parseReleaseUsageBudget := func() {}
	if parseReleaseUsageBudget, parseErr = parseS.parseRequireUsageBudget(parseUserID); parseErr != nil {
		parseSendAccessErr := parseBuildSendAccessDeniedStatus(parseErr, parseBillingPolicy.PlanCode)
		parseLogger.Warn("rpc.Send: usage budget gate denied send",
			slog.Int64("user_id", parseUserID),
			slog.String("plan_code", parseBillingPolicy.PlanCode),
			slog.String("status_code", status.Code(parseSendAccessErr).String()),
			slog.String("error", parseSendAccessErr.Error()),
		)
		return parseSendAccessErr
	}
	defer parseReleaseUsageBudget()
	parseS.parseTrackFirstChatFunnelStep(parseStream.Context(), parseUserID, parseFirstChatStepFirstSendStarted, map[string]any{
		"requested_conversation_id": parseReq.GetConversationId(),
		"history_len":               len(parseReq.History),
		"model":                     parseResolvedModel,
	})

	var parseAssistantResponseBuffer strings.Builder
	var parsePromptTokenCount int64
	var parseCompletionTokenCount int64
	var parseConversationID int64
	parseProviderRequestID := ""
	parseUsageSource := provider.UsageSourceMissing
	parseUsageEventID := ""
	parseUsageTotalCostUSD := 0.0
	isParseUsagePersisted := false
	parseSendThoughtDelta := func(parseDeltaText string) error {
		if parseDeltaText == "" {
			return nil
		}
		if parseErr2 := parseStream.Send(&chatpb.ChatChunk{Model: thoughtChunkModelPrefix + parseDeltaText}); parseErr2 != nil {
			parseLogger.Error("rpc.Send: downstream thought stream send failed",
				slog.String("error", parseErr2.Error()),
			)
			return status.Errorf(codes.Canceled, "thought stream send: %v", parseErr2)
		}
		return nil
	}
	isParseThoughtDoneSent := false
	parseSendThoughtDone := func() error {
		if isParseThoughtDoneSent {
			return nil
		}
		isParseThoughtDoneSent = true
		if parseErr3 := parseStream.Send(&chatpb.ChatChunk{Model: thoughtChunkModelDone}); parseErr3 != nil {
			parseLogger.Error("rpc.Send: downstream thought complete send failed",
				slog.String("error", parseErr3.Error()),
			)
			return status.Errorf(codes.Canceled, "thought complete send: %v", parseErr3)
		}
		return nil
	}
	parseChatResult, parseErr := parseChatProvider.ParseStreamChat(parseStream.Context(), provider.ChatRequest{
		Model:           parseResolvedModel,
		SystemPrompt:    parseSystemPrompt,
		History:         parseHistory,
		UserMessage:     parseReq.Message,
		ThinkingEnabled: parseReq.GetThinkingEnabled(),
		ThinkingEffort:  parseReq.GetThinkingEffort(),
	}, func(parseEvent provider.ChatEvent) error {
		if parseEvent.ThoughtDelta != "" {
			if parseErr4 := parseSendThoughtDelta(parseEvent.ThoughtDelta); parseErr4 != nil {
				return parseErr4
			}
		}
		if parseEvent.TextDelta != "" {
			parseAssistantResponseBuffer.WriteString(parseEvent.TextDelta)
			if parseErr5 := parseStream.Send(&chatpb.ChatChunk{Delta: parseEvent.TextDelta}); parseErr5 != nil {
				parseLogger.Error("rpc.Send: downstream stream send failed",
					slog.String("error", parseErr5.Error()),
				)
				return status.Errorf(codes.Canceled, "stream send: %v", parseErr5)
			}
		}
		if parseEvent.ThoughtDone {
			if parseErr6 := parseSendThoughtDone(); parseErr6 != nil {
				return parseErr6
			}
		}
		return nil
	})
	if parseErr != nil {
		parseLogger.Error("rpc.Send: provider stream error",
			slog.String("error", parseErr.Error()),
			slog.String("resolved_model", parseResolvedModel),
			slog.String("provider", parseChatProvider.ParseID()),
		)
		parseErrorConversationID := parseReq.GetConversationId()
		parseErrorUsageSource := provider.UsageSourceMissing
		parseErrorCompletionTokens := int64(0)
		parseErrorUsageEventID := ""
		parseErrorUsageTotalCostUSD := 0.0
		isParseErrorUsagePersisted := false
		if strings.TrimSpace(parseAssistantResponseBuffer.String()) != "" {
			parseErrorCompletionTokens = int64(len(strings.Fields(parseAssistantResponseBuffer.String()))) * 6
			parseErrorUsageSource = provider.UsageSourceEstimated
		}
		// Keep a failed usage ledger record so billing reconciliation can account for partial/error streams.
		if parseS.store != nil {
			parseSessionConversationID, parseSavedMessageCount, parseSessionErr := parseS.parseLoadOrCreateConversationSession(parsePeerAddress, parseUserID, parseReq.GetConversationId(), len(parseReq.History))
			if parseSessionErr != nil {
				parseLogger.Warn("rpc.Send: db: failed usage session resolution failed",
					slog.Int64("user_id", parseUserID),
					slog.Int64("requested_conv_id", parseReq.GetConversationId()),
					slog.String("error", parseSessionErr.Error()),
				)
			} else {
				parseConversationID = parseSessionConversationID
				parseErrorConversationID = parseSessionConversationID
				parseS.parsePersistFailedConversationMessages(
					parseLogger,
					parsePeerAddress,
					parseUserID,
					parseReq.GetConversationId(),
					parseSessionConversationID,
					parseSavedMessageCount,
					parseReq.History,
					parseReq.Message,
					parseAssistantResponseBuffer.String(),
					parseResolvedModel,
				)
				parseErrorUsage := parseUsageEventWrite{
					EventID:           uuid.NewString(),
					UserID:            parseUserID,
					ConversationID:    parseSessionConversationID,
					ProviderID:        parseChatProvider.ParseID(),
					ModelID:           parseResolvedModel,
					PromptTokens:      0,
					CompletionTokens:  parseErrorCompletionTokens,
					UsageSource:       parseErrorUsageSource,
					ProviderRequestID: parseProviderRequestID,
					PricingCurrency:   "USD",
					ClientID:          parseClientID,
					TraceID:           parseTraceID,
					SpanID:            parseSpanID,
					TraceState:        parseTraceState,
					Status:            "failed",
					ErrorMessage:      parseErr.Error(),
				}
				_, parsePricing, hasParsePricing, parsePricingErr := parseS.store.parseGetModelPricingForModel(parseResolvedModel)
				if parsePricingErr != nil {
					parseLogger.Warn("rpc.Send: failed usage pricing lookup failed",
						slog.Int64("conv_id", parseSessionConversationID),
						slog.String("resolved_model", parseResolvedModel),
						slog.String("error", parsePricingErr.Error()),
					)
				}
				if hasParsePricing {
					parseErrorUsage.InputCostPerMillionUSD = parsePricing.InputPerMillionUSD
					parseErrorUsage.OutputCostPerMillionUSD = parsePricing.OutputPerMillionUSD
					parseErrorUsage.PricingCurrency = parsePricing.Currency
					parseErrorUsageCost := provider.ParseEstimateCost(0, parseErrorCompletionTokens, parsePricing)
					parseErrorUsage.InputCostUSD = parseErrorUsageCost.InputCostUSD
					parseErrorUsage.OutputCostUSD = parseErrorUsageCost.OutputCostUSD
					parseErrorUsage.TotalCostUSD = parseErrorUsageCost.TotalCostUSD
					parseErrorUsageTotalCostUSD = parseErrorUsageCost.TotalCostUSD
				}
				if parseUsageSaveErr := parseS.store.parseSaveUsageEvent(parseErrorUsage); parseUsageSaveErr != nil {
					parseLogger.Error("rpc.Send: db: save failed usage event failed",
						slog.Int64("user_id", parseUserID),
						slog.Int64("conv_id", parseSessionConversationID),
						slog.String("usage_event_id", parseErrorUsage.EventID),
						slog.String("provider", parseErrorUsage.ProviderID),
						slog.String("resolved_model", parseResolvedModel),
						slog.String("error", parseUsageSaveErr.Error()),
					)
				} else {
					parseErrorUsageEventID = parseErrorUsage.EventID
					isParseErrorUsagePersisted = true
				}
			}
		}
		if parseThoughtDoneErr := parseSendThoughtDone(); parseThoughtDoneErr != nil {
			return parseThoughtDoneErr
		}
		if parseStreamErr := parseStream.Send(&chatpb.ChatChunk{
			Done:             true,
			Error:            parseUserFacingStreamError(parseChatProvider.ParseID(), parseResolvedModel, parseErr),
			ConversationId:   parseErrorConversationID,
			Model:            parseResolvedModel,
			CompletionTokens: parseErrorCompletionTokens,
			UsageEventId:     parseErrorUsageEventID,
			ProviderId:       parseChatProvider.ParseID(),
			TotalCostUsd:     parseErrorUsageTotalCostUSD,
			UsageSource:      parseErrorUsageSource,
			UsagePersisted:   isParseErrorUsagePersisted,
		}); parseStreamErr != nil {
			return status.Errorf(codes.Canceled, "stream error send: %v", parseStreamErr)
		}
		return nil
	}
	if parseChatResult.Model != "" {
		parseResolvedModel = parseNormalizeSelectedModelID(parseChatResult.Model)
	}
	parsePromptTokenCount = parseChatResult.PromptTokens
	parseCompletionTokenCount = parseChatResult.CompletionTokens
	parseProviderRequestID = strings.TrimSpace(parseChatResult.ProviderRequestID)
	if parseChatUsageSource := strings.TrimSpace(parseChatResult.UsageSource); parseChatUsageSource != "" {
		parseUsageSource = parseChatUsageSource
	}
	if parseUsageSource == provider.UsageSourceMissing && (parsePromptTokenCount > 0 || parseCompletionTokenCount > 0) {
		parseUsageSource = provider.UsageSourceExact
	}

	parseLogger.Debug("rpc.Send: provider stream complete",
		slog.String("provider", parseChatProvider.ParseID()),
		slog.String("resolved_model", parseResolvedModel),
	)
	// Persist the exchange.
	isParseFirstExchange := false
	if parseS.store != nil {
		var parseSavedMessageCount int
		parseConversationID, parseSavedMessageCount, parseErr = parseS.parseLoadOrCreateConversationSession(parsePeerAddress, parseUserID, parseReq.GetConversationId(), len(parseReq.History))
		if parseErr != nil {
			parseLogger.Error("rpc.Send: db: loadOrCreateConversationSession failed",
				slog.Int64("user_id", parseUserID),
				slog.Int64("requested_conv_id", parseReq.GetConversationId()),
				slog.String("error", parseErr.Error()),
			)
			if isStoreGuardError(parseErr) {
				return parseStatusForStoreGuard(parseErr)
			}
			return parseErr
		} else {
			isParseFirstExchange = parseSavedMessageCount == 0
			parseLogger.Debug("rpc.Send: db: session resolved",
				slog.Int64("user_id", parseUserID),
				slog.Int64("conv_id", parseConversationID),
				slog.Int("already_saved", parseSavedMessageCount),
				slog.Int("history_len", len(parseReq.History)),
			)
			if parseReq.GetConversationId() <= 0 {
				parseS.parseTrackFirstChatFunnelStep(parseStream.Context(), parseUserID, parseFirstChatStepFirstThreadCreate, map[string]any{
					"conversation_id": parseConversationID,
					"history_len":     len(parseReq.History),
				})
			}
			// Save any history messages not yet persisted.
			for parseHistoryIndex := parseSavedMessageCount; parseHistoryIndex < len(parseReq.History); parseHistoryIndex++ {
				parseHistoryMessage2 := parseReq.History[parseHistoryIndex]
				if parseDbErr := parseS.store.parseSaveConversationMessage(parseUserID, parseConversationID, parseHistoryMessage2.Role, parseHistoryMessage2.Content, parseHistoryMessage2.GetModelId(), parseHistoryMessage2.GetPromptTokens(), parseHistoryMessage2.GetCompletionTokens()); parseDbErr != nil {
					parseLogger.Error("rpc.Send: db: save history message failed",
						slog.Int64("user_id", parseUserID),
						slog.Int64("requested_conv_id", parseReq.GetConversationId()),
						slog.Int64("conv_id", parseConversationID),
						slog.Int("index", parseHistoryIndex),
						slog.String("role", parseHistoryMessage2.Role),
						slog.String("error", parseDbErr.Error()),
					)
					if errors.Is(parseDbErr, errStoreConversationMissing) {
						parseS.clearPeerSession(parsePeerAddress)
						break
					}
				}
			}
			if parseDbErr2 := parseS.store.parseSaveConversationMessage(parseUserID, parseConversationID, "user", parseReq.Message, "", 0, 0); parseDbErr2 != nil {
				parseLogger.Error("rpc.Send: db: save user message failed",
					slog.Int64("user_id", parseUserID),
					slog.Int64("requested_conv_id", parseReq.GetConversationId()),
					slog.Int64("conv_id", parseConversationID),
					slog.String("error", parseDbErr2.Error()),
				)
				if errors.Is(parseDbErr2, errStoreConversationMissing) {
					parseS.clearPeerSession(parsePeerAddress)
				}
			}
			if parseDbErr3 := parseS.store.parseSaveConversationMessage(parseUserID, parseConversationID, "assistant", parseAssistantResponseBuffer.String(), parseResolvedModel, parsePromptTokenCount, parseCompletionTokenCount); parseDbErr3 != nil {
				parseLogger.Error("rpc.Send: db: save assistant message failed",
					slog.Int64("user_id", parseUserID),
					slog.Int64("requested_conv_id", parseReq.GetConversationId()),
					slog.Int64("conv_id", parseConversationID),
					slog.String("resolved_model", parseResolvedModel),
					slog.Int64("prompt_tokens", parsePromptTokenCount),
					slog.Int64("completion_tokens", parseCompletionTokenCount),
					slog.String("error", parseDbErr3.Error()),
				)
				if errors.Is(parseDbErr3, errStoreConversationMissing) {
					parseS.clearPeerSession(parsePeerAddress)
				}
			}
			// Snapshot usage + pricing at completion time so billing can be audited later even if catalog prices change.
			parseUsageEvent := parseUsageEventWrite{
				EventID:           uuid.NewString(),
				UserID:            parseUserID,
				ConversationID:    parseConversationID,
				ProviderID:        parseChatProvider.ParseID(),
				ModelID:           parseResolvedModel,
				PromptTokens:      parsePromptTokenCount,
				CompletionTokens:  parseCompletionTokenCount,
				UsageSource:       parseUsageSource,
				ProviderRequestID: parseProviderRequestID,
				PricingCurrency:   "USD",
				ClientID:          parseClientID,
				TraceID:           parseTraceID,
				SpanID:            parseSpanID,
				TraceState:        parseTraceState,
				Status:            "completed",
			}
			_, parsePricing, hasParsePricing, parsePricingErr := parseS.store.parseGetModelPricingForModel(parseResolvedModel)
			if parsePricingErr != nil {
				parseLogger.Warn("rpc.Send: usage pricing lookup failed",
					slog.Int64("conv_id", parseConversationID),
					slog.String("resolved_model", parseResolvedModel),
					slog.String("error", parsePricingErr.Error()),
				)
			}
			if hasParsePricing {
				parseUsageEvent.InputCostPerMillionUSD = parsePricing.InputPerMillionUSD
				parseUsageEvent.OutputCostPerMillionUSD = parsePricing.OutputPerMillionUSD
				parseUsageEvent.PricingCurrency = parsePricing.Currency
				parseUsageCost := provider.ParseEstimateCost(parsePromptTokenCount, parseCompletionTokenCount, parsePricing)
				parseUsageEvent.InputCostUSD = parseUsageCost.InputCostUSD
				parseUsageEvent.OutputCostUSD = parseUsageCost.OutputCostUSD
				parseUsageEvent.TotalCostUSD = parseUsageCost.TotalCostUSD
				parseUsageTotalCostUSD = parseUsageCost.TotalCostUSD
			}
			if parseUsageSaveErr := parseS.store.parseSaveUsageEvent(parseUsageEvent); parseUsageSaveErr != nil {
				parseLogger.Error("rpc.Send: db: save usage event failed",
					slog.Int64("user_id", parseUserID),
					slog.Int64("conv_id", parseConversationID),
					slog.String("usage_event_id", parseUsageEvent.EventID),
					slog.String("provider", parseUsageEvent.ProviderID),
					slog.String("resolved_model", parseResolvedModel),
					slog.String("error", parseUsageSaveErr.Error()),
				)
			} else {
				parseUsageEventID = parseUsageEvent.EventID
				isParseUsagePersisted = true
			}
			parseS.parseMarkConversationMessagesSaved(parsePeerAddress, len(parseReq.History)+2)
			go parseS.parseExtractAndStoreUserMemories(parseStream.Context(), parseUserID, parseReq.Message)
			// Generate a title for any conversation that is completing its first
			// exchange (savedMessageCount == 0 means no prior messages existed in the DB).
			if parseSavedMessageCount == 0 {
				go parseS.parseGenerateAndSaveConversationTitle(parseUserID, parseConversationID, parseResolvedModel, parseReq.Message, parseAssistantResponseBuffer.String())
			}
		}
	}
	if isParseFirstExchange {
		parseS.parseUpsertStarterMilestone(parseUserID, parseStarterMilestoneFirstThreadCreated, map[string]any{
			"conversation_id": parseConversationID,
			"source":          "rpc.Send",
		})
		parseS.parseUpsertStarterMilestone(parseUserID, parseStarterMilestoneFirstReplyCompleted, map[string]any{
			"conversation_id": parseConversationID,
			"usage_event_id":  parseUsageEventID,
			"source":          "rpc.Send",
		})
		parseS.parseTrackFirstChatFunnelStep(parseStream.Context(), parseUserID, parseFirstChatStepFirstReplyDone, map[string]any{
			"conversation_id": parseConversationID,
			"usage_event_id":  parseUsageEventID,
			"provider_id":     parseChatProvider.ParseID(),
			"model":           parseResolvedModel,
		})
	}

	parseLogger.Info("rpc.Send: complete",
		slog.Int64("conv_id", parseConversationID),
		slog.Int64("prompt_tokens", parsePromptTokenCount),
		slog.Int64("completion_tokens", parseCompletionTokenCount),
		slog.String("resolved_model", parseResolvedModel),
		slog.String("usage_event_id", parseUsageEventID),
		slog.Float64("total_cost_usd", parseUsageTotalCostUSD),
	)
	return parseStream.Send(&chatpb.ChatChunk{
		Done:             true,
		ConversationId:   parseConversationID,
		Model:            parseResolvedModel,
		PromptTokens:     parsePromptTokenCount,
		CompletionTokens: parseCompletionTokenCount,
		UsageEventId:     parseUsageEventID,
		ProviderId:       parseChatProvider.ParseID(),
		TotalCostUsd:     parseUsageTotalCostUSD,
		UsageSource:      parseUsageSource,
		UsagePersisted:   isParseUsagePersisted,
	})
}
