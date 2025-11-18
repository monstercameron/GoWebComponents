# Counter Example

**Level:** Beginner  
**Concepts:** Basic state management, event handling

## What This Example Demonstrates

- Using `hooks.UseState` for number state
- Button click handlers with `js.FuncOf`
- Simple increment/decrement operations
- State updates triggering re-renders
- Basic Tailwind CSS styling

## Key Code

```go
count, setCount := hooks.UseState(0)
currentCount := count()

increment := func(this js.Value, args []js.Value) interface{} {
    setCount(currentCount + 1)
    return nil
}
```

## Learning Objectives

1. Understand how to create and update state in GoWebComponents
2. Learn to attach event handlers to DOM elements
3. See how state changes trigger component re-renders
4. Build interactive UI with functional programming patterns

## Running

```bash
cd examples
./build.ps1 -Example 01-counter
```

Then open `01-counter/counter.html` in a web server.
