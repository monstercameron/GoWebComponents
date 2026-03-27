# GoGRPCBridge Submodule Lifecycle

This document defines the canonical lifecycle for the `third_party/GoGRPCBridge` submodule in this repository.

## Scope

- Submodule path: `third_party/GoGRPCBridge`
- Submodule URL source of truth: `.gitmodules`
- Go module wiring source of truth: root `go.mod` `replace` directive

Current wiring:
- `.gitmodules` points `third_party/GoGRPCBridge` to `https://github.com/monstercameron/grpc-tunnel`
- root `go.mod` includes `replace github.com/monstercameron/grpc-tunnel => ./third_party/GoGRPCBridge`

## Init

Initialize the submodule from repo root:

```powershell
git submodule update --init --recursive third_party/GoGRPCBridge
```

If you cloned without submodules:

```powershell
git submodule update --init --recursive
```

## Bootstrap Command

Run this one command from repo root to verify toolchain/runtime prerequisites and submodule state:

```powershell
.\scripts\bootstrap-gogrpcbridge.ps1
```

By default this command performs actionable dependency checks for:
- `go`
- `git`
- `protoc`
- `protoc-gen-go`
- `protoc-gen-go-grpc`
- Playwright Go CLI (`go run .../playwright@latest --version`)

Optional fast path:

```powershell
.\scripts\bootstrap-gogrpcbridge.ps1 -SkipDoctor -SkipBridgeChecks
```

## First 10 Minutes (Bridge Contributor Path)

From repo root:

1. Run bootstrap and dependency checks:

```powershell
.\scripts\bootstrap-gogrpcbridge.ps1
```

2. Run the bridge quick validation path:

```powershell
go run ./third_party/GoGRPCBridge/tools/runner.go test-short
```

3. Open bridge-specific docs:

```powershell
Get-Content .\third_party\GoGRPCBridge\README.md
Get-Content .\third_party\GoGRPCBridge\CONTRIBUTING.md
```

Success criteria within first 10 minutes:
- submodule is initialized and pinned
- required tools are present with actionable failures if missing
- bridge quick tests run from root via the bridge runner

## Verify

Verify submodule checkout and module wiring from repo root:

```powershell
git submodule status -- third_party/GoGRPCBridge
go list -m github.com/monstercameron/grpc-tunnel
```

Expected:
- `git submodule status` shows a pinned commit for `third_party/GoGRPCBridge`
- `go list -m` resolves cleanly with the local `replace` in root `go.mod`

Smoke-check active integration:

```powershell
Set-Location third_party/GoGRPCBridge
go run ./tools/runner.go test-short
Set-Location ../..
```

## Update

Update submodule to upstream tracked branch:

```powershell
git submodule update --remote --checkout third_party/GoGRPCBridge
```

Then run focused verification:

```powershell
Set-Location third_party/GoGRPCBridge
go run ./tools/runner.go check
Set-Location ../..
```

## Pin

Pin to a specific commit or tag:

```powershell
Set-Location third_party/GoGRPCBridge
git fetch --tags origin
git checkout <commit-or-tag>
Set-Location ../..
git add third_party/GoGRPCBridge
git commit -m "chore: pin GoGRPCBridge submodule to <commit-or-tag>"
```

Use explicit pins for reproducible builds and release branches.

## Policy

- Do not edit submodule URL or branch tracking ad hoc; changes must be intentional and reviewed.
- Every submodule bump should include:
  - integration verification results
  - migration notes if API behavior changed
  - rationale for pin choice (bugfix, security, perf, or feature)
- If upstream and local module path conventions diverge, prefer preserving consumer import stability in root `go.mod`.
