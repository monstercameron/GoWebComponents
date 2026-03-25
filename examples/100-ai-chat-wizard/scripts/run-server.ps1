param(
    [switch]$BuildClient,
    [string]$ListenAddr
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..\..")

Push-Location $repoRoot
try {
    if ($BuildClient) {
        & (Join-Path $PSScriptRoot "build-client.ps1")
    }

    if ($ListenAddr) {
        $env:LISTEN_ADDR = $ListenAddr
    }

    go run ./examples/100-ai-chat-wizard/cmd/server
}
finally {
    Pop-Location
}
