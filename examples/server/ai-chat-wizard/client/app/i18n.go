//go:build js && wasm

package app

import (
	"strings"

	catalog "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/client/catalog"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
)

type localeOption struct {
	ID          string
	NativeLabel string
}

var availableLocales = []localeOption{
	{ID: "en", NativeLabel: "English"},
	{ID: "es", NativeLabel: "Español"},
	{ID: "fr", NativeLabel: "Français"},
}

var chatWizardBundle = catalog.BuildBundle()

type chatWizardRouteProps struct {
	CurrentPath     string
	CurrentLocation string
}

// buildAppRouteProps captures the current browser location so shared-shell route changes force one rerender.
func buildAppRouteProps() chatWizardRouteProps {
	return chatWizardRouteProps{
		CurrentPath:     router.GetCurrentPath(),
		CurrentLocation: parseCurrentLocationPathSearch(),
	}
}

func parseChatWizardRoot(parseProps chatWizardRouteProps) ui.Node {
	parseLocale := i18n.UseLocale(i18n.LocaleOptions{
		InitialLocale:    "en",
		SupportedLocales: parseSupportedChatLocaleIDs(),
		FallbackLocale:   "en",
		PersistenceKey:   chatLocalePersistenceKey,
		DetectBrowser:    true,
	})
	// Observe the catalog version atom so this root re-renders when the server catalog
	// bootstrap merges into chatWizardBundle, propagating updated strings to all children.
	_ = state.UseAtom(chatCatalogVersionAtomKey, 0).Get()
	return i18n.Provider(i18n.ProviderProps{
		Locale: parseLocale,
		Bundle: chatWizardBundle,
		Child:  ui.CreateElement(ParseApp, parseProps),
	})
}

func parseSupportedChatLocaleIDs() []string {
	parseLocales := make([]string, 0, len(availableLocales))
	for _, parseOption := range availableLocales {
		parseLocales = append(parseLocales, parseOption.ID)
	}
	return parseLocales
}

func parseNormalizeChatLocaleID(parseLocale string) string {
	parseNormalized := i18n.NormalizeLocale(parseLocale)
	for _, parseOption := range availableLocales {
		if parseOption.ID == parseNormalized {
			return parseOption.ID
		}
	}
	return availableLocales[0].ID
}

func parseLocaleLabel(parseId string) string {
	for _, parseOption := range availableLocales {
		if parseOption.ID == parseId {
			return parseOption.NativeLabel
		}
	}
	return strings.TrimSpace(parseId)
}

func parseToneLabel(parseIntl i18n.Runtime, parseId string) string {
	return parseIntl.T(chatI18nNamespace, "tone."+parseId+".label")
}

// parseSystemPromptNavSummary returns a brief status label for the system-prompt
// settings nav item so it stays the same height as neighboring items.
func parseSystemPromptNavSummary(parseIntl i18n.Runtime, parsePromptInput string) string {
	if strings.TrimSpace(parsePromptInput) != "" {
		return parseIntl.T(chatI18nNamespace, "modal.systemPromptStatusCustom")
	}
	return parseIntl.T(chatI18nNamespace, "modal.systemPromptStatusDefault")
}

func parseToneDescription(parseIntl i18n.Runtime, parseId string) string {
	return parseIntl.T(chatI18nNamespace, "tone."+parseId+".desc")
}

func parseThinkingEffortLabel(parseIntl i18n.Runtime, parseId string) string {
	return parseIntl.T(chatI18nNamespace, "thinking."+parseId)
}

func parseThoughtHeadingLabel(parseIntl i18n.Runtime, parseHeading string) string {
	if strings.EqualFold(strings.TrimSpace(parseHeading), "thinking") {
		return parseIntl.T(chatI18nNamespace, "message.thinking")
	}
	return parseHeading
}
