package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/desktop"
)

const (
	storageMaxKeyBytes   = 256
	storageMaxValueBytes = 1 << 20
	storageMaxKeys       = 4096
	storageMaxTotalBytes = 16 << 20
)

// StorageService owns one app-scoped, atomically committed JSON store.
type StorageService struct {
	parseMutex   sync.Mutex
	parsePath    string
	parseLock    *os.File
	parseRecords map[string]storageEntry
	parseCommit  func(desktop.StorageCommit)
}

type storageEntry struct {
	Key       string `json:"key"`
	Value     []byte `json:"value,omitempty"`
	Version   int64  `json:"version"`
	UpdatedAt int64  `json:"updatedAt"`
	Deleted   bool   `json:"deleted,omitempty"`
}

type storageFile struct {
	Records map[string]storageEntry `json:"records"`
}

// NewStorageService opens an app-owned store, using an explicit path in tests.
func NewStorageService(parsePath string, parseCommit func(desktop.StorageCommit)) (*StorageService, error) {
	if parsePath == "" {
		parseDir, parseErr := os.UserConfigDir()
		if parseErr != nil {
			return nil, fmt.Errorf("resolve user config directory: %w", parseErr)
		}
		parsePath = filepath.Join(parseDir, "GWCWailsCounter", "storage.json")
	}
	parsePath, parseErr := filepath.Abs(parsePath)
	if parseErr != nil {
		return nil, fmt.Errorf("resolve storage path: %w", parseErr)
	}
	if parseErr = os.MkdirAll(filepath.Dir(parsePath), 0o700); parseErr != nil {
		return nil, fmt.Errorf("create storage directory: %w", parseErr)
	}
	parseLockPath := parsePath + ".lock"
	parseLock, parseErr := acquireStorageLock(parseLockPath)
	if parseErr != nil {
		return nil, fmt.Errorf("storage already owned or lock unavailable: %w", parseErr)
	}
	parseService := &StorageService{parsePath: parsePath, parseLock: parseLock, parseRecords: map[string]storageEntry{}, parseCommit: parseCommit}
	if parseErr = parseService.loadFile(); parseErr != nil {
		_ = releaseStorageLock(parseLockPath, parseLock)
		return nil, parseErr
	}
	return parseService, nil
}

// ServiceShutdown releases the process ownership lock.
func (parseService *StorageService) ServiceShutdown() error { return parseService.closeStorage() }

// closeStorage flushes ownership resources and is safe to call repeatedly.
func (parseService *StorageService) closeStorage() error {
	parseService.parseMutex.Lock()
	defer parseService.parseMutex.Unlock()
	if parseService.parseLock == nil {
		return nil
	}
	parseErr := releaseStorageLock(parseService.parsePath+".lock", parseService.parseLock)
	parseService.parseLock = nil
	return parseErr
}

// Load reads a live record; tombstones intentionally return found=false.
func (parseService *StorageService) Load(parseContext context.Context, parseKey string) (struct {
	Record *desktop.StorageWireRecord `json:"record"`
	Found  bool                       `json:"found"`
}, error) {
	if parseErr := parseStorageContext(parseContext); parseErr != nil {
		return struct {
			Record *desktop.StorageWireRecord `json:"record"`
			Found  bool                       `json:"found"`
		}{}, parseErr
	}
	if parseErr := validateStorageKey(parseKey); parseErr != nil {
		return struct {
			Record *desktop.StorageWireRecord `json:"record"`
			Found  bool                       `json:"found"`
		}{}, parseErr
	}
	parseService.parseMutex.Lock()
	defer parseService.parseMutex.Unlock()
	if parseService.parseLock == nil {
		return struct {
			Record *desktop.StorageWireRecord `json:"record"`
			Found  bool                       `json:"found"`
		}{}, errors.New("storage is closed")
	}
	parseEntry, parseFound := parseService.parseRecords[parseKey]
	if !parseFound || parseEntry.Deleted {
		if parseFound {
			parseWire := encodeStorageEntry(parseEntry)
			return struct {
				Record *desktop.StorageWireRecord `json:"record"`
				Found  bool                       `json:"found"`
			}{Record: &parseWire, Found: false}, nil
		}
		return struct {
			Record *desktop.StorageWireRecord `json:"record"`
			Found  bool                       `json:"found"`
		}{Found: false}, nil
	}
	parseWire := encodeStorageEntry(parseEntry)
	return struct {
		Record *desktop.StorageWireRecord `json:"record"`
		Found  bool                       `json:"found"`
	}{Record: &parseWire, Found: true}, nil
}

