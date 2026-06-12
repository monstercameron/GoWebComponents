//go:build !(js && wasm)
// +build !js !wasm

package interop

import "context"

// GenerateAESKey is a non-browser stub that always returns an unavailable error.
func GenerateAESKey(parseCtx context.Context) (CryptoKey, error) {
	_ = parseCtx
	return CryptoKey{}, unavailable("GenerateAESKey", "crypto.subtle.generateKey")
}

// Encrypt is a non-browser stub that always returns an unavailable error.
func Encrypt(parseCtx context.Context, parseKey CryptoKey, parsePlaintext []byte) ([]byte, []byte, error) {
	_ = parseCtx
	_ = parseKey
	_ = parsePlaintext
	return nil, nil, unavailable("Encrypt", "crypto.subtle.encrypt")
}

// Decrypt is a non-browser stub that always returns an unavailable error.
func Decrypt(parseCtx context.Context, parseKey CryptoKey, parseCiphertext []byte, parseIV []byte) ([]byte, error) {
	_ = parseCtx
	_ = parseKey
	_ = parseCiphertext
	_ = parseIV
	return nil, unavailable("Decrypt", "crypto.subtle.decrypt")
}

// NewEncryptedStore returns a zero EncryptedStore; all methods on it return
// unavailable errors.
func NewEncryptedStore(parseKey CryptoKey) (EncryptedStore, error) {
	_ = parseKey
	return EncryptedStore{}, nil
}

// PutJSON is a non-browser stub that always returns an unavailable error.
func (parseES EncryptedStore) PutJSON(parseCtx context.Context, parseStorageKey string, parseValue any) error {
	_ = parseCtx
	_ = parseStorageKey
	_ = parseValue
	return unavailable("EncryptedStore.PutJSON", "crypto.subtle.encrypt")
}

// GetJSON is a non-browser stub that always returns an unavailable error.
func (parseES EncryptedStore) GetJSON(parseCtx context.Context, parseStorageKey string, parseOut any) (bool, error) {
	_ = parseCtx
	_ = parseStorageKey
	_ = parseOut
	return false, unavailable("EncryptedStore.GetJSON", "crypto.subtle.decrypt")
}
