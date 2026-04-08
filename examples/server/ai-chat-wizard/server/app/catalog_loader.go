package app

import (
	"errors"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/server/catalog"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

var errCatalogNamespaceRequired = errors.New("catalog namespace is required")
var errCatalogBundleUnavailable = errors.New("catalog bundle unavailable")

type parseCatalogSource interface {
	parseLoadCatalogBundle() (*i18n.Bundle, parseCatalogSourceWrite, error)
}

type parseCatalogSourceGo struct{}

type parseCatalogLoader struct {
	parseSource  parseCatalogSource
	parseVersion string
}

// parseLoadCatalogBundle loads one Go-backed catalog bundle and source metadata.
func (parseSource parseCatalogSourceGo) parseLoadCatalogBundle() (*i18n.Bundle, parseCatalogSourceWrite, error) {
	return catalog.BuildBundle(), parseCatalogSourceWrite{
		SourceLayer:   "go",
		SourceID:      "server/catalog/bundle.go",
		SourceVersion: "v1",
	}, nil
}

// parseBuildCatalogLoader builds one catalog loader over one pluggable catalog source.
func parseBuildCatalogLoader(parseSource parseCatalogSource, parseVersion string) *parseCatalogLoader {
	if parseSource == nil {
		parseSource = parseCatalogSourceGo{}
	}
	return &parseCatalogLoader{
		parseSource:  parseSource,
		parseVersion: parseNormalizeCatalogVersion(parseVersion),
	}
}

// parseLoadCatalogNamespace loads one catalog namespace payload for one locale through one source abstraction.
func (parseLoader *parseCatalogLoader) parseLoadCatalogNamespace(parseNamespace string, parseLocale string) (*chatpb.CatalogNamespacePayload, error) {
	if parseLoader == nil || parseLoader.parseSource == nil {
		return nil, errCatalogBundleUnavailable
	}
	parseNamespace = strings.TrimSpace(parseNamespace)
	if parseNamespace == "" {
		return nil, errCatalogNamespaceRequired
	}
	parseBundle, parseSourceWrite, parseErr := parseLoader.parseSource.parseLoadCatalogBundle()
	if parseErr != nil {
		return nil, parseErr
	}
	if parseBundle == nil {
		return nil, errCatalogBundleUnavailable
	}
	parseLocale = parseNormalizeCatalogLocale(parseLocale)
	parseFallbackLocale := parseNormalizeCatalogFallbackLocale(parseBundle.FallbackLocale(), parseLocale)
	parseBootstrap := parseBundle.ToSSRBootstrap(i18n.SSRBootstrapOptions{
		Locale:            parseLocale,
		FallbackLocale:    parseFallbackLocale,
		IncludeLocales:    []string{parseLocale, parseFallbackLocale},
		IncludeNamespaces: []string{parseNamespace},
	})
	parseMessages := parseBuildCatalogNamespaceMessages(parseBootstrap.Messages, parseNamespace, parseLocale, parseFallbackLocale)
	parseMessages = parseApplyCatalogLaunchTruthFilter(parseMessages, parseLocale)
	return parseBuildCatalogNamespacePayload(parseNamespace, parseLocale, parseFallbackLocale, parseLoader.parseVersion, parseSourceWrite, parseMessages), nil
}

// parseBuildCatalogNamespaceMessages extracts one namespace message map from SSR bootstrap payload rows.
func parseBuildCatalogNamespaceMessages(parseMessageByLocale map[string]map[string]ui.SSRI18nMessage, parseNamespace string, parseLocale string, parseFallbackLocale string) map[string]string {
	parsePrefix := strings.TrimSpace(parseNamespace) + "."
	parseMessages := make(map[string]string)
	parseAppendMessages := func(parseCandidateLocale string) {
		parseLocaleRows, hasParseLocaleRows := parseMessageByLocale[parseCandidateLocale]
		if !hasParseLocaleRows {
			return
		}
		for parseCombinedKey, parseMessage := range parseLocaleRows {
			if !strings.HasPrefix(parseCombinedKey, parsePrefix) {
				continue
			}
			parseMessageKey := strings.TrimPrefix(parseCombinedKey, parsePrefix)
			if strings.TrimSpace(parseMessageKey) == "" {
				continue
			}
			parseMessages[parseMessageKey] = strings.TrimSpace(parseMessage.Text)
		}
	}
	parseAppendMessages(parseFallbackLocale)
	parseAppendMessages(parseLocale)
	return parseMessages
}
