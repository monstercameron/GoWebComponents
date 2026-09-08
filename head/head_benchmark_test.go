package head

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/router"
)

func BenchmarkRenderToString(parseB *testing.B) {
	parseB.ReportAllocs()
	parseDocument := Document{
		Metadata: router.Metadata{
			Title:        "Benchmark",
			Description:  "Benchmark document rendering",
			CanonicalURL: "https://example.com/bench",
		},
		Robots: "index,follow",
		Social: SocialMetadata{
			Type:        "website",
			Title:       "Benchmark",
			Description: "Benchmark document rendering",
			ImageURL:    "https://example.com/bench.png",
			URL:         "https://example.com/bench",
			TwitterCard: "summary_large_image",
		},
		Alternates: []AlternateLink{
			{Href: "https://example.com/bench", HrefLang: "en"},
			{Href: "https://example.com/es/bench", HrefLang: "es"},
		},
		ResourceHints: []ResourceHint{
			{Rel: "preconnect", Href: "https://cdn.example.com"},
			{Rel: "dns-prefetch", Href: "https://api.example.com"},
		},
		JSONLD: []JSONLDBlock{
			{
				ID: "bench-jsonld",
				Value: map[string]any{
					"@context": "https://schema.org",
					"@type":    "WebPage",
					"name":     "Benchmark",
				},
			},
		},
	}
	for parseB.Loop() {
		parseMarkup, parseErr := RenderToString(parseDocument)
		if parseErr != nil {
			parseB.Fatalf("RenderToString: %v", parseErr)
		}
		if parseMarkup == "" {
			parseB.Fatal("RenderToString returned empty markup")
		}
	}
}

func BenchmarkRenderJSONLD(parseB *testing.B) {
	parseB.ReportAllocs()
	parseValue := map[string]any{
		"@context": "https://schema.org",
		"@type":    "FAQPage",
		"mainEntity": []map[string]any{
			{
				"@type": "Question",
				"name":  "What is this?",
				"acceptedAnswer": map[string]any{
					"@type": "Answer",
					"text":  "A benchmark payload",
				},
			},
		},
	}
	for parseB.Loop() {
		parseScript, parseErr := RenderJSONLD(parseValue, "faq")
		if parseErr != nil {
			parseB.Fatalf("RenderJSONLD: %v", parseErr)
		}
		if parseScript == "" {
			parseB.Fatal("RenderJSONLD returned empty script")
		}
	}
}
