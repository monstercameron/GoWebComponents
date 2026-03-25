param()

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..\..")

Push-Location $repoRoot
try {
    go run ./examples/100-ai-chat-wizard/cmd/build-client
}
finally {
    Pop-Location
}
