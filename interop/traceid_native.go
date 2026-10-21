//go:build !(js && wasm)

package interop

import "crypto/rand"

// fillRandomBytes fills parseBuf from the operating system CSPRNG on native builds.
func fillRandomBytes(parseOp string, parseBuf []byte) error {
	if _, parseErr := rand.Read(parseBuf); parseErr != nil {
		return wrapError(parseOp, "crypto/rand", CodeUnavailable, parseErr)
	}
	return nil
}
