package app

import (
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/i18n"
)

type parseCatalogSourceStub struct {
	parseBundle      *i18n.Bundle
	parseSourceWrite parseCatalogSourceWrite
	parseErr         error
}

// parseLoadCatalogBundle returns one stub bundle and source metadata payload.
func (parseSource parseCatalogSourceStub) parseLoadCatalogBundle() (*i18n.Bundle, parseCatalogSourceWrite, error) {
	return parseSource.parseBundle, parseSource.parseSourceWrite, parseSource.parseErr
}

// TestBuildCatalogLoaderGoBackedNamespace verifies Go-backed catalog loading returns typed namespace payloads.
func TestBuildCatalogLoaderGoBackedNamespace(parseT *testing.T) {
	parseLoader := parseBuildCatalogLoader(nil, "2026-03-28")
	parsePayload, parseErr := parseLoader.parseLoadCatalogNamespace("chat", "en")
	if parseErr != nil {
		parseT.Fatalf("parseLoadCatalogNamespace: %v", parseErr)
	}
	if parsePayload.GetNamespace() != "chat" || parsePayload.GetLocale() != "en" {
		parseT.Fatalf("unexpected namespace payload scope: %+v", parsePayload)
	}
	if parsePayload.GetSource().GetSourceLayer() != "go" {
		parseT.Fatalf("expected go source layer, got %+v", parsePayload.GetSource())
	}
	if len(parsePayload.GetMessages()) == 0 {
		parseT.Fatal("expected chat namespace messages from Go-backed source")
	}
	if !parseHasCatalogPayloadMessage(parsePayload.GetMessages(), "auth.signIn") {
		parseT.Fatalf("expected auth.signIn message key in payload, rows=%+v", parsePayload.GetMessages())
	}
}

// TestBuildCatalogLoaderRequiresNamespace verifies empty namespace requests fail closed.
func TestBuildCatalogLoaderRequiresNamespace(parseT *testing.T) {
	parseLoader := parseBuildCatalogLoader(nil, "v1")
	if _, parseErr := parseLoader.parseLoadCatalogNamespace("  ", "en"); parseErr != errCatalogNamespaceRequired {
		parseT.Fatalf("expected errCatalogNamespaceRequired, got %v", parseErr)
	}
}

// TestBuildCatalogLoaderSupportsSourceSwaps verifies source-loader indirection supports non-Go backed sources.
func TestBuildCatalogLoaderSupportsSourceSwaps(parseT *testing.T) {
	parseBundle := i18n.NewBundle(i18n.BundleOptions{DefaultLocale: "en", FallbackLocale: "en"})
	parseBundle.Register("en", i18n.Catalog{
		"marketing": {
			"hero.title": {Text: "Usage pricing"},
		},
	})
	parseLoader := parseBuildCatalogLoader(parseCatalogSourceStub{
		parseBundle: parseBundle,
		parseSourceWrite: parseCatalogSourceWrite{
			SourceLayer: "file",
			SourceID:    "test/catalog/en.marketing.json",
		},
	}, "2026.03.28")
	parsePayload, parseErr := parseLoader.parseLoadCatalogNamespace("marketing", "en")
	if parseErr != nil {
		parseT.Fatalf("parseLoadCatalogNamespace marketing: %v", parseErr)
	}
	if parsePayload.GetSource().GetSourceLayer() != "file" {
		parseT.Fatalf("expected file source layer, got %+v", parsePayload.GetSource())
	}
	if !parseHasCatalogPayloadMessage(parsePayload.GetMessages(), "hero.title") {
		parseT.Fatalf("expected hero.title in source-swapped payload, rows=%+v", parsePayload.GetMessages())
	}
}

// parseHasCatalogPayloadMessage reports whether one catalog message payload slice contains one key.
func parseHasCatalogPayloadMessage(parseRows []*chatpb.CatalogMessagePayload, parseMessageKey string) bool {
	for _, parseRow := range parseRows {
		if parseRow != nil && parseRow.GetMessageKey() == parseMessageKey {
			return true
		}
	}
	return false
}
