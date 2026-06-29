package app

import (
	"context"
	"log/slog"
	"strings"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/server/provider"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

// Conversation management RPCs.

// ListConversations returns the caller's conversation list.
func (parseS *chatServer) ListConversations(parseCtx context.Context, parseReq *chatpb.ListConversationsRequest) (*chatpb.ListConversationsResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "ListConversations"))
	parseLogger.Info("rpc.ListConversations: started")
	if parseReq == nil {
		parseReq = &chatpb.ListConversationsRequest{}
	}
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}

	if parseS.store == nil {
		parseLogger.Warn("rpc.ListConversations: store unavailable - returning empty list")
		return &chatpb.ListConversationsResponse{}, nil
	}
	parseConversationSummaries, parseErr := parseS.store.parseListConversations(parseUserID)
	if parseErr != nil {
		parseLogger.Error("rpc.ListConversations: db query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list conversations: %v", parseErr)
	}
	parseStarterState := parseS.parseResolveStarterStateForConversationCount(parseUserID, len(parseConversationSummaries))
	parseS.parseSyncStarterStateMilestones(parseUserID, parseStarterState)
	parsePagedConversationSummaries, parseNextOffset, isHasMore := parsePaginateConversationSummaries(parseConversationSummaries, parseReq.GetPageSize(), parseReq.GetPageOffset())
	parseResponseSummaries := make([]*chatpb.ConversationSummary, 0, len(parsePagedConversationSummaries))
	for _, parseConversationSummary := range parsePagedConversationSummaries {
		parsePreview := parseConversationSummary.Preview
		if len(parsePreview) > 60 {
			parsePreview = parsePreview[:60] + "..."
		}
		parseResponseSummaries = append(parseResponseSummaries, &chatpb.ConversationSummary{
			Id:        parseConversationSummary.ID,
			PublicId:  parseConversationSummary.PublicID,
			StartedAt: parseConversationSummary.StartedAt,
			Preview:   parsePreview,
		})
	}
	parseLogger.Info("rpc.ListConversations: complete",
		slog.Int("count", len(parseResponseSummaries)),
		slog.Bool("has_more", isHasMore),
		slog.Int("next_offset", int(parseNextOffset)),
		slog.Int("total_count", len(parseConversationSummaries)),
		slog.Int("requested_page_size", int(parseReq.GetPageSize())),
		slog.Int("requested_page_offset", int(parseReq.GetPageOffset())),
	)
	return &chatpb.ListConversationsResponse{
		Conversations: parseResponseSummaries,
		HasMore:       isHasMore,
		NextOffset:    parseNextOffset,
	}, nil
}

// parsePaginateConversationSummaries slices newest-first summaries for one page request.
//
// A non-positive page size preserves backward compatibility by returning all rows.
func parsePaginateConversationSummaries(parseSummaries []conversationSummaryRow, parsePageSizeRaw, parsePageOffsetRaw int32) ([]conversationSummaryRow, int32, bool) {
	const parseMaxConversationPageSize = 200
	parseSummaryCount := len(parseSummaries)
	if parsePageSizeRaw <= 0 {
		return parseSummaries, int32(parseSummaryCount), false
	}
	if parsePageSizeRaw > parseMaxConversationPageSize {
		parsePageSizeRaw = parseMaxConversationPageSize
	}
	parsePageOffset := max(int(parsePageOffsetRaw), 0)
	if parsePageOffset >= parseSummaryCount {
		return []conversationSummaryRow{}, int32(parseSummaryCount), false
	}
	parsePageEnd := min(parsePageOffset+int(parsePageSizeRaw), parseSummaryCount)
	isHasMore := parsePageEnd < parseSummaryCount
	return parseSummaries[parsePageOffset:parsePageEnd], int32(parsePageEnd), isHasMore
}

