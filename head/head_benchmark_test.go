package head

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/router"
)

func BenchmarkRenderToString(b *testing.B) {
	b.ReportAllocs()
	document := Document{
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
				Value: map[string]interface{}{
					"@context": "https://schema.org",
					"@type":    "WebPage",
					"name":     "Benchmark",
				},
			},
		},
	}
	for b.Loop() {
		markup, err := RenderToString(document)
		if err != nil {
			b.Fatalf("RenderToString: %v", err)
		}
		if markup == "" {
			b.Fatal("RenderToString returned empty markup")
		}
	}
}

func BenchmarkRenderJSONLD(b *testing.B) {
	b.ReportAllocs()
	value := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "FAQPage",
		"mainEntity": []map[string]interface{}{
			{
				"@type": "Question",
				"name":  "What is this?",
				"acceptedAnswer": map[string]interface{}{
					"@type": "Answer",
					"text":  "A benchmark payload",
				},
			},
		},
	}
	for b.Loop() {
		script, err := RenderJSONLD(value, "faq")
		if err != nil {
			b.Fatalf("RenderJSONLD: %v", err)
		}
		if script == "" {
			b.Fatal("RenderJSONLD returned empty script")
		}
	}
}
