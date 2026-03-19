# Router Package

The `router` package provides client-side routing for GoWebComponents single-page applications.

It supports:

- Hash routers and browser/history routers
- Exact routes, parameter routes, and prefix wildcard routes
- Nested layout routes with explicit `Outlet()` rendering
- Programmatic navigation with `UseNavigate`
- Query parsing with `UseQuery`
- Query updates and serialization with `UseSearchParams`
- Route params with `UseParams`
- Manual route revalidation with `UseRevalidator`
- Route-level async loaders with cancellation and query-aware revalidation

## Core API

### Router creation

```go
r := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/"})
// or
r := router.NewRouter(router.RouterOptions{DefaultRoute: "/"})
```

### Route registration

```go
r.Register("/", HomePage)
r.Register("/users/:id", UserProfile)
r.Register("/settings*", SettingsArea)
r.Register("*", NotFoundPage)
```

Routes may also include loader options:

```go
r.Register("/users/:id", UserProfile, router.Options{
    Loader: func(ctx context.Context, routeCtx router.RouteContext) (router.Attrs, error) {
        return router.Attrs{"userID": routeCtx.Params.Get("id")}, nil
    },
})
```

Routes can also declare titles and redirects through `router.Options`:

```go
r.Register("/legacy", LegacyPage, router.Options{
    Redirect: "/dashboard",
})

r.Register("/dashboard", DashboardPage, router.Options{
    Title: "Dashboard",
})
```

Route metadata can also be managed declaratively through the same options:

```go
r.Register("/docs", DocsPage, router.Options{
    Title:        "Docs",
    Description:  "Framework guides and API documentation",
    CanonicalURL: "https://example.com/docs",
})
```

Routes can also declare synchronous navigation guards:

```go
r.Register("/settings", SettingsPage, router.Options{
    BeforeEnter: func(ctx router.RouteContext) router.GuardResult {
        if !userIsSignedIn {
            return router.RedirectNavigation("/login")
        }
        return router.AllowNavigation()
    },
    BeforeLeave: func(current router.RouteContext, next router.RouteContext) router.GuardResult {
        if hasUnsavedChanges {
            return router.BlockNavigation("Unsaved changes")
        }
        return router.AllowNavigation()
    },
})
```

Nested layout routes use the same registration API with `Options{Layout: true}`:

```go
r.Register("/dashboard", DashboardLayout, router.Options{Layout: true})
r.Register("/dashboard/reports/:id", ReportPage)
```

### Mounting

```go
r.Mount("#app")
```

## Basic Usage

```go
import (
    "github.com/monstercameron/GoWebComponents/html"
    "github.com/monstercameron/GoWebComponents/router"
)

func HomePage(props router.Attrs) *router.Element {
    return html.Div(html.Props{},
        html.H1(html.Props{}, html.Text("Home Page")),
        html.P(html.Props{}, html.Text("Welcome!")),
    )
}

func AboutPage(props router.Attrs) *router.Element {
    return html.Div(html.Props{},
        html.H1(html.Props{}, html.Text("About Page")),
    )
}

func NotFoundPage(props router.Attrs) *router.Element {
    return html.Div(html.Props{},
        html.H1(html.Props{}, html.Text("404 - Page Not Found")),
    )
}

func main() {
    r := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/"})
    r.Register("/", HomePage)
    r.Register("/about", AboutPage)
    r.Register("*", NotFoundPage)
    r.Mount("#app")

    select {}
}
```

## Router Types

### Hash router

Use `NewHashRouter` when you want routing to work without server rewrites.

Pros:

- Works from static hosting and simple file-serving setups
- No server rewrite rules required

Cons:

- URLs include `#`

### Browser/history router

Use `NewRouter` when you want clean path-based URLs.

Pros:

- Clean URLs like `/about`
- Better fit for production sites with server routing support

Cons:

- Requires server rewrites to your app entry point
- Not suitable for `file://`-style usage

## Navigation

### Programmatic navigation

```go
func LoginButton(props router.Attrs) *router.Element {
    nav := router.UseNavigate()

    handleLogin := ui.UseEvent(func() {
        if loginSuccessful {
            nav.Navigate("/dashboard")
        }
    })

    return html.Button(html.Props{OnClick: handleLogin}, html.Text("Login"))
}
```

Use `Replace` when you want to overwrite the current history entry:

```go
nav := router.UseNavigate()
nav.Replace("/login")
```

Declarative redirects use replacement semantics as well, so redirect routes do
not leave an extra stale history entry behind.

Synchronous `BeforeEnter` and `BeforeLeave` guards can also redirect or block
router-driven navigation. Async guard behavior is still a separate future item.

For current protected-route guidance, safe `return_to` handling, and the
manual authorizing or unauthorized pattern that exists before async auth-aware
route primitives land, see `docs/ROUTER_AUTH.md` and
`examples/92-protected-routes`.

