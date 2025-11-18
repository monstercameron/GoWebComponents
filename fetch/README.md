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

The `fetch` package provides utilities for making HTTP requests from WebAssembly applications. It offers both declarative (hook-based) and imperative APIs for data fetching, with built-in loading states, error handling, and automatic re-fetching.

## Core APIs

### `UseFetch(url string, options ...FetchOptions) (func() FetchState, func())`

Declarative hook for data fetching within components. Automatically manages loading, error, and data states.

```go
import "github.com/monstercameron/GoWebComponents/fetch"

func UserProfile(props dom.Attrs) *dom.Element {
    // Returns: (stateGetter, refetch)
    getState, refetch := fetch.UseFetch("https://api.example.com/user/1")
    state := getState()
    
    if state.Loading {
        return dom.P(nil, dom.Text("Loading..."))
    }
    
    if state.Error != "" {
        return dom.Div(nil,
            dom.P(nil, dom.Text("Error: "+state.Error)),
            dom.Button(dom.Attrs{
                "onclick": hooks.GoUseFunc(func(e dom.GoEvent) {
                    refetch()  // Retry the request
                }),
            }, dom.Text("Retry")),
        )
    }
    
    user := state.Data.(map[string]interface{})
    return dom.Div(nil,
        dom.H1(nil, dom.Text(user["name"].(string))),
        dom.P(nil, dom.Text(user["email"].(string))),
    )
}
```

### `GoFetch(url string, options FetchOptions) <-chan FetchResult`

Imperative API for making HTTP requests from anywhere (event handlers, goroutines, etc.).

```go
import "github.com/monstercameron/GoWebComponents/fetch"

func CreateUser(userData map[string]interface{}) {
    resultChan := fetch.GoFetch("https://api.example.com/users", fetch.FetchOptions{
        Method:  "POST",
        Headers: map[string]interface{}{
            "Content-Type": "application/json",
        },
        Body: userData,
    })
    
    go func() {
        result := <-resultChan
        fetch.ReturnFetchChannel(resultChan)  // IMPORTANT: Prevent memory leaks
        
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

### FetchResult
```go
type FetchResult struct {
    Data interface{}  // Response data
    Err  error        // Error if request failed
}
```

### FetchOptions
```go
type FetchOptions struct {
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
        
        resultChan := fetch.GoFetch("https://jsonplaceholder.typicode.com/posts", 
            fetch.FetchOptions{
                Method: "POST",
                Headers: map[string]interface{}{
                    "Content-Type": "application/json",
                },
                Body: postData,
            })
        
        go func() {
            result := <-resultChan
            fetch.ReturnFetchChannel(resultChan)
            
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
    
    resultChan := fetch.GoFetch(url, fetch.FetchOptions{
        Method: "PUT",
        Headers: map[string]interface{}{
            "Content-Type": "application/json",
        },
        Body: updates,
    })
    
    go func() {
        result := <-resultChan
        fetch.ReturnFetchChannel(resultChan)
        
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
    
    resultChan := fetch.GoFetch(url, fetch.FetchOptions{
        Method: "DELETE",
    })
    
    go func() {
        result := <-resultChan
        fetch.ReturnFetchChannel(resultChan)
        
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
    
    resultChan := fetch.GoFetch("https://api.example.com/protected", 
        fetch.FetchOptions{
            Method: "GET",
            Headers: map[string]interface{}{
                "Authorization": "Bearer " + token,
                "Accept":        "application/json",
            },
        })
    
    go func() {
        result := <-resultChan
        fetch.ReturnFetchChannel(resultChan)
        
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
    getState, _ := fetch.UseFetch(url)
    state := getState()
    
    // ... render posts
}
```

### Polling/Auto-Refresh

```go
func LiveData(props dom.Attrs) *dom.Element {
    getState, refetch := fetch.UseFetch("https://api.example.com/live-data")
    
    // Poll every 5 seconds
    hooks.UseEffect(func() {
        ticker := time.NewTicker(5 * time.Second)
        done := make(chan bool)
        
        go func() {
            for {
                select {
                case <-ticker.C:
                    refetch()
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
    
    state := getState()
    // ... render data
}
```

### Parallel Requests

```go
func Dashboard(props dom.Attrs) *dom.Element {
    getUsers, _ := fetch.UseFetch("https://api.example.com/users")
    getPosts, _ := fetch.UseFetch("https://api.example.com/posts")
    getComments, _ := fetch.UseFetch("https://api.example.com/comments")
    
    usersState := getUsers()
    postsState := getPosts()
    commentsState := getComments()
    
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
    getState, refetch := fetch.UseFetch("https://api.example.com/data")
    retryCount, setRetryCount := hooks.UseState(0)
    
    state := getState()
    
    if state.Error != "" {
        return dom.Div(nil,
            dom.P(nil, dom.Text("Error: "+state.Error)),
            dom.P(nil, dom.Text(fmt.Sprintf("Retries: %d", retryCount()))),
            dom.Button(dom.Attrs{
                "onclick": hooks.GoUseFunc(func(e dom.GoEvent) {
                    setRetryCount(retryCount() + 1)
                    refetch()
                }),
            }, dom.Text("Retry")),
        )
    }
    
    // ... render data
}
```

### Timeout Handling

```go
func FetchWithTimeout(url string, timeout time.Duration) <-chan FetchResult {
    resultChan := fetch.GoFetch(url, fetch.FetchOptions{Method: "GET"})
    timeoutChan := make(chan FetchResult, 1)
    
    go func() {
        select {
        case result := <-resultChan:
            fetch.ReturnFetchChannel(resultChan)
            timeoutChan <- result
        case <-time.After(timeout):
            fetch.ReturnFetchChannel(resultChan)
            timeoutChan <- FetchResult{
                Err: fmt.Errorf("request timeout after %v", timeout),
            }
        }
    }()
    
    return timeoutChan
}
```

## Best Practices

### 1. Always Return Channels
```go
// ✅ Good
result := <-resultChan
fetch.ReturnFetchChannel(resultChan)

// ❌ Bad - memory leak
result := <-resultChan
// Channel not returned!
```

### 2. Use Declarative API in Components
```go
// ✅ Good - automatic state management
func Component(props dom.Attrs) *dom.Element {
    getState, _ := fetch.UseFetch(url)
    state := getState()
    // ...
}

// ❌ Less ideal - manual state management
func Component(props dom.Attrs) *dom.Element {
    data, setData := hooks.UseState(nil)
    loading, setLoading := hooks.UseState(true)
    
    hooks.UseEffect(func() {
        // Manual fetch with GoFetch...
    }, nil)
}
```

### 3. Handle All States
```go
func GoodComponent(props dom.Attrs) *dom.Element {
    getState, _ := fetch.UseFetch(url)
    state := getState()
    
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
