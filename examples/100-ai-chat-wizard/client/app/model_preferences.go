//go:build js && wasm

package app

import (
	"context"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/logging"
	"github.com/monstercameron/GoWebComponents/ui"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

// modelPreferencesController owns model and thinking preference sync.
//
// App() only needs the resulting handlers plus the selected-tone cache for the
// settings hook. The fetch resources and persistence plumbing stay here.
type modelPreferencesController struct {
	SelectedToneCache            fetch.CachedResource[string]
	SelectedThinkingEnabledCache fetch.CachedResource[bool]
	SelectedThinkingEffortCache  fetch.CachedResource[string]
	SetProvider                  ui.Handler
	SetModel                     ui.Handler
	SetThinkingMode              ui.Handler
	ResetDraftModel              func()
}

// useModelPreferences hides model-catalog loading, selected-model sync, and
// thinking preference persistence behind a feature-specific hook.
func useModelPreferences(
	app ui.Reducer[appState, appAction],
	chatClientRef ui.Ref[chatpb.ChatServiceClient],
	handleAuthFailure func(error) bool,
) modelPreferencesController {
	modelCatalogCacheKey := ""
	if app.Get().GRPCReady {
		modelCatalogCacheKey = cacheKeyModelCatalog
	}
	modelCatalogCache := fetch.UseCachedResource(modelCatalogCacheKey, func(ctx context.Context) (modelCatalog, error) {
		catalog := defaultModelCatalog()
		client := chatClientRef.Get()
		if client == nil {
			return catalog, nil
		}
		resp, err := client.ListModelOptions(ctx, &chatpb.ListModelOptionsRequest{})
		if err != nil {
			return modelCatalog{}, err
		}
		options := make([]modelOption, 0, len(resp.Models))
		for _, option := range resp.Models {
			capabilities := option.GetCapabilities()
			options = append(options, modelOption{
				ID:    option.GetId(),
				Label: option.GetLabel(),
				Note:  option.GetNote(),
				Capabilities: modelCapabilities{
					ProviderID:       capabilities.GetProviderId(),
					ProviderLabel:    capabilities.GetProviderLabel(),
					SupportsThinking: capabilities.GetSupportsThinking(),
					SupportsSpeech:   capabilities.GetSupportsSpeech(),
				},
			})
		}
		if len(options) == 0 {
			return catalog, nil
		}
		catalog.Models = options
		catalog.DefaultModel = normalizeSelectedModelID(resp.GetDefaultModel(), options, catalog.DefaultModel)
		return catalog, nil
	}, fetch.CacheOptions{StaleAfter: modelTTL, MaxAge: modelTTL, Persist: true})
	modelCatalogCacheState := modelCatalogCache.Get()

	selectedModelCacheKey := ""
	if app.Get().GRPCReady && app.Get().Authenticated {
		selectedModelCacheKey = cacheKeySelectedModel
	}
	selectedModelCache := fetch.UseCachedResource(selectedModelCacheKey, func(ctx context.Context) (string, error) {
		client := chatClientRef.Get()
		if client == nil {
			return app.Get().DefaultModel, nil
		}
		resp, err := client.GetSelectedModel(ctx, &emptypb.Empty{})
		if err != nil {
			if handleAuthFailure != nil && handleAuthFailure(err) {
				return app.Get().DefaultModel, nil
			}
			return "", err
		}
		currentState := app.Get()
		return normalizeSelectedModelID(resp.GetValue(), currentState.ModelOptions, currentState.DefaultModel), nil
	}, fetch.CacheOptions{StaleAfter: modelTTL, MaxAge: modelTTL, Persist: true})
	selectedModelCacheState := selectedModelCache.Get()

	selectedToneCacheKey := ""
	if app.Get().GRPCReady && app.Get().Authenticated {
		selectedToneCacheKey = cacheKeySelectedTone
	}
	selectedToneCache := fetch.UseCachedResource(selectedToneCacheKey, func(ctx context.Context) (string, error) {
		client := chatClientRef.Get()
		if client == nil {
			return defaultTone, nil
		}
		resp, err := client.GetSelectedTone(ctx, &emptypb.Empty{})
		if err != nil {
			if handleAuthFailure != nil && handleAuthFailure(err) {
				return defaultTone, nil
			}
			return "", err
		}
		return normalizeSelectedToneID(resp.GetValue()), nil
	}, fetch.CacheOptions{StaleAfter: toneTTL, MaxAge: toneTTL, Persist: true})
	selectedToneCacheState := selectedToneCache.Get()

	selectedThinkingEnabledCacheKey := ""
	if app.Get().GRPCReady && app.Get().Authenticated {
		selectedThinkingEnabledCacheKey = cacheKeySelectedThinkingEnabled
	}
	selectedThinkingEnabledCache := fetch.UseCachedResource(selectedThinkingEnabledCacheKey, func(ctx context.Context) (bool, error) {
		client := chatClientRef.Get()
		if client == nil {
			return defaultThinkingEnabled, nil
		}
		resp, err := client.GetSelectedThinkingEnabled(ctx, &emptypb.Empty{})
		if err != nil {
			if handleAuthFailure != nil && handleAuthFailure(err) {
				return defaultThinkingEnabled, nil
			}
			return false, err
		}
		return resp.GetValue(), nil
	}, fetch.CacheOptions{StaleAfter: toneTTL, MaxAge: toneTTL, Persist: true})
	selectedThinkingEnabledCacheState := selectedThinkingEnabledCache.Get()

	selectedThinkingEffortCacheKey := ""
	if app.Get().GRPCReady && app.Get().Authenticated {
		selectedThinkingEffortCacheKey = cacheKeySelectedThinkingEffort
	}
	selectedThinkingEffortCache := fetch.UseCachedResource(selectedThinkingEffortCacheKey, func(ctx context.Context) (string, error) {
		client := chatClientRef.Get()
		if client == nil {
			return defaultThinkingEffort, nil
		}
		resp, err := client.GetSelectedThinkingEffort(ctx, &emptypb.Empty{})
		if err != nil {
			if handleAuthFailure != nil && handleAuthFailure(err) {
				return defaultThinkingEffort, nil
			}
			return "", err
		}
		return normalizeSelectedThinkingEffort(resp.GetValue()), nil
	}, fetch.CacheOptions{StaleAfter: toneTTL, MaxAge: toneTTL, Persist: true})
	selectedThinkingEffortCacheState := selectedThinkingEffortCache.Get()

	ui.UseEffect(func() func() {
		currentState := app.Get()
		if !currentState.GRPCReady || !modelCatalogCacheState.Ready {
			return nil
		}
		catalog := modelCatalogCacheState.Value
		if len(catalog.Models) == 0 {
			catalog = defaultModelCatalog()
		}
		resolvedDefaultModel := normalizeSelectedModelID(catalog.DefaultModel, catalog.Models, defaultModel)
		if !sameModelOptions(currentState.ModelOptions, catalog.Models) || currentState.DefaultModel != resolvedDefaultModel {
			app.Dispatch(appAction{Type: appActionSetModelCatalog, ModelOptions: catalog.Models, DefaultModel: resolvedDefaultModel})
		}
		return nil
	}, app.Get().GRPCReady, modelCatalogCacheState.Ready, modelCatalogCacheState.Value)

	ui.UseEffect(func() func() {
		currentState := app.Get()
		if !currentState.GRPCReady || !currentState.Authenticated || !selectedModelCacheState.Ready {
			return nil
		}
		if currentState.ActiveConvID > 0 || len(currentState.Messages) > 0 {
			return nil
		}
		resolvedModel := normalizeSelectedModelID(selectedModelCacheState.Value, currentState.ModelOptions, currentState.DefaultModel)
		if resolvedModel != currentState.SelectedModel {
			app.Dispatch(appAction{Type: appActionSetSelectedModel, SelectedModel: resolvedModel})
		}
		return nil
	}, app.Get().GRPCReady, selectedModelCacheState.Ready, selectedModelCacheState.Value, app.Get().ActiveConvID, len(app.Get().Messages))

	ui.UseEffect(func() func() {
		currentState := app.Get()
		if !currentState.GRPCReady || !currentState.Authenticated || !selectedToneCacheState.Ready {
			return nil
		}
		resolvedTone := normalizeSelectedToneID(selectedToneCacheState.Value)
		if resolvedTone != currentState.SelectedTone {
			app.Dispatch(appAction{Type: appActionSetSelectedTone, SelectedTone: resolvedTone})
		}
		if !currentState.ShowNameModal && resolvedTone != currentState.ToneInput {
			app.Dispatch(appAction{Type: appActionSetToneInput, ToneInput: resolvedTone})
		}
		return nil
	}, app.Get().GRPCReady, selectedToneCacheState.Ready, selectedToneCacheState.Value, app.Get().ShowNameModal)

	ui.UseEffect(func() func() {
		currentState := app.Get()
		if !currentState.GRPCReady || !currentState.Authenticated || !selectedThinkingEnabledCacheState.Ready {
			return nil
		}
		if selectedThinkingEnabledCacheState.Value != currentState.SelectedThinkingEnabled {
			app.Dispatch(appAction{Type: appActionSetSelectedThinkingEnabled, SelectedThinkingEnabled: selectedThinkingEnabledCacheState.Value})
		}
		return nil
	}, app.Get().GRPCReady, selectedThinkingEnabledCacheState.Ready, selectedThinkingEnabledCacheState.Value)

	ui.UseEffect(func() func() {
		currentState := app.Get()
		if !currentState.GRPCReady || !currentState.Authenticated || !selectedThinkingEffortCacheState.Ready {
			return nil
		}
		resolvedThinkingEffort := normalizeSelectedThinkingEffort(selectedThinkingEffortCacheState.Value)
		if resolvedThinkingEffort != currentState.SelectedThinkingEffort {
			app.Dispatch(appAction{Type: appActionSetSelectedThinkingEffort, SelectedThinkingEffort: resolvedThinkingEffort})
		}
		return nil
	}, app.Get().GRPCReady, selectedThinkingEffortCacheState.Ready, selectedThinkingEffortCacheState.Value)

	resetDraftModel := func() {
		currentDefaultModel := app.Get().DefaultModel
		if app.Get().SelectedModel != currentDefaultModel {
			app.Dispatch(appAction{Type: appActionSetSelectedModel, SelectedModel: currentDefaultModel})
		}
		selectedModelCache.Set(currentDefaultModel)
		if client := chatClientRef.Get(); client != nil {
			go func() {
				_, err := client.SetSelectedModel(context.Background(), wrapperspb.String(currentDefaultModel))
				if err != nil {
					if handleAuthFailure != nil && handleAuthFailure(err) {
						return
					}
					chatLog.Error("reset selected model failed", logging.Fields{"error": err})
					selectedModelCache.Invalidate()
				}
			}()
		}
	}

	setModel := ui.UseEvent(func(e ui.Event) {
		if app.Get().Streaming {
			return
		}
		currentState := app.Get()
		newModel := normalizeSelectedModelID(eventDatasetValue(e, dataModel), currentState.ModelOptions, currentState.DefaultModel)
		if newModel == app.Get().SelectedModel {
			return
		}
		chatLog.Info("model set", logging.Fields{"model": newModel})
		app.Dispatch(appAction{Type: appActionSetSelectedModel, SelectedModel: newModel})
		selectedModelCache.Set(newModel)
		if client := chatClientRef.Get(); client != nil {
			go func() {
				_, err := client.SetSelectedModel(context.Background(), wrapperspb.String(newModel))
				if err != nil {
					if handleAuthFailure != nil && handleAuthFailure(err) {
						return
					}
					chatLog.Error("set selected model failed", logging.Fields{"error": err})
					selectedModelCache.Invalidate()
				}
			}()
		}
		if len(app.Get().Messages) > 0 {
			app.Dispatch(appAction{
				Type: appActionUpdateMessages,
				UpdateMessages: func(prev []message) []message {
					return append(append([]message{}, prev...), message{Role: roleSwitch, Content: newModel})
				},
			})
		}
	})

	setProvider := ui.UseEvent(func(e ui.Event) {
		if app.Get().Streaming {
			return
		}
		currentState := app.Get()
		providerID := strings.TrimSpace(eventDatasetValue(e, dataProvider))
		if providerID == "" {
			return
		}
		currentProvider := providerForModel(currentState.SelectedModel, currentState.ModelOptions, currentState.DefaultModel)
		if currentProvider.ID == providerID {
			return
		}
		nextModel := defaultModelForProvider(providerID, currentState.ModelOptions, currentState.DefaultModel)
		if nextModel == "" || nextModel == currentState.SelectedModel {
			return
		}
		app.Dispatch(appAction{Type: appActionSetSelectedModel, SelectedModel: nextModel})
		selectedModelCache.Set(nextModel)
		if client := chatClientRef.Get(); client != nil {
			go func() {
				_, err := client.SetSelectedModel(context.Background(), wrapperspb.String(nextModel))
				if err != nil {
					if handleAuthFailure != nil && handleAuthFailure(err) {
						return
					}
					chatLog.Error("set selected provider model failed", logging.Fields{"error": err, "provider": providerID, "model": nextModel})
					selectedModelCache.Invalidate()
				}
			}()
		}
		if len(app.Get().Messages) > 0 {
			app.Dispatch(appAction{
				Type: appActionUpdateMessages,
				UpdateMessages: func(prev []message) []message {
					return append(append([]message{}, prev...), message{Role: roleSwitch, Content: nextModel})
				},
			})
		}
	})

	setThinkingMode := ui.UseEvent(func(e ui.Event) {
		if app.Get().Streaming {
			return
		}
		currentState := app.Get()
		if !modelSupportsThinking(currentState.SelectedModel, currentState.ModelOptions, currentState.DefaultModel) {
			return
		}
		nextMode := strings.TrimSpace(strings.ToLower(eventDatasetValue(e, dataThinkingEffort)))
		nextEnabled := nextMode != "off"
		nextEffort := currentState.SelectedThinkingEffort
		if nextEnabled {
			nextEffort = normalizeSelectedThinkingEffort(nextMode)
		}
		if currentState.SelectedThinkingEnabled == nextEnabled && (!nextEnabled || currentState.SelectedThinkingEffort == nextEffort) {
			return
		}
		app.Dispatch(appAction{Type: appActionSetSelectedThinkingEnabled, SelectedThinkingEnabled: nextEnabled})
		if nextEnabled {
			app.Dispatch(appAction{Type: appActionSetSelectedThinkingEffort, SelectedThinkingEffort: nextEffort})
			selectedThinkingEffortCache.Set(nextEffort)
		}
		selectedThinkingEnabledCache.Set(nextEnabled)
		if client := chatClientRef.Get(); client != nil {
			go func() {
				_, err := client.SetSelectedThinkingEnabled(context.Background(), wrapperspb.Bool(nextEnabled))
				if err != nil {
					if handleAuthFailure != nil && handleAuthFailure(err) {
						return
					}
					chatLog.Error("set selected thinking enabled failed", logging.Fields{"error": err})
					selectedThinkingEnabledCache.Invalidate()
				}
			}()
			if nextEnabled {
				go func() {
					_, err := client.SetSelectedThinkingEffort(context.Background(), wrapperspb.String(nextEffort))
					if err != nil {
						if handleAuthFailure != nil && handleAuthFailure(err) {
							return
						}
						chatLog.Error("set selected thinking effort failed", logging.Fields{"error": err})
						selectedThinkingEffortCache.Invalidate()
					}
				}()
			}
		}
	})

	return modelPreferencesController{
		SelectedToneCache:            selectedToneCache,
		SelectedThinkingEnabledCache: selectedThinkingEnabledCache,
		SelectedThinkingEffortCache:  selectedThinkingEffortCache,
		SetProvider:                  setProvider,
		SetModel:                     setModel,
		SetThinkingMode:              setThinkingMode,
		ResetDraftModel:              resetDraftModel,
	}
}
