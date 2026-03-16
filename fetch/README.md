# Fetch Package

**Location:** `/fetch`

```
GoWebComponents/
├── dom/
├── hooks/
├── state/
├── render/
├── router/
├── fetch/            ← YOU ARE HERE
│   ├── doc.go
│   └── fetch.go
├── internal/
├── examples/
└── ...
```

## Overview

The `fetch` package provides utilities for making HTTP requests from WebAssembly applications. It offers both low-level fetch hooks and higher-level typed async resources.

## Which API to Use

- Use `UseFetch` when you want raw fetch state around a URL and are comfortable parsing `state.Data` yourself.
- Use `UseResource[T]` when you want typed values, cancellation, dependency-driven reloads, or loader logic that does more than one direct fetch call.
- Use `UseCachedResource[T]` when the same typed query should be shared across components, deduplicated in flight, or updated optimistically before a revalidation.
- Use `Fetch` when you need imperative access from an event handler, goroutine, or other non-hook code.

## Core APIs

### `UseFetch(url string, options ...Options) Resource`

Declarative hook for low-level browser fetch state within components. It manages loading, error, and raw response data, but leaves parsing and higher-level orchestration to the caller.

```go
import "github.com/monstercameron/GoWebComponents/fetch"

func UserProfile(props dom.Attrs) *dom.Element {
    resource := fetch.UseFetch("https://api.example.com/user/1")
    state := resource.Get()

    if state.Loading {
        return dom.P(nil, dom.Text("Loading..."))
    }

    if state.Error != "" {
        return dom.Div(nil,
            dom.P(nil, dom.Text("Error: "+state.Error)),
            dom.Button(dom.Attrs{
                "onclick": hooks.GoUseFunc(func(e dom.GoEvent) {
                    resource.Refetch()
                }),
            }, dom.Text("Retry")),
        )
    }

    userJSON := state.Data.(string)
    return dom.Div(nil,
        dom.Pre(nil, dom.Text(userJSON)),
    )
}
```

### `UseResource[T](loader func(context.Context) (T, error), deps ...interface{})`

Typed async resource hook for non-trivial loading flows.

```go
import (
    "context"

    "github.com/monstercameron/GoWebComponents/fetch"
)

func UserCount(props dom.Attrs) *dom.Element {
    resource := fetch.UseResource(func(ctx context.Context) (int, error) {
        // Replace with real async work.
        // The context is cancelled on unmount, dependency change, or Cancel().
        return 42, nil
    })

    state := resource.Get()
    if state.Loading {
        return dom.P(nil, dom.Text("Loading count..."))
    }
    if state.Error != nil {
        return dom.P(nil, dom.Text("Error: "+state.Error.Error()))
    }
    if !state.Ready {
        return dom.P(nil, dom.Text("Idle"))
    }

    return dom.Div(nil,
        dom.P(nil, dom.Text(fmt.Sprintf("Count: %d", state.Value))),
        dom.Button(dom.Attrs{"onclick": hooks.GoUseFunc(func(e dom.GoEvent) {
            resource.Reload()
        })}, dom.Text("Reload")),
    )
}
```

### `UseCachedResource[T](key string, loader func(context.Context) (T, error), options ...CacheOptions)`

Typed shared-cache hook for list/detail queries that should be reused across components.

```go
users := fetch.UseCachedResource("users", func(ctx context.Context) ([]User, error) {
    return loadUsers(ctx)
}, fetch.CacheOptions{StaleAfter: 30 * time.Second})

state := users.Get()
if state.Loading && !state.Ready {
    return html.P(html.Props{}, html.Text("Loading users..."))
}
if state.Error != nil && !state.Ready {
    return html.P(html.Props{}, html.Text("Users failed to load"))
}

users.Update(func(prev []User) []User {
    next := append([]User(nil), prev...)
    next[0].Name = "Ada (local)"
    return next
})

fetch.InvalidateResource("users")
```

`UseCachedResource[T]` keeps the last ready value visible while background refreshes run, so it composes cleanly with `ui.AsyncBoundary` by using `Pending: state.Loading && !state.Ready` for the first load and showing content during stale revalidation.

### `Fetch(url string, options Options) <-chan Result`

Imperative API for making HTTP requests from anywhere (event handlers, goroutines, etc.).

```go
import "github.com/monstercameron/GoWebComponents/fetch"

func CreateUser(userData map[string]interface{}) {
    resultChan := fetch.Fetch("https://api.example.com/users", fetch.Options{
        Method:  "POST",
        Headers: map[string]interface{}{
            "Content-Type": "application/json",
        },
        Body: userData,
    })

    go func() {
        result := <-resultChan
        fetch.ReturnChannel(resultChan)

        if result.Err != nil {
            fmt.Println("Error:", result.Err)
            return
        }

        fmt.Println("Success:", result.Data)
    }()
}
```

