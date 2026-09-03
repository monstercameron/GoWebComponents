[CmdletBinding()]
param(
    [switch]$SkipDoctor,
    [switch]$SkipBridgeChecks,
    [switch]$SkipPlaywrightCheck
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

function Invoke-BootstrapStep {
    param(
        [Parameter(Mandatory = $true)][string]$StepName,
        [Parameter(Mandatory = $true)][scriptblock]$StepAction
    )

    Write-Host ("==> " + $StepName)
    & $StepAction
}

function Assert-CommandAvailable {
    param(
        [Parameter(Mandatory = $true)][string]$CommandName,
        [Parameter(Mandatory = $true)][string]$InstallHint
    )

    if (-not (Get-Command $CommandName -ErrorAction SilentlyContinue)) {
        throw ("Missing required command '" + $CommandName + "'. " + $InstallHint)
    }
}

if (-not (Test-Path -Path ".\third_party\GoGRPCBridge")) {
    throw "Expected third_party/GoGRPCBridge in repo root. Run this script from repository root."
}

Invoke-BootstrapStep -StepName "Verify required command-line dependencies" -StepAction {
    Assert-CommandAvailable -CommandName "go" -InstallHint "Install Go and ensure it is on PATH."
    Assert-CommandAvailable -CommandName "git" -InstallHint "Install Git and ensure it is on PATH."
    Assert-CommandAvailable -CommandName "protoc" -InstallHint "Install Protocol Buffers compiler (protoc) and ensure it is on PATH."
    Assert-CommandAvailable -CommandName "protoc-gen-go" -InstallHint "Install with: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
    Assert-CommandAvailable -CommandName "protoc-gen-go-grpc" -InstallHint "Install with: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
}

if (-not $SkipPlaywrightCheck) {
    Invoke-BootstrapStep -StepName "Verify Playwright Go CLI availability" -StepAction {
        try {
            go run github.com/mxschmitt/playwright-go/cmd/playwright@latest --version
        }
        catch {
            throw "Playwright CLI is not available. Install browser runtime with: go run github.com/mxschmitt/playwright-go/cmd/playwright@latest install --with-deps chromium"
        }
    }
}

if (-not $SkipDoctor) {
    Invoke-BootstrapStep -StepName "Run gwc doctor (toolchain/runtime checks)" -StepAction {
        go run ./tools/gwc doctor
    }
}

Invoke-BootstrapStep -StepName "Initialize GoGRPCBridge submodule" -StepAction {
    git submodule update --init --recursive third_party/GoGRPCBridge
}

Invoke-BootstrapStep -StepName "Verify GoGRPCBridge submodule status" -StepAction {
    git submodule status -- third_party/GoGRPCBridge
}

Invoke-BootstrapStep -StepName "Verify GoGRPCBridge runner command surface" -StepAction {
    go run ./third_party/GoGRPCBridge/tools/runner.go help
}

if (-not $SkipBridgeChecks) {
    Invoke-BootstrapStep -StepName "Run GoGRPCBridge quick checks" -StepAction {
        go run ./third_party/GoGRPCBridge/tools/runner.go test-short
    }
}

Write-Host "GoGRPCBridge bootstrap complete."
