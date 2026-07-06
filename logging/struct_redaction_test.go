package logging

import "testing"

// TestStructFieldsAreWalkedAndRedactable pins the fix for the struct redaction
// blind spot: a struct log value is now walked into a map[string]any (honoring
// json tags, skipping unexported fields) so telemetry redaction can match its
// sensitive field names. Previously the struct collapsed to a fmt.Sprint string
// that redaction could not decompose, leaking secrets in full.
func TestStructFieldsAreWalkedAndRedactable(parseT *testing.T) {
	parsePrev := CurrentTelemetryRedaction()
	ConfigureTelemetryRedaction(RedactionPolicy{Fields: []string{"password", "token"}})
	defer ConfigureTelemetryRedaction(parsePrev)

	type parseCreds struct {
		User     string
		Password string `json:"password"`
		Token    string
		secret   string // unexported — must be skipped, not leaked
	}
	parseValue := parseCreds{User: "alice", Password: "hunter2", Token: "tok-abc", secret: "nope"}

	parseNormalized := normalizeLogValue(parseValue)
	parseMap, parseOk := parseNormalized.(map[string]any)
	if !parseOk {
		parseT.Fatalf("struct must normalize to a map (so its fields are redactable), got %T: %#v", parseNormalized, parseNormalized)
	}
	if _, parsePresent := parseMap["secret"]; parsePresent {
		parseT.Fatal("unexported struct field must not be walked into the log object")
	}

	parseRedacted, parseOkRedacted := RedactTelemetryValue("log.attributes.creds", parseMap).(map[string]any)
	if !parseOkRedacted {
		parseT.Fatalf("redacted value should stay a map, got %#v", parseRedacted)
	}
	if parseRedacted["password"] != "[redacted]" {
		parseT.Fatalf("password field must be redacted, got %#v", parseRedacted["password"])
	}
	if parseRedacted["Token"] != "[redacted]" {
		parseT.Fatalf("Token field must be redacted (case-insensitive match), got %#v", parseRedacted["Token"])
	}
	if parseRedacted["User"] != "alice" {
		parseT.Fatalf("non-sensitive field must survive, got %#v", parseRedacted["User"])
	}
}

// TestStructWalkHonorsJSONDashAndNestedStructs pins that json:"-" opts a field out
// and that a nested struct is walked recursively (so nested secrets are redactable).
func TestStructWalkHonorsJSONDashAndNestedStructs(parseT *testing.T) {
	type parseInner struct {
		APIKey string `json:"apiKey"`
	}
	type parseOuter struct {
		Name   string
		Hidden string `json:"-"`
		Inner  parseInner
	}
	parseNormalized := normalizeLogValue(parseOuter{Name: "n", Hidden: "h", Inner: parseInner{APIKey: "k"}})
	parseMap := parseNormalized.(map[string]any)
	if _, parsePresent := parseMap["Hidden"]; parsePresent {
		parseT.Fatal(`json:"-" field must be excluded`)
	}
	parseInnerMap, parseOk := parseMap["Inner"].(map[string]any)
	if !parseOk {
		parseT.Fatalf("nested struct must be walked to a map, got %#v", parseMap["Inner"])
	}
	if parseInnerMap["apiKey"] != "k" {
		parseT.Fatalf("nested struct field should be present under its json tag, got %#v", parseInnerMap)
	}
}
