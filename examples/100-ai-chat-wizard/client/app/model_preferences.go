//go:build js && wasm

package app

import (
	"context"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/interop"
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
	RefreshCatalog               func()
	SetProvider                  ui.Handler
	SetModel                     ui.Handler
	SetThinkingMode              ui.Handler
}

func modelCatalogFromResponse(resp *chatpb.ListModelOptionsResponse) modelCatalog {
	catalog := defaultModelCatalog()
	if resp == nil {
		return catalog
	}
	options := make([]modelOption, 0, len(resp.Models))
	for _, option := range resp.Models {
		capabilities := option.GetCapabilities()
		pricing := option.GetPricing()
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
			Pricing: modelPricing{
				InputDollarsPerMillion:  pricing.GetInputCostPerMillionUsd(),
				OutputDollarsPerMillion: pricing.GetOutputCostPerMillionUsd(),
				Currency:                pricing.GetCurrency(),
			},
		})
	}
	if len(options) == 0 {
		return catalog
	}
	catalog.Models = options
	catalog.DefaultModel = normalizeSelectedModelID(resp.GetDefaultModel(), options, catalog.DefaultModel)
	return catalog
}

// useModelPreferences hides model-catalog loading, selected-model sync, and
// thinking preference persistence behind a feature-specific hook.
func useModelPreferences(
	app ui.Reducer[appState, appAction],
	chatClientRef ui.Ref[chatpb.ChatServiceClient],
	handleAuthFailure func(error) bool,
) modelPreferencesController {
	modelCatalogRefreshOnConnect := ui.UseRef(false)
	modelCatalogBootstrapRequested := ui.UseRef(false)
	selectedModelChannelRef := ui.UseRef(interop.CrossTabChannel{})
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
		return modelCatalogFromResponse(resp), nil
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

	publishSelectedModelCrossTab := func(nextModel string) {
		channel := selectedModelChannelRef.Get()
		if channel.Name() == "" {
			return
		}
		if err := channel.Publish(selectedModelCrossTabMessage{Model: nextModel}); err != nil {
			chatLog.Warn("publish selected model cross-tab failed", logging.Fields{"error": err, "model": nextModel, "channel": channel.Name()})
		}
	}

	persistSelectedModel := func(nextModel string) {
		nextModel = strings.TrimSpace(nextModel)
		if nextModel == "" {
			return
		}
		selectedModelCache.Set(nextModel)
		if client := chatClientRef.Get(); client != nil {
			go func(modelID string) {
				_, err := client.SetSelectedModel(context.Background(), wrapperspb.String(modelID))
				if err != nil {
					if handleAuthFailure != nil && handleAuthFailure(err) {
						return
					}
					chatLog.Error("set selected model failed", logging.Fields{"error": err, "model": modelID})
					selectedModelCache.Invalidate()
					return
				}
				publishSelectedModelCrossTab(modelID)
			}(nextModel)
		}
	}

	persistThinkingPreferences := func(enabled bool, effort string) {
		resolvedEffort := normalizeSelectedThinkingEffort(effort)
		selectedThinkingEnabledCache.Set(enabled)
		selectedThinkingEffortCache.Set(resolvedEffort)
		if client := chatClientRef.Get(); client != nil {
			go func(nextEnabled bool) {
				_, err := client.SetSelectedThinkingEnabled(context.Background(), wrapperspb.Bool(nextEnabled))
				if err != nil {
					if handleAuthFailure != nil && handleAuthFailure(err) {
						return
					}
					chatLog.Error("set selected thinking enabled failed", logging.Fields{"error": err})
					selectedThinkingEnabledCache.Invalidate()
				}
			}(enabled)
			go func(nextEffort string) {
				_, err := client.SetSelectedThinkingEffort(context.Background(), wrapperspb.String(nextEffort))
				if err != nil {
					if handleAuthFailure != nil && handleAuthFailure(err) {
						return
					}
					chatLog.Error("set selected thinking effort failed", logging.Fields{"error": err})
					selectedThinkingEffortCache.Invalidate()
				}
			}(resolvedEffort)
		}
	}

	refreshModelCatalog := func() {
		client := chatClientRef.Get()
		if client == nil {
			return
		}
		go func() {
			resp, err := client.ListModelOptions(context.Background(), &chatpb.ListModelOptionsRequest{})
			if err != nil {
				if handleAuthFailure != nil && handleAuthFailure(err) {
					return
				}
				chatLog.Error("list model options failed", logging.Fields{"error": err})
				return
			}
			modelCatalogCache.Set(modelCatalogFromResponse(resp))
		}()
	}

	applyModelSelection := func(nextModel string) bool {
		currentState := app.Get()
		resolvedModel := normalizeSelectedModelID(nextModel, currentState.ModelOptions, currentState.DefaultModel)
		if resolvedModel == "" {
			return false
		}
		if resolvedModel == currentState.SelectedModel {
			return true
		}
		app.Dispatch(appAction{Type: appActionSetSelectedModel, SelectedModel: resolvedModel})
		persistSelectedModel(resolvedModel)
		if len(currentState.Messages) > 0 {
			app.Dispatch(appAction{
				Type: appActionUpdateMessages,
				UpdateMessages: func(prev []message) []message {
					return append(append([]message{}, prev...), message{Role: roleSwitch, Content: resolvedModel})
				},
			})
		}
		return true
	}

	ui.UseEffect(func() func() {
		if !app.Get().GRPCReady {
			modelCatalogRefreshOnConnect.Set(false)
			return nil
		}
		if modelCatalogRefreshOnConnect.Get() {
			return nil
		}
		modelCatalogRefreshOnConnect.Set(true)
		modelCatalogCache.Invalidate()
		return nil
	}, app.Get().GRPCReady)

	ui.UseEffect(func() func() {
		currentState := app.Get()
		if !currentState.GRPCReady || !currentState.Authenticated {
			selectedModelChannelRef.Set(interop.CrossTabChannel{})
			return nil
		}
		channel, err := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{
			Name: selectedModelCrossTabChannelName(currentState.SessionEmail),
		})
		if err != nil {
			selectedModelChannelRef.Set(interop.CrossTabChannel{})
			chatLog.Warn("open selected model cross-tab channel failed", logging.Fields{"error": err, "email": currentState.SessionEmail})
			return nil
		}
		selectedModelChannelRef.Set(channel)
		subscription, err := interop.SubscribeDecodedCrossTab[selectedModelCrossTabMessage](channel, func(message interop.DecodedCrossTabEnvelope[selectedModelCrossTabMessage], subErr error) {
			if subErr != nil {
				chatLog.Warn("selected model cross-tab subscribe failed", logging.Fields{"error": subErr, "channel": channel.Name()})
				return
			}
			currentState := app.Get()
			if !currentState.GRPCReady || !currentState.Authenticated || currentState.Streaming {
				return
			}
			resolvedModel := normalizeSelectedModelID(message.Payload.Model, currentState.ModelOptions, currentState.DefaultModel)
			if resolvedModel == "" || resolvedModel == currentState.SelectedModel {
				return
			}
			selectedModelCache.Set(resolvedModel)
			app.Dispatch(appAction{Type: appActionSetSelectedModel, SelectedModel: resolvedModel})
		})
		if err != nil {
			selectedModelChannelRef.Set(interop.CrossTabChannel{})
			_ = channel.Close()
			chatLog.Warn("selected model cross-tab setup failed", logging.Fields{"error": err, "channel": channel.Name()})
			return nil
		}
		return func() {
			selectedModelChannelRef.Set(interop.CrossTabChannel{})
			subscription.Cancel()
			_ = channel.Close()
		}
	}, app.Get().GRPCReady, app.Get().Authenticated, app.Get().SessionEmail)

	ui.UseEffect(func() func() {
		currentState := app.Get()
		if !currentState.GRPCReady || !currentState.Authenticated {
			modelCatalogBootstrapRequested.Set(false)
			return nil
		}
		if len(currentState.ModelOptions) > 0 {
			modelCatalogBootstrapRequested.Set(false)
			return nil
		}
		if chatClientRef.Get() == nil {
			return nil
		}
		if modelCatalogBootstrapRequested.Get() {
			return nil
		}
		modelCatalogBootstrapRequested.Set(true)
		refreshModelCatalog()
		return nil
	}, app.Get().GRPCReady, app.Get().Authenticated, len(app.Get().ModelOptions), modelCatalogCacheState.Loading)

	ui.UseEffect(func() func() {
		currentState := app.Get()
		if !currentState.GRPCReady || !modelCatalogCacheState.Ready {
			return nil
		}
		catalog := modelCatalogCacheState.Value
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
		if !currentState.GRPCReady || !currentState.Authenticated || !selectedModelCacheState.Ready {
			return nil
		}
		if currentState.ActiveConvID > 0 || len(currentState.Messages) > 0 || len(currentState.ModelOptions) == 0 {
			return nil
		}
		recoveredModel, usedFallback := recoverPersistedModelSelection(selectedModelCacheState.Value, currentState.ModelOptions)
		if recoveredModel == "" || strings.TrimSpace(selectedModelCacheState.Value) == recoveredModel {
			return nil
		}
		if recoveredModel != currentState.SelectedModel {
			app.Dispatch(appAction{Type: appActionSetSelectedModel, SelectedModel: recoveredModel})
		}
		persistSelectedModel(recoveredModel)
		if usedFallback {
			if !currentState.SelectedThinkingEnabled {
				app.Dispatch(appAction{Type: appActionSetSelectedThinkingEnabled, SelectedThinkingEnabled: true})
			}
			if currentState.SelectedThinkingEffort != defaultThinkingEffort {
				app.Dispatch(appAction{Type: appActionSetSelectedThinkingEffort, SelectedThinkingEffort: defaultThinkingEffort})
			}
			persistThinkingPreferences(true, defaultThinkingEffort)
		}
		return nil
	}, app.Get().GRPCReady, selectedModelCacheState.Ready, selectedModelCacheState.Value, app.Get().ActiveConvID, len(app.Get().Messages), app.Get().ModelOptions, app.Get().SelectedModel, app.Get().SelectedThinkingEnabled, app.Get().SelectedThinkingEffort)

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

	setModel := ui.UseEvent(func(e ui.Event) {
		if app.Get().Streaming {
			return
		}
		currentState := app.Get()
		newModel := normalizeSelectedModelID(eventValueOrDataset(e, dataModel), currentState.ModelOptions, currentState.DefaultModel)
		if newModel == app.Get().SelectedModel {
			return
		}
		chatLog.Info("model set", logging.Fields{"model": newModel})
		_ = applyModelSelection(newModel)
	})

	setProvider := ui.UseEvent(func(e ui.Event) {
		if app.Get().Streaming {
			return
		}
		currentState := app.Get()
		providerID := strings.TrimSpace(eventValueOrDataset(e, dataProvider))
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
		_ = applyModelSelection(nextModel)
	})

	setThinkingMode := ui.UseEvent(func(e ui.Event) {
		if app.Get().Streaming {
			return
		}
		currentState := app.Get()
		if !modelSupportsThinking(currentState.SelectedModel, currentState.ModelOptions, currentState.DefaultModel) {
			return
		}
		nextMode := strings.TrimSpace(strings.ToLower(eventValueOrDataset(e, dataThinkingEffort)))
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
		}
		persistThinkingPreferences(nextEnabled, nextEffort)
	})

	return modelPreferencesController{
		SelectedToneCache:            selectedToneCache,
		SelectedThinkingEnabledCache: selectedThinkingEnabledCache,
		SelectedThinkingEffortCache:  selectedThinkingEffortCache,
		RefreshCatalog:               refreshModelCatalog,
		SetProvider:                  setProvider,
		SetModel:                     setModel,
		SetThinkingMode:              setThinkingMode,
	}
}
