//go:build js && wasm

package app

import (
	"fmt"
)

type editableUserMemory struct {
	Key             string
	Category        string
	Summary         string
	Detail          string
	SourceMessage   string
	UsefulnessScore int
	ConfidenceScore float64
	RubricReason    string
	UpdatedAt       string
}

type appState struct {
	Messages                []message
	ExpandedThoughtSections map[string]bool
	CanvasSession           canvasSessionState
	InputText               string
	Streaming               bool
	ModelOptions            []modelOption
	DefaultModel            string
	SelectedModel           string
	ConversationList        []convSummary
	ActiveConvID            int64
	ActiveConvPublicID      string
	DeleteTarget            int64
	SelectedTone            string
	CustomSystemPrompt      string
	SelectedThinkingEnabled bool
	SelectedThinkingEffort  string
	AuthResolved            bool
	Authenticated           bool
	AuthMode                string
	AuthError               string
	AuthSubmitting          bool
	AuthEmail               string
	AuthPassword            string
	AuthDisplayName         string
	SessionEmail            string
	ShowNameModal           bool
	ActiveSettingsSection   string
	NameInput               string
	ToneInput               string
	ThinkingEnabledInput    bool
	ThinkingEffortInput     string
	SystemPromptInput       string
	UserMemories            []editableUserMemory
	DeletedUserMemoryKeys   []string
	LocaleInput             string
	GRPCReady               bool
	MarkdownWorkerFallback  bool
	EditIdx                 int
	EditText                string
}

type appActionType string

const (
	appActionSetMessages                appActionType = "set_messages"
	appActionUpdateMessages             appActionType = "update_messages"
	appActionToggleThoughtSection       appActionType = "toggle_thought_section"
	appActionSetInputText               appActionType = "set_input_text"
	appActionOpenCanvasSession          appActionType = "open_canvas_session"
	appActionCloseCanvasSession         appActionType = "close_canvas_session"
	appActionApplyCanvasArtifact        appActionType = "apply_canvas_artifact"
	appActionSetCanvasLayoutMode        appActionType = "set_canvas_layout_mode"
	appActionSetCanvasSplitRatio        appActionType = "set_canvas_split_ratio"
	appActionSetCanvasFocus             appActionType = "set_canvas_focus"
	appActionSetCanvasFocusDraft        appActionType = "set_canvas_focus_draft"
	appActionApplyCanvasFocusDraft      appActionType = "apply_canvas_focus_draft"
	appActionCanvasRefreshPreview       appActionType = "canvas_refresh_preview"
	appActionCanvasToggleConsole        appActionType = "canvas_toggle_console"
	appActionCanvasClearConsole         appActionType = "canvas_clear_console"
	appActionCanvasAppendConsole        appActionType = "canvas_append_console"
	appActionCanvasSetStatus            appActionType = "canvas_set_status"
	appActionCanvasResetOriginal        appActionType = "canvas_reset_original"
	appActionCanvasRevertLastPatch      appActionType = "canvas_revert_last_patch"
	appActionSetStreaming               appActionType = "set_streaming"
	appActionSetModelCatalog            appActionType = "set_model_catalog"
	appActionSetSelectedModel           appActionType = "set_selected_model"
	appActionSetConversationList        appActionType = "set_conversation_list"
	appActionSetActiveConvID            appActionType = "set_active_conv_id"
	appActionSetDeleteTarget            appActionType = "set_delete_target"
	appActionSetSelectedTone            appActionType = "set_selected_tone"
	appActionSetCustomSystemPrompt      appActionType = "set_custom_system_prompt"
	appActionSetSelectedThinkingEnabled appActionType = "set_selected_thinking_enabled"
	appActionSetSelectedThinkingEffort  appActionType = "set_selected_thinking_effort"
	appActionSetAuthResolved            appActionType = "set_auth_resolved"
	appActionSetAuthenticated           appActionType = "set_authenticated"
	appActionSetAuthMode                appActionType = "set_auth_mode"
	appActionSetAuthError               appActionType = "set_auth_error"
	appActionSetAuthSubmitting          appActionType = "set_auth_submitting"
	appActionSetAuthEmail               appActionType = "set_auth_email"
	appActionSetAuthPassword            appActionType = "set_auth_password"
	appActionSetAuthDisplayName         appActionType = "set_auth_display_name"
	appActionSetSessionEmail            appActionType = "set_session_email"
	appActionResetWorkspace             appActionType = "reset_workspace"
	appActionSetShowNameModal           appActionType = "set_show_name_modal"
	appActionSetActiveSettingsSection   appActionType = "set_active_settings_section"
	appActionSetNameInput               appActionType = "set_name_input"
	appActionSetToneInput               appActionType = "set_tone_input"
	appActionSetThinkingEnabledInput    appActionType = "set_thinking_enabled_input"
	appActionSetThinkingEffortInput     appActionType = "set_thinking_effort_input"
	appActionSetSystemPromptInput       appActionType = "set_system_prompt_input"
	appActionSetUserMemories            appActionType = "set_user_memories"
	appActionAddUserMemory              appActionType = "add_user_memory"
	appActionUpdateUserMemoryField      appActionType = "update_user_memory_field"
	appActionDeleteUserMemory           appActionType = "delete_user_memory"
	appActionSetLocaleInput             appActionType = "set_locale_input"
	appActionSetGRPCReady               appActionType = "set_grpc_ready"
	appActionSetMarkdownWorkerFallback  appActionType = "set_markdown_worker_fallback"
	appActionSetEditIdx                 appActionType = "set_edit_idx"
	appActionSetEditText                appActionType = "set_edit_text"
)