For the planned async-guard and auth-aware routing contract, see
`docs/ROUTER_AUTH.md`.

Route-managed metadata uses replacement semantics too: when a later route omits
description or canonical metadata, the router removes the previously managed
values instead of leaving stale tags behind.

## SSR Metadata Ownership

Route metadata for `Title`, `Description`, and `CanonicalURL` is jointly
reconciled across SSR and the client router.

The intended model is:

- the server emits the initial route metadata into `<head>`
- the emitted tags are marked as router-managed metadata
- hydration leaves the existing server-rendered tags in place
- later client navigation updates or removes only router-managed tags

Use `router.MetadataNode(...)` together with `ui.RenderToString(...)` when a
server-rendered route needs head metadata during the first HTML response.

For the broader head and SEO policy, including robots tags, social cards,
structured data, resource hints, and canonical-URL guidance, see
`docs/HEAD_MANAGEMENT.md`.

The router currently owns only three fields through this path:

- document title
- `meta[name="description"]`
- `link[rel="canonical"]`

This is intentionally narrower than full head management. Social metadata,
robots directives, preload hints, and structured data remain a separate future
surface.

Hydration reconciliation rules for this metadata slice are:

- server-rendered router metadata should be emitted with `router.MetadataNode(...)`
- client cleanup removes only tags marked `data-gwc-router-managed="true"`
- when the initial SSR title is router-managed, navigating to a route with no
    title clears the stale managed title instead of preserving it indefinitely
- unmanaged head tags are outside the router-owned contract and should be
    handled through explicit application markup until a broader head-management
    API exists

### Manual route revalidation

```go
revalidator := router.UseRevalidator()
revalidator.Revalidate()
```

Use this when you want to rerun the current route loader without changing the path or query string.

### Link-style navigation

```go
func Navigation(props router.Attrs) *router.Element {
    return html.Nav(html.Props{},
        html.A(html.Props{Href: "#/"}, html.Text("Home")),
        html.A(html.Props{Href: "#/about"}, html.Text("About")),
    )
}
```

For browser routers, omit the `#` and use paths like `/about`.

## Route Patterns

### Exact routes

```go
r.Register("/about", AboutPage)
```

### Parameter routes

```go
r.Register("/users/:id", UserProfile)
```

### Prefix wildcard routes

```go
r.Register("/settings*", SettingsArea)
```

This matches `/settings`, `/settings/profile`, and `/settings/security`.

## Nested Routes And Layout Routes

Mark a parent route with `Layout: true` when it should stay mounted while a more
specific child route renders into its outlet.

```go
func DashboardLayout(props router.Attrs) *router.Element {
    return html.Div(html.Props{},
        html.Header(html.Props{}, html.Text("Dashboard")),
        html.Main(html.Props{}, router.Outlet()),
    )
}

func ReportPage(props router.Attrs) *router.Element {
    params := router.UseParams()
    return html.Div(html.Props{}, html.Text("Report "+params.Get("id")))
}

r.Register("/dashboard", DashboardLayout, router.Options{Layout: true})
r.Register("/dashboard/reports/:id", ReportPage)
```

Matching rules for nested layouts:

- only routes marked with `Layout: true` participate as parent layouts
- parent layouts render from the shallowest matching prefix to the deepest match
- the final matched child route renders into `router.Outlet()`
- parent layouts receive params captured by their own matched prefix
- child routes receive the full merged param set for the final matched path
- layout guards run for nested navigation just like leaf-route guards

If no child route matches, the router falls back to the best leaf match it can
resolve, including the catch-all route when one is registered.

### Path normalization rules

The router applies the same normalization rules across registration, matching,
and navigation:

- blank paths normalize to `/`
- missing leading slashes are added
- trailing slashes are trimmed except for the root path
- query strings are ignored for route matching but preserved for navigation targets
- hash-router query strings are read from the `#/path?query=value` portion
- route params are URL-decoded before they are exposed through `UseParams`
- optional segment syntax like `:id?` is not currently supported

## Route Params

Use `UseParams()` inside routed components.

```go
func UserProfile(props router.Attrs) *router.Element {
    params := router.UseParams()
    userID := params.Get("id")
    numericID, _ := params.Int("id")

    return html.Div(html.Props{},
        html.H1(html.Props{}, html.Text("User Profile")),
        html.P(html.Props{}, html.Text("User ID: "+userID)),
        html.P(html.Props{}, html.Text(fmt.Sprintf("Numeric ID: %d", numericID))),
    )
}
```

Available helpers:

```go
params := router.UseParams()

slug := params.Get("slug")
id, ok := params.Int("id")
enabled, ok := params.Bool("enabled")
all := params.Values()
```

## Query Parameters

Use `UseQuery()` for URL query state.

