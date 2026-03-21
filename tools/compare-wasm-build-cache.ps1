param(
    [string]$Package = "./examples/21-ui-render",
    [string]$OutDir = "tmp/wasm-build-cache-comparison",
    [string]$BinaryName = "app.wasm",
    [string]$SummaryName = "wasm-build-cache-comparison.json",
    [switch]$ReleaseProfile
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function New-DirectoryIfMissing {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Path $Path | Out-Null
    }
}

function Reset-Directory {
    param([string]$Path)
    if (Test-Path -LiteralPath $Path) {
        Remove-Item -LiteralPath $Path -Recurse -Force
    }
    New-Item -ItemType Directory -Path $Path | Out-Null
}

function Read-JsonFile {
    param([string]$Path)
    return Get-Content -LiteralPath $Path -Raw | ConvertFrom-Json
}

function Resolve-PackageEditFile {
    param([string]$PackagePath)

    $resolvedPackagePath = if ([System.IO.Path]::IsPathRooted($PackagePath)) {
        $PackagePath
    } else {
        Join-Path (Get-Location) $PackagePath
    }
    $resolvedPackagePath = [System.IO.Path]::GetFullPath($resolvedPackagePath)

    if (-not (Test-Path -LiteralPath $resolvedPackagePath -PathType Container)) {
        throw "Small-edit measurement requires a package directory path: $PackagePath"
    }

    $candidate = Get-ChildItem -LiteralPath $resolvedPackagePath -Filter *.go -File |
        Where-Object { -not $_.Name.EndsWith('_test.go') } |
        Sort-Object Name |
        Select-Object -First 1

    if ($null -eq $candidate) {
        throw "No editable Go source file found for package: $PackagePath"
    }

    return $candidate.FullName
}

