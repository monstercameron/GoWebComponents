package events

import "sort"

// TopicInfo describes one live pub/sub topic for agent/devtools introspection.
type TopicInfo struct {
	// Topic is the topic string.
	Topic string `json:"topic"`
	// Subscribers is the number of handlers currently registered on the topic.
	// Note this counts handlers of every element type T; a Publish[T] only
	// delivers to handlers whose registered T matches, so a non-zero count is
	// an upper bound on who will actually receive a given typed publish.
	Subscribers int `json:"subscribers"`
}

// Topics returns every topic the registry currently knows about, sorted by
// topic string. A topic appears once it has been published to or subscribed
// on at least once. This is the discovery surface an agent bridge uses to
// learn the in-app event vocabulary instead of guessing topic names.
func Topics() []TopicInfo {
	parseInfos := make([]TopicInfo, 0)
	globalRegistry.Range(func(parseKey, parseValue any) bool {
		parseTopic, parseOk := parseKey.(string)
		if !parseOk {
			return true
		}
		parseState, parseStateOk := parseValue.(*topicState)
		if !parseStateOk {
			return true
		}
		parseState.mu.Lock()
		parseCount := len(parseState.entries)
		parseState.mu.Unlock()
		parseInfos = append(parseInfos, TopicInfo{Topic: parseTopic, Subscribers: parseCount})
		return true
	})
	sort.Slice(parseInfos, func(parseA, parseB int) bool {
		return parseInfos[parseA].Topic < parseInfos[parseB].Topic
	})
	return parseInfos
}

// SubscriberCount returns the number of handlers currently registered on a
// topic, across all element types. It is the exported counterpart of the
// package-internal test helper.
func SubscriberCount(parseTopic string) int {
	return subscriberCount(parseTopic)
}
