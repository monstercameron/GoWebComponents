package runtime2

import (
	"fmt"
	"hash/crc32"
)

// BinaryMountEnvelope stores the mount payload encoded over runtime2 binary transport.
type BinaryMountEnvelope struct {
	RegionInstanceID RegionInstanceID
	RendererID       RendererID
	SourceIDs        []string
	Snapshot         SnapshotEnvelope
}

// BinaryUpdateEnvelope stores the update payload encoded over runtime2 binary transport.
type BinaryUpdateEnvelope struct {
	RegionInstanceID RegionInstanceID
	InputVersion     uint64
	Snapshot         SnapshotEnvelope
}

// BuildBinaryMountEnvelope encodes one validated mount envelope using the runtime2 binary transport.
func BuildBinaryMountEnvelope(parseEnvelope BinaryMountEnvelope) ([]byte, error) {
	if _, parseErr := ParseRegionInstanceID(string(parseEnvelope.RegionInstanceID)); parseErr != nil {
		return nil, parseErr
	}
	if _, parseErr := ParseRendererID(string(parseEnvelope.RendererID)); parseErr != nil {
		return nil, parseErr
	}
	parseSourceIDs, parseErr := NormalizeSourceIDs(parseEnvelope.SourceIDs)
	if parseErr != nil {
		return nil, parseErr
	}
	parseEnvelope.SourceIDs = parseSourceIDs
	if parseErr := ValidateSnapshotEnvelope(parseEnvelope.Snapshot); parseErr != nil {
		return nil, parseErr
	}
	if parseEnvelope.Snapshot.RegionInstanceID != parseEnvelope.RegionInstanceID {
		return nil, fmt.Errorf("runtime2: mount snapshot region instance ID mismatch")
	}
	// Pre-reserve header space; encode body and snapshot inline to avoid intermediate allocations.
	buildSourceTableCap := 2
	for _, parseID := range parseEnvelope.SourceIDs {
		buildSourceTableCap += 2 + len(parseID)
	}
	buildSnapshotBodyCap := 2 + len(string(parseEnvelope.Snapshot.RegionInstanceID)) + 24 + 4 + 64 + 4 + 4 + (len(parseEnvelope.Snapshot.Sources)+1)*16
	parsePayload := make([]byte, binaryEnvelopeHeaderSize, binaryEnvelopeHeaderSize+2+len(string(parseEnvelope.RegionInstanceID))+2+len(string(parseEnvelope.RendererID))+4+buildSourceTableCap+4+buildSnapshotBodyCap)
	parsePayload, parseErr = appendBinaryLengthPrefixedString(parsePayload, string(parseEnvelope.RegionInstanceID))
	if parseErr != nil {
		return nil, parseErr
	}
	parsePayload, parseErr = appendBinaryLengthPrefixedString(parsePayload, string(parseEnvelope.RendererID))
	if parseErr != nil {
		return nil, parseErr
	}
	parseSourceIDTableLenOff := len(parsePayload)
	parsePayload = append(parsePayload, 0, 0, 0, 0)
	parseSourceIDTableStart := len(parsePayload)
	parsePayload, parseErr = appendBinarySourceIDTableFromNormalized(parsePayload, parseEnvelope.SourceIDs)
	if parseErr != nil {
		return nil, parseErr
	}
	setBinaryUint32At(parsePayload, parseSourceIDTableLenOff, uint32(len(parsePayload)-parseSourceIDTableStart))
	// Snapshot body: write-back length, encode directly into the same buffer.
	parseSnapshotBodyLenOff := len(parsePayload)
	parsePayload = append(parsePayload, 0, 0, 0, 0)
	parseSnapshotBodyStart := len(parsePayload)
	parsePayload, parseErr = appendBinarySnapshotBody(parsePayload, parseEnvelope.Snapshot)
	if parseErr != nil {
		return nil, parseErr
	}
	setBinaryUint32At(parsePayload, parseSnapshotBodyLenOff, uint32(len(parsePayload)-parseSnapshotBodyStart))
	parseBodyLen := uint32(len(parsePayload) - binaryEnvelopeHeaderSize)
	parseChecksum := crc32.ChecksumIEEE(parsePayload[binaryEnvelopeHeaderSize:])
	writeBinaryEnvelopeHeaderAt(parsePayload, BinaryEnvelopeKindMount, parseBodyLen, parseChecksum)
	return parsePayload, nil
}

