package fetch

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// QueryOptions configures the ergonomic query wrapper around UseCachedResource.
type QueryOptions struct {
	Cache CacheOptions
	Tags  []string
}

// Query wraps a cached resource with query metadata such as tags.
type Query[T any] struct {
	resource CachedResource[T]
	key      string
	tags     []string
}

// QueryPageRequest describes one page load for UseInfiniteQuery.
type QueryPageRequest[C any] struct {
	Cursor    C
	PageIndex int
}

// QueryPage is one page of an infinite query result.
type QueryPage[T any, C any] struct {
	Items      []T
	NextCursor C
	HasNext    bool
}

// InfiniteQueryData is the cached value stored for an infinite query.
type InfiniteQueryData[T any, C any] struct {
	Pages      []QueryPage[T, C]
	Items      []T
	NextCursor C
	HasNext    bool
}

// InfiniteQueryOptions configures UseInfiniteQuery.
type InfiniteQueryOptions[C any] struct {
	Cache         CacheOptions
	Tags          []string
	InitialCursor C
}

// InfiniteQueryState describes the current state of a paginated query.
type InfiniteQueryState[T any, C any] struct {
	InfiniteQueryData[T, C]
	Loading   bool
	Error     error
	Ready     bool
	Stale     bool
	UpdatedAt time.Time
}

// InfiniteQuery wraps cached paginated data and page-loading helpers.
type InfiniteQuery[T any, C any] struct {
	resource      CachedResource[InfiniteQueryData[T, C]]
	key           string
	tags          []string
	initialCursor C
	loader        func(context.Context, QueryPageRequest[C]) (QueryPage[T, C], error)
}

// OptimisticUpdate can roll back one optimistic cache write.
type OptimisticUpdate[T any] struct {
	key      string
	previous cachedResourceSnapshot
	active   bool
}

type queryTagEntry struct {
	mu   sync.Mutex
	keys map[string]struct{}
}

var queryTagIndex sync.Map

// UseQuery is a tag-aware wrapper around UseCachedResource.
func UseQuery[T any](parseKey string, parseLoader func(context.Context) (T, error), parseOptions ...QueryOptions) Query[T] {
	parseResolved := resolveQueryOptions(parseOptions)
	parseTags := registerQueryTags(parseKey, parseResolved.Tags)
	return Query[T]{
		resource: UseCachedResource(parseKey, parseLoader, parseResolved.Cache),
		key:      parseKey,
		tags:     parseTags,
	}
}

// LoadQuery reuses the shared cache from imperative code and registers query tags.
func LoadQuery[T any](parseCtx context.Context, parseKey string, parseLoader func(context.Context) (T, error), parseOptions ...QueryOptions) (T, error) {
	parseResolved := resolveQueryOptions(parseOptions)
	registerQueryTags(parseKey, parseResolved.Tags)
	return LoadCached(parseCtx, parseKey, parseLoader, parseResolved.Cache)
}

// Get returns the current query state.
func (parseQ Query[T]) Get() CachedResourceState[T] {
	return parseQ.resource.Get()
}

// Reload starts a new query load.
func (parseQ Query[T]) Reload() {
	parseQ.resource.Reload()
}

// Cancel cancels the active query load, if any.
func (parseQ Query[T]) Cancel() {
	parseQ.resource.Cancel()
}

// Invalidate marks the query stale and eligible for revalidation.
func (parseQ Query[T]) Invalidate() {
	parseQ.resource.Invalidate()
}

// Dispose clears the query cache entry.
func (parseQ Query[T]) Dispose() {
	parseQ.resource.Dispose()
}

// Set replaces the query value optimistically.
func (parseQ Query[T]) Set(parseValue T) {
	parseQ.resource.Set(parseValue)
}

// Update replaces the query value using the previous value.
func (parseQ Query[T]) Update(parseFn func(T) T) {
	parseQ.resource.Update(parseFn)
}

