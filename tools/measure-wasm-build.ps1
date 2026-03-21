param(
    [string]$Package = ".",
    [string]$OutDir = "dist/wasm-build-experiment",
    [string]$BinaryName = "app.wasm",
    [string]$ManifestName = "wasm-build-experiment.json",
    [string]$GoExecutable = "go",
    [string]$LdFlags = "-s -w",
    [switch]$ReleaseProfile,
    [switch]$SkipCompression,
    [long]$ServeReloadMs
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

$resolvedOutDirInput = if ([System.IO.Path]::IsPathRooted($OutDir)) {
    $OutDir
} else {
    Join-Path (Get-Location) $OutDir
}
$resolvedOutDir = [System.IO.Path]::GetFullPath($resolvedOutDirInput)
New-DirectoryIfMissing -Path $resolvedOutDir

$wasmPath = Join-Path $resolvedOutDir $BinaryName
$gzipPath = "$wasmPath.gz"
$brotliPath = "$wasmPath.br"
$manifestPath = Join-Path $resolvedOutDir $ManifestName

$phaseTimings = [ordered]@{}
$totalStopwatch = [System.Diagnostics.Stopwatch]::StartNew()

$oldGoos = $env:GOOS
$oldGoarch = $env:GOARCH
$buildArgs = @("build", "-o", $wasmPath)
if ($ReleaseProfile) {
    $buildArgs += "-trimpath"
    if ($LdFlags -ne "") {
        $buildArgs += "-ldflags=$LdFlags"
    }
    $buildArgs += "-buildvcs=false"
}
$buildArgs += $Package

try {
    $env:GOOS = "js"
    $env:GOARCH = "wasm"

    $buildStopwatch = [System.Diagnostics.Stopwatch]::StartNew()
    & $GoExecutable @buildArgs
    $buildStopwatch.Stop()
    $phaseTimings["go_build_ms"] = [int64]$buildStopwatch.ElapsedMilliseconds

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
    $compressionStopwatch = [System.Diagnostics.Stopwatch]::StartNew()

    $gzipStopwatch = [System.Diagnostics.Stopwatch]::StartNew()
    Write-GZipFile -SourcePath $wasmPath -TargetPath $gzipPath
    $gzipStopwatch.Stop()
    $phaseTimings["gzip_ms"] = [int64]$gzipStopwatch.ElapsedMilliseconds
    $artifacts["gzip"] = Get-ArtifactRecord -BaseDir $resolvedOutDir -ArtifactPath $gzipPath

    if (Test-BrotliSupport) {
        $brotliStopwatch = [System.Diagnostics.Stopwatch]::StartNew()
        Write-BrotliFile -SourcePath $wasmPath -TargetPath $brotliPath
        $brotliStopwatch.Stop()
        $phaseTimings["brotli_ms"] = [int64]$brotliStopwatch.ElapsedMilliseconds
        $artifacts["brotli"] = Get-ArtifactRecord -BaseDir $resolvedOutDir -ArtifactPath $brotliPath
    }

    $compressionStopwatch.Stop()
    $phaseTimings["compression_total_ms"] = [int64]$compressionStopwatch.ElapsedMilliseconds
} else {
    $phaseTimings["compression_total_ms"] = 0
}

if ($PSBoundParameters.ContainsKey("ServeReloadMs")) {
    $phaseTimings["serve_reload_ms"] = [int64]$ServeReloadMs
}

$totalStopwatch.Stop()
$phaseTimings["total_wall_ms"] = [int64]$totalStopwatch.ElapsedMilliseconds

$manifest = [ordered]@{
    package       = $Package
    profile       = if ($ReleaseProfile) { "release" } else { "debug" }
    go_executable = $GoExecutable
    go_version    = (& $GoExecutable version)
    goos          = "js"
    goarch        = "wasm"
    build_args    = $buildArgs
    phases        = $phaseTimings
    artifacts     = $artifacts
}

$manifest | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath $manifestPath -Encoding utf8

Write-Host ("Measured wasm build phases for {0}" -f $Package) -ForegroundColor Green
Write-Host ("Manifest: {0}" -f $manifestPath) -ForegroundColor Green
