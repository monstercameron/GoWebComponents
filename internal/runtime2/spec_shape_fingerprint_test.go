package runtime2

import "testing"

// TestBuildSerializablePropsFlatShapeFingerprintStableAcrossKeyOrder verifies map key ordering does not affect flat-shape fingerprints.
func TestBuildSerializablePropsFlatShapeFingerprintStableAcrossKeyOrder(parseT *testing.T) {
	getKeyCountLeft, getKeyHashLeft, getTypeHashLeft, getScratch, getScratchTypes, hasLeft := buildSerializablePropsFlatShapeFingerprintWithTypeScratch(
		map[string]any{
			"title": "Orders",
			"count": 5,
		},
		nil,
		nil,
	)
	if !hasLeft {
		parseT.Fatal("expected left flat props fingerprint")
	}
	getKeyCountRight, getKeyHashRight, getTypeHashRight, _, _, hasRight := buildSerializablePropsFlatShapeFingerprintWithTypeScratch(
		map[string]any{
			"count": 9,
			"title": "Invoices",
		},
		getScratch[:0],
		getScratchTypes[:0],
	)
	if !hasRight {
		parseT.Fatal("expected right flat props fingerprint")
	}
	if getKeyCountLeft != getKeyCountRight {
		parseT.Fatalf("expected equal key counts, got left=%d right=%d", getKeyCountLeft, getKeyCountRight)
	}
	if getKeyHashLeft != getKeyHashRight {
		parseT.Fatalf("expected equal key hashes, got left=%d right=%d", getKeyHashLeft, getKeyHashRight)
	}
	if getTypeHashLeft != getTypeHashRight {
		parseT.Fatalf("expected equal type hashes, got left=%d right=%d", getTypeHashLeft, getTypeHashRight)
	}
}

// TestBuildSerializablePropsFlatShapeFingerprintDetectsTypeChanges verifies type-hash changes when flat map value types change.
func TestBuildSerializablePropsFlatShapeFingerprintDetectsTypeChanges(parseT *testing.T) {
	getKeyCountLeft, getKeyHashLeft, getTypeHashLeft, getScratch, getScratchTypes, hasLeft := buildSerializablePropsFlatShapeFingerprintWithTypeScratch(
		map[string]any{
			"title": "Orders",
			"count": 5,
		},
		nil,
		nil,
	)
	if !hasLeft {
		parseT.Fatal("expected left flat props fingerprint")
	}
	getKeyCountRight, getKeyHashRight, getTypeHashRight, _, _, hasRight := buildSerializablePropsFlatShapeFingerprintWithTypeScratch(
		map[string]any{
			"title": "Orders",
			"count": "5",
		},
		getScratch[:0],
		getScratchTypes[:0],
	)
	if !hasRight {
		parseT.Fatal("expected right flat props fingerprint")
	}
	if getKeyCountLeft != getKeyCountRight {
		parseT.Fatalf("expected equal key counts, got left=%d right=%d", getKeyCountLeft, getKeyCountRight)
	}
	if getKeyHashLeft != getKeyHashRight {
		parseT.Fatalf("expected equal key hashes, got left=%d right=%d", getKeyHashLeft, getKeyHashRight)
	}
	if getTypeHashLeft == getTypeHashRight {
		parseT.Fatalf("expected distinct type hashes, got left=%d right=%d", getTypeHashLeft, getTypeHashRight)
	}
}

// TestBuildSerializablePropsFlatShapeFingerprintRejectsNestedContainers verifies nested container values stay on the full validation path.
func TestBuildSerializablePropsFlatShapeFingerprintRejectsNestedContainers(parseT *testing.T) {
	_, _, _, _, _, hasFingerprint := buildSerializablePropsFlatShapeFingerprintWithTypeScratch(
		map[string]any{
			"title": "Orders",
			"meta": map[string]any{
				"count": 5,
			},
		},
		nil,
		nil,
	)
	if hasFingerprint {
		parseT.Fatal("expected nested containers to bypass flat props fingerprinting")
	}
}

// TestBuildSerializablePropsFlatShapeFingerprintAcceptsSingleKeyMaps verifies one-key flat maps are fingerprintable for no-op validation skips.
func TestBuildSerializablePropsFlatShapeFingerprintAcceptsSingleKeyMaps(parseT *testing.T) {
	_, _, _, _, _, hasFingerprint := buildSerializablePropsFlatShapeFingerprintWithTypeScratch(
		map[string]any{
			"tick": 1,
		},
		nil,
		nil,
	)
	if !hasFingerprint {
		parseT.Fatal("expected one-key flat maps to produce props fingerprint")
	}
}