type appAction struct {
	Type appActionType

	Messages                []message
	UpdateMessages          func([]message) []message
	ThoughtSectionKey       string
	InputText               string
	CanvasArtifact          canvasArtifact
	CanvasLayoutMode        string
	CanvasSplitRatio        float64
	CanvasFocus             canvasFocusRegion
	CanvasFocusDraft        string
	CanvasPreviewStatus     string
	CanvasRuntimeStatus     string
	CanvasConsoleEntry      canvasConsoleEntry
	Streaming               bool
	ModelOptions            []modelOption
	DefaultModel            string
	SelectedModel           string
	ConversationList        []convSummary
	ActiveConvID            int64
	ActiveConvPublicID      string
	DeleteTarget            int64
	SelectedTone            string
	CustomSystemPrompt      string
	SelectedThinkingEnabled bool
	SelectedThinkingEffort  string
	AuthResolved            bool
	Authenticated           bool
	AuthMode                string
	AuthError               string
	AuthSubmitting          bool
	AuthEmail               string
	AuthPassword            string
	AuthDisplayName         string
	SessionEmail            string
	ShowNameModal           bool
	ActiveSettingsSection   string
	NameInput               string
	ToneInput               string
	ThinkingEnabledInput    bool
	ThinkingEffortInput     string
	SystemPromptInput       string
	UserMemories            []editableUserMemory
	UserMemoryIndex         int
	UserMemoryField         string
	UserMemoryValue         string
	DeletedUserMemoryKeys   []string
	LocaleInput             string
	GRPCReady               bool
	MarkdownWorkerFallback  bool
	EditIdx                 int
	EditText                string
}

