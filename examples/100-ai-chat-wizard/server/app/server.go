package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/server/provider"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
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

const thoughtChunkModelPrefix = "__thought_delta__:"
const thoughtChunkModelDone = "__thought_done__"

const (
	modelGPT54     = "gpt-5.4"
	modelGPT54Mini = "gpt-5.4-mini"
	modelGPT54Nano = "gpt-5.4-nano"
)

// ─── gRPC service implementation ─────────────────────────────────────────────

type chatServer struct {
	chatpb.UnimplementedChatServiceServer
	providerRegistry      *provider.Registry
	defaultModel          string
	store                 *Store
	logger                *slog.Logger
	sessionsMutex         sync.Mutex
	sessions              map[string]*sessionState
	authMutex             sync.RWMutex
	authUsers             map[string]authUser
	authManager           *authManager
	activeTTSStreams      atomic.Int64
	memoryExtractionSlots chan struct{}
	memoryExtractionModel string
}

// sessionState tracks the active SQLite conversation for one WebSocket peer.
type sessionState struct {
	userID            int64
	conversationID    int64
	savedMessageCount int // number of messages already persisted in this conversation
}

func newChatServiceServer(openAIAPIKey, anthropicAPIKey, cerebrasAPIKey, defaultModel string, store *Store, logger *slog.Logger, stubProviders ...string) *chatServer {
	defaultModel = normalizeSelectedModelID(defaultModel)
	catalogConfig, err := loadModelCatalogConfig(store)
	if err != nil {
		logger.Warn("chat provider catalog load failed", slog.String("error", err.Error()))
	}
	stubSet := normalizeStubProviders(stubProviders)
	providerRegistry := provider.NewRegistry(
		selectRuntimeProvider("openai", strings.TrimSpace(openAIAPIKey), stubSet, catalogConfig.ProviderCatalogs["openai"]),
		selectRuntimeProvider("anthropic", strings.TrimSpace(anthropicAPIKey), stubSet, catalogConfig.ProviderCatalogs["anthropic"]),
		selectRuntimeProvider("cerebras", strings.TrimSpace(cerebrasAPIKey), stubSet, catalogConfig.ProviderCatalogs["cerebras"]),
	)
	if defaultModel == "" {
		defaultModel = normalizeSelectedModelID(catalogConfig.DefaultModel)
	}
	if _, resolvedModel, err := providerRegistry.Resolve(defaultModel); err == nil {
		defaultModel = normalizeSelectedModelID(resolvedModel)
	} else if _, resolvedModel, fallbackErr := providerRegistry.Resolve(""); fallbackErr == nil {
		defaultModel = normalizeSelectedModelID(resolvedModel)
		logger.Warn("chat provider default model override",
			slog.String("requested_model", strings.TrimSpace(defaultModel)),
			slog.String("fallback_model", resolvedModel),
			slog.String("error", err.Error()),
		)
	}
	chatService := &chatServer{
		providerRegistry:      providerRegistry,
		defaultModel:          defaultModel,
		store:                 store,
		logger:                logger,
		sessions:              make(map[string]*sessionState),
		authUsers:             make(map[string]authUser),
		authManager:           newAuthManager("", store, logger.With(slog.String("component", "auth"))),
		memoryExtractionSlots: make(chan struct{}, 2),
		memoryExtractionModel: normalizeSelectedModelID(catalogConfig.MemoryExtractionModel),
	}
	if chatService.memoryExtractionModel == "" {
		chatService.memoryExtractionModel = defaultModel
	}
	return chatService
}

func normalizeStubProviders(values []string) map[string]struct{} {
	normalized := map[string]struct{}{}
	for _, value := range values {
		for _, token := range strings.Split(value, ",") {
			resolved := strings.TrimSpace(strings.ToLower(token))
			if resolved == "" {
				continue
			}
			if resolved == "all" {
				normalized["openai"] = struct{}{}
				normalized["anthropic"] = struct{}{}
				normalized["cerebras"] = struct{}{}
				continue
			}
			normalized[resolved] = struct{}{}
		}
	}
	return normalized
}

