package app

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const parseChatSettingsDefaultSystemPromptKey = "chat.default_system_prompt"
const parseChatSettingsDefaultModelKey = "chat.default_model_id"
const parseChatSettingsDefaultReasoningKey = "chat.default_reasoning_effort"
const parseChatSettingsMemoryExtractionEnabledKey = "chat.memory_extraction_enabled"
const parseChatSettingsMemoryExtractionModelKey = "chat.memory_extraction_model_id"

// parseBuildSiteConfigRowByKey indexes site-config rows by normalized config key.
func parseBuildSiteConfigRowByKey(parseRows []parseSiteConfigRow) map[string]parseSiteConfigRow {
	parseByKey := make(map[string]parseSiteConfigRow, len(parseRows))
	for _, parseRow := range parseRows {
		parseByKey[parseNormalizeSUKey(parseRow.ConfigKey)] = parseRow
	}
	return parseByKey
}

// parseResolveSiteConfigValue resolves one normalized site-config key with one fallback value.
func parseResolveSiteConfigValue(parseByKey map[string]parseSiteConfigRow, parseConfigKey string, parseFallback string) string {
	parseRow, hasParseRow := parseByKey[parseNormalizeSUKey(parseConfigKey)]
	if !hasParseRow {
		return strings.TrimSpace(parseFallback)
	}
	parseValue := strings.TrimSpace(parseRow.ConfigValue)
	if parseValue == "" {
		return strings.TrimSpace(parseFallback)
	}
	return parseValue
}

// parseResolveSiteConfigBool resolves one normalized site-config boolean key with one fallback value.
func parseResolveSiteConfigBool(parseByKey map[string]parseSiteConfigRow, parseConfigKey string, isParseFallback bool) bool {
	parseRow, hasParseRow := parseByKey[parseNormalizeSUKey(parseConfigKey)]
	if !hasParseRow {
		return isParseFallback
	}
	switch strings.TrimSpace(strings.ToLower(parseRow.ConfigValue)) {
	case "1", "true", "yes", "on", "enabled":
		return true
	case "0", "false", "no", "off", "disabled":
		return false
	default:
		return isParseFallback
	}
}

// parseResolveChatsSettingsWorkspaceID resolves one effective workspace target for chat settings reads.
func parseResolveChatsSettingsWorkspaceID(parseScope parseAdminAccessScope, parseWorkspaceID int64) (int64, error) {
	if parseWorkspaceID > 0 {
		if parseScope.isPlatformScope {
			return parseWorkspaceID, nil
		}
		if _, hasParseWorkspace := parseScope.workspaceIDs[parseWorkspaceID]; hasParseWorkspace {
			return parseWorkspaceID, nil
		}
		return 0, status.Error(codes.PermissionDenied, "workspace outside workspace-admin scope")
	}
	if parseScope.isPlatformScope {
		return 0, nil
	}
	parseResolvedWorkspaceID := parseResolveAdminScopeWorkspaceID(parseScope.workspaceIDs)
	if parseResolvedWorkspaceID <= 0 {
		return 0, status.Error(codes.InvalidArgument, "workspace id is required for workspace-admin chat settings")
	}
	return parseResolvedWorkspaceID, nil
}

// parseFindSavedWorkflowRowByWorkspaceAndKey resolves one saved-workflow row by workspace and workflow key.
func parseFindSavedWorkflowRowByWorkspaceAndKey(parseRows []parseSavedWorkflowRow, parseWorkspaceID int64, parseWorkflowKey string) (parseSavedWorkflowRow, bool) {
	parseWorkflowKey = strings.TrimSpace(parseWorkflowKey)
	for _, parseRow := range parseRows {
		if parseRow.WorkspaceID != parseWorkspaceID {
			continue
		}
		if strings.TrimSpace(parseRow.WorkflowKey) != parseWorkflowKey {
			continue
		}
		return parseRow, true
	}
	return parseSavedWorkflowRow{}, false
}

// parseFindPromptLibraryItemRowByWorkspaceAndKey resolves one prompt-library row by workspace and item key.
func parseFindPromptLibraryItemRowByWorkspaceAndKey(parseRows []parsePromptLibraryItemRow, parseWorkspaceID int64, parseItemKey string) (parsePromptLibraryItemRow, bool) {
	parseItemKey = strings.TrimSpace(parseItemKey)
	for _, parseRow := range parseRows {
		if parseRow.WorkspaceID != parseWorkspaceID {
			continue
		}
		if strings.TrimSpace(parseRow.ItemKey) != parseItemKey {
			continue
		}
		return parseRow, true
	}
	return parsePromptLibraryItemRow{}, false
}

