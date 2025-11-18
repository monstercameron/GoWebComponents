# Router Package

**Location:** `/router`

```
GoWebComponents/
├── dom/
├── hooks/
├── state/
├── render/
├── router/           ← YOU ARE HERE
│   ├── doc.go
│   └── router.go
├── fetch/
├── internal/
├── examples/
└── ...
```

## Overview

The `router` package provides client-side routing for single-page applications (SPAs). It supports both hash-based routing (`#/path`) and browser history API routing, enabling navigation without full page reloads.

## Router Types

### Hash Router

Uses URL hash fragments for routing (e.g., `example.com/#/about`).

**Pros:**

- Works without server configuration
- Compatible with static hosting
- No server-side setup needed

**Cons:**

- Hash in URL (aesthetics)
- SEO limitations

### Browser Router

Uses HTML5 History API for clean URLs (e.g., `example.com/about`).

**Pros:**

- Clean, semantic URLs
- Better SEO
- Professional appearance

**Cons:**

- Requires server configuration (all routes → index.html)
- Doesn't work with file:// protocol

## Basic Usage

### Hash Router Example

```go
import (
    "github.com/monstercameron/GoWebComponents/dom"
    "github.com/monstercameron/GoWebComponents/router"
    "github.com/monstercameron/GoWebComponents/render"
)

func HomePage(props dom.Attrs) *dom.Element {
    return dom.Div(nil,
        dom.H1(nil, dom.Text("Home Page")),
        dom.P(nil, dom.Text("Welcome!")),
    )
}

func AboutPage(props dom.Attrs) *dom.Element {
    return dom.Div(nil,
        dom.H1(nil, dom.Text("About Page")),
    )
}

func NotFound(props dom.Attrs) *dom.Element {
    return dom.Div(nil,
        dom.H1(nil, dom.Text("404 - Page Not Found")),
    )
}

func main() {
    // Create router with routes
    r := router.NewHashRouter(map[string]router.RouteComponent{
        "/":      HomePage,
        "/about": AboutPage,
        "*":      NotFound,  // Catch-all route
    })

    // Render router
    container := js.Global().Get("document").Call("getElementById", "app")
    render.ToElement(r.Render(), container)

    select {}
}
```

### Browser Router Example

```go
func main() {
    r := router.NewBrowserRouter(map[string]router.RouteComponent{
        "/":         HomePage,
        "/about":    AboutPage,
        "/contact":  ContactPage,
        "/products": ProductsPage,
        "*":         NotFound,
    })

    container := js.Global().Get("document").Call("getElementById", "app")
    render.ToElement(r.Render(), container)

    select {}
}
```

## Navigation

### Using Links

Create navigation links with the `dom.A` element:

```go
func Navigation(props dom.Attrs) *dom.Element {
    return dom.Nav(nil,
        dom.A(dom.Attrs{
            "href": "#/",  // Hash router
        }, dom.Text("Home")),

        dom.A(dom.Attrs{
            "href": "#/about",
        }, dom.Text("About")),

        dom.A(dom.Attrs{
            "href": "#/contact",
        }, dom.Text("Contact")),
    )
}

// For Browser Router, omit the hash
func Navigation(props dom.Attrs) *dom.Element {
    return dom.Nav(nil,
        dom.A(dom.Attrs{
            "href": "/",  // Browser router
        }, dom.Text("Home")),

        dom.A(dom.Attrs{
            "href": "/about",
        }, dom.Text("About")),
    )
}
```

### Programmatic Navigation

Navigate programmatically using JavaScript:

```go
handleLogin := hooks.GoUseFunc(func(e dom.GoEvent) {
    e.PreventDefault()

    // Perform login logic
    if loginSuccessful {
        // Navigate to dashboard
        js.Global().Get("location").Set("hash", "/dashboard")
        // OR for browser router:
        // js.Global().Get("history").Call("pushState", nil, "", "/dashboard")
    }
})
```

## Dynamic Routes

### URL Parameters

Parse URL parameters manually:

```go
func UserProfile(props dom.Attrs) *dom.Element {
    // Get current hash
    hash := js.Global().Get("location").Get("hash").String()

    // Parse user ID from hash like #/users/123
    parts := strings.Split(hash, "/")
    userID := parts[len(parts)-1]

    return dom.Div(nil,
        dom.H1(nil, dom.Text("User Profile")),
        dom.P(nil, dom.Text("User ID: "+userID)),
    )
}

// Register with pattern
routes := map[string]router.RouteComponent{
    "/users":  UsersList,    // List all users
    "/users*": UserProfile,  // Match /users/anything
}
```

