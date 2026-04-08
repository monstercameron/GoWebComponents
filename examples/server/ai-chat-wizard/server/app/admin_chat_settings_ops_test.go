package app

import (
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestAdminChatSettingsRPCs verifies typed chat-settings read/write RPCs for superuser and workspace-scoped publishing rows.
func TestAdminChatSettingsRPCs(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseBobAuth, parseErr := parseStore.getUserAuthByEmail("bob@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail bob: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseBobWorkspaceID := parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-chat-settings")

	if parseErr = parseStore.parseUpsertSavedWorkflow(parseSavedWorkflowWrite{
		WorkspaceID:  parseBobWorkspaceID,
		UserID:       parseBobAuth.ID,
		WorkflowKey:  "workflow-bob-settings",
		Name:         "Bob Settings Workflow",
		Description:  "Workflow publish settings fixture",
		WorkflowJSON: `{"steps":["collect","summarize"]}`,
		IsPublic:     false,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSavedWorkflow bob: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertPromptLibraryItem(parsePromptLibraryItemWrite{
		WorkspaceID: parseBobWorkspaceID,
		UserID:      parseBobAuth.ID,
		ItemKey:     "skill-bob-settings",
		Title:       "Bob Skill Prompt",
		Category:    "skill",
		PromptText:  "Act as a concise support copilot.",
		TagsJSON:    `["skill","support"]`,
		IsPublic:    false,
		UseCount:    0,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertPromptLibraryItem bob: %v", parseErr)
	}

	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-chat-settings-alice", parseAliceAuth.ID, parseAliceAuth.Email)
	parseDefaultsResp, parseErr := parseServer.SetAdminChatDefaults(parseAliceCtx, &chatpb.SetAdminChatDefaultsRequest{
		Confirm:                 true,
		Reason:                  "chat defaults refresh",
		DefaultSystemPrompt:     "Be clear and accurate.",
		DefaultModelId:          modelGPT54,
		DefaultReasoningEffort:  "high",
		MemoryExtractionEnabled: true,
		MemoryExtractionModelId: modelGPT54,
	})
	if parseErr != nil {
		parseT.Fatalf("SetAdminChatDefaults: %v", parseErr)
	}
	if parseDefaultsResp.GetDefaultModelId() != modelGPT54 || parseDefaultsResp.GetDefaultReasoningEffort() != "high" {
		parseT.Fatalf("unexpected defaults response: %+v", parseDefaultsResp)
	}

	parseTemplateResp, parseErr := parseServer.SetAdminOnboardingTemplate(parseAliceCtx, &chatpb.SetAdminOnboardingTemplateRequest{
		WorkspaceId:  parseBobWorkspaceID,
		Confirm:      true,
		Reason:       "chat onboarding copy refresh",
		TemplateKey:  "chat-starter-settings",
		Title:        "Chat Starter",
		Category:     "onboarding",
		PromptText:   "Draft my first project update.",
		ChecklistJson: `["set goal","pick audience"]`,
		IsDefault:    true,
		SortOrder:    1,
	})
	if parseErr != nil {
		parseT.Fatalf("SetAdminOnboardingTemplate: %v", parseErr)
	}
	if parseTemplateResp.GetTemplate() == nil || parseTemplateResp.GetTemplate().GetTemplateKey() != "chat-starter-settings" {
		parseT.Fatalf("unexpected onboarding template response: %+v", parseTemplateResp)
	}

	parseWorkflowResp, parseErr := parseServer.SetAdminWorkflowPublishState(parseAliceCtx, &chatpb.SetAdminWorkflowPublishStateRequest{
		WorkspaceId: parseBobWorkspaceID,
		WorkflowKey: "workflow-bob-settings",
		IsPublic:    true,
		Confirm:     true,
		Reason:      "publish workflow to workspace",
	})
	if parseErr != nil {
		parseT.Fatalf("SetAdminWorkflowPublishState: %v", parseErr)
	}
	if parseWorkflowResp.GetWorkflow() == nil || !parseWorkflowResp.GetWorkflow().GetIsPublic() {
		parseT.Fatalf("unexpected workflow publish response: %+v", parseWorkflowResp)
	}

	parseSkillResp, parseErr := parseServer.SetAdminSkillPublishState(parseAliceCtx, &chatpb.SetAdminSkillPublishStateRequest{
		WorkspaceId: parseBobWorkspaceID,
		ItemKey:     "skill-bob-settings",
		IsPublic:    true,
		Confirm:     true,
		Reason:      "publish skill for workspace",
	})
	if parseErr != nil {
		parseT.Fatalf("SetAdminSkillPublishState: %v", parseErr)
	}
	if parseSkillResp.GetSkill() == nil || !parseSkillResp.GetSkill().GetIsPublic() {
		parseT.Fatalf("unexpected skill publish response: %+v", parseSkillResp)
	}

	parseSettingsResp, parseErr := parseServer.GetAdminChatsSettings(parseAliceCtx, &chatpb.GetAdminChatsSettingsRequest{
		WorkspaceId: parseBobWorkspaceID,
		Limit:       25,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminChatsSettings: %v", parseErr)
	}
	if parseSettingsResp.GetDefaultModelId() != modelGPT54 || parseSettingsResp.GetDefaultReasoningEffort() != "high" {
		parseT.Fatalf("unexpected chat settings defaults: %+v", parseSettingsResp)
	}
	if len(parseSettingsResp.GetOnboardingTemplates()) == 0 || len(parseSettingsResp.GetWorkflows()) == 0 || len(parseSettingsResp.GetSkills()) == 0 {
		parseT.Fatalf(
			"expected non-empty chat settings slices, got onboarding=%d workflows=%d skills=%d",
			len(parseSettingsResp.GetOnboardingTemplates()),
			len(parseSettingsResp.GetWorkflows()),
			len(parseSettingsResp.GetSkills()),
		)
	}
}

// TestAdminChatSettingsScopeGuards verifies workspace-admin boundaries for chat-settings reads and mutations.
func TestAdminChatSettingsScopeGuards(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseBobAuth, parseErr := parseStore.getUserAuthByEmail("bob@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail bob: %v", parseErr)
	}
	parseBobWorkspaceID := parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-chat-settings-scope")
	parseAliceWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseAliceAuth.ID, "ws-alice-chat-settings-scope")

	if parseErr = parseStore.parseUpsertSavedWorkflow(parseSavedWorkflowWrite{
		WorkspaceID:  parseAliceWorkspaceID,
		UserID:       parseAliceAuth.ID,
		WorkflowKey:  "workflow-alice-scope",
		Name:         "Alice Scope Workflow",
		Description:  "Out-of-scope workflow fixture",
		WorkflowJSON: `{"steps":["audit"]}`,
		IsPublic:     false,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSavedWorkflow alice: %v", parseErr)
	}
	if parseErr = parseStore.parseUpsertPromptLibraryItem(parsePromptLibraryItemWrite{
		WorkspaceID: parseAliceWorkspaceID,
		UserID:      parseAliceAuth.ID,
		ItemKey:     "skill-alice-scope",
		Title:       "Alice Scope Skill",
		Category:    "skill",
		PromptText:  "Out-of-scope skill fixture",
		TagsJSON:    `["skill"]`,
		IsPublic:    false,
		UseCount:    0,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertPromptLibraryItem alice: %v", parseErr)
	}

	parseBobCtx := parseBindAuthUser(parseServer, "peer-admin-chat-settings-bob", parseBobAuth.ID, parseBobAuth.Email)
	if _, parseErr = parseServer.SetAdminChatDefaults(parseBobCtx, &chatpb.SetAdminChatDefaultsRequest{
		Confirm:                 true,
		Reason:                  "workspace-admin should be denied",
		DefaultSystemPrompt:     "Denied prompt",
		DefaultModelId:          modelGPT54,
		DefaultReasoningEffort:  "high",
		MemoryExtractionEnabled: true,
		MemoryExtractionModelId: modelGPT54,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("SetAdminChatDefaults workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.GetAdminChatsSettings(parseBobCtx, &chatpb.GetAdminChatsSettingsRequest{
		WorkspaceId: parseAliceWorkspaceID,
		Limit:       10,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("GetAdminChatsSettings out-of-scope workspace status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.SetAdminWorkflowPublishState(parseBobCtx, &chatpb.SetAdminWorkflowPublishStateRequest{
		WorkspaceId: parseAliceWorkspaceID,
		WorkflowKey: "workflow-alice-scope",
		IsPublic:    true,
		Confirm:     true,
		Reason:      "workspace-admin out-of-scope",
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("SetAdminWorkflowPublishState out-of-scope status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.SetAdminSkillPublishState(parseBobCtx, &chatpb.SetAdminSkillPublishStateRequest{
		WorkspaceId: parseAliceWorkspaceID,
		ItemKey:     "skill-alice-scope",
		IsPublic:    true,
		Confirm:     true,
		Reason:      "workspace-admin out-of-scope",
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("SetAdminSkillPublishState out-of-scope status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.SetAdminOnboardingTemplate(parseBobCtx, &chatpb.SetAdminOnboardingTemplateRequest{
		WorkspaceId: parseBobWorkspaceID,
		Confirm:     true,
		Reason:      "workspace-admin in-scope",
		TemplateKey: "chat-template-bob-scope",
		Title:       "Bob Scope Template",
		Category:    "onboarding",
		PromptText:  "Bob in-scope template",
	}); parseErr != nil {
		parseT.Fatalf("SetAdminOnboardingTemplate workspace-admin in-scope: %v", parseErr)
	}
}
