param(
    [string]$Package = ".",
    [string]$OutDir = "bin/wasm-release",
    [string]$BinaryName = "app.wasm",
    [string]$ManifestName = "wasm-release-manifest.json",
    [string]$BudgetsPath = "",
    [string]$LdFlags = "-s -w",
    [switch]$SkipCompression,
    [switch]$KeepBuildInfo
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$scriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$brotliHelperScript = Join-Path $scriptRoot "write-brotli-file.mjs"

function New-DirectoryIfMissing {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Path $Path | Out-Null
    }
}

function Get-Sha256Hex {
    param([string]$Path)
    return (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()
}

function Write-GZipFile {
    param(
        [string]$SourcePath,
        [string]$TargetPath
    )

    $inputBytes = [System.IO.File]::ReadAllBytes($SourcePath)
    $outputStream = [System.IO.File]::Create($TargetPath)
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
}

function Write-BrotliFile {
    param(
        [string]$SourcePath,
        [string]$TargetPath
    )

    if (Test-DotNetBrotliSupport) {
        $inputBytes = [System.IO.File]::ReadAllBytes($SourcePath)
        $outputStream = [System.IO.File]::Create($TargetPath)
        try {
            $brotli = New-Object System.IO.Compression.BrotliStream($outputStream, [System.IO.Compression.CompressionLevel]::Optimal)
            try {
                $brotli.Write($inputBytes, 0, $inputBytes.Length)
            } finally {
                $brotli.Dispose()
            }
        } finally {
            $outputStream.Dispose()
        }
        return
    }

    if (Test-NodeBrotliSupport) {
        & node $brotliHelperScript $SourcePath $TargetPath
        if ($LASTEXITCODE -ne 0) {
            throw "Node Brotli compression failed"
        }
        return
    }

    throw "Brotli compression is unavailable on this host"
}

function Test-DotNetBrotliSupport {
    return $null -ne ("System.IO.Compression.BrotliStream" -as [type])
}

function Test-NodeBrotliSupport {
    return $null -ne (Get-Command node -ErrorAction SilentlyContinue) -and (Test-Path -LiteralPath $brotliHelperScript)
}

function Test-BrotliSupport {
    return (Test-DotNetBrotliSupport) -or (Test-NodeBrotliSupport)
}

function Get-ArtifactRecord {
    param(
        [string]$BaseDir,
        [string]$ArtifactPath
    )

    $item = Get-Item -LiteralPath $ArtifactPath
    $baseUri = New-Object System.Uri(([System.IO.Path]::GetFullPath($BaseDir).TrimEnd('\') + '\'))
    $artifactUri = New-Object System.Uri([System.IO.Path]::GetFullPath($ArtifactPath))
    $relativePath = $baseUri.MakeRelativeUri($artifactUri).ToString()
    return [ordered]@{
        path   = [System.Uri]::UnescapeDataString($relativePath)
        bytes  = [int64]$item.Length
        sha256 = Get-Sha256Hex -Path $ArtifactPath
    }
}

function Assert-Budgets {
    param(
        [hashtable]$Budgets,
        [hashtable]$Artifacts
    )

    $checks = @(
        @{ key = "raw_bytes"; artifact = "wasm"; label = "raw wasm" },
        @{ key = "gzip_bytes"; artifact = "gzip"; label = "gzip sidecar" },
        @{ key = "brotli_bytes"; artifact = "brotli"; label = "brotli sidecar" }
    )

    foreach ($check in $checks) {
        if (-not $Budgets.ContainsKey($check.key)) {
            continue
        }
        if (-not $Artifacts.ContainsKey($check.artifact)) {
            continue
        }

        $limit = [int64]$Budgets[$check.key]
        $actual = [int64]$Artifacts[$check.artifact].bytes
        if ($actual -gt $limit) {
            throw ("Artifact budget exceeded for {0}: {1} bytes > {2} bytes" -f $check.label, $actual, $limit)
        }
    }
}

function ConvertTo-HashtableCompat {
    param([object]$Value)

    if ($null -eq $Value) {
        return $null
    }

    if ($Value -is [System.Collections.IDictionary]) {
        $map = @{}
        foreach ($key in $Value.Keys) {
            $map[$key] = ConvertTo-HashtableCompat -Value $Value[$key]
        }
        return $map
    }

    if ($Value -is [System.Collections.IEnumerable] -and -not ($Value -is [string])) {
        $items = @()
        foreach ($item in $Value) {
            $items += ,(ConvertTo-HashtableCompat -Value $item)
        }
        return $items
    }

    if ($Value -is [pscustomobject]) {
        $map = @{}
        foreach ($property in $Value.PSObject.Properties) {
            $map[$property.Name] = ConvertTo-HashtableCompat -Value $property.Value
        }
        return $map
    }

    return $Value
}

$resolvedOutDir = [System.IO.Path]::GetFullPath((Join-Path (Get-Location) $OutDir))
New-DirectoryIfMissing -Path $resolvedOutDir

$wasmPath = Join-Path $resolvedOutDir $BinaryName
$gzipPath = "$wasmPath.gz"
$brotliPath = "$wasmPath.br"
$manifestPath = Join-Path $resolvedOutDir $ManifestName

$oldGoos = $env:GOOS
$oldGoarch = $env:GOARCH
try {
    $env:GOOS = "js"
    $env:GOARCH = "wasm"

    $buildArgs = @("build", "-o", $wasmPath)
    if (-not $KeepBuildInfo) {
        $buildArgs += "-trimpath"
        if ($LdFlags -ne "") {
            $buildArgs += "-ldflags=$LdFlags"
        }
        $buildArgs += "-buildvcs=false"
    }
    $buildArgs += $Package

    & go @buildArgs
    if ($LASTEXITCODE -ne 0) {
        throw "go build failed"
    }
} finally {
    $env:GOOS = $oldGoos
    $env:GOARCH = $oldGoarch
}

$artifacts = [ordered]@{}
$artifacts["wasm"] = Get-ArtifactRecord -BaseDir $resolvedOutDir -ArtifactPath $wasmPath

if (-not $SkipCompression) {
    Write-GZipFile -SourcePath $wasmPath -TargetPath $gzipPath
    $artifacts["gzip"] = Get-ArtifactRecord -BaseDir $resolvedOutDir -ArtifactPath $gzipPath
    if (Test-BrotliSupport) {
        Write-BrotliFile -SourcePath $wasmPath -TargetPath $brotliPath
        $artifacts["brotli"] = Get-ArtifactRecord -BaseDir $resolvedOutDir -ArtifactPath $brotliPath
    } else {
        Write-Warning "Brotli sidecar skipped because neither the PowerShell runtime nor Node-based fallback compression is available."
    }
}

if ($BudgetsPath -ne "") {
    $resolvedBudgetsPath = [System.IO.Path]::GetFullPath((Join-Path (Get-Location) $BudgetsPath))
    $budgetObject = ConvertTo-HashtableCompat -Value (Get-Content -LiteralPath $resolvedBudgetsPath -Raw | ConvertFrom-Json)
    Assert-Budgets -Budgets $budgetObject -Artifacts $artifacts
}

$manifest = [ordered]@{
    package   = $Package
    profile   = if ($KeepBuildInfo) { "debug" } else { "production" }
    goos      = "js"
    goarch    = "wasm"
    flags     = [ordered]@{
        trimpath    = (-not $KeepBuildInfo)
        ldflags     = if ($KeepBuildInfo) { "" } else { $LdFlags }
        buildvcs    = if ($KeepBuildInfo) { "default" } else { "false" }
        compression = (-not $SkipCompression)
    }
    artifacts = $artifacts
}

$manifest | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath $manifestPath -Encoding utf8

Write-Host ("Built wasm release artifacts in {0}" -f $resolvedOutDir) -ForegroundColor Green
Write-Host ("Manifest: {0}" -f $manifestPath) -ForegroundColor Green