// parseHasPromptLibrarySkillCategory reports whether one prompt-library category should be treated as a skill row.
func parseHasPromptLibrarySkillCategory(parseCategory string) bool {
	parseCategory = strings.TrimSpace(strings.ToLower(parseCategory))
	return parseCategory == "skill" || strings.Contains(parseCategory, "skill")
}

// GetAdminChatsSettings returns typed chats settings defaults plus onboarding/workflow/skill publishing rows.
func (parseS *chatServer) GetAdminChatsSettings(parseCtx context.Context, parseReq *chatpb.GetAdminChatsSettingsRequest) (*chatpb.GetAdminChatsSettingsResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetAdminChatsSettings"))
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.conversations")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseWorkspaceID int64
	var parseLimit int32
	if parseReq != nil {
		parseWorkspaceID = parseReq.GetWorkspaceId()
		parseLimit = parseReq.GetLimit()
	}
	parseLimit = parseClampAdminListLimit(parseLimit)
	parseWorkspaceID, parseErr = parseResolveChatsSettingsWorkspaceID(parseScope, parseWorkspaceID)
	if parseErr != nil {
		return nil, parseErr
	}
	parseResponse := &chatpb.GetAdminChatsSettingsResponse{
		WorkspaceId:             parseWorkspaceID,
		DefaultSystemPrompt:     defaultCustomSystemPromptTemplate,
		DefaultModelId:          parseS.defaultModel,
		DefaultReasoningEffort:  defaultThinkingEffort,
		MemoryExtractionEnabled: true,
		MemoryExtractionModelId: parseS.memoryExtractionModel,
		OnboardingTemplates:     make([]*chatpb.OnboardingTemplateEntry, 0),
		Workflows:               make([]*chatpb.SavedWorkflowEntry, 0),
		Skills:                  make([]*chatpb.PromptLibraryItemEntry, 0),
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.slice.view",
		"slice",
		"chat-settings",
		"Admin chat settings viewed",
		"{}",
		parseWorkspaceID,
	)
	if parseS.store == nil {
		parseLogger.Warn(
			"rpc.GetAdminChatsSettings: store unavailable",
			slog.String("next_action", "restore store availability before retrying chat settings"),
		)
		return parseResponse, nil
	}
	parseSiteConfigRows, parseErr := parseS.store.parseListSiteConfigs()
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminChatsSettings: site config query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list chat settings site config: %v", parseErr)
	}
	parseConfigByKey := parseBuildSiteConfigRowByKey(parseSiteConfigRows)
	parseResponse.DefaultSystemPrompt = parseResolveSiteConfigValue(parseConfigByKey, parseChatSettingsDefaultSystemPromptKey, defaultCustomSystemPromptTemplate)
	parseResponse.DefaultModelId = parseNormalizeSelectedModelID(parseResolveSiteConfigValue(parseConfigByKey, parseChatSettingsDefaultModelKey, parseS.defaultModel))
	parseResponse.DefaultReasoningEffort = parseNormalizeSelectedThinkingEffort(parseResolveSiteConfigValue(parseConfigByKey, parseChatSettingsDefaultReasoningKey, defaultThinkingEffort))
	parseResponse.MemoryExtractionEnabled = parseResolveSiteConfigBool(parseConfigByKey, parseChatSettingsMemoryExtractionEnabledKey, true)
	parseResponse.MemoryExtractionModelId = parseNormalizeSelectedModelID(parseResolveSiteConfigValue(parseConfigByKey, parseChatSettingsMemoryExtractionModelKey, parseS.memoryExtractionModel))

	parseTemplateRows, parseErr := parseS.store.parseListOnboardingTemplates(int64(parseLimit))
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminChatsSettings: onboarding-template query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list chat settings onboarding templates: %v", parseErr)
	}
	for _, parseTemplateRow := range parseTemplateRows {
		parseResponse.OnboardingTemplates = append(parseResponse.OnboardingTemplates, parseBuildOnboardingTemplateEntry(parseTemplateRow))
	}
	parseWorkflowRows, parseErr := parseS.store.parseListSavedWorkflows(parseAdminScopedScanLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminChatsSettings: workflow query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list chat settings workflows: %v", parseErr)
	}
	if parseWorkspaceID > 0 {
		parseWorkflowRows = parseFilterSavedWorkflowRowsByScope(parseWorkflowRows, map[int64]struct{}{parseWorkspaceID: {}})
	}
	parseWorkflowRows = parseApplyAdminSliceWindow(parseWorkflowRows, 0, parseLimit)
	for _, parseWorkflowRow := range parseWorkflowRows {
		parseResponse.Workflows = append(parseResponse.Workflows, parseBuildSavedWorkflowEntry(parseWorkflowRow))
	}
	parsePromptRows, parseErr := parseS.store.parseListPromptLibraryItems(parseAdminScopedScanLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminChatsSettings: prompt-library query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list chat settings prompt library: %v", parseErr)
	}
	if parseWorkspaceID > 0 {
		parsePromptRows = parseFilterPromptLibraryRowsByScope(parsePromptRows, map[int64]struct{}{parseWorkspaceID: {}})
	}
	parseSkillRows := make([]parsePromptLibraryItemRow, 0, len(parsePromptRows))
	for _, parsePromptRow := range parsePromptRows {
		if !parseHasPromptLibrarySkillCategory(parsePromptRow.Category) {
			continue
		}
		parseSkillRows = append(parseSkillRows, parsePromptRow)
	}
	parseSkillRows = parseApplyAdminSliceWindow(parseSkillRows, 0, parseLimit)
	for _, parseSkillRow := range parseSkillRows {
		parseResponse.Skills = append(parseResponse.Skills, parseBuildPromptLibraryItemEntry(parseSkillRow))
	}
	return parseResponse, nil
}

