# Development server wrapper for path-based Go WASM apps.
# Builds the live-reload server and runs it against a specific main.go or app directory.

[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Main,

    [string]$Root = "",
    [string]$Index = "",
    [string]$Output = "",
    [string]$ListenHost = "127.0.0.1",
    [string]$Port = "8080",
    [switch]$NoHotReload
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = (Resolve-Path (Join-Path $ScriptDir "..")).Path
$ServerDir = Join-Path $ScriptDir "livereload"
$BinaryPath = Join-Path $ServerDir "livereload.exe"

if (-not (Test-Path $ServerDir)) {
    Write-Host "Error: live reload server directory not found at $ServerDir" -ForegroundColor Red
    exit 1
}

$goCmd = Get-Command go -ErrorAction SilentlyContinue
if ($null -eq $goCmd) {
    Write-Host "Error: Go is required to run the dev server." -ForegroundColor Red
    exit 1
}

Push-Location $ServerDir
try {
    Write-Host "Building live reload server..." -ForegroundColor Yellow
    go build -o livereload.exe .
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Error: live reload server build failed." -ForegroundColor Red
        exit $LASTEXITCODE
    }
}
finally {
    Pop-Location
}

if ($Root -eq "") {
    $Root = Split-Path -Parent $Main
}

$args = @(
    "-main", $Main,
    "-root", $Root,
    "-host", $ListenHost,
    "-port", $Port
)

if ($Index -ne "") {
    $args += @("-index", $Index)
}
if ($Output -ne "") {
    $args += @("-output", $Output)
}
if ($NoHotReload) {
    $args += "-hot=false"
}

Write-Host "Starting hot-reload dev server..." -ForegroundColor Green
Write-Host "Main:  $Main" -ForegroundColor Cyan
Write-Host "Root:  $Root" -ForegroundColor Cyan
Write-Host "Press Ctrl+C to stop" -ForegroundColor Yellow

Push-Location $RepoRoot
try {
    & $BinaryPath @args
    exit $LASTEXITCODE
}
finally {
    Pop-Location
}
