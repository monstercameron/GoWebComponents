# GWC | Head Library

# GoWebComponents (GWC)

## High-Level Overview

The `head` library manages document head metadata integration used by GWC renderers and routing flows.

## Public APIs

### `github.com/monstercameron/GoWebComponents/head` (`package head`)
- Functions: `AlternateLinks`, `Compose`, `Hreflang`, `LinkRel`, `Merge`, `MetaName`, `MetaProperty`, `OpenGraph`, `Render`, `RenderJSONLD`, `RenderToString`, `Resolve`, `ResourceHints`, `Robots`, `SocialTags`, `Twitter`
- Types: `AlternateLink`, `Document`, `JSONLDBlock`, `MergeOptions`, `ResourceHint`, `RouteLayer`, `SocialMetadata`
- Variables: _none_
- Constants: _none_

## Subfiles And Purpose

- `doc.go` - Package-level Go documentation
- `head.go` - Core implementation for head
- `head_additional_test.go` - Tests for head_additional behavior
- `head_test.go` - Tests for head behavior

## File Map

```text
head/
|-- doc.go
|-- head.go
|-- head_additional_test.go
\-- head_test.go
```



