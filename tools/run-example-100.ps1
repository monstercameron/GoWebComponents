param(
    [switch]$BuildClient,
    [string]$ListenAddr
)

$ErrorActionPreference = "Stop"

$scriptPath = Join-Path $PSScriptRoot "..\examples\100-ai-chat-wizard\scripts\run-server.ps1"

if (-not (Test-Path $scriptPath)) {
    throw "Could not find example 100 server script at $scriptPath"
}

& $scriptPath -BuildClient:$BuildClient -ListenAddr $ListenAddr