## Data Types

### FetchState

```go
type FetchState struct {
    Data    interface{}  // Response data (parsed JSON)
    Loading bool         // Is request in progress?
    Error   string       // Error message if request failed
}
```

### Result

```go
type Result struct {
    Data interface{}  // Response data
    Err  error        // Error if request failed
}
```

### Options

```go
type Options struct {
    Method  string                 // HTTP method: GET, POST, PUT, DELETE, etc.
    Headers map[string]interface{} // Request headers
    Body    interface{}            // Request body (auto-JSON encoded if struct/map)
}
```

## Usage Patterns

### GET Request (Declarative)

```go
func PostsList(props dom.Attrs) *dom.Element {
    getState, _ := fetch.UseFetch("https://jsonplaceholder.typicode.com/posts")
    state := getState()

    if state.Loading {
        return dom.P(nil, dom.Text("Loading posts..."))
    }

    if state.Error != "" {
        return dom.P(nil, dom.Text("Failed to load posts"))
    }

    posts := state.Data.([]interface{})
    items := make([]interface{}, len(posts))

    for i, post := range posts {
        p := post.(map[string]interface{})
        items[i] = dom.Li(dom.Attrs{"key": p["id"]},
            dom.Text(p["title"].(string)))
    }

    return dom.Ul(nil, items...)
}
```

### POST Request (Imperative)

```go
func CreatePostForm(props dom.Attrs) *dom.Element {
    title, setTitle := hooks.UseState("")
    body, setBody := hooks.UseState("")
    status, setStatus := hooks.UseState("")

    handleSubmit := hooks.GoUseFunc(func(e dom.GoEvent) {
        e.PreventDefault()
        setStatus("Creating...")

        postData := map[string]interface{}{
            "title": title(),
            "body":  body(),
            "userId": 1,
        }

        resultChan := fetch.Fetch("https://jsonplaceholder.typicode.com/posts",
            fetch.Options{
                Method: "POST",
                Headers: map[string]interface{}{
                    "Content-Type": "application/json",
                },
                Body: postData,
            })

        go func() {
            result := <-resultChan

            if result.Err != nil {
                setStatus("Error: " + result.Err.Error())
            } else {
                setStatus("Post created successfully!")
                setTitle("")
                setBody("")
            }
        }()
    })

    return dom.Form(dom.Attrs{"onsubmit": handleSubmit},
        dom.Input(dom.Attrs{
            "value": title(),
            "oninput": hooks.GoUseFunc(func(e dom.GoEvent) {
                setTitle(e.GetValue())
            }),
            "placeholder": "Title",
        }),
        dom.Textarea(dom.Attrs{
            "value": body(),
            "oninput": hooks.GoUseFunc(func(e dom.GoEvent) {
                setBody(e.GetValue())
            }),
            "placeholder": "Body",
        }),
        dom.Button(nil, dom.Text("Create Post")),
        dom.P(nil, dom.Text(status())),
    )
}
```

### PUT/PATCH Request

```go
func UpdateUser(userID int, updates map[string]interface{}) {
    url := fmt.Sprintf("https://api.example.com/users/%d", userID)

    resultChan := fetch.Fetch(url, fetch.Options{
        Method: "PUT",
        Headers: map[string]interface{}{
            "Content-Type": "application/json",
        },
        Body: updates,
    })

    go func() {
        result := <-resultChan

        if result.Err != nil {
            fmt.Println("Update failed:", result.Err)
        } else {
            fmt.Println("User updated:", result.Data)
        }
    }()
}
```

### DELETE Request

```go
func DeletePost(postID int) {
    url := fmt.Sprintf("https://api.example.com/posts/%d", postID)

    resultChan := fetch.Fetch(url, fetch.Options{
        Method: "DELETE",
    })

    go func() {
        result := <-resultChan

        if result.Err != nil {
            fmt.Println("Delete failed:", result.Err)
        } else {
            fmt.Println("Post deleted successfully")
        }
    }()
}
```

### Custom Headers & Authentication

```go
func AuthenticatedRequest() {
    token := "your-jwt-token"

    resultChan := fetch.Fetch("https://api.example.com/protected",
        fetch.Options{
            Method: "GET",
            Headers: map[string]interface{}{
                "Authorization": "Bearer " + token,
                "Accept":        "application/json",
            },
        })

    go func() {
        result := <-resultChan

        // Handle result
    }()
}
```

## Advanced Patterns

### Conditional Fetching

