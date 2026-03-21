param(
    [Parameter(Mandatory = $true)][string]$Package,
    [Parameter(Mandatory = $true)][string]$BaselineGo,
    [Parameter(Mandatory = $true)][string]$CandidateGo,
    [string]$OutDir = "tmp/wasm-toolchain-comparison",
    [double]$TimingRegressionPercent = 10,
    [double]$SizeRegressionPercent = 0,
    [double]$OtherRegressionPercent = 0,
    [switch]$ReleaseProfile,
    [switch]$SkipCompression
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function New-DirectoryIfMissing {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Path $Path | Out-Null
    }
}

$scriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$measureScript = Join-Path $scriptRoot "measure-wasm-build.ps1"
$compareScript = Join-Path $scriptRoot "compare-wasm-experiment.ps1"

$resolvedOutDirInput = if ([System.IO.Path]::IsPathRooted($OutDir)) {
    $OutDir
} else {
    Join-Path (Get-Location) $OutDir
}
$resolvedOutDir = [System.IO.Path]::GetFullPath($resolvedOutDirInput)
New-DirectoryIfMissing -Path $resolvedOutDir

$baselineOutDir = Join-Path $resolvedOutDir "baseline"
$candidateOutDir = Join-Path $resolvedOutDir "candidate"
$comparisonPath = Join-Path $resolvedOutDir "toolchain-comparison.json"
$summaryPath = Join-Path $resolvedOutDir "wasm-toolchain-comparison.json"

$sharedArgs = @{
    Package = $Package
}
if ($ReleaseProfile) {
    $sharedArgs["ReleaseProfile"] = $true
}
if ($SkipCompression) {
    $sharedArgs["SkipCompression"] = $true
}

& $measureScript @sharedArgs -GoExecutable $BaselineGo -OutDir $baselineOutDir
$baselineExitCode = $LASTEXITCODE
if ($baselineExitCode -ne 0) {
    throw "Baseline toolchain build failed with exit code $baselineExitCode"
}

& $measureScript @sharedArgs -GoExecutable $CandidateGo -OutDir $candidateOutDir
$candidateExitCode = $LASTEXITCODE
if ($candidateExitCode -ne 0) {
    throw "Candidate toolchain build failed with exit code $candidateExitCode"
}

$baselineManifest = Join-Path $baselineOutDir "wasm-build-experiment.json"
$candidateManifest = Join-Path $candidateOutDir "wasm-build-experiment.json"

& $compareScript `
    -Baseline $baselineManifest `
    -Candidate $candidateManifest `
    -OutFile $comparisonPath `
    -TimingRegressionPercent $TimingRegressionPercent `
    -SizeRegressionPercent $SizeRegressionPercent `
    -OtherRegressionPercent $OtherRegressionPercent
$comparisonExitCode = $LASTEXITCODE

$summary = [ordered]@{
    package = $Package
    compared_at = (Get-Date).ToString("o")
    baseline = [ordered]@{
        go_executable = $BaselineGo
        go_version = (& $BaselineGo version)
        manifest = "baseline/wasm-build-experiment.json"
    }
    candidate = [ordered]@{
        go_executable = $CandidateGo
        go_version = (& $CandidateGo version)
        manifest = "candidate/wasm-build-experiment.json"
    }
    thresholds = [ordered]@{
        timing_regression_percent = $TimingRegressionPercent
        size_regression_percent = $SizeRegressionPercent
        other_regression_percent = $OtherRegressionPercent
    }
    comparison = "toolchain-comparison.json"
    regression_exit_code = $comparisonExitCode
}

$summary | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath $summaryPath -Encoding utf8

Write-Host ("Saved toolchain comparison summary to {0}" -f $summaryPath) -ForegroundColor Green
exit $comparisonExitCode