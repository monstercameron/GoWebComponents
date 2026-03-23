# Master Build Script for GoWebComponents Examples
# Automatically discovers and builds all example WASM binaries

param(
    [string]$Example = "",
    [switch]$Verbose = $false
)

Write-Host "GoWebComponents Examples Build System" -ForegroundColor Cyan
Write-Host "======================================" -ForegroundColor Cyan
Write-Host ""

# Preserve caller environment and scope WASM settings to this script run.
$previousGOOS = $env:GOOS
$previousGOARCH = $env:GOARCH
$env:GOOS = 'js'
$env:GOARCH = 'wasm'

function Restore-BuildEnvironment {
    if ([string]::IsNullOrEmpty($previousGOOS)) {
        Remove-Item Env:GOOS -ErrorAction SilentlyContinue
    } else {
        $env:GOOS = $previousGOOS
    }

    if ([string]::IsNullOrEmpty($previousGOARCH)) {
        Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
    } else {
        $env:GOARCH = $previousGOARCH
    }
}

# Get script directory
$scriptDir = $PSScriptRoot
$repoRoot = Split-Path -Parent $scriptDir
$staticDir = Join-Path $scriptDir "static"

function Resolve-ExamplesBuildDir {
    $runnerConfigPath = $env:GWC_RUNNER_CONFIG
    if ([string]::IsNullOrWhiteSpace($runnerConfigPath)) {
        $localConfigPath = Join-Path $repoRoot "gwc-runner.json"
        if (Test-Path $localConfigPath) {
            $runnerConfigPath = $localConfigPath
        } else {
            $homeConfigPath = Join-Path $HOME ".gwc\runner.json"
            if (Test-Path $homeConfigPath) {
                $runnerConfigPath = $homeConfigPath
            }
        }
    }

    if (-not [string]::IsNullOrWhiteSpace($runnerConfigPath)) {
        if (-not [System.IO.Path]::IsPathRooted($runnerConfigPath)) {
            $runnerConfigPath = [System.IO.Path]::GetFullPath((Join-Path $repoRoot $runnerConfigPath))
        }
        if (Test-Path $runnerConfigPath) {
            $overrides = Get-Content $runnerConfigPath -Raw | ConvertFrom-Json
            $configuredBuildRoot = $overrides.paths.workspaceBuildRoot
            if (-not [string]::IsNullOrWhiteSpace($configuredBuildRoot)) {
                if ([System.IO.Path]::IsPathRooted($configuredBuildRoot)) {
                    return Join-Path $configuredBuildRoot "examples"
                }
                return Join-Path (Split-Path -Parent $runnerConfigPath) $configuredBuildRoot "examples"
            }
        }
    }

    return Join-Path $repoRoot "bin\examples"
}

Write-Host "[INFO] Refreshing shared Tailwind CSS" -ForegroundColor Yellow
Push-Location $staticDir
npm run build:css | Out-Null
$cssExitCode = $LASTEXITCODE
Pop-Location

if ($cssExitCode -ne 0) {
    Write-Host "[ERROR] Failed to rebuild shared Tailwind CSS" -ForegroundColor Red
    Restore-BuildEnvironment
    exit $cssExitCode
}

# Ensure bin directory exists
$binDir = Resolve-ExamplesBuildDir
if (-not (Test-Path $binDir)) {
    New-Item -ItemType Directory -Path $binDir -Force | Out-Null
    Write-Host "[OK] Created bin directory" -ForegroundColor Green
}

# Discover all example directories (numbered pattern)
$exampleDirs = Get-ChildItem -Path $scriptDir -Directory | 
    Where-Object { $_.Name -match '^\d{2}-' } | 
    Sort-Object Name

if ($exampleDirs.Count -eq 0) {
    Write-Host "[ERROR] No example directories found matching pattern '##-*'" -ForegroundColor Red
    Restore-BuildEnvironment
    exit 1
}

Write-Host "[INFO] Found $($exampleDirs.Count) examples" -ForegroundColor Yellow
Write-Host ""