// ResolveConversationRoute returns the route for one conversation.
func (parseS *chatServer) ResolveConversationRoute(parseCtx context.Context, parseReq *chatpb.ResolveConversationRouteRequest) (*chatpb.ResolveConversationRouteResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "ResolveConversationRoute"), slog.String("public_id", strings.TrimSpace(parseReq.GetPublicId())))
	parseLogger.Info("rpc.ResolveConversationRoute: started")
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS.store == nil {
		parseLogger.Warn("rpc.ResolveConversationRoute: store unavailable - returning inaccessible")
		return &chatpb.ResolveConversationRouteResponse{Accessible: false}, nil
	}
	parseSummary, parseOk, parseErr := parseS.store.parseResolveConversationRoute(parseUserID, parseReq.GetPublicId())
	if parseErr != nil {
		parseLogger.Error("rpc.ResolveConversationRoute: db query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "resolve conversation route: %v", parseErr)
	}
	if !parseOk {
		parseLogger.Info("rpc.ResolveConversationRoute: inaccessible")
		return &chatpb.ResolveConversationRouteResponse{Accessible: false}, nil
	}
	parseLogger.Info("rpc.ResolveConversationRoute: complete", slog.Int64("conv_id", parseSummary.ID))
	return &chatpb.ResolveConversationRouteResponse{
		Id:         parseSummary.ID,
		PublicId:   parseSummary.PublicID,
		Accessible: true,
	}, nil
}

// LoadConversation returns one conversation by public ID.
func (parseS *chatServer) LoadConversation(parseCtx context.Context, parseReq *chatpb.LoadConversationRequest) (*chatpb.LoadConversationResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "LoadConversation"), slog.Int64("conv_id", parseReq.GetId()))
	parseLogger.Info("rpc.LoadConversation: started")
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}

	if parseS.store == nil {
		parseLogger.Warn("rpc.LoadConversation: store unavailable - returning empty response")
		return &chatpb.LoadConversationResponse{}, nil
	}
	parseConversationRows, parseErr := parseS.store.parseLoadConversation(parseUserID, parseReq.GetId())
	if parseErr != nil {
		parseLogger.Error("rpc.LoadConversation: db query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "load conversation: %v", parseErr)
	}
	parseResponseMessages := make([]*chatpb.ChatMessage, 0, len(parseConversationRows))
	for _, parseRow := range parseConversationRows {
		parseResponseMessages = append(parseResponseMessages, &chatpb.ChatMessage{Role: provider.ParseNormalizeRole(parseRow.Role), Content: parseRow.Content, ModelId: parseRow.ModelID, PromptTokens: parseRow.PromptTokens, CompletionTokens: parseRow.CompletionTokens})
	}
	if len(parseResponseMessages) > 0 {
		parseS.parseTrackFirstChatFunnelStep(parseCtx, parseUserID, parseFirstChatStepThreadReopened, map[string]any{
			"conversation_id": parseReq.GetId(),
			"message_count":   len(parseResponseMessages),
			"source":          "rpc.LoadConversation",
		})
	}
	parseLogger.Info("rpc.LoadConversation: complete", slog.Int("messages", len(parseResponseMessages)))
	return &chatpb.LoadConversationResponse{Messages: parseResponseMessages}, nil
}

// ListModelOptions returns the available model options for the current user.
func (parseS *chatServer) ListModelOptions(_ context.Context, _ *chatpb.ListModelOptionsRequest) (*chatpb.ListModelOptionsResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "ListModelOptions"))
	if parseS.providerRegistry == nil {
		parseLogger.Warn("rpc.ListModelOptions: provider registry unavailable")
		return &chatpb.ListModelOptionsResponse{}, nil
	}
	parseOptions := parseS.providerRegistry.ParseModelOptions()
	parseResponseOptions := make([]*chatpb.ModelOption, 0, len(parseOptions))
	for _, parseOption := range parseOptions {
		parseResponseOptions = append(parseResponseOptions, &chatpb.ModelOption{
			Id:    parseOption.ID,
			Label: parseOption.Label,
			Note:  parseOption.Note,
			Capabilities: &chatpb.ModelCapabilities{
				SupportsThinking: parseOption.Capabilities.SupportsThinking,
				SupportsSpeech:   parseOption.Capabilities.SupportsSpeech,
				ProviderId:       parseOption.Capabilities.ProviderID,
				ProviderLabel:    parseOption.Capabilities.ProviderLabel,
			},
			Pricing: &chatpb.ModelPricing{
				InputCostPerMillionUsd:  parseOption.Pricing.InputPerMillionUSD,
				OutputCostPerMillionUsd: parseOption.Pricing.OutputPerMillionUSD,
				Currency:                parseOption.Pricing.Currency,
			},
		})
	}
	parseDefaultModel := parseS.defaultModel
	if parseDefaultModel == "" {
		parseDefaultModel = parseS.providerRegistry.ParseDefaultModel()
	}
	parseLogger.Info("rpc.ListModelOptions: complete", slog.Int("count", len(parseResponseOptions)), slog.String("default_model", parseDefaultModel))
	return &chatpb.ListModelOptionsResponse{Models: parseResponseOptions, DefaultModel: parseDefaultModel}, nil
}

