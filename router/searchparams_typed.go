package router

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/validate"
)

// DecodeQuery decodes URL query values into a typed struct T using `query:"name"` tags
// (falling back to the lowercased field name), then validates the result with the validate
// package using any `validate:"..."` tags on the same struct. It is the typed-search-params
// path: read `?page=2&sort=desc` into a struct once, with conversion and validation, instead
// of scattering `q.Get("page")` + manual strconv + bounds checks across the codebase.
//
// Supported field kinds: string, bool, the sized int/uint kinds, float32/64, and []string
// (for repeated params). A missing param leaves the field at its zero value — mark it
// `validate:"required"` to reject that. A malformed value (e.g. page=abc for an int) or an
// unsupported field kind returns an error; validation failures return the validate.Result
// (which implements error and carries per-field messages).
//
//	type ListQuery struct {
//	    Page int    `query:"page" validate:"gte=1"`
//	    Sort string `query:"sort" validate:"omitempty,oneof=asc desc"`
//	    Tags []string `query:"tag"`
//	}
//	q, err := router.DecodeQuery[ListQuery](values) // err is a validate.Result on bad input
func DecodeQuery[T any](parseValues url.Values) (T, error) {
	var parseTarget T
	parseRV := reflect.ValueOf(&parseTarget).Elem()
	parseRT := parseRV.Type()
	if parseRT.Kind() != reflect.Struct {
		return parseTarget, fmt.Errorf("DecodeQuery: T must be a struct, got %s", parseRT.Kind())
	}

	for parseI := 0; parseI < parseRT.NumField(); parseI++ {
		parseField := parseRT.Field(parseI)
		if !parseField.IsExported() {
			continue
		}
		parseName := queryFieldName(parseField)
		if parseName == "-" {
			continue
		}
		parseRaw, parseHas := parseValues[parseName]
		if !parseHas || len(parseRaw) == 0 {
			continue
		}
		if parseErr := setQueryField(parseRV.Field(parseI), parseRaw); parseErr != nil {
			return parseTarget, fmt.Errorf("query param %q: %w", parseName, parseErr)
		}
	}

	if parseResult := validate.Struct(parseTarget); !parseResult.Valid() {
		return parseTarget, parseResult
	}
	return parseTarget, nil
}

// EncodeQuery renders a typed struct (or a non-nil pointer to one) back into url.Values
// using the same `query:"name"` tags, skipping zero-valued fields so the resulting URL stays
// clean. It is the inverse of DecodeQuery, for building links and updating the address bar
// from typed state. By contract it takes a struct: a non-struct, nil pointer, or other value
// has no query fields and returns an empty url.Values (it never panics) — DecodeQuery is the
// validated counterpart, so encoding stays intentionally permissive.
func EncodeQuery(parseValue any) url.Values {
	parseValues := url.Values{}
	parseRV := reflect.ValueOf(parseValue)
	if parseRV.Kind() == reflect.Pointer {
		if parseRV.IsNil() {
			return parseValues
		}
		parseRV = parseRV.Elem()
	}
	if parseRV.Kind() != reflect.Struct {
		return parseValues
	}
	parseRT := parseRV.Type()

	for parseI := 0; parseI < parseRT.NumField(); parseI++ {
		parseField := parseRT.Field(parseI)
		if !parseField.IsExported() {
			continue
		}
		parseName := queryFieldName(parseField)
		if parseName == "-" {
			continue
		}
		parseFieldValue := parseRV.Field(parseI)
		if parseFieldValue.Kind() == reflect.Slice {
			for parseJ := 0; parseJ < parseFieldValue.Len(); parseJ++ {
				parseValues.Add(parseName, fmt.Sprint(parseFieldValue.Index(parseJ).Interface()))
			}
			continue
		}
		if parseFieldValue.IsZero() {
			continue
		}
		parseValues.Set(parseName, fmt.Sprint(parseFieldValue.Interface()))
	}
	return parseValues
}

// queryFieldName returns the query key for a struct field: the first segment of its
// `query` tag, or the lowercased field name when there is no tag.
func queryFieldName(parseField reflect.StructField) string {
	parseTag := parseField.Tag.Get("query")
	if parseTag == "" {
		return strings.ToLower(parseField.Name)
	}
	parseName, _, _ := strings.Cut(parseTag, ",")
	if parseName == "" {
		return strings.ToLower(parseField.Name)
	}
	return parseName
}

// setQueryField converts raw query strings into the field's Go type.
func setQueryField(parseField reflect.Value, parseRaw []string) error {
	switch parseField.Kind() {
	case reflect.String:
		parseField.SetString(parseRaw[0])
	case reflect.Bool:
		parseBool, parseErr := strconv.ParseBool(parseRaw[0])
		if parseErr != nil {
			return fmt.Errorf("invalid boolean %q", parseRaw[0])
		}
		parseField.SetBool(parseBool)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parseInt, parseErr := strconv.ParseInt(parseRaw[0], 10, 64)
		if parseErr != nil {
			return fmt.Errorf("invalid integer %q", parseRaw[0])
		}
		parseField.SetInt(parseInt)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		parseUint, parseErr := strconv.ParseUint(parseRaw[0], 10, 64)
		if parseErr != nil {
			return fmt.Errorf("invalid unsigned integer %q", parseRaw[0])
		}
		parseField.SetUint(parseUint)
	case reflect.Float32, reflect.Float64:
		parseFloat, parseErr := strconv.ParseFloat(parseRaw[0], 64)
		if parseErr != nil {
			return fmt.Errorf("invalid number %q", parseRaw[0])
		}
		parseField.SetFloat(parseFloat)
	case reflect.Slice:
		if parseField.Type().Elem().Kind() != reflect.String {
			return fmt.Errorf("unsupported slice element kind %s (only []string)", parseField.Type().Elem().Kind())
		}
		parseField.Set(reflect.ValueOf(append([]string(nil), parseRaw...)))
	default:
		return fmt.Errorf("unsupported field kind %s", parseField.Kind())
	}
	return nil
}
