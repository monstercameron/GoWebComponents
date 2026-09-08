//go:build js && wasm

package app

import (
	"context"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/v6/i18n"
	"github.com/monstercameron/GoWebComponents/v6/logging"
	"github.com/monstercameron/GoWebComponents/v6/state"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// chatCatalogVersionAtomKey is the shared atom key that signals when the server catalog has been merged.
// Components that re-render on catalog updates observe this atom.
const chatCatalogVersionAtomKey = "chat-wizard:catalog-version"

// parseUseCatalogVersionAtom returns the shared catalog-version atom.
// Observing its value in a component causes that component to re-render
// when server catalog data arrives and is merged into chatWizardBundle.
func parseUseCatalogVersionAtom() state.Atom[int] {
	return state.UseAtom(chatCatalogVersionAtomKey, 0)
}

// parseUseCatalogServerSynced returns true once the server catalog bootstrap has been
// merged into chatWizardBundle at least once for the current gRPC session.
// Surfaces with non-critical copy can gate skeleton or fallback states on this value,
// ensuring they render definitive server-managed copy rather than stale client defaults
// once the server has responded.
//
// When the catalog version is zero (initial value), the server has not yet responded
// and only client-embedded emergency fallback strings are active.
func parseUseCatalogServerSynced() bool {
	return state.UseAtom(chatCatalogVersionAtomKey, 0).Get() > 0
}

// parseUseCatalogServerSync fetches the server-owned catalog bootstrap payload once gRPC is ready
// and merges each namespace into chatWizardBundle so server-managed copy overwrites the emergency
// client fallback without changing overall UI composition.
//
// Safe to call from ParseApp: the merge happens in a goroutine and the atom increment is the
// only state mutation from async context, which is safe across the framework's scheduler.
func parseUseCatalogServerSync(
	parseCurrentState appState,
	parseChatClientRef ui.Ref[chatpb.ChatServiceClient],
	parseLocale string,
) {
	parseCatalogVersion := parseUseCatalogVersionAtom()
	parseFetchedKeyRef := ui.UseRef("")

	ui.UseEffect(func() func() {
		// Wait until gRPC is connected and a client is available.
		if !parseCurrentState.GRPCReady {
			return nil
		}
		parseClient := parseChatClientRef.Get()
		if parseClient == nil {
			return nil
		}
		parseResolvedLocale := strings.TrimSpace(i18n.NormalizeLocale(parseLocale))
		if parseResolvedLocale == "" {
			parseResolvedLocale = "en"
		}
		// Deduplicate fetches: only re-fetch when gRPC readiness or locale changes.
		parseFetchKey := parseResolvedLocale
		if parseFetchedKeyRef.Get() == parseFetchKey {
			return nil
		}
		go func(parseTargetLocale string, parsePrevVersion int) {
			parseResp, parseErr := parseClient.GetCatalogBootstrap(context.Background(), &chatpb.GetCatalogBootstrapRequest{
				Locale:     parseTargetLocale,
				Namespaces: []string{"chat", "marketing"},
			})
			if parseErr != nil {
				chatLog.Warn("server catalog bootstrap failed; using client fallback", logging.Fields{
					"locale": parseTargetLocale,
					"error":  parseErr,
				})
				return
			}
			if parseResp.GetIsNotModified() {
				return
			}
			parseMergeCatalogBootstrapResponse(parseResp)
			parseFetchedKeyRef.Set(parseFetchKey)
			parseNextVersion := parsePrevVersion + 1
			parseCatalogVersion.Set(parseNextVersion)
			chatLog.Info("server catalog merged into bundle", logging.Fields{
				"locale":     parseTargetLocale,
				"namespaces": len(parseResp.GetCatalogs()),
				"version":    parseResp.GetBundleVersion(),
			})
		}(parseResolvedLocale, parseCatalogVersion.Get())
		return nil
	}, parseCurrentState.GRPCReady, parseLocale)
}

// parseMergeCatalogBootstrapResponse registers all namespace payloads from a bootstrap response
// into chatWizardBundle, overwriting any prior client-fallback values for matching keys.
func parseMergeCatalogBootstrapResponse(parseResp *chatpb.GetCatalogBootstrapResponse) {
	if parseResp == nil {
		return
	}
	parseLocale := strings.TrimSpace(parseResp.GetLocale())
	if parseLocale == "" {
		parseLocale = "en"
	}
	for _, parsePayload := range parseResp.GetCatalogs() {
		parseMergeCatalogNamespacePayload(parseLocale, parsePayload)
	}
}

// parseMergeCatalogNamespacePayload registers one server namespace payload into chatWizardBundle.
// The fallback locale is also registered when different from the active locale so locale
// fallback chains resolve correctly from server content.
func parseMergeCatalogNamespacePayload(parseLocale string, parsePayload *chatpb.CatalogNamespacePayload) {
	if parsePayload == nil {
		return
	}
	parseNamespace := strings.TrimSpace(parsePayload.GetNamespace())
	if parseNamespace == "" {
		return
	}
	parseMessages := make(i18n.NamespaceCatalog, len(parsePayload.GetMessages()))
	for _, parseMsg := range parsePayload.GetMessages() {
		parseMsgKey := strings.TrimSpace(parseMsg.GetMessageKey())
		parseMsgVal := parseMsg.GetMessageValue()
		if parseMsgKey == "" {
			continue
		}
		parseMessages[parseMsgKey] = i18n.Message{Text: parseMsgVal}
	}
	if len(parseMessages) == 0 {
		return
	}
	chatWizardBundle.RegisterNamespace(parseLocale, parseNamespace, parseMessages)
	// Also register under the fallback locale so the bundle chain resolves correctly
	// when the requested locale is not the fallback.
	parseFallback := strings.TrimSpace(parsePayload.GetFallbackLocale())
	if parseFallback != "" && parseFallback != parseLocale {
		chatWizardBundle.RegisterNamespace(parseFallback, parseNamespace, parseMessages)
	}
}