### Query Parameters

```go
func SearchResults(props dom.Attrs) *dom.Element {
    // Get URL search params
    url := js.Global().Get("location").Get("href").String()
    // Parse query: #/search?q=golang&sort=relevance

    return dom.Div(nil,
        dom.H1(nil, dom.Text("Search Results")),
        // Display results
    )
}
```

## Layouts

### App Layout with Router

```go
func AppLayout(props dom.Attrs) *dom.Element {
    r := router.NewHashRouter(map[string]router.RouteComponent{
        "/":        HomePage,
        "/about":   AboutPage,
        "/contact": ContactPage,
        "*":        NotFound,
    })

    return dom.Div(nil,
        Navigation(nil),  // Always visible nav
        dom.Main(nil,
            r.Render(),   // Routed content
        ),
        Footer(nil),      // Always visible footer
    )
}

func Navigation(props dom.Attrs) *dom.Element {
    return dom.Nav(dom.Attrs{"class": "navbar"},
        dom.A(dom.Attrs{"href": "#/"}, dom.Text("Home")),
        dom.A(dom.Attrs{"href": "#/about"}, dom.Text("About")),
        dom.A(dom.Attrs{"href": "#/contact"}, dom.Text("Contact")),
    )
}

func Footer(props dom.Attrs) *dom.Element {
    return dom.Footer(nil,
        dom.P(nil, dom.Text("© 2024 My App")),
    )
}
```

## Protected Routes

```go
func ProtectedRoute(component router.RouteComponent) router.RouteComponent {
    return func(props dom.Attrs) *dom.Element {
        isAuthenticated, _ := state.UseAtom("user-authenticated", false)

        if !isAuthenticated() {
            // Redirect to login
            js.Global().Get("location").Set("hash", "/login")
            return dom.Div(nil, dom.Text("Redirecting..."))
        }

        return component(props)
    }
}

// Usage
routes := map[string]router.RouteComponent{
    "/":         HomePage,
    "/login":    LoginPage,
    "/dashboard": ProtectedRoute(DashboardPage),
    "/settings":  ProtectedRoute(SettingsPage),
}
```

## Server Configuration

### For Browser Router

Configure your server to serve `index.html` for all routes:

**Nginx:**

```nginx
location / {
    try_files $uri $uri/ /index.html;
}
```

**Apache (.htaccess):**

```apache
<IfModule mod_rewrite.c>
    RewriteEngine On
    RewriteBase /
    RewriteRule ^index\.html$ - [L]
    RewriteCond %{REQUEST_FILENAME} !-f
    RewriteCond %{REQUEST_FILENAME} !-d
    RewriteRule . /index.html [L]
</IfModule>
```

**Go HTTP Server:**

```go
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "./static/index.html")
})
```

## Best Practices

### 1. Use Catch-All Route

Always include a `*` route for 404 pages:

```go
routes := map[string]router.RouteComponent{
    "/": HomePage,
    "*": NotFoundPage,
}
```

### 2. Active Link Highlighting

```go
func NavLink(props dom.Attrs) *dom.Element {
    href := props["href"].(string)
    text := props["text"].(string)

    currentHash := js.Global().Get("location").Get("hash").String()
    isActive := currentHash == href

    className := "nav-link"
    if isActive {
        className += " active"
    }

    return dom.A(dom.Attrs{
        "href":  href,
        "class": className,
    }, dom.Text(text))
}
```

### 3. Route Constants

```go
const (
    RouteHome    = "/"
    RouteAbout   = "/about"
    RouteContact = "/contact"
)

routes := map[string]router.RouteComponent{
    RouteHome:    HomePage,
    RouteAbout:   AboutPage,
    RouteContact: ContactPage,
}
```

## Related Packages

- **[/dom](../dom/)** - Element creation for route components
- **[/render](../render/)** - Renders router output
- **[/state](../state/)** - Manage auth state for protected routes
- **[/examples/12-portfolio-site](../examples/12-portfolio-site/)** - Full routing example

## Examples

See routing in action:

- **[/examples/12-portfolio-site](../examples/12-portfolio-site/)** - Complete SPA with navigation
- **[/test/specs/browser_router.spec.js](../test/specs/browser_router.spec.js)** - Browser router tests
- **[/test/specs/hash_router.spec.js](../test/specs/hash_router.spec.js)** - Hash router tests

## Documentation

See [doc.go](./doc.go) for the official Go package documentation.
