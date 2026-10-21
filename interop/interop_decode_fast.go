package interop

import (
	"encoding"
	"encoding/json"
	"reflect"
	"strings"
)

var interopJSONUnmarshalerType = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()

// decodeSimpleMapToStruct handles the small, already-decoded payloads commonly
// received from browser messages without a marshal/unmarshal round trip. It
// deliberately accepts only plain exported fields and scalar/map values. Any
// shape outside that conservative subset falls back to encoding/json so Decode
// retains its complete JSON semantics.
func decodeSimpleMapToStruct(parseValue any, parseTarget any) (bool, error) {
	parseMap, parseMapOK := parseValue.(map[string]any)
	if !parseMapOK {
		return false, nil
	}
	parseTargetValue := reflect.ValueOf(parseTarget)
	if !parseTargetValue.IsValid() || parseTargetValue.Kind() != reflect.Pointer || parseTargetValue.IsNil() || parseTargetValue.Elem().Kind() != reflect.Struct {
		return false, nil
	}
	parseStructType := parseTargetValue.Elem().Type()
	if parseStructType.Implements(interopJSONUnmarshalerType) || reflect.PointerTo(parseStructType).Implements(interopJSONUnmarshalerType) {
		return false, nil
	}
	parseFields := make(map[string]reflect.Value, parseStructType.NumField())
	for parseIndex := 0; parseIndex < parseStructType.NumField(); parseIndex++ {
		parseFieldType := parseStructType.Field(parseIndex)
		if parseFieldType.PkgPath != "" || parseFieldType.Anonymous {
			return false, nil
		}
		parseTag := parseFieldType.Tag.Get("json")
		parseName := parseTag
		if parseName != "" {
			if parseName == "-" {
				continue
			}
			if parseComma := strings.IndexByte(parseName, ','); parseComma >= 0 {
				// Options can alter JSON decoding (notably ",string"); keep
				// the conservative encoding/json path for all of them.
				return false, nil
			}
		}
		if parseName == "" {
			parseName = parseFieldType.Name
		}
		parseFieldValue := parseTargetValue.Elem().Field(parseIndex)
		if !parseFieldValue.CanSet() || !simpleJSONField(parseFieldType.Type) {
			return false, nil
		}
		if _, parseDuplicate := parseFields[parseName]; parseDuplicate {
			return false, nil
		}
		parseFields[parseName] = parseFieldValue
	}
	for parseName, parseValue := range parseMap {
		parseFieldValue, parseFieldOK := parseFields[parseName]
		if !parseFieldOK {
			// Preserve encoding/json's handling of unknown values. In
			// particular, an unknown channel/function can make Marshal fail.
			// Falling back also preserves case-insensitive field matching.
			return false, nil
		}
		if !canAssignSimpleJSONValue(parseFieldValue, parseValue) {
			return false, nil
		}
	}
	for parseName, parseValue := range parseMap {
		parseFieldValue, parseFieldOK := parseFields[parseName]
		if parseFieldOK {
			assignSimpleJSONValue(parseFieldValue, parseValue)
		}
	}
	return true, nil
}

func simpleJSONField(parseType reflect.Type) bool {
	if parseType.Implements(interopJSONUnmarshalerType) || reflect.PointerTo(parseType).Implements(interopJSONUnmarshalerType) {
		return false
	}
	switch parseType.Kind() {
	case reflect.String, reflect.Bool:
		return true
	case reflect.Map:
		parseKeyType, parseValueType := parseType.Key(), parseType.Elem()
		if parseKeyType.Kind() != reflect.String || parseValueType.Kind() != reflect.String {
			return false
		}
		parseTextUnmarshalerType := reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
		if parseKeyType.Implements(parseTextUnmarshalerType) || reflect.PointerTo(parseKeyType).Implements(parseTextUnmarshalerType) {
			return false
		}
		if parseValueType.Implements(interopJSONUnmarshalerType) || reflect.PointerTo(parseValueType).Implements(interopJSONUnmarshalerType) {
			return false
		}
		return true
	default:
		return false
	}
}

func assignSimpleJSONValue(parseTarget reflect.Value, parseValue any) bool {
	if !canAssignSimpleJSONValue(parseTarget, parseValue) {
		return false
	}
	switch parseTarget.Kind() {
	case reflect.String:
		parseString, parseOK := parseValue.(string)
		if parseOK {
			parseTarget.SetString(parseString)
		}
		return parseOK
	case reflect.Bool:
		parseBool, parseOK := parseValue.(bool)
		if parseOK {
			parseTarget.SetBool(parseBool)
		}
		return parseOK
	case reflect.Map:
		parseMap, parseOK := parseValue.(map[string]any)
		if !parseOK {
			return false
		}
		parseResult := parseTarget
		if parseResult.IsNil() {
			parseResult = reflect.MakeMapWithSize(parseTarget.Type(), len(parseMap))
		}
		for parseKey, parseRaw := range parseMap {
			parseString := parseRaw.(string)
			parseResult.SetMapIndex(reflect.ValueOf(parseKey).Convert(parseTarget.Type().Key()), reflect.ValueOf(parseString).Convert(parseTarget.Type().Elem()))
		}
		parseTarget.Set(parseResult)
		return true
	default:
		return false
	}
}

func canAssignSimpleJSONValue(parseTarget reflect.Value, parseValue any) bool {
	switch parseTarget.Kind() {
	case reflect.String:
		_, parseOK := parseValue.(string)
		return parseOK
	case reflect.Bool:
		_, parseOK := parseValue.(bool)
		return parseOK
	case reflect.Map:
		parseMap, parseOK := parseValue.(map[string]any)
		if !parseOK {
			return false
		}
		for _, parseRaw := range parseMap {
			if _, parseStringOK := parseRaw.(string); !parseStringOK {
				return false
			}
		}
		return true
	default:
		return false
	}
}
