# GWC | Feature Flags

The `flags` package provides typed browser-visible feature flag and experiment
helpers.

Use it for evaluated, non-secret rollout decisions that are safe to transfer
through SSR bootstrap, hydrated state, or browser storage. Keep secret policy,
entitlement checks, and server-only rollout rules on the application server.

## Public APIs

### `github.com/monstercameron/GoWebComponents/flags` (`package flags`)

- Functions: `BuildSet`, `UseExperiment`, `UseFlag`, `UseRegistry`
- Types: `Assignment`, `Experiment`, `ExperimentHandle`, `Flag`,
  `FlagHandle`, `Registry`, `Set`, `Variant`

## Example

Inside a component render:

```go
registry := flags.UseRegistry(flags.BuildSet(
	map[string]flags.Flag{
		"new-nav": {Enabled: true, Value: "compact"},
	},
	map[string]flags.Experiment{
		"pricing-copy": {
			Enabled: true,
			Salt:    "2026-06",
			Variants: []flags.Variant{
				{Name: "control", Weight: 50},
				{Name: "direct", Weight: 50},
			},
		},
	},
))

newNav := flags.UseFlag("new-nav", false)
assignment := registry.Get().GetAssignment("pricing-copy", userID)
```

## File Map

```text
flags/
|-- doc.go
|-- flags.go
|-- flags_test.go
\-- README.md
```
