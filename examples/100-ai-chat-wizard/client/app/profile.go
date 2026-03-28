//go:build js && wasm

package app

import (
	"context"
	"strings"
	"syscall/js"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/logging"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// profileSettingsController groups the user-profile and settings-modal flow.
//
// Keeping this separate lets App() render the modal while this hook owns the
// fetch, persistence, and input handler details.
type profileSettingsController struct {
	Refresh            func(bool)
	Open               ui.Handler
	Close              ui.Handler
	NavigateSection    ui.Handler
	HandleNameInput    ui.Handler
	HandleNameKey      ui.Handler
	HandleToneChange   ui.Handler
	HandleThinkingMode ui.Handler
	HandleTTSProvider  ui.Handler
	HandleSystemPrompt ui.Handler
	HandleMemoryChange ui.Handler
	AddMemory          ui.Handler
	DeleteMemory       ui.Handler
	HandleLocaleChange ui.Handler
	Save               ui.Handler
}

// useProfileSettings hides the username and settings persistence workflow
// behind a feature-specific hook built from the public primitive hooks.
func parseUseProfileSettings(
	parseIntl i18n.Runtime,
	parseApp ui.Reducer[appState, appAction],
	parseUserNameState state.Atom[string],
	parseTtsProviderState state.Atom[string],
	parseChatClientRef ui.Ref[chatpb.ChatServiceClient],
	parseNav router.Navigator,
	parseCurrentPath string,
	parseSettingsPanelRouteID string,
	parseSettingsReturnRoute ui.Ref[string],
	parseUserNameFetchedAt ui.Ref[time.Time],
	parseSelectedToneCache fetch.CachedResource[string],
	parseSelectedThinkingEnabledCache fetch.CachedResource[bool],
	parseSelectedThinkingEffortCache fetch.CachedResource[string],
	handleAuthFailure func(error) bool,
) profileSettingsController {
	parseSeedSettingsInputs := func() {
		parseApp.Dispatch(appAction{Type: appActionSetSettingsError, SettingsError: ""})
		parseApp.Dispatch(appAction{Type: appActionSetNameInput, NameInput: parseUserNameState.Get()})
		parseApp.Dispatch(appAction{Type: appActionSetToneInput, ToneInput: parseApp.Get().SelectedTone})
		parseApp.Dispatch(appAction{Type: appActionSetThinkingEnabledInput, ThinkingEnabledInput: parseApp.Get().SelectedThinkingEnabled})
		parseApp.Dispatch(appAction{Type: appActionSetThinkingEffortInput, ThinkingEffortInput: parseApp.Get().SelectedThinkingEffort})
		parseApp.Dispatch(appAction{Type: appActionSetTTSProviderInput, TTSProviderInput: parseApp.Get().SelectedTTSProvider})
		parseApp.Dispatch(appAction{Type: appActionSetSystemPromptInput, SystemPromptInput: parseApp.Get().CustomSystemPrompt})
		parseApp.Dispatch(appAction{Type: appActionSetLocaleInput, LocaleInput: parseNormalizeChatLocaleID(parseIntl.Locale())})
		parseApp.Dispatch(appAction{Type: appActionSetUserMemories, UserMemories: parseEnsureManagedUserNameMemory(parseUserNameState.Get(), parseApp.Get().UserMemories), DeletedUserMemoryKeys: []string{}})
	}

	parseOpenSettingsSection := func(parseSection3 string) {
		parseNormalized := parseNormalizeSettingsSectionID(parseSection3)
		if parseNormalized == "" {
			parseNormalized = defaultSettingsSectionID
		}
		parseApp.Dispatch(appAction{Type: appActionSetActiveSettingsSection, ActiveSettingsSection: parseNormalized})
		if !parseApp.Get().ShowNameModal {
			parseSeedSettingsInputs()
			parseApp.Dispatch(appAction{Type: appActionSetShowNameModal, ShowNameModal: true})
		}
	}

	parseCloseSettingsRoute := func() {
		parseApp.Dispatch(appAction{Type: appActionSetSettingsError, SettingsError: ""})
		parseSection := parseNormalizeSettingsSectionID(parseApp.Get().ActiveSettingsSection)
		if parseSection == "" {
			parseSection = defaultSettingsSectionID
		}
		parseReturnPath := strings.TrimSpace(parseSettingsReturnRoute.Get())
		if parseReturnPath == "" {
			if parseActivePublicID := strings.TrimSpace(parseApp.Get().ActiveConvPublicID); parseActivePublicID != "" {
				parseReturnPath = parseChatThreadPath(parseActivePublicID)
			} else {
				parseReturnPath = chatRouteRoot
			}
		}
		parseApp.Dispatch(appAction{Type: appActionSetShowNameModal, ShowNameModal: false})
		parseNav.Navigate(buildSettingsReturnRoute(parseReturnPath, parseSection))
	}

	parseCustomSystemPromptCacheKey := ""
	if parseApp.Get().GRPCReady && parseApp.Get().Authenticated {
		parseCustomSystemPromptCacheKey = cacheKeyCustomSystemPrompt
	}
	parseCustomSystemPromptCache := fetch.UseCachedResource(parseCustomSystemPromptCacheKey, func(parseCtx context.Context) (string, error) {
		parseClient := parseChatClientRef.Get()
		if parseClient == nil {
			return "", nil
		}
		parseResp, parseErr := parseClient.GetCustomSystemPrompt(parseCtx, &emptypb.Empty{})
		if parseErr != nil {
			if handleAuthFailure != nil && handleAuthFailure(parseErr) {
				return "", nil
			}
			return "", parseErr
		}
		return strings.TrimSpace(parseResp.GetValue()), nil
	}, fetch.CacheOptions{StaleAfter: toneTTL, MaxAge: toneTTL, Persist: true})
	parseCustomSystemPromptCacheState := parseCustomSystemPromptCache.Get()

	ui.UseEffect(func() func() {
		parseCurrentState := parseApp.Get()
		if !parseCurrentState.GRPCReady || !parseCustomSystemPromptCacheState.Ready {
			return nil
		}
		parseResolvedPrompt := strings.TrimSpace(parseCustomSystemPromptCacheState.Value)
		if parseResolvedPrompt != parseCurrentState.CustomSystemPrompt {
			parseApp.Dispatch(appAction{Type: appActionSetCustomSystemPrompt, CustomSystemPrompt: parseResolvedPrompt})
		}
		if !parseCurrentState.ShowNameModal && parseResolvedPrompt != parseCurrentState.SystemPromptInput {
			parseApp.Dispatch(appAction{Type: appActionSetSystemPromptInput, SystemPromptInput: parseResolvedPrompt})
		}
		return nil
	}, parseApp.Get().GRPCReady, parseCustomSystemPromptCacheState.Ready, parseCustomSystemPromptCacheState.Value, parseApp.Get().ShowNameModal)

	ui.UseEffect(func() func() {
		if !parseApp.Get().Authenticated {
			return nil
		}
		if isSettingsRoute(parseCurrentPath) {
			parseOpenSettingsSection(parseSettingsPanelRouteID)
			return nil
		}
		if parseApp.Get().ShowNameModal {
			parseApp.Dispatch(appAction{Type: appActionSetShowNameModal, ShowNameModal: false})
		}
		return nil
	}, parseApp.Get().Authenticated, parseCurrentPath, parseSettingsPanelRouteID)

	ui.UseEffect(func() func() {
		if !parseApp.Get().ShowNameModal {
			return nil
		}
		parseSection2 := parseNormalizeSettingsSectionID(parseApp.Get().ActiveSettingsSection)
		if parseSection2 == "" {
			parseSection2 = defaultSettingsSectionID
		}
		parseWindow := js.Global().Get("window")
		if !parseWindow.Truthy() || parseWindow.Get("requestAnimationFrame").Type() != js.TypeFunction {
			parseScrollSettingsSectionIntoView(parseSection2)
			return nil
		}
		var parseCallback js.Func
		parseCallback = js.FuncOf(func(_ js.Value, _ []js.Value) interface{} {
			parseScrollSettingsSectionIntoView(parseSection2)
			parseCallback.Release()
			return nil
		})
		parseWindow.Call("requestAnimationFrame", parseCallback)
		return nil
	}, parseApp.Get().ShowNameModal, parseApp.Get().ActiveSettingsSection)

	parseRefresh := func(isForce bool) {
		if !parseApp.Get().Authenticated {
			return
		}
		if !isForce && time.Since(parseUserNameFetchedAt.Get()) < userNameTTL {
			return
		}
		parseClient2 := parseChatClientRef.Get()
		if parseClient2 == nil {
			return
		}
		go func() {
			parseResp2, parseErr2 := parseClient2.GetUserName(context.Background(), &chatpb.GetUserNameRequest{})
			if parseErr2 != nil {
				if handleAuthFailure != nil && handleAuthFailure(parseErr2) {
					return
				}
				chatLog.Error("get user name failed", logging.Fields{"error": parseErr2})
				return
			}
			if parseResp2.Name != "" {
				parseUserNameState.Set(parseResp2.Name)
			}
			parseMemoriesResp, parseMemoriesErr := parseClient2.ListUserMemories(context.Background(), &chatpb.ListUserMemoriesRequest{})
			if parseMemoriesErr != nil {
				if handleAuthFailure != nil && handleAuthFailure(parseMemoriesErr) {
					return
				}
				chatLog.Error("list user memories failed", logging.Fields{"error": parseMemoriesErr})
				return
			}
			parseMemories := make([]editableUserMemory, 0, len(parseMemoriesResp.Memories))
			for _, parseMemory := range parseMemoriesResp.Memories {
				parseMemories = append(parseMemories, editableUserMemory{
					Key:             parseMemory.GetKey(),
					Category:        parseMemory.GetCategory(),
					Summary:         parseMemory.GetSummary(),
					Detail:          parseMemory.GetDetail(),
					SourceMessage:   parseMemory.GetSourceMessage(),
					UsefulnessScore: int(parseMemory.GetUsefulnessScore()),
					ConfidenceScore: parseMemory.GetConfidenceScore(),
					RubricReason:    parseMemory.GetRubricReason(),
					UpdatedAt:       parseMemory.GetUpdatedAt(),
				})
			}
			parseMemories = parseEnsureManagedUserNameMemory(parseResp2.Name, parseMemories)
			parseApp.Dispatch(appAction{Type: appActionSetUserMemories, UserMemories: parseMemories, DeletedUserMemoryKeys: []string{}})
			parseUserNameFetchedAt.Set(time.Now())
			if isForce {
				chatLog.Info("profile", logging.Fields{"name": parseResp2.Name, "memories": len(parseMemories)})
			}
		}()
	}

	parseSaveSettings := func() {
		parseCurrentState2 := parseApp.Get()
		parseApp.Dispatch(appAction{Type: appActionSetSettingsError, SettingsError: ""})
		parseName := strings.TrimSpace(parseCurrentState2.NameInput)
		parseSelectedToneValue := parseNormalizeSelectedToneID(parseCurrentState2.ToneInput)
		parseSelectedThinkingEnabledValue := parseCurrentState2.ThinkingEnabledInput
		parseSelectedThinkingEffortValue := parseNormalizeSelectedThinkingEffort(parseCurrentState2.ThinkingEffortInput)
		parseSelectedTTSProviderValue := parseResolveTTSProviderID(parseCurrentState2.TTSProviderInput)
		parseSystemPromptValue := strings.TrimSpace(parseCurrentState2.SystemPromptInput)
		parseSelectedLocaleValue := parseNormalizeChatLocaleID(parseCurrentState2.LocaleInput)
		if parseName != "" {
			chatLog.Info("profile save", logging.Fields{"name": parseName, "tone": parseSelectedToneValue, "system_prompt_len": len([]rune(parseSystemPromptValue))})
			parseUserNameState.Set(parseName)
			parseClient3 := parseChatClientRef.Get()
			if parseClient3 != nil {
				go func() {
					_, parseErr3 := parseClient3.SetUserName(context.Background(), &chatpb.SetUserNameRequest{Name: parseName})
					if parseErr3 != nil {
						if handleAuthFailure != nil && handleAuthFailure(parseErr3) {
							return
						}
						chatLog.Error("set user name failed", logging.Fields{"error": parseErr3})
						return
					}
					parseUserNameFetchedAt.Set(time.Now())
				}()
			}
		}
		parseIntl.SetLocale(parseSelectedLocaleValue)
		parseApp.Dispatch(appAction{Type: appActionSetSelectedTone, SelectedTone: parseSelectedToneValue})
		parseApp.Dispatch(appAction{Type: appActionSetSelectedThinkingEnabled, SelectedThinkingEnabled: parseSelectedThinkingEnabledValue})
		if parseSelectedThinkingEnabledValue {
			parseApp.Dispatch(appAction{Type: appActionSetSelectedThinkingEffort, SelectedThinkingEffort: parseSelectedThinkingEffortValue})
		}
		parseApp.Dispatch(appAction{Type: appActionSetSelectedTTSProvider, SelectedTTSProvider: parseSelectedTTSProviderValue})
		parseTtsProviderState.Set(parseSelectedTTSProviderValue)
		parseApp.Dispatch(appAction{Type: appActionSetCustomSystemPrompt, CustomSystemPrompt: parseSystemPromptValue})
		parseSelectedToneCache.Set(parseSelectedToneValue)
		parseSelectedThinkingEnabledCache.Set(parseSelectedThinkingEnabledValue)
		if parseSelectedThinkingEnabledValue {
			parseSelectedThinkingEffortCache.Set(parseSelectedThinkingEffortValue)
		}
		parseCustomSystemPromptCache.Set(parseSystemPromptValue)
		parseClient4 := parseChatClientRef.Get()
		if parseClient4 == nil {
			parseApp.Dispatch(appAction{Type: appActionSetSettingsError, SettingsError: parseBuildUserErrorText(userErrorScopeSettings, nil)})
			return
		}
		if _, parseErr4 := parseClient4.SetSelectedTone(context.Background(), wrapperspb.String(parseSelectedToneValue)); parseErr4 != nil {
			chatLog.Error("set selected tone failed", logging.Fields{"error": parseErr4})
			parseApp.Dispatch(appAction{Type: appActionSetSettingsError, SettingsError: parseBuildUserErrorText(userErrorScopeSettings, parseErr4)})
			parseSelectedToneCache.Invalidate()
			return
		}
		go func() {
			_, parseErr5 := parseClient4.SetSelectedThinkingEnabled(context.Background(), wrapperspb.Bool(parseSelectedThinkingEnabledValue))
			if parseErr5 != nil {
				if handleAuthFailure != nil && handleAuthFailure(parseErr5) {
					return
				}
				chatLog.Error("set selected thinking enabled failed", logging.Fields{"error": parseErr5})
				parseApp.Dispatch(appAction{Type: appActionSetSettingsError, SettingsError: parseBuildUserErrorText(userErrorScopeSettings, parseErr5)})
				parseSelectedThinkingEnabledCache.Invalidate()
			}
		}()
		if parseSelectedThinkingEnabledValue {
			go func() {
				_, parseErr6 := parseClient4.SetSelectedThinkingEffort(context.Background(), wrapperspb.String(parseSelectedThinkingEffortValue))
				if parseErr6 != nil {
					if handleAuthFailure != nil && handleAuthFailure(parseErr6) {
						return
					}
					chatLog.Error("set selected thinking effort failed", logging.Fields{"error": parseErr6})
					parseApp.Dispatch(appAction{Type: appActionSetSettingsError, SettingsError: parseBuildUserErrorText(userErrorScopeSettings, parseErr6)})
					parseSelectedThinkingEffortCache.Invalidate()
				}
			}()
		}
		go func() {
			_, parseErr7 := parseClient4.SetCustomSystemPrompt(context.Background(), wrapperspb.String(parseSystemPromptValue))
			if parseErr7 != nil {
				if handleAuthFailure != nil && handleAuthFailure(parseErr7) {
					return
				}
				chatLog.Error("set custom system prompt failed", logging.Fields{"error": parseErr7})
				parseApp.Dispatch(appAction{Type: appActionSetSettingsError, SettingsError: parseBuildUserErrorText(userErrorScopeSettings, parseErr7)})
				parseCustomSystemPromptCache.Invalidate()
			}
		}()
		for _, parseKey := range parseCurrentState2.DeletedUserMemoryKeys {
			parseDeleteKey := strings.TrimSpace(parseKey)
			if parseDeleteKey == "" {
				continue
			}
			go func(parseMemoryKey string) {
				if _, parseErr8 := parseClient4.DeleteUserMemory(context.Background(), &chatpb.DeleteUserMemoryRequest{Key: parseMemoryKey}); parseErr8 != nil {
					if handleAuthFailure != nil && handleAuthFailure(parseErr8) {
						return
					}
					chatLog.Error("delete user memory failed", logging.Fields{"error": parseErr8, "key": parseMemoryKey})
					parseApp.Dispatch(appAction{Type: appActionSetSettingsError, SettingsError: parseBuildUserErrorText(userErrorScopeSettings, parseErr8)})
				}
			}(parseDeleteKey)
		}
		for _, parseMemory2 := range parseCurrentState2.UserMemories {
			if isManagedUserNameMemory(parseMemory2) {
				continue
			}
			parseSummary := strings.TrimSpace(parseMemory2.Summary)
			if parseSummary == "" {
				continue
			}
			parseRequest := &chatpb.UpsertUserMemoryRequest{
				Memory: &chatpb.UserMemory{
					Key:             strings.TrimSpace(parseMemory2.Key),
					Category:        strings.TrimSpace(parseMemory2.Category),
					Summary:         parseSummary,
					Detail:          strings.TrimSpace(parseMemory2.Detail),
					SourceMessage:   strings.TrimSpace(parseMemory2.SourceMessage),
					UsefulnessScore: int32(parseMemory2.UsefulnessScore),
					ConfidenceScore: parseMemory2.ConfidenceScore,
					RubricReason:    strings.TrimSpace(parseMemory2.RubricReason),
					UpdatedAt:       parseMemory2.UpdatedAt,
				},
			}
			go func(parseReq *chatpb.UpsertUserMemoryRequest) {
				if _, parseErr9 := parseClient4.UpsertUserMemory(context.Background(), parseReq); parseErr9 != nil {
					if handleAuthFailure != nil && handleAuthFailure(parseErr9) {
						return
					}
					chatLog.Error("upsert user memory failed", logging.Fields{"error": parseErr9, "summary": parseReq.GetMemory().GetSummary()})
					parseApp.Dispatch(appAction{Type: appActionSetSettingsError, SettingsError: parseBuildUserErrorText(userErrorScopeSettings, parseErr9)})
				}
			}(parseRequest)
		}
		parseApp.Dispatch(appAction{Type: appActionSetToneInput, ToneInput: parseSelectedToneValue})
		parseApp.Dispatch(appAction{Type: appActionSetTTSProviderInput, TTSProviderInput: parseSelectedTTSProviderValue})
		parseApp.Dispatch(appAction{Type: appActionSetSystemPromptInput, SystemPromptInput: parseSystemPromptValue})
		parseApp.Dispatch(appAction{Type: appActionSetUserMemories, UserMemories: parseEnsureManagedUserNameMemory(parseName, parseCurrentState2.UserMemories), DeletedUserMemoryKeys: []string{}})
		parseCloseSettingsRoute()
	}

	parseOpen := ui.UseEvent(func() {
		if !isSettingsRoute(parseCurrentPath) {
			parseSettingsReturnRoute.Set(parseCurrentLocationPathSearch())
			parseNav.Navigate(buildSettingsRoute(defaultSettingsSectionID))
			return
		}
		parseNav.Replace(buildSettingsRoute(defaultSettingsSectionID))
	})

	parseClose := ui.UseEvent(func() {
		parseCloseSettingsRoute()
	})

	parseNavigateSection := ui.UseEvent(func(parseE ui.Event) {
		parseE.PreventDefault()
		parseNormalized2 := parseNormalizeSettingsSectionID(parseEventDatasetValue(parseE, dataSettingsSection))
		if parseNormalized2 == "" {
			return
		}
		parseApp.Dispatch(appAction{Type: appActionSetActiveSettingsSection, ActiveSettingsSection: parseNormalized2})
		parseNav.Replace(buildSettingsRoute(parseNormalized2))
	})

	handleNameInput := ui.UseEvent(func(parseE2 ui.Event) {
		parseNextName := parseE2.GetValue()
		parseApp.Dispatch(appAction{Type: appActionSetNameInput, NameInput: parseNextName})
		parseApp.Dispatch(appAction{Type: appActionSetUserMemories, UserMemories: parseEnsureManagedUserNameMemory(parseNextName, parseApp.Get().UserMemories), DeletedUserMemoryKeys: parseApp.Get().DeletedUserMemoryKeys})
	})

	handleNameKey := ui.UseEvent(func(parseE3 ui.Event) {
		switch parseE3.GetKey() {
		case "Enter":
			parseE3.PreventDefault()
			parseSaveSettings()
		case "Escape":
			parseCloseSettingsRoute()
		}
	})

	handleToneChange := ui.UseEvent(func(parseE4 ui.Event) {
		parseApp.Dispatch(appAction{Type: appActionSetToneInput, ToneInput: parseNormalizeSelectedToneID(parseEventDatasetValue(parseE4, dataTone))})
	})

	handleThinkingMode := ui.UseEvent(func(parseE5 ui.Event) {
		parseNextMode := strings.TrimSpace(strings.ToLower(parseEventValueOrDataset(parseE5, dataThinkingEffort)))
		isParseNextEnabled := parseNextMode != "off"
		parseApp.Dispatch(appAction{Type: appActionSetThinkingEnabledInput, ThinkingEnabledInput: isParseNextEnabled})
		if isParseNextEnabled {
			parseApp.Dispatch(appAction{Type: appActionSetThinkingEffortInput, ThinkingEffortInput: parseNormalizeSelectedThinkingEffort(parseNextMode)})
		}
	})

	handleTTSProvider := ui.UseEvent(func(parseE6 ui.Event) {
		parseNextProvider := parseResolveTTSProviderID(parseEventValueOrDataset(parseE6, dataTTSProvider))
		parseApp.Dispatch(appAction{Type: appActionSetTTSProviderInput, TTSProviderInput: parseNextProvider})
	})

	handleSystemPrompt := ui.UseEvent(func(parseE7 ui.Event) {
		parseApp.Dispatch(appAction{Type: appActionSetSystemPromptInput, SystemPromptInput: parseE7.GetValue()})
	})

	handleMemoryChange := ui.UseEvent(func(parseE8 ui.Event) {
		parseIndex, parseOk := parseEventDatasetInt(parseE8, dataMemoryIndex)
		if !parseOk {
			return
		}
		parseCurrentState3 := parseApp.Get()
		if parseIndex < 0 || parseIndex >= len(parseCurrentState3.UserMemories) {
			return
		}
		if isManagedUserNameMemory(parseCurrentState3.UserMemories[parseIndex]) {
			return
		}
		parseField := parseEventDatasetValue(parseE8, dataMemoryField)
		parseApp.Dispatch(appAction{Type: appActionUpdateUserMemoryField, UserMemoryIndex: parseIndex, UserMemoryField: parseField, UserMemoryValue: parseE8.GetValue()})
	})

	parseAddMemory := ui.UseEvent(func() {
		parseApp.Dispatch(appAction{Type: appActionAddUserMemory})
	})

	parseDeleteMemory := ui.UseEvent(func(parseE9 ui.Event) {
		parseIndex2, parseOk2 := parseEventDatasetInt(parseE9, dataMemoryIndex)
		if !parseOk2 {
			return
		}
		parseCurrentState4 := parseApp.Get()
		if parseIndex2 < 0 || parseIndex2 >= len(parseCurrentState4.UserMemories) {
			return
		}
		if isManagedUserNameMemory(parseCurrentState4.UserMemories[parseIndex2]) {
			return
		}
		parseApp.Dispatch(appAction{Type: appActionDeleteUserMemory, UserMemoryIndex: parseIndex2})
	})

	handleLocaleChange := ui.UseEvent(func(parseE10 ui.Event) {
		parseApp.Dispatch(appAction{Type: appActionSetLocaleInput, LocaleInput: parseNormalizeChatLocaleID(parseEventDatasetValue(parseE10, dataLocale))})
	})

	parseSave := ui.UseEvent(func() {
		parseSaveSettings()
	})

	return profileSettingsController{
		Refresh:            parseRefresh,
		Open:               parseOpen,
		Close:              parseClose,
		NavigateSection:    parseNavigateSection,
		HandleNameInput:    handleNameInput,
		HandleNameKey:      handleNameKey,
		HandleToneChange:   handleToneChange,
		HandleThinkingMode: handleThinkingMode,
		HandleTTSProvider:  handleTTSProvider,
		HandleSystemPrompt: handleSystemPrompt,
		HandleMemoryChange: handleMemoryChange,
		AddMemory:          parseAddMemory,
		DeleteMemory:       parseDeleteMemory,
		HandleLocaleChange: handleLocaleChange,
		Save:               parseSave,
	}
}