# Filter to specific example if requested
if ($Example) {
    $exampleDirs = $exampleDirs | Where-Object { $_.Name -eq $Example }
    if ($exampleDirs.Count -eq 0) {
        Write-Host "[ERROR] Example '$Example' not found" -ForegroundColor Red
        Restore-BuildEnvironment
        exit 1
    }
    Write-Host "[INFO] Building specific example: $Example" -ForegroundColor Cyan
    Write-Host ""
}

# Build function
function Invoke-ExampleBuild {
    param([Parameter(Mandatory=$true)][System.IO.DirectoryInfo]$ExampleDir)
    
    $exampleName = $ExampleDir.Name
    $entryCandidates = @(
        Join-Path $ExampleDir.FullName "main.go"
        Join-Path $ExampleDir.FullName "client\main.go"
    )
    $entryGoPath = $entryCandidates | Where-Object { Test-Path $_ } | Select-Object -First 1

    if (-not $entryGoPath) {
        Write-Host "[SKIP] $exampleName - no supported main.go entrypoint found" -ForegroundColor Yellow
        return $null
    }

    $buildDir = Split-Path -Parent $entryGoPath
    
    # Extract clean name (remove number prefix)
    $cleanName = $exampleName -replace '^\d{2}-', ''
    $outputPath = Join-Path $binDir "$cleanName.wasm"
    
    Write-Host "[BUILD] $exampleName..." -ForegroundColor Yellow
    
    $startTime = Get-Date
    
    # Build the WASM binary
    Push-Location $buildDir
    # Use . to build all files in the package, handling multi-file examples like 11-blog
    $buildOutput = go build -o $outputPath . 2>&1
    $exitCode = $LASTEXITCODE
    Pop-Location
    
    $duration = (Get-Date) - $startTime
    
    if ($exitCode -eq 0) {
        $size = (Get-Item $outputPath).Length
        $sizeKB = [math]::Round($size / 1KB, 2)
        $durationSec = [math]::Round($duration.TotalSeconds, 2)
        Write-Host "  [OK] $cleanName.wasm ($sizeKB KB) in ${durationSec}s" -ForegroundColor Green
        return [PSCustomObject]@{
            Name = $exampleName
            CleanName = $cleanName
            Success = $true
            Size = $size
            Duration = $duration
        }
    } else {
        Write-Host "  [FAIL] Failed to build $exampleName" -ForegroundColor Red
        if ($Verbose) {
            Write-Host $buildOutput -ForegroundColor Red
        }
        return [PSCustomObject]@{
            Name = $exampleName
            CleanName = $cleanName
            Success = $false
            Size = 0
            Duration = $duration
        }
    }
}

# Build all examples
$results = @()

foreach ($dir in $exampleDirs) {
    $result = Invoke-ExampleBuild -ExampleDir $dir
    if ($result) {
        $results += $result
    }
}

# Summary
Write-Host ""
Write-Host "Build Summary" -ForegroundColor Cyan
Write-Host "=============" -ForegroundColor Cyan

$successful = ($results | Where-Object { $_.Success }).Count
$failed = ($results | Where-Object { -not $_.Success }).Count
$totalSize = ($results | Where-Object { $_.Success } | Measure-Object -Property Size -Sum).Sum
$totalSizeKB = [math]::Round($totalSize / 1KB, 2)

Write-Host ""
Write-Host "Successful: $successful" -ForegroundColor Green
Write-Host "Failed: $failed" -ForegroundColor $(if ($failed -gt 0) { "Red" } else { "Gray" })
Write-Host "Total size: $totalSizeKB KB" -ForegroundColor Cyan
Write-Host ""

if ($successful -gt 0) {
    Write-Host "Built WASM files:" -ForegroundColor Yellow
    $results | Where-Object { $_.Success } | ForEach-Object {
        $sizeKB = [math]::Round($_.Size / 1KB, 2)
        Write-Host "  - $($_.CleanName).wasm - $sizeKB KB" -ForegroundColor Gray
    }
}

Write-Host ""

Restore-BuildEnvironment

if ($failed -eq 0) {
    Write-Host "All examples built successfully!" -ForegroundColor Green
    exit 0
} else {
    Write-Host "Some examples failed to build" -ForegroundColor Yellow
    Write-Host "Run with -Verbose flag for detailed error messages" -ForegroundColor Gray
    exit 1
}
