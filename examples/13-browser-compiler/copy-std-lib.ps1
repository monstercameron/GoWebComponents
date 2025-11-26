$env:GOOS = 'js'
$env:GOARCH = 'wasm'
$destDir = Join-Path $PSScriptRoot "static\pkg\js_wasm"

if (-not (Test-Path $destDir)) {
    New-Item -ItemType Directory -Path $destDir -Force | Out-Null
}

Write-Host "Determining standard library paths..."
# We use 'go list' to find the export file for each package in std
# This relies on the build cache being populated.
# We run 'go install std' first to ensure everything is built and cached.
go install std

$packages = go list std
foreach ($pkg in $packages) {
    $export = go list -export -f '{{.Export}}' $pkg
    if ($export) {
        $pkgPath = $pkg -replace "/", "\"
        $destPath = Join-Path $destDir "$pkgPath.a"
        $parentDir = Split-Path $destPath
        
        if (-not (Test-Path $parentDir)) {
            New-Item -ItemType Directory -Path $parentDir -Force | Out-Null
        }
        
        Write-Host "Copying $pkg..."
        Copy-Item $export $destPath -Force
    }
}

Write-Host "Standard library copied to $destDir"
