//go:build !js || !wasm
// +build !js !wasm

package interop

import (
	"context"
	"testing"
)

// TestCryptoNativeStubsReportUnavailable asserts that all WebCrypto bridge
// functions and EncryptedStore methods return CodeUnavailable on native builds,
// where the browser WebCrypto API is absent.
func TestCryptoNativeStubsReportUnavailable(parseT *testing.T) {
	parseCtx := context.Background()

	parseChecks := []struct {
		name string
		err  error
	}{
		{
			name: "GenerateAESKey",
			err: func() error {
				_, parseErr := GenerateAESKey(parseCtx)
				return parseErr
			}(),
		},
		{
			name: "Encrypt",
			err: func() error {
				_, _, parseErr := Encrypt(parseCtx, CryptoKey{}, []byte("plaintext"))
				return parseErr
			}(),
		},
		{
			name: "Decrypt",
			err: func() error {
				_, parseErr := Decrypt(parseCtx, CryptoKey{}, []byte("ciphertext"), []byte("iv"))
				return parseErr
			}(),
		},
		{
			name: "EncryptedStore.PutJSON",
			err: func() error {
				parseStore, _ := NewEncryptedStore(CryptoKey{})
				return parseStore.PutJSON(parseCtx, "key", map[string]any{"x": 1})
			}(),
		},
		{
			name: "EncryptedStore.GetJSON",
			err: func() error {
				parseStore, _ := NewEncryptedStore(CryptoKey{})
				var parseOut any
				_, parseErr := parseStore.GetJSON(parseCtx, "key", &parseOut)
				return parseErr
			}(),
		},
	}

	for _, parseCheck := range parseChecks {
		if !IsCode(parseCheck.err, CodeUnavailable) {
			parseT.Fatalf("%s: expected unavailable error, got %v", parseCheck.name, parseCheck.err)
		}
		parseInteropErr, parseOk := AsError(parseCheck.err)
		if !parseOk || parseInteropErr.Code != CodeUnavailable {
			parseT.Fatalf("%s: expected structured interop error with CodeUnavailable, got %#v ok=%t", parseCheck.name, parseInteropErr, parseOk)
		}
	}
}
