package app

import "testing"

// TestBuildCatalogNamespacePayloadNormalizesContract verifies typed catalog payload fields normalize deterministically.
func TestBuildCatalogNamespacePayloadNormalizesContract(parseT *testing.T) {
	parsePayload := parseBuildCatalogNamespacePayload(
		" marketing.home ",
		"EN_us",
		"",
		"",
		parseCatalogSourceWrite{
			SourceLayer:   "",
			SourceID:      " client/app/i18n.go ",
			SourceVersion: "commit-123",
		},
		map[string]string{
			" hero.title ": " RelayDesk ",
			"hero.cta":     " Start ",
			"":             "ignored",
		},
	)
	if parsePayload.GetNamespace() != "marketing.home" {
		parseT.Fatalf("unexpected namespace: %q", parsePayload.GetNamespace())
	}
	if parsePayload.GetLocale() != "en-us" {
		parseT.Fatalf("unexpected locale: %q", parsePayload.GetLocale())
	}
	if parsePayload.GetFallbackLocale() != "en-us" {
		parseT.Fatalf("unexpected fallback locale: %q", parsePayload.GetFallbackLocale())
	}
	if parsePayload.GetVersion() != "v1" {
		parseT.Fatalf("unexpected version: %q", parsePayload.GetVersion())
	}
	if parsePayload.GetSource().GetSourceLayer() != "go_embed" {
		parseT.Fatalf("unexpected source layer: %q", parsePayload.GetSource().GetSourceLayer())
	}
	if parsePayload.GetSource().GetSourceId() != "client/app/i18n.go" {
		parseT.Fatalf("unexpected source id: %q", parsePayload.GetSource().GetSourceId())
	}
	if len(parsePayload.GetMessages()) != 2 {
		parseT.Fatalf("unexpected message count: %d", len(parsePayload.GetMessages()))
	}
	if parsePayload.GetMessages()[0].GetMessageKey() != "hero.cta" {
		parseT.Fatalf("expected sorted first key hero.cta, got %q", parsePayload.GetMessages()[0].GetMessageKey())
	}
	if parsePayload.GetMessages()[1].GetMessageKey() != "hero.title" {
		parseT.Fatalf("expected sorted second key hero.title, got %q", parsePayload.GetMessages()[1].GetMessageKey())
	}
	if parsePayload.GetMessages()[1].GetMessageValue() != "RelayDesk" {
		parseT.Fatalf("expected trimmed message value, got %q", parsePayload.GetMessages()[1].GetMessageValue())
	}
	if parsePayload.GetContentHash() == "" {
		parseT.Fatal("expected non-empty content hash")
	}
}

// TestBuildCatalogNamespacePayloadContentHashStable verifies catalog hashing is stable for equivalent unordered maps.
func TestBuildCatalogNamespacePayloadContentHashStable(parseT *testing.T) {
	parsePayloadA := parseBuildCatalogNamespacePayload(
		"marketing.pricing",
		"en",
		"en",
		"2026-03-28",
		parseCatalogSourceWrite{SourceLayer: "file"},
		map[string]string{
			"pricing.title": "Usage pricing",
			"pricing.cta":   "Start now",
		},
	)
	parsePayloadB := parseBuildCatalogNamespacePayload(
		"marketing.pricing",
		"en",
		"en",
		"2026-03-28",
		parseCatalogSourceWrite{SourceLayer: "file"},
		map[string]string{
			"pricing.cta":   "Start now",
			"pricing.title": "Usage pricing",
		},
	)
	if parsePayloadA.GetContentHash() != parsePayloadB.GetContentHash() {
		parseT.Fatalf("expected stable content hash for equivalent maps, got %q vs %q", parsePayloadA.GetContentHash(), parsePayloadB.GetContentHash())
	}
}