// SetAdminChatDefaults updates platform-wide default prompt/model/reasoning/memory-extraction settings.
func (parseS *chatServer) SetAdminChatDefaults(parseCtx context.Context, parseReq *chatpb.SetAdminChatDefaultsRequest) (*chatpb.SetAdminChatDefaultsResponse, error) {
	if parseReq == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseScope, parseErr := parseS.parseAuthorizeAdminChatMutationScope(parseCtx, parseAdminChatMutationSetSystemDefaultPrompt, parseAdminChatMutationTarget{})
	if parseErr != nil {
		return nil, parseErr
	}
	if _, parseErr = parseS.parseAuthorizeAdminChatMutationScope(parseCtx, parseAdminChatMutationSetModelDefault, parseAdminChatMutationTarget{}); parseErr != nil {
		return nil, parseErr
	}
	if _, parseErr = parseS.parseAuthorizeAdminChatMutationScope(parseCtx, parseAdminChatMutationSetMemoryRule, parseAdminChatMutationTarget{}); parseErr != nil {
		return nil, parseErr
	}
	var parseReason string
	var parseDefaultSystemPrompt string
	var parseDefaultModelID string
	var parseDefaultReasoningEffort string
	var parseMemoryExtractionModelID string
	var isParseMemoryExtractionEnabled bool
	parseReason, parseErr = parseRequireAdminControlMutationConfirmation(parseReq.GetConfirm(), parseReq.GetReason())
	if parseErr != nil {
		return nil, parseErr
	}
	parseDefaultSystemPrompt = strings.TrimSpace(parseReq.GetDefaultSystemPrompt())
	parseDefaultModelID = parseNormalizeSelectedModelID(parseReq.GetDefaultModelId())
	parseDefaultReasoningEffort = parseNormalizeSelectedThinkingEffort(parseReq.GetDefaultReasoningEffort())
	isParseMemoryExtractionEnabled = parseReq.GetMemoryExtractionEnabled()
	parseMemoryExtractionModelID = parseNormalizeSelectedModelID(parseReq.GetMemoryExtractionModelId())
	if parseDefaultSystemPrompt == "" {
		parseDefaultSystemPrompt = defaultCustomSystemPromptTemplate
	}
	if parseDefaultModelID == "" {
		parseDefaultModelID = parseNormalizeSelectedModelID(parseS.defaultModel)
	}
	if parseDefaultModelID == "" {
		return nil, status.Error(codes.InvalidArgument, "default model id is required")
	}
	if parseDefaultReasoningEffort == "" {
		parseDefaultReasoningEffort = defaultThinkingEffort
	}
	if parseMemoryExtractionModelID == "" {
		parseMemoryExtractionModelID = parseNormalizeSelectedModelID(parseS.memoryExtractionModel)
		if parseMemoryExtractionModelID == "" {
			parseMemoryExtractionModelID = parseDefaultModelID
		}
	}
	parseConfigRows := []parseSiteConfigWrite{
		{
			ConfigKey:       parseChatSettingsDefaultSystemPromptKey,
			ConfigValue:     parseDefaultSystemPrompt,
			ValueType:       "string",
			Description:     "Default chat system prompt",
			UpdatedByUserID: parseScope.adminUserID,
		},
		{
			ConfigKey:       parseChatSettingsDefaultModelKey,
			ConfigValue:     parseDefaultModelID,
			ValueType:       "string",
			Description:     "Default chat model id",
			UpdatedByUserID: parseScope.adminUserID,
		},
		{
			ConfigKey:       parseChatSettingsDefaultReasoningKey,
			ConfigValue:     parseDefaultReasoningEffort,
			ValueType:       "string",
			Description:     "Default chat reasoning effort",
			UpdatedByUserID: parseScope.adminUserID,
		},
		{
			ConfigKey:       parseChatSettingsMemoryExtractionEnabledKey,
			ConfigValue:     strconv.FormatBool(isParseMemoryExtractionEnabled),
			ValueType:       "bool",
			Description:     "Memory extraction enabled",
			UpdatedByUserID: parseScope.adminUserID,
		},
		{
			ConfigKey:       parseChatSettingsMemoryExtractionModelKey,
			ConfigValue:     parseMemoryExtractionModelID,
			ValueType:       "string",
			Description:     "Memory extraction model id",
			UpdatedByUserID: parseScope.adminUserID,
		},
	}
	for _, parseConfigRow := range parseConfigRows {
		if parseErr = parseS.store.parseUpsertSiteConfig(parseConfigRow); parseErr != nil {
			return nil, status.Errorf(codes.Internal, "set chat default config %s: %v", parseConfigRow.ConfigKey, parseErr)
		}
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.chat.defaults.updated",
		"chat_settings",
		"defaults",
		"Admin chat defaults updated",
		"{}",
		0,
	)
	if parseS.logger != nil {
		parseS.logger.Info(
			"rpc.SetAdminChatDefaults: complete",
			slog.Int64("admin_user_id", parseScope.adminUserID),
			slog.String("reason", parseReason),
		)
	}
	return &chatpb.SetAdminChatDefaultsResponse{
		DefaultSystemPrompt:     parseDefaultSystemPrompt,
		DefaultModelId:          parseDefaultModelID,
		DefaultReasoningEffort:  parseDefaultReasoningEffort,
		MemoryExtractionEnabled: isParseMemoryExtractionEnabled,
		MemoryExtractionModelId: parseMemoryExtractionModelID,
	}, nil
}

