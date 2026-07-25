//go:build js && wasm

package app

import (
	"context"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/v5/fetch"
	"github.com/monstercameron/GoWebComponents/v5/interop"
	"github.com/monstercameron/GoWebComponents/v5/logging"
	"github.com/monstercameron/GoWebComponents/v5/ui"
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

func parseModelCatalogFromResponse(parseResp *chatpb.ListModelOptionsResponse) modelCatalog {
	parseCatalog := parseDefaultModelCatalog()
	if parseResp == nil {
		return parseCatalog
	}
	parseOptions := make([]modelOption, 0, len(parseResp.Models))
	for _, parseOption := range parseResp.Models {
		parseCapabilities := parseOption.GetCapabilities()
		parsePricing := parseOption.GetPricing()
		parseOptions = append(parseOptions, modelOption{
			ID:    parseOption.GetId(),
			Label: parseOption.GetLabel(),
			Note:  parseOption.GetNote(),
			Capabilities: modelCapabilities{
				ProviderID:       parseCapabilities.GetProviderId(),
				ProviderLabel:    parseCapabilities.GetProviderLabel(),
				SupportsThinking: parseCapabilities.GetSupportsThinking(),
				SupportsSpeech:   parseCapabilities.GetSupportsSpeech(),
			},
			Pricing: modelPricing{
				InputDollarsPerMillion:  parsePricing.GetInputCostPerMillionUsd(),
				OutputDollarsPerMillion: parsePricing.GetOutputCostPerMillionUsd(),
				Currency:                parsePricing.GetCurrency(),
			},
		})
	}
	if len(parseOptions) == 0 {
		return parseCatalog
	}
	parseCatalog.Models = parseOptions
	parseCatalog.DefaultModel = parseNormalizeSelectedModelID(parseResp.GetDefaultModel(), parseOptions, parseCatalog.DefaultModel)
	return parseCatalog
}

func shouldApplySelectedModelBootstrap(isBootstrapComplete bool, parseCurrentState appState, isCacheReady bool) bool {
	if isBootstrapComplete || !isCacheReady {
		return false
	}
	if !parseCurrentState.GRPCReady || !parseCurrentState.Authenticated {
		return false
	}
	if parseCurrentState.ActiveConvID > 0 || len(parseCurrentState.Messages) > 0 {
		return false
	}
	return true
}

func shouldApplySelectedModelRecovery(isRecoveryComplete bool, parseCurrentState appState, isCacheReady bool) bool {
	if isRecoveryComplete || !isCacheReady {
		return false
	}
	if !parseCurrentState.GRPCReady || !parseCurrentState.Authenticated {
		return false
	}
	if parseCurrentState.ActiveConvID > 0 || len(parseCurrentState.Messages) > 0 || len(parseCurrentState.ModelOptions) == 0 {
		return false
	}
	return true
}

func shouldApplyThinkingPreferencesBootstrap(isBootstrapComplete bool, parseCurrentState appState, isEnabledCacheReady, isEffortCacheReady bool) bool {
	if isBootstrapComplete || !isEnabledCacheReady || !isEffortCacheReady {
		return false
	}
	if !parseCurrentState.GRPCReady || !parseCurrentState.Authenticated {
		return false
	}
	return true
}

// useModelPreferences hides model-catalog loading, selected-model sync, and
// thinking preference persistence behind a feature-specific hook.
func parseUseModelPreferences(
	parseApp ui.Reducer[appState, appAction],
	parseChatClientRef ui.Ref[chatpb.ChatServiceClient],
	handleAuthFailure func(error) bool,
) modelPreferencesController {
	parseModelCatalogRefreshOnConnect := ui.UseRef(false)
	parseModelCatalogBootstrapRequested := ui.UseRef(false)
	parseSelectedModelChannelRef := ui.UseRef(interop.CrossTabChannel{})
	parseModelPreferenceSessionKey := ui.UseRef("")
	parseSelectedModelBootstrapComplete := ui.UseRef(false)
	parseSelectedModelRecoveryComplete := ui.UseRef(false)
	parseThinkingPreferencesBootstrapComplete := ui.UseRef(false)
	parseModelCatalogCacheKey := ""
	if parseApp.Get().GRPCReady {
		parseModelCatalogCacheKey = cacheKeyModelCatalog
	}
	parseModelCatalogCache := fetch.UseCachedResource(parseModelCatalogCacheKey, func(parseCtx context.Context) (modelCatalog, error) {
		parseCatalog := parseDefaultModelCatalog()
		parseClient := parseChatClientRef.Get()
		if parseClient == nil {
			return parseCatalog, nil
		}
		parseResp, parseErr := parseClient.ListModelOptions(parseCtx, &chatpb.ListModelOptionsRequest{})
		if parseErr != nil {
			return modelCatalog{}, parseErr
		}
		return parseModelCatalogFromResponse(parseResp), nil
	}, fetch.CacheOptions{StaleAfter: modelTTL, MaxAge: modelTTL, Persist: true})
	parseModelCatalogCacheState := parseModelCatalogCache.Get()

	parseSelectedModelCacheKey := ""
	if parseApp.Get().GRPCReady && parseApp.Get().Authenticated {
		parseSelectedModelCacheKey = cacheKeySelectedModel
	}
	parseSelectedModelCache := fetch.UseCachedResource(parseSelectedModelCacheKey, func(parseCtx2 context.Context) (string, error) {
		parseClient2 := parseChatClientRef.Get()
		if parseClient2 == nil {
			return parseApp.Get().DefaultModel, nil
		}
		parseResp2, parseErr2 := parseClient2.GetSelectedModel(parseCtx2, &emptypb.Empty{})
		if parseErr2 != nil {
			if handleAuthFailure != nil && handleAuthFailure(parseErr2) {
				return parseApp.Get().DefaultModel, nil
			}
			return "", parseErr2
		}
		parseCurrentState := parseApp.Get()
		return parseNormalizeSelectedModelID(parseResp2.GetValue(), parseCurrentState.ModelOptions, parseCurrentState.DefaultModel), nil
	}, fetch.CacheOptions{StaleAfter: modelTTL, MaxAge: modelTTL, Persist: true})
	parseSelectedModelCacheState := parseSelectedModelCache.Get()

	parseSelectedToneCacheKey := ""
	if parseApp.Get().GRPCReady && parseApp.Get().Authenticated {
		parseSelectedToneCacheKey = cacheKeySelectedTone
	}
	parseSelectedToneCache := fetch.UseCachedResource(parseSelectedToneCacheKey, func(parseCtx3 context.Context) (string, error) {
		parseClient3 := parseChatClientRef.Get()
		if parseClient3 == nil {
			return defaultTone, nil
		}
		parseResp3, parseErr3 := parseClient3.GetSelectedTone(parseCtx3, &emptypb.Empty{})
		if parseErr3 != nil {
			if handleAuthFailure != nil && handleAuthFailure(parseErr3) {
				return defaultTone, nil
			}
			return "", parseErr3
		}
		return parseNormalizeSelectedToneID(parseResp3.GetValue()), nil
	}, fetch.CacheOptions{StaleAfter: toneTTL, MaxAge: toneTTL, Persist: true})
	parseSelectedToneCacheState := parseSelectedToneCache.Get()

	parseSelectedThinkingEnabledCacheKey := ""
	if parseApp.Get().GRPCReady && parseApp.Get().Authenticated {
		parseSelectedThinkingEnabledCacheKey = cacheKeySelectedThinkingEnabled
	}
	parseSelectedThinkingEnabledCache := fetch.UseCachedResource(parseSelectedThinkingEnabledCacheKey, func(parseCtx4 context.Context) (bool, error) {
		parseClient4 := parseChatClientRef.Get()
		if parseClient4 == nil {
			return defaultThinkingEnabled, nil
		}
		parseResp4, parseErr4 := parseClient4.GetSelectedThinkingEnabled(parseCtx4, &emptypb.Empty{})
		if parseErr4 != nil {
			if handleAuthFailure != nil && handleAuthFailure(parseErr4) {
				return defaultThinkingEnabled, nil
			}
			return false, parseErr4
		}
		return parseResp4.GetValue(), nil
	}, fetch.CacheOptions{StaleAfter: toneTTL, MaxAge: toneTTL, Persist: true})
	parseSelectedThinkingEnabledCacheState := parseSelectedThinkingEnabledCache.Get()

	parseSelectedThinkingEffortCacheKey := ""
	if parseApp.Get().GRPCReady && parseApp.Get().Authenticated {
		parseSelectedThinkingEffortCacheKey = cacheKeySelectedThinkingEffort
	}
	parseSelectedThinkingEffortCache := fetch.UseCachedResource(parseSelectedThinkingEffortCacheKey, func(parseCtx5 context.Context) (string, error) {
		parseClient5 := parseChatClientRef.Get()
		if parseClient5 == nil {
			return defaultThinkingEffort, nil
		}
		parseResp5, parseErr5 := parseClient5.GetSelectedThinkingEffort(parseCtx5, &emptypb.Empty{})
		if parseErr5 != nil {
			if handleAuthFailure != nil && handleAuthFailure(parseErr5) {
				return defaultThinkingEffort, nil
			}
			return "", parseErr5
		}
		return parseNormalizeSelectedThinkingEffort(parseResp5.GetValue()), nil
	}, fetch.CacheOptions{StaleAfter: toneTTL, MaxAge: toneTTL, Persist: true})
	parseSelectedThinkingEffortCacheState := parseSelectedThinkingEffortCache.Get()

	parsePublishSelectedModelCrossTab := func(parseNextModel2 string) {
		parseChannel := parseSelectedModelChannelRef.Get()
		if parseChannel.Name() == "" {
			return
		}
		if parseErr6 := parseChannel.Publish(selectedModelCrossTabMessage{Model: parseNextModel2}); parseErr6 != nil {
			chatLog.Warn("publish selected model cross-tab failed", logging.Fields{"error": parseErr6, "model": parseNextModel2, "channel": parseChannel.Name()})
		}
	}

	parsePersistSelectedModel := func(parseNextModel3 string) {
		parseNextModel3 = strings.TrimSpace(parseNextModel3)
		if parseNextModel3 == "" {
			return
		}
		parseSelectedModelCache.Set(parseNextModel3)
		if parseClient6 := parseChatClientRef.Get(); parseClient6 != nil {
			go func(parseModelID string) {
				_, parseErr7 := parseClient6.SetSelectedModel(context.Background(), wrapperspb.String(parseModelID))
				if parseErr7 != nil {
					if handleAuthFailure != nil && handleAuthFailure(parseErr7) {
						return
					}
					chatLog.Error("set selected model failed", logging.Fields{"error": parseErr7, "model": parseModelID})
					parseSelectedModelCache.Invalidate()
					return
				}
				parsePublishSelectedModelCrossTab(parseModelID)
			}(parseNextModel3)
		}
	}

	parsePersistThinkingPreferences := func(isEnabled bool, parseEffort string) {
		parseResolvedEffort := parseNormalizeSelectedThinkingEffort(parseEffort)
		parseSelectedThinkingEnabledCache.Set(isEnabled)
		parseSelectedThinkingEffortCache.Set(parseResolvedEffort)
		if parseClient7 := parseChatClientRef.Get(); parseClient7 != nil {
			go func(isNextEnabled bool) {
				_, parseErr8 := parseClient7.SetSelectedThinkingEnabled(context.Background(), wrapperspb.Bool(isNextEnabled))
				if parseErr8 != nil {
					if handleAuthFailure != nil && handleAuthFailure(parseErr8) {
						return
					}
					chatLog.Error("set selected thinking enabled failed", logging.Fields{"error": parseErr8})
					parseSelectedThinkingEnabledCache.Invalidate()
				}
			}(isEnabled)
			go func(parseNextEffort2 string) {
				_, parseErr9 := parseClient7.SetSelectedThinkingEffort(context.Background(), wrapperspb.String(parseNextEffort2))
				if parseErr9 != nil {
					if handleAuthFailure != nil && handleAuthFailure(parseErr9) {
						return
					}
					chatLog.Error("set selected thinking effort failed", logging.Fields{"error": parseErr9})
					parseSelectedThinkingEffortCache.Invalidate()
				}
			}(parseResolvedEffort)
		}
	}

	parseRefreshModelCatalog := func() {
		parseClient8 := parseChatClientRef.Get()
		if parseClient8 == nil {
			chatLog.Warn("model catalog refresh skipped; grpc client unavailable", logging.Fields{
				"grpc_ready":         parseApp.Get().GRPCReady,
				"authenticated":      parseApp.Get().Authenticated,
				"cached_model_count": len(parseApp.Get().ModelOptions),
			})
			return
		}
		go func() {
			parseResp6, parseErr10 := parseClient8.ListModelOptions(context.Background(), &chatpb.ListModelOptionsRequest{})
			if parseErr10 != nil {
				if handleAuthFailure != nil && handleAuthFailure(parseErr10) {
					return
				}
				chatLog.Error("model catalog load failed", logging.Fields{
					"error":              parseErr10,
					"grpc_ready":         parseApp.Get().GRPCReady,
					"authenticated":      parseApp.Get().Authenticated,
					"cached_model_count": len(parseApp.Get().ModelOptions),
				})
				return
			}
			parseCatalog := parseModelCatalogFromResponse(parseResp6)
			parseModelCatalogCache.Set(parseCatalog)
			chatLog.Info("model catalog loaded", logging.Fields{
				"models":        len(parseCatalog.Models),
				"default_model": parseCatalog.DefaultModel,
			})
		}()
	}

	applyModelSelection := func(parseNextModel4 string) bool {
		parseCurrentState2 := parseApp.Get()
		parseResolvedModel := parseNormalizeSelectedModelID(parseNextModel4, parseCurrentState2.ModelOptions, parseCurrentState2.DefaultModel)
		if parseResolvedModel == "" {
			return false
		}
		if parseResolvedModel == parseCurrentState2.SelectedModel {
			return true
		}
		parseSelectedModelBootstrapComplete.Set(true)
		parseSelectedModelRecoveryComplete.Set(true)
		parseApp.Dispatch(appAction{Type: appActionSetSelectedModel, SelectedModel: parseResolvedModel})
		parsePersistSelectedModel(parseResolvedModel)
		if len(parseCurrentState2.Messages) > 0 {
			parseApp.Dispatch(appAction{
				Type: appActionUpdateMessages,
				UpdateMessages: func(parsePrev []message) []message {
					return append(append([]message{}, parsePrev...), message{Role: roleSwitch, Content: parseResolvedModel})
				},
			})
		}
		return true
	}

	ui.UseEffect(func() func() {
		if !parseApp.Get().GRPCReady {
			parseModelCatalogRefreshOnConnect.Set(false)
			return nil
		}
		if parseModelCatalogRefreshOnConnect.Get() {
			return nil
		}
		parseModelCatalogRefreshOnConnect.Set(true)
		parseModelCatalogCache.Invalidate()
		return nil
	}, parseApp.Get().GRPCReady)

	ui.UseEffect(func() func() {
		parseCurrentState3 := parseApp.Get()
		if !parseCurrentState3.GRPCReady || !parseCurrentState3.Authenticated {
			parseModelPreferenceSessionKey.Set("")
			parseSelectedModelBootstrapComplete.Set(false)
			parseSelectedModelRecoveryComplete.Set(false)
			parseThinkingPreferencesBootstrapComplete.Set(false)
			return nil
		}
		parseSessionKey := strings.TrimSpace(strings.ToLower(parseCurrentState3.SessionEmail))
		if parseSessionKey == "" {
			parseSessionKey = "__anonymous__"
		}
		if parseModelPreferenceSessionKey.Get() == parseSessionKey {
			return nil
		}
		parseModelPreferenceSessionKey.Set(parseSessionKey)
		parseSelectedModelBootstrapComplete.Set(false)
		parseSelectedModelRecoveryComplete.Set(false)
		parseThinkingPreferencesBootstrapComplete.Set(false)
		return nil
	}, parseApp.Get().GRPCReady, parseApp.Get().Authenticated, parseApp.Get().SessionEmail)

	ui.UseEffect(func() func() {
		parseCurrentState4 := parseApp.Get()
		if !parseCurrentState4.GRPCReady || !parseCurrentState4.Authenticated {
			parseSelectedModelChannelRef.Set(interop.CrossTabChannel{})
			return nil
		}
		parseChannel2, parseErr11 := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{
			Name: parseSelectedModelCrossTabChannelName(parseCurrentState4.SessionEmail),
		})
		if parseErr11 != nil {
			parseSelectedModelChannelRef.Set(interop.CrossTabChannel{})
			chatLog.Warn("open selected model cross-tab channel failed", logging.Fields{"error": parseErr11, "email": parseCurrentState4.SessionEmail})
			return nil
		}
		parseSelectedModelChannelRef.Set(parseChannel2)
		parseSubscription, parseErr11 := interop.SubscribeDecodedCrossTab(parseChannel2, func(parseMessage interop.DecodedCrossTabEnvelope[selectedModelCrossTabMessage], parseSubErr error) {
			if parseSubErr != nil {
				chatLog.Warn("selected model cross-tab subscribe failed", logging.Fields{"error": parseSubErr, "channel": parseChannel2.Name()})
				return
			}
			parseCurrentState5 := parseApp.Get()
			if !parseCurrentState5.GRPCReady || !parseCurrentState5.Authenticated || parseCurrentState5.Streaming {
				return
			}
			parseResolvedModel2 := parseNormalizeSelectedModelID(parseMessage.Payload.Model, parseCurrentState5.ModelOptions, parseCurrentState5.DefaultModel)
			if parseResolvedModel2 == "" || parseResolvedModel2 == parseCurrentState5.SelectedModel {
				return
			}
			parseSelectedModelCache.Set(parseResolvedModel2)
			parseSelectedModelBootstrapComplete.Set(true)
			parseSelectedModelRecoveryComplete.Set(true)
			parseApp.Dispatch(appAction{Type: appActionSetSelectedModel, SelectedModel: parseResolvedModel2})
		})
		if parseErr11 != nil {
			parseSelectedModelChannelRef.Set(interop.CrossTabChannel{})
			_ = parseChannel2.Close()
			chatLog.Warn("selected model cross-tab setup failed", logging.Fields{"error": parseErr11, "channel": parseChannel2.Name()})
			return nil
		}
		return func() {
			parseSelectedModelChannelRef.Set(interop.CrossTabChannel{})
			parseSubscription.Cancel()
			_ = parseChannel2.Close()
		}
	}, parseApp.Get().GRPCReady, parseApp.Get().Authenticated, parseApp.Get().SessionEmail)

	ui.UseEffect(func() func() {
		parseCurrentState6 := parseApp.Get()
		if !parseCurrentState6.GRPCReady || !parseCurrentState6.Authenticated {
			parseModelCatalogBootstrapRequested.Set(false)
			return nil
		}
		if len(parseCurrentState6.ModelOptions) > 0 {
			parseModelCatalogBootstrapRequested.Set(false)
			return nil
		}
		if parseChatClientRef.Get() == nil {
			return nil
		}
		if parseModelCatalogBootstrapRequested.Get() {
			return nil
		}
		parseModelCatalogBootstrapRequested.Set(true)
		parseRefreshModelCatalog()
		return nil
	}, parseApp.Get().GRPCReady, parseApp.Get().Authenticated, len(parseApp.Get().ModelOptions), parseModelCatalogCacheState.Loading)

	ui.UseEffect(func() func() {
		parseCurrentState7 := parseApp.Get()
		if !parseCurrentState7.GRPCReady || !parseModelCatalogCacheState.Ready {
			return nil
		}
		parseCatalog2 := parseModelCatalogCacheState.Value
		parseResolvedDefaultModel := parseNormalizeSelectedModelID(parseCatalog2.DefaultModel, parseCatalog2.Models, defaultModel)
		if !parseSameModelOptions(parseCurrentState7.ModelOptions, parseCatalog2.Models) || parseCurrentState7.DefaultModel != parseResolvedDefaultModel {
			parseApp.Dispatch(appAction{Type: appActionSetModelCatalog, ModelOptions: parseCatalog2.Models, DefaultModel: parseResolvedDefaultModel})
		}
		return nil
	}, parseApp.Get().GRPCReady, parseModelCatalogCacheState.Ready, parseModelCatalogCacheState.Value)

	ui.UseEffect(func() func() {
		parseCurrentState8 := parseApp.Get()
		if !shouldApplySelectedModelBootstrap(parseSelectedModelBootstrapComplete.Get(), parseCurrentState8, parseSelectedModelCacheState.Ready) {
			return nil
		}
		parseResolvedModel3 := parseNormalizeSelectedModelID(parseSelectedModelCacheState.Value, parseCurrentState8.ModelOptions, parseCurrentState8.DefaultModel)
		if parseResolvedModel3 != parseCurrentState8.SelectedModel {
			parseApp.Dispatch(appAction{Type: appActionSetSelectedModel, SelectedModel: parseResolvedModel3})
		}
		parseSelectedModelBootstrapComplete.Set(true)
		return nil
	}, parseApp.Get().GRPCReady, parseSelectedModelCacheState.Ready, parseSelectedModelCacheState.Value, parseApp.Get().ActiveConvID, len(parseApp.Get().Messages))

	ui.UseEffect(func() func() {
		parseCurrentState9 := parseApp.Get()
		if !shouldApplySelectedModelRecovery(parseSelectedModelRecoveryComplete.Get(), parseCurrentState9, parseSelectedModelCacheState.Ready) {
			return nil
		}
		parseRecoveredModel, parseUsedFallback := parseRecoverPersistedModelSelection(parseSelectedModelCacheState.Value, parseCurrentState9.ModelOptions)
		if parseRecoveredModel == "" || strings.TrimSpace(parseSelectedModelCacheState.Value) == parseRecoveredModel {
			parseSelectedModelRecoveryComplete.Set(true)
			return nil
		}
		if parseRecoveredModel != parseCurrentState9.SelectedModel {
			parseApp.Dispatch(appAction{Type: appActionSetSelectedModel, SelectedModel: parseRecoveredModel})
		}
		parsePersistSelectedModel(parseRecoveredModel)
		if parseUsedFallback {
			if !parseCurrentState9.SelectedThinkingEnabled {
				parseApp.Dispatch(appAction{Type: appActionSetSelectedThinkingEnabled, SelectedThinkingEnabled: true})
			}
			if parseCurrentState9.SelectedThinkingEffort != defaultThinkingEffort {
				parseApp.Dispatch(appAction{Type: appActionSetSelectedThinkingEffort, SelectedThinkingEffort: defaultThinkingEffort})
			}
			parseThinkingPreferencesBootstrapComplete.Set(true)
			parsePersistThinkingPreferences(true, defaultThinkingEffort)
		}
		parseSelectedModelRecoveryComplete.Set(true)
		return nil
	}, parseApp.Get().GRPCReady, parseSelectedModelCacheState.Ready, parseSelectedModelCacheState.Value, parseApp.Get().ActiveConvID, len(parseApp.Get().Messages), parseApp.Get().ModelOptions, parseApp.Get().SelectedModel, parseApp.Get().SelectedThinkingEnabled, parseApp.Get().SelectedThinkingEffort)

	ui.UseEffect(func() func() {
		parseCurrentState10 := parseApp.Get()
		if !parseCurrentState10.GRPCReady || !parseCurrentState10.Authenticated || !parseSelectedToneCacheState.Ready {
			return nil
		}
		parseResolvedTone := parseNormalizeSelectedToneID(parseSelectedToneCacheState.Value)
		if parseResolvedTone != parseCurrentState10.SelectedTone {
			parseApp.Dispatch(appAction{Type: appActionSetSelectedTone, SelectedTone: parseResolvedTone})
		}
		if !parseCurrentState10.ShowNameModal && parseResolvedTone != parseCurrentState10.ToneInput {
			parseApp.Dispatch(appAction{Type: appActionSetToneInput, ToneInput: parseResolvedTone})
		}
		return nil
	}, parseApp.Get().GRPCReady, parseSelectedToneCacheState.Ready, parseSelectedToneCacheState.Value, parseApp.Get().ShowNameModal)

	ui.UseEffect(func() func() {
		parseCurrentState11 := parseApp.Get()
		if !shouldApplyThinkingPreferencesBootstrap(parseThinkingPreferencesBootstrapComplete.Get(), parseCurrentState11, parseSelectedThinkingEnabledCacheState.Ready, parseSelectedThinkingEffortCacheState.Ready) {
			return nil
		}
		if parseSelectedThinkingEnabledCacheState.Value != parseCurrentState11.SelectedThinkingEnabled {
			parseApp.Dispatch(appAction{Type: appActionSetSelectedThinkingEnabled, SelectedThinkingEnabled: parseSelectedThinkingEnabledCacheState.Value})
		}
		parseResolvedThinkingEffort := parseNormalizeSelectedThinkingEffort(parseSelectedThinkingEffortCacheState.Value)
		if parseResolvedThinkingEffort != parseCurrentState11.SelectedThinkingEffort {
			parseApp.Dispatch(appAction{Type: appActionSetSelectedThinkingEffort, SelectedThinkingEffort: parseResolvedThinkingEffort})
		}
		parseThinkingPreferencesBootstrapComplete.Set(true)
		return nil
	}, parseApp.Get().GRPCReady, parseSelectedThinkingEnabledCacheState.Ready, parseSelectedThinkingEnabledCacheState.Value, parseSelectedThinkingEffortCacheState.Ready, parseSelectedThinkingEffortCacheState.Value, parseApp.Get().SelectedThinkingEnabled, parseApp.Get().SelectedThinkingEffort)

	setModel := ui.UseEvent(func(parseE ui.Event) {
		if parseApp.Get().Streaming {
			return
		}
		parseCurrentState12 := parseApp.Get()
		parseNewModel := parseNormalizeSelectedModelID(parseEventValueOrDataset(parseE, dataModel), parseCurrentState12.ModelOptions, parseCurrentState12.DefaultModel)
		if parseNewModel == parseApp.Get().SelectedModel {
			return
		}
		chatLog.Info("model set", logging.Fields{"model": parseNewModel})
		_ = applyModelSelection(parseNewModel)
	})

	setProvider := ui.UseEvent(func(parseE2 ui.Event) {
		if parseApp.Get().Streaming {
			return
		}
		parseCurrentState13 := parseApp.Get()
		parseProviderID := strings.TrimSpace(parseEventValueOrDataset(parseE2, dataProvider))
		if parseProviderID == "" {
			return
		}
		parseCurrentProvider := parseProviderForModel(parseCurrentState13.SelectedModel, parseCurrentState13.ModelOptions, parseCurrentState13.DefaultModel)
		if parseCurrentProvider.ID == parseProviderID {
			return
		}
		parseNextModel := parseDefaultModelForProvider(parseProviderID, parseCurrentState13.ModelOptions, parseCurrentState13.DefaultModel)
		if parseNextModel == "" || parseNextModel == parseCurrentState13.SelectedModel {
			return
		}
		_ = applyModelSelection(parseNextModel)
	})

	setThinkingMode := ui.UseEvent(func(parseE3 ui.Event) {
		if parseApp.Get().Streaming {
			return
		}
		parseCurrentState14 := parseApp.Get()
		if !parseModelSupportsThinking(parseCurrentState14.SelectedModel, parseCurrentState14.ModelOptions, parseCurrentState14.DefaultModel) {
			return
		}
		parseNextMode := strings.TrimSpace(strings.ToLower(parseEventValueOrDataset(parseE3, dataThinkingEffort)))
		isParseNextEnabled := parseNextMode != "off"
		parseNextEffort := parseCurrentState14.SelectedThinkingEffort
		if isParseNextEnabled {
			parseNextEffort = parseNormalizeSelectedThinkingEffort(parseNextMode)
		}
		if parseCurrentState14.SelectedThinkingEnabled == isParseNextEnabled && (!isParseNextEnabled || parseCurrentState14.SelectedThinkingEffort == parseNextEffort) {
			return
		}
		parseThinkingPreferencesBootstrapComplete.Set(true)
		parseApp.Dispatch(appAction{Type: appActionSetSelectedThinkingEnabled, SelectedThinkingEnabled: isParseNextEnabled})
		if isParseNextEnabled {
			parseApp.Dispatch(appAction{Type: appActionSetSelectedThinkingEffort, SelectedThinkingEffort: parseNextEffort})
		}
		parsePersistThinkingPreferences(isParseNextEnabled, parseNextEffort)
	})

	return modelPreferencesController{
		SelectedToneCache:            parseSelectedToneCache,
		SelectedThinkingEnabledCache: parseSelectedThinkingEnabledCache,
		SelectedThinkingEffortCache:  parseSelectedThinkingEffortCache,
		RefreshCatalog:               parseRefreshModelCatalog,
		SetProvider:                  setProvider,
		SetModel:                     setModel,
		SetThinkingMode:              setThinkingMode,
	}
}
