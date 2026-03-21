# Development server wrapper for path-based Go WASM apps.
# Builds the live-reload server and runs it against a specific main.go or app directory.

[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [Alias("Main")]
    [string]$App,

    [string]$Root = "",
    [Alias("Index")]
    [string]$Html = "",
    [Alias("Output")]
    [string]$Wasm = "",
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

$previousGoos = $null
$previousGoarch = $null
$hadGoos = Test-Path Env:\GOOS
$hadGoarch = Test-Path Env:\GOARCH
if ($hadGoos) {
    $previousGoos = $env:GOOS
    Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
}
if ($hadGoarch) {
    $previousGoarch = $env:GOARCH
    Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
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
    if ($hadGoos) {
        $env:GOOS = $previousGoos
    } else {
        Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
    }
    if ($hadGoarch) {
        $env:GOARCH = $previousGoarch
    } else {
        Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
    }
    Pop-Location
}

if ($Root -eq "") {
    $Root = Split-Path -Parent $App
}

$args = @(
    "-app", $App,
    "-root", $Root,
    "-host", $ListenHost,
    "-port", $Port
)

if ($Html -ne "") {
    $args += @("-html", $Html)
}
if ($Wasm -ne "") {
    $args += @("-wasm", $Wasm)
}
if ($NoHotReload) {
    $args += "-hot=false"
}

Write-Host "Starting hot-reload dev server..." -ForegroundColor Green
Write-Host "App:   $App" -ForegroundColor Cyan
Write-Host "Root:  $Root" -ForegroundColor Cyan
if ($Html -ne "") {
    Write-Host "HTML:  $Html" -ForegroundColor Cyan
}
if ($Wasm -ne "") {
    Write-Host "WASM:  $Wasm" -ForegroundColor Cyan
}
Write-Host "Press Ctrl+C to stop" -ForegroundColor Yellow

Push-Location $RepoRoot
try {
    if ($hadGoos) {
        Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
    }
    if ($hadGoarch) {
        Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
    }
    & $BinaryPath @args
    exit $LASTEXITCODE
}
finally {
    if ($hadGoos) {
        $env:GOOS = $previousGoos
    }
    if ($hadGoarch) {
        $env:GOARCH = $previousGoarch
    }
    Pop-Location
}
