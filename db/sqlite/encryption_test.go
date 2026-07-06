package sqlite

import (
	"bytes"
	"strconv"
	"testing"
)

// TestDeriveKeyCacheIsBounded pins the #55 unbounded-growth fix: decrypting data
// sealed by many replicas (each with its own random salt) derives one key per
// distinct salt, so the salt->key memo must stay bounded rather than retaining a
// derived key (sensitive material) per salt forever.
func TestDeriveKeyCacheIsBounded(parseT *testing.T) {
	parseEnc, parseOk := NewPassphraseEncryptor("pw", 1).(*passphraseEncryptor)
	if !parseOk {
		parseT.Fatal("expected a *passphraseEncryptor")
	}

	for parseI := 0; parseI < maxDerivedKeyCacheEntries*3; parseI++ {
		if _, parseErr := parseEnc.deriveKey([]byte("salt-" + strconv.Itoa(parseI))); parseErr != nil {
			parseT.Fatalf("deriveKey: %v", parseErr)
		}
	}

	parseEnc.mu.Lock()
	parseCount := len(parseEnc.keys)
	parseOrder := len(parseEnc.keyOrder)
	parseEnc.mu.Unlock()

	if parseCount > maxDerivedKeyCacheEntries {
		parseT.Fatalf("derived-key cache is unbounded: %d entries, want <= %d", parseCount, maxDerivedKeyCacheEntries)
	}
	if parseOrder != parseCount {
		parseT.Fatalf("keyOrder (%d) and keys (%d) drifted out of sync", parseOrder, parseCount)
	}

	// A repeated salt still in the retained window is served from cache and does
	// not corrupt the derivation.
	parseSalt := []byte("salt-" + strconv.Itoa(maxDerivedKeyCacheEntries*3-1))
	parseFirst, _ := parseEnc.deriveKey(parseSalt)
	parseSecond, _ := parseEnc.deriveKey(parseSalt)
	if !bytes.Equal(parseFirst, parseSecond) {
		parseT.Fatal("repeated salt derived different keys")
	}
}

// Fast iteration count for tests (the default 600k PBKDF2 is intentionally slow).
func testEncryptor(parsePass string) Encryptor { return NewPassphraseEncryptor(parsePass, 1000) }

func TestSealOpenRoundTrip(t *testing.T) {
	parseEnc := testEncryptor("correct horse battery staple")
	parsePlain := []byte("the quick brown fox \x00\x01 binary image bytes")
	parseBlob, parseErr := parseEnc.Seal(parsePlain)
	if parseErr != nil {
		t.Fatalf("seal: %v", parseErr)
	}
	if bytes.Contains(parseBlob, parsePlain) {
		t.Fatal("ciphertext must not contain the plaintext")
	}
	parseBack, parseErr := parseEnc.Open(parseBlob)
	if parseErr != nil {
		t.Fatalf("open: %v", parseErr)
	}
	if !bytes.Equal(parseBack, parsePlain) {
		t.Fatalf("round-trip mismatch: %q", parseBack)
	}
}

func TestWrongPassphraseFails(t *testing.T) {
	parseBlob, _ := testEncryptor("right").Seal([]byte("secret"))
	if _, parseErr := testEncryptor("wrong").Open(parseBlob); parseErr != ErrDecrypt {
		t.Fatalf("wrong passphrase should give ErrDecrypt, got %v", parseErr)
	}
}

func TestTamperDetected(t *testing.T) {
	parseEnc := testEncryptor("k")
	parseBlob, _ := parseEnc.Seal([]byte("important data"))
	// Flip a byte in the ciphertext body (after the header).
	parseTampered := append([]byte(nil), parseBlob...)
	parseTampered[len(parseTampered)-1] ^= 0xFF
	if _, parseErr := parseEnc.Open(parseTampered); parseErr != ErrDecrypt {
		t.Fatalf("tampered ciphertext should fail GCM auth, got %v", parseErr)
	}
}

func TestDistinctCiphertextPerSeal(t *testing.T) {
	parseEnc := testEncryptor("k")
	parsePlain := []byte("same input")
	parseA, _ := parseEnc.Seal(parsePlain)
	parseB, _ := parseEnc.Seal(parsePlain)
	if bytes.Equal(parseA, parseB) {
		t.Fatal("two seals of the same data must differ (random nonce)")
	}
	// Both still decrypt.
	for _, parseBlob := range [][]byte{parseA, parseB} {
		if parseBack, parseErr := parseEnc.Open(parseBlob); parseErr != nil || !bytes.Equal(parseBack, parsePlain) {
			t.Fatalf("both ciphertexts must decrypt: %v", parseErr)
		}
	}
}

func TestOpenRejectsMalformed(t *testing.T) {
	parseEnc := testEncryptor("k")
	for _, parseBad := range [][]byte{nil, []byte("short"), []byte("XXXXX" + "0123456789abcdef" + "012345678901" + "ct")} {
		if _, parseErr := parseEnc.Open(parseBad); parseErr != ErrDecrypt {
			t.Fatalf("malformed blob should give ErrDecrypt, got %v for %q", parseErr, parseBad)
		}
	}
}

func TestEmptyPlaintextRoundTrips(t *testing.T) {
	parseEnc := testEncryptor("k")
	parseBlob, parseErr := parseEnc.Seal([]byte{})
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	parseBack, parseErr := parseEnc.Open(parseBlob)
	if parseErr != nil || len(parseBack) != 0 {
		t.Fatalf("empty round-trip failed: back=%v err=%v", parseBack, parseErr)
	}
}
