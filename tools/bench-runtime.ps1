param(
    [string]$Package = "./internal/runtime",
    [int]$Count = 5,
    [string]$Bench = ".",
    [string]$Output = "",
    [string]$Exec = ""
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($Output)) {
    $stamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $safePackage = $Package -replace '[^A-Za-z0-9._-]', '_'
    $Output = Join-Path $PSScriptRoot ("bench-{0}-{1}.txt" -f $safePackage, $stamp)
}

$commandParts = @("test")
if (-not [string]::IsNullOrWhiteSpace($Exec)) {
    $commandParts += @("-exec", $Exec)
}
$commandParts += @($Package, "-run", "^$", "-bench", $Bench, "-benchmem", "-count", $Count)

$command = @("go") + $commandParts
Write-Host ("Running: {0}" -f (($command | ForEach-Object { $_ }) -join ' '))
$result = & go @commandParts 2>&1
$result | Set-Content -Path $Output
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}
Write-Host "Wrote benchmark output to $Output"
