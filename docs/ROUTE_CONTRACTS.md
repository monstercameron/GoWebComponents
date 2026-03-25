# Route Contracts

Use this page when deciding how far route contracts should go beyond basic
reverse routing.

## Current Decision

The shipped core surface is `router.RouteContract` for runtime-owned route
definitions and validated reverse routing.

That is the supported default for:

- route registration
- links and redirects
- typed param and query adapters
- tests that should stop repeating string paths

The project is not shipping first-party route-manifest code generation in core
today.

## Why The Decision Stays Runtime-First

- small and medium apps need one shared route definition more than they need a
  manifest pipeline
- metadata ownership, prerender enumeration, and starter scaffolds vary by app
  shape and deployment model
- freezing a generated manifest schema too early would create more migration
  pressure than the current runtime helper surface
- runtime contracts already remove the most common string drift between route
  registration, navigation, and tests

## What Stays Application-Owned Today

- grouping route contracts into app-specific registries
- mapping contracts to metadata keys or head composition
- prerender route enumeration
- starter-template conventions for route layout, sitemap ownership, or example
  generation

Apps can already layer those decisions on top of `router.RouteContract`
directly.

## If Tooling Expands Later

If larger apps need generated route manifests later, that work should land as
companion tooling rather than expanding `router` itself.

That tooling should:

- consume the same route-contract source definitions apps already use at runtime
- emit stable manifests for navigation, metadata ownership, or prerender inputs
- stay optional for smaller apps that only need runtime reverse routing
- avoid inventing a second route-definition format unrelated to
  `router.RouteContract`

## Recommended Shape Today

Keep one application-owned registry that pairs each `router.RouteContract` with
the extra policy the app needs.

For example, an app-level route record can bundle:

- the contract
- route metadata or a metadata lookup key
- whether the route is prerenderable
- auth or loader policy owned by the app

That keeps navigation, metadata, and export decisions aligned without requiring
framework-owned code generation.