func selectRuntimeProvider(providerID string, apiKey string, stubProviders map[string]struct{}, catalog provider.Catalog) provider.ChatProvider {
	if strings.TrimSpace(apiKey) != "" {
		switch providerID {
		case "openai":
			return provider.NewOpenAIProvider(apiKey, catalog)
		case "anthropic":
			return provider.NewAnthropicProvider(apiKey, catalog)
		case "cerebras":
			return provider.NewCerebrasProvider(apiKey, catalog)
		default:
			return nil
		}
	}
	if _, ok := stubProviders[strings.TrimSpace(providerID)]; ok {
		return provider.NewStubProvider(providerID, catalog)
	}
	switch providerID {
	case "openai":
		return provider.NewOpenAIProvider("", catalog)
	case "anthropic":
		return provider.NewAnthropicProvider("", catalog)
	case "cerebras":
		return provider.NewCerebrasProvider("", catalog)
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
func (s *chatServer) loadOrCreateConversationSession(peerAddress string, userID, requestedConversationID int64, historyLength int) (int64, int, error) {
	s.sessionsMutex.Lock()
	defer s.sessionsMutex.Unlock()

	if requestedConversationID > 0 {
		owned, err := s.store.conversationOwnedByUser(userID, requestedConversationID)
		if err != nil {
			return 0, 0, err
		}
		if !owned {
			return 0, 0, status.Error(codes.NotFound, "conversation no longer exists for authenticated user")
		}
		s.sessions[peerAddress] = &sessionState{userID: userID, conversationID: requestedConversationID, savedMessageCount: historyLength}
		return requestedConversationID, historyLength, nil
	}

	conversationID, err := s.store.createConversation(userID)
	if err != nil {
		return 0, 0, err
	}
	s.sessions[peerAddress] = &sessionState{userID: userID, conversationID: conversationID, savedMessageCount: 0}
	return conversationID, 0, nil
}

func (s *chatServer) markConversationMessagesSaved(peerAddress string, savedMessageCount int) {
	s.sessionsMutex.Lock()
	defer s.sessionsMutex.Unlock()
	if session, ok := s.sessions[peerAddress]; ok {
		session.savedMessageCount = savedMessageCount
	}
}

func (s *chatServer) bindAuthenticatedPeer(peerAddress string, user authUser) {
	s.authMutex.Lock()
	defer s.authMutex.Unlock()
	s.authUsers[peerAddress] = user
}

func (s *chatServer) unbindAuthenticatedPeer(peerAddress string) {
	s.authMutex.Lock()
	defer s.authMutex.Unlock()
	delete(s.authUsers, peerAddress)

	s.clearPeerSession(peerAddress)
}

func (s *chatServer) clearPeerSession(peerAddress string) {
	s.sessionsMutex.Lock()
	defer s.sessionsMutex.Unlock()
	delete(s.sessions, peerAddress)
}

func (s *chatServer) authenticatedUserFromContext(ctx context.Context) (authUser, bool) {
	if s.authManager != nil {
		if user, ok := s.authManager.authenticatedUserFromContext(ctx); ok && user.ID > 0 {
			return user, true
		}
	}
	peerInfo, ok := peer.FromContext(ctx)
	if !ok || peerInfo.Addr == nil {
		return authUser{}, false
	}
	peerAddress := peerInfo.Addr.String()

	s.authMutex.RLock()
	user, foundUser := s.authUsers[peerAddress]
	s.authMutex.RUnlock()
	if !foundUser || user.ID <= 0 {
		return authUser{}, false
	}
	return user, true
}

func (s *chatServer) requireAuthenticatedUserID(ctx context.Context) (int64, error) {
	authUser, ok := s.authenticatedUserFromContext(ctx)
	if !ok || authUser.ID <= 0 {
		return 0, status.Error(codes.Unauthenticated, "authentication required")
	}
	return authUser.ID, nil
}

func statusForStoreGuard(err error) error {
	switch {
	case errors.Is(err, errStoreUserMissing):
		return status.Error(codes.Unauthenticated, "authenticated user no longer exists; sign in again")
	case errors.Is(err, errStoreConversationMissing):
		return status.Error(codes.FailedPrecondition, "conversation no longer exists; refresh and retry")
	default:
		return err
	}
}

func isStoreGuardError(err error) bool {
	return errors.Is(err, errStoreUserMissing) || errors.Is(err, errStoreConversationMissing)
}

func (s *chatServer) displayNameForUser(userID int64, fallbackEmail string) string {
	if s.store == nil || userID <= 0 {
		return defaultDisplayNameFromEmail(fallbackEmail)
	}
	name, _, err := s.store.getUserName(userID)
	if err != nil {
		return defaultDisplayNameFromEmail(fallbackEmail)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return defaultDisplayNameFromEmail(fallbackEmail)
	}
	return name
}

func (s *chatServer) authResponseForUser(user authUser, token string) *chatpb.AuthResponse {
	return &chatpb.AuthResponse{
		AuthToken:   token,
		UserId:      user.ID,
		Email:       user.Email,
		DisplayName: s.displayNameForUser(user.ID, user.Email),
	}
}

func (s *chatServer) Signup(ctx context.Context, req *chatpb.SignupRequest) (*chatpb.AuthResponse, error) {
	if s.authManager == nil {
		return nil, status.Error(codes.Internal, "auth unavailable")
	}
	user, err := s.authManager.signup(req.GetEmail(), req.GetPassword(), req.GetDisplayName())
	if err != nil {
		if errors.Is(err, errUserAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "an account with that email already exists")
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	token, err := s.authManager.issueToken(user)
	if err != nil {
		return nil, status.Error(codes.Internal, "issue auth token")
	}
	return s.authResponseForUser(user, token), nil
}

func (s *chatServer) Login(ctx context.Context, req *chatpb.LoginRequest) (*chatpb.AuthResponse, error) {
	if s.authManager == nil {
		return nil, status.Error(codes.Internal, "auth unavailable")
	}
	user, err := s.authManager.login(req.GetEmail(), req.GetPassword())
	if err != nil {
		if errors.Is(err, errInvalidCredentials) {
			return nil, status.Error(codes.Unauthenticated, "invalid email or password")
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	token, err := s.authManager.issueToken(user)
	if err != nil {
		return nil, status.Error(codes.Internal, "issue auth token")
	}
	return s.authResponseForUser(user, token), nil
}

func (s *chatServer) Logout(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if peerInfo, ok := peer.FromContext(ctx); ok && peerInfo.Addr != nil {
		s.clearPeerSession(peerInfo.Addr.String())
	}
	return &emptypb.Empty{}, nil
}

func (s *chatServer) GetSession(ctx context.Context, _ *emptypb.Empty) (*chatpb.GetSessionResponse, error) {
	user, ok := s.authenticatedUserFromContext(ctx)
	if !ok || user.ID <= 0 {
		return &chatpb.GetSessionResponse{}, nil
	}
	return &chatpb.GetSessionResponse{
		Authenticated: true,
		UserId:        user.ID,
		Email:         user.Email,
		DisplayName:   s.displayNameForUser(user.ID, user.Email),
	}, nil
}

func (s *chatServer) RefreshSession(ctx context.Context, _ *emptypb.Empty) (*chatpb.AuthResponse, error) {
	if s.authManager == nil {
		return nil, status.Error(codes.Internal, "auth unavailable")
	}
	user, ok := s.authenticatedUserFromContext(ctx)
	if !ok || user.ID <= 0 {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	token, err := s.authManager.issueToken(user)
	if err != nil {
		return nil, status.Error(codes.Internal, "issue auth token")
	}
	return s.authResponseForUser(user, token), nil
}

// generateAndSaveConversationTitle calls OpenAI (fastest model) to produce a concise 3-7 word
// title from the very first user message and assistant reply, then persists it.
// Must be called in a goroutine — it blocks until the completion returns.
func (s *chatServer) generateAndSaveConversationTitle(userID, conversationID int64, model, userMessage, assistantMessage string) {
	logger := s.logger.With(slog.String("op", "generateTitle"), slog.Int64("conv_id", conversationID))
	if s.providerRegistry == nil || s.store == nil {
		return
	}

	// Truncate inputs so the title-gen call is fast and cheap.
	truncateText := func(text string, maxLength int) string {
		if len(text) <= maxLength {
			return text
		}
		return text[:maxLength] + "…"
	}
	titlePrompt := "User: " + truncateText(userMessage, 400) + "\n\nAssistant: " + truncateText(assistantMessage, 400)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	chatProvider, _, err := s.providerRegistry.Resolve(model)
	if err != nil {
		logger.Warn("generateTitle: provider unavailable", slog.String("model", model), slog.String("error", err.Error()))
		return
	}
	title, err := chatProvider.GenerateTitle(ctx, provider.TitleRequest{
		Model: model,
		SystemPrompt: "You are a conversation title generator. " +
			"Given the first user message and assistant reply, produce a clear and concise title of 3-7 words " +
			"that captures the topic. Return only the title with no punctuation at the end, no quotes, and no explanation.",
		Prompt: titlePrompt,
	})
	if err != nil {
		logger.Error("generateTitle: provider call failed", slog.String("model", model), slog.String("error", err.Error()))
		return
	}
	if title == "" {
		logger.Warn("generateTitle: empty title returned")
		return
	}
	if err := s.store.saveConversationTitle(userID, conversationID, title); err != nil {
		logger.Error("generateTitle: db save failed", slog.String("error", err.Error()))
		return
	}
	logger.Info("generateTitle: saved", slog.String("title", title))
}

// Send streams a chat completion from OpenAI and forwards each token to the
// gRPC client via the tunnel.
func (s *chatServer) Send(req *chatpb.SendRequest, stream chatpb.ChatService_SendServer) error {
	peerAddress := ""
	if p, ok := peer.FromContext(stream.Context()); ok {
		peerAddress = p.Addr.String()
	}
	userID, err := s.requireAuthenticatedUserID(stream.Context())
	if err != nil {
		return err
	}

	messagePreview := req.GetMessage()
	if len(messagePreview) > 80 {
		messagePreview = messagePreview[:80] + "…"
	}

	logger := s.logger.With(
		slog.String("rpc", "Send"),
		slog.String("peer", peerAddress),
		slog.String("model", req.GetModel()),
		slog.Int("history_len", len(req.History)),
		slog.Int64("conv_id_req", req.GetConversationId()),
	)
	logger.Info("rpc.Send: started", slog.String("message_preview", messagePreview))
	if s.store != nil {
		userExists, userErr := s.store.userExists(userID)
		if userErr != nil {
			logger.Error("rpc.Send: authenticated user lookup failed",
				slog.Int64("user_id", userID),
				slog.String("error", userErr.Error()),
			)
			return status.Errorf(codes.Internal, "lookup authenticated user: %v", userErr)
		}
		if !userExists {
			logger.Warn("rpc.Send: authenticated user missing from store",
				slog.Int64("user_id", userID),
				slog.String("peer", peerAddress),
			)
			return status.Error(codes.Unauthenticated, "authenticated user no longer exists; sign in again")
		}
		if req.GetConversationId() > 0 {
			owned, ownedErr := s.store.conversationOwnedByUser(userID, req.GetConversationId())
			if ownedErr != nil {
				logger.Error("rpc.Send: requested conversation lookup failed",
					slog.Int64("user_id", userID),
					slog.Int64("requested_conv_id", req.GetConversationId()),
					slog.String("error", ownedErr.Error()),
				)
				return status.Errorf(codes.Internal, "lookup conversation: %v", ownedErr)
			}
			if !owned {
				logger.Warn("rpc.Send: requested conversation missing or inaccessible",
					slog.Int64("user_id", userID),
					slog.Int64("requested_conv_id", req.GetConversationId()),
				)
				return status.Error(codes.NotFound, "conversation no longer exists for authenticated user")
			}
		}
	}

	customSystemPrompt := ""
	var injectedUserMemories []userMemoryRow
	if s.store != nil {
		storedPrompt, promptErr := s.store.getSelectedSystemPrompt(userID, "")
		if promptErr != nil {
			logger.Warn("rpc.Send: custom system prompt lookup failed", slog.String("error", promptErr.Error()))
		} else {
			customSystemPrompt = storedPrompt
		}
		storedMemories, memoryErr := s.store.listUserMemories(userID)
		if memoryErr != nil {
			logger.Warn("rpc.Send: user memory lookup failed", slog.String("error", memoryErr.Error()))
		} else {
			injectedUserMemories = storedMemories
		}
	}

	// Build the system prompt: base + tone modifier + user custom prompt + remembered preferences.
	systemPrompt := buildSystemPrompt(req.GetTone(), customSystemPrompt, injectedUserMemories)

	// Use the per-request model when the client sends one; fall back to the
	// server default (OPENAI_MODEL env / hard-coded default).
	resolvedModel := s.defaultModel
	if requestedModel := strings.TrimSpace(req.GetModel()); requestedModel != "" {
		resolvedModel = normalizeSelectedModelID(requestedModel)
	}

	logger.Debug("rpc.Send: opening OpenAI stream",
		slog.String("resolved_model", resolvedModel),
		slog.String("tone", req.GetTone()),
		slog.Bool("thinking_enabled", req.GetThinkingEnabled()),
		slog.String("thinking_effort", req.GetThinkingEffort()),
	)
	var chatProvider provider.ChatProvider
	if req.GetThinkingEnabled() {
		chatProvider, resolvedModel, _, err = s.providerRegistry.RequireCapability(resolvedModel, provider.CapabilityThinking)
	} else {
		chatProvider, resolvedModel, err = s.providerRegistry.Resolve(resolvedModel)
	}
	if err != nil {
		logger.Error("rpc.Send: provider resolution failed",
			slog.String("requested_model", req.GetModel()),
			slog.String("resolved_model", resolvedModel),
			slog.String("error", err.Error()),
		)
		if capabilityErr := capabilityStatusError(err); capabilityErr != nil {
			return capabilityErr
		}
		if strings.TrimSpace(req.GetModel()) != "" {
			return status.Errorf(codes.InvalidArgument, "unsupported model %q", strings.TrimSpace(req.GetModel()))
		}
		return status.Error(codes.Unavailable, "no configured model provider available")
	}

	history := make([]provider.ChatMessage, 0, len(req.History))
	for _, historyMessage := range req.History {
		history = append(history, provider.ChatMessage{
			Role:    provider.NormalizeRole(historyMessage.GetRole()),
			Content: historyMessage.GetContent(),
		})
	}

	var assistantResponseBuffer strings.Builder
	var promptTokenCount int64
	var completionTokenCount int64
	sendThoughtDelta := func(deltaText string) error {
		if deltaText == "" {
			return nil
		}
		if err := stream.Send(&chatpb.ChatChunk{Model: thoughtChunkModelPrefix + deltaText}); err != nil {
			logger.Error("rpc.Send: downstream thought stream send failed",
				slog.String("error", err.Error()),
			)
			return status.Errorf(codes.Canceled, "thought stream send: %v", err)
		}
		return nil
	}
	thoughtDoneSent := false
	sendThoughtDone := func() error {
		if thoughtDoneSent {
			return nil
		}
		thoughtDoneSent = true
		if err := stream.Send(&chatpb.ChatChunk{Model: thoughtChunkModelDone}); err != nil {
			logger.Error("rpc.Send: downstream thought complete send failed",
				slog.String("error", err.Error()),
			)
			return status.Errorf(codes.Canceled, "thought complete send: %v", err)
		}
		return nil
	}
	chatResult, err := chatProvider.StreamChat(stream.Context(), provider.ChatRequest{
		Model:           resolvedModel,
		SystemPrompt:    systemPrompt,
		History:         history,
		UserMessage:     req.Message,
		ThinkingEnabled: req.GetThinkingEnabled(),
		ThinkingEffort:  req.GetThinkingEffort(),
	}, func(event provider.ChatEvent) error {
		if event.ThoughtDelta != "" {
			if err := sendThoughtDelta(event.ThoughtDelta); err != nil {
				return err
			}
		}
		if event.TextDelta != "" {
			assistantResponseBuffer.WriteString(event.TextDelta)
			if err := stream.Send(&chatpb.ChatChunk{Delta: event.TextDelta}); err != nil {
				logger.Error("rpc.Send: downstream stream send failed",
					slog.String("error", err.Error()),
				)
				return status.Errorf(codes.Canceled, "stream send: %v", err)
			}
		}
		if event.ThoughtDone {
			if err := sendThoughtDone(); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		logger.Error("rpc.Send: provider stream error",
			slog.String("error", err.Error()),
			slog.String("resolved_model", resolvedModel),
			slog.String("provider", chatProvider.ID()),
		)
		if thoughtDoneErr := sendThoughtDone(); thoughtDoneErr != nil {
			return thoughtDoneErr
		}
		if streamErr := stream.Send(&chatpb.ChatChunk{
			Done:           true,
			Error:          userFacingStreamError(chatProvider.ID(), resolvedModel, err),
			ConversationId: req.GetConversationId(),
			Model:          resolvedModel,
		}); streamErr != nil {
			return status.Errorf(codes.Canceled, "stream error send: %v", streamErr)
		}
		return nil
	}
	if chatResult.Model != "" {
		resolvedModel = normalizeSelectedModelID(chatResult.Model)
	}
	promptTokenCount = chatResult.PromptTokens
	completionTokenCount = chatResult.CompletionTokens

	logger.Debug("rpc.Send: provider stream complete",
		slog.String("provider", chatProvider.ID()),
		slog.String("resolved_model", resolvedModel),
	)

	// ── Persist the exchange ────────────────────────────────────────────────
	var conversationID int64
	if s.store != nil {
		var savedMessageCount int
		conversationID, savedMessageCount, err = s.loadOrCreateConversationSession(peerAddress, userID, req.GetConversationId(), len(req.History))
		if err != nil {
			logger.Error("rpc.Send: db: loadOrCreateConversationSession failed",
				slog.Int64("user_id", userID),
				slog.Int64("requested_conv_id", req.GetConversationId()),
				slog.String("error", err.Error()),
			)
			if isStoreGuardError(err) {
				return statusForStoreGuard(err)
			}
			return err
		} else {
			logger.Debug("rpc.Send: db: session resolved",
				slog.Int64("user_id", userID),
				slog.Int64("conv_id", conversationID),
				slog.Int("already_saved", savedMessageCount),
				slog.Int("history_len", len(req.History)),
			)
			// Save any history messages not yet persisted.
			for historyIndex := savedMessageCount; historyIndex < len(req.History); historyIndex++ {
				historyMessage := req.History[historyIndex]
				if dbErr := s.store.saveConversationMessage(userID, conversationID, historyMessage.Role, historyMessage.Content, historyMessage.GetModelId(), historyMessage.GetPromptTokens(), historyMessage.GetCompletionTokens()); dbErr != nil {
					logger.Error("rpc.Send: db: save history message failed",
						slog.Int64("user_id", userID),
						slog.Int64("requested_conv_id", req.GetConversationId()),
						slog.Int64("conv_id", conversationID),
						slog.Int("index", historyIndex),
						slog.String("role", historyMessage.Role),
						slog.String("error", dbErr.Error()),
					)
					if errors.Is(dbErr, errStoreConversationMissing) {
						s.clearPeerSession(peerAddress)
						break
					}
				}
			}
			if dbErr := s.store.saveConversationMessage(userID, conversationID, "user", req.Message, "", 0, 0); dbErr != nil {
				logger.Error("rpc.Send: db: save user message failed",
					slog.Int64("user_id", userID),
					slog.Int64("requested_conv_id", req.GetConversationId()),
					slog.Int64("conv_id", conversationID),
					slog.String("error", dbErr.Error()),
				)
				if errors.Is(dbErr, errStoreConversationMissing) {
					s.clearPeerSession(peerAddress)
				}
			}
			if dbErr := s.store.saveConversationMessage(userID, conversationID, "assistant", assistantResponseBuffer.String(), resolvedModel, promptTokenCount, completionTokenCount); dbErr != nil {
				logger.Error("rpc.Send: db: save assistant message failed",
					slog.Int64("user_id", userID),
					slog.Int64("requested_conv_id", req.GetConversationId()),
					slog.Int64("conv_id", conversationID),
					slog.String("resolved_model", resolvedModel),
					slog.Int64("prompt_tokens", promptTokenCount),
					slog.Int64("completion_tokens", completionTokenCount),
					slog.String("error", dbErr.Error()),
				)
				if errors.Is(dbErr, errStoreConversationMissing) {
					s.clearPeerSession(peerAddress)
				}
			}
			s.markConversationMessagesSaved(peerAddress, len(req.History)+2)
			go s.extractAndStoreUserMemories(userID, req.Message)
			// Generate a title for any conversation that is completing its first
			// exchange (savedMessageCount == 0 means no prior messages existed in the DB).
			if savedMessageCount == 0 {
				go s.generateAndSaveConversationTitle(userID, conversationID, resolvedModel, req.Message, assistantResponseBuffer.String())
			}
		}
	}

	logger.Info("rpc.Send: complete",
		slog.Int64("conv_id", conversationID),
		slog.Int64("prompt_tokens", promptTokenCount),
		slog.Int64("completion_tokens", completionTokenCount),
		slog.String("resolved_model", resolvedModel),
	)
	return stream.Send(&chatpb.ChatChunk{Done: true, ConversationId: conversationID, Model: resolvedModel, PromptTokens: promptTokenCount, CompletionTokens: completionTokenCount})
}

// ─── Conversation management RPCs ────────────────────────────────────────────

func (s *chatServer) ListConversations(ctx context.Context, _ *chatpb.ListConversationsRequest) (*chatpb.ListConversationsResponse, error) {
	logger := s.logger.With(slog.String("rpc", "ListConversations"))
	logger.Info("rpc.ListConversations: started")
	userID, err := s.requireAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}

	if s.store == nil {
		logger.Warn("rpc.ListConversations: store unavailable — returning empty list")
		return &chatpb.ListConversationsResponse{}, nil
	}
	conversationSummaries, err := s.store.listConversations(userID)
	if err != nil {
		logger.Error("rpc.ListConversations: db query failed", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "list conversations: %v", err)
	}
	responseSummaries := make([]*chatpb.ConversationSummary, 0, len(conversationSummaries))
	for _, conversationSummary := range conversationSummaries {
		preview := conversationSummary.Preview
		if len(preview) > 60 {
			preview = preview[:60] + "…"
		}
		responseSummaries = append(responseSummaries, &chatpb.ConversationSummary{
			Id:        conversationSummary.ID,
			PublicId:  conversationSummary.PublicID,
			StartedAt: conversationSummary.StartedAt,
			Preview:   preview,
		})
	}
	logger.Info("rpc.ListConversations: complete", slog.Int("count", len(responseSummaries)))
	return &chatpb.ListConversationsResponse{Conversations: responseSummaries}, nil
}

func (s *chatServer) ResolveConversationRoute(ctx context.Context, req *chatpb.ResolveConversationRouteRequest) (*chatpb.ResolveConversationRouteResponse, error) {
	logger := s.logger.With(slog.String("rpc", "ResolveConversationRoute"), slog.String("public_id", strings.TrimSpace(req.GetPublicId())))
	logger.Info("rpc.ResolveConversationRoute: started")
	userID, err := s.requireAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		logger.Warn("rpc.ResolveConversationRoute: store unavailable — returning inaccessible")
		return &chatpb.ResolveConversationRouteResponse{Accessible: false}, nil
	}
	summary, ok, err := s.store.resolveConversationRoute(userID, req.GetPublicId())
	if err != nil {
		logger.Error("rpc.ResolveConversationRoute: db query failed", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "resolve conversation route: %v", err)
	}
	if !ok {
		logger.Info("rpc.ResolveConversationRoute: inaccessible")
		return &chatpb.ResolveConversationRouteResponse{Accessible: false}, nil
	}
	logger.Info("rpc.ResolveConversationRoute: complete", slog.Int64("conv_id", summary.ID))
	return &chatpb.ResolveConversationRouteResponse{
		Id:         summary.ID,
		PublicId:   summary.PublicID,
		Accessible: true,
	}, nil
}

func (s *chatServer) LoadConversation(ctx context.Context, req *chatpb.LoadConversationRequest) (*chatpb.LoadConversationResponse, error) {
	logger := s.logger.With(slog.String("rpc", "LoadConversation"), slog.Int64("conv_id", req.GetId()))
	logger.Info("rpc.LoadConversation: started")
	userID, err := s.requireAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}

	if s.store == nil {
		logger.Warn("rpc.LoadConversation: store unavailable — returning empty response")
		return &chatpb.LoadConversationResponse{}, nil
	}
	conversationRows, err := s.store.loadConversation(userID, req.GetId())
	if err != nil {
		logger.Error("rpc.LoadConversation: db query failed", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "load conversation: %v", err)
	}
	responseMessages := make([]*chatpb.ChatMessage, 0, len(conversationRows))
	for _, row := range conversationRows {
		responseMessages = append(responseMessages, &chatpb.ChatMessage{Role: provider.NormalizeRole(row.Role), Content: row.Content, ModelId: row.ModelID, PromptTokens: row.PromptTokens, CompletionTokens: row.CompletionTokens})
	}
	logger.Info("rpc.LoadConversation: complete", slog.Int("messages", len(responseMessages)))
	return &chatpb.LoadConversationResponse{Messages: responseMessages}, nil
}

func (s *chatServer) ListModelOptions(_ context.Context, _ *chatpb.ListModelOptionsRequest) (*chatpb.ListModelOptionsResponse, error) {
	logger := s.logger.With(slog.String("rpc", "ListModelOptions"))
	if s.providerRegistry == nil {
		logger.Warn("rpc.ListModelOptions: provider registry unavailable")
		return &chatpb.ListModelOptionsResponse{}, nil
	}
	options := s.providerRegistry.ModelOptions()
	responseOptions := make([]*chatpb.ModelOption, 0, len(options))
	for _, option := range options {
		responseOptions = append(responseOptions, &chatpb.ModelOption{
			Id:    option.ID,
			Label: option.Label,
			Note:  option.Note,
			Capabilities: &chatpb.ModelCapabilities{
				SupportsThinking: option.Capabilities.SupportsThinking,
				SupportsSpeech:   option.Capabilities.SupportsSpeech,
				ProviderId:       option.Capabilities.ProviderID,
				ProviderLabel:    option.Capabilities.ProviderLabel,
			},
			Pricing: &chatpb.ModelPricing{
				InputCostPerMillionUsd:  option.Pricing.InputPerMillionUSD,
				OutputCostPerMillionUsd: option.Pricing.OutputPerMillionUSD,
				Currency:                option.Pricing.Currency,
			},
		})
	}
	defaultModel := s.defaultModel
	if defaultModel == "" {
		defaultModel = s.providerRegistry.DefaultModel()
	}
	logger.Info("rpc.ListModelOptions: complete", slog.Int("count", len(responseOptions)), slog.String("default_model", defaultModel))
	return &chatpb.ListModelOptionsResponse{Models: responseOptions, DefaultModel: defaultModel}, nil
}

func (s *chatServer) DeleteConversation(ctx context.Context, req *chatpb.DeleteConversationRequest) (*chatpb.DeleteConversationResponse, error) {
	logger := s.logger.With(slog.String("rpc", "DeleteConversation"), slog.Int64("conv_id", req.GetId()))
	logger.Info("rpc.DeleteConversation: started")
	userID, err := s.requireAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}

	if s.store == nil {
		logger.Warn("rpc.DeleteConversation: store unavailable — no-op")
		return &chatpb.DeleteConversationResponse{}, nil
	}
	if err := s.store.deleteConversation(userID, req.GetId()); err != nil {
		logger.Error("rpc.DeleteConversation: db delete failed", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "delete conversation: %v", err)
	}
	logger.Info("rpc.DeleteConversation: complete")
	return &chatpb.DeleteConversationResponse{}, nil
}

func (s *chatServer) SetUserName(ctx context.Context, req *chatpb.SetUserNameRequest) (*chatpb.SetUserNameResponse, error) {
	logger := s.logger.With(slog.String("rpc", "SetUserName"))
	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name must not be empty")
	}
	userID, err := s.requireAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		logger.Warn("rpc.SetUserName: store unavailable — no-op")
		return &chatpb.SetUserNameResponse{}, nil
	}
	now := time.Now().Unix()
	if err := s.store.setUserName(userID, name, now); err != nil {
		logger.Error("rpc.SetUserName: db upsert failed", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "set user name: %v", err)
	}
	logger.Info("rpc.SetUserName: complete", slog.String("name", name))
	return &chatpb.SetUserNameResponse{}, nil
}

func (s *chatServer) GetUserName(ctx context.Context, _ *chatpb.GetUserNameRequest) (*chatpb.GetUserNameResponse, error) {
	logger := s.logger.With(slog.String("rpc", "GetUserName"))
	userID, err := s.requireAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		logger.Warn("rpc.GetUserName: store unavailable — returning default")
		return &chatpb.GetUserNameResponse{Name: "User", UpdatedAt: 0}, nil
	}
	name, updatedAt, err := s.store.getUserName(userID)
	if err != nil {
		logger.Error("rpc.GetUserName: db query failed", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "get user name: %v", err)
	}
	logger.Info("rpc.GetUserName: complete", slog.String("name", name), slog.Int64("updated_at", updatedAt))
	return &chatpb.GetUserNameResponse{Name: name, UpdatedAt: updatedAt}, nil
}

func (s *chatServer) ListUserMemories(ctx context.Context, _ *chatpb.ListUserMemoriesRequest) (*chatpb.ListUserMemoriesResponse, error) {
	logger := s.logger.With(slog.String("rpc", "ListUserMemories"))
	userID, err := s.requireAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		logger.Warn("rpc.ListUserMemories: store unavailable — returning empty list")
		return &chatpb.ListUserMemoriesResponse{}, nil
	}
	rows, err := s.store.listUserMemories(userID)
	if err != nil {
		logger.Error("rpc.ListUserMemories: db query failed", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "list user memories: %v", err)
	}
	memories := make([]*chatpb.UserMemory, 0, len(rows))
	for _, row := range rows {
		memories = append(memories, &chatpb.UserMemory{
			Key:             row.Key,
			Category:        row.Category,
			Summary:         row.Summary,
			Detail:          row.Detail,
			SourceMessage:   row.SourceMessage,
			UsefulnessScore: int32(row.UsefulnessScore),
			ConfidenceScore: row.ConfidenceScore,
			RubricReason:    row.RubricReason,
			UpdatedAt:       row.UpdatedAt,
		})
	}
	return &chatpb.ListUserMemoriesResponse{Memories: memories}, nil
}

