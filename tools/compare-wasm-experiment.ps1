param(
    [Parameter(Mandatory = $true)][string]$Baseline,
    [Parameter(Mandatory = $true)][string]$Candidate,
    [string]$OutFile,
    [double]$TimingRegressionPercent = 10,
    [double]$SizeRegressionPercent = 0,
    [double]$OtherRegressionPercent = 0
)

$ErrorActionPreference = "Stop"

function Resolve-InputPath {
    param([Parameter(Mandatory = $true)][string]$Path)

    return (Resolve-Path -LiteralPath $Path).ProviderPath
}

function Read-JsonFile {
    param([Parameter(Mandatory = $true)][string]$Path)

    return Get-Content -LiteralPath $Path -Raw | ConvertFrom-Json
}

function Add-NumericMetric {
    param(
        [Parameter(Mandatory = $true)]$Map,
        [AllowEmptyString()][string]$Path,
        [Parameter(Mandatory = $true)]$Value
    )

    if ($Value -is [bool]) {
        return
    }

    if ($Value -is [byte] -or
        $Value -is [sbyte] -or
        $Value -is [int16] -or
        $Value -is [uint16] -or
        $Value -is [int32] -or
        $Value -is [uint32] -or
        $Value -is [int64] -or
        $Value -is [uint64] -or
        $Value -is [single] -or
        $Value -is [double] -or
        $Value -is [decimal]) {
        $Map[$Path] = [double]$Value
    }
}

function Collect-NumericMetrics {
    param(
        [Parameter(Mandatory = $true)]$Value,
        [AllowEmptyString()][string]$Path = "",
        [Parameter(Mandatory = $true)]$Metrics
    )

    if ($null -eq $Value) {
        return
    }

    if ($Value -is [System.Management.Automation.PSCustomObject]) {
        foreach ($property in $Value.PSObject.Properties) {
            $childPath = if ([string]::IsNullOrEmpty($Path)) { $property.Name } else { "$Path.$($property.Name)" }
            Collect-NumericMetrics -Value $property.Value -Path $childPath -Metrics $Metrics
        }
        return
    }

    if ($Value -is [System.Collections.IDictionary]) {
        foreach ($entry in $Value.GetEnumerator()) {
            $childPath = if ([string]::IsNullOrEmpty($Path)) { [string]$entry.Key } else { "$Path.$($entry.Key)" }
            Collect-NumericMetrics -Value $entry.Value -Path $childPath -Metrics $Metrics
        }
        return
    }

    if ($Value -is [System.Collections.IEnumerable] -and -not ($Value -is [string])) {
        $index = 0
        foreach ($item in $Value) {
            $childPath = if ([string]::IsNullOrEmpty($Path)) { "[$index]" } else { "$Path[$index]" }
            Collect-NumericMetrics -Value $item -Path $childPath -Metrics $Metrics
            $index += 1
        }
        return
    }

    Add-NumericMetric -Map $Metrics -Path $Path -Value $Value
}

function Get-MetricCategory {
    param([Parameter(Mandatory = $true)][string]$MetricPath)

    if ($MetricPath -match '(^|\.)(?:[^.]+_ms|module_download_ms)$') {
        return "timing"
    }

    if ($MetricPath -match '(^|\.)bytes$') {
        return "size"
    }

    return "other"
}

function Get-ThresholdPercent {
    param([Parameter(Mandatory = $true)][string]$MetricPath)

    switch (Get-MetricCategory -MetricPath $MetricPath) {
        "timing" { return $TimingRegressionPercent }
        "size" { return $SizeRegressionPercent }
        default { return $OtherRegressionPercent }
    }
}

function Format-Percent {
    param([double]$Value)

    if ([double]::IsNaN($Value)) {
        return "n/a"
    }

    return ("{0:N2}%" -f $Value)
}

$baselinePath = Resolve-InputPath -Path $Baseline
$candidatePath = Resolve-InputPath -Path $Candidate

$baselineJson = Read-JsonFile -Path $baselinePath
$candidateJson = Read-JsonFile -Path $candidatePath

$baselineMetrics = @{}
$candidateMetrics = @{}

Collect-NumericMetrics -Value $baselineJson -Path "" -Metrics $baselineMetrics
Collect-NumericMetrics -Value $candidateJson -Path "" -Metrics $candidateMetrics

$metricPaths = @($baselineMetrics.Keys + $candidateMetrics.Keys | Sort-Object -Unique)
$results = New-Object System.Collections.Generic.List[object]
$regressions = New-Object System.Collections.Generic.List[object]

