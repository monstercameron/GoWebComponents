package desktop

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/interop"
	"github.com/monstercameron/GoWebComponents/v6/kvstate"
)

type storageTransport struct {
	parseRecord  StorageWireRecord
	parseStarted string
	parseData    json.RawMessage
}

func (parseTransport *storageTransport) Capabilities() (Capabilities, error) {
	return Capabilities{Protocol: ProtocolVersion, Methods: []string{"storage.load", "storage.save", "storage.delete", "storage.keys"}}, nil
}
func (parseTransport *storageTransport) Start(parseMethod string, parseArgs json.RawMessage) (string, error) {
	parseTransport.parseStarted = parseMethod
	return "1", nil
}
func (parseTransport *storageTransport) Poll(parseID string) (Reply, error) {
	if parseTransport.parseData != nil {
		return Reply{Done: true, Data: parseTransport.parseData}, nil
	}
	if parseTransport.parseStarted == "storage.load" {
		parseData, _ := json.Marshal(struct {
			Record *StorageWireRecord `json:"record"`
			Found  bool               `json:"found"`
		}{Record: &parseTransport.parseRecord, Found: true})
		return Reply{Done: true, Data: parseData}, nil
	}
	return Reply{Done: true, Data: json.RawMessage("null")}, nil
}

// TestStorageBackendRejectsMalformedRecords keeps malformed native payloads observable.
func TestStorageBackendRejectsMalformedRecords(parseTest *testing.T) {
	for _, parsePayload := range []string{
		`{"found":true,"record":null}`,
		`{"found":true,"record":{"key":"other","value":"","version":"1","updatedAt":"0"}}`,
		`{"found":true,"record":{"key":"a","value":"","version":"0","updatedAt":"0"}}`,
		`{"found":true,"record":{"key":"a","value":"","version":"1","updatedAt":"-1"}}`,
		`{"found":true,"record":{"key":"a","value":"!","version":"1","updatedAt":"0"}}`,
	} {
		parseBackend := NewStorageBackend(NewClient(&storageTransport{parseData: json.RawMessage(parsePayload)}))
		if _, _, parseErr := parseBackend.Load(context.Background(), "a"); !interop.IsCode(parseErr, interop.CodeDecode) {
			parseTest.Fatalf("payload %s: %v", parsePayload, parseErr)
		}
	}
	parseBackend := NewStorageBackend(NewClient(&storageTransport{parseData: json.RawMessage(`{"found":false,"record":{"key":"a","value":"","version":"8","updatedAt":"0"}}`)}))
	parseRecord, parseFound, parseErr := parseBackend.Load(context.Background(), "a")
	if parseErr != nil || parseFound || parseRecord.Version != 8 {
		parseTest.Fatalf("tombstone lost: %#v %t %v", parseRecord, parseFound, parseErr)
	}
}
func (parseTransport *storageTransport) Cancel(string) error           { return nil }
func (parseTransport *storageTransport) Listen(string) (string, error) { return "1", nil }
func (parseTransport *storageTransport) Next(string) (Reply, error)    { return Reply{Done: false}, nil }
func (parseTransport *storageTransport) Unlisten(string) error         { return nil }

func TestStorageBackendWireRoundTrip(t *testing.T) {
	parseValue := []byte{0, 1, 2, 255}
	parseTransport := &storageTransport{parseRecord: StorageWireRecord{Key: "wide", Value: base64.StdEncoding.EncodeToString(parseValue), Version: "9223372036854775807", UpdatedAt: "9223372036854775806"}}
	parseRecord, parseFound, parseErr := NewStorageBackend(NewClient(parseTransport)).Load(context.Background(), "wide")
	if parseErr != nil || !parseFound {
		t.Fatalf("load: %#v %v", parseRecord, parseErr)
	}
	if parseRecord.Key != "wide" || string(parseRecord.Value) != string(parseValue) || parseRecord.Version != 9223372036854775807 || parseRecord.UpdatedAt != 9223372036854775806 {
		t.Fatalf("wire decode mismatch: %#v", parseRecord)
	}
}

func TestStorageBackendImplementsPersistenceBackend(t *testing.T) {
	var parseBackend kvstate.PersistenceBackend = NewStorageBackend(NewClient(&storageTransport{}))
	parseKeys, parseErr := parseBackend.Keys(context.Background())
	if parseErr != nil || parseKeys != nil {
		t.Fatalf("backend interface call: %#v %v", parseKeys, parseErr)
	}
}