// OptimisticUpdate applies an optimistic query update and returns a rollback handle.
func (parseQ Query[T]) OptimisticUpdate(parseFn func(T) T) OptimisticUpdate[T] {
	return parseQ.resource.OptimisticUpdate(parseFn)
}

// CacheKey returns the stable cache key behind this query.
func (parseQ Query[T]) CacheKey() string {
	return parseQ.key
}

// Tags returns the normalized tags registered for this query.
func (parseQ Query[T]) Tags() []string {
	return cloneStringSlice(parseQ.tags)
}

// UseInfiniteQuery caches the first page and exposes LoadNext for pagination.
func UseInfiniteQuery[T any, C any](parseKey string, parseLoader func(context.Context, QueryPageRequest[C]) (QueryPage[T, C], error), parseOptions ...InfiniteQueryOptions[C]) InfiniteQuery[T, C] {
	parseResolved := resolveInfiniteQueryOptions(parseOptions)
	parseTags := registerQueryTags(parseKey, parseResolved.Tags)
	parseResource := UseCachedResource(parseKey, func(parseCtx context.Context) (InfiniteQueryData[T, C], error) {
		if parseLoader == nil {
			var parseZero InfiniteQueryData[T, C]
			return parseZero, fmt.Errorf("fetch: infinite query loader cannot be nil")
		}
		parsePage, parseErr := parseLoader(parseCtx, QueryPageRequest[C]{Cursor: parseResolved.InitialCursor, PageIndex: 0})
		if parseErr != nil {
			var parseZero InfiniteQueryData[T, C]
			return parseZero, parseErr
		}
		return buildInfiniteQueryData([]QueryPage[T, C]{parsePage}), nil
	}, parseResolved.Cache)

	return InfiniteQuery[T, C]{
		resource:      parseResource,
		key:           parseKey,
		tags:          parseTags,
		initialCursor: parseResolved.InitialCursor,
		loader:        parseLoader,
	}
}

// Get returns the current infinite query state.
func (parseQ InfiniteQuery[T, C]) Get() InfiniteQueryState[T, C] {
	parseState := parseQ.resource.Get()
	return InfiniteQueryState[T, C]{
		InfiniteQueryData: parseState.Value,
		Loading:           parseState.Loading,
		Error:             parseState.Error,
		Ready:             parseState.Ready,
		Stale:             parseState.Stale,
		UpdatedAt:         parseState.UpdatedAt,
	}
}

// Reload reloads the first page and replaces currently cached pages.
func (parseQ InfiniteQuery[T, C]) Reload() {
	parseQ.resource.Reload()
}

// LoadNext loads and appends the next page when HasNext is true.
func (parseQ InfiniteQuery[T, C]) LoadNext() {
	if parseQ.key == "" || parseQ.loader == nil {
		return
	}

	parseState := parseQ.resource.Get()
	if !parseState.Ready || parseState.Loading || !parseState.Value.HasNext {
		return
	}

	parseCursor := parseState.Value.NextCursor
	parsePageIndex := len(parseState.Value.Pages)
	updateCachedSnapshot(parseQ.key, func(parsePrev cachedResourceSnapshot) cachedResourceSnapshot {
		parsePrev.Loading = true
		parsePrev.Error = nil
		if parsePrev.Ready {
			parsePrev.Stale = true
		}
		return parsePrev
	})

	go func() {
		parsePage, parseErr := parseQ.loader(context.Background(), QueryPageRequest[C]{
			Cursor:    parseCursor,
			PageIndex: parsePageIndex,
		})
		if parseErr != nil {
			updateCachedSnapshot(parseQ.key, func(parsePrev cachedResourceSnapshot) cachedResourceSnapshot {
				parsePrev.Loading = false
				parsePrev.Error = parseErr
				parsePrev.Stale = parsePrev.Ready
				return parsePrev
			})
			return
		}

		updateCachedSnapshot(parseQ.key, func(parsePrev cachedResourceSnapshot) cachedResourceSnapshot {
			parseCurrent, _ := castCachedValue[InfiniteQueryData[T, C]](parsePrev.Value)
			parsePrev.Value = appendInfiniteQueryPage(parseCurrent, parsePage)
			parsePrev.Loading = false
			parsePrev.Error = nil
			parsePrev.Ready = true
			parsePrev.Stale = false
			parsePrev.UpdatedAt = time.Now()
			return parsePrev
		})
		markCachedEntryFresh(parseQ.key)
		persistCachedSnapshot(parseQ.key)
	}()
}

