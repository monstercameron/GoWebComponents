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
	SelectedTTSProvider     string
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
	TTSProviderInput        string
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
	appActionSetSelectedTTSProvider     appActionType = "set_selected_tts_provider"
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
	appActionSetTTSProviderInput        appActionType = "set_tts_provider_input"
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
	SelectedTTSProvider     string
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
	TTSProviderInput        string
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

func parseInitialAppState() appState {
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
		SelectedTTSProvider:     defaultTTSProvider,
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
		TTSProviderInput:        defaultTTSProvider,
		SystemPromptInput:       "",
		UserMemories:            []editableUserMemory{},
		DeletedUserMemoryKeys:   []string{},
		LocaleInput:             parseNormalizeChatLocaleID("en"),
		GRPCReady:               false,
		MarkdownWorkerFallback:  false,
		EditIdx:                 -1,
		EditText:                "",
	}
}

func parseReduceAppState(parseState appState, parseAction appAction) appState {
	parseNext := parseState
	switch parseAction.Type {
	case appActionSetMessages:
		parseNext.Messages = parseAction.Messages
		parseNext.ExpandedThoughtSections = map[string]bool{}
	case appActionUpdateMessages:
		if parseAction.UpdateMessages != nil {
			parseNext.Messages = parseAction.UpdateMessages(parseState.Messages)
		}
	case appActionOpenCanvasSession:
		parseArtifact := parseAction.CanvasArtifact
		parseSplitRatio := parseState.CanvasSession.SplitRatio
		if parseAction.CanvasSplitRatio >= canvasSplitMin && parseAction.CanvasSplitRatio <= canvasSplitMax {
			parseSplitRatio = parseAction.CanvasSplitRatio
		}
		parseNext.CanvasSession = canvasSessionState{
			Active:                true,
			SessionID:             parseArtifact.ParseID,
			ArtifactID:            parseArtifact.ParseID,
			SourceMessageIndex:    parseArtifact.MessageIndex,
			CurrentFileID:         parseArtifact.Label,
			FocusedRegion:         parseArtifact.Focus,
			FocusOptions:          append([]canvasFocusRegion(nil), parseArtifact.FocusOptions...),
			OriginalSource:        parseArtifact.Source,
			CurrentSource:         parseArtifact.Source,
			FocusDraft:            canvasFocusSnippet(parseArtifact.Source, parseArtifact.Focus),
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
			SplitRatio:   parseSplitRatio,
			ConsoleOpen:  false,
		}
	case appActionCloseCanvasSession:
		parseNext.CanvasSession.Active = false
		parseNext.CanvasSession.LayoutMode = canvasLayoutHidden
		parseNext.CanvasSession.ConsoleOpen = false
	case appActionApplyCanvasArtifact:
		parseArtifact2 := parseAction.CanvasArtifact
		if !parseState.CanvasSession.Active {
			parseNext.CanvasSession = parseState.CanvasSession
			parseNext.CanvasSession.Active = true
		}
		parseNext.CanvasSession.LayoutMode = canvasLayoutSplit
		parseNext.CanvasSession.SessionID = parseArtifact2.ParseID
		parseNext.CanvasSession.ArtifactID = parseArtifact2.ParseID
		parseNext.CanvasSession.SourceMessageIndex = parseArtifact2.MessageIndex
		parseNext.CanvasSession.CurrentFileID = parseArtifact2.Label
		parseNext.CanvasSession.FocusOptions = append([]canvasFocusRegion(nil), parseArtifact2.FocusOptions...)
		if len(parseNext.CanvasSession.FocusOptions) == 0 {
			parseNext.CanvasSession.FocusOptions = []canvasFocusRegion{parseArtifact2.Focus}
		}
		parseFocus := parseState.CanvasSession.FocusedRegion
		if parseFocus.StartLine <= 0 {
			parseFocus = parseArtifact2.Focus
		}
		parsePatch, parseOk := parseDeriveCanvasPatch(parseState.CanvasSession.CurrentSource, parseArtifact2.Source, parseFocus)
		if parseOk {
			parsePatch.ParseID = fmt.Sprintf("patch-%d", parseState.CanvasSession.LatestPatchVersion+1)
			parseNext.CanvasSession.PatchHistory = append(append([]canvasPatchRecord(nil), parseState.CanvasSession.PatchHistory...), parsePatch)
			parseNext.CanvasSession.LatestPatchVersion = parseState.CanvasSession.LatestPatchVersion + 1
		}
		parseNext.CanvasSession.CurrentSource = parseArtifact2.Source
		parseNext.CanvasSession.FocusedRegion = parseArtifact2.Focus
		parseNext.CanvasSession.FocusDraft = canvasFocusSnippet(parseArtifact2.Source, parseArtifact2.Focus)
		parseNext.CanvasSession.Dirty = false
		parseNext.CanvasSession.PreviewStatus = canvasPreviewRendering
		parseNext.CanvasSession.RuntimeStatus = "booting"
		parseNext.CanvasSession.LatestRenderedVersion = parseState.CanvasSession.LatestRenderedVersion + 1
	case appActionSetCanvasLayoutMode:
		parseNext.CanvasSession.LayoutMode = parseAction.CanvasLayoutMode
	case appActionSetCanvasSplitRatio:
		if parseAction.CanvasSplitRatio >= canvasSplitMin && parseAction.CanvasSplitRatio <= canvasSplitMax {
			parseNext.CanvasSession.SplitRatio = parseAction.CanvasSplitRatio
		}
	case appActionSetCanvasFocus:
		parseNext.CanvasSession.FocusedRegion = parseAction.CanvasFocus
		parseNext.CanvasSession.FocusDraft = canvasFocusSnippet(parseState.CanvasSession.CurrentSource, parseAction.CanvasFocus)
	case appActionSetCanvasFocusDraft:
		parseNext.CanvasSession.FocusDraft = parseAction.CanvasFocusDraft
	case appActionApplyCanvasFocusDraft:
		parseNextSource, parsePatch2, parseOk2 := parseReplaceCanvasFocusRegion(parseState.CanvasSession.CurrentSource, parseState.CanvasSession.FocusedRegion, parseState.CanvasSession.FocusDraft)
		if parseOk2 {
			parsePatch2.ParseID = fmt.Sprintf("patch-%d", parseState.CanvasSession.LatestPatchVersion+1)
			parseNext.CanvasSession.CurrentSource = parseNextSource
			parseNext.CanvasSession.PatchHistory = append(append([]canvasPatchRecord(nil), parseState.CanvasSession.PatchHistory...), parsePatch2)
			parseNext.CanvasSession.LatestPatchVersion = parseState.CanvasSession.LatestPatchVersion + 1
			parseNext.CanvasSession.Dirty = parseState.CanvasSession.OriginalSource != parseNextSource
			parseNext.CanvasSession.PreviewStatus = canvasPreviewPatched
			parseNext.CanvasSession.RuntimeStatus = "patched"
		}
	case appActionCanvasRefreshPreview:
		parseNext.CanvasSession.LatestRenderedVersion = parseState.CanvasSession.LatestRenderedVersion + 1
		parseNext.CanvasSession.PreviewStatus = canvasPreviewRendering
		parseNext.CanvasSession.RuntimeStatus = "booting"
	case appActionCanvasToggleConsole:
		parseNext.CanvasSession.ConsoleOpen = !parseState.CanvasSession.ConsoleOpen
	case appActionCanvasClearConsole:
		parseNext.CanvasSession.ConsoleEntries = nil
	case appActionCanvasAppendConsole:
		parseNext.CanvasSession.ConsoleEntries = append(append([]canvasConsoleEntry(nil), parseState.CanvasSession.ConsoleEntries...), parseAction.CanvasConsoleEntry)
	case appActionCanvasSetStatus:
		if parseAction.CanvasPreviewStatus != "" {
			parseNext.CanvasSession.PreviewStatus = parseAction.CanvasPreviewStatus
		}
		if parseAction.CanvasRuntimeStatus != "" {
			parseNext.CanvasSession.RuntimeStatus = parseAction.CanvasRuntimeStatus
		}
	case appActionCanvasResetOriginal:
		parseNext.CanvasSession.CurrentSource = parseState.CanvasSession.OriginalSource
		parseNext.CanvasSession.FocusDraft = canvasFocusSnippet(parseState.CanvasSession.OriginalSource, parseState.CanvasSession.FocusedRegion)
		parseNext.CanvasSession.Dirty = false
		parseNext.CanvasSession.PreviewStatus = canvasPreviewStale
		parseNext.CanvasSession.RuntimeStatus = "reset"
	case appActionCanvasRevertLastPatch:
		if len(parseState.CanvasSession.PatchHistory) > 0 {
			parseLastPatch := parseState.CanvasSession.PatchHistory[len(parseState.CanvasSession.PatchHistory)-1]
			parseReverted := parseLastPatch.SourceBefore
			parseNext.CanvasSession.CurrentSource = parseReverted
			parseNext.CanvasSession.PatchHistory = append([]canvasPatchRecord(nil), parseState.CanvasSession.PatchHistory[:len(parseState.CanvasSession.PatchHistory)-1]...)
			parseNext.CanvasSession.LatestPatchVersion = parseMaxInt(0, parseState.CanvasSession.LatestPatchVersion-1)
			parseNext.CanvasSession.Dirty = parseState.CanvasSession.OriginalSource != parseReverted
			parseNext.CanvasSession.FocusDraft = canvasFocusSnippet(parseReverted, parseState.CanvasSession.FocusedRegion)
			parseNext.CanvasSession.PreviewStatus = canvasPreviewStale
			parseNext.CanvasSession.RuntimeStatus = "reverted"
		}
	case appActionToggleThoughtSection:
		parseNext.ExpandedThoughtSections = make(map[string]bool, len(parseState.ExpandedThoughtSections))
		for parseKey, parseExpanded := range parseState.ExpandedThoughtSections {
			parseNext.ExpandedThoughtSections[parseKey] = parseExpanded
		}
		parseNext.ExpandedThoughtSections[parseAction.ThoughtSectionKey] = !parseState.ExpandedThoughtSections[parseAction.ThoughtSectionKey]
	case appActionSetInputText:
		parseNext.InputText = parseAction.InputText
	case appActionSetStreaming:
		parseNext.Streaming = parseAction.Streaming
	case appActionSetModelCatalog:
		parseNext.ParseModelOptions = append([]modelOption(nil), parseAction.ParseModelOptions...)
		parseNext.ParseDefaultModel = parseAction.ParseDefaultModel
		parseNext.SelectedModel = parseNormalizeSelectedModelID(parseState.SelectedModel, parseNext.ParseModelOptions, parseNext.ParseDefaultModel)
	case appActionSetSelectedModel:
		parseNext.SelectedModel = parseAction.SelectedModel
	case appActionSetConversationList:
		parseNext.ConversationList = parseAction.ConversationList
	case appActionSetActiveConvID:
		parseNext.ActiveConvID = parseAction.ActiveConvID
		parseNext.ActiveConvPublicID = parseAction.ActiveConvPublicID
	case appActionSetDeleteTarget:
		parseNext.DeleteTarget = parseAction.DeleteTarget
	case appActionSetSelectedTone:
		parseNext.SelectedTone = parseAction.SelectedTone
	case appActionSetCustomSystemPrompt:
		parseNext.CustomSystemPrompt = parseAction.CustomSystemPrompt
	case appActionSetSelectedThinkingEnabled:
		parseNext.SelectedThinkingEnabled = parseAction.SelectedThinkingEnabled
	case appActionSetSelectedThinkingEffort:
		parseNext.SelectedThinkingEffort = parseAction.SelectedThinkingEffort
	case appActionSetSelectedTTSProvider:
		parseNext.SelectedTTSProvider = parseResolveTTSProviderID(parseAction.SelectedTTSProvider)
	case appActionSetAuthResolved:
		parseNext.AuthResolved = parseAction.AuthResolved
	case appActionSetAuthenticated:
		parseNext.Authenticated = parseAction.Authenticated
	case appActionSetAuthMode:
		parseNext.AuthMode = parseAction.AuthMode
	case appActionSetAuthError:
		parseNext.AuthError = parseAction.AuthError
	case appActionSetAuthSubmitting:
		parseNext.AuthSubmitting = parseAction.AuthSubmitting
	case appActionSetAuthEmail:
		parseNext.AuthEmail = parseAction.AuthEmail
	case appActionSetAuthPassword:
		parseNext.AuthPassword = parseAction.AuthPassword
	case appActionSetAuthDisplayName:
		parseNext.AuthDisplayName = parseAction.AuthDisplayName
	case appActionSetSessionEmail:
		parseNext.SessionEmail = parseAction.SessionEmail
	case appActionResetWorkspace:
		parseNext.Messages = []message{}
		parseNext.ExpandedThoughtSections = map[string]bool{}
		parseNext.CanvasSession = canvasSessionState{
			LayoutMode:    canvasLayoutHidden,
			SplitRatio:    parseState.CanvasSession.SplitRatio,
			PreviewStatus: canvasPreviewNotRendered,
			RuntimeStatus: "idle",
		}
		parseNext.InputText = ""
		parseNext.Streaming = false
		parseNext.ConversationList = []convSummary{}
		parseNext.ActiveConvID = 0
		parseNext.ActiveConvPublicID = ""
		parseNext.DeleteTarget = 0
		parseNext.CustomSystemPrompt = ""
		parseNext.ShowNameModal = false
		parseNext.NameInput = ""
		parseNext.ToneInput = parseNext.SelectedTone
		parseNext.ThinkingEnabledInput = parseNext.SelectedThinkingEnabled
		parseNext.ThinkingEffortInput = parseNext.SelectedThinkingEffort
		parseNext.TTSProviderInput = parseNext.SelectedTTSProvider
		parseNext.SystemPromptInput = ""
		parseNext.UserMemories = []editableUserMemory{}
		parseNext.DeletedUserMemoryKeys = []string{}
		parseNext.EditIdx = -1
		parseNext.EditText = ""
	case appActionSetShowNameModal:
		parseNext.ShowNameModal = parseAction.ShowNameModal
	case appActionSetActiveSettingsSection:
		parseNext.ActiveSettingsSection = parseNormalizeSettingsSectionID(parseAction.ActiveSettingsSection)
		if parseNext.ActiveSettingsSection == "" {
			parseNext.ActiveSettingsSection = defaultSettingsSectionID
		}
	case appActionSetNameInput:
		parseNext.NameInput = parseAction.NameInput
	case appActionSetToneInput:
		parseNext.ToneInput = parseAction.ToneInput
	case appActionSetThinkingEnabledInput:
		parseNext.ThinkingEnabledInput = parseAction.ThinkingEnabledInput
	case appActionSetThinkingEffortInput:
		parseNext.ThinkingEffortInput = parseAction.ThinkingEffortInput
	case appActionSetTTSProviderInput:
		parseNext.TTSProviderInput = parseResolveTTSProviderID(parseAction.TTSProviderInput)
	case appActionSetSystemPromptInput:
		parseNext.SystemPromptInput = parseAction.SystemPromptInput
	case appActionSetUserMemories:
		parseNext.UserMemories = append([]editableUserMemory(nil), parseAction.UserMemories...)
		parseNext.DeletedUserMemoryKeys = append([]string(nil), parseAction.DeletedUserMemoryKeys...)
	case appActionAddUserMemory:
		parseNext.UserMemories = append(append([]editableUserMemory(nil), parseState.UserMemories...), editableUserMemory{
			Category:        "preference",
			UsefulnessScore: 70,
			ConfidenceScore: 0.8,
		})
	case appActionUpdateUserMemoryField:
		parseNext.UserMemories = append([]editableUserMemory(nil), parseState.UserMemories...)
		if parseAction.UserMemoryIndex >= 0 && parseAction.UserMemoryIndex < len(parseNext.UserMemories) {
			parseCurrent := parseNext.UserMemories[parseAction.UserMemoryIndex]
			switch parseAction.UserMemoryField {
			case "category":
				parseCurrent.Category = parseAction.UserMemoryValue
			case "summary":
				parseCurrent.Summary = parseAction.UserMemoryValue
			case "detail":
				parseCurrent.Detail = parseAction.UserMemoryValue
			case "rubric_reason":
				parseCurrent.RubricReason = parseAction.UserMemoryValue
			}
			parseNext.UserMemories[parseAction.UserMemoryIndex] = parseCurrent
		}
	case appActionDeleteUserMemory:
		if parseAction.UserMemoryIndex >= 0 && parseAction.UserMemoryIndex < len(parseState.UserMemories) {
			parseCurrent2 := parseState.UserMemories[parseAction.UserMemoryIndex]
			parseNext.UserMemories = append([]editableUserMemory(nil), parseState.UserMemories[:parseAction.UserMemoryIndex]...)
			parseNext.UserMemories = append(parseNext.UserMemories, parseState.UserMemories[parseAction.UserMemoryIndex+1:]...)
			parseNext.DeletedUserMemoryKeys = append([]string(nil), parseState.DeletedUserMemoryKeys...)
			if parseCurrent2.Key != "" {
				parseNext.DeletedUserMemoryKeys = append(parseNext.DeletedUserMemoryKeys, parseCurrent2.Key)
			}
		}
	case appActionSetLocaleInput:
		parseNext.LocaleInput = parseAction.LocaleInput
	case appActionSetGRPCReady:
		parseNext.GRPCReady = parseAction.GRPCReady
	case appActionSetMarkdownWorkerFallback:
		parseNext.MarkdownWorkerFallback = parseAction.MarkdownWorkerFallback
	case appActionSetEditIdx:
		parseNext.EditIdx = parseAction.EditIdx
	case appActionSetEditText:
		parseNext.EditText = parseAction.EditText
	}
	return parseNext
}
