// Tests for the topic-type codec registry (typed_publish.go).
//
// The central proof: PublishJSON on a topic with a RegisterTopic[T] codec
// DOES reach a Subscribe[T] handler.  Without RegisterTopic the typed
// subscriber is silently skipped (the documented boundary).
package events

import (
	"encoding/json"
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// resetCodecAndTopic clears both the codec registry entry and the event
// registry entry for parseTopic so tests start from a clean slate.
func resetCodecAndTopic(parseTopic string) {
	codecRegistry.Delete(parseTopic)
	globalRegistry.Delete(parseTopic)
}

// ---------------------------------------------------------------------------
// 1. Core fix: typed subscriber receives JSON publish when codec registered.
// ---------------------------------------------------------------------------

// TestPublishJSONReachesTypedSubscriber is the primary regression test for the
// "any-publish skips typed subscribers" gap.
//
// The gap is clearest for struct types: without RegisterTopic, a JSON object
// decodes to map[string]any and fails the type assertion inside
// Subscribe[testEvent].  With RegisterTopic[testEvent], the JSON is unmarshalled
// directly into testEvent before Publish[testEvent] is called, so the assertion
// always succeeds.
func TestPublishJSONReachesTypedSubscriber(t *testing.T) {
	// Struct sub-test: demonstrates the core fix (map[string]any → testEvent gap).
	t.Run("struct", func(t *testing.T) {
		parseTopic := "typed.greeting.struct"
		resetCodecAndTopic(parseTopic)

		RegisterTopic[testEvent](parseTopic)
		defer codecRegistry.Delete(parseTopic)

		parseReceived := []testEvent{}
		parseUnsub := Subscribe(parseTopic, func(parseV testEvent) {
			parseReceived = append(parseReceived, parseV)
		})
		defer parseUnsub()

		parseErr := PublishJSON(parseTopic, json.RawMessage(`{"name":"hello","score":1}`))
		if parseErr != nil {
			t.Fatalf("PublishJSON returned unexpected error: %v", parseErr)
		}
		if len(parseReceived) != 1 {
			t.Fatalf("typed subscriber received %d values, want 1", len(parseReceived))
		}
		if parseReceived[0].Name != "hello" {
			t.Fatalf("typed subscriber got %+v, want name=hello", parseReceived[0])
		}
	})

	// String sub-test: RegisterTopic[string] also routes correctly.
	t.Run("string", func(t *testing.T) {
		parseTopic := "typed.greeting.string"
		resetCodecAndTopic(parseTopic)

		RegisterTopic[string](parseTopic)
		defer codecRegistry.Delete(parseTopic)

		parseReceived := []string{}
		parseUnsub := Subscribe(parseTopic, func(parseV string) {
			parseReceived = append(parseReceived, parseV)
		})
		defer parseUnsub()

		parseErr := PublishJSON(parseTopic, json.RawMessage(`"hello"`))
		if parseErr != nil {
			t.Fatalf("PublishJSON returned unexpected error: %v", parseErr)
		}
		if len(parseReceived) != 1 || parseReceived[0] != "hello" {
			t.Fatalf("typed subscriber got %v, want [hello]", parseReceived)
		}
	})
}

// TestPublishJSONWithoutCodecDoesNotReachTypedSubscriber documents the
// boundary: when no RegisterTopic is called, PublishJSON falls back to
// Publish[any].  For struct types, json.Unmarshal decodes a JSON object into
// map[string]any, which does NOT satisfy a Subscribe[testEvent] type
// assertion.  The typed subscriber is therefore silently skipped.
//
// Note: for JSON primitives (strings, numbers, booleans) the fallback DOES
// reach typed subscribers because json decodes them into the matching Go
// primitive types.  The gap only manifests for composite types (structs,
// custom types).
func TestPublishJSONWithoutCodecDoesNotReachTypedSubscriber(t *testing.T) {
	parseTopicNoCodec := "typed.struct.nocodec"
	resetCodecAndTopic(parseTopicNoCodec)
	// Intentionally do NOT call RegisterTopic[testEvent] for this topic.

	parseReceived := []testEvent{}
	parseUnsub := Subscribe(parseTopicNoCodec, func(parseV testEvent) {
		parseReceived = append(parseReceived, parseV)
	})
	defer parseUnsub()

	// JSON object -> json.Unmarshal into any -> map[string]any, not testEvent.
	parseErr := PublishJSON(parseTopicNoCodec, json.RawMessage(`{"name":"alice","score":42}`))
	if parseErr != nil {
		t.Fatalf("PublishJSON fallback returned unexpected error: %v", parseErr)
	}

	// The typed subscriber must NOT receive the value: Publish[any] delivers
	// map[string]any and the type assertion to testEvent fails.
	if len(parseReceived) != 0 {
		t.Fatalf("typed struct subscriber should NOT receive value without codec (got map[string]any), got %d deliveries", len(parseReceived))
	}
}

// ---------------------------------------------------------------------------
// 2. Struct type: RegisterTopic[T] works for composite types.
// ---------------------------------------------------------------------------

type testEvent struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

func TestPublishJSONStructType(t *testing.T) {
	parseTopic := "typed.event.struct"
	resetCodecAndTopic(parseTopic)

	RegisterTopic[testEvent](parseTopic)
	defer codecRegistry.Delete(parseTopic)

	parseReceived := []testEvent{}
	parseUnsub := Subscribe(parseTopic, func(parseV testEvent) {
		parseReceived = append(parseReceived, parseV)
	})
	defer parseUnsub()

	parsePayload := json.RawMessage(`{"name":"alice","score":42}`)
	parseErr := PublishJSON(parseTopic, parsePayload)
	if parseErr != nil {
		t.Fatalf("PublishJSON returned unexpected error: %v", parseErr)
	}

	if len(parseReceived) != 1 {
		t.Fatalf("typed subscriber received %d values, want 1", len(parseReceived))
	}
	parseGot := parseReceived[0]
	if parseGot.Name != "alice" || parseGot.Score != 42 {
		t.Fatalf("unexpected struct value: %+v", parseGot)
	}
}

// ---------------------------------------------------------------------------
// 3. Wrong-shape JSON returns an error and does not panic.
// ---------------------------------------------------------------------------

func TestPublishJSONWrongShapeReturnsError(t *testing.T) {
	parseTopic := "typed.event.wrongshape"
	resetCodecAndTopic(parseTopic)

	RegisterTopic[testEvent](parseTopic)
	defer codecRegistry.Delete(parseTopic)

	// Subscribe so we can verify no delivery occurs on error.
	parseReceived := []testEvent{}
	parseUnsub := Subscribe(parseTopic, func(parseV testEvent) {
		parseReceived = append(parseReceived, parseV)
	})
	defer parseUnsub()

	// A plain JSON string cannot unmarshal into testEvent.
	parseErr := PublishJSON(parseTopic, json.RawMessage(`"not a struct"`))
	if parseErr == nil {
		t.Fatal("PublishJSON should return an error for wrong-shape JSON, got nil")
	}

	if len(parseReceived) != 0 {
		t.Fatalf("subscriber received %d values despite unmarshal error, want 0", len(parseReceived))
	}
}

// ---------------------------------------------------------------------------
// 4. RegisterTopic replaces an existing codec (re-registration).
// ---------------------------------------------------------------------------

func TestRegisterTopicReplacesExistingCodec(t *testing.T) {
	parseTopic := "typed.replace"
	resetCodecAndTopic(parseTopic)
	defer resetCodecAndTopic(parseTopic)

	// Register once for int, then replace with string.
	RegisterTopic[int](parseTopic)
	RegisterTopic[string](parseTopic)

	parseReceived := []string{}
	parseUnsub := Subscribe(parseTopic, func(parseV string) {
		parseReceived = append(parseReceived, parseV)
	})
	defer parseUnsub()

	parseErr := PublishJSON(parseTopic, json.RawMessage(`"replaced"`))
	if parseErr != nil {
		t.Fatalf("PublishJSON returned unexpected error after re-registration: %v", parseErr)
	}
	if len(parseReceived) != 1 || parseReceived[0] != "replaced" {
		t.Fatalf("unexpected received values after re-registration: %v", parseReceived)
	}
}

// ---------------------------------------------------------------------------
// 5. RegisteredTopicCodecs returns a sorted list of registered topics.
// ---------------------------------------------------------------------------

func TestRegisteredTopicCodecsSorted(t *testing.T) {
	// Use a fresh sub-registry by clearing then adding known topics.
	clearCodecRegistry()
	defer clearCodecRegistry()

	parseTopics := []string{"zoo.topic", "alpha.topic", "middle.topic"}
	for _, parseTopic := range parseTopics {
		RegisterTopic[string](parseTopic)
	}

	parseGot := RegisteredTopicCodecs()

	// Must be sorted ascending.
	parseWant := []string{"alpha.topic", "middle.topic", "zoo.topic"}
	if len(parseGot) != len(parseWant) {
		t.Fatalf("RegisteredTopicCodecs returned %d topics, want %d: %v", len(parseGot), len(parseWant), parseGot)
	}
	for parseI, parseV := range parseGot {
		if parseV != parseWant[parseI] {
			t.Fatalf("index %d: got %q, want %q", parseI, parseV, parseWant[parseI])
		}
	}
}

// ---------------------------------------------------------------------------
// 6. Fallback path: unregistered topic, untyped subscriber receives value.
// ---------------------------------------------------------------------------

func TestPublishJSONFallbackUntypedSubscriberReceives(t *testing.T) {
	parseTopic := "typed.fallback.any"
	resetCodecAndTopic(parseTopic)
	// No RegisterTopic — exercises fallback to Publish[any].

	parseReceived := []any{}
	parseUnsub := Subscribe(parseTopic, func(parseV any) {
		parseReceived = append(parseReceived, parseV)
	})
	defer parseUnsub()

	parseErr := PublishJSON(parseTopic, json.RawMessage(`"world"`))
	if parseErr != nil {
		t.Fatalf("PublishJSON fallback returned unexpected error: %v", parseErr)
	}

	if len(parseReceived) != 1 {
		t.Fatalf("untyped subscriber received %d values, want 1", len(parseReceived))
	}
	parseStr, parseOk := parseReceived[0].(string)
	if !parseOk || parseStr != "world" {
		t.Fatalf("untyped subscriber got unexpected value: %v", parseReceived[0])
	}
}

// ---------------------------------------------------------------------------
// 7. Malformed JSON in fallback path returns an error.
// ---------------------------------------------------------------------------

func TestPublishJSONMalformedJSONFallbackError(t *testing.T) {
	parseTopic := "typed.fallback.malformed"
	resetCodecAndTopic(parseTopic)

	parseErr := PublishJSON(parseTopic, json.RawMessage(`{bad json`))
	if parseErr == nil {
		t.Fatal("PublishJSON should return error for malformed JSON in fallback path, got nil")
	}
}