// Save atomically commits a non-stale record and rejects equal versions.
func (parseService *StorageService) Save(parseContext context.Context, parseRecord desktop.StorageWireRecord) error {
	if parseErr := parseStorageContext(parseContext); parseErr != nil {
		return parseErr
	}
	parseEntry, parseErr := decodeStorageEntry(parseRecord)
	if parseErr != nil {
		return parseErr
	}
	if parseEntry.Deleted {
		return errors.New("deleted storage record is invalid")
	}
	return parseService.commit(parseContext, parseEntry)
}

// Delete atomically commits a tombstone newer than the current record.
func (parseService *StorageService) Delete(parseContext context.Context, parseKey string) error {
	if parseErr := parseStorageContext(parseContext); parseErr != nil {
		return parseErr
	}
	if parseErr := validateStorageKey(parseKey); parseErr != nil {
		return parseErr
	}
	parseService.parseMutex.Lock()
	parseCurrent := parseService.parseRecords[parseKey]
	parseService.parseMutex.Unlock()
	return parseService.commit(parseContext, storageEntry{Key: parseKey, Version: parseCurrent.Version + 1, UpdatedAt: time.Now().UnixMilli(), Deleted: true})
}

// Keys returns sorted live keys only.
func (parseService *StorageService) Keys(parseContext context.Context) ([]string, error) {
	if parseErr := parseStorageContext(parseContext); parseErr != nil {
		return nil, parseErr
	}
	parseService.parseMutex.Lock()
	defer parseService.parseMutex.Unlock()
	if parseService.parseLock == nil {
		return nil, errors.New("storage is closed")
	}
	parseKeys := make([]string, 0, len(parseService.parseRecords))
	for parseKey, parseEntry := range parseService.parseRecords {
		if !parseEntry.Deleted {
			parseKeys = append(parseKeys, parseKey)
		}
	}
	sort.Strings(parseKeys)
	return parseKeys, nil
}

// commit validates and durably replaces a record before notifying subscribers.
func (parseService *StorageService) commit(parseContext context.Context, parseEntry storageEntry) error {
	if parseEntry.Version < 1 {
		return errors.New("storage version must be positive")
	}
	if len(parseEntry.Key) == 0 || len(parseEntry.Key) > storageMaxKeyBytes || strings.ContainsAny(parseEntry.Key, "\\/\x00") {
		return errors.New("invalid storage key")
	}
	if len(parseEntry.Value) > storageMaxValueBytes {
		return errors.New("storage value exceeds limit")
	}
	parseService.parseMutex.Lock()
	if parseErr := parseStorageContext(parseContext); parseErr != nil {
		parseService.parseMutex.Unlock()
		return parseErr
	}
	if parseService.parseLock == nil {
		parseService.parseMutex.Unlock()
		return errors.New("storage is closed")
	}
	parseCurrent, parseExists := parseService.parseRecords[parseEntry.Key]
	if parseExists && parseEntry.Version <= parseCurrent.Version {
		parseService.parseMutex.Unlock()
		return errors.New("stale storage version")
	}
	if !parseExists && len(parseService.parseRecords) >= storageMaxKeys {
		parseService.parseMutex.Unlock()
		return errors.New("storage key limit exceeded")
	}
	parseNext := make(map[string]storageEntry, len(parseService.parseRecords)+1)
	for parseKey, parseValue := range parseService.parseRecords {
		parseNext[parseKey] = parseValue
	}
	parseNext[parseEntry.Key] = parseEntry
	if parseErr := validateStorageTotal(parseNext); parseErr != nil {
		parseService.parseMutex.Unlock()
		return parseErr
	}
	if parseErr := parseService.writeFile(parseNext); parseErr != nil {
		parseService.parseMutex.Unlock()
		return parseErr
	}
	parseService.parseRecords = parseNext
	parseCommit := parseService.parseCommit
	parseService.parseMutex.Unlock()
	if parseCommit != nil {
		parseCommit(desktop.StorageCommit{Key: parseEntry.Key, Version: fmt.Sprint(parseEntry.Version), Deleted: parseEntry.Deleted})
	}
	return nil
}

// loadFile bounds the actual read, including a file that grows after opening.
func (parseService *StorageService) loadFile() error {
	parseFile, parseErr := os.Open(parseService.parsePath)
	if errors.Is(parseErr, os.ErrNotExist) {
		return nil
	}
	if parseErr != nil {
		return fmt.Errorf("read storage: %w", parseErr)
	}
	defer parseFile.Close()
	parseBytes, parseErr := io.ReadAll(io.LimitReader(parseFile, storageMaxTotalBytes+1))
	if parseErr != nil {
		return fmt.Errorf("read storage: %w", parseErr)
	}
	if len(parseBytes) > storageMaxTotalBytes {
		return errors.New("storage file exceeds limit")
	}
	var parseData storageFile
	if parseErr = json.Unmarshal(parseBytes, &parseData); parseErr != nil {
		return fmt.Errorf("decode storage: %w", parseErr)
	}
	if parseData.Records == nil {
		parseData.Records = map[string]storageEntry{}
	}
	if parseErr = validateStorageTotal(parseData.Records); parseErr != nil {
		return parseErr
	}
	parseService.parseRecords = parseData.Records
	return nil
}

