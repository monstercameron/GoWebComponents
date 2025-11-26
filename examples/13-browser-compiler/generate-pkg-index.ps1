$pkgDir = "$PSScriptRoot/static/pkg/js_wasm"
$outputFile = "$PSScriptRoot/static/pkg/index.json"

$files = Get-ChildItem -Path $pkgDir -Recurse -Filter "*.a" | ForEach-Object {
    $relativePath = $_.FullName.Substring($pkgDir.Length + 1).Replace("\", "/")
    # Remove .a extension
    $relativePath.Substring(0, $relativePath.Length - 2)
}

$json = $files | ConvertTo-Json
$json | Set-Content -Path $outputFile
Write-Host "Generated index.json with $($files.Count) packages"