package app

import (
	"strings"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/v5/i18n"
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

// TestBuildCatalogLoaderAppliesLaunchTruthFilter verifies unsupported marketing claims are replaced during load.
func TestBuildCatalogLoaderAppliesLaunchTruthFilter(parseT *testing.T) {
	parseBundle := i18n.NewBundle(i18n.BundleOptions{DefaultLocale: "en", FallbackLocale: "en"})
	parseBundle.Register("en", i18n.Catalog{
		"marketing": {
			"hero.home.body":                {Text: "RelayDesk turns internal docs into instant answers."},
			"product.home.card.docqa.title": {Text: "Document Q&A"},
			"product.home.card.docqa.body":  {Text: "Paste a PDF and ask plain-English questions directly against it."},
			"product.capabilities.body":     {Text: "Chat assistant and document Q&A with team knowledge base."},
			"hero.pricing.body":             {Text: "Flat monthly fee per seat and enterprise options."},
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
	parseDocQaTitle := parseGetCatalogPayloadMessage(parsePayload.GetMessages(), "product.home.card.docqa.title")
	parseDocQaBody := parseGetCatalogPayloadMessage(parsePayload.GetMessages(), "product.home.card.docqa.body")
	parseCapabilityBody := parseGetCatalogPayloadMessage(parsePayload.GetMessages(), "product.capabilities.body")
	parseHeroPriceBody := parseGetCatalogPayloadMessage(parsePayload.GetMessages(), "hero.pricing.body")
	if parseDocQaTitle == "" || !strings.Contains(parseDocQaTitle, "planned for a later release") {
		parseT.Fatalf("expected launch-safe docqa title, got %q", parseDocQaTitle)
	}
	if parseDocQaBody == "" || !strings.Contains(parseDocQaBody, "planned for a later release") {
		parseT.Fatalf("expected launch-safe docqa body, got %q", parseDocQaBody)
	}
	if parseCapabilityBody == "" || !strings.Contains(parseCapabilityBody, "planned for a later release") {
		parseT.Fatalf("expected launch-safe capability body, got %q", parseCapabilityBody)
	}
	if parseHeroPriceBody == "" || !strings.Contains(parseHeroPriceBody, "planned for a later release") {
		parseT.Fatalf("expected launch-safe pricing body, got %q", parseHeroPriceBody)
	}
}

// TestBuildCatalogLaunchTruthFilterCanBeOptedIn verifies all claims can stay enabled via explicit config.
func TestBuildCatalogLaunchTruthFilterCanBeOptedIn(parseT *testing.T) {
	parseMessages := map[string]string{
		"product.home.card.docqa.title": "Document Q&A",
		"hero.home.body":                "RelayDesk turns internal docs into instant answers.",
	}
	parseConfig := parseBuildCatalogLaunchTruthConfig()
	parseConfig[parseCatalogLaunchTruthCapabilityDocsSearch] = true
	parseFiltered := parseApplyCatalogLaunchTruthFilterWithConfig(parseMessages, "en", parseConfig)
	if parseGet := parseFiltered["product.home.card.docqa.title"]; parseGet != "Document Q&A" {
		parseT.Fatalf("expected opt-in message preserved, got %q", parseGet)
	}
	if parseFiltered["hero.home.body"] != "RelayDesk turns internal docs into instant answers." {
		parseT.Fatalf("expected non-overridden hero body preserved, got %q", parseFiltered["hero.home.body"])
	}
}

// BenchmarkCatalogLaunchTruthFilterWithConfig measures launch-truth filter overhead on representative payloads.
func BenchmarkCatalogLaunchTruthFilterWithConfig(parseB *testing.B) {
	parseMessages := map[string]string{
		"hero.home.body":                    "RelayDesk turns internal docs into instant answers.",
		"product.home.card.docqa.body":      "Paste a PDF and ask plain-English questions directly against it.",
		"product.capabilities.body":         "Chat assistant and document Q&A with team knowledge base.",
		"product.home.card.enterprise.body": "SSO, role-based access, data retention controls, and SLA guarantees baked in.",
		"info.security.access.body":         "Role-based access for workspace members, SSO on Enterprise plans.",
		"product.home.body":                 "Use RelayDesk to answer repeat questions faster.",
	}
	parseConfig := parseBuildCatalogLaunchTruthConfig()
	parseB.ReportAllocs()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		parseFiltered := parseApplyCatalogLaunchTruthFilterWithConfig(parseMessages, "en", parseConfig)
		if len(parseFiltered) == 0 {
			parseB.Fatalf("unexpected empty launch-truth filter result")
		}
	}
}

// parseGetCatalogPayloadMessage returns one catalog message payload value by key.
func parseGetCatalogPayloadMessage(parseRows []*chatpb.CatalogMessagePayload, parseMessageKey string) string {
	for _, parseRow := range parseRows {
		if parseRow != nil && parseRow.GetMessageKey() == parseMessageKey {
			return parseRow.GetMessageValue()
		}
	}
	return ""
}

// parseHasCatalogPayloadMessage reports whether one catalog message payload slice contains one key.
func parseHasCatalogPayloadMessage(parseRows []*chatpb.CatalogMessagePayload, parseMessageKey string) bool {
	return parseGetCatalogPayloadMessage(parseRows, parseMessageKey) != ""
}