// Cancel cancels the active first-page query load, if any.
func (parseQ InfiniteQuery[T, C]) Cancel() {
	parseQ.resource.Cancel()
}

// Invalidate marks the infinite query stale and eligible for first-page revalidation.
func (parseQ InfiniteQuery[T, C]) Invalidate() {
	parseQ.resource.Invalidate()
}

// Dispose clears the infinite query cache entry.
func (parseQ InfiniteQuery[T, C]) Dispose() {
	parseQ.resource.Dispose()
}

// Set replaces the complete infinite query data optimistically.
func (parseQ InfiniteQuery[T, C]) Set(parseValue InfiniteQueryData[T, C]) {
	parseQ.resource.Set(parseValue)
}

// Update replaces the complete infinite query data using the previous value.
func (parseQ InfiniteQuery[T, C]) Update(parseFn func(InfiniteQueryData[T, C]) InfiniteQueryData[T, C]) {
	parseQ.resource.Update(parseFn)
}

// OptimisticUpdate applies an optimistic infinite-query update and returns a rollback handle.
func (parseQ InfiniteQuery[T, C]) OptimisticUpdate(parseFn func(InfiniteQueryData[T, C]) InfiniteQueryData[T, C]) OptimisticUpdate[InfiniteQueryData[T, C]] {
	return parseQ.resource.OptimisticUpdate(parseFn)
}

// CacheKey returns the stable cache key behind this infinite query.
func (parseQ InfiniteQuery[T, C]) CacheKey() string {
	return parseQ.key
}

// Tags returns the normalized tags registered for this infinite query.
func (parseQ InfiniteQuery[T, C]) Tags() []string {
	return cloneStringSlice(parseQ.tags)
}

// OptimisticUpdate applies an optimistic resource update and returns a rollback handle.
func (parseR CachedResource[T]) OptimisticUpdate(parseFn func(T) T) OptimisticUpdate[T] {
	return ApplyOptimisticUpdate(parseR.key, parseFn)
}

// ApplyOptimisticUpdate updates a cache entry and returns a rollback handle.
func ApplyOptimisticUpdate[T any](parseKey string, parseFn func(T) T) OptimisticUpdate[T] {
	if parseKey == "" || parseFn == nil {
		return OptimisticUpdate[T]{}
	}

	cancelCachedLoad(parseKey)
	parsePrevious := currentCachedSnapshot(parseKey)
	parseCurrent, _ := castCachedValue[T](parsePrevious.Value)
	updateCachedSnapshot(parseKey, func(parsePrev cachedResourceSnapshot) cachedResourceSnapshot {
		parsePrev.Value = parseFn(parseCurrent)
		parsePrev.Loading = false
		parsePrev.Error = nil
		parsePrev.Ready = true
		parsePrev.Stale = false
		parsePrev.UpdatedAt = time.Now()
		return parsePrev
	})
	markCachedEntryFresh(parseKey)
	persistCachedSnapshot(parseKey)

	return OptimisticUpdate[T]{
		key:      parseKey,
		previous: parsePrevious,
		active:   true,
	}
}

// Active reports whether the optimistic update can still be committed or rolled back.
func (parseU OptimisticUpdate[T]) Active() bool {
	return parseU.active
}

// Commit makes the optimistic update final and disables rollback.
func (parseU *OptimisticUpdate[T]) Commit() {
	if parseU != nil {
		parseU.active = false
	}
}