// DeleteConversation deletes one conversation by public ID.
func (parseS *chatServer) DeleteConversation(parseCtx context.Context, parseReq *chatpb.DeleteConversationRequest) (*chatpb.DeleteConversationResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "DeleteConversation"), slog.Int64("conv_id", parseReq.GetId()))
	parseLogger.Info("rpc.DeleteConversation: started")
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}

	if parseS.store == nil {
		parseLogger.Warn("rpc.DeleteConversation: store unavailable - no-op")
		return &chatpb.DeleteConversationResponse{}, nil
	}
	if parseErr2 := parseS.store.parseDeleteConversation(parseUserID, parseReq.GetId()); parseErr2 != nil {
		parseLogger.Error("rpc.DeleteConversation: db delete failed", slog.String("error", parseErr2.Error()))
		return nil, status.Errorf(codes.Internal, "delete conversation: %v", parseErr2)
	}
	parseLogger.Info("rpc.DeleteConversation: complete")
	return &chatpb.DeleteConversationResponse{}, nil
}

// SetUserName stores the current user's display name.
func (parseS *chatServer) SetUserName(parseCtx context.Context, parseReq *chatpb.SetUserNameRequest) (*chatpb.SetUserNameResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "SetUserName"))
	parseLogger = parseLogger.With(parseResolveTraceabilityAttrs(parseCtx)...)
	parseName := strings.TrimSpace(parseReq.GetName())
	if parseName == "" {
		return nil, status.Error(codes.InvalidArgument, "name must not be empty")
	}
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS.store == nil {
		parseLogger.Error("rpc.SetUserName: store unavailable; rejecting persisted profile write")
		return nil, status.Error(codes.Unavailable, "profile persistence unavailable")
	}
	parseNow := time.Now().Unix()
	if parseErr2 := parseS.store.setUserName(parseUserID, parseName, parseNow); parseErr2 != nil {
		parseLogger.Error("rpc.SetUserName: db upsert failed", slog.String("error", parseErr2.Error()))
		return nil, parseBuildSanitizedInternalStatus(parseCtx, "set user name", parseErr2)
	}
	parseLogger.Info("rpc.SetUserName: complete", slog.String("name", parseName))
	return &chatpb.SetUserNameResponse{}, nil
}

