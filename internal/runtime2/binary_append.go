package runtime2

import "encoding/binary"

// appendBinaryUint16 appends one uint16 value to the provided payload buffer.
func appendBinaryUint16(parsePayload []byte, parseValue uint16) []byte {
	return binary.LittleEndian.AppendUint16(parsePayload, parseValue)
}

// appendBinaryUint32 appends one uint32 value to the provided payload buffer.
func appendBinaryUint32(parsePayload []byte, parseValue uint32) []byte {
	return binary.LittleEndian.AppendUint32(parsePayload, parseValue)
}

// appendBinaryUint64 appends one uint64 value to the provided payload buffer.
func appendBinaryUint64(parsePayload []byte, parseValue uint64) []byte {
	return binary.LittleEndian.AppendUint64(parsePayload, parseValue)
}

// setBinaryUint32At writes one uint32 value into an existing payload offset.
func setBinaryUint32At(parsePayload []byte, parseOffset int, parseValue uint32) {
	binary.LittleEndian.PutUint32(parsePayload[parseOffset:parseOffset+4], parseValue)
}
