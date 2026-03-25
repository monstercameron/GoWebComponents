package head

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/router"
)

func BenchmarkMergeDocumentMicro(b *testing.B) {
	base := Document{
		Metadata: router.Metadata{
			Title:       "Home",
			Description: "Base page",
		},
		Robots: "index,follow",
	}
	override := Document{
		Metadata: router.Metadata{
			Title:        "Profile",
			CanonicalURL: "https://example.com/profile",
		},
		Robots: "noindex,nofollow",
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Merge(base, override)
	}
}