```go
func UserPosts(props dom.Attrs) *dom.Element {
    userID, _ := state.UseAtom("selected-user-id", 0)
    currentUserID := userID()

    // Only fetch if userID is set
    if currentUserID == 0 {
        return dom.P(nil, dom.Text("Select a user"))
    }

    url := fmt.Sprintf("https://api.example.com/users/%d/posts", currentUserID)
    resource := fetch.UseFetch(url)
    state := resource.Get()

    // ... render posts
}
```

### Polling/Auto-Refresh

```go
func LiveData(props dom.Attrs) *dom.Element {
    resource := fetch.UseFetch("https://api.example.com/live-data")

    // Poll every 5 seconds
    hooks.UseEffect(func() {
        ticker := time.NewTicker(5 * time.Second)
        done := make(chan bool)

        go func() {
            for {
                select {
                case <-ticker.C:
                    resource.Refetch()
                case <-done:
                    ticker.Stop()
                    return
                }
            }
        }()

        // Cleanup
        return func() {
            done <- true
        }
    }, nil)

    state := resource.Get()
    // ... render data
}
```

### Parallel Requests

```go
func Dashboard(props dom.Attrs) *dom.Element {
    usersResource := fetch.UseFetch("https://api.example.com/users")
    postsResource := fetch.UseFetch("https://api.example.com/posts")
    commentsResource := fetch.UseFetch("https://api.example.com/comments")

    usersState := usersResource.Get()
    postsState := postsResource.Get()
    commentsState := commentsResource.Get()

    if usersState.Loading || postsState.Loading || commentsState.Loading {
        return dom.P(nil, dom.Text("Loading dashboard..."))
    }

    // All three requests complete, render dashboard
}
```

## Error Handling

### Retry Logic

```go
func RobustFetch(props dom.Attrs) *dom.Element {
    resource := fetch.UseFetch("https://api.example.com/data")
    retryCount, setRetryCount := hooks.UseState(0)

    state := resource.Get()

    if state.Error != "" {
        return dom.Div(nil,
            dom.P(nil, dom.Text("Error: "+state.Error)),
            dom.P(nil, dom.Text(fmt.Sprintf("Retries: %d", retryCount()))),
            dom.Button(dom.Attrs{
                "onclick": hooks.GoUseFunc(func(e dom.GoEvent) {
                    setRetryCount(retryCount() + 1)
                    resource.Refetch()
                }),
            }, dom.Text("Retry")),
        )
    }

    // ... render data
}
```

### Timeout Handling

```go
func FetchWithTimeout(url string, timeout time.Duration) <-chan fetch.Result {
    resultChan := fetch.Fetch(url, fetch.Options{Method: "GET"})
    timeoutChan := make(chan fetch.Result, 1)

    go func() {
        select {
        case result := <-resultChan:
            timeoutChan <- result
        case <-time.After(timeout):
            timeoutChan <- fetch.Result{
                Err: fmt.Errorf("request timeout after %v", timeout),
            }
        }
    }()

    return timeoutChan
}
```

## Best Practices

### 1. Prefer `UseResource[T]` for typed app data

```go
// ✅ Good - typed loader with cancellation and reload support
resource := fetch.UseResource(func(ctx context.Context) (User, error) {
    return loadUser(ctx)
})
```

### 2. Use `UseFetch` when you want raw response state

```go
// ✅ Good - simple raw fetch state
func Component(props dom.Attrs) *dom.Element {
    resource := fetch.UseFetch(url)
    state := resource.Get()
    // ...
}

// ❌ Less ideal - manual state management for the same flow
func Component(props dom.Attrs) *dom.Element {
    data, setData := hooks.UseState(nil)
    loading, setLoading := hooks.UseState(true)

    hooks.UseEffect(func() {
        // Manual fetch with fetch.Fetch...
    }, nil)
}
```

### 3. Handle All States

```go
func GoodComponent(props dom.Attrs) *dom.Element {
    resource := fetch.UseFetch(url)
    state := resource.Get()

    // Always handle all three states
    if state.Loading {
        return LoadingSpinner()
    }

    if state.Error != "" {
        return ErrorDisplay(state.Error)
    }

    return DataDisplay(state.Data)
}
```

## Related Packages

- **[/hooks](../hooks/)** - `UseFetch` is built on hooks system
- **[/dom](../dom/)** - For creating UI elements
- **[/examples/08-fetch](../examples/08-fetch/)** - Complete fetch example

## Examples

See fetching in action:

- **[/examples/08-fetch](../examples/08-fetch/)** - Comprehensive fetch demonstration with multiple APIs

## Documentation

See [doc.go](./doc.go) for the official Go package documentation.
