param(
    [Parameter(Mandatory = $true)][string]$Baseline,
    [Parameter(Mandatory = $true)][string]$Candidate
)

$ErrorActionPreference = "Stop"

if (Get-Command benchstat -ErrorAction SilentlyContinue) {
    benchstat $Baseline $Candidate
    exit 0
}

Write-Host "benchstat is not installed. Install with: go install golang.org/x/perf/cmd/benchstat@latest"
Write-Host "Baseline:  $Baseline"
Write-Host "Candidate: $Candidate"