function Use-EnvValue {
    param(
        [string]$Name,
        [AllowNull()][string]$Value
    )

    if ($null -eq $Value -or $Value -eq "") {
        Remove-Item ("Env:\" + $Name) -ErrorAction SilentlyContinue
    } else {
        Set-Item ("Env:\" + $Name) -Value $Value
    }
}

function Invoke-VariantMeasurement {
    param(
        [hashtable]$Variant,
        [string]$MeasureScript,
        [string]$ResolvedOutDir,
        [string]$DefaultGOMODCACHE,
        [bool]$ReleaseProfileEnabled,
        [string]$BinaryNameValue
    )

    $variantOutDir = Join-Path $ResolvedOutDir $Variant.key
    New-DirectoryIfMissing -Path $variantOutDir

    $oldGoCache = $env:GOCACHE
    $oldGoModCache = $env:GOMODCACHE

    try {
        Use-EnvValue -Name "GOCACHE" -Value $Variant.gocache
        Use-EnvValue -Name "GOMODCACHE" -Value $Variant.gomodcache

        $result = [ordered]@{
            label = $Variant.label
            gocache = $Variant.gocache
            gomodcache = $Variant.gomodcache
            status = "pending"
            notes = @($Variant.note)
        }

        if ($Variant.ContainsKey("prepareModuleCache") -and $Variant.prepareModuleCache) {
            $downloadTimer = [System.Diagnostics.Stopwatch]::StartNew()
            & go mod download
            $downloadExit = $LASTEXITCODE
            $downloadTimer.Stop()
            $result["module_download_ms"] = [int64]$downloadTimer.ElapsedMilliseconds

            if ($downloadExit -ne 0) {
                $result["status"] = "failed"
                $result["error"] = "go mod download failed"
                return $result
            }
        }

        $editFilePath = $null
        $originalEditBytes = $null
        if ($Variant.ContainsKey("smallEdit") -and $Variant.smallEdit) {
            $editFilePath = Resolve-PackageEditFile -PackagePath $Package
            $originalEditBytes = [System.IO.File]::ReadAllBytes($editFilePath)
            $markerBytes = [System.Text.Encoding]::UTF8.GetBytes([Environment]::NewLine + "// cache experiment marker" + [Environment]::NewLine)
            $updatedBytes = New-Object byte[] ($originalEditBytes.Length + $markerBytes.Length)
            [Array]::Copy($originalEditBytes, 0, $updatedBytes, 0, $originalEditBytes.Length)
            [Array]::Copy($markerBytes, 0, $updatedBytes, $originalEditBytes.Length, $markerBytes.Length)
            [System.IO.File]::WriteAllBytes($editFilePath, $updatedBytes)
            $result["edited_file"] = $editFilePath
            $result["notes"] += "A temporary comment edit was applied and restored to measure rebuild invalidation."
        }

        try {
            $measureParams = @{
                Package = $Package
                OutDir = $variantOutDir
                BinaryName = $BinaryNameValue
                ManifestName = "wasm-build-experiment.json"
            }
            if ($ReleaseProfileEnabled) {
                $measureParams["ReleaseProfile"] = $true
            }

            & $MeasureScript @measureParams
            $measureExit = $LASTEXITCODE
            if ($measureExit -ne 0) {
                $result["status"] = "failed"
                $result["error"] = "measure-wasm-build.ps1 failed"
                return $result
            }

            $manifest = Read-JsonFile -Path (Join-Path $variantOutDir "wasm-build-experiment.json")
            $result["status"] = "ok"
            $result["measurement"] = $manifest
            return $result
        }
        finally {
            if ($null -ne $editFilePath) {
                [System.IO.File]::WriteAllBytes($editFilePath, $originalEditBytes)
            }
        }
    }
    finally {
        Use-EnvValue -Name "GOCACHE" -Value $oldGoCache
        Use-EnvValue -Name "GOMODCACHE" -Value $oldGoModCache
    }
}

$scriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$measureScript = Join-Path $scriptRoot "measure-wasm-build.ps1"

$resolvedOutDirInput = if ([System.IO.Path]::IsPathRooted($OutDir)) {
    $OutDir
} else {
    Join-Path (Get-Location) $OutDir
}
$resolvedOutDir = [System.IO.Path]::GetFullPath($resolvedOutDirInput)
New-DirectoryIfMissing -Path $resolvedOutDir

$sharedBuildCache = Join-Path $resolvedOutDir "shared-gocache"
$ciBuildCache = Join-Path $resolvedOutDir "ci-gocache"
$ciModuleCache = Join-Path $resolvedOutDir "ci-gomodcache"

Reset-Directory -Path $sharedBuildCache
Reset-Directory -Path $ciBuildCache
Reset-Directory -Path $ciModuleCache

$defaultGoCache = (& go env GOCACHE).Trim()
$defaultGoModCache = (& go env GOMODCACHE).Trim()

$variants = @(
    @{
        key = "shared-cache-cold"
        label = "shared cache cold"
        gocache = $sharedBuildCache
        gomodcache = $defaultGoModCache
        note = "Empty dedicated build cache with the normal module cache."
    },
    @{
        key = "shared-cache-warm"
        label = "shared cache warm"
        gocache = $sharedBuildCache
        gomodcache = $defaultGoModCache
        note = "Repeat build with the same dedicated build cache and reused module cache."
    },
    @{
        key = "shared-cache-small-edit"
        label = "shared cache small edit"
        gocache = $sharedBuildCache
        gomodcache = $defaultGoModCache
        note = "Small edit rebuild with the warmed shared build cache and reused module cache."
        smallEdit = $true
    },
    @{
        key = "isolated-build-cache"
        label = "isolated build cache"
        gocache = (Join-Path $resolvedOutDir "isolated-gocache")
        gomodcache = $defaultGoModCache
        note = "Fresh build cache with the normal module cache, approximating a cold compile on a prepared machine."
    },
    @{
        key = "ci-style-cold"
        label = "ci-style cold"
        gocache = $ciBuildCache
        gomodcache = $ciModuleCache
        note = "Fresh build cache and fresh module cache, approximating a clean CI worker."
        prepareModuleCache = $true
    },
    @{
        key = "ci-style-warm"
        label = "ci-style warm"
        gocache = $ciBuildCache
        gomodcache = $ciModuleCache
        note = "Repeat build after the CI-style caches were hydrated once in the same comparison run."
    },
    @{
        key = "ci-style-small-edit"
        label = "ci-style small edit"
        gocache = $ciBuildCache
        gomodcache = $ciModuleCache
        note = "Small edit rebuild after the CI-style caches were hydrated once in the same comparison run."
        smallEdit = $true
    }
)

Reset-Directory -Path (Join-Path $resolvedOutDir "isolated-gocache")

$results = [ordered]@{}
foreach ($variant in $variants) {
    if ($variant.key -eq "ci-style-cold") {
        Reset-Directory -Path $ciBuildCache
        Reset-Directory -Path $ciModuleCache
    }
    $results[$variant.key] = Invoke-VariantMeasurement -Variant $variant -MeasureScript $measureScript -ResolvedOutDir $resolvedOutDir -DefaultGOMODCACHE $defaultGoModCache -ReleaseProfileEnabled:$ReleaseProfile.IsPresent -BinaryNameValue $BinaryName
}

$summary = [ordered]@{
    package = $Package
    generated_at = (Get-Date).ToString("o")
    environment = [ordered]@{
        go_version = (& go version)
        default_gocache = $defaultGoCache
        default_gomodcache = $defaultGoModCache
    }
    variants = $results
}

$summaryPath = Join-Path $resolvedOutDir $SummaryName
$summary | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $summaryPath -Encoding utf8

Write-Host ("Wrote wasm build cache comparison to {0}" -f $summaryPath) -ForegroundColor Green