func (s *chatServer) UpsertUserMemory(ctx context.Context, req *chatpb.UpsertUserMemoryRequest) (*emptypb.Empty, error) {
	logger := s.logger.With(slog.String("rpc", "UpsertUserMemory"))
	userID, err := s.requireAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		logger.Warn("rpc.UpsertUserMemory: store unavailable — no-op")
		return &emptypb.Empty{}, nil
	}
	memory := req.GetMemory()
	if memory == nil {
		return nil, status.Error(codes.InvalidArgument, "memory is required")
	}
	summary := strings.TrimSpace(memory.GetSummary())
	if summary == "" {
		return nil, status.Error(codes.InvalidArgument, "memory summary is required")
	}
	category := normalizeUserMemoryCategory(memory.GetCategory())
	key := normalizeUserMemoryKey(memory.GetKey(), category, summary)
	if key == "" {
		return nil, status.Error(codes.InvalidArgument, "memory key could not be derived")
	}
	if err := s.store.upsertUserMemory(userID, userMemoryRow{
		Key:             key,
		Category:        category,
		Summary:         summary,
		Detail:          strings.TrimSpace(memory.GetDetail()),
		SourceMessage:   strings.TrimSpace(memory.GetSourceMessage()),
		UsefulnessScore: clampUsefulnessScore(int(memory.GetUsefulnessScore())),
		ConfidenceScore: clampConfidenceScore(memory.GetConfidenceScore()),
		RubricReason:    strings.TrimSpace(memory.GetRubricReason()),
	}); err != nil {
		logger.Error("rpc.UpsertUserMemory: db upsert failed", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "upsert user memory: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (s *chatServer) DeleteUserMemory(ctx context.Context, req *chatpb.DeleteUserMemoryRequest) (*emptypb.Empty, error) {
	logger := s.logger.With(slog.String("rpc", "DeleteUserMemory"))
	userID, err := s.requireAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		logger.Warn("rpc.DeleteUserMemory: store unavailable — no-op")
		return &emptypb.Empty{}, nil
	}
	key := strings.TrimSpace(req.GetKey())
	if key == "" {
		return nil, status.Error(codes.InvalidArgument, "memory key is required")
	}
	if err := s.store.deleteUserMemory(userID, key); err != nil {
		logger.Error("rpc.DeleteUserMemory: db delete failed", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "delete user memory: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (s *chatServer) SynthesizeSpeech(req *chatpb.SynthesizeSpeechRequest, stream grpc.ServerStreamingServer[chatpb.SynthesizeSpeechChunk]) error {
	logger := s.logger.With(slog.String("rpc", "SynthesizeSpeech"))
	if _, err := s.requireAuthenticatedUserID(stream.Context()); err != nil {
		return err
	}
	if s.providerRegistry == nil {
		logger.Error("rpc.SynthesizeSpeech: provider registry unavailable")
		return status.Error(codes.Unavailable, "no configured model provider available")
	}

	script := sanitizeTextForTTS(req.GetText())
	if script == "" {
		return status.Error(codes.InvalidArgument, "text must contain speakable content")
	}
	resolvedModel := s.defaultModel
	if requestedModel := strings.TrimSpace(req.GetModel()); requestedModel != "" {
		resolvedModel = normalizeSelectedModelID(requestedModel)
	}
	speechProvider, resolvedModel, _, err := s.providerRegistry.RequireCapability(resolvedModel, provider.CapabilitySpeech)
	if err != nil {
		logger.Error("rpc.SynthesizeSpeech: provider resolution failed",
			slog.String("requested_model", req.GetModel()),
			slog.String("resolved_model", resolvedModel),
			slog.String("error", err.Error()),
		)
		if capabilityErr := capabilityStatusError(err); capabilityErr != nil {
			return capabilityErr
		}
		if strings.TrimSpace(req.GetModel()) != "" {
			return status.Errorf(codes.InvalidArgument, "unsupported model %q", strings.TrimSpace(req.GetModel()))
		}
		return status.Error(codes.Unavailable, "no configured model provider available")
	}

	activeStreams := s.activeTTSStreams.Add(1)
	defer s.activeTTSStreams.Add(-1)
	totalAudioBytes := 0
	speechResult, err := speechProvider.SynthesizeSpeech(stream.Context(), provider.SpeechRequest{Model: resolvedModel, Text: script}, func(chunk provider.SpeechChunk) error {
		if len(chunk.AudioChunk) > 0 {
			totalAudioBytes += len(chunk.AudioChunk)
		}
		if err := stream.Send(&chatpb.SynthesizeSpeechChunk{
			AudioChunk: chunk.AudioChunk,
			Done:       chunk.Done,
			MimeType:   chunk.MimeType,
			Model:      chunk.Model,
			Voice:      chunk.Voice,
			Script:     chunk.Script,
		}); err != nil {
			logger.Warn("rpc.SynthesizeSpeech: downstream stream send failed",
				slog.String("error", err.Error()),
				slog.Int("audio_bytes", totalAudioBytes),
			)
			return status.Errorf(codes.Canceled, "speech stream send: %v", err)
		}
		return nil
	})
	if err != nil {
		logger.Error("rpc.SynthesizeSpeech: provider call failed",
			slog.String("provider", speechProvider.ID()),
			slog.String("model", resolvedModel),
			slog.String("error", err.Error()),
		)
		if capabilityErr := capabilityStatusError(err); capabilityErr != nil {
			return capabilityErr
		}
		if status.Code(err) != codes.Unknown {
			return err
		}
		return status.Errorf(codes.Internal, "synthesize speech: %v", err)
	}
	if totalAudioBytes == 0 {
		return status.Error(codes.Internal, "synthesized audio was empty")
	}

	logger.Info("rpc.SynthesizeSpeech: complete",
		slog.Int("script_len", len(script)),
		slog.Int("audio_bytes", totalAudioBytes),
		slog.Int64("active_streams", activeStreams),
		slog.String("provider", speechProvider.ID()),
		slog.String("model", speechResult.Model),
		slog.String("voice", speechResult.Voice),
	)

	return nil
}

func (s *chatServer) SetSelectedModel(ctx context.Context, req *wrapperspb.StringValue) (*emptypb.Empty, error) {
	logger := s.logger.With(slog.String("rpc", "SetSelectedModel"))
	selectedModel := normalizeSelectedModelID(req.GetValue())
	if selectedModel == "" {
		return nil, status.Error(codes.InvalidArgument, "model must not be empty")
	}
	userID, err := s.requireAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	if s.providerRegistry != nil {
		if _, _, err := s.providerRegistry.Resolve(selectedModel); err != nil {
			if capabilityErr := capabilityStatusError(err); capabilityErr != nil {
				return nil, capabilityErr
			}
			return nil, status.Errorf(codes.InvalidArgument, "unsupported model %q", selectedModel)
		}
	}
	if s.store == nil {
		logger.Warn("rpc.SetSelectedModel: store unavailable — no-op")
		return &emptypb.Empty{}, nil
	}
	if err := s.store.setSelectedModel(userID, selectedModel); err != nil {
		logger.Error("rpc.SetSelectedModel: db upsert failed", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "set selected model: %v", err)
	}
	logger.Info("rpc.SetSelectedModel: complete", slog.String("model", selectedModel))
	return &emptypb.Empty{}, nil
}

func (s *chatServer) getSelectedModelLegacy(ctx context.Context, _ *emptypb.Empty) (*wrapperspb.StringValue, error) {
	logger := s.logger.With(slog.String("rpc", "GetSelectedModel"))
	userID, err := s.requireAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		logger.Warn("rpc.GetSelectedModel: store unavailable — returning default")
		return wrapperspb.String(s.defaultModel), nil
	}
	selectedModel, err := s.store.getSelectedModel(userID, s.defaultModel)
	if err != nil {
		logger.Error("rpc.GetSelectedModel: db query failed", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "get selected model: %v", err)
	}
	selectedModel = normalizeSelectedModelID(selectedModel)
	if s.providerRegistry != nil {
		if _, resolvedModel, resolveErr := s.providerRegistry.Resolve(selectedModel); resolveErr == nil {
			selectedModel = normalizeSelectedModelID(resolvedModel)
		} else if fallbackModel := s.providerRegistry.DefaultModel(); fallbackModel != "" {
			selectedModel = normalizeSelectedModelID(fallbackModel)
		}
	}
	logger.Info("rpc.GetSelectedModel: complete", slog.String("model", selectedModel))
	return wrapperspb.String(selectedModel), nil
}

func (s *chatServer) GetSelectedModel(ctx context.Context, _ *emptypb.Empty) (*wrapperspb.StringValue, error) {
	logger := s.logger.With(slog.String("rpc", "GetSelectedModel"))
	userID, err := s.requireAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		logger.Warn("rpc.GetSelectedModel: store unavailable, returning default")
		return wrapperspb.String(s.defaultModel), nil
	}

	fallbackModel := normalizeSelectedModelID(s.defaultModel)
	if s.providerRegistry != nil {
		modelOptions := s.providerRegistry.ModelOptions()
		if len(modelOptions) > 0 {
			firstModel := normalizeSelectedModelID(modelOptions[0].ID)
			if firstModel != "" {
				fallbackModel = firstModel
			}
		}
		if fallbackModel == "" {
			fallbackModel = normalizeSelectedModelID(s.providerRegistry.DefaultModel())
		}
	}
	if fallbackModel == "" {
		fallbackModel = normalizeSelectedModelID("")
	}

	selectedModel, err := s.store.getSelectedModel(userID, fallbackModel)
	if err != nil {
		logger.Error("rpc.GetSelectedModel: db query failed", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "get selected model: %v", err)
	}
	originalSelectedModel := strings.TrimSpace(selectedModel)
	selectedModel = normalizeSelectedModelID(selectedModel)

	if s.providerRegistry != nil {
		if _, resolvedModel, resolveErr := s.providerRegistry.Resolve(selectedModel); resolveErr == nil {
			selectedModel = normalizeSelectedModelID(resolvedModel)
		} else if fallbackModel != "" {
			selectedModel = fallbackModel
		} else if registryDefault := s.providerRegistry.DefaultModel(); registryDefault != "" {
			selectedModel = normalizeSelectedModelID(registryDefault)
		}
	}
	if selectedModel == "" {
		selectedModel = fallbackModel
	}
	if selectedModel == "" {
		selectedModel = normalizeSelectedModelID("")
	}

	if setErr := s.store.setSelectedModel(userID, selectedModel); setErr != nil {
		logger.Warn("rpc.GetSelectedModel: failed to persist repaired model", slog.String("error", setErr.Error()), slog.String("model", selectedModel))
	} else if selectedModel != originalSelectedModel {
		logger.Info("rpc.GetSelectedModel: repaired persisted model", slog.String("from", originalSelectedModel), slog.String("to", selectedModel))
	}

	logger.Info("rpc.GetSelectedModel: complete", slog.String("model", selectedModel))
	return wrapperspb.String(selectedModel), nil
}