func initialAppState() appState {
	return appState{
		Messages:                []message{},
		ExpandedThoughtSections: map[string]bool{},
		CanvasSession: canvasSessionState{
			LayoutMode:    canvasLayoutHidden,
			SplitRatio:    canvasSplitDefault,
			PreviewStatus: canvasPreviewNotRendered,
			RuntimeStatus: "idle",
		},
		InputText:               "",
		Streaming:               false,
		ModelOptions:            nil,
		DefaultModel:            defaultModel,
		SelectedModel:           defaultModel,
		ConversationList:        []convSummary{},
		ActiveConvID:            0,
		ActiveConvPublicID:      "",
		DeleteTarget:            0,
		SelectedTone:            defaultTone,
		CustomSystemPrompt:      "",
		SelectedThinkingEnabled: defaultThinkingEnabled,
		SelectedThinkingEffort:  defaultThinkingEffort,
		AuthResolved:            false,
		Authenticated:           false,
		AuthMode:                authModeLogin,
		AuthError:               "",
		AuthSubmitting:          false,
		AuthEmail:               "",
		AuthPassword:            "",
		AuthDisplayName:         "",
		SessionEmail:            "",
		ShowNameModal:           false,
		ActiveSettingsSection:   defaultSettingsSectionID,
		NameInput:               "",
		ToneInput:               defaultTone,
		ThinkingEnabledInput:    defaultThinkingEnabled,
		ThinkingEffortInput:     defaultThinkingEffort,
		SystemPromptInput:       "",
		UserMemories:            []editableUserMemory{},
		DeletedUserMemoryKeys:   []string{},
		LocaleInput:             normalizeChatLocaleID("en"),
		GRPCReady:               false,
		MarkdownWorkerFallback:  false,
		EditIdx:                 -1,
		EditText:                "",
	}
}

