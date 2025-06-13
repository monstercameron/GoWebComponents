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
- [x] implement virtual DOM reconciliation
- [x] implement object pooling for performance
- [x] implement fast equality checks

## React Hooks (Implemented)
- [x] useState (with getter/setter pattern)
- [x] useEffect
- [x] useMemo
- [x] useFetch (with multiple patterns)

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
- [x] implement object pooling for fibers
- [x] implement object pooling for hooks
- [x] implement object pooling for elements
- [x] implement fast equality checks
- [x] optimize DOM updates
- [x] implement debounced input handling
- [ ] fix useEffect edge cases
- [x] add useMemo (already implemented)
- [ ] revise useState to not return a getter func
- [ ] add comprehensive docs
- [ ] check for low hanging perf optimizations
- [x] abstract away jsfunc with useFunc hook
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
- [ ] Implement fiber tree optimization
- [ ] Add fiber tree visualization tools
- [ ] Implement fiber tree debugging tools

## Testing & Debugging
- [ ] Testing utilities (render, fireEvent, waitFor)
- [ ] Mock hooks for testing
- [ ] Component inspector/debugger
- [ ] Performance profiling tools
- [ ] Fiber tree visualization
- [ ] Hook debugging tools

## Fetch & Async (Implemented)
- [x] useFetch hook
- [x] GoFetch with channels
- [x] Fetch with callbacks
- [x] Fetch state management
- [x] Fetch error handling

## Missing HTTP Features
- [ ] useMutation for POST/PUT/DELETE operations
- [ ] Request caching and deduplication
- [ ] Request cancellation support
- [ ] Retry logic for failed requests
- [ ] Optimistic updates pattern

## Memory Management
- [x] Implement object pooling for fibers
- [x] Implement object pooling for hooks
- [x] Implement object pooling for elements
- [x] Implement object pooling for props
- [x] Implement object pooling for children
- [ ] Add memory leak detection
- [ ] Add memory usage monitoring
- [ ] Implement garbage collection hints

## Build & Development
- [ ] Fix syscall/js build constraints
- [ ] Add proper build configuration
- [ ] Add development mode
- [ ] Add production mode
- [ ] Add source maps support
- [ ] Add proper error handling for build issues