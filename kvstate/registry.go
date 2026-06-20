package kvstate

import "sync"

// A named registry lets a user define a custom strategy/codec/backend once and
// reuse it across the app by name (mirrors flags.BuildSet). Lookups are by the
// helper functions below; Options still accept the concrete value directly.

var (
	registryMu       sync.RWMutex
	strategyRegistry = map[string]WriteStrategy{}
	codecRegistry    = map[string]Codec{}
	backendRegistry  = map[string]PersistenceBackend{}
	resolverRegistry = map[string]ConflictResolver{}
)

// RegisterStrategy registers a WriteStrategy under name.
func RegisterStrategy(parseName string, parseStrategy WriteStrategy) {
	registryMu.Lock()
	defer registryMu.Unlock()
	strategyRegistry[parseName] = parseStrategy
}

// StrategyByName returns a registered WriteStrategy.
func StrategyByName(parseName string) (WriteStrategy, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	parseV, parseOK := strategyRegistry[parseName]
	return parseV, parseOK
}

// RegisterCodec registers a Codec under name.
func RegisterCodec(parseName string, parseCodec Codec) {
	registryMu.Lock()
	defer registryMu.Unlock()
	codecRegistry[parseName] = parseCodec
}

// CodecByName returns a registered Codec.
func CodecByName(parseName string) (Codec, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	parseV, parseOK := codecRegistry[parseName]
	return parseV, parseOK
}

// RegisterBackend registers a PersistenceBackend under name.
func RegisterBackend(parseName string, parseBackend PersistenceBackend) {
	registryMu.Lock()
	defer registryMu.Unlock()
	backendRegistry[parseName] = parseBackend
}

// BackendByName returns a registered PersistenceBackend.
func BackendByName(parseName string) (PersistenceBackend, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	parseV, parseOK := backendRegistry[parseName]
	return parseV, parseOK
}

// RegisterResolver registers a ConflictResolver under name.
func RegisterResolver(parseName string, parseResolver ConflictResolver) {
	registryMu.Lock()
	defer registryMu.Unlock()
	resolverRegistry[parseName] = parseResolver
}

// ResolverByName returns a registered ConflictResolver.
func ResolverByName(parseName string) (ConflictResolver, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	parseV, parseOK := resolverRegistry[parseName]
	return parseV, parseOK
}
