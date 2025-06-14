# Debug Logging for GoWebComponents Fiber System

This document describes the comprehensive debug logging system added to the GoWebComponents fiber architecture for advanced diagnostics and troubleshooting.

## Debug Namespaces

The logging system uses namespaces to categorize different types of debug information:

### 🎯 HOOKS
Tracks hook lifecycle, state changes, and validation:
- Hook initialization and cleanup
- State getter/setter operations with timing
- Effect dependency tracking and execution
- Memo computation and cache hits/misses
- Hook order validation and errors
- Memory diagnostics for hook operations

### 🔄 FIBER  
Tracks fiber rendering lifecycle and performance:
- Component rendering timing and memory usage
- Work unit processing and scheduling
- DOM tree traversal and reconciliation
- Component function execution timing
- Fiber creation, updates, and deletions
- Next unit of work determination

### 💾 COMMIT
Tracks the commit phase operations:
- DOM manipulation timing and memory impact
- Effect execution with individual timing
- Deletion cleanup and memory release
- Update operations and performance metrics
- Memory pressure monitoring

### ♻️ MEMORY
Tracks memory management and optimization:
- Object pool usage and efficiency
- Hook allocation and recycling
- Element pool operations
- Memory cleanup and garbage collection
- Callback management and cleanup

### 🔧 UTILS
Tracks utility operations and scheduling:
- UI queue processing and timing
- Scheduler section entry/exit
- Equality checking operations
- Background task coordination

## Enabling Debug Logging

```go
// Enable all debug logging globally
fiber.EnableAllDebug()

// Enable specific namespaces
fiber.SetDebugNamespace("HOOKS", true)
fiber.SetDebugNamespace("FIBER", true)
fiber.SetDebugNamespace("COMMIT", true)
fiber.SetDebugNamespace("MEMORY", true)
fiber.SetDebugNamespace("UTILS", true)

// Enable multiple namespaces at once
fiber.SetDebugNamespaces(map[string]bool{
    "HOOKS":  true,
    "FIBER":  true,
    "COMMIT": true,
    "MEMORY": true,
    "UTILS":  true,
})

// Check current debug status
status := fiber.GetDebugStatus()
fmt.Printf("Debug status: %+v\n", status)

// Disable all debug logging
fiber.DisableAllDebug()
```

## Key Debug Features

### Performance Monitoring
- **Render Timing**: Complete render cycle timing from start to DOM commit
- **Component Timing**: Individual component function execution timing  
- **Hook Timing**: State updates, effect execution, and memo computation timing
- **Memory Tracking**: Before/after memory usage for major operations
- **Work Unit Metrics**: Processing speed and yielding behavior

### Memory Diagnostics
- **Pool Efficiency**: Object pool hit rates and memory savings
- **Memory Pressure**: Automatic cleanup when limits exceeded
- **Callback Tracking**: JavaScript callback lifecycle and cleanup
- **Leak Detection**: Tracking of unreleased resources
- **GC Monitoring**: Garbage collection triggering and effectiveness

### State Management
- **Hook Order**: Validation of consistent hook calling patterns
- **State Changes**: Before/after values with change detection
- **Effect Dependencies**: Dependency comparison and execution decisions
- **Memo Cache**: Cache hits/misses and computation savings

### Error Detection
- **Hook Violations**: Rules of hooks enforcement with detailed errors
- **Memory Leaks**: Detection of unreleased callbacks and objects
- **Invalid Operations**: Null pointer and type assertion failures
- **Reconciliation Issues**: Problems with fiber tree construction

## Sample Debug Output

```
[FIBER] 🎯 render called with element type: func, container: object
[FIBER] 📊 render: total renders so far: 1
[HOOKS] 🎯 GoUseState called with type string, value: "initial"
[HOOKS] 📍 GoUseState: hook position 0, total hooks so far: 1
[HOOKS] 💾 GoUseState: heap objects: 1234, allocs: 5678, total alloc: 890 KB
[FIBER] 🔄 performUnitOfWork: processing fiber 0x123456 (type: func)
[FIBER] 💾 performUnitOfWork: memory - heap objects: 1234, allocs: 5678, sys: 2048 KB
[FIBER] ⚡ performUnitOfWork: component function completed in 125.5µs
[COMMIT] 🎯 commitRoot: starting commit phase
[COMMIT] 💾 commitRoot: memory before - heap: 1024 KB, objects: 1234
[COMMIT] ✅ commitRoot: commit cycle completed in 2.5ms
[MEMORY] ♻️ getHooksFromPool: reusing hooks 0x789abc from pool
```

## Best Practices

### Development
- Enable all namespaces during development for comprehensive debugging
- Use timing information to identify performance bottlenecks
- Monitor memory usage patterns to detect leaks

### Production
- Disable debug logging or enable only specific namespaces
- Use memory pressure monitoring to trigger cleanup
- Enable error-only logging for critical issues

### Performance Analysis
- Compare render times before and after optimizations
- Track hook efficiency and unnecessary re-computations
- Monitor memory pool effectiveness

## Configuration

The debug system is designed to have minimal performance impact when disabled and provides granular control over what information is logged. All debug output uses emoji prefixes for easy visual parsing and includes timing/memory information where relevant. 