// GetUserName returns the current user's display name.
func (parseS *chatServer) GetUserName(parseCtx context.Context, _ *chatpb.GetUserNameRequest) (*chatpb.GetUserNameResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetUserName"))
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS.store == nil {
		parseLogger.Warn("rpc.GetUserName: store unavailable - returning default")
		return &chatpb.GetUserNameResponse{Name: "User", UpdatedAt: 0}, nil
	}
	parseName, parseUpdatedAt, parseErr := parseS.store.getUserName(parseUserID)
	if parseErr != nil {
		parseLogger.Error("rpc.GetUserName: db query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "get user name: %v", parseErr)
	}
	parseLogger.Info("rpc.GetUserName: complete", slog.String("name", parseName), slog.Int64("updated_at", parseUpdatedAt))
	return &chatpb.GetUserNameResponse{Name: parseName, UpdatedAt: parseUpdatedAt}, nil
}

// ListUserMemories returns the current user's memory rows.
func (parseS *chatServer) ListUserMemories(parseCtx context.Context, _ *chatpb.ListUserMemoriesRequest) (*chatpb.ListUserMemoriesResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "ListUserMemories"))
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS.store == nil {
		parseLogger.Warn("rpc.ListUserMemories: store unavailable - returning empty list")
		return &chatpb.ListUserMemoriesResponse{}, nil
	}
	parseRows, parseErr := parseS.store.parseListUserMemories(parseUserID)
	if parseErr != nil {
		parseLogger.Error("rpc.ListUserMemories: db query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list user memories: %v", parseErr)
	}
	parseMemories := make([]*chatpb.UserMemory, 0, len(parseRows))
	for _, parseRow := range parseRows {
		parseMemories = append(parseMemories, &chatpb.UserMemory{
			Key:             parseRow.Key,
			Category:        parseRow.Category,
			Summary:         parseRow.Summary,
			Detail:          parseRow.Detail,
			SourceMessage:   parseRow.SourceMessage,
			UsefulnessScore: int32(parseRow.UsefulnessScore),
			ConfidenceScore: parseRow.ConfidenceScore,
			RubricReason:    parseRow.RubricReason,
			UpdatedAt:       parseRow.UpdatedAt,
		})
	}
	return &chatpb.ListUserMemoriesResponse{Memories: parseMemories}, nil
}

// UpsertUserMemory stores one memory row for the current user.
func (parseS *chatServer) UpsertUserMemory(parseCtx context.Context, parseReq *chatpb.UpsertUserMemoryRequest) (*emptypb.Empty, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "UpsertUserMemory"))
	parseLogger = parseLogger.With(parseResolveTraceabilityAttrs(parseCtx)...)
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS.store == nil {
		parseLogger.Error("rpc.UpsertUserMemory: store unavailable; rejecting remembered-preferences write")
		return nil, parseBuildStoreUnavailableRPCStatus(parseCtx, "/chat.ChatService/UpsertUserMemory", "memory persistence unavailable")
	}
	parseMemory := parseReq.GetMemory()
	if parseMemory == nil {
		return nil, status.Error(codes.InvalidArgument, "memory is required")
	}
	parseSummary := strings.TrimSpace(parseMemory.GetSummary())
	if parseSummary == "" {
		return nil, status.Error(codes.InvalidArgument, "memory summary is required")
	}
	parseCategory := parseNormalizeUserMemoryCategory(parseMemory.GetCategory())
	parseKey := parseNormalizeUserMemoryKey(parseMemory.GetKey(), parseCategory, parseSummary)
	if parseKey == "" {
		return nil, status.Error(codes.InvalidArgument, "memory key could not be derived")
	}
	if parseErr2 := parseS.store.parseUpsertUserMemory(parseUserID, userMemoryRow{
		Key:             parseKey,
		Category:        parseCategory,
		Summary:         parseSummary,
		Detail:          strings.TrimSpace(parseMemory.GetDetail()),
		SourceMessage:   strings.TrimSpace(parseMemory.GetSourceMessage()),
		UsefulnessScore: parseClampUsefulnessScore(int(parseMemory.GetUsefulnessScore())),
		ConfidenceScore: parseClampConfidenceScore(parseMemory.GetConfidenceScore()),
		RubricReason:    strings.TrimSpace(parseMemory.GetRubricReason()),
	}); parseErr2 != nil {
		parseLogger.Error("rpc.UpsertUserMemory: db upsert failed", slog.String("error", parseErr2.Error()))
		return nil, parseBuildSanitizedInternalStatus(parseCtx, "upsert user memory", parseErr2)
	}
	return &emptypb.Empty{}, nil
}

