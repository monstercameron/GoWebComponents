// Package events — typed_publish.go adds a topic-type codec registry that lets
// an AI agent bridge (or any JSON source) publish events that reach typed
// subscribers registered with Subscribe[T].
//
// Background: Publish[any](topic, v) does NOT reach a Subscribe[string]
// handler because the deliver wrapper performs parseValue.(T) — when T is
// string but parseValue carries the dynamic type any (double-boxed), the
// assertion fails and the handler is skipped silently.  The codec registry
// fixes this by decoding JSON into the concrete Go type T and then calling
// Publish[T], so the assertion always succeeds.
package events

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

// ---------------------------------------------------------------------------
// Internal codec registry
// ---------------------------------------------------------------------------

// topicCodec holds the registered JSON decoder for one topic.
type topicCodec struct {
	// decode unmarshals raw JSON into the topic's concrete type T and calls
	// Publish[T] so typed subscribers receive the value.
	decode func(parseRaw json.RawMessage) error
}

// codecRegistry maps topic string → topicCodec.  sync.Map is chosen for
// lock-free reads on the hot path; writes (RegisterTopic) are rare.
var codecRegistry sync.Map

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// RegisterTopic registers a JSON codec for parseTopic so that PublishJSON
// can decode an incoming JSON payload into the concrete Go type T and deliver
// it to every Subscribe[T] handler on that topic.
//
// Applications call RegisterTopic once per typed topic they want an agent
// bridge (or any JSON source) to drive.  Re-registration replaces the
// previous codec for the topic.
//
// Example:
//
//	events.RegisterTopic[string]("chat.message")
//	events.RegisterTopic[MyStruct]("app.state")
func RegisterTopic[T any](parseTopic string) {
	parseCodec := topicCodec{
		decode: func(parseRaw json.RawMessage) error {
			var parseValue T
			if parseErr := json.Unmarshal(parseRaw, &parseValue); parseErr != nil {
				return fmt.Errorf("events.PublishJSON: topic %q: cannot unmarshal JSON into %T: %w",
					parseTopic, parseValue, parseErr)
			}
			Publish[T](parseTopic, parseValue)
			return nil
		},
	}
	codecRegistry.Store(parseTopic, parseCodec)
}

// PublishJSON decodes parseRaw and publishes the result on parseTopic.
//
// If a codec has been registered for the topic via RegisterTopic[T], the JSON
// is unmarshalled into T and Publish[T] is called — this reaches every
// Subscribe[T] handler on that topic.
//
// If NO codec is registered, the JSON is decoded into a generic any value
// (numbers become float64, objects become map[string]any, etc.) and
// Publish[any] is called.  This preserves the existing behaviour for
// untyped subscribers but will NOT reach typed Subscribe[T] handlers.
//
// An error is returned when the registered codec's json.Unmarshal step fails
// (wrong JSON shape for T).  JSON decode failures in the untyped fallback
// path are also returned.  A missing topic codec is not an error.
func PublishJSON(parseTopic string, parseRaw json.RawMessage) error {
	parseLoaded, parseFound := codecRegistry.Load(parseTopic)
	if parseFound {
		parseCodec := parseLoaded.(topicCodec)
		return parseCodec.decode(parseRaw)
	}

	// Fallback: decode to generic any and publish untyped.
	var parseGeneric any
	if parseErr := json.Unmarshal(parseRaw, &parseGeneric); parseErr != nil {
		return fmt.Errorf("events.PublishJSON: topic %q: cannot unmarshal JSON: %w", parseTopic, parseErr)
	}
	Publish[any](parseTopic, parseGeneric)
	return nil
}

// RegisteredTopicCodecs returns a sorted list of topic strings that have a
// registered codec.  Agent bridges and devtools use this to discover which
// topics accept a typed JSON publish.
func RegisteredTopicCodecs() []string {
	parseTopics := make([]string, 0)
	codecRegistry.Range(func(parseKey, _ any) bool {
		if parseTopic, parseOk := parseKey.(string); parseOk {
			parseTopics = append(parseTopics, parseTopic)
		}
		return true
	})
	sort.Strings(parseTopics)
	return parseTopics
}

// clearCodecRegistry removes all registered codecs.  It is used by tests to
// avoid cross-test pollution; it is unexported so it does not appear in the
// public API.
func clearCodecRegistry() {
	codecRegistry.Range(func(parseKey, _ any) bool {
		codecRegistry.Delete(parseKey)
		return true
	})
}

// codecMu is a package-level mutex used by tests that need to synchronise
// access to clearCodecRegistry across goroutines.
var codecMu sync.Mutex