func capabilityStatusError(err error) error {
	var capabilityErr *provider.UnsupportedCapabilityError
	if !errors.As(err, &capabilityErr) {
		return nil
	}
	return status.Errorf(codes.FailedPrecondition, "model %q does not support %s", capabilityErr.Model, capabilityErr.Capability)
}

func userFacingStreamError(providerID string, modelID string, err error) string {
	providerID = strings.TrimSpace(providerID)
	modelID = strings.TrimSpace(modelID)
	raw := strings.TrimSpace(err.Error())
	if raw == "" {
		if providerID == "" {
			return "model provider stream failed"
		}
		return fmt.Sprintf("%s stream failed", providerID)
	}
	if providerID == "cerebras" && strings.Contains(raw, "404 Not Found") {
		if modelID != "" {
			return fmt.Sprintf("Cerebras model %q is unavailable for this API key. Try llama3.1-8b or another available Cerebras model.", modelID)
		}
		return "Selected Cerebras model is unavailable for this API key. Try llama3.1-8b or another available Cerebras model."
	}
	return raw
}

func normalizeSelectedModelID(modelID string) string {
	modelID = strings.TrimSpace(modelID)
	switch modelID {
	case "", modelGPT54Mini, "gpt-5.4-mini-2026-03-17":
		return modelGPT54Mini
	case modelGPT54, "gpt-5.4-2026-03-17":
		return modelGPT54
	case modelGPT54Nano, "gpt-5.4-nano-2026-03-17":
		return modelGPT54Nano
	default:
		return modelID
	}
}

