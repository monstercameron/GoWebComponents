## Core WASM & DOM Features
- [x] explore wasm
- [x] embed wasm on the page
- [x] create a wasm module
- [x] interact with the DOM from wasm
- [x] add and remove elements from the DOM
- [x] add event listeners to elements
- [x] add styles to elements
- [x] add classes to elements
- [x] add ids to elements
- [x] add reactive data binding to elements

## React Hooks (Implemented)
- [x] useState (with getter/setter pattern)
- [x] useEffect
- [x] useMemo

## React Hooks (Missing)
- [ ] useCallback
- [ ] useRef
- [ ] useReducer
- [ ] useContext
- [ ] useLayoutEffect
- [ ] useId
- [ ] useDeferredValue
- [ ] useTransition
- [ ] useSyncExternalStore
- [ ] useInsertionEffect

## Core React Features (Missing)
- [ ] Fragment support (React.Fragment / <>)
- [ ] Key prop support for list reconciliation
- [ ] Ref forwarding
- [ ] Error boundaries
- [ ] Suspense for async components
- [ ] Portals (render to different DOM nodes)
- [ ] React.memo equivalent for component memoization
- [ ] Context API (Provider/Consumer pattern)
- [ ] Strict mode equivalent
- [ ] Profiler for performance monitoring

## Component Patterns (Missing)
- [ ] Higher-order components (HOCs)
- [ ] Render props pattern
- [ ] Compound components
- [ ] Custom hooks support

## Performance & Developer Experience
- [ ] fix useEffect edge cases
- [x] add useMemo (already implemented)
- [ ] revise useState to not return a getter func
- [ ] add comprehensive docs
- [ ] check for low hanging perf optimizations
- [ ] abstract away jsfunc with useFunc hook
- [ ] Component dev tools integration
- [ ] Hot reloading support
- [ ] Better error messages and stack traces

## Advanced Features
- [ ] Server-side rendering (SSR) equivalent
- [ ] Code splitting support
- [ ] Lazy loading components
- [ ] Concurrent rendering features
- [ ] Time slicing for better performance
- [ ] Automatic batching of state updates

## Testing & Debugging
- [ ] Testing utilities (render, fireEvent, waitFor)
- [ ] Mock hooks for testing
- [ ] Component inspector/debugger
- [ ] Performance profiling tools

## Fetch & Async (Implemented)
- [x] useFetch hook
- [x] GoFetch with channels
- [x] Fetch with callbacks

## Missing HTTP Features
- [ ] useMutation for POST/PUT/DELETE operations
- [ ] Request caching and deduplication
- [ ] Request cancellation support
- [ ] Retry logic for failed requests
- [ ] Optimistic updates pattern