// SetAdminOnboardingTemplate upserts one onboarding template used by chats onboarding flows.
func (parseS *chatServer) SetAdminOnboardingTemplate(parseCtx context.Context, parseReq *chatpb.SetAdminOnboardingTemplateRequest) (*chatpb.SetAdminOnboardingTemplateResponse, error) {
	if parseReq == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	var parseWorkspaceID int64
	var parseReason string
	parseWorkspaceID = parseReq.GetWorkspaceId()
	parseScope, parseErr := parseS.parseAuthorizeAdminChatMutationScope(parseCtx, parseAdminChatMutationSetOnboardingTemplate, parseAdminChatMutationTarget{workspaceID: parseWorkspaceID})
	if parseErr != nil {
		return nil, parseErr
	}
	parseReason, parseErr = parseRequireAdminControlMutationConfirmation(parseReq.GetConfirm(), parseReq.GetReason())
	if parseErr != nil {
		return nil, parseErr
	}
	if parseErr = parseS.store.parseUpsertOnboardingTemplate(parseOnboardingTemplateWrite{
		TemplateKey:   parseReq.GetTemplateKey(),
		Title:         parseReq.GetTitle(),
		Category:      parseReq.GetCategory(),
		PromptText:    parseReq.GetPromptText(),
		ChecklistJSON: parseReq.GetChecklistJson(),
		IsDefault:     parseReq.GetIsDefault(),
		SortOrder:     parseReq.GetSortOrder(),
	}); parseErr != nil {
		return nil, status.Errorf(codes.Internal, "set onboarding template: %v", parseErr)
	}
	parseTemplateRows, parseErr := parseS.store.parseListOnboardingTemplates(parseAdminScopedScanLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list onboarding templates: %v", parseErr)
	}
	parseTemplateKey := strings.TrimSpace(parseReq.GetTemplateKey())
	for _, parseTemplateRow := range parseTemplateRows {
		if strings.TrimSpace(parseTemplateRow.TemplateKey) != parseTemplateKey {
			continue
		}
		parseS.parseTrackAdminAuditEvent(
			parseScope,
			"admin.chat.onboarding_template.updated",
			"onboarding_template",
			parseTemplateKey,
			"Admin onboarding template updated",
			"{}",
			parseWorkspaceID,
		)
		if parseS.logger != nil {
			parseS.logger.Info(
				"rpc.SetAdminOnboardingTemplate: complete",
				slog.Int64("admin_user_id", parseScope.adminUserID),
				slog.Int64("workspace_id", parseWorkspaceID),
				slog.String("template_key", parseTemplateKey),
				slog.String("reason", parseReason),
			)
		}
		return &chatpb.SetAdminOnboardingTemplateResponse{
			WorkspaceId: parseWorkspaceID,
			Template:    parseBuildOnboardingTemplateEntry(parseTemplateRow),
		}, nil
	}
	return nil, status.Error(codes.NotFound, "onboarding template not found after update")
}