// DeleteUserMemory deletes one memory row for the current user.
func (parseS *chatServer) DeleteUserMemory(parseCtx context.Context, parseReq *chatpb.DeleteUserMemoryRequest) (*emptypb.Empty, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "DeleteUserMemory"))
	parseLogger = parseLogger.With(parseResolveTraceabilityAttrs(parseCtx)...)
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS.store == nil {
		parseLogger.Error("rpc.DeleteUserMemory: store unavailable; rejecting remembered-preferences delete")
		return nil, parseBuildStoreUnavailableRPCStatus(parseCtx, "/chat.ChatService/DeleteUserMemory", "memory persistence unavailable")
	}
	parseKey := strings.TrimSpace(parseReq.GetKey())
	if parseKey == "" {
		return nil, status.Error(codes.InvalidArgument, "memory key is required")
	}
	if parseErr2 := parseS.store.parseDeleteUserMemory(parseUserID, parseKey); parseErr2 != nil {
		parseLogger.Error("rpc.DeleteUserMemory: db delete failed", slog.String("error", parseErr2.Error()))
		return nil, parseBuildSanitizedInternalStatus(parseCtx, "delete user memory", parseErr2)
	}
	return &emptypb.Empty{}, nil
}

// SynthesizeSpeech streams one speech-synthesis response for the current user.
func (parseS *chatServer) SynthesizeSpeech(parseReq *chatpb.SynthesizeSpeechRequest, parseStream grpc.ServerStreamingServer[chatpb.SynthesizeSpeechChunk]) error {
	parseLogger := parseS.logger.With(parseBuildLogFieldAttrs(parseStream.Context(), parseLogFieldSpec{
		ParseRPC: "SynthesizeSpeech",
	})...)
	if _, parseErr := parseS.parseRequireAuthenticatedUserID(parseStream.Context()); parseErr != nil {
		return parseErr
	}
	if parseS.providerRegistry == nil {
		parseLogger.Error("rpc.SynthesizeSpeech: provider registry unavailable")
		return status.Error(codes.Unavailable, "no configured model provider available")
	}

	parseScript := parseSanitizeTextForTTS(parseReq.GetText())
	if parseScript == "" {
		return status.Error(codes.InvalidArgument, "text must contain speakable content")
	}
	parseResolvedModel := parseS.defaultModel
	if parseRequestedModel := strings.TrimSpace(parseReq.GetModel()); parseRequestedModel != "" {
		parseResolvedModel = parseNormalizeSelectedModelID(parseRequestedModel)
	}
	parseSpeechProvider, parseResolvedModel, _, parseErr2 := parseS.providerRegistry.ParseRequireCapability(parseResolvedModel, provider.CapabilitySpeech)
	if parseErr2 != nil {
		parseLogger.Error("rpc.SynthesizeSpeech: provider resolution failed",
			slog.String("requested_model", parseReq.GetModel()),
			slog.String("resolved_model", parseResolvedModel),
			slog.String("error", parseErr2.Error()),
		)
		if parseCapabilityErr := parseCapabilityStatusError(parseErr2); parseCapabilityErr != nil {
			return parseCapabilityErr
		}
		if strings.TrimSpace(parseReq.GetModel()) != "" {
			return status.Errorf(codes.InvalidArgument, "unsupported model %q", strings.TrimSpace(parseReq.GetModel()))
		}
		return status.Error(codes.Unavailable, "no configured model provider available")
	}
	parseLogger = parseLogger.With(parseBuildLogFieldAttrs(parseStream.Context(), parseLogFieldSpec{
		ParseProvider: parseSpeechProvider.ParseID(),
	})...)

	parseActiveStreams := parseS.activeTTSStreams.Add(1)
	defer parseS.activeTTSStreams.Add(-1)
	parseTotalAudioBytes := 0
	parseSpeechResult, parseErr2 := parseSpeechProvider.ParseSynthesizeSpeech(parseStream.Context(), provider.SpeechRequest{Model: parseResolvedModel, Text: parseScript}, func(parseChunk provider.SpeechChunk) error {
		if len(parseChunk.AudioChunk) > 0 {
			parseTotalAudioBytes += len(parseChunk.AudioChunk)
		}
		if parseErr3 := parseStream.Send(&chatpb.SynthesizeSpeechChunk{
			AudioChunk: parseChunk.AudioChunk,
			Done:       parseChunk.Done,
			MimeType:   parseChunk.MimeType,
			Model:      parseChunk.Model,
			Voice:      parseChunk.Voice,
			Script:     parseChunk.Script,
		}); parseErr3 != nil {
			parseLogger.Warn("rpc.SynthesizeSpeech: downstream stream send failed",
				slog.String("error", parseErr3.Error()),
				slog.Int("audio_bytes", parseTotalAudioBytes),
			)
			return status.Errorf(codes.Canceled, "speech stream send: %v", parseErr3)
		}
		return nil
	})
	if parseErr2 != nil {
		parseLogger.Error("rpc.SynthesizeSpeech: provider call failed",
			slog.String("provider", parseSpeechProvider.ParseID()),
			slog.String("model", parseResolvedModel),
			slog.String("error", parseErr2.Error()),
		)
		if parseCapabilityErr2 := parseCapabilityStatusError(parseErr2); parseCapabilityErr2 != nil {
			return parseCapabilityErr2
		}
		if status.Code(parseErr2) != codes.Unknown {
			return parseErr2
		}
		return status.Errorf(codes.Internal, "synthesize speech: %v", parseErr2)
	}
	if parseTotalAudioBytes == 0 {
		return status.Error(codes.Internal, "synthesized audio was empty")
	}

	parseLogger.Info("rpc.SynthesizeSpeech: complete",
		slog.Int("script_len", len(parseScript)),
		slog.Int("audio_bytes", parseTotalAudioBytes),
		slog.Int64("active_streams", parseActiveStreams),
		slog.String("provider", parseSpeechProvider.ParseID()),
		slog.String("model", parseSpeechResult.Model),
		slog.String("voice", parseSpeechResult.Voice),
	)

	return nil
}