// Rollback restores the cache snapshot captured before the optimistic update.
func (parseU *OptimisticUpdate[T]) Rollback() {
	if parseU == nil || !parseU.active || parseU.key == "" {
		return
	}
	updateCachedSnapshot(parseU.key, func(cachedResourceSnapshot) cachedResourceSnapshot {
		return parseU.previous
	})
	restoreCachedEntryAfterOptimisticRollback(parseU.key, parseU.previous)
	persistCachedSnapshot(parseU.key)
	parseU.active = false
}

// InvalidateQueryTag marks all live queries with tag stale.
func InvalidateQueryTag(parseTag string) int {
	return invalidateQueryKeys(QueryKeysForTag(parseTag))
}

// InvalidateQueryTags marks all live queries with any supplied tag stale.
func InvalidateQueryTags(parseTags ...string) int {
	return invalidateQueryKeys(queryKeysForTags(parseTags))
}

// DisposeQueryTag clears all live cache entries registered for tag.
func DisposeQueryTag(parseTag string) int {
	parseDisposed := 0
	for _, parseKey := range QueryKeysForTag(parseTag) {
		if _, parseOk := cachedResourceRegistry.Load(parseKey); !parseOk {
			continue
		}
		DisposeResource(parseKey)
		parseDisposed++
	}
	return parseDisposed
}

// QueryKeysForTag returns sorted cache keys registered for a tag.
func QueryKeysForTag(parseTag string) []string {
	parseNormalized := normalizeQueryTag(parseTag)
	if parseNormalized == "" {
		return nil
	}
	parseRaw, parseOk := queryTagIndex.Load(parseNormalized)
	if !parseOk {
		return nil
	}
	parseEntry, _ := parseRaw.(*queryTagEntry)
	if parseEntry == nil {
		return nil
	}

	parseEntry.mu.Lock()
	defer parseEntry.mu.Unlock()
	parseKeys := make([]string, 0, len(parseEntry.keys))
	for parseKey := range parseEntry.keys {
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)
	return parseKeys
}

// QueryTagsForKey returns sorted tags registered for one cache key.
func QueryTagsForKey(parseKey string) []string {
	if parseKey == "" {
		return nil
	}
	parseTags := make([]string, 0)
	queryTagIndex.Range(func(parseTag, parseValue any) bool {
		parseTagString, _ := parseTag.(string)
		parseEntry, _ := parseValue.(*queryTagEntry)
		if parseTagString == "" || parseEntry == nil {
			return true
		}
		parseEntry.mu.Lock()
		_, parseHasKey := parseEntry.keys[parseKey]
		parseEntry.mu.Unlock()
		if parseHasKey {
			parseTags = append(parseTags, parseTagString)
		}
		return true
	})
	sort.Strings(parseTags)
	return parseTags
}

func resolveQueryOptions(parseOptions []QueryOptions) QueryOptions {
	if len(parseOptions) == 0 {
		return QueryOptions{}
	}
	return parseOptions[0]
}

func resolveInfiniteQueryOptions[C any](parseOptions []InfiniteQueryOptions[C]) InfiniteQueryOptions[C] {
	if len(parseOptions) == 0 {
		return InfiniteQueryOptions[C]{}
	}
	return parseOptions[0]
}

func registerQueryTags(parseKey string, parseTags []string) []string {
	if parseKey == "" {
		return nil
	}
	parseNormalized := normalizeQueryTags(parseTags)
	for _, parseTag := range parseNormalized {
		parseRaw, _ := queryTagIndex.LoadOrStore(parseTag, &queryTagEntry{keys: map[string]struct{}{}})
		parseEntry := parseRaw.(*queryTagEntry)
		parseEntry.mu.Lock()
		if parseEntry.keys == nil {
			parseEntry.keys = map[string]struct{}{}
		}
		parseEntry.keys[parseKey] = struct{}{}
		parseEntry.mu.Unlock()
	}
	return parseNormalized
}

func unregisterQueryKey(parseKey string) {
	if parseKey == "" {
		return
	}
	queryTagIndex.Range(func(parseTag, parseValue any) bool {
		parseEntry, _ := parseValue.(*queryTagEntry)
		if parseEntry == nil {
			return true
		}
		parseEntry.mu.Lock()
		delete(parseEntry.keys, parseKey)
		parseEmpty := len(parseEntry.keys) == 0
		parseEntry.mu.Unlock()
		if parseEmpty {
			queryTagIndex.Delete(parseTag)
		}
		return true
	})
}