func (s *chatServer) SetSelectedTone(ctx context.Context, req *wrapperspb.StringValue) (*emptypb.Empty, error) {
	logger := s.logger.With(slog.String("rpc", "SetSelectedTone"))
	selectedTone := normalizeSelectedToneID(req.GetValue())
	userID, err := s.requireAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		logger.Warn("rpc.SetSelectedTone: store unavailable — no-op")
		return &emptypb.Empty{}, nil
	}
	if err := s.store.setSelectedTone(userID, selectedTone); err != nil {
		logger.Error("rpc.SetSelectedTone: db upsert failed", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "set selected tone: %v", err)
	}
	logger.Info("rpc.SetSelectedTone: complete", slog.String("tone", selectedTone))
	return &emptypb.Empty{}, nil
}

func (s *chatServer) GetSelectedTone(ctx context.Context, _ *emptypb.Empty) (*wrapperspb.StringValue, error) {
	logger := s.logger.With(slog.String("rpc", "GetSelectedTone"))
	userID, err := s.requireAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		logger.Warn("rpc.GetSelectedTone: store unavailable — returning default")
		return wrapperspb.String(defaultToneID), nil
	}
	selectedTone, err := s.store.getSelectedTone(userID, defaultToneID)
	if err != nil {
		logger.Error("rpc.GetSelectedTone: db query failed", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "get selected tone: %v", err)
	}
	selectedTone = normalizeSelectedToneID(selectedTone)
	logger.Info("rpc.GetSelectedTone: complete", slog.String("tone", selectedTone))
	return wrapperspb.String(selectedTone), nil
}

func (s *chatServer) SetSelectedThinkingEnabled(ctx context.Context, req *wrapperspb.BoolValue) (*emptypb.Empty, error) {
	logger := s.logger.With(slog.String("rpc", "SetSelectedThinkingEnabled"))
	userID, err := s.requireAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		logger.Warn("rpc.SetSelectedThinkingEnabled: store unavailable — no-op")
		return &emptypb.Empty{}, nil
	}
	if err := s.store.setSelectedThinkingEnabled(userID, req.GetValue()); err != nil {
		logger.Error("rpc.SetSelectedThinkingEnabled: db upsert failed", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "set selected thinking enabled: %v", err)
	}
	logger.Info("rpc.SetSelectedThinkingEnabled: complete", slog.Bool("enabled", req.GetValue()))
	return &emptypb.Empty{}, nil
}