// SetSelectedModel stores the current user's selected model.
func (parseS *chatServer) SetSelectedModel(parseCtx context.Context, parseReq *wrapperspb.StringValue) (*emptypb.Empty, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "SetSelectedModel"))
	parseLogger = parseLogger.With(parseResolveTraceabilityAttrs(parseCtx)...)
	parseSelectedModel := parseNormalizeOptionalSelectedModelID(parseReq.GetValue())
	if parseSelectedModel == "" {
		return nil, status.Error(codes.InvalidArgument, "model must not be empty")
	}
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS.providerRegistry != nil {
		if _, _, parseErr2 := parseS.providerRegistry.ParseResolve(parseSelectedModel); parseErr2 != nil {
			if parseCapabilityErr := parseCapabilityStatusError(parseErr2); parseCapabilityErr != nil {
				return nil, parseCapabilityErr
			}
			return nil, status.Errorf(codes.InvalidArgument, "unsupported model %q", parseSelectedModel)
		}
	}
	parseBillingPolicy, parseBillingPolicyErr := parseS.parseResolveBillingModelPolicy(parseUserID, time.Now().UTC())
	if parseBillingPolicyErr != nil {
		parseLogger.Error("rpc.SetSelectedModel: billing model policy lookup failed", slog.String("error", parseBillingPolicyErr.Error()))
		return nil, parseBuildSanitizedInternalStatus(parseCtx, "resolve billing model policy", parseBillingPolicyErr)
	}
	if !parseHasAllowedModel(parseBillingPolicy, parseSelectedModel) {
		parseDeniedMessage := parseFormatModelDeniedByPlan(parseSelectedModel, parseBillingPolicy)
		parseLogger.Warn("rpc.SetSelectedModel: model denied by billing policy",
			slog.Int64("user_id", parseUserID),
			slog.String("requested_model", parseSelectedModel),
			slog.String("plan_code", parseBillingPolicy.PlanCode),
		)
		return nil, status.Error(codes.FailedPrecondition, parseDeniedMessage)
	}
	if parseS.store == nil {
		parseLogger.Error("rpc.SetSelectedModel: store unavailable; rejecting settings write")
		return nil, status.Error(codes.Unavailable, "settings persistence unavailable")
	}
	if parseErr3 := parseS.store.setSelectedModel(parseUserID, parseSelectedModel); parseErr3 != nil {
		parseLogger.Error("rpc.SetSelectedModel: db upsert failed", slog.String("error", parseErr3.Error()))
		return nil, parseBuildSanitizedInternalStatus(parseCtx, "set selected model", parseErr3)
	}
	parseLogger.Info("rpc.SetSelectedModel: complete", slog.String("model", parseSelectedModel))
	return &emptypb.Empty{}, nil
}