// writeFile rejects encoded quota overflow before replacing the readable store.
func (parseService *StorageService) writeFile(parseRecords map[string]storageEntry) error {
	parseBytes, parseErr := json.Marshal(storageFile{Records: parseRecords})
	if parseErr != nil {
		return parseErr
	}
	if len(parseBytes) > storageMaxTotalBytes {
		return errors.New("storage file exceeds limit")
	}
	parseTemp, parseErr := os.CreateTemp(filepath.Dir(parseService.parsePath), ".storage-*")
	if parseErr != nil {
		return parseErr
	}
	parseTempPath := parseTemp.Name()
	defer os.Remove(parseTempPath)
	if _, parseErr = parseTemp.Write(parseBytes); parseErr == nil {
		parseErr = parseTemp.Sync()
	}
	if parseCloseErr := parseTemp.Close(); parseErr == nil {
		parseErr = parseCloseErr
	}
	if parseErr != nil {
		return parseErr
	}
	return os.Rename(parseTempPath, parseService.parsePath)
}

// validateStorageTotal checks decoded resource bounds and persisted record shape.
func validateStorageTotal(parseRecords map[string]storageEntry) error {
	if len(parseRecords) > storageMaxKeys {
		return errors.New("storage key limit exceeded")
	}
	parseTotal := 0
	for parseKey, parseEntry := range parseRecords {
		if parseKey != parseEntry.Key || validateStorageKey(parseKey) != nil || parseEntry.Version < 1 || parseEntry.UpdatedAt < 0 || len(parseEntry.Value) > storageMaxValueBytes {
			return errors.New("invalid storage record")
		}
		parseTotal += len(parseKey) + len(parseEntry.Value)
		if parseTotal > storageMaxTotalBytes {
			return errors.New("storage total size exceeded")
		}
	}
	return nil
}

// validateStorageKey excludes empty, oversized, and path-like keys.
func validateStorageKey(parseKey string) error {
	if len(parseKey) == 0 || len(parseKey) > storageMaxKeyBytes || strings.ContainsAny(parseKey, "\\/\x00") {
		return errors.New("invalid storage key")
	}
	return nil
}

// parseStorageContext checks optional native call cancellation before work.
func parseStorageContext(parseContext context.Context) error {
	if parseContext == nil {
		return nil
	}
	select {
	case <-parseContext.Done():
		return parseContext.Err()
	default:
		return nil
	}
}

// encodeStorageEntry preserves integers and bytes across JSON transport.
func encodeStorageEntry(parseEntry storageEntry) desktop.StorageWireRecord {
	return desktop.StorageWireRecord{Key: parseEntry.Key, Value: encodeBytes(parseEntry.Value), Version: fmt.Sprint(parseEntry.Version), UpdatedAt: fmt.Sprint(parseEntry.UpdatedAt)}
}

// decodeStorageEntry validates wire shape before allocating decoded payloads.
func decodeStorageEntry(parseWire desktop.StorageWireRecord) (storageEntry, error) {
	if parseErr := validateStorageKey(parseWire.Key); parseErr != nil {
		return storageEntry{}, parseErr
	}
	parseVersion, parseErr := strconv.ParseInt(parseWire.Version, 10, 64)
	if parseErr != nil || parseVersion < 1 {
		return storageEntry{}, errors.New("invalid storage version")
	}
	parseUpdated, parseErr := strconv.ParseInt(parseWire.UpdatedAt, 10, 64)
	if parseErr != nil || parseUpdated < 0 {
		return storageEntry{}, errors.New("invalid storage timestamp")
	}
	if len(parseWire.Value) > base64.StdEncoding.EncodedLen(storageMaxValueBytes) {
		return storageEntry{}, errors.New("storage value exceeds limit")
	}
	parseValue, parseErr := decodeBytes(parseWire.Value)
	if parseErr != nil {
		return storageEntry{}, errors.New("invalid storage bytes")
	}
	if len(parseValue) > storageMaxValueBytes {
		return storageEntry{}, errors.New("storage value exceeds limit")
	}
	return storageEntry{Key: parseWire.Key, Value: parseValue, Version: parseVersion, UpdatedAt: parseUpdated}, nil
}

// encodeBytes serializes binary payloads without text interpretation.
func encodeBytes(parseValue []byte) string { return base64.StdEncoding.EncodeToString(parseValue) }

// decodeBytes restores the wire payload after size validation.
func decodeBytes(parseValue string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(parseValue)
}
