package runtime2

// BuildBinaryPropsValue encodes one props payload using the same binary value graph as source values.
func BuildBinaryPropsValue(parseProps any) ([]byte, error) {
	if parseErr := ValidateSerializableProps(parseProps); parseErr != nil {
		return nil, parseErr
	}
	return BuildBinarySourceValue(parseProps)
}

// ParseBinaryPropsValue decodes one binary props payload back into the runtime2 value graph.
func ParseBinaryPropsValue(parsePayload []byte) (any, error) {
	return ParseBinarySourceValue(parsePayload)
}
