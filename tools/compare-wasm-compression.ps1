param(
    [string]$Package = "./examples/21-ui-render",
    [string]$OutDir = "tmp/wasm-compression-comparison",
    [string]$BinaryName = "app.wasm",
    [string]$SummaryName = "wasm-compression-comparison.json"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function New-DirectoryIfMissing {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Path $Path | Out-Null
    }
}

function Read-JsonFile {
    param([string]$Path)
    return Get-Content -LiteralPath $Path -Raw | ConvertFrom-Json
}

function Get-Sha256Hex {
    param([string]$Path)
    return (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()
}

function Get-ArtifactRecord {
    param(
        [string]$BaseDir,
        [string]$ArtifactPath
    )

    $item = Get-Item -LiteralPath $ArtifactPath
    $baseUri = New-Object System.Uri(([System.IO.Path]::GetFullPath($BaseDir).TrimEnd('\\') + '\\'))
    $artifactUri = New-Object System.Uri([System.IO.Path]::GetFullPath($ArtifactPath))
    $relativePath = $baseUri.MakeRelativeUri($artifactUri).ToString()

    return [ordered]@{
        path   = [System.Uri]::UnescapeDataString($relativePath)
        bytes  = [int64]$item.Length
        sha256 = Get-Sha256Hex -Path $ArtifactPath
    }
}

function Get-WasmOptCommandInfo {
    $direct = Get-Command wasm-opt -ErrorAction SilentlyContinue
    if ($null -ne $direct) {
        return @{
            available = $true
            mode = "path"
            label = $direct.Source
        }
    }

    if ($null -ne (Get-Command npx -ErrorAction SilentlyContinue)) {
        return @{
            available = $true
            mode = "npx-binaryen"
            label = "npx --yes --package binaryen wasm-opt"
        }
    }

    return @{
        available = $false
        mode = "unavailable"
        label = ""
    }
}

function Invoke-WasmOpt {
    param(
        [Parameter(Mandatory = $true)][string]$SourcePath,
        [Parameter(Mandatory = $true)][string]$TargetPath,
        [Parameter(Mandatory = $true)]$CommandInfo
    )

    if (-not $CommandInfo.available) {
        throw "wasm-opt is unavailable"
    }

    $args = @($SourcePath, "-Oz", "-o", $TargetPath)
    if ($CommandInfo.mode -eq "path") {
        & wasm-opt @args
    } else {
        & npx --yes --package binaryen wasm-opt @args
    }

    if ($LASTEXITCODE -ne 0) {
        throw "wasm-opt failed"
    }
}

$scriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$measureScript = Join-Path $scriptRoot "measure-wasm-build.ps1"
$resolvedOutDir = [System.IO.Path]::GetFullPath((Join-Path (Get-Location) $OutDir))
New-DirectoryIfMissing -Path $resolvedOutDir

$variantDefinitions = @(
    [ordered]@{
        key = "plain_raw"
        label = "plain go build"
        out_dir = Join-Path $resolvedOutDir "plain-raw"
        params = @{
            Package = $Package
            OutDir = Join-Path $resolvedOutDir "plain-raw"
            BinaryName = $BinaryName
            ManifestName = "wasm-build-experiment.json"
            SkipCompression = $true
        }
    },
    [ordered]@{
        key = "stripped_raw"
        label = "release raw"
        out_dir = Join-Path $resolvedOutDir "stripped-raw"
        params = @{
            Package = $Package
            OutDir = Join-Path $resolvedOutDir "stripped-raw"
            BinaryName = $BinaryName
            ManifestName = "wasm-build-experiment.json"
            ReleaseProfile = $true
            SkipCompression = $true
        }
    },
    [ordered]@{
        key = "stripped_compressed"
        label = "release plus compression"
        out_dir = Join-Path $resolvedOutDir "stripped-compressed"
        params = @{
            Package = $Package
            OutDir = Join-Path $resolvedOutDir "stripped-compressed"
            BinaryName = $BinaryName
            ManifestName = "wasm-build-experiment.json"
            ReleaseProfile = $true
        }
    }
)

$variants = [ordered]@{}
foreach ($definition in $variantDefinitions) {
    $variantParams = $definition.params
    & $measureScript @variantParams
    if ($LASTEXITCODE -ne 0) {
        throw "variant measurement failed: $($definition.key)"
    }

    $manifestPath = Join-Path (Join-Path $resolvedOutDir ($definition.key -replace '_', '-')) "wasm-build-experiment.json"
    if (-not (Test-Path -LiteralPath $manifestPath)) {
        $manifestPath = Join-Path $definition.out_dir "wasm-build-experiment.json"
    }
    $manifest = Read-JsonFile -Path $manifestPath
    $variants[$definition.key] = $manifest
}

$brotliSupported = $null -ne ($variants["stripped_compressed"].artifacts.brotli)
$wasmOptCommand = Get-WasmOptCommandInfo

if ($wasmOptCommand.available) {
    $strippedRawManifest = $variants["stripped_raw"]
    $strippedRawDir = Join-Path $resolvedOutDir "stripped-raw"
    $strippedWasmPath = Join-Path $strippedRawDir $BinaryName

    $optimizedRawDir = Join-Path $resolvedOutDir "optimized-raw"
    $optimizedCompressedDir = Join-Path $resolvedOutDir "optimized-compressed"
    New-DirectoryIfMissing -Path $optimizedRawDir
    New-DirectoryIfMissing -Path $optimizedCompressedDir

    $optimizedRawWasmPath = Join-Path $optimizedRawDir $BinaryName
    $optimizedCompressedWasmPath = Join-Path $optimizedCompressedDir $BinaryName

    $optTimer = [System.Diagnostics.Stopwatch]::StartNew()
    Invoke-WasmOpt -SourcePath $strippedWasmPath -TargetPath $optimizedRawWasmPath -CommandInfo $wasmOptCommand
    $optTimer.Stop()

    Copy-Item -LiteralPath $optimizedRawWasmPath -Destination $optimizedCompressedWasmPath -Force
    $optimizedCompressedManifest = Read-JsonFile -Path (Join-Path $resolvedOutDir "stripped-compressed\wasm-build-experiment.json")
    $gzipTimer = [System.Diagnostics.Stopwatch]::StartNew()
    $gzipPath = "$optimizedCompressedWasmPath.gz"
    $brotliPath = "$optimizedCompressedWasmPath.br"
    $inputBytes = [System.IO.File]::ReadAllBytes($optimizedCompressedWasmPath)
    $outputStream = [System.IO.File]::Create($gzipPath)
    try {
        $gzip = New-Object System.IO.Compression.GZipStream($outputStream, [System.IO.Compression.CompressionLevel]::Optimal)
        try {
            $gzip.Write($inputBytes, 0, $inputBytes.Length)
        } finally {
            $gzip.Dispose()
        }
    } finally {
        $outputStream.Dispose()
    }
    $gzipTimer.Stop()

    $brotliMs = $null
    if ($brotliSupported) {
        $brotliTimer = [System.Diagnostics.Stopwatch]::StartNew()
        & node (Join-Path $scriptRoot "write-brotli-file.mjs") $optimizedCompressedWasmPath $brotliPath
        if ($LASTEXITCODE -ne 0) {
            throw "optimized brotli compression failed"
        }
        $brotliTimer.Stop()
        $brotliMs = [int64]$brotliTimer.ElapsedMilliseconds
    }

    $variants["optimized_raw"] = [ordered]@{
        package = $Package
        profile = "release-optimized"
        go_version = $strippedRawManifest.go_version
        goos = "js"
        goarch = "wasm"
        build_args = $strippedRawManifest.build_args
        optimizer = [ordered]@{
            tool = $wasmOptCommand.label
            args = @($strippedWasmPath, "-Oz", "-o", $optimizedRawWasmPath)
        }
        phases = [ordered]@{
            go_build_ms = [int64]$strippedRawManifest.phases.go_build_ms
            wasm_opt_ms = [int64]$optTimer.ElapsedMilliseconds
            compression_total_ms = 0
            total_wall_ms = [int64]$strippedRawManifest.phases.go_build_ms + [int64]$optTimer.ElapsedMilliseconds
        }
        artifacts = [ordered]@{
            wasm = Get-ArtifactRecord -BaseDir $optimizedRawDir -ArtifactPath $optimizedRawWasmPath
        }
    }

    $optimizedCompressedArtifacts = [ordered]@{
        wasm = Get-ArtifactRecord -BaseDir $optimizedCompressedDir -ArtifactPath $optimizedCompressedWasmPath
        gzip = Get-ArtifactRecord -BaseDir $optimizedCompressedDir -ArtifactPath $gzipPath
    }
    if ($brotliSupported) {
        $optimizedCompressedArtifacts["brotli"] = Get-ArtifactRecord -BaseDir $optimizedCompressedDir -ArtifactPath $brotliPath
    }

    $optimizedCompressionTotal = [int64]$gzipTimer.ElapsedMilliseconds
    if ($null -ne $brotliMs) {
        $optimizedCompressionTotal += $brotliMs
    }

    $variants["optimized_compressed"] = [ordered]@{
        package = $Package
        profile = "release-optimized"
        go_version = $strippedRawManifest.go_version
        goos = "js"
        goarch = "wasm"
        build_args = $strippedRawManifest.build_args
        optimizer = [ordered]@{
            tool = $wasmOptCommand.label
            args = @($strippedWasmPath, "-Oz", "-o", $optimizedCompressedWasmPath)
        }
        phases = [ordered]@{
            go_build_ms = [int64]$strippedRawManifest.phases.go_build_ms
            wasm_opt_ms = [int64]$optTimer.ElapsedMilliseconds
            gzip_ms = [int64]$gzipTimer.ElapsedMilliseconds
            brotli_ms = $brotliMs
            compression_total_ms = $optimizedCompressionTotal
            total_wall_ms = [int64]$strippedRawManifest.phases.go_build_ms + [int64]$optTimer.ElapsedMilliseconds + $optimizedCompressionTotal
        }
        artifacts = $optimizedCompressedArtifacts
    }
}

$summary = [ordered]@{
    package = $Package
    generated_at = (Get-Date).ToString("o")
    environment = [ordered]@{
        go_version = (& go version)
        brotli_supported = $brotliSupported
        wasm_opt_available = $wasmOptCommand.available
        wasm_opt_path = $wasmOptCommand.label
    }
    variants = $variants
    unsupported = [ordered]@{
        brotli_delivery = if ($brotliSupported) { "supported" } else { "neither the PowerShell runtime nor Node-based fallback compression is available" }
        optimized_wasm = if ($wasmOptCommand.available) { "supported" } else { "wasm-opt is unavailable and npx binaryen fallback could not be resolved" }
    }
}

$summaryPath = Join-Path $resolvedOutDir $SummaryName
$summary | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $summaryPath -Encoding utf8

Write-Host ("Wrote wasm compression comparison to {0}" -f $summaryPath) -ForegroundColor Green