func (s *chatServer) GetSelectedThinkingEnabled(ctx context.Context, _ *emptypb.Empty) (*wrapperspb.BoolValue, error) {
	logger := s.logger.With(slog.String("rpc", "GetSelectedThinkingEnabled"))
	userID, err := s.requireAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		logger.Warn("rpc.GetSelectedThinkingEnabled: store unavailable — returning default")
		return wrapperspb.Bool(defaultThinkingEnabled), nil
	}
	enabled, err := s.store.getSelectedThinkingEnabled(userID, defaultThinkingEnabled)
	if err != nil {
		logger.Error("rpc.GetSelectedThinkingEnabled: db query failed", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "get selected thinking enabled: %v", err)
	}
	logger.Info("rpc.GetSelectedThinkingEnabled: complete", slog.Bool("enabled", enabled))
	return wrapperspb.Bool(enabled), nil
}

func (s *chatServer) SetSelectedThinkingEffort(ctx context.Context, req *wrapperspb.StringValue) (*emptypb.Empty, error) {
	logger := s.logger.With(slog.String("rpc", "SetSelectedThinkingEffort"))
	selectedThinkingEffort := normalizeSelectedThinkingEffort(req.GetValue())
	userID, err := s.requireAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		logger.Warn("rpc.SetSelectedThinkingEffort: store unavailable — no-op")
		return &emptypb.Empty{}, nil
	}
	if err := s.store.setSelectedThinkingEffort(userID, selectedThinkingEffort); err != nil {
		logger.Error("rpc.SetSelectedThinkingEffort: db upsert failed", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "set selected thinking effort: %v", err)
	}
	logger.Info("rpc.SetSelectedThinkingEffort: complete", slog.String("effort", selectedThinkingEffort))
	return &emptypb.Empty{}, nil
}

func (s *chatServer) GetSelectedThinkingEffort(ctx context.Context, _ *emptypb.Empty) (*wrapperspb.StringValue, error) {
	logger := s.logger.With(slog.String("rpc", "GetSelectedThinkingEffort"))
	userID, err := s.requireAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		logger.Warn("rpc.GetSelectedThinkingEffort: store unavailable — returning default")
		return wrapperspb.String(defaultThinkingEffort), nil
	}
	selectedThinkingEffort, err := s.store.getSelectedThinkingEffort(userID, defaultThinkingEffort)
	if err != nil {
		logger.Error("rpc.GetSelectedThinkingEffort: db query failed", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "get selected thinking effort: %v", err)
	}
	selectedThinkingEffort = normalizeSelectedThinkingEffort(selectedThinkingEffort)
	logger.Info("rpc.GetSelectedThinkingEffort: complete", slog.String("effort", selectedThinkingEffort))
	return wrapperspb.String(selectedThinkingEffort), nil
}

func (s *chatServer) SetCustomSystemPrompt(ctx context.Context, req *wrapperspb.StringValue) (*emptypb.Empty, error) {
	logger := s.logger.With(slog.String("rpc", "SetCustomSystemPrompt"))
	userID, err := s.requireAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		logger.Warn("rpc.SetCustomSystemPrompt: store unavailable — no-op")
		return &emptypb.Empty{}, nil
	}
	customPrompt := normalizeCustomSystemPrompt(req.GetValue())
	if err := s.store.setSelectedSystemPrompt(userID, customPrompt); err != nil {
		logger.Error("rpc.SetCustomSystemPrompt: db upsert failed", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "set custom system prompt: %v", err)
	}
	logger.Info("rpc.SetCustomSystemPrompt: complete", slog.Int("runes", len([]rune(customPrompt))))
	return &emptypb.Empty{}, nil
}

func (s *chatServer) GetCustomSystemPrompt(ctx context.Context, _ *emptypb.Empty) (*wrapperspb.StringValue, error) {
	logger := s.logger.With(slog.String("rpc", "GetCustomSystemPrompt"))
	userID, err := s.requireAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	if s.store == nil {
		logger.Warn("rpc.GetCustomSystemPrompt: store unavailable — returning default")
		return wrapperspb.String(""), nil
	}
	customPrompt, err := s.store.getSelectedSystemPrompt(userID, "")
	if err != nil {
		logger.Error("rpc.GetCustomSystemPrompt: db query failed", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "get custom system prompt: %v", err)
	}
	customPrompt = normalizeCustomSystemPrompt(customPrompt)
	logger.Info("rpc.GetCustomSystemPrompt: complete", slog.Int("runes", len([]rune(customPrompt))))
	return wrapperspb.String(customPrompt), nil
}

// baseAssistantSystemPrompt is the foundation injected on every request. Extend this
// to shape the assistant's personality, knowledge scope, and safety rails.
const baseAssistantSystemPrompt = `You are a helpful AI assistant built into a Go WebAssembly chat application.
You have broad general knowledge and can help with coding, writing, research, and everyday questions.
Always be accurate; if you are uncertain, say so rather than guessing.
Format responses with Markdown when it improves readability (code blocks, lists, headers).`

// toneInstructionByID maps the client-selected tone to an additional system directive.
var toneInstructionByID = map[string]string{
	"balanced":     "Communicate in a warm, conversational tone — friendly but informative.",
	"friendly":     "Be upbeat, approachable, and kind. Prefer plain language and a supportive tone while staying accurate.",
	"professional": "Communicate formally and precisely. Avoid casual language. Prefer structured, direct responses.",
	"concise":      "Be as brief as possible. Omit pleasantries and filler. Lead with the answer, then add detail only if essential.",
}

func normalizeSelectedToneID(toneID string) string {
	toneID = strings.TrimSpace(toneID)
	if toneID == "" {
		return defaultToneID
	}
	if _, ok := toneInstructionByID[toneID]; ok {
		return toneID
	}
	return defaultToneID
}

func normalizeSelectedThinkingEffort(effort string) string {
	effort = strings.TrimSpace(strings.ToLower(effort))
	switch effort {
	case "low", "medium", "high":
		return effort
	default:
		return defaultThinkingEffort
	}
}

var fencedCodeBlockPattern = regexp.MustCompile("(?s)```.*?```|~~~.*?~~~")
var inlineCodePattern = regexp.MustCompile("`[^`]+`")
var markdownLinkPattern = regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`)
var markdownFormattingPattern = regexp.MustCompile(`(?m)^#{1,6}\s+|[*_~]+|^>\s?`)
var htmlCommentPattern = regexp.MustCompile(`(?s)<!--.*?-->`)
var blankLinePattern = regexp.MustCompile(`\n{3,}`)

func sanitizeTextForTTS(source string) string {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return ""
	}
	trimmed = htmlCommentPattern.ReplaceAllString(trimmed, " ")
	trimmed = fencedCodeBlockPattern.ReplaceAllString(trimmed, " ")
	trimmed = inlineCodePattern.ReplaceAllString(trimmed, " ")
	trimmed = markdownLinkPattern.ReplaceAllString(trimmed, "$1")
	trimmed = markdownFormattingPattern.ReplaceAllString(trimmed, "")

	lines := strings.Split(trimmed, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		resolved := strings.TrimSpace(line)
		if resolved == "" {
			kept = append(kept, "")
			continue
		}
		if strings.HasPrefix(resolved, "//") || strings.HasPrefix(resolved, "/*") || strings.HasPrefix(resolved, "*") || strings.HasPrefix(resolved, "*/") {
			continue
		}
		kept = append(kept, resolved)
	}

	trimmed = strings.TrimSpace(strings.Join(kept, "\n"))
	trimmed = blankLinePattern.ReplaceAllString(trimmed, "\n\n")
	trimmed = strings.Join(strings.Fields(trimmed), " ")
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) > maxTTSScriptRunes {
		trimmed = strings.TrimSpace(string(runes[:maxTTSScriptRunes]))
	}
	return trimmed
}

// buildSystemPrompt combines the base prompt with the tone directive for the
// given tone ID. Falls back to balanced when the ID is unrecognised.
func buildSystemPrompt(tone string, customPrompt string, memories []userMemoryRow) string {
	instruction := toneInstructionByID[normalizeSelectedToneID(tone)]
	customPrompt = normalizeCustomSystemPrompt(customPrompt)
	memoryBlock := buildUserMemoryPromptBlock(memories)
	customPromptUsesMemories := strings.Contains(customPrompt, "{{memories}}")
	customPrompt = resolveSystemPromptTemplate(customPrompt, memoryBlock, time.Now())
	prompt := baseAssistantSystemPrompt + "\n\n" + instruction
	if customPrompt != "" {
		prompt += "\n\nAdditional user-configured instructions:\n" + customPrompt
	}
	if memoryBlock != "" && !customPromptUsesMemories {
		prompt += "\n\nKnown user context:\n" + memoryBlock
	}
	return prompt
}

func resolveSystemPromptTemplate(prompt, memoryBlock string, now time.Time) string {
	resolved := strings.TrimSpace(prompt)
	if resolved == "" {
		return ""
	}
	resolvedMemories := strings.TrimSpace(memoryBlock)
	if resolvedMemories == "" {
		resolvedMemories = "- No stored memories yet."
	}
	replacer := strings.NewReplacer(
		"{{date}}", now.Format("2006-01-02"),
		"{{time}}", now.Format("15:04:05 -0700"),
		"{{memories}}", resolvedMemories,
	)
	return replacer.Replace(resolved)
}

func normalizeCustomSystemPrompt(prompt string) string {
	trimmed := strings.TrimSpace(prompt)
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) > maxCustomSystemPromptRunes {
		trimmed = strings.TrimSpace(string(runes[:maxCustomSystemPromptRunes]))
	}
	return trimmed
}

