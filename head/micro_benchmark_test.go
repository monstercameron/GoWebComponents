package head

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/router"
)

func BenchmarkMergeDocumentMicro(parseB *testing.B) {
	parseBase := Document{
		Metadata: router.Metadata{
			Title:       "Home",
			Description: "Base page",
		},
		Robots: "index,follow",
	}
	parseOverride := Document{
		Metadata: router.Metadata{
			Title:        "Profile",
			CanonicalURL: "https://example.com/profile",
		},
		Robots: "noindex,nofollow",
	}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = Merge(parseBase, parseOverride)
	}
}