// ParseBinaryMountEnvelope decodes and validates one runtime2 binary mount envelope payload.
func ParseBinaryMountEnvelope(parsePayload []byte) (BinaryMountEnvelope, error) {
	parseHeader, parseBody, parseErr := parseBinaryEnvelopePayload(parsePayload, BinaryEnvelopeKindMount)
	if parseErr != nil {
		return BinaryMountEnvelope{}, parseErr
	}
	parseChecksum := crc32.ChecksumIEEE(parseBody)
	if parseChecksum != parseHeader.Checksum {
		return BinaryMountEnvelope{}, fmt.Errorf("runtime2: binary mount checksum mismatch expected=%d actual=%d", parseHeader.Checksum, parseChecksum)
	}
	parseRegionText, parseOffset, parseErr := parseBinaryString(parseBody, 0, "mount-body", "region_instance_id")
	if parseErr != nil {
		return BinaryMountEnvelope{}, parseErr
	}
	parseRendererText, parseOffset, parseErr := parseBinaryString(parseBody, parseOffset, "mount-body", "renderer_id")
	if parseErr != nil {
		return BinaryMountEnvelope{}, parseErr
	}
	parseSourceTableLength, parseOffset, parseErr := parseBinaryUint32(parseBody, parseOffset, "mount-body", "source-id-table length")
	if parseErr != nil {
		return BinaryMountEnvelope{}, parseErr
	}
	parseSourceTableSpan, parseOffset, parseErr := parseBinaryPayloadSpan(parseBody, parseOffset, int(parseSourceTableLength), "mount-body", "source-id-table payload")
	if parseErr != nil {
		return BinaryMountEnvelope{}, parseErr
	}
	parseSourceIDs, parseErr := ParseBinarySourceIDTable(parseSourceTableSpan)
	if parseErr != nil {
		return BinaryMountEnvelope{}, fmt.Errorf("runtime2: decode mount source-id table: %w", parseErr)
	}
	parseSnapshotLength, parseOffset, parseErr := parseBinaryUint32(parseBody, parseOffset, "mount-body", "snapshot length")
	if parseErr != nil {
		return BinaryMountEnvelope{}, parseErr
	}
	parseSnapshotSpan, parseOffset, parseErr := parseBinaryPayloadSpan(parseBody, parseOffset, int(parseSnapshotLength), "mount-body", "snapshot payload")
	if parseErr != nil {
		return BinaryMountEnvelope{}, parseErr
	}
	parseSnapshot, parseErr := ParseBinarySnapshotBody(parseSnapshotSpan)
	if parseErr != nil {
		return BinaryMountEnvelope{}, fmt.Errorf("runtime2: decode mount snapshot: %w", parseErr)
	}
	if parseOffset != len(parseBody) {
		return BinaryMountEnvelope{}, fmt.Errorf("runtime2: mount-body has %d trailing bytes", len(parseBody)-parseOffset)
	}
	parseRegionInstanceID, parseErr := ParseRegionInstanceID(parseRegionText)
	if parseErr != nil {
		return BinaryMountEnvelope{}, parseErr
	}
	parseRendererID, parseErr := ParseRendererID(parseRendererText)
	if parseErr != nil {
		return BinaryMountEnvelope{}, parseErr
	}
	if parseSnapshot.RegionInstanceID != parseRegionInstanceID {
		return BinaryMountEnvelope{}, fmt.Errorf("runtime2: mount snapshot region instance ID mismatch")
	}
	return BinaryMountEnvelope{
		RegionInstanceID: parseRegionInstanceID,
		RendererID:       parseRendererID,
		SourceIDs:        parseSourceIDs,
		Snapshot:         parseSnapshot,
	}, nil
}