foreach ($metricPath in $metricPaths) {
    $baselineHasMetric = $baselineMetrics.ContainsKey($metricPath)
    $candidateHasMetric = $candidateMetrics.ContainsKey($metricPath)

    if (-not $baselineHasMetric) {
        $results.Add([ordered]@{
            path = $metricPath
            status = "added"
            category = Get-MetricCategory -MetricPath $metricPath
            baseline = $null
            candidate = $candidateMetrics[$metricPath]
            delta = $null
            delta_percent = $null
            threshold_percent = $null
        }) | Out-Null
        continue
    }

    if (-not $candidateHasMetric) {
        $results.Add([ordered]@{
            path = $metricPath
            status = "removed"
            category = Get-MetricCategory -MetricPath $metricPath
            baseline = $baselineMetrics[$metricPath]
            candidate = $null
            delta = $null
            delta_percent = $null
            threshold_percent = $null
        }) | Out-Null
        continue
    }

    $baselineValue = [double]$baselineMetrics[$metricPath]
    $candidateValue = [double]$candidateMetrics[$metricPath]
    $delta = $candidateValue - $baselineValue
    $deltaPercent = if ($baselineValue -eq 0) {
        if ($candidateValue -eq 0) { 0 } else { [double]::NaN }
    } else {
        ($delta / $baselineValue) * 100
    }
    $thresholdPercent = Get-ThresholdPercent -MetricPath $metricPath
    $status = if ($delta -lt 0) {
        "improved"
    } elseif ($delta -gt 0) {
        if (-not [double]::IsNaN($deltaPercent) -and $deltaPercent -gt $thresholdPercent) {
            "regressed"
        } else {
            "within-threshold"
        }
    } else {
        "unchanged"
    }

    $record = [ordered]@{
        path = $metricPath
        status = $status
        category = Get-MetricCategory -MetricPath $metricPath
        baseline = $baselineValue
        candidate = $candidateValue
        delta = $delta
        delta_percent = if ([double]::IsNaN($deltaPercent)) { $null } else { [math]::Round($deltaPercent, 4) }
        threshold_percent = $thresholdPercent
    }

    $results.Add($record) | Out-Null

    if ($status -eq "regressed") {
        $regressions.Add($record) | Out-Null
    }
}

$summary = [ordered]@{
    baseline = $baselinePath
    candidate = $candidatePath
    compared_at = (Get-Date).ToString("o")
    thresholds = [ordered]@{
        timing_regression_percent = $TimingRegressionPercent
        size_regression_percent = $SizeRegressionPercent
        other_regression_percent = $OtherRegressionPercent
    }
    counts = [ordered]@{
        total = $results.Count
        improved = @($results | Where-Object { $_.status -eq "improved" }).Count
        unchanged = @($results | Where-Object { $_.status -eq "unchanged" }).Count
        within_threshold = @($results | Where-Object { $_.status -eq "within-threshold" }).Count
        regressed = $regressions.Count
        added = @($results | Where-Object { $_.status -eq "added" }).Count
        removed = @($results | Where-Object { $_.status -eq "removed" }).Count
    }
    metrics = $results
}

Write-Host "Comparing wasm experiment manifests"
Write-Host "Baseline:  $baselinePath"
Write-Host "Candidate: $candidatePath"
Write-Host ""

$interestingResults = @($results | Where-Object { $_.status -ne "unchanged" })

if ($interestingResults.Count -eq 0) {
    Write-Host "No numeric differences found."
} else {
    foreach ($result in $interestingResults) {
        $thresholdLabel = if ($null -eq $result.threshold_percent) { "n/a" } else { ("{0:N2}%" -f [double]$result.threshold_percent) }
        $deltaLabel = if ($null -eq $result.delta) { "n/a" } else { ("{0:N2}" -f [double]$result.delta) }
        $percentLabel = if ($null -eq $result.delta_percent) { "n/a" } else { Format-Percent -Value ([double]$result.delta_percent) }
        $baselineLabel = if ($null -eq $result.baseline) { "n/a" } else { ("{0:N2}" -f [double]$result.baseline) }
        $candidateLabel = if ($null -eq $result.candidate) { "n/a" } else { ("{0:N2}" -f [double]$result.candidate) }
        Write-Host (("[{0}] {1} baseline={2} candidate={3} delta={4} ({5}) threshold={6}") -f $result.status.ToUpperInvariant(), $result.path, $baselineLabel, $candidateLabel, $deltaLabel, $percentLabel, $thresholdLabel)
    }
}

if ($OutFile) {
    $outPath = if ([System.IO.Path]::IsPathRooted($OutFile)) {
        $OutFile
    } else {
        Join-Path -Path (Get-Location) -ChildPath $OutFile
    }
    $outDir = Split-Path -Parent $outPath
    if ($outDir) {
        New-Item -ItemType Directory -Force -Path $outDir | Out-Null
    }
    $summary | ConvertTo-Json -Depth 100 | Set-Content -LiteralPath $outPath
    Write-Host ""
    Write-Host "Wrote comparison summary to $outPath"
}

if ($regressions.Count -gt 0) {
    Write-Host ""
    Write-Host ("Detected {0} metric regressions beyond configured thresholds." -f $regressions.Count)
    exit 1
}

Write-Host ""
Write-Host "No regressions exceeded the configured thresholds."
exit 0