func reduceAppState(state appState, action appAction) appState {
	next := state
	switch action.Type {
	case appActionSetMessages:
		next.Messages = action.Messages
		next.ExpandedThoughtSections = map[string]bool{}
	case appActionUpdateMessages:
		if action.UpdateMessages != nil {
			next.Messages = action.UpdateMessages(state.Messages)
		}
	case appActionOpenCanvasSession:
		artifact := action.CanvasArtifact
		splitRatio := state.CanvasSession.SplitRatio
		if action.CanvasSplitRatio >= canvasSplitMin && action.CanvasSplitRatio <= canvasSplitMax {
			splitRatio = action.CanvasSplitRatio
		}
		next.CanvasSession = canvasSessionState{
			Active:                true,
			SessionID:             artifact.ID,
			ArtifactID:            artifact.ID,
			SourceMessageIndex:    artifact.MessageIndex,
			CurrentFileID:         artifact.Label,
			FocusedRegion:         artifact.Focus,
			FocusOptions:          append([]canvasFocusRegion(nil), artifact.FocusOptions...),
			OriginalSource:        artifact.Source,
			CurrentSource:         artifact.Source,
			FocusDraft:            canvasFocusSnippet(artifact.Source, artifact.Focus),
			LatestRenderedVersion: 1,
			LatestPatchVersion:    0,
			Dirty:                 false,
			PreviewStatus:         canvasPreviewRendering,
			RuntimeStatus:         "booting",
			ConsoleEntries: []canvasConsoleEntry{
				{Level: "info", Message: "Canvas session opened", RenderVersion: 1},
			},
			PatchHistory: nil,
			LayoutMode:   canvasLayoutSplit,
			SplitRatio:   splitRatio,
			ConsoleOpen:  false,
		}
	case appActionCloseCanvasSession:
		next.CanvasSession.Active = false
		next.CanvasSession.LayoutMode = canvasLayoutHidden
		next.CanvasSession.ConsoleOpen = false
	case appActionApplyCanvasArtifact:
		artifact := action.CanvasArtifact
		if !state.CanvasSession.Active {
			next.CanvasSession = state.CanvasSession
			next.CanvasSession.Active = true
		}
		next.CanvasSession.LayoutMode = canvasLayoutSplit
		next.CanvasSession.SessionID = artifact.ID
		next.CanvasSession.ArtifactID = artifact.ID
		next.CanvasSession.SourceMessageIndex = artifact.MessageIndex
		next.CanvasSession.CurrentFileID = artifact.Label
		next.CanvasSession.FocusOptions = append([]canvasFocusRegion(nil), artifact.FocusOptions...)
		if len(next.CanvasSession.FocusOptions) == 0 {
			next.CanvasSession.FocusOptions = []canvasFocusRegion{artifact.Focus}
		}
		focus := state.CanvasSession.FocusedRegion
		if focus.StartLine <= 0 {
			focus = artifact.Focus
		}
		patch, ok := deriveCanvasPatch(state.CanvasSession.CurrentSource, artifact.Source, focus)
		if ok {
			patch.ID = fmt.Sprintf("patch-%d", state.CanvasSession.LatestPatchVersion+1)
			next.CanvasSession.PatchHistory = append(append([]canvasPatchRecord(nil), state.CanvasSession.PatchHistory...), patch)
			next.CanvasSession.LatestPatchVersion = state.CanvasSession.LatestPatchVersion + 1
		}
		next.CanvasSession.CurrentSource = artifact.Source
		next.CanvasSession.FocusedRegion = artifact.Focus
		next.CanvasSession.FocusDraft = canvasFocusSnippet(artifact.Source, artifact.Focus)
		next.CanvasSession.Dirty = false
		next.CanvasSession.PreviewStatus = canvasPreviewRendering
		next.CanvasSession.RuntimeStatus = "booting"
		next.CanvasSession.LatestRenderedVersion = state.CanvasSession.LatestRenderedVersion + 1
	case appActionSetCanvasLayoutMode:
		next.CanvasSession.LayoutMode = action.CanvasLayoutMode
	case appActionSetCanvasSplitRatio:
		if action.CanvasSplitRatio >= canvasSplitMin && action.CanvasSplitRatio <= canvasSplitMax {
			next.CanvasSession.SplitRatio = action.CanvasSplitRatio
		}
	case appActionSetCanvasFocus:
		next.CanvasSession.FocusedRegion = action.CanvasFocus
		next.CanvasSession.FocusDraft = canvasFocusSnippet(state.CanvasSession.CurrentSource, action.CanvasFocus)
	case appActionSetCanvasFocusDraft:
		next.CanvasSession.FocusDraft = action.CanvasFocusDraft
	case appActionApplyCanvasFocusDraft:
		nextSource, patch, ok := replaceCanvasFocusRegion(state.CanvasSession.CurrentSource, state.CanvasSession.FocusedRegion, state.CanvasSession.FocusDraft)
		if ok {
			patch.ID = fmt.Sprintf("patch-%d", state.CanvasSession.LatestPatchVersion+1)
			next.CanvasSession.CurrentSource = nextSource
			next.CanvasSession.PatchHistory = append(append([]canvasPatchRecord(nil), state.CanvasSession.PatchHistory...), patch)
			next.CanvasSession.LatestPatchVersion = state.CanvasSession.LatestPatchVersion + 1
			next.CanvasSession.Dirty = state.CanvasSession.OriginalSource != nextSource
			next.CanvasSession.PreviewStatus = canvasPreviewPatched
			next.CanvasSession.RuntimeStatus = "patched"
		}
	case appActionCanvasRefreshPreview:
		next.CanvasSession.LatestRenderedVersion = state.CanvasSession.LatestRenderedVersion + 1
		next.CanvasSession.PreviewStatus = canvasPreviewRendering
		next.CanvasSession.RuntimeStatus = "booting"
	case appActionCanvasToggleConsole:
		next.CanvasSession.ConsoleOpen = !state.CanvasSession.ConsoleOpen
	case appActionCanvasClearConsole:
		next.CanvasSession.ConsoleEntries = nil
	case appActionCanvasAppendConsole:
		next.CanvasSession.ConsoleEntries = append(append([]canvasConsoleEntry(nil), state.CanvasSession.ConsoleEntries...), action.CanvasConsoleEntry)
	case appActionCanvasSetStatus:
		if action.CanvasPreviewStatus != "" {
			next.CanvasSession.PreviewStatus = action.CanvasPreviewStatus
		}
		if action.CanvasRuntimeStatus != "" {
			next.CanvasSession.RuntimeStatus = action.CanvasRuntimeStatus
		}
	case appActionCanvasResetOriginal:
		next.CanvasSession.CurrentSource = state.CanvasSession.OriginalSource
		next.CanvasSession.FocusDraft = canvasFocusSnippet(state.CanvasSession.OriginalSource, state.CanvasSession.FocusedRegion)
		next.CanvasSession.Dirty = false
		next.CanvasSession.PreviewStatus = canvasPreviewStale
		next.CanvasSession.RuntimeStatus = "reset"
	case appActionCanvasRevertLastPatch:
		if len(state.CanvasSession.PatchHistory) > 0 {
			lastPatch := state.CanvasSession.PatchHistory[len(state.CanvasSession.PatchHistory)-1]
			reverted := lastPatch.SourceBefore
			next.CanvasSession.CurrentSource = reverted
			next.CanvasSession.PatchHistory = append([]canvasPatchRecord(nil), state.CanvasSession.PatchHistory[:len(state.CanvasSession.PatchHistory)-1]...)
			next.CanvasSession.LatestPatchVersion = maxInt(0, state.CanvasSession.LatestPatchVersion-1)
			next.CanvasSession.Dirty = state.CanvasSession.OriginalSource != reverted
			next.CanvasSession.FocusDraft = canvasFocusSnippet(reverted, state.CanvasSession.FocusedRegion)
			next.CanvasSession.PreviewStatus = canvasPreviewStale
			next.CanvasSession.RuntimeStatus = "reverted"
		}
	case appActionToggleThoughtSection:
		next.ExpandedThoughtSections = make(map[string]bool, len(state.ExpandedThoughtSections))
		for key, expanded := range state.ExpandedThoughtSections {
			next.ExpandedThoughtSections[key] = expanded
		}
		next.ExpandedThoughtSections[action.ThoughtSectionKey] = !state.ExpandedThoughtSections[action.ThoughtSectionKey]
	case appActionSetInputText:
		next.InputText = action.InputText
	case appActionSetStreaming:
		next.Streaming = action.Streaming
	case appActionSetModelCatalog:
		next.ModelOptions = append([]modelOption(nil), action.ModelOptions...)
		next.DefaultModel = action.DefaultModel
		next.SelectedModel = normalizeSelectedModelID(state.SelectedModel, next.ModelOptions, next.DefaultModel)
	case appActionSetSelectedModel:
		next.SelectedModel = action.SelectedModel
	case appActionSetConversationList:
		next.ConversationList = action.ConversationList
	case appActionSetActiveConvID:
		next.ActiveConvID = action.ActiveConvID
		next.ActiveConvPublicID = action.ActiveConvPublicID
	case appActionSetDeleteTarget:
		next.DeleteTarget = action.DeleteTarget
	case appActionSetSelectedTone:
		next.SelectedTone = action.SelectedTone
	case appActionSetCustomSystemPrompt:
		next.CustomSystemPrompt = action.CustomSystemPrompt
	case appActionSetSelectedThinkingEnabled:
		next.SelectedThinkingEnabled = action.SelectedThinkingEnabled
	case appActionSetSelectedThinkingEffort:
		next.SelectedThinkingEffort = action.SelectedThinkingEffort
	case appActionSetAuthResolved:
		next.AuthResolved = action.AuthResolved
	case appActionSetAuthenticated:
		next.Authenticated = action.Authenticated
	case appActionSetAuthMode:
		next.AuthMode = action.AuthMode
	case appActionSetAuthError:
		next.AuthError = action.AuthError
	case appActionSetAuthSubmitting:
		next.AuthSubmitting = action.AuthSubmitting
	case appActionSetAuthEmail:
		next.AuthEmail = action.AuthEmail
	case appActionSetAuthPassword:
		next.AuthPassword = action.AuthPassword
	case appActionSetAuthDisplayName:
		next.AuthDisplayName = action.AuthDisplayName
	case appActionSetSessionEmail:
		next.SessionEmail = action.SessionEmail
	case appActionResetWorkspace:
		next.Messages = []message{}
		next.ExpandedThoughtSections = map[string]bool{}
		next.CanvasSession = canvasSessionState{
			LayoutMode:    canvasLayoutHidden,
			SplitRatio:    state.CanvasSession.SplitRatio,
			PreviewStatus: canvasPreviewNotRendered,
			RuntimeStatus: "idle",
		}
		next.InputText = ""
		next.Streaming = false
		next.ConversationList = []convSummary{}
		next.ActiveConvID = 0
		next.ActiveConvPublicID = ""
		next.DeleteTarget = 0
		next.CustomSystemPrompt = ""
		next.ShowNameModal = false
		next.NameInput = ""
		next.ToneInput = next.SelectedTone
		next.ThinkingEnabledInput = next.SelectedThinkingEnabled
		next.ThinkingEffortInput = next.SelectedThinkingEffort
		next.SystemPromptInput = ""
		next.UserMemories = []editableUserMemory{}
		next.DeletedUserMemoryKeys = []string{}
		next.EditIdx = -1
		next.EditText = ""
	case appActionSetShowNameModal:
		next.ShowNameModal = action.ShowNameModal
	case appActionSetActiveSettingsSection:
		next.ActiveSettingsSection = normalizeSettingsSectionID(action.ActiveSettingsSection)
		if next.ActiveSettingsSection == "" {
			next.ActiveSettingsSection = defaultSettingsSectionID
		}
	case appActionSetNameInput:
		next.NameInput = action.NameInput
	case appActionSetToneInput:
		next.ToneInput = action.ToneInput
	case appActionSetThinkingEnabledInput:
		next.ThinkingEnabledInput = action.ThinkingEnabledInput
	case appActionSetThinkingEffortInput:
		next.ThinkingEffortInput = action.ThinkingEffortInput
	case appActionSetSystemPromptInput:
		next.SystemPromptInput = action.SystemPromptInput
	case appActionSetUserMemories:
		next.UserMemories = append([]editableUserMemory(nil), action.UserMemories...)
		next.DeletedUserMemoryKeys = append([]string(nil), action.DeletedUserMemoryKeys...)
	case appActionAddUserMemory:
		next.UserMemories = append(append([]editableUserMemory(nil), state.UserMemories...), editableUserMemory{
			Category:        "preference",
			UsefulnessScore: 70,
			ConfidenceScore: 0.8,
		})
	case appActionUpdateUserMemoryField:
		next.UserMemories = append([]editableUserMemory(nil), state.UserMemories...)
		if action.UserMemoryIndex >= 0 && action.UserMemoryIndex < len(next.UserMemories) {
			current := next.UserMemories[action.UserMemoryIndex]
			switch action.UserMemoryField {
			case "category":
				current.Category = action.UserMemoryValue
			case "summary":
				current.Summary = action.UserMemoryValue
			case "detail":
				current.Detail = action.UserMemoryValue
			case "rubric_reason":
				current.RubricReason = action.UserMemoryValue
			}
			next.UserMemories[action.UserMemoryIndex] = current
		}
	case appActionDeleteUserMemory:
		if action.UserMemoryIndex >= 0 && action.UserMemoryIndex < len(state.UserMemories) {
			current := state.UserMemories[action.UserMemoryIndex]
			next.UserMemories = append([]editableUserMemory(nil), state.UserMemories[:action.UserMemoryIndex]...)
			next.UserMemories = append(next.UserMemories, state.UserMemories[action.UserMemoryIndex+1:]...)
			next.DeletedUserMemoryKeys = append([]string(nil), state.DeletedUserMemoryKeys...)
			if current.Key != "" {
				next.DeletedUserMemoryKeys = append(next.DeletedUserMemoryKeys, current.Key)
			}
		}
	case appActionSetLocaleInput:
		next.LocaleInput = action.LocaleInput
	case appActionSetGRPCReady:
		next.GRPCReady = action.GRPCReady
	case appActionSetMarkdownWorkerFallback:
		next.MarkdownWorkerFallback = action.MarkdownWorkerFallback
	case appActionSetEditIdx:
		next.EditIdx = action.EditIdx
	case appActionSetEditText:
		next.EditText = action.EditText
	}
	return next
}
