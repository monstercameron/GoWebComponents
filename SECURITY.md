# Security Policy

## Reporting a vulnerability

Please report security vulnerabilities **privately** — do not open a public
issue, pull request, or discussion for a suspected vulnerability.

The preferred channel is GitHub's private vulnerability reporting:

1. Go to the repository's **Security** tab → **Report a vulnerability**
   (Private Vulnerability Reporting), or visit
   `https://github.com/monstercameron/GoWebComponents/security/advisories/new`.
2. Describe the issue, the affected version or commit, and a minimal
   reproduction. Include the impact you believe it has (for example: stored
   XSS in a public rendering API, a dev-server origin bypass, a hydration
   trust-boundary break).

If you cannot use GitHub advisories, contact the maintainer through the address
listed on the GitHub profile of the repository owner.

Please give us a reasonable window to investigate and ship a fix before any
public disclosure.

## Scope

This project is a Go + WebAssembly UI framework. Reports are in scope when they
affect the framework's own code, for example:

- A rendering, SSR, or hydration path that emits attacker-controlled markup or
  executable URLs into the DOM (the framework's render pipeline inserts content
  as text/attributes, never as parsed HTML — a way to bypass that is in scope).
- The public APIs that *intentionally* process untrusted input: `html.RenderMarkdown`
  (URL-scheme allowlisted) and the `sanitize` package.
- The development tooling that binds local sockets — notably the `gwc dev`
  live-reload WebSocket server (Origin-validated by default).
- The plugin host, interop bridges, and persisted-state/snapshot transport.

Out of scope: vulnerabilities in **example applications** under `examples/`,
which are reference material and explicitly not hardened production code;
issues that require a malicious local toolchain or a compromised developer
machine; and best-practice suggestions with no concrete exploit.

## Supported versions

Security fixes target the latest released `v*` tag and `master`. There is no
long-term support branch for older majors at this time.

## Defensive posture (what the framework already does)

- **Render pipeline:** the DOM adapter exposes no raw-HTML sink; untrusted
  content reaches the DOM only as text or attribute values.
- **Markdown:** `html.RenderMarkdown` allowlists URL schemes
  (http/https/mailto/relative/#) and drops `javascript:`/`data:`/`vbscript:`
  and obfuscated variants in links, images, and autolinks.
- **Dev server:** the live-reload WebSocket validates the request `Origin`
  against the dev host:port by default (opt-out is explicit and logs a warning).
- **Crash containment:** runtime panics are contained and reported as
  structured diagnostics rather than terminating the page.

## Dependency and build hygiene

Contributors are encouraged to run `govulncheck ./...` before submitting
changes. Release builds use `-trimpath` for reproducibility. Expanding
automated scanning (govulncheck/gosec in CI) and SBOM emission for releases on
the root module is tracked in the project backlog.
