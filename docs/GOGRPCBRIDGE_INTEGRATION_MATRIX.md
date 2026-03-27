# GoGRPCBridge Integration Matrix

This matrix records where this repository currently depends on `third_party/GoGRPCBridge`.

## Module and Source Wiring

| Area | Path | Dependency Shape | Validation |
| --- | --- | --- | --- |
| Go module wiring | `go.mod` | Requires `github.com/monstercameron/grpc-tunnel` and maps it to `./third_party/GoGRPCBridge` via `replace` | `go list -m github.com/monstercameron/grpc-tunnel` |
| Submodule source mapping | `.gitmodules` | Pins submodule path `third_party/GoGRPCBridge` to upstream source repository | `git submodule status -- third_party/GoGRPCBridge` |

## Runtime Consumers in This Repo

| Area | Path | Dependency Shape | Validation |
| --- | --- | --- | --- |
| Browser gRPC client | `examples/100-ai-chat-wizard/client/app/runtime.go` | Imports `pkg/grpctunnel` and dials browser tunnel (`DialContext`) | `go test ./examples/100-ai-chat-wizard/client/app` |
| Server tunnel adapter | `examples/100-ai-chat-wizard/server/app/tunnel_handler.go` | Imports `pkg/grpctunnel` and wraps gRPC server (`Wrap`, hooks) | `go test ./examples/100-ai-chat-wizard/server/app` |
| WASM runtime asset resolution | `examples/100-ai-chat-wizard/server/app/server.go` | Resolves `wasm_exec.js` from `third_party/GoGRPCBridge/examples/_shared/public/wasm_exec.js` fallback paths | `go test ./examples/100-ai-chat-wizard/server/app -run TestTransportHelpersCoverTunnelAndWasmResolution` |

## Tooling and Ops Integration

| Area | Path | Dependency Shape | Validation |
| --- | --- | --- | --- |
| Bootstrap workflow | `scripts/bootstrap-gogrpcbridge.ps1` | Initializes submodule and runs bridge runner checks from repo root | `.\scripts\bootstrap-gogrpcbridge.ps1 -SkipDoctor -SkipBridgeChecks` |
| Recommended CI path | `.github/workflows/gogrpcbridge-ci.yml` | Runs lint/unit/wasm/browser lanes, benchmark trend tracking, fast-PR and full-gate aggregators, plus root integration smoke checks | Trigger workflow on PR touching bridge paths |
| Required check policy | `docs/GOGRPCBRIDGE_REQUIRED_CHECKS.md` | Defines required status-check lanes and branch-protection setup | Verify branch protection contains listed check names |
| Lifecycle operations | `docs/GOGRPCBRIDGE_SUBMODULE_LIFECYCLE.md` | Canonical `init`, `verify`, `update`, `pin` commands for submodule operations | Run commands in doc from repo root |

## Known Gap

- Direct `go get github.com/monstercameron/grpc-tunnel@latest` from a clean external consumer module is not yet verified as green in this repo; track under roadmap item `S10.3`.
