package runtime2

import "testing"

// TestBuildPatchStreamIdentityMatchesLegacyMarshalEncoding verifies the streaming identity path preserves the legacy JSON hash.
func TestBuildPatchStreamIdentityMatchesLegacyMarshalEncoding(parseTesting *testing.T) {
	parseTesting.Run("CanonicalDiff", func(parseTesting *testing.T) {
		parsePatchStream := parseBuildPatchStreamForTest(
			parseTesting,
			"region-a",
			1,
			2,
			2,
			map[string]any{
				"kind": "host-element",
				"tag":  "div",
				"props": map[string]any{
					"class":    "active",
					"data-old": "legacy",
				},
				"children": []any{
					map[string]any{"kind": "text", "text": "before"},
				},
			},
			map[string]any{
				"kind": "host-element",
				"tag":  "div",
				"props": map[string]any{
					"class":    "idle",
					"data-new": "fresh",
				},
				"children": []any{
					map[string]any{"kind": "text", "text": "after"},
				},
			},
		)
		handlePatchStreamIdentityMatch(parseTesting, parsePatchStream)
	})
	parseTesting.Run("ReplaceSubtree", func(parseTesting *testing.T) {
		parsePatchStream := parseBuildPatchStreamForTest(
			parseTesting,
			"region-a",
			1,
			2,
			2,
			map[string]any{
				"kind": "host-element",
				"tag":  "div",
				"children": []any{
					map[string]any{"kind": "text", "text": "before"},
				},
			},
			map[string]any{
				"kind": "host-element",
				"tag":  "section",
				"children": []any{
					map[string]any{"kind": "host-element", "tag": "span"},
				},
			},
		)
		handlePatchStreamIdentityMatch(parseTesting, parsePatchStream)
	})
	parseTesting.Run("NilOps", func(parseTesting *testing.T) {
		parsePatchStream := PatchStreamRaw{
			GetHeader: PatchStreamHeaderRaw{
				ProtocolVersion: PatchStreamProtocolVersion,
				RegionID:        "region-a",
				Epoch:           1,
				InputVersion:    2,
				PatchVersion:    2,
			},
		}
		handlePatchStreamIdentityMatch(parseTesting, parsePatchStream)
	})
}

// handlePatchStreamIdentityMatch compares the current patch identity against the legacy marshal-based digest.
func handlePatchStreamIdentityMatch(parseTesting *testing.T, parsePatchStream PatchStreamRaw) {
	parseTesting.Helper()
	parseCurrentIdentity, parseCurrentErr := BuildPatchStreamIdentity(parsePatchStream)
	if parseCurrentErr != nil {
		parseTesting.Fatalf("BuildPatchStreamIdentity returned error: %v", parseCurrentErr)
	}
	parseLegacyIdentity, parseLegacyErr := buildRuntime2LegacyPatchStreamIdentity(parsePatchStream)
	if parseLegacyErr != nil {
		parseTesting.Fatalf("buildRuntime2LegacyPatchStreamIdentity returned error: %v", parseLegacyErr)
	}
	if parseCurrentIdentity != parseLegacyIdentity {
		parseTesting.Fatalf("BuildPatchStreamIdentity mismatch: current=%q legacy=%q", parseCurrentIdentity, parseLegacyIdentity)
	}
	if parsePatchStream.GetPatchIdentity != "" && parsePatchStream.GetPatchIdentity != parseCurrentIdentity {
		parseTesting.Fatalf("patch stream identity = %q, want %q", parsePatchStream.GetPatchIdentity, parseCurrentIdentity)
	}
}