// SetAdminWorkflowPublishState updates publish visibility for one workspace workflow.
func (parseS *chatServer) SetAdminWorkflowPublishState(parseCtx context.Context, parseReq *chatpb.SetAdminWorkflowPublishStateRequest) (*chatpb.SetAdminWorkflowPublishStateResponse, error) {
	if parseReq == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	var parseWorkspaceID int64
	var parseWorkflowKey string
	var isParsePublic bool
	var parseReason string
	parseWorkspaceID = parseReq.GetWorkspaceId()
	parseWorkflowKey = parseReq.GetWorkflowKey()
	isParsePublic = parseReq.GetIsPublic()
	parseScope, parseErr := parseS.parseAuthorizeAdminChatMutationScope(parseCtx, parseAdminChatMutationPublishWorkflow, parseAdminChatMutationTarget{workspaceID: parseWorkspaceID})
	if parseErr != nil {
		return nil, parseErr
	}
	parseReason, parseErr = parseRequireAdminControlMutationConfirmation(parseReq.GetConfirm(), parseReq.GetReason())
	if parseErr != nil {
		return nil, parseErr
	}
	parseWorkflowRows, parseErr := parseS.store.parseListSavedWorkflows(parseAdminScopedScanLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list saved workflows: %v", parseErr)
	}
	parseWorkflowRow, hasParseWorkflow := parseFindSavedWorkflowRowByWorkspaceAndKey(parseWorkflowRows, parseWorkspaceID, parseWorkflowKey)
	if !hasParseWorkflow {
		return nil, status.Error(codes.NotFound, "workflow not found")
	}
	if parseErr = parseS.store.parseUpsertSavedWorkflow(parseSavedWorkflowWrite{
		WorkspaceID:  parseWorkflowRow.WorkspaceID,
		UserID:       parseWorkflowRow.UserID,
		WorkflowKey:  parseWorkflowRow.WorkflowKey,
		Name:         parseWorkflowRow.Name,
		Description:  parseWorkflowRow.Description,
		WorkflowJSON: parseWorkflowRow.WorkflowJSON,
		IsPublic:     isParsePublic,
	}); parseErr != nil {
		return nil, status.Errorf(codes.Internal, "set workflow publish state: %v", parseErr)
	}
	parseUpdatedRows, parseErr := parseS.store.parseListSavedWorkflows(parseAdminScopedScanLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list updated workflows: %v", parseErr)
	}
	parseUpdatedRow, hasParseUpdated := parseFindSavedWorkflowRowByWorkspaceAndKey(parseUpdatedRows, parseWorkspaceID, parseWorkflowKey)
	if !hasParseUpdated {
		return nil, status.Error(codes.NotFound, "workflow not found after update")
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.chat.workflow.publish_state.updated",
		"workflow",
		parseWorkflowKey,
		"Admin workflow publish state updated",
		"{}",
		parseWorkspaceID,
	)
	if parseS.logger != nil {
		parseS.logger.Info(
			"rpc.SetAdminWorkflowPublishState: complete",
			slog.Int64("admin_user_id", parseScope.adminUserID),
			slog.Int64("workspace_id", parseWorkspaceID),
			slog.String("workflow_key", parseWorkflowKey),
			slog.Bool("is_public", isParsePublic),
			slog.String("reason", parseReason),
		)
	}
	return &chatpb.SetAdminWorkflowPublishStateResponse{
		WorkspaceId: parseWorkspaceID,
		Workflow:    parseBuildSavedWorkflowEntry(parseUpdatedRow),
	}, nil
}