// BuildBinaryUpdateEnvelope encodes one validated update envelope using the runtime2 binary transport.
func BuildBinaryUpdateEnvelope(parseEnvelope BinaryUpdateEnvelope) ([]byte, error) {
	if _, parseErr := ParseRegionInstanceID(string(parseEnvelope.RegionInstanceID)); parseErr != nil {
		return nil, parseErr
	}
	if parseEnvelope.InputVersion == 0 {
		return nil, fmt.Errorf("runtime2: update input version is required")
	}
	if parseErr := ValidateSnapshotEnvelope(parseEnvelope.Snapshot); parseErr != nil {
		return nil, parseErr
	}
	if parseEnvelope.Snapshot.RegionInstanceID != parseEnvelope.RegionInstanceID {
		return nil, fmt.Errorf("runtime2: update snapshot region instance ID mismatch")
	}
	if parseEnvelope.Snapshot.InputVersion != parseEnvelope.InputVersion {
		return nil, fmt.Errorf("runtime2: update snapshot input version mismatch")
	}
	// Pre-reserve header space; encode body and snapshot inline to avoid intermediate allocations.
	buildSnapshotBodyCap := 2 + len(string(parseEnvelope.Snapshot.RegionInstanceID)) + 24 + 4 + 64 + 4 + 4 + (len(parseEnvelope.Snapshot.Sources)+1)*16
	parsePayload := make([]byte, binaryEnvelopeHeaderSize, binaryEnvelopeHeaderSize+2+len(string(parseEnvelope.RegionInstanceID))+8+4+buildSnapshotBodyCap)
	var parseErr error
	parsePayload, parseErr = appendBinaryLengthPrefixedString(parsePayload, string(parseEnvelope.RegionInstanceID))
	if parseErr != nil {
		return nil, parseErr
	}
	parsePayload = appendBinaryUint64(parsePayload, parseEnvelope.InputVersion)
	// Snapshot body: write-back length, encode directly into the same buffer.
	parseSnapshotBodyLenOff := len(parsePayload)
	parsePayload = append(parsePayload, 0, 0, 0, 0)
	parseSnapshotBodyStart := len(parsePayload)
	parsePayload, parseErr = appendBinarySnapshotBody(parsePayload, parseEnvelope.Snapshot)
	if parseErr != nil {
		return nil, parseErr
	}
	setBinaryUint32At(parsePayload, parseSnapshotBodyLenOff, uint32(len(parsePayload)-parseSnapshotBodyStart))
	parseBodyLen := uint32(len(parsePayload) - binaryEnvelopeHeaderSize)
	parseChecksum := crc32.ChecksumIEEE(parsePayload[binaryEnvelopeHeaderSize:])
	writeBinaryEnvelopeHeaderAt(parsePayload, BinaryEnvelopeKindUpdate, parseBodyLen, parseChecksum)
	return parsePayload, nil
}

// ParseBinaryUpdateEnvelope decodes and validates one runtime2 binary update envelope payload.
func ParseBinaryUpdateEnvelope(parsePayload []byte) (BinaryUpdateEnvelope, error) {
	parseHeader, parseBody, parseErr := parseBinaryEnvelopePayload(parsePayload, BinaryEnvelopeKindUpdate)
	if parseErr != nil {
		return BinaryUpdateEnvelope{}, parseErr
	}
	parseChecksum := crc32.ChecksumIEEE(parseBody)
	if parseChecksum != parseHeader.Checksum {
		return BinaryUpdateEnvelope{}, fmt.Errorf("runtime2: binary update checksum mismatch expected=%d actual=%d", parseHeader.Checksum, parseChecksum)
	}
	parseRegionText, parseOffset, parseErr := parseBinaryString(parseBody, 0, "update-body", "region_instance_id")
	if parseErr != nil {
		return BinaryUpdateEnvelope{}, parseErr
	}
	parseInputVersion, parseOffset, parseErr := parseBinaryUint64(parseBody, parseOffset, "update-body", "input_version")
	if parseErr != nil {
		return BinaryUpdateEnvelope{}, parseErr
	}
	parseSnapshotLength, parseOffset, parseErr := parseBinaryUint32(parseBody, parseOffset, "update-body", "snapshot length")
	if parseErr != nil {
		return BinaryUpdateEnvelope{}, parseErr
	}
	parseSnapshotSpan, parseOffset, parseErr := parseBinaryPayloadSpan(parseBody, parseOffset, int(parseSnapshotLength), "update-body", "snapshot payload")
	if parseErr != nil {
		return BinaryUpdateEnvelope{}, parseErr
	}
	parseSnapshot, parseErr := ParseBinarySnapshotBody(parseSnapshotSpan)
	if parseErr != nil {
		return BinaryUpdateEnvelope{}, fmt.Errorf("runtime2: decode update snapshot: %w", parseErr)
	}
	if parseOffset != len(parseBody) {
		return BinaryUpdateEnvelope{}, fmt.Errorf("runtime2: update-body has %d trailing bytes", len(parseBody)-parseOffset)
	}
	parseRegionInstanceID, parseErr := ParseRegionInstanceID(parseRegionText)
	if parseErr != nil {
		return BinaryUpdateEnvelope{}, parseErr
	}
	if parseInputVersion == 0 {
		return BinaryUpdateEnvelope{}, fmt.Errorf("runtime2: update input version is required")
	}
	if parseSnapshot.RegionInstanceID != parseRegionInstanceID {
		return BinaryUpdateEnvelope{}, fmt.Errorf("runtime2: update snapshot region instance ID mismatch")
	}
	if parseSnapshot.InputVersion != parseInputVersion {
		return BinaryUpdateEnvelope{}, fmt.Errorf("runtime2: update snapshot input version mismatch")
	}
	return BinaryUpdateEnvelope{
		RegionInstanceID: parseRegionInstanceID,
		InputVersion:     parseInputVersion,
		Snapshot:         parseSnapshot,
	}, nil
}
