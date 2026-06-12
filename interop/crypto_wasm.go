//go:build js && wasm

package interop

import (
	"context"
	"encoding/json"
	"errors"
	"syscall/js"
)

// GenerateAESKey asks the browser's WebCrypto API to create a fresh
// non-extractable AES-GCM-256 key and returns an opaque handle to it.
func GenerateAESKey(parseCtx context.Context) (CryptoKey, error) {
	parseCrypto, parseErr := globalPath("GenerateAESKey", "crypto", "subtle")
	if parseErr != nil {
		return CryptoKey{}, parseErr
	}
	parseAlgoObj := js.Global().Get("Object").New()
	parseAlgoObj.Set("name", "AES-GCM")
	parseAlgoObj.Set("length", 256)

	parseUsages := js.Global().Get("Array").New(2)
	parseUsages.SetIndex(0, "encrypt")
	parseUsages.SetIndex(1, "decrypt")

	parseSubtle := Value{raw: parseCrypto}
	parseResult, parseErr2 := parseSubtle.Call("generateKey", parseAlgoObj, false, parseUsages)
	if parseErr2 != nil {
		return CryptoKey{}, parseErr2
	}
	parseRawResult, parseOk := parseResult.rawValue()
	if !parseOk {
		return CryptoKey{}, unavailable("GenerateAESKey", "crypto.subtle.generateKey")
	}
	parseResolved, parseErr3 := awaitValue(parseCtx, "GenerateAESKey", "crypto.subtle.generateKey", parseRawResult)
	if parseErr3 != nil {
		return CryptoKey{}, parseErr3
	}
	return CryptoKey{raw: parseResolved}, nil
}

// Encrypt encrypts parsePlaintext with parseKey using AES-GCM. It generates a
// fresh 12-byte IV via crypto.getRandomValues and returns both the ciphertext
// and the IV; callers must store the IV alongside the ciphertext to decrypt later.
func Encrypt(parseCtx context.Context, parseKey CryptoKey, parsePlaintext []byte) (parseCiphertext []byte, parseIV []byte, parseErr error) {
	parseKeyVal, parseOk := parseKey.raw.(js.Value)
	if !parseOk || parseKeyVal.IsNull() || parseKeyVal.IsUndefined() {
		return nil, nil, unavailable("Encrypt", "crypto.subtle.encrypt")
	}

	// Build a 12-byte IV using crypto.getRandomValues.
	parseIVArray := js.Global().Get("Uint8Array").New(12)
	parseCryptoGlobal, parseErr2 := globalProperty("Encrypt", "crypto")
	if parseErr2 != nil {
		return nil, nil, parseErr2
	}
	parseCryptoGlobal.Call("getRandomValues", parseIVArray)
	parseIV = make([]byte, 12)
	js.CopyBytesToGo(parseIV, parseIVArray)

	// Build the algorithm descriptor.
	parseAlgoObj := js.Global().Get("Object").New()
	parseAlgoObj.Set("name", "AES-GCM")
	parseAlgoObj.Set("iv", parseIVArray)

	// Copy plaintext into a JS Uint8Array.
	parsePlaintextJS := js.Global().Get("Uint8Array").New(len(parsePlaintext))
	js.CopyBytesToJS(parsePlaintextJS, parsePlaintext)

	parseSubtle, parseErr3 := globalPath("Encrypt", "crypto", "subtle")
	if parseErr3 != nil {
		return nil, nil, parseErr3
	}
	parseSubtleWrapped := Value{raw: parseSubtle}
	parseResult, parseErr4 := parseSubtleWrapped.Call("encrypt", parseAlgoObj, parseKeyVal, parsePlaintextJS)
	if parseErr4 != nil {
		return nil, nil, parseErr4
	}
	parseRawResult, parseOk2 := parseResult.rawValue()
	if !parseOk2 {
		return nil, nil, unavailable("Encrypt", "crypto.subtle.encrypt")
	}
	parseResolved, parseErr5 := awaitValue(parseCtx, "Encrypt", "crypto.subtle.encrypt", parseRawResult)
	if parseErr5 != nil {
		return nil, nil, parseErr5
	}

	// Copy the ArrayBuffer result into a Go []byte.
	parseView := js.Global().Get("Uint8Array").New(parseResolved)
	parseCiphertext = make([]byte, parseView.Length())
	js.CopyBytesToGo(parseCiphertext, parseView)
	return parseCiphertext, parseIV, nil
}

