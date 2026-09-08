package desktop

import (
	"context"
	"encoding/base64"
	"errors"
	"strconv"

	"github.com/monstercameron/GoWebComponents/v5/interop"
	"github.com/monstercameron/GoWebComponents/v5/kvstate"
)

// StorageWireRecord is the JSON-safe storage record exchanged with a native host.
// Wide integers are decimal strings and bytes are base64 strings by contract.
type StorageWireRecord struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Version   string `json:"version"`
	UpdatedAt string `json:"updatedAt"`
}

// StorageCommit describes a successful durable mutation for event forwarding.
type StorageCommit struct {
	Key     string `json:"key"`
	Version string `json:"version"`
	Deleted bool   `json:"deleted"`
}

// StorageBackend adapts the named desktop storage methods to kvstate.
type StorageBackend struct{ parseClient Client }

// NewStorageBackend constructs a service-backed kvstate persistence backend.
func NewStorageBackend(parseClient Client) StorageBackend {
	return StorageBackend{parseClient: parseClient}
}

// Load retrieves one record and reports whether the key exists.
func (parseBackend StorageBackend) Load(parseContext context.Context, parseKey string) (kvstate.Record, bool, error) {
	parseWire, parseErr := Call[struct {
		Record *StorageWireRecord `json:"record"`
		Found  bool               `json:"found"`
	}](parseContext, parseBackend.parseClient, "storage.load", parseKey)
	if parseErr != nil {
		return kvstate.Record{Key: parseKey}, false, parseErr
	}
	if parseWire.Record == nil {
		if parseWire.Found {
			return kvstate.Record{}, false, getError("storage.load", interop.CodeDecode, errors.New("found storage record is missing"))
		}
		return kvstate.Record{Key: parseKey}, false, nil
	}
	if parseWire.Record.Key != parseKey {
		return kvstate.Record{}, false, getError("storage.load", interop.CodeDecode, errors.New("storage record key mismatch"))
	}
	parseRecord, parseErr := decodeStorageRecord(*parseWire.Record)
	if parseErr != nil {
		parseErr = getError("storage.load", interop.CodeDecode, parseErr)
	}
	return parseRecord, parseErr == nil && parseWire.Found, parseErr
}

// Save durably writes one record through the native service.
func (parseBackend StorageBackend) Save(parseContext context.Context, parseRecord kvstate.Record) error {
	_, parseErr := Call[struct{}](parseContext, parseBackend.parseClient, "storage.save", encodeStorageRecord(parseRecord))
	return parseErr
}

// Delete creates a durable tombstone for one key.
func (parseBackend StorageBackend) Delete(parseContext context.Context, parseKey string) error {
	_, parseErr := Call[struct{}](parseContext, parseBackend.parseClient, "storage.delete", parseKey)
	return parseErr
}

// Keys returns live keys in the native store.
func (parseBackend StorageBackend) Keys(parseContext context.Context) ([]string, error) {
	parseResult, parseErr := Call[[]string](parseContext, parseBackend.parseClient, "storage.keys")
	return parseResult, parseErr
}

// encodeStorageRecord preserves wide integers and opaque bytes in JSON.
func encodeStorageRecord(parseRecord kvstate.Record) StorageWireRecord {
	return StorageWireRecord{Key: parseRecord.Key, Value: base64.StdEncoding.EncodeToString(parseRecord.Value), Version: strconv.FormatInt(parseRecord.Version, 10), UpdatedAt: strconv.FormatInt(parseRecord.UpdatedAt, 10)}
}

// decodeStorageRecord rejects invalid version and timestamp semantics.
func decodeStorageRecord(parseWire StorageWireRecord) (kvstate.Record, error) {
	parseVersion, parseErr := strconv.ParseInt(parseWire.Version, 10, 64)
	if parseErr != nil || parseVersion < 1 {
		return kvstate.Record{}, errors.New("invalid storage version")
	}
	parseUpdated, parseErr := strconv.ParseInt(parseWire.UpdatedAt, 10, 64)
	if parseErr != nil || parseUpdated < 0 {
		return kvstate.Record{}, errors.New("invalid storage timestamp")
	}
	parseValue, parseErr := base64.StdEncoding.DecodeString(parseWire.Value)
	if parseErr != nil {
		return kvstate.Record{}, errors.New("invalid storage bytes")
	}
	return kvstate.Record{Key: parseWire.Key, Value: parseValue, Version: parseVersion, UpdatedAt: parseUpdated}, nil
}