// SetAdminSkillPublishState updates publish visibility for one workspace skill-style prompt item.
func (parseS *chatServer) SetAdminSkillPublishState(parseCtx context.Context, parseReq *chatpb.SetAdminSkillPublishStateRequest) (*chatpb.SetAdminSkillPublishStateResponse, error) {
	if parseReq == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	var parseWorkspaceID int64
	var parseItemKey string
	var isParsePublic bool
	var parseReason string
	parseWorkspaceID = parseReq.GetWorkspaceId()
	parseItemKey = parseReq.GetItemKey()
	isParsePublic = parseReq.GetIsPublic()
	parseScope, parseErr := parseS.parseAuthorizeAdminChatMutationScope(parseCtx, parseAdminChatMutationPublishSkill, parseAdminChatMutationTarget{workspaceID: parseWorkspaceID})
	if parseErr != nil {
		return nil, parseErr
	}
	parseReason, parseErr = parseRequireAdminControlMutationConfirmation(parseReq.GetConfirm(), parseReq.GetReason())
	if parseErr != nil {
		return nil, parseErr
	}
	parsePromptRows, parseErr := parseS.store.parseListPromptLibraryItems(parseAdminScopedScanLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list prompt library items: %v", parseErr)
	}
	parsePromptRow, hasParsePrompt := parseFindPromptLibraryItemRowByWorkspaceAndKey(parsePromptRows, parseWorkspaceID, parseItemKey)
	if !hasParsePrompt {
		return nil, status.Error(codes.NotFound, "skill item not found")
	}
	if parseErr = parseS.store.parseUpsertPromptLibraryItem(parsePromptLibraryItemWrite{
		WorkspaceID: parsePromptRow.WorkspaceID,
		UserID:      parsePromptRow.UserID,
		ItemKey:     parsePromptRow.ItemKey,
		Title:       parsePromptRow.Title,
		Category:    parsePromptRow.Category,
		PromptText:  parsePromptRow.PromptText,
		TagsJSON:    parsePromptRow.TagsJSON,
		IsPublic:    isParsePublic,
		UseCount:    parsePromptRow.UseCount,
	}); parseErr != nil {
		return nil, status.Errorf(codes.Internal, "set skill publish state: %v", parseErr)
	}
	parseUpdatedRows, parseErr := parseS.store.parseListPromptLibraryItems(parseAdminScopedScanLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list updated prompt library items: %v", parseErr)
	}
	parseUpdatedRow, hasParseUpdated := parseFindPromptLibraryItemRowByWorkspaceAndKey(parseUpdatedRows, parseWorkspaceID, parseItemKey)
	if !hasParseUpdated {
		return nil, status.Error(codes.NotFound, "skill item not found after update")
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.chat.skill.publish_state.updated",
		"skill",
		parseItemKey,
		"Admin skill publish state updated",
		"{}",
		parseWorkspaceID,
	)
	if parseS.logger != nil {
		parseS.logger.Info(
			"rpc.SetAdminSkillPublishState: complete",
			slog.Int64("admin_user_id", parseScope.adminUserID),
			slog.Int64("workspace_id", parseWorkspaceID),
			slog.String("item_key", parseItemKey),
			slog.Bool("is_public", isParsePublic),
			slog.String("reason", parseReason),
		)
	}
	return &chatpb.SetAdminSkillPublishStateResponse{
		WorkspaceId: parseWorkspaceID,
		Skill:       parseBuildPromptLibraryItemEntry(parseUpdatedRow),
	}, nil
}
