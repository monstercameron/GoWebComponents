param(
    [string]$ListenAddr
)

$ErrorActionPreference = "Stop"

$runServerArgs = @("-BuildClient")
if ($ListenAddr) {
    $runServerArgs += @("-ListenAddr", $ListenAddr)
}

& (Join-Path $PSScriptRoot "run-server.ps1") @runServerArgs