func (s *chatServer) extractAndStoreUserMemories(userID int64, userMessage string) {
	if s == nil || s.store == nil || s.providerRegistry == nil {
		return
	}
	startedAt := time.Now()
	trimmedMessage := strings.TrimSpace(userMessage)
	if trimmedMessage == "" {
		s.logger.Debug("memory extraction skipped: blank message", slog.Int64("user_id", userID))
		return
	}
	if slots := s.memoryExtractionSlots; slots != nil {
		select {
		case slots <- struct{}{}:
			defer func() { <-slots }()
		default:
			s.logger.Info("memory extraction skipped: queue full",
				slog.Int64("user_id", userID),
				slog.Int("message_chars", len([]rune(trimmedMessage))),
			)
			return
		}
	}

	extractionModel := normalizeSelectedModelID(s.memoryExtractionModel)
	if extractionModel == "" {
		extractionModel = s.defaultModel
	}
	extractionProvider, resolvedModel, err := s.providerRegistry.Resolve(extractionModel)
	if err != nil {
		s.logger.Warn("memory extraction skipped: provider unavailable",
			slog.Int64("user_id", userID),
			slog.String("requested_model", extractionModel),
			slog.String("error", err.Error()),
			slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		)
		return
	}
	s.logger.Debug("memory extraction started",
		slog.Int64("user_id", userID),
		slog.String("provider", extractionProvider.ID()),
		slog.String("model", resolvedModel),
		slog.Int("message_chars", len([]rune(trimmedMessage))),
	)

	ctx, cancel := context.WithTimeout(context.Background(), userMemoryExtractionTimeout)
	defer cancel()

	candidates, err := extractionProvider.ExtractUserMemories(ctx, provider.MemoryExtractionRequest{
		Model:       resolvedModel,
		UserMessage: trimmedMessage,
	})
	if err != nil {
		s.logger.Warn("memory extraction failed",
			slog.Int64("user_id", userID),
			slog.String("provider", extractionProvider.ID()),
			slog.String("model", resolvedModel),
			slog.String("error", err.Error()),
			slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		)
		return
	}

	usefulCandidates := filterUsefulUserMemories(candidates)
	savedCount := 0
	saveFailureCount := 0
	for _, candidate := range usefulCandidates {
		if err := s.store.upsertUserMemory(userID, userMemoryRow{
			Key:             normalizeUserMemoryKey(candidate.Key, candidate.Category, candidate.Summary),
			Category:        normalizeUserMemoryCategory(candidate.Category),
			Summary:         strings.TrimSpace(candidate.Summary),
			Detail:          strings.TrimSpace(candidate.Detail),
			SourceMessage:   trimmedMessage,
			UsefulnessScore: clampUsefulnessScore(candidate.UsefulnessScore),
			ConfidenceScore: clampConfidenceScore(candidate.ConfidenceScore),
			RubricReason:    strings.TrimSpace(candidate.RubricReason),
		}); err != nil {
			saveFailureCount++
			s.logger.Warn("memory extraction save failed",
				slog.Int64("user_id", userID),
				slog.String("provider", extractionProvider.ID()),
				slog.String("model", resolvedModel),
				slog.String("memory_key", candidate.Key),
				slog.String("error", err.Error()),
			)
			continue
		}
		savedCount++
	}
	s.logger.Info("memory extraction completed",
		slog.Int64("user_id", userID),
		slog.String("provider", extractionProvider.ID()),
		slog.String("model", resolvedModel),
		slog.Int("candidate_count", len(candidates)),
		slog.Int("useful_candidate_count", len(usefulCandidates)),
		slog.Int("saved_count", savedCount),
		slog.Int("save_failure_count", saveFailureCount),
		slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
	)
}

func filterUsefulUserMemories(candidates []provider.UserMemoryCandidate) []provider.UserMemoryCandidate {
	filtered := make([]provider.UserMemoryCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		candidate.Category = normalizeUserMemoryCategory(candidate.Category)
		candidate.Key = normalizeUserMemoryKey(candidate.Key, candidate.Category, candidate.Summary)
		candidate.Summary = strings.TrimSpace(candidate.Summary)
		candidate.Detail = strings.TrimSpace(candidate.Detail)
		candidate.RubricReason = strings.TrimSpace(candidate.RubricReason)
		candidate.UsefulnessScore = clampUsefulnessScore(candidate.UsefulnessScore)
		candidate.ConfidenceScore = clampConfidenceScore(candidate.ConfidenceScore)
		if candidate.Key == "" || candidate.Summary == "" {
			continue
		}
		if candidate.UsefulnessScore < userMemoryUsefulnessThreshold || candidate.ConfidenceScore < userMemoryConfidenceThreshold {
			continue
		}
		filtered = append(filtered, candidate)
	}
	return filtered
}

func normalizeUserMemoryCategory(category string) string {
	switch strings.TrimSpace(strings.ToLower(category)) {
	case "preference", "profile", "constraint", "project":
		return strings.TrimSpace(strings.ToLower(category))
	default:
		return "other"
	}
}

