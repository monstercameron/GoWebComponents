package sqlite

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"sync"
)

// Encryption-at-rest for the persisted database image.
//
// # Threat model — read this before relying on it
//
// Client-side encryption can only protect data AT REST, and only when the key is
// NOT reachable by the running page. NewPassphraseEncryptor derives the AES key
// from a user passphrase via PBKDF2 and never stores it, so:
//
//   - PROTECTED: someone who reads the IndexedDB/OPFS bytes off the disk, a backup,
//     a shared profile, or devtools sees only AES-256-GCM ciphertext. Without the
//     passphrase it is unreadable, and any tampering fails the GCM auth tag.
//   - NOT PROTECTED: code running in your origin while the app is unlocked (an XSS
//     payload, a malicious dependency) can ask the live Encryptor to decrypt, or
//     read the plaintext SQLite image out of wasm memory. No browser storage
//     defends against same-origin code — the key has to live somewhere reachable.
//
// In short: this is the password-manager model (safe while locked, on-disk), not a
// defense against a compromised page. Derive the passphrase from a real user
// secret; do not hard-code it or persist it. base64 alone (the default, no
// Encryptor) is NOT security.

const (
	encMagic          = "GWCE1"
	encSaltLen        = 16
	encNonceLen       = 12
	encKeyLen         = 32 // AES-256
	defaultPBKDF2Iter = 600_000
)

// ErrDecrypt is returned when an image cannot be opened — wrong passphrase, or
// corrupted/tampered ciphertext (the GCM auth tag did not verify).
var ErrDecrypt = errors.New("sqlite: decryption failed (wrong passphrase or corrupted/tampered data)")

// Encryptor seals and opens the persisted database image. Implement it to plug a
// different scheme (e.g. a stored non-extractable WebCrypto key); the default is
// NewPassphraseEncryptor.
type Encryptor interface {
	Seal(parsePlaintext []byte) ([]byte, error)
	Open(parseBlob []byte) ([]byte, error)
}

// NewPassphraseEncryptor returns an Encryptor that derives an AES-256-GCM key
// from passphrase via PBKDF2-HMAC-SHA256. iterations defaults to 600000 when <= 0.
// A random salt is minted once per instance and travels (with a fresh per-seal
// nonce) in the ciphertext header; the key itself is never persisted.
func NewPassphraseEncryptor(parsePassphrase string, parseIterations int) Encryptor {
	if parseIterations <= 0 {
		parseIterations = defaultPBKDF2Iter
	}
	return &passphraseEncryptor{
		passphrase: parsePassphrase,
		iterations: parseIterations,
		keys:       map[string][]byte{},
	}
}

// maxDerivedKeyCacheEntries bounds the salt->derived-key memo. Sealing reuses
// this instance's single salt, but decryption derives from each ciphertext's
// header salt, so decrypting data sealed by many replicas (each with its own
// random salt) would otherwise grow the cache — and retain sensitive key
// material — without bound. Past the cap the oldest entry is evicted; a later
// hit on an evicted salt simply re-runs PBKDF2.
const maxDerivedKeyCacheEntries = 128

type passphraseEncryptor struct {
	passphrase string
	iterations int

	mu       sync.Mutex
	salt     []byte            // minted once, reused for every Seal by this instance
	keys     map[string][]byte // salt -> derived key (PBKDF2 runs once per distinct salt)
	keyOrder []string          // insertion order, for bounded FIFO eviction
}

func (parseE *passphraseEncryptor) deriveKey(parseSalt []byte) ([]byte, error) {
	parseE.mu.Lock()
	defer parseE.mu.Unlock()
	parseSaltKey := string(parseSalt)
	if parseKey, parseOk := parseE.keys[parseSaltKey]; parseOk {
		return parseKey, nil
	}
	parseKey, parseErr := pbkdf2.Key(sha256.New, parseE.passphrase, parseSalt, parseE.iterations, encKeyLen)
	if parseErr != nil {
		return nil, parseErr
	}
	if len(parseE.keys) >= maxDerivedKeyCacheEntries && len(parseE.keyOrder) > 0 {
		parseOldest := parseE.keyOrder[0]
		parseE.keyOrder = parseE.keyOrder[1:]
		delete(parseE.keys, parseOldest)
	}
	parseE.keys[parseSaltKey] = parseKey
	parseE.keyOrder = append(parseE.keyOrder, parseSaltKey)
	return parseKey, nil
}

func (parseE *passphraseEncryptor) sealSalt() ([]byte, error) {
	parseE.mu.Lock()
	if parseE.salt == nil {
		parseSalt := make([]byte, encSaltLen)
		if _, parseErr := rand.Read(parseSalt); parseErr != nil {
			parseE.mu.Unlock()
			return nil, parseErr
		}
		parseE.salt = parseSalt
	}
	parseSalt := parseE.salt
	parseE.mu.Unlock()
	return parseSalt, nil
}

func (parseE *passphraseEncryptor) aead(parseSalt []byte) (cipher.AEAD, error) {
	parseKey, parseErr := parseE.deriveKey(parseSalt)
	if parseErr != nil {
		return nil, parseErr
	}
	parseBlock, parseErr := aes.NewCipher(parseKey)
	if parseErr != nil {
		return nil, parseErr
	}
	return cipher.NewGCM(parseBlock)
}

func (parseE *passphraseEncryptor) Seal(parsePlaintext []byte) ([]byte, error) {
	parseSalt, parseErr := parseE.sealSalt()
	if parseErr != nil {
		return nil, parseErr
	}
	parseAEAD, parseErr := parseE.aead(parseSalt)
	if parseErr != nil {
		return nil, parseErr
	}
	parseNonce := make([]byte, encNonceLen)
	if _, parseErr := rand.Read(parseNonce); parseErr != nil {
		return nil, parseErr
	}
	parseCipher := parseAEAD.Seal(nil, parseNonce, parsePlaintext, nil)

	parseOut := make([]byte, 0, len(encMagic)+encSaltLen+encNonceLen+len(parseCipher))
	parseOut = append(parseOut, encMagic...)
	parseOut = append(parseOut, parseSalt...)
	parseOut = append(parseOut, parseNonce...)
	parseOut = append(parseOut, parseCipher...)
	return parseOut, nil
}

func (parseE *passphraseEncryptor) Open(parseBlob []byte) ([]byte, error) {
	parseHeader := len(encMagic) + encSaltLen + encNonceLen
	if len(parseBlob) < parseHeader || string(parseBlob[:len(encMagic)]) != encMagic {
		return nil, ErrDecrypt
	}
	parseSalt := parseBlob[len(encMagic) : len(encMagic)+encSaltLen]
	parseNonce := parseBlob[len(encMagic)+encSaltLen : parseHeader]
	parseCipher := parseBlob[parseHeader:]

	parseAEAD, parseErr := parseE.aead(parseSalt)
	if parseErr != nil {
		return nil, parseErr
	}
	parsePlain, parseErr := parseAEAD.Open(nil, parseNonce, parseCipher, nil)
	if parseErr != nil {
		return nil, ErrDecrypt
	}
	return parsePlain, nil
}
