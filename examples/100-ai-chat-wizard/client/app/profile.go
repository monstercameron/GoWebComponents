//go:build js && wasm

package app

import (
	"context"
	"strings"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/logging"
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
	HandleNameInput    ui.Handler
	HandleNameKey      ui.Handler
	HandleToneChange   ui.Handler
	HandleThinkingMode ui.Handler
	HandleSystemPrompt ui.Handler
	HandleMemoryChange ui.Handler
	AddMemory          ui.Handler
	DeleteMemory       ui.Handler
	HandleLocaleChange ui.Handler
	Save               ui.Handler
}

// useProfileSettings hides the username and settings persistence workflow
// behind a feature-specific hook built from the public primitive hooks.
func useProfileSettings(
	intl i18n.Runtime,
	app ui.Reducer[appState, appAction],
	userNameState state.Atom[string],
	chatClientRef ui.Ref[chatpb.ChatServiceClient],
	userNameFetchedAt ui.Ref[time.Time],
	selectedToneCache fetch.CachedResource[string],
	selectedThinkingEnabledCache fetch.CachedResource[bool],
	selectedThinkingEffortCache fetch.CachedResource[string],
	handleAuthFailure func(error) bool,
) profileSettingsController {
	customSystemPromptCacheKey := ""
	if app.Get().GRPCReady && app.Get().Authenticated {
		customSystemPromptCacheKey = cacheKeyCustomSystemPrompt
	}
	customSystemPromptCache := fetch.UseCachedResource(customSystemPromptCacheKey, func(ctx context.Context) (string, error) {
		client := chatClientRef.Get()
		if client == nil {
			return "", nil
		}
		resp, err := client.GetCustomSystemPrompt(ctx, &emptypb.Empty{})
		if err != nil {
			if handleAuthFailure != nil && handleAuthFailure(err) {
				return "", nil
			}
			return "", err
		}
		return strings.TrimSpace(resp.GetValue()), nil
	}, fetch.CacheOptions{StaleAfter: toneTTL, MaxAge: toneTTL, Persist: true})
	customSystemPromptCacheState := customSystemPromptCache.Get()

	ui.UseEffect(func() func() {
		currentState := app.Get()
		if !currentState.GRPCReady || !customSystemPromptCacheState.Ready {
			return nil
		}
		resolvedPrompt := strings.TrimSpace(customSystemPromptCacheState.Value)
		if resolvedPrompt != currentState.CustomSystemPrompt {
			app.Dispatch(appAction{Type: appActionSetCustomSystemPrompt, CustomSystemPrompt: resolvedPrompt})
		}
		if !currentState.ShowNameModal && resolvedPrompt != currentState.SystemPromptInput {
			app.Dispatch(appAction{Type: appActionSetSystemPromptInput, SystemPromptInput: resolvedPrompt})
		}
		return nil
	}, app.Get().GRPCReady, customSystemPromptCacheState.Ready, customSystemPromptCacheState.Value, app.Get().ShowNameModal)

	refresh := func(force bool) {
		if !app.Get().Authenticated {
			return
		}
		if !force && time.Since(userNameFetchedAt.Get()) < userNameTTL {
			return
		}
		client := chatClientRef.Get()
		if client == nil {
			return
		}
		go func() {
			resp, err := client.GetUserName(context.Background(), &chatpb.GetUserNameRequest{})
			if err != nil {
				if handleAuthFailure != nil && handleAuthFailure(err) {
					return
				}
				chatLog.Error("get user name failed", logging.Fields{"error": err})
				return
			}
			if resp.Name != "" {
				userNameState.Set(resp.Name)
			}
			memoriesResp, memoriesErr := client.ListUserMemories(context.Background(), &chatpb.ListUserMemoriesRequest{})
			if memoriesErr != nil {
				if handleAuthFailure != nil && handleAuthFailure(memoriesErr) {
					return
				}
				chatLog.Error("list user memories failed", logging.Fields{"error": memoriesErr})
				return
			}
			memories := make([]editableUserMemory, 0, len(memoriesResp.Memories))
			for _, memory := range memoriesResp.Memories {
				memories = append(memories, editableUserMemory{
					Key:             memory.GetKey(),
					Category:        memory.GetCategory(),
					Summary:         memory.GetSummary(),
					Detail:          memory.GetDetail(),
					SourceMessage:   memory.GetSourceMessage(),
					UsefulnessScore: int(memory.GetUsefulnessScore()),
					ConfidenceScore: memory.GetConfidenceScore(),
					RubricReason:    memory.GetRubricReason(),
					UpdatedAt:       memory.GetUpdatedAt(),
				})
			}
			memories = ensureManagedUserNameMemory(resp.Name, memories)
			app.Dispatch(appAction{Type: appActionSetUserMemories, UserMemories: memories, DeletedUserMemoryKeys: []string{}})
			userNameFetchedAt.Set(time.Now())
			if force {
				chatLog.Info("profile", logging.Fields{"name": resp.Name, "memories": len(memories)})
			}
		}()
	}

	saveSettings := func() {
		currentState := app.Get()
		name := strings.TrimSpace(currentState.NameInput)
		selectedToneValue := normalizeSelectedToneID(currentState.ToneInput)
		selectedThinkingEnabledValue := currentState.ThinkingEnabledInput
		selectedThinkingEffortValue := normalizeSelectedThinkingEffort(currentState.ThinkingEffortInput)
		systemPromptValue := strings.TrimSpace(currentState.SystemPromptInput)
		selectedLocaleValue := normalizeChatLocaleID(currentState.LocaleInput)
		if name != "" {
			chatLog.Info("profile save", logging.Fields{"name": name, "tone": selectedToneValue, "system_prompt_len": len([]rune(systemPromptValue))})
			userNameState.Set(name)
			client := chatClientRef.Get()
			if client != nil {
				go func() {
					_, err := client.SetUserName(context.Background(), &chatpb.SetUserNameRequest{Name: name})
					if err != nil {
						if handleAuthFailure != nil && handleAuthFailure(err) {
							return
						}
						chatLog.Error("set user name failed", logging.Fields{"error": err})
						return
					}
					userNameFetchedAt.Set(time.Now())
				}()
			}
		}
		intl.SetLocale(selectedLocaleValue)
		app.Dispatch(appAction{Type: appActionSetSelectedTone, SelectedTone: selectedToneValue})
		app.Dispatch(appAction{Type: appActionSetSelectedThinkingEnabled, SelectedThinkingEnabled: selectedThinkingEnabledValue})
		if selectedThinkingEnabledValue {
			app.Dispatch(appAction{Type: appActionSetSelectedThinkingEffort, SelectedThinkingEffort: selectedThinkingEffortValue})
		}
		app.Dispatch(appAction{Type: appActionSetCustomSystemPrompt, CustomSystemPrompt: systemPromptValue})
		selectedToneCache.Set(selectedToneValue)
		selectedThinkingEnabledCache.Set(selectedThinkingEnabledValue)
		if selectedThinkingEnabledValue {
			selectedThinkingEffortCache.Set(selectedThinkingEffortValue)
		}
		customSystemPromptCache.Set(systemPromptValue)
		if client := chatClientRef.Get(); client != nil {
			go func() {
				_, err := client.SetSelectedTone(context.Background(), wrapperspb.String(selectedToneValue))
				if err != nil {
					if handleAuthFailure != nil && handleAuthFailure(err) {
						return
					}
					chatLog.Error("set selected tone failed", logging.Fields{"error": err})
					selectedToneCache.Invalidate()
				}
			}()
			go func() {
				_, err := client.SetSelectedThinkingEnabled(context.Background(), wrapperspb.Bool(selectedThinkingEnabledValue))
				if err != nil {
					if handleAuthFailure != nil && handleAuthFailure(err) {
						return
					}
					chatLog.Error("set selected thinking enabled failed", logging.Fields{"error": err})
					selectedThinkingEnabledCache.Invalidate()
				}
			}()
			if selectedThinkingEnabledValue {
				go func() {
					_, err := client.SetSelectedThinkingEffort(context.Background(), wrapperspb.String(selectedThinkingEffortValue))
					if err != nil {
						if handleAuthFailure != nil && handleAuthFailure(err) {
							return
						}
						chatLog.Error("set selected thinking effort failed", logging.Fields{"error": err})
						selectedThinkingEffortCache.Invalidate()
					}
				}()
			}
			go func() {
				_, err := client.SetCustomSystemPrompt(context.Background(), wrapperspb.String(systemPromptValue))
				if err != nil {
					if handleAuthFailure != nil && handleAuthFailure(err) {
						return
					}
					chatLog.Error("set custom system prompt failed", logging.Fields{"error": err})
					customSystemPromptCache.Invalidate()
				}
			}()
			for _, key := range currentState.DeletedUserMemoryKeys {
				deleteKey := strings.TrimSpace(key)
				if deleteKey == "" {
					continue
				}
				go func(memoryKey string) {
					if _, err := client.DeleteUserMemory(context.Background(), &chatpb.DeleteUserMemoryRequest{Key: memoryKey}); err != nil {
						if handleAuthFailure != nil && handleAuthFailure(err) {
							return
						}
						chatLog.Error("delete user memory failed", logging.Fields{"error": err, "key": memoryKey})
					}
				}(deleteKey)
			}
			for _, memory := range currentState.UserMemories {
				if isManagedUserNameMemory(memory) {
					continue
				}
				summary := strings.TrimSpace(memory.Summary)
				if summary == "" {
					continue
				}
				request := &chatpb.UpsertUserMemoryRequest{
					Memory: &chatpb.UserMemory{
						Key:             strings.TrimSpace(memory.Key),
						Category:        strings.TrimSpace(memory.Category),
						Summary:         summary,
						Detail:          strings.TrimSpace(memory.Detail),
						SourceMessage:   strings.TrimSpace(memory.SourceMessage),
						UsefulnessScore: int32(memory.UsefulnessScore),
						ConfidenceScore: memory.ConfidenceScore,
						RubricReason:    strings.TrimSpace(memory.RubricReason),
						UpdatedAt:       memory.UpdatedAt,
					},
				}
				go func(req *chatpb.UpsertUserMemoryRequest) {
					if _, err := client.UpsertUserMemory(context.Background(), req); err != nil {
						if handleAuthFailure != nil && handleAuthFailure(err) {
							return
						}
						chatLog.Error("upsert user memory failed", logging.Fields{"error": err, "summary": req.GetMemory().GetSummary()})
					}
				}(request)
			}
		}
		app.Dispatch(appAction{Type: appActionSetToneInput, ToneInput: selectedToneValue})
		app.Dispatch(appAction{Type: appActionSetSystemPromptInput, SystemPromptInput: systemPromptValue})
		app.Dispatch(appAction{Type: appActionSetShowNameModal, ShowNameModal: false})
		app.Dispatch(appAction{Type: appActionSetUserMemories, UserMemories: ensureManagedUserNameMemory(name, currentState.UserMemories), DeletedUserMemoryKeys: []string{}})
	}

	open := ui.UseEvent(func() {
		app.Dispatch(appAction{Type: appActionSetNameInput, NameInput: userNameState.Get()})
		app.Dispatch(appAction{Type: appActionSetToneInput, ToneInput: app.Get().SelectedTone})
		app.Dispatch(appAction{Type: appActionSetThinkingEnabledInput, ThinkingEnabledInput: app.Get().SelectedThinkingEnabled})
		app.Dispatch(appAction{Type: appActionSetThinkingEffortInput, ThinkingEffortInput: app.Get().SelectedThinkingEffort})
		app.Dispatch(appAction{Type: appActionSetSystemPromptInput, SystemPromptInput: app.Get().CustomSystemPrompt})
		app.Dispatch(appAction{Type: appActionSetLocaleInput, LocaleInput: normalizeChatLocaleID(intl.Locale())})
		app.Dispatch(appAction{Type: appActionSetUserMemories, UserMemories: ensureManagedUserNameMemory(userNameState.Get(), app.Get().UserMemories), DeletedUserMemoryKeys: []string{}})
		app.Dispatch(appAction{Type: appActionSetShowNameModal, ShowNameModal: true})
	})

	close := ui.UseEvent(func() {
		app.Dispatch(appAction{Type: appActionSetShowNameModal, ShowNameModal: false})
	})

	handleNameInput := ui.UseEvent(func(e ui.Event) {
		nextName := e.GetValue()
		app.Dispatch(appAction{Type: appActionSetNameInput, NameInput: nextName})
		app.Dispatch(appAction{Type: appActionSetUserMemories, UserMemories: ensureManagedUserNameMemory(nextName, app.Get().UserMemories), DeletedUserMemoryKeys: app.Get().DeletedUserMemoryKeys})
	})

	handleNameKey := ui.UseEvent(func(e ui.Event) {
		switch e.GetKey() {
		case "Enter":
			e.PreventDefault()
			saveSettings()
		case "Escape":
			app.Dispatch(appAction{Type: appActionSetShowNameModal, ShowNameModal: false})
		}
	})

	handleToneChange := ui.UseEvent(func(e ui.Event) {
		app.Dispatch(appAction{Type: appActionSetToneInput, ToneInput: normalizeSelectedToneID(eventDatasetValue(e, dataTone))})
	})

	handleThinkingMode := ui.UseEvent(func(e ui.Event) {
		nextMode := strings.TrimSpace(strings.ToLower(eventDatasetValue(e, dataThinkingEffort)))
		nextEnabled := nextMode != "off"
		app.Dispatch(appAction{Type: appActionSetThinkingEnabledInput, ThinkingEnabledInput: nextEnabled})
		if nextEnabled {
			app.Dispatch(appAction{Type: appActionSetThinkingEffortInput, ThinkingEffortInput: normalizeSelectedThinkingEffort(nextMode)})
		}
	})

	handleSystemPrompt := ui.UseEvent(func(e ui.Event) {
		app.Dispatch(appAction{Type: appActionSetSystemPromptInput, SystemPromptInput: e.GetValue()})
	})

	handleMemoryChange := ui.UseEvent(func(e ui.Event) {
		index, ok := eventDatasetInt(e, dataMemoryIndex)
		if !ok {
			return
		}
		currentState := app.Get()
		if index < 0 || index >= len(currentState.UserMemories) {
			return
		}
		if isManagedUserNameMemory(currentState.UserMemories[index]) {
			return
		}
		field := eventDatasetValue(e, dataMemoryField)
		app.Dispatch(appAction{Type: appActionUpdateUserMemoryField, UserMemoryIndex: index, UserMemoryField: field, UserMemoryValue: e.GetValue()})
	})

	addMemory := ui.UseEvent(func() {
		app.Dispatch(appAction{Type: appActionAddUserMemory})
	})

	deleteMemory := ui.UseEvent(func(e ui.Event) {
		index, ok := eventDatasetInt(e, dataMemoryIndex)
		if !ok {
			return
		}
		currentState := app.Get()
		if index < 0 || index >= len(currentState.UserMemories) {
			return
		}
		if isManagedUserNameMemory(currentState.UserMemories[index]) {
			return
		}
		app.Dispatch(appAction{Type: appActionDeleteUserMemory, UserMemoryIndex: index})
	})

	handleLocaleChange := ui.UseEvent(func(e ui.Event) {
		app.Dispatch(appAction{Type: appActionSetLocaleInput, LocaleInput: normalizeChatLocaleID(eventDatasetValue(e, dataLocale))})
	})

	save := ui.UseEvent(func() {
		saveSettings()
	})

	return profileSettingsController{
		Refresh:            refresh,
		Open:               open,
		Close:              close,
		HandleNameInput:    handleNameInput,
		HandleNameKey:      handleNameKey,
		HandleToneChange:   handleToneChange,
		HandleThinkingMode: handleThinkingMode,
		HandleSystemPrompt: handleSystemPrompt,
		HandleMemoryChange: handleMemoryChange,
		AddMemory:          addMemory,
		DeleteMemory:       deleteMemory,
		HandleLocaleChange: handleLocaleChange,
		Save:               save,
	}
}