// Decrypt decrypts parseCiphertext with parseKey and parseIV using AES-GCM.
// A tampered ciphertext causes the browser's subtle.decrypt to reject — that
// rejection surfaces as an error; partial or garbage plaintext is never returned.
func Decrypt(parseCtx context.Context, parseKey CryptoKey, parseCiphertext []byte, parseIV []byte) ([]byte, error) {
	parseKeyVal, parseOk := parseKey.raw.(js.Value)
	if !parseOk || parseKeyVal.IsNull() || parseKeyVal.IsUndefined() {
		return nil, unavailable("Decrypt", "crypto.subtle.decrypt")
	}

	parseIVArray := js.Global().Get("Uint8Array").New(len(parseIV))
	js.CopyBytesToJS(parseIVArray, parseIV)

	parseAlgoObj := js.Global().Get("Object").New()
	parseAlgoObj.Set("name", "AES-GCM")
	parseAlgoObj.Set("iv", parseIVArray)

	parseCTArray := js.Global().Get("Uint8Array").New(len(parseCiphertext))
	js.CopyBytesToJS(parseCTArray, parseCiphertext)

	parseSubtle, parseErr := globalPath("Decrypt", "crypto", "subtle")
	if parseErr != nil {
		return nil, parseErr
	}
	parseSubtleWrapped := Value{raw: parseSubtle}
	parseResult, parseErr2 := parseSubtleWrapped.Call("decrypt", parseAlgoObj, parseKeyVal, parseCTArray)
	if parseErr2 != nil {
		return nil, parseErr2
	}
	parseRawResult, parseOk2 := parseResult.rawValue()
	if !parseOk2 {
		return nil, unavailable("Decrypt", "crypto.subtle.decrypt")
	}
	parseResolved, parseErr3 := awaitValue(parseCtx, "Decrypt", "crypto.subtle.decrypt", parseRawResult)
	if parseErr3 != nil {
		// The promise rejection from a tampered ciphertext is surfaced here.
		return nil, wrapError("Decrypt", "crypto.subtle.decrypt", CodeDecode,
			errors.New("authentication tag mismatch or decryption failure: "+parseErr3.Error()))
	}

	parseView := js.Global().Get("Uint8Array").New(parseResolved)
	parsePlaintext := make([]byte, parseView.Length())
	js.CopyBytesToGo(parsePlaintext, parseView)
	return parsePlaintext, nil
}

// NewEncryptedStore returns an EncryptedStore that uses parseKey for
// encryption and browser LocalStorage as the backing store.
func NewEncryptedStore(parseKey CryptoKey) (EncryptedStore, error) {
	parseStorage, parseErr := GetLocalStorage()
	if parseErr != nil {
		return EncryptedStore{}, parseErr
	}
	return EncryptedStore{key: parseKey, storage: parseStorage}, nil
}

// PutJSON marshals parseValue as JSON, encrypts it, and writes the
// base64-encoded iv:ciphertext envelope to LocalStorage under parseStorageKey.
func (parseES EncryptedStore) PutJSON(parseCtx context.Context, parseStorageKey string, parseValue any) error {
	parseJSON, parseErr := json.Marshal(parseValue)
	if parseErr != nil {
		return wrapError("EncryptedStore.PutJSON", parseStorageKey, CodeEncode, parseErr)
	}
	parseCT, parseIV, parseErr2 := Encrypt(parseCtx, parseES.key, parseJSON)
	if parseErr2 != nil {
		return parseErr2
	}
	parseEnvelope := encodeEnvelope(parseIV, parseCT)
	return parseES.storage.SetItem(parseStorageKey, parseEnvelope)
}

// GetJSON reads the envelope from LocalStorage under parseStorageKey, decrypts
// it, and JSON-unmarshals the plaintext into parseOut. It returns (false, nil)
// when the key is absent. A tampered or corrupted stored value always returns
// a non-nil error; plaintext is never returned in that case.
func (parseES EncryptedStore) GetJSON(parseCtx context.Context, parseStorageKey string, parseOut any) (bool, error) {
	parseEnvelope, parseFound, parseErr := parseES.storage.GetItem(parseStorageKey)
	if parseErr != nil {
		return false, parseErr
	}
	if !parseFound {
		return false, nil
	}
	parseIV, parseCT, parseErr2 := decodeEnvelope(parseEnvelope)
	if parseErr2 != nil {
		return false, parseErr2
	}
	parsePlaintext, parseErr3 := Decrypt(parseCtx, parseES.key, parseCT, parseIV)
	if parseErr3 != nil {
		return false, parseErr3
	}
	if parseErr4 := json.Unmarshal(parsePlaintext, parseOut); parseErr4 != nil {
		return false, wrapError("EncryptedStore.GetJSON", parseStorageKey, CodeDecode, parseErr4)
	}
	return true, nil
}
