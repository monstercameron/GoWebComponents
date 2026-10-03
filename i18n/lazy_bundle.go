package i18n

import (
	"fmt"
	"sync"
)

// LocaleLoader loads the full Catalog for one locale on demand — e.g. fetching a locale JSON
// bundle in the browser (via interop.ImportModule or fetch) or reading an embedded file on the
// server. It is called at most once per locale by LazyBundle.
type LocaleLoader func(parseLocale string) (Catalog, error)

// LazyBundle wraps a Bundle with on-demand locale loading: a locale's catalog is fetched and
// registered the first time it is needed, not all up front. The application supplies the
// transport (the LocaleLoader); LazyBundle owns the load-once orchestration — dedup + register —
// so a locale is never registered twice and concurrent callers share one registration. This is
// the "lazy locale loading" path: ship only the active locale's strings on first paint and pull
// other locales when the user actually switches.
type LazyBundle struct {
	bundle *Bundle
	load   LocaleLoader
	mu     sync.Mutex
	loaded map[string]bool
	// inflight tracks locales whose loader is running, so concurrent callers share one load.
	inflight map[string]*lazyLoad
}

// lazyLoad is one in-progress locale load that concurrent EnsureLocale callers wait on.
type lazyLoad struct {
	done chan struct{}
	err  error
}

// NewLazyBundle wraps bundle with the given on-demand loader. Any locales already registered on
// bundle (e.g. the initial/SSR locale) are treated as pre-loaded and never re-fetched.
func NewLazyBundle(parseBundle *Bundle, parseLoad LocaleLoader) *LazyBundle {
	parseLoaded := map[string]bool{}
	if parseBundle != nil {
		for _, parseLocale := range parseBundle.Locales() {
			parseLoaded[NormalizeLocale(parseLocale)] = true
		}
	}
	return &LazyBundle{bundle: parseBundle, load: parseLoad, loaded: parseLoaded}
}

// Bundle returns the underlying Bundle for Provider wiring, Translate, formatting, etc.
func (parseL *LazyBundle) Bundle() *Bundle { return parseL.bundle }

// Loaded reports whether a locale's catalog has been loaded and registered.
func (parseL *LazyBundle) Loaded(parseLocale string) bool {
	parseKey := NormalizeLocale(parseLocale)
	parseL.mu.Lock()
	defer parseL.mu.Unlock()
	return parseL.loaded[parseKey]
}

// EnsureLocale loads and registers a locale's catalog if not already present, exactly once.
// Call it before switching to a locale (e.g. in a route guard or before Runtime.SetLocale) so the
// strings are present when the UI renders. It is idempotent and concurrency-safe; an
// already-loaded locale, a nil loader, or a nil bundle is a no-op returning nil. A loader error is
// returned and the locale stays unloaded so a later call can retry.
func (parseL *LazyBundle) EnsureLocale(parseLocale string) error {
	parseKey := NormalizeLocale(parseLocale)
	if parseKey == "" {
		return nil
	}
	parseL.mu.Lock()
	if parseL.loaded[parseKey] || parseL.load == nil || parseL.bundle == nil {
		parseL.mu.Unlock()
		return nil
	}
	// Join a load already running for this locale: the LocaleLoader contract is "at most once",
	// and a second concurrent fetch would waste a request and race the registration.
	if parseRunning, parseIsRunning := parseL.inflight[parseKey]; parseIsRunning {
		parseL.mu.Unlock()
		<-parseRunning.done
		return parseRunning.err
	}
	parseCall := &lazyLoad{done: make(chan struct{})}
	if parseL.inflight == nil {
		parseL.inflight = map[string]*lazyLoad{}
	}
	parseL.inflight[parseKey] = parseCall
	parseL.mu.Unlock()

	// The deferred finish also runs when the loader panics, so waiters are always released and
	// the locale can be retried.
	parseCall.err = fmt.Errorf("i18n: loader for %q did not complete", parseKey)
	defer func() {
		parseL.mu.Lock()
		delete(parseL.inflight, parseKey)
		parseL.mu.Unlock()
		close(parseCall.done)
	}()
	parseCatalog, parseErr := parseL.load(parseKey)
	if parseErr != nil {
		parseCall.err = parseErr
		return parseErr
	}

	parseL.mu.Lock()
	parseL.bundle.Register(parseKey, parseCatalog)
	parseL.loaded[parseKey] = true
	parseL.mu.Unlock()
	parseCall.err = nil
	return nil
}