// GetSelectedModel returns the current user's selected model.
func (parseS *chatServer) GetSelectedModel(parseCtx context.Context, _ *emptypb.Empty) (*wrapperspb.StringValue, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetSelectedModel"))
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS.store == nil {
		parseLogger.Warn("rpc.GetSelectedModel: store unavailable, returning default")
		return wrapperspb.String(parseS.defaultModel), nil
	}

	parseFallbackModel := parseNormalizeSelectedModelID(parseS.defaultModel)
	if parseS.providerRegistry != nil {
		parseModelOptions := parseS.providerRegistry.ParseModelOptions()
		if len(parseModelOptions) > 0 {
			parseFirstModel := parseNormalizeSelectedModelID(parseModelOptions[0].ID)
			if parseFirstModel != "" {
				parseFallbackModel = parseFirstModel
			}
		}
		if parseFallbackModel == "" {
			parseFallbackModel = parseNormalizeSelectedModelID(parseS.providerRegistry.ParseDefaultModel())
		}
	}
	if parseFallbackModel == "" {
		parseFallbackModel = parseNormalizeSelectedModelID("")
	}
	parseBillingPolicy, parseBillingPolicyErr := parseS.parseResolveBillingModelPolicy(parseUserID, time.Now().UTC())
	if parseBillingPolicyErr != nil {
		parseLogger.Error("rpc.GetSelectedModel: billing model policy lookup failed", slog.String("error", parseBillingPolicyErr.Error()))
		return nil, status.Errorf(codes.Internal, "resolve billing model policy: %v", parseBillingPolicyErr)
	}

	parseSelectedModel, parseErr := parseS.store.getSelectedModel(parseUserID, parseFallbackModel)
	if parseErr != nil {
		parseLogger.Error("rpc.GetSelectedModel: db query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "get selected model: %v", parseErr)
	}
	parseOriginalSelectedModel := strings.TrimSpace(parseSelectedModel)
	parseSelectedModel = parseNormalizeSelectedModelID(parseSelectedModel)

	if parseS.providerRegistry != nil {
		if _, parseResolvedModel, parseResolveErr := parseS.providerRegistry.ParseResolve(parseSelectedModel); parseResolveErr == nil {
			parseSelectedModel = parseNormalizeSelectedModelID(parseResolvedModel)
		} else if parseFallbackModel != "" {
			parseSelectedModel = parseFallbackModel
		} else if parseRegistryDefault := parseS.providerRegistry.ParseDefaultModel(); parseRegistryDefault != "" {
			parseSelectedModel = parseNormalizeSelectedModelID(parseRegistryDefault)
		}
	}
	parseSelectedModel, _ = parseResolvePlanAwareModel("", parseSelectedModel, parseBillingPolicy)
	if parseSelectedModel == "" {
		parseSelectedModel = parseFallbackModel
	}
	if parseSelectedModel == "" {
		parseSelectedModel = parseNormalizeSelectedModelID("")
	}

	if setErr := parseS.store.setSelectedModel(parseUserID, parseSelectedModel); setErr != nil {
		parseLogger.Warn("rpc.GetSelectedModel: failed to persist repaired model", slog.String("error", setErr.Error()), slog.String("model", parseSelectedModel))
	} else if parseSelectedModel != parseOriginalSelectedModel {
		parseLogger.Info("rpc.GetSelectedModel: repaired persisted model", slog.String("from", parseOriginalSelectedModel), slog.String("to", parseSelectedModel))
	}

	parseLogger.Info("rpc.GetSelectedModel: complete", slog.String("model", parseSelectedModel))
	return wrapperspb.String(parseSelectedModel), nil
}