```go
func SearchResults(props router.Attrs) *router.Element {
    query := router.UseQuery()
    term := query.Get("q")
    sort := query.Get("sort")

    return html.Div(html.Props{},
        html.H1(html.Props{}, html.Text("Search Results")),
        html.P(html.Props{}, html.Text("Query: "+term)),
        html.P(html.Props{}, html.Text("Sort: "+sort)),
    )
}
```

`UseQuery()` works for both:

- browser-router URLs like `/search?q=golang`
- hash-router URLs like `#/search?q=golang&sort=relevance`

When you need to update search params while preserving the current route path, use `UseSearchParams()`:

```go
func SearchToolbar(props router.Attrs) *router.Element {
    search := router.UseSearchParams()

    applyRecent := ui.UseEvent(func() {
        search.Set("sort", "recent")
    })

    clearQuery := ui.UseEvent(func() {
        search.Delete("q")
    })

    replaceFilter := ui.UseEvent(func() {
        search.ReplaceAll(url.Values{"filter": {"active"}})
    })

    return html.Div(html.Props{},
        html.P(html.Props{}, html.Text("Current query: "+search.Encode())),
        html.Button(html.Props{OnClick: applyRecent}, html.Text("Sort Recent")),
        html.Button(html.Props{OnClick: clearQuery}, html.Text("Clear Query")),
        html.Button(html.Props{OnClick: replaceFilter}, html.Text("Active Only")),
    )
}
```

## Route Loaders

Route loaders let the router fetch route-scoped data before the final page component renders.

```go
func UserProfile(props router.Attrs) *router.Element {
    userID := props["userID"].(string)
    userName := props["name"].(string)

    return html.Div(html.Props{},
        html.H1(html.Props{}, html.Text(userName)),
        html.P(html.Props{}, html.Text("User ID: "+userID)),
    )
}

func RouteLoading(props router.Attrs) *router.Element {
    return html.Div(html.Props{}, html.Text("Loading route..."))
}

func RouteError(props router.Attrs) *router.Element {
    return html.Div(html.Props{}, html.Text("Route error: "+props["error"].(string)))
}

r.Register("/users/:id", UserProfile, router.Options{
    Loader: func(ctx context.Context, routeCtx router.RouteContext) (router.Attrs, error) {
        return router.Attrs{
            "userID": routeCtx.Params.Get("id"),
            "name":   "Ada Lovelace",
        }, nil
    },
    Loading: RouteLoading,
    Error:   RouteError,
})
```

Loader behavior:

- loaders rerun when the route path or query string changes
- in-flight loaders are cancelled when navigation changes
- loader results are reused for the current route key until the path or query changes
- loader data is merged into the routed component props

You can also manually rerun the current route loader:

```go
func UserToolbar(props router.Attrs) *router.Element {
    revalidator := router.UseRevalidator()

    return html.Button(html.Props{
        OnClick: ui.UseEvent(func() {
            revalidator.Revalidate()
        }),
    }, html.Text("Reload route data"))
}
```

If needed, the final routed component can also read loader data with `router.UseRouteData()`.

Layout routes can also use `router.UseRouteData()`; each layout or page sees the
loader data for its own matched route while it renders.

## Layout Pattern

```go
func AppShell(props router.Attrs) *router.Element {
    return html.Div(html.Props{},
        html.Header(html.Props{}, html.Text("My App")),
        html.Main(html.Props{}, router.GetRoute()),
    )
}
```

In most apps you register routes once in `main()` and let the router mount directly into your root container.

## Protected Route Pattern

```go
func protectedGuard(ctx router.RouteContext) router.GuardResult {
    if !userIsSignedIn {
        values := url.Values{}
        values.Set(router.ReturnToParam, router.PreserveReturnTo(ctx.Path, ctx.Query.Values()))
        return router.RedirectNavigation("/login?" + values.Encode())
    }
    return router.AllowNavigation()
}

r.Register("/workspace", WorkspacePage, router.Options{
    BeforeEnter: protectedGuard,
})
```

When auth is still unresolved or a subsection is forbidden, keep the route
mounted and render manual authorizing or unauthorized UI in the page itself.
That is the current shipped pattern until route-level `Authorizing` and
`Unauthorized` options exist.

## Best Practices

### 1. Always register a catch-all route

```go
r.Register("*", NotFoundPage)
```

### 2. Keep route strings centralized

```go
const (
    RouteHome    = "/"
    RouteAbout   = "/about"
    RouteContact = "/contact"
)
```

### 3. Prefer router helpers over manual `location` parsing

- Use `UseNavigate()` instead of mutating `window.location` directly
- Use `UseParams()` instead of splitting paths by hand
- Use `UseQuery()` instead of manually parsing the query string

## Browser Router Server Configuration

For browser/history routing, configure your server to serve `index.html` for unknown routes.

### Nginx

```nginx
location / {
    try_files $uri $uri/ /index.html;
}
```

### Apache

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

### Go HTTP server

```go
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "./static/index.html")
})
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