func normalizeUserMemoryKey(key, category, summary string) string {
	base := strings.TrimSpace(strings.ToLower(key))
	if base == "" {
		base = normalizeUserMemoryCategory(category) + "-" + strings.TrimSpace(strings.ToLower(summary))
	}
	var builder strings.Builder
	lastDash := false
	for _, currentRune := range base {
		switch {
		case currentRune >= 'a' && currentRune <= 'z', currentRune >= '0' && currentRune <= '9':
			builder.WriteRune(currentRune)
			lastDash = false
		default:
			if !lastDash {
				builder.WriteRune('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(builder.String(), "-")
}

func clampUsefulnessScore(score int) int {
	switch {
	case score < 0:
		return 0
	case score > 100:
		return 100
	default:
		return score
	}
}

func clampConfidenceScore(score float64) float64 {
	switch {
	case score < 0:
		return 0
	case score > 1:
		return 1
	default:
		return score
	}
}

func buildUserMemoryPromptBlock(memories []userMemoryRow) string {
	if len(memories) == 0 {
		return ""
	}
	var builder strings.Builder
	count := 0
	for _, memory := range memories {
		if count >= maxInjectedUserMemoryCount {
			break
		}
		summary := strings.TrimSpace(memory.Summary)
		if summary == "" {
			continue
		}
		builder.WriteString("- ")
		builder.WriteString(summary)
		if detail := strings.TrimSpace(memory.Detail); detail != "" && !strings.EqualFold(detail, summary) {
			builder.WriteString(" (")
			builder.WriteString(detail)
			builder.WriteString(")")
		}
		builder.WriteString("\n")
		count++
		if builder.Len() >= maxInjectedUserMemoryRunes {
			break
		}
	}
	block := strings.TrimSpace(builder.String())
	if block == "" {
		return ""
	}
	runes := []rune(block)
	if len(runes) > maxInjectedUserMemoryRunes {
		block = strings.TrimSpace(string(runes[:maxInjectedUserMemoryRunes]))
	}
	return block
}

// ─── main ────────────────────────────────────────────────────────────────────

func loadFirstDotEnv(load func(string) error, candidates []string) string {
	for _, candidate := range candidates {
		if err := load(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

type serverRuntimeConfig struct {
	openAIAPIKey    string
	anthropicAPIKey string
	cerebrasAPIKey  string
	stubProviders   []string
	defaultModel    string
	addr            string
	dbPath          string
	authSecret      string
	usagePremiumPct float64
}

func readServerRuntimeConfig(getenv func(string) string) serverRuntimeConfig {
	defaultModel := strings.TrimSpace(getenv("CHAT_MODEL"))
	if defaultModel == "" {
		defaultModel = strings.TrimSpace(getenv("OPENAI_MODEL"))
	}
	addr := strings.TrimSpace(getenv("LISTEN_ADDR"))
	if addr == "" {
		addr = "127.0.0.1:8095"
	}
	dbPath := strings.TrimSpace(getenv("CHAT_DB_PATH"))
	if dbPath == "" {
		dbPath = "examples/100-ai-chat-wizard/bin/runtime/chat_history.db"
	}
	usagePremiumPct := parseUsagePremiumPercent(getenv("CHAT_USAGE_PREMIUM_PERCENT"), 5.0)
	return serverRuntimeConfig{
		openAIAPIKey:    strings.TrimSpace(getenv("OPENAI_API_KEY")),
		anthropicAPIKey: strings.TrimSpace(getenv("ANTHROPIC_API_KEY")),
		cerebrasAPIKey:  strings.TrimSpace(getenv("CEREBRAS_API_KEY")),
		stubProviders:   splitAndTrim(getenv("CHAT_PROVIDER_STUBS")),
		defaultModel:    defaultModel,
		addr:            addr,
		dbPath:          dbPath,
		authSecret:      strings.TrimSpace(getenv("CHAT_AUTH_SECRET")),
		usagePremiumPct: usagePremiumPct,
	}
}

func splitAndTrim(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	trimmed := make([]string, 0, len(parts))
	for _, part := range parts {
		resolved := strings.TrimSpace(part)
		if resolved != "" {
			trimmed = append(trimmed, resolved)
		}
	}
	return trimmed
}

func parseUsagePremiumPercent(rawValue string, fallback float64) float64 {
	trimmed := strings.TrimSpace(rawValue)
	if trimmed == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || parsed < 0 {
		return fallback
	}
	if parsed > 1000 {
		return 1000
	}
	return parsed
}

func chatShellHandler(fileServer http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/chat-bootstrap.js", "/app/chat-bootstrap.js":
			serveChatBootstrapJS(w, r)
			return
		}
		rewritten := rewriteLegacyClientAssetRequest(r)
		if rewritten != r {
			fileServer.ServeHTTP(w, rewritten)
			return
		}
		if shouldServeClientShell(r.URL.Path) {
			serveChatShell(w, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

func rewriteLegacyClientAssetRequest(r *http.Request) *http.Request {
	switch r.URL.Path {
	case "/chat.wasm":
		return cloneRequestWithPath(r, "/app/chat.wasm")
	case "/background-worker.wasm":
		return cloneRequestWithPath(r, "/worker/background-worker.wasm")
	default:
		return r
	}
}

func shouldServeClientShell(requestPath string) bool {
	trimmedPath := strings.TrimSpace(requestPath)
	if trimmedPath == "" || trimmedPath == "/" {
		return true
	}
	cleanedPath := filepath.ToSlash(filepath.Clean("/" + strings.TrimLeft(trimmedPath, "/")))
	if cleanedPath == "/" || cleanedPath == "/app" || cleanedPath == "/app/" {
		return true
	}
	if cleanedPath == "/home" || cleanedPath == "/capabilities" || cleanedPath == "/pricing" {
		return true
	}
	if cleanedPath == "/thread" || strings.HasPrefix(cleanedPath, "/thread/") {
		return true
	}
	if strings.HasPrefix(cleanedPath, "/app/thread/") {
		return true
	}
	if filepath.Ext(cleanedPath) != "" {
		return false
	}
	switch cleanedPath {
	case "/login", "/signup", "/logout":
		return true
	default:
		return false
	}
}

func cloneRequestWithPath(r *http.Request, path string) *http.Request {
	cloned := r.Clone(r.Context())
	if cloned.URL != nil {
		urlCopy := *cloned.URL
		cloned.URL = &urlCopy
	}
	cloned.URL.Path = path
	cloned.URL.RawPath = path
	cloned.RequestURI = path
	if cloned.URL.RawQuery != "" {
		cloned.RequestURI += "?" + cloned.URL.RawQuery
	}
	return cloned
}
func Run() {
	logger, closeLogger, loggerErr := newServerLogger()
	if loggerErr != nil {
		logger = newOTELLogger(os.Stderr, serverServiceName)
		logger.Error("logging: failed to initialize file sink; continuing with stderr only",
			slog.String("error", loggerErr.Error()),
			slog.String("log.dir", serverLogDir),
		)
		closeLogger = func() {}
	}
	defer closeLogger()
	slog.SetDefault(logger)
	// Load .env from the example directory, the server directory, or the repo
	// root — whichever is found first. Existing environment variables are never
	// overwritten, so explicit exports always take precedence.
	loadedEnvPath := loadFirstDotEnv(func(path string) error { return godotenv.Load(path) }, []string{
		".env",
		"../.env",
		"examples/100-ai-chat-wizard/.env",
	})
	if loadedEnvPath != "" {
		logger.Info("env: loaded .env file", slog.String("path", loadedEnvPath))
	}

	config := readServerRuntimeConfig(os.Getenv)
	setChatUsagePremiumPercent(config.usagePremiumPct)
	logger.Info("billing: usage premium configured", slog.Float64("usage_premium_percent", config.usagePremiumPct))
	stubSet := normalizeStubProviders(config.stubProviders)

	openAIAPIKey := config.openAIAPIKey
	if openAIAPIKey == "" {
		if _, ok := stubSet["openai"]; ok {
			logger.Info("env: OPENAI_API_KEY not set; using OpenAI stub provider for local development")
		} else {
			logger.Warn("env: OPENAI_API_KEY not set — chat RPCs will return Unavailable until configured")
		}
	}
	anthropicAPIKey := config.anthropicAPIKey
	if anthropicAPIKey == "" {
		if _, ok := stubSet["anthropic"]; ok {
			logger.Info("env: ANTHROPIC_API_KEY not set; using Anthropic stub provider for local development")
		} else {
			logger.Warn("env: ANTHROPIC_API_KEY not set — Claude models will be unavailable until configured")
		}
	}

	cerebrasAPIKey := config.cerebrasAPIKey
	if cerebrasAPIKey == "" {
		if _, ok := stubSet["cerebras"]; ok {
			logger.Info("env: CEREBRAS_API_KEY not set; using Cerebras stub provider for local development")
		} else {
			logger.Warn("env: CEREBRAS_API_KEY not set â€” Cerebras models will be unavailable until configured")
		}
	}

	defaultModel := config.defaultModel
	if defaultModel == "" {
		logger.Debug("env: CHAT_MODEL not set — server default will be chosen from available providers")
	} else {
		logger.Info("env: model override", slog.String("model", defaultModel))
	}

	addr := config.addr

	// ── SQLite persistence ────────────────────────────────────────────────────
	dbPath := config.dbPath
	store, dbErr := openChatStore(dbPath)
	if dbErr != nil {
		logger.Error("db: failed to open — running without persistence",
			slog.String("path", dbPath),
			slog.String("error", dbErr.Error()),
		)
		store = nil
	} else {
		logger.Info("db: opened", slog.String("path", dbPath))
		defer store.close()
	}
	authManager := newAuthManager(config.authSecret, store, logger.With(slog.String("component", "auth")))

	// ── gRPC server ───────────────────────────────────────────────────────────
	grpcSrv := grpc.NewServer()
	svcLog := logger.With(slog.String("component", "chat-service"))
	chatService := newChatServiceServer(openAIAPIKey, anthropicAPIKey, cerebrasAPIKey, defaultModel, store, svcLog, config.stubProviders...)
	chatService.authManager = authManager
	chatpb.RegisterChatServiceServer(grpcSrv, chatService)

	// ── GoGRPCBridge: expose gRPC over WebSocket ──────────────────────────────
	tunnelHandler := newGRPCTunnelHandler(grpcSrv, logger)

	// ── HTTP mux ──────────────────────────────────────────────────────────────
	mux := http.NewServeMux()
	mux.Handle("/socket", tunnelHandler)
	mux.Handle("/socket/", tunnelHandler)

	clientDir, sharedDir := resolveStaticDirectories()
	if sharedDir != "" {
		mux.Handle("/static/", http.StripPrefix("/static/", newPrecompressedWASMFileServer(sharedDir)))
	}
	// wasm_exec.js is served from its known location in third_party.
	wasmExecPath := resolveWasmExecPath()
	mux.HandleFunc("/static/script/wasm_exec.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		http.ServeFile(w, r, wasmExecPath)
	})
	fileServer := newPrecompressedWASMFileServer(clientDir)
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	// ── Marketing pages (public, no auth required) ────────────────────────────

	// ── Catch-all: bare "/" → marketing home; all other paths → chat shell ───
	mux.Handle("/", chatShellHandler(fileServer))

	logger.Info("server: starting",
		slog.String("addr", addr),
		slog.String("client_dir", clientDir),
		slog.String("static_dir", sharedDir),
		slog.String("grpc_ws", "ws://"+addr+"/socket"),
	)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-quit
		logger.Info("server: shutdown signal received", slog.String("signal", sig.String()))
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		grpcSrv.GracefulStop()
		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("server: graceful shutdown failed", slog.String("error", err.Error()))
		} else {
			logger.Info("server: shutdown complete")
		}
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server: ListenAndServe failed",
			slog.String("addr", addr),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
}

// resolveWasmExecPath locates wasm_exec.js relative to common invocation roots.
func resolveWasmExecPath() string {
	candidates := []string{
		"third_party/GoGRPCBridge/examples/_shared/public/wasm_exec.js",
		"../../../third_party/GoGRPCBridge/examples/_shared/public/wasm_exec.js",
		"../../../../third_party/GoGRPCBridge/examples/_shared/public/wasm_exec.js",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return candidates[0]
}

// resolveStaticDirectories returns (clientDir, sharedStaticDir).
// clientDir contains the WASM binaries and related browser assets.
// sharedStaticDir contains the examples-wide static assets (tailwind, wasm_exec.js).
// Both resolve relative to common invocation roots (from repo root or from
// the server/ subdirectory).
func resolveStaticDirectories() (clientDir, sharedDir string) {
	clientCandidates := []string{
		"../bin/client",
		"bin/client",
		"examples/100-ai-chat-wizard/bin/client",
	}
	for _, c := range clientCandidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			clientDir = c
			break
		}
	}
	if clientDir == "" {
		clientDir = "bin/client"
	}

	sharedCandidates := []string{
		"../../static",
		"../../../static",
		"examples/static",
	}
	for _, c := range sharedCandidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			sharedDir = c
			break
		}
	}
	return
}

func newPrecompressedWASMFileServer(rootDir string) http.Handler {
	fileServer := http.FileServer(http.Dir(rootDir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if tryServeBrotliWASM(w, r, rootDir) {
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

func tryServeBrotliWASM(w http.ResponseWriter, r *http.Request, rootDir string) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	if filepath.Ext(r.URL.Path) != ".wasm" {
		return false
	}
	queryValue := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("br")))
	if queryValue == "false" || queryValue == "0" {
		return false
	}
	if queryValue != "" && queryValue != "true" && queryValue != "1" {
		return false
	}

	relativePath, ok := resolveRelativeAssetPath(r.URL.Path)
	if !ok {
		return false
	}
	brotliPath := filepath.Join(rootDir, relativePath) + ".br"
	artifactInfo, err := os.Stat(brotliPath)
	if err != nil || artifactInfo.IsDir() {
		return false
	}
	if artifactInfo.Size() <= 0 {
		return false
	}

	outputFile, err := os.Open(brotliPath)
	if err != nil {
		return false
	}
	defer outputFile.Close()

	w.Header().Set("Content-Encoding", "br")
	w.Header().Set("Content-Type", "application/wasm")
	w.Header().Set("Vary", "Accept-Encoding")
	http.ServeContent(w, r, filepath.Base(r.URL.Path), artifactInfo.ModTime(), outputFile)
	return true
}

func resolveRelativeAssetPath(requestPath string) (string, bool) {
	cleanedPath := filepath.ToSlash(filepath.Clean("/" + requestPath))
	if strings.Contains(cleanedPath, "..") {
		return "", false
	}
	cleanedPath = strings.TrimLeft(cleanedPath, "/")
	cleanedPath = strings.TrimPrefix(cleanedPath, "./")
	return cleanedPath, cleanedPath != ""
}