func normalizeQueryTags(parseTags []string) []string {
	if len(parseTags) == 0 {
		return nil
	}
	parseSeen := make(map[string]struct{}, len(parseTags))
	parseOut := make([]string, 0, len(parseTags))
	for _, parseTag := range parseTags {
		parseNormalized := normalizeQueryTag(parseTag)
		if parseNormalized == "" {
			continue
		}
		if _, parseExists := parseSeen[parseNormalized]; parseExists {
			continue
		}
		parseSeen[parseNormalized] = struct{}{}
		parseOut = append(parseOut, parseNormalized)
	}
	sort.Strings(parseOut)
	return parseOut
}

func normalizeQueryTag(parseTag string) string {
	return strings.TrimSpace(parseTag)
}

func queryKeysForTags(parseTags []string) []string {
	parseSeen := map[string]struct{}{}
	for _, parseTag := range normalizeQueryTags(parseTags) {
		for _, parseKey := range QueryKeysForTag(parseTag) {
			parseSeen[parseKey] = struct{}{}
		}
	}
	parseKeys := make([]string, 0, len(parseSeen))
	for parseKey := range parseSeen {
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)
	return parseKeys
}

func invalidateQueryKeys(parseKeys []string) int {
	parseInvalidated := 0
	for _, parseKey := range parseKeys {
		if _, parseOk := cachedResourceRegistry.Load(parseKey); !parseOk {
			continue
		}
		InvalidateResource(parseKey)
		parseInvalidated++
	}
	return parseInvalidated
}

func buildInfiniteQueryData[T any, C any](parsePages []QueryPage[T, C]) InfiniteQueryData[T, C] {
	var parseData InfiniteQueryData[T, C]
	for _, parsePage := range parsePages {
		parseData = appendInfiniteQueryPage(parseData, parsePage)
	}
	return parseData
}

func appendInfiniteQueryPage[T any, C any](parseData InfiniteQueryData[T, C], parsePage QueryPage[T, C]) InfiniteQueryData[T, C] {
	parseItems := cloneSlice(parseData.Items)
	parseItems = append(parseItems, parsePage.Items...)
	parsePages := append([]QueryPage[T, C](nil), parseData.Pages...)
	parsePages = append(parsePages, QueryPage[T, C]{
		Items:      cloneSlice(parsePage.Items),
		NextCursor: parsePage.NextCursor,
		HasNext:    parsePage.HasNext,
	})
	parseData.Pages = parsePages
	parseData.Items = parseItems
	parseData.NextCursor = parsePage.NextCursor
	parseData.HasNext = parsePage.HasNext
	return parseData
}

func restoreCachedEntryAfterOptimisticRollback(parseKey string, parseSnapshot cachedResourceSnapshot) {
	parseRaw, parseOk := cachedResourceRegistry.Load(parseKey)
	if !parseOk {
		return
	}
	parseEntry, _ := parseRaw.(*cachedResourceEntry)
	if parseEntry == nil {
		return
	}
	parseEntry.mu.Lock()
	parseEntry.invalidated = parseSnapshot.Stale
	parseEntry.pending = false
	parseEntry.lastAccess = time.Now()
	if !parseSnapshot.UpdatedAt.IsZero() {
		parseEntry.lastLoaded = parseSnapshot.UpdatedAt
	}
	parseEntry.mu.Unlock()
}

func cloneSlice[T any](parseIn []T) []T {
	if len(parseIn) == 0 {
		return nil
	}
	parseOut := make([]T, len(parseIn))
	copy(parseOut, parseIn)
	return parseOut
}

func cloneStringSlice(parseIn []string) []string {
	if len(parseIn) == 0 {
		return nil
	}
	parseOut := make([]string, len(parseIn))
	copy(parseOut, parseIn)
	return parseOut
}
