package runtime2

import "encoding/binary"

// appendBinaryUint16 appends one uint16 value to the provided payload buffer.
func appendBinaryUint16(parsePayload []byte, parseValue uint16) []byte {
	var parseBytes [2]byte
	binary.LittleEndian.PutUint16(parseBytes[:], parseValue)
	return append(parsePayload, parseBytes[:]...)
}

// appendBinaryUint32 appends one uint32 value to the provided payload buffer.
func appendBinaryUint32(parsePayload []byte, parseValue uint32) []byte {
	var parseBytes [4]byte
	binary.LittleEndian.PutUint32(parseBytes[:], parseValue)
	return append(parsePayload, parseBytes[:]...)
}

// appendBinaryUint64 appends one uint64 value to the provided payload buffer.
func appendBinaryUint64(parsePayload []byte, parseValue uint64) []byte {
	var parseBytes [8]byte
	binary.LittleEndian.PutUint64(parseBytes[:], parseValue)
	return append(parsePayload, parseBytes[:]...